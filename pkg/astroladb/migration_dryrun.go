package astroladb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/ui"
)

// PhaseEstimate represents time and row count estimates for a migration phase.
type PhaseEstimate struct {
	Phase         engine.Phase
	PhaseNumber   int
	TotalPhases   int
	RowCount      int64  // Estimated rows affected
	EstimatedTime string // Human-readable time estimate
	SQL           string // Generated SQL for this phase
}

// MigrationEstimate represents all phase estimates for a single migration.
type MigrationEstimate struct {
	Revision      string
	Name          string
	Phases        []PhaseEstimate
	TotalEstimate string // Total estimated time for all phases
}

// dryRunMigrationsEnhanced generates phase-by-phase SQL with time and row estimates.
// This is the Week 7 enhanced version that shows:
//  1. SQL for each phase (DDL, Index, Data)
//  2. Row count estimates (from pg_class or COUNT)
//  3. Time estimates (instant for DDL, variable for Index, calculated for Backfill)
//  4. Total estimated time across all phases
func (c *Client) dryRunMigrationsEnhanced(ctx context.Context, plan *engine.Plan, output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	dialect := c.dialect.Name()

	// Generate estimates for each migration
	var allEstimates []MigrationEstimate
	for _, m := range plan.Migrations {
		estimate, err := c.estimateMigration(ctx, m, plan.Direction, dialect)
		if err != nil {
			// Don't fail on estimation errors - just show what we can
			c.log("Warning: failed to estimate migration %s: %v", m.Revision, err)
			continue
		}
		allEstimates = append(allEstimates, estimate)
	}

	// Print formatted output
	direction := "UP"
	if plan.Direction == engine.Down {
		direction = "DOWN"
	}

	fmt.Fprintf(output, "\n%s\n\n", ui.SectionTitle("Migration Dry-Run ("+direction+")"))

	for _, estimate := range allEstimates {
		c.printMigrationEstimate(output, estimate)
	}

	// Print total summary
	totalTime := c.calculateTotalTime(allEstimates)
	fmt.Fprintf(output, "%s\n", strings.Repeat("━", 80))
	fmt.Fprintf(output, "Total estimated time: %s\n\n", ui.Info(totalTime))

	return nil
}

// estimateMigration estimates all phases of a migration.
func (c *Client) estimateMigration(ctx context.Context, m engine.Migration, dir engine.Direction, dialect string) (MigrationEstimate, error) {
	ops := c.getOperations(m, dir)

	// Split into phases
	phases := engine.SplitIntoPhases(ops)

	estimate := MigrationEstimate{
		Revision: m.Revision,
		Name:     m.Name,
		Phases:   make([]PhaseEstimate, 0, len(phases)),
	}

	totalPhases := len(phases)

	for i, phase := range phases {
		phaseEst, err := c.estimatePhase(ctx, phase, i+1, totalPhases, dialect)
		if err != nil {
			return estimate, err
		}
		estimate.Phases = append(estimate.Phases, phaseEst)
	}

	// Calculate total estimate
	estimate.TotalEstimate = c.calculateMigrationTime(estimate.Phases)

	return estimate, nil
}

// estimatePhase estimates time and generates SQL for a single phase.
func (c *Client) estimatePhase(ctx context.Context, phase engine.Phase, phaseNum, totalPhases int, dialect string) (PhaseEstimate, error) {
	estimate := PhaseEstimate{
		Phase:       phase,
		PhaseNumber: phaseNum,
		TotalPhases: totalPhases,
	}

	// Generate SQL for this phase
	var sqlStatements []string
	for _, op := range phase.Ops {
		sqls, err := c.operationToSQL(op)
		if err != nil {
			return estimate, err
		}
		sqlStatements = append(sqlStatements, sqls...)
	}

	// Wrap SQL with appropriate transaction/lock timeout based on phase type
	combinedSQL := strings.Join(sqlStatements, ";\n")
	if phase.Type == engine.Index {
		// Index phase: CONCURRENTLY, no transaction
		// For dry-run, just add lock timeout (no concurrent wrapper needed)
		estimate.SQL = fmt.Sprintf("SET lock_timeout = '%s';\n%s", engine.DefaultLockTimeout, combinedSQL)
	} else {
		// DDL or Data phase: wrap in transaction with lock timeout
		estimate.SQL = engine.InjectLockTimeout(combinedSQL, phase.InTransaction, dialect)
	}

	// Estimate row count and time
	estimate.RowCount = c.estimateRowCount(ctx, phase)
	estimate.EstimatedTime = c.estimatePhaseTime(phase, estimate.RowCount, dialect)

	return estimate, nil
}

// estimateRowCount estimates number of rows affected by a phase.
// Uses pg_class.reltuples for PostgreSQL (fast but approximate),
// falls back to COUNT(*) for small tables or SQLite.
func (c *Client) estimateRowCount(ctx context.Context, phase engine.Phase) int64 {
	if c.db == nil {
		return 0
	}

	dialect := c.dialect.Name()

	// Extract table name from phase operations
	tableName := c.extractTableName(phase)
	if tableName == "" {
		return 0
	}

	// Try fast estimate first (PostgreSQL only)
	if dialect == "postgres" || dialect == "postgresql" {
		if count := c.estimateRowCountFast(ctx, tableName); count > 0 {
			return count
		}
	}

	// Fallback to exact count (slow for large tables)
	return c.estimateRowCountExact(ctx, tableName)
}

// estimateRowCountFast uses pg_class.reltuples for fast (but approximate) row count.
// Only works on PostgreSQL. Returns 0 if table doesn't exist or query fails.
func (c *Client) estimateRowCountFast(ctx context.Context, tableName string) int64 {
	query := `SELECT reltuples::bigint FROM pg_class WHERE relname = $1`

	var count int64
	err := c.db.QueryRowContext(ctx, query, tableName).Scan(&count)
	if err == sql.ErrNoRows || err != nil {
		return 0
	}

	return count
}

// estimateRowCountExact uses COUNT(*) for exact row count.
// Slow for large tables but works on all databases.
func (c *Client) estimateRowCountExact(ctx context.Context, tableName string) int64 {
	quotedTable := c.dialect.QuoteIdent(tableName)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)

	var count int64
	err := c.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0
	}

	return count
}

// extractTableName extracts the primary table name from a phase's operations.
func (c *Client) extractTableName(phase engine.Phase) string {
	if len(phase.Ops) == 0 {
		return ""
	}

	// Get table from first operation
	return phase.Ops[0].Table()
}

// estimatePhaseTime estimates execution time for a phase based on type and row count.
func (c *Client) estimatePhaseTime(phase engine.Phase, rowCount int64, dialect string) string {
	switch phase.Type {
	case engine.DDL:
		// DDL is instant for non-rewriting operations
		return "<1s (instant, no table rewrite)"

	case engine.Index:
		// Index creation time depends on table size and load
		if rowCount == 0 {
			return "<1s (table is empty)"
		}
		if rowCount < 100_000 {
			return "~10-30s (depends on server load)"
		}
		if rowCount < 1_000_000 {
			return "~1-3 minutes (depends on server load)"
		}
		return "~3-10 minutes (depends on server load)"

	case engine.Data:
		// Backfill time = (rows / batch_size) * time_per_batch
		// Batch size: 5,000 rows
		// Time per batch: ~2s (1s update + 1s pg_sleep)
		if rowCount == 0 {
			return "<1s (no rows to backfill)"
		}

		batchSize := int64(engine.DefaultBatchSize)
		batchSleep := int64(engine.DefaultBatchSleep)
		updateTime := int64(1) // ~1s per UPDATE

		numBatches := (rowCount + batchSize - 1) / batchSize // Ceiling division
		totalSeconds := numBatches * (updateTime + batchSleep)

		return formatDuration(time.Duration(totalSeconds) * time.Second)

	default:
		return "unknown"
	}
}

// calculateMigrationTime calculates total estimated time for a migration.
func (c *Client) calculateMigrationTime(phases []PhaseEstimate) string {
	// For now, just say "depends on load" since we can't accurately sum variable times
	hasIndex := false
	totalBackfillSeconds := int64(0)

	for _, p := range phases {
		if p.Phase.Type == engine.Index {
			hasIndex = true
		}
		if p.Phase.Type == engine.Data && p.RowCount > 0 {
			batchSize := int64(engine.DefaultBatchSize)
			numBatches := (p.RowCount + batchSize - 1) / batchSize
			totalBackfillSeconds += numBatches * 2
		}
	}

	if hasIndex && totalBackfillSeconds > 0 {
		return fmt.Sprintf("~%s + index creation time (variable)", formatDuration(time.Duration(totalBackfillSeconds)*time.Second))
	}
	if hasIndex {
		return "depends on index creation time (variable)"
	}
	if totalBackfillSeconds > 0 {
		return fmt.Sprintf("~%s", formatDuration(time.Duration(totalBackfillSeconds)*time.Second))
	}
	return "<1s (DDL only, instant)"
}

// calculateTotalTime calculates total time across all migrations.
func (c *Client) calculateTotalTime(estimates []MigrationEstimate) string {
	if len(estimates) == 0 {
		return "<1s"
	}

	// Just show the sum of individual migration estimates
	// (This is approximate since index times are variable)
	totalSeconds := int64(0)
	hasVariable := false

	for _, est := range estimates {
		for _, phase := range est.Phases {
			if phase.Phase.Type == engine.Index {
				hasVariable = true
			}
			if phase.Phase.Type == engine.Data && phase.RowCount > 0 {
				batchSize := int64(engine.DefaultBatchSize)
				numBatches := (phase.RowCount + batchSize - 1) / batchSize
				totalSeconds += numBatches * 2
			}
		}
	}

	if hasVariable {
		if totalSeconds > 0 {
			return fmt.Sprintf("~%s + index creation time (variable)", formatDuration(time.Duration(totalSeconds)*time.Second))
		}
		return "depends on index creation time (variable)"
	}
	if totalSeconds > 0 {
		return fmt.Sprintf("~%s", formatDuration(time.Duration(totalSeconds)*time.Second))
	}
	return "<1s"
}

// printMigrationEstimate prints a formatted migration estimate with all phases.
func (c *Client) printMigrationEstimate(output io.Writer, estimate MigrationEstimate) {
	// Migration header
	fmt.Fprintf(output, "Migration: %s: %s\n", ui.Info(estimate.Revision), estimate.Name)
	fmt.Fprintf(output, "%s\n\n", strings.Repeat("━", 80))

	// Print each phase
	for _, phase := range estimate.Phases {
		// Phase header
		phaseType := phase.Phase.Type.String()
		transactionNote := ""
		if phase.Phase.InTransaction {
			transactionNote = " (in transaction)"
		} else {
			transactionNote = " (outside transaction)"
		}

		fmt.Fprintf(output, "Phase %d/%d: %s%s\n", phase.PhaseNumber, phase.TotalPhases, ui.Success(phaseType), ui.Dim(transactionNote))
		fmt.Fprintf(output, "%s\n", strings.Repeat("─", 80))

		// SQL
		fmt.Fprintln(output, phase.SQL)
		fmt.Fprintln(output)

		// Estimates
		if phase.RowCount > 0 {
			fmt.Fprintf(output, "Rows affected: %s\n", ui.Info(formatRowCount(phase.RowCount)))
		}
		fmt.Fprintf(output, "Estimated time: %s\n\n", ui.Info(phase.EstimatedTime))
	}
}

// formatDuration formats a duration in human-readable form.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) - (minutes * 60)
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) - (hours * 60)
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// formatRowCount formats a row count with thousands separators.
func formatRowCount(count int64) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}

	// Add thousands separators
	s := fmt.Sprintf("%d", count)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// getOperations returns operations for a migration based on direction.
func (c *Client) getOperations(m engine.Migration, dir engine.Direction) []ast.Operation {
	switch dir {
	case engine.Up:
		return m.Operations
	case engine.Down:
		if len(m.DownOps) > 0 {
			return m.DownOps
		}
		// Auto-generate down operations if not provided
		// (This would need to be implemented, for now return empty)
		return nil
	default:
		return nil
	}
}

// operationToSQL converts an operation to SQL (helper method).
// This is a simplified version - the full implementation is in runner.go
func (c *Client) operationToSQL(op ast.Operation) ([]string, error) {
	// Use the dialect to generate SQL
	switch v := op.(type) {
	case *ast.CreateTable:
		sql, err := c.dialect.CreateTableSQL(v)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.AddColumn:
		sql, err := c.dialect.AddColumnSQL(v)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.CreateIndex:
		sql, err := c.dialect.CreateIndexSQL(v)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.RawSQL:
		if v.SQL == "" {
			return nil, nil
		}
		return []string{v.SQL}, nil

	// Add other operation types as needed...

	default:
		return nil, fmt.Errorf("unsupported operation type for dry-run: %T", op)
	}
}
