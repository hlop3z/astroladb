package runner

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/engine/topo"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// lockReleaseTimeout is the maximum time to wait when releasing a migration lock.
const lockReleaseTimeout = 5 * time.Second

// Runner executes migration plans against a database.
type Runner struct {
	db       *sql.DB
	dialect  dialect.Dialect
	versions *VersionManager
}

// NewRunner creates a new migration runner.
// Returns nil if db or dialect is nil.
func NewRunner(db *sql.DB, d dialect.Dialect) *Runner {
	if db == nil || d == nil {
		return nil
	}
	return &Runner{
		db:       db,
		dialect:  d,
		versions: NewVersionManager(db, d),
	}
}

// Run executes all migrations in the plan.
// Returns an error if any migration fails.
func (r *Runner) Run(ctx context.Context, plan *engine.Plan) error {
	if plan.IsEmpty() {
		return nil
	}

	// Ensure version tracking table exists
	if err := r.versions.EnsureTable(ctx); err != nil {
		return err
	}

	// Execute each migration
	for _, m := range plan.Migrations {
		if err := r.runOne(ctx, m, plan.Direction); err != nil {
			return alerr.Wrap(alerr.ErrMigrationFailed, err, "migration failed").
				With("revision", m.Revision).
				With("name", m.Name).
				With("direction", plan.Direction.String())
		}
	}

	return nil
}

// RunWithLock executes all migrations in the plan with distributed locking.
// This prevents concurrent migration execution across multiple processes.
// The lock is automatically released after execution (success or failure).
//
// Parameters:
//   - lockTimeout: Maximum time to wait for lock acquisition (0 = default 30s)
//
// This is the recommended method for production deployments.
func (r *Runner) RunWithLock(ctx context.Context, plan *engine.Plan, lockTimeout time.Duration) error {
	if plan.IsEmpty() {
		return nil
	}

	// Acquire the migration lock
	if err := r.versions.AcquireLock(ctx, lockTimeout); err != nil {
		return err
	}

	// Ensure lock is released when done (success or failure)
	defer func() {
		// Use background context for release in case ctx is cancelled
		releaseCtx, cancel := context.WithTimeout(context.Background(), lockReleaseTimeout)
		defer cancel()
		if err := r.versions.ReleaseLock(releaseCtx); err != nil {
			slog.Warn("failed to release migration lock", "error", err)
		}
	}()

	// Run the migrations
	return r.Run(ctx, plan)
}

// RunDryRun returns the SQL that would be executed without actually running it.
// Useful for review before applying migrations.
func (r *Runner) RunDryRun(ctx context.Context, plan *engine.Plan) ([]string, error) {
	if plan.IsEmpty() {
		return nil, nil
	}

	var allSQL []string

	for _, m := range plan.Migrations {
		ops := r.getOperations(m, plan.Direction)
		for _, op := range ops {
			sqls, err := r.operationToSQL(op)
			if err != nil {
				return nil, err
			}
			allSQL = append(allSQL, sqls...)
		}
	}

	return allSQL, nil
}

// runOne executes a single migration using multi-phase execution (v0.0.9).
// Phases: DDL (in transaction) → Indexes (concurrent) → Data (batched)
func (r *Runner) runOne(ctx context.Context, m engine.Migration, dir engine.Direction) error {
	start := time.Now()

	ops := r.getOperations(m, dir)

	// Split operations into execution phases
	phases := engine.SplitIntoPhases(ops)

	// Check if migration is partially complete (resume capability)
	currentPhase, err := r.versions.GetPhase(ctx, m.Revision)
	if err != nil {
		return err
	}

	// Execute each phase
	for i, phase := range phases {
		phaseNum := i + 1

		// Skip already completed phases
		if phaseNum <= currentPhase {
			slog.Info("skipping completed phase",
				"revision", m.Revision,
				"phase", phaseNum,
				"type", phase.Type.String())
			continue
		}

		// Execute the phase
		slog.Info("executing phase",
			"revision", m.Revision,
			"phase", phaseNum,
			"type", phase.Type.String(),
			"in_transaction", phase.InTransaction)

		if err := r.executePhase(ctx, m, phase, dir); err != nil {
			return alerr.Wrap(alerr.ErrMigrationFailed, err, "phase execution failed").
				With("phase", phaseNum).
				With("phase_type", phase.Type.String())
		}

		// Mark phase as complete
		if err := r.versions.UpdatePhase(ctx, m.Revision, phaseNum); err != nil {
			return err
		}
	}

	// Mark migration as fully complete (phase=0)
	if err := r.versions.UpdatePhase(ctx, m.Revision, 0); err != nil {
		return err
	}

	execTime := time.Since(start)

	// Collect all SQL for checksum (for backward compatibility)
	allSQL, err := r.collectSQL(m, dir)
	if err != nil {
		return err
	}

	switch dir {
	case engine.Up:
		return r.versions.RecordApplied(ctx, ApplyRecord{
			Revision:        m.Revision,
			Checksum:        m.Checksum,
			ExecTime:        execTime,
			Description:     m.Description,
			SQLChecksum:     ComputeSQLChecksum(allSQL),
			SquashedThrough: m.SquashedThrough,
		})
	case engine.Down:
		return r.versions.RecordRollback(ctx, m.Revision)
	default:
		return nil
	}
}

// executePhase executes a single phase of a migration with appropriate safety injections.
func (r *Runner) executePhase(ctx context.Context, m engine.Migration, phase engine.Phase, dir engine.Direction) error {
	if len(phase.Ops) == 0 {
		return nil // Empty phase, nothing to do
	}

	// Handle Index phase specially (CREATE INDEX CONCURRENTLY)
	if phase.Type == engine.Index {
		return r.executeIndexPhase(ctx, phase)
	}

	// Handle DDL and Data phases (run in transaction with lock timeout)
	return r.executeDDLOrDataPhase(ctx, phase)
}

// executeIndexPhase executes the Index phase (CREATE INDEX CONCURRENTLY, outside transaction).
func (r *Runner) executeIndexPhase(ctx context.Context, phase engine.Phase) error {
	// Set lock timeout for index creation (PostgreSQL only)
	dialectName := r.dialect.Name()
	if dialectName == "postgres" || dialectName == "postgresql" {
		if _, err := r.db.ExecContext(ctx, "SET lock_timeout = '2s'"); err != nil {
			return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to set lock timeout")
		}
	}

	// Execute each index creation with CONCURRENT
	for _, op := range phase.Ops {
		createIdx, ok := op.(*ast.CreateIndex)
		if !ok {
			return alerr.New(alerr.ErrSchemaInvalid, "non-index operation in Index phase").
				With("type", op.Type().String())
		}

		// Generate CONCURRENT index SQL with validation
		sql := engine.GenerateConcurrentIndex(createIdx, dialectName)

		// Execute the concurrent index creation
		if _, err := r.db.ExecContext(ctx, sql); err != nil {
			return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to create concurrent index").
				WithSQL(sql)
		}
	}

	return nil
}

// executeDDLOrDataPhase executes DDL or Data phases (in transaction with lock timeout).
func (r *Runner) executeDDLOrDataPhase(ctx context.Context, phase engine.Phase) error {
	// Generate SQL for all operations in this phase
	var sqlStatements []string
	for _, op := range phase.Ops {
		sqls, err := r.operationToSQL(op)
		if err != nil {
			return err
		}
		sqlStatements = append(sqlStatements, sqls...)
	}

	if len(sqlStatements) == 0 {
		return nil
	}

	dialectName := r.dialect.Name()

	// SQLite: Execute statements individually within a transaction
	// (SQLite driver doesn't support multiple statements in one Exec call)
	if dialectName == "sqlite" {
		return r.executeSQLiteTransaction(ctx, sqlStatements)
	}

	// PostgreSQL: Combine all SQL and wrap with transaction + lock timeout
	combinedSQL := strings.Join(sqlStatements, ";\n")
	wrappedSQL := engine.InjectLockTimeout(combinedSQL, true, dialectName)

	// Execute in transaction
	_, err := r.db.ExecContext(ctx, wrappedSQL)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to execute phase").
			WithSQL(wrappedSQL)
	}

	return nil
}

// executeSQLiteTransaction executes multiple SQL statements in a SQLite transaction.
// SQLite driver doesn't support multiple statements in one Exec call, so we execute
// each statement individually within a manual transaction.
func (r *Runner) executeSQLiteTransaction(ctx context.Context, statements []string) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to begin transaction")
	}
	defer tx.Rollback() // Rollback if not committed

	// Execute each statement
	for _, sql := range statements {
		if _, err := tx.ExecContext(ctx, sql); err != nil {
			return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to execute statement in transaction").
				WithSQL(sql)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to commit transaction")
	}

	return nil
}

// collectSQL generates all SQL statements for a migration without executing them.
func (r *Runner) collectSQL(m engine.Migration, dir engine.Direction) ([]string, error) {
	var allSQL []string

	ops := r.getOperations(m, dir)
	for _, op := range ops {
		sqls, err := r.operationToSQL(op)
		if err != nil {
			return nil, err
		}
		allSQL = append(allSQL, sqls...)
	}

	return allSQL, nil
}

// ComputeSQLChecksum computes a SHA-256 hash of the concatenated SQL statements.
func ComputeSQLChecksum(statements []string) string {
	if len(statements) == 0 {
		return ""
	}
	h := sha256.New()
	for _, s := range statements {
		h.Write([]byte(s))
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// runInTransaction executes a migration within a transaction.
// Used for PostgreSQL and SQLite which support transactional DDL.
// The entire migration is atomic - all statements succeed or all fail.
func (r *Runner) runInTransaction(ctx context.Context, m engine.Migration, dir engine.Direction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLTransaction, err, "failed to begin transaction")
	}

	// Track if we need to rollback
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	ops := r.getOperations(m, dir)
	for _, op := range ops {
		sqls, err := r.operationToSQL(op)
		if err != nil {
			return err
		}

		for _, sqlStmt := range sqls {
			if _, err := tx.ExecContext(ctx, sqlStmt); err != nil {
				return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to execute statement").
					WithSQL(sqlStmt)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return alerr.Wrap(alerr.ErrSQLTransaction, err, "failed to commit transaction")
	}
	committed = true

	return nil
}

// runWithoutTransaction executes a migration without a transaction.
// Used for databases that don't support transactional DDL.
func (r *Runner) runWithoutTransaction(ctx context.Context, m engine.Migration, dir engine.Direction) error {
	ops := r.getOperations(m, dir)
	for _, op := range ops {
		sqls, err := r.operationToSQL(op)
		if err != nil {
			return err
		}

		for _, sqlStmt := range sqls {
			if _, err := r.db.ExecContext(ctx, sqlStmt); err != nil {
				return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to execute statement").
					WithSQL(sqlStmt)
			}
		}
	}

	return nil
}

// getOperations returns the appropriate operations based on direction.
func (r *Runner) getOperations(m engine.Migration, dir engine.Direction) []ast.Operation {
	switch dir {
	case engine.Up:
		return m.Operations
	case engine.Down:
		var ops []ast.Operation
		if len(m.DownOps) > 0 {
			ops = m.DownOps
		} else {
			// Auto-generate down operations if not provided
			ops = GenerateDownOps(m.Operations)
		}
		// Sort down operations to handle FK dependencies correctly:
		// DROP FKs must come before DROP TABLEs for referenced tables
		return sortDownOperations(ops)
	default:
		return nil
	}
}

// sortDownOperations sorts rollback operations to ensure FK dependencies are
// handled correctly. The order is:
//  1. DROP FOREIGN KEY operations (must come before dropping referenced tables)
//  2. DROP INDEX operations
//  3. DROP COLUMN operations
//  4. DROP TABLE operations (tables with FKs referencing other tables first)
//  5. Other operations (CREATE, ALTER, etc.)
//
// This prevents FK constraint violations during rollback when tables reference
// each other (e.g., posts.user_id -> users.id means posts must be dropped first).
func sortDownOperations(ops []ast.Operation) []ast.Operation {
	if len(ops) <= 1 {
		return ops
	}

	// Group operations by type
	var dropFKs []*ast.DropForeignKey
	var dropIndexes []*ast.DropIndex
	var dropColumns []*ast.DropColumn
	var dropTables []*ast.DropTable
	var other []ast.Operation

	for _, op := range ops {
		switch o := op.(type) {
		case *ast.DropForeignKey:
			dropFKs = append(dropFKs, o)
		case *ast.DropIndex:
			dropIndexes = append(dropIndexes, o)
		case *ast.DropColumn:
			dropColumns = append(dropColumns, o)
		case *ast.DropTable:
			dropTables = append(dropTables, o)
		default:
			other = append(other, op)
		}
	}

	// Sort drop tables by reverse dependency order.
	// Tables that reference other tables must be dropped first.
	dropTables = sortDropTablesByFKDependency(dropTables, ops)

	// Build result in correct order for rollback
	var result []ast.Operation

	// 1. Drop foreign keys first (releases constraints)
	for _, op := range dropFKs {
		result = append(result, op)
	}

	// 2. Drop indexes
	for _, op := range dropIndexes {
		result = append(result, op)
	}

	// 3. Drop columns
	for _, op := range dropColumns {
		result = append(result, op)
	}

	// 4. Drop tables (dependents first)
	for _, op := range dropTables {
		result = append(result, op)
	}

	// 5. Other operations at the end
	result = append(result, other...)

	return result
}

// sortDropTablesByFKDependency sorts DropTable operations so that tables
// referencing other tables are dropped first. This examines the original
// CreateTable operations (which become DropTable on rollback) to find FK deps.
func sortDropTablesByFKDependency(dropTables []*ast.DropTable, allOps []ast.Operation) []*ast.DropTable {
	if len(dropTables) <= 1 {
		return dropTables
	}

	// Build a map of table name -> tables it references (from original CREATE ops)
	// We look for matching CreateTable operations in the original ops to get FK info
	tableDeps := make(map[string][]string)
	tableSet := make(map[string]bool)

	for _, dt := range dropTables {
		tableName := dt.Table()
		tableSet[tableName] = true
		tableDeps[tableName] = []string{}
	}

	// Find CreateTable operations to extract FK dependencies
	for _, op := range allOps {
		ct, ok := op.(*ast.CreateTable)
		if !ok {
			continue
		}

		tableName := ct.Table()
		if !tableSet[tableName] {
			continue
		}

		// Extract dependencies from FK definitions
		for _, fk := range ct.ForeignKeys {
			refTable := fk.RefTable
			// Convert reference to SQL table name format
			refTableSQL := refToSQLName(refTable, ct.Namespace)
			if refTableSQL != tableName { // Don't count self-references
				tableDeps[tableName] = append(tableDeps[tableName], refTableSQL)
			}
		}

		// Also check columns for FK references
		for _, col := range ct.Columns {
			if col.Reference == nil {
				continue
			}
			refTable := col.Reference.Table
			refTableSQL := refToSQLName(refTable, ct.Namespace)
			if refTableSQL != tableName { // Don't count self-references
				tableDeps[tableName] = append(tableDeps[tableName], refTableSQL)
			}
		}
	}

	// Build nodes for topological sort
	nodes := make([]*dropTableSortNode, len(dropTables))
	for i, dt := range dropTables {
		tableName := dt.Table()
		// For drop order, we need REVERSE dependency order:
		// If A references B, we need to drop A before B.
		// So A's "dependency" for drop ordering is... nothing (A comes first).
		// B's "dependency" is A (B must wait for A to be dropped).
		// We invert the graph: find tables that reference this table.
		var dependents []string
		for otherTable, refs := range tableDeps {
			if otherTable == tableName {
				continue
			}
			for _, ref := range refs {
				if ref == tableName {
					dependents = append(dependents, otherTable)
					break
				}
			}
		}
		nodes[i] = &dropTableSortNode{
			table: tableName,
			deps:  dependents, // Tables that must be dropped before this one
			op:    dt,
		}
	}

	// Perform topological sort
	sorted, err := topo.TopoSort(nodes)
	if err != nil {
		// Circular dependency - return in reverse alphabetical order as fallback
		sortStrings := func(s []*ast.DropTable) {
			for i := 1; i < len(s); i++ {
				for j := i; j > 0 && s[j].Table() > s[j-1].Table(); j-- {
					s[j], s[j-1] = s[j-1], s[j]
				}
			}
		}
		sortStrings(dropTables)
		return dropTables
	}

	// Extract sorted operations
	result := make([]*ast.DropTable, len(sorted))
	for i, node := range sorted {
		result[i] = node.op
	}
	return result
}

// dropTableSortNode wraps a DropTable for topological sorting.
type dropTableSortNode struct {
	table string
	deps  []string
	op    *ast.DropTable
}

func (n *dropTableSortNode) ID() string             { return n.table }
func (n *dropTableSortNode) Dependencies() []string { return n.deps }

// refToSQLName converts a table reference (e.g., "auth.user") to SQL name format (e.g., "auth_user").
func refToSQLName(ref, defaultNS string) string {
	ns, table, _ := engine.ParseSimpleRef(ref)
	if ns == "" {
		ns = defaultNS
	}
	return strutil.SQLName(ns, table)
}

// operationToSQL converts an operation to SQL statements.
// May return multiple statements for complex operations.
func (r *Runner) operationToSQL(op ast.Operation) ([]string, error) {
	if err := op.Validate(); err != nil {
		return nil, err
	}
	switch o := op.(type) {
	case *ast.CreateTable:
		sql, err := r.dialect.CreateTableSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.DropTable:
		sql, err := r.dialect.DropTableSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.AddColumn:
		return r.addColumnSQL(o)

	case *ast.DropColumn:
		sql, err := r.dialect.DropColumnSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.RenameColumn:
		sql, err := r.dialect.RenameColumnSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.AlterColumn:
		sql, err := r.dialect.AlterColumnSQL(o)
		if err != nil {
			return nil, err
		}
		// AlterColumn may return multiple statements separated by semicolons
		return r.splitStatements(sql), nil

	case *ast.CreateIndex:
		sql, err := r.dialect.CreateIndexSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.DropIndex:
		sql, err := r.dialect.DropIndexSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.RenameTable:
		sql, err := r.dialect.RenameTableSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.AddForeignKey:
		sql, err := r.dialect.AddForeignKeySQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.DropForeignKey:
		sql, err := r.dialect.DropForeignKeySQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.AddCheck:
		sql, err := r.dialect.AddCheckSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.DropCheck:
		sql, err := r.dialect.DropCheckSQL(o)
		if err != nil {
			return nil, err
		}
		return []string{sql}, nil

	case *ast.RawSQL:
		return r.rawSQL(o), nil

	default:
		return nil, alerr.New(alerr.ErrSchemaInvalid, "unsupported operation type").
			With("type", op.Type().String())
	}
}

// addColumnSQL generates SQL for adding a column.
// Handles backfill for NOT NULL columns with proper locking to prevent race conditions.
func (r *Runner) addColumnSQL(op *ast.AddColumn) ([]string, error) {
	var statements []string

	// If column has backfill and is NOT NULL, we need special handling
	hasBackfill := op.Column.BackfillSet && op.Column.Backfill != nil
	isNotNull := !op.Column.Nullable

	if hasBackfill && isNotNull {
		tableName := op.Table()
		quotedTable := r.dialect.QuoteIdent(tableName)
		colName := r.dialect.QuoteIdent(op.Column.Name)
		backfillValue := r.formatBackfillValue(op.Column.Backfill)

		// Step 0: Lock table to prevent concurrent inserts during backfill
		// This prevents the race condition where new rows with NULL values
		// are inserted between UPDATE and SET NOT NULL.
		// PostgreSQL: LOCK TABLE ... IN EXCLUSIVE MODE
		// SQLite: Already serialized within transaction (no explicit lock needed)
		if r.dialect.Name() == "postgres" {
			statements = append(statements, "LOCK TABLE "+quotedTable+" IN EXCLUSIVE MODE")
		}

		// Step 1: Add column as nullable first
		tempOp := *op
		tempCol := *op.Column
		tempCol.Nullable = true
		tempOp.Column = &tempCol

		sql, err := r.dialect.AddColumnSQL(&tempOp)
		if err != nil {
			return nil, err
		}
		statements = append(statements, sql)

		// Step 2: Backfill existing rows
		backfillSQL := "UPDATE " + quotedTable +
			" SET " + colName + " = " + backfillValue +
			" WHERE " + colName + " IS NULL"
		statements = append(statements, backfillSQL)

		// Step 3: Set NOT NULL constraint
		setNotNullSQL, err := r.setNotNullSQL(tableName, op.Column.Name)
		if err != nil {
			return nil, err
		}
		statements = append(statements, setNotNullSQL)

		// Lock is automatically released at end of transaction
	} else {
		// Simple case: just add the column
		sql, err := r.dialect.AddColumnSQL(op)
		if err != nil {
			return nil, err
		}
		statements = append(statements, sql)

		// If there's a backfill but column is nullable, still do the backfill
		if hasBackfill {
			tableName := op.Table()
			colName := r.dialect.QuoteIdent(op.Column.Name)
			backfillValue := r.formatBackfillValue(op.Column.Backfill)

			backfillSQL := "UPDATE " + r.dialect.QuoteIdent(tableName) +
				" SET " + colName + " = " + backfillValue +
				" WHERE " + colName + " IS NULL"
			statements = append(statements, backfillSQL)
		}
	}

	return statements, nil
}

// formatBackfillValue formats a backfill value for SQL.
func (r *Runner) formatBackfillValue(v any) string {
	if expr, ok := v.(*ast.SQLExpr); ok {
		return expr.Expr
	}

	// Format as literal
	switch val := v.(type) {
	case string:
		escaped := strings.ReplaceAll(val, "'", "''")
		return "'" + escaped + "'"
	case bool:
		switch r.dialect.Name() {
		case "postgres":
			if val {
				return "TRUE"
			}
			return "FALSE"
		default:
			if val {
				return "1"
			}
			return "0"
		}
	case nil:
		return "NULL"
	default:
		// Safe fallback: convert to string and escape
		s := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(s, "'", "''")
		return "'" + escaped + "'"
	}
}

// setNotNullSQL generates SQL to set a column as NOT NULL.
// Returns an error for SQLite since it doesn't support ALTER COLUMN SET NOT NULL.
func (r *Runner) setNotNullSQL(table, column string) (string, error) {
	quotedTable := r.dialect.QuoteIdent(table)
	quotedColumn := r.dialect.QuoteIdent(column)

	switch r.dialect.Name() {
	case "postgres":
		return "ALTER TABLE " + quotedTable + " ALTER COLUMN " + quotedColumn + " SET NOT NULL", nil
	case "sqlite":
		// SQLite doesn't support ALTER COLUMN to add NOT NULL.
		// This MUST return an error - silent failures are unacceptable.
		return "", alerr.New(alerr.ErrSQLExecution, "SQLite does not support ALTER COLUMN SET NOT NULL; use table recreation").
			WithTable("", table).
			WithColumn(column)
	default:
		return "ALTER TABLE " + quotedTable + " ALTER COLUMN " + quotedColumn + " SET NOT NULL", nil
	}
}

// rawSQL handles RawSQL operations with dialect-specific overrides.
func (r *Runner) rawSQL(op *ast.RawSQL) []string {
	var sql string

	// Check for dialect-specific override
	switch r.dialect.Name() {
	case "postgres":
		if op.Postgres != "" {
			sql = op.Postgres
		} else {
			sql = op.SQL
		}
	case "sqlite":
		if op.SQLite != "" {
			sql = op.SQLite
		} else {
			sql = op.SQL
		}
	default:
		sql = op.SQL
	}

	if sql == "" {
		return nil
	}

	return r.splitStatements(sql)
}

// splitStatements splits a multi-statement SQL string.
// Uses a simple state machine to avoid splitting on semicolons inside string literals.
func (r *Runner) splitStatements(sql string) []string {
	if sql == "" {
		return nil
	}

	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDollarQuote := false
	dollarTag := "" // the $tag$ delimiter

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		// Inside dollar-quoted string
		if inDollarQuote {
			current.WriteByte(ch)
			// Check if we're at the closing dollar tag
			if ch == '$' && i+len(dollarTag)-1 < len(sql) && sql[i:i+len(dollarTag)] == dollarTag {
				current.WriteString(dollarTag[1:])
				i += len(dollarTag) - 1
				inDollarQuote = false
				dollarTag = ""
			}
			continue
		}

		// Inside single-quoted string
		if inSingleQuote {
			current.WriteByte(ch)
			if ch == '\'' {
				// Check for escaped quote ('')
				if i+1 < len(sql) && sql[i+1] == '\'' {
					current.WriteByte(sql[i+1])
					i++
				} else {
					inSingleQuote = false
				}
			}
			continue
		}

		switch ch {
		case '\'':
			inSingleQuote = true
			current.WriteByte(ch)
		case '$':
			// Try to find a dollar-quote tag: $tag$ or $$
			end := strings.Index(sql[i+1:], "$")
			if end >= 0 {
				tag := sql[i : i+end+2] // e.g. "$$" or "$func$"
				// Validate tag content (alphanumeric/underscore only)
				validTag := true
				for _, r := range tag[1 : len(tag)-1] {
					if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
						validTag = false
						break
					}
				}
				if validTag {
					inDollarQuote = true
					dollarTag = tag
					current.WriteString(tag)
					i += len(tag) - 1
				} else {
					current.WriteByte(ch)
				}
			} else {
				current.WriteByte(ch)
			}
		case ';':
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}

	// Don't forget the last statement
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// Migrate is a convenience function that runs all pending migrations.
func (r *Runner) Migrate(ctx context.Context, all []engine.Migration) error {
	// Get applied migrations
	applied, err := r.versions.GetApplied(ctx)
	if err != nil {
		// Table might not exist yet
		if err := r.versions.EnsureTable(ctx); err != nil {
			return err
		}
		applied = nil
	}

	// Create plan for pending migrations
	plan, err := PlanMigrations(all, applied, "", engine.Up)
	if err != nil {
		return err
	}

	return r.Run(ctx, plan)
}

// Rollback is a convenience function that rolls back the last N migrations.
func (r *Runner) Rollback(ctx context.Context, all []engine.Migration, count int) error {
	if count <= 0 {
		return nil // count=0 means no rollback
	}

	// Get applied migrations
	applied, err := r.versions.GetApplied(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return nil
	}

	// Create rollback plan
	plan, err := PlanMigrations(all, applied, "", engine.Down)
	if err != nil {
		return err
	}

	// Limit to count
	if count < len(plan.Migrations) {
		plan.Migrations = plan.Migrations[:count]
	}

	return r.Run(ctx, plan)
}

// RollbackTo rolls back to a specific revision (the target stays applied).
func (r *Runner) RollbackTo(ctx context.Context, all []engine.Migration, target string) error {
	// Get applied migrations
	applied, err := r.versions.GetApplied(ctx)
	if err != nil {
		return err
	}

	// Create rollback plan
	plan, err := PlanMigrations(all, applied, target, engine.Down)
	if err != nil {
		return err
	}

	return r.Run(ctx, plan)
}

// Status returns the status of all migrations.
func (r *Runner) Status(ctx context.Context, all []engine.Migration) ([]engine.MigrationStatus, error) {
	// Ensure version tracking table exists
	if err := r.versions.EnsureTable(ctx); err != nil {
		return nil, err
	}

	applied, err := r.versions.GetApplied(ctx)
	if err != nil {
		return nil, err
	}

	return GetStatus(all, applied), nil
}

// VersionManager returns the version manager for direct access.
func (r *Runner) VersionManager() *VersionManager {
	return r.versions
}
