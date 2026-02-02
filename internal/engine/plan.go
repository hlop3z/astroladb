// Package engine provides the core migration execution functionality.
package engine

import (
	"sort"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// Direction indicates whether migrations run up (apply) or down (rollback).
type Direction int

const (
	// Up applies migrations (creates/modifies schema).
	Up Direction = iota
	// Down rolls back migrations (reverts schema changes).
	Down
)

// String returns the string representation of the direction.
func (d Direction) String() string {
	if d == Up {
		return "up"
	}
	return "down"
}

// Migration represents a single migration with its operations.
type Migration struct {
	// Revision is the unique identifier (e.g., "001", "20240101120000").
	Revision string

	// Name is the human-readable name (e.g., "add_users_table").
	Name string

	// Path is the file path to the migration script.
	Path string

	// Checksum is the SHA256 hash of the migration content for integrity checking.
	Checksum string

	// Operations are the forward (up) operations to apply.
	Operations []ast.Operation

	// DownOps are the reverse operations (auto-generated or explicit).
	DownOps []ast.Operation

	// Irreversible marks migrations that cannot be rolled back.
	Irreversible bool

	// Dependencies are revisions that must be applied before this one.
	Dependencies []string

	// BeforeHooks are SQL statements to execute before DDL operations.
	BeforeHooks []string

	// AfterHooks are SQL statements to execute after DDL operations.
	AfterHooks []string

	// DownBeforeHooks are SQL statements to execute before DDL operations during rollback.
	DownBeforeHooks []string

	// DownAfterHooks are SQL statements to execute after DDL operations during rollback.
	DownAfterHooks []string

	// Description is a human-readable summary of what this migration does.
	Description string

	// IsBaseline marks this as a squashed baseline migration.
	IsBaseline bool

	// SquashedThrough is the last revision that was squashed into this baseline.
	SquashedThrough string
}

// Plan represents a set of migrations to execute in a specific direction.
type Plan struct {
	// Migrations to execute, in order.
	Migrations []Migration

	// Direction of execution (Up or Down).
	Direction Direction
}

// IsEmpty returns true if the plan has no migrations to execute.
func (p *Plan) IsEmpty() bool {
	return len(p.Migrations) == 0
}

// PlanMigrations creates an execution plan based on current state.
//
// Parameters:
//   - all: All available migrations (must be sorted by revision)
//   - applied: Migrations already applied to the database
//   - target: Target revision (empty = latest for Up, empty = none for Down)
//   - dir: Direction of migration (Up or Down)
//
// For Up direction:
//   - Returns pending migrations from current state to target (or latest)
//   - Pending = all migrations not in applied list
//
// For Down direction:
//   - Returns applied migrations from current state down to target (or all)
//   - Ordered in reverse (most recent first)
func PlanMigrations(all []Migration, applied []AppliedMigration, target string, dir Direction) (*Plan, error) {
	// Create a set of applied revisions for quick lookup
	appliedSet := make(map[string]bool, len(applied))
	appliedChecksums := make(map[string]string, len(applied))
	for _, a := range applied {
		appliedSet[a.Revision] = true
		appliedChecksums[a.Revision] = a.Checksum
	}

	switch dir {
	case Up:
		// Verify checksums of applied migrations against current files
		if err := verifyChecksums(all, appliedChecksums); err != nil {
			return nil, err
		}
		return planUp(all, appliedSet, target)
	case Down:
		return planDown(all, applied, target)
	default:
		return &Plan{Direction: dir}, nil
	}
}

// verifyChecksums checks that all applied migrations still match their recorded checksums.
// Returns an error if any migration file has been modified after being applied.
func verifyChecksums(all []Migration, appliedChecksums map[string]string) error {
	for _, m := range all {
		recorded, ok := appliedChecksums[m.Revision]
		if !ok {
			continue // not applied yet
		}
		if recorded == "" || m.Checksum == "" {
			continue // no checksum to compare
		}
		if recorded != m.Checksum {
			return alerr.New(alerr.ErrMigrationChecksum, "migration file was modified after being applied").
				With("revision", m.Revision).
				With("name", m.Name).
				With("expected", recorded).
				With("actual", m.Checksum)
		}
	}
	return nil
}

// planUp creates a plan for applying migrations.
func planUp(all []Migration, appliedSet map[string]bool, target string) (*Plan, error) {
	plan := &Plan{
		Direction:  Up,
		Migrations: make([]Migration, 0),
	}

	// Sort all migrations by revision
	sorted := make([]Migration, len(all))
	copy(sorted, all)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Revision < sorted[j].Revision
	})

	hasApplied := len(appliedSet) > 0

	// Find pending migrations
	for _, m := range sorted {
		if appliedSet[m.Revision] {
			continue // Already applied
		}

		// Skip baseline for already-migrated environments
		if m.IsBaseline && hasApplied {
			continue
		}

		plan.Migrations = append(plan.Migrations, m)

		// Stop if we've reached the target
		if target != "" && m.Revision == target {
			break
		}
	}

	// If target was specified but not found, return error
	if target != "" && len(plan.Migrations) > 0 && plan.Migrations[len(plan.Migrations)-1].Revision != target {
		// Check if target exists at all
		found := false
		for _, m := range all {
			if m.Revision == target {
				found = true
				break
			}
		}
		if !found {
			return nil, alerr.New(alerr.ErrMigrationNotFound, "target migration not found").
				With("target", target)
		}
		// Target exists but is already applied - return partial plan
	}

	return plan, nil
}

// planDown creates a plan for rolling back migrations.
func planDown(all []Migration, applied []AppliedMigration, target string) (*Plan, error) {
	plan := &Plan{
		Direction:  Down,
		Migrations: make([]Migration, 0),
	}

	// Create a map of all migrations for lookup
	migrationMap := ToMap(all, func(m Migration) string { return m.Revision })

	// Sort applied migrations in reverse order (most recent first)
	sortedApplied := make([]AppliedMigration, len(applied))
	copy(sortedApplied, applied)
	sort.Slice(sortedApplied, func(i, j int) bool {
		return sortedApplied[i].Revision > sortedApplied[j].Revision
	})

	// Add migrations to rollback in reverse order
	for _, a := range sortedApplied {
		// Stop if we've reached the target (target stays applied)
		if target != "" && a.Revision == target {
			break
		}

		m, ok := migrationMap[a.Revision]
		if !ok {
			// Migration file is missing - this is a problem
			return nil, alerr.New(alerr.ErrMigrationNotFound, "migration file not found for applied revision").
				With("revision", a.Revision)
		}

		// Check if migration is irreversible
		if m.Irreversible && len(m.DownOps) == 0 {
			return nil, alerr.New(alerr.ErrMigrationFailed, "cannot rollback irreversible migration").
				With("revision", m.Revision).
				With("name", m.Name)
		}

		plan.Migrations = append(plan.Migrations, m)
	}

	return plan, nil
}

// PlanSingle creates a plan for a single migration.
// Useful for executing individual migrations out of order (dangerous!).
func PlanSingle(m Migration, dir Direction) *Plan {
	return &Plan{
		Direction:  dir,
		Migrations: []Migration{m},
	}
}

// PlanStatus represents the status of a migration.
type PlanStatus int

const (
	// StatusPending means the migration has not been applied.
	StatusPending PlanStatus = iota
	// StatusApplied means the migration has been applied.
	StatusApplied
	// StatusMissing means the migration file is missing but was previously applied.
	StatusMissing
	// StatusModified means the migration checksum doesn't match.
	StatusModified
)

// String returns the string representation of the status.
func (s PlanStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusApplied:
		return "applied"
	case StatusMissing:
		return "missing"
	case StatusModified:
		return "modified"
	default:
		return "unknown"
	}
}

// MigrationStatus provides status information about a migration.
type MigrationStatus struct {
	Revision  string
	Name      string
	Status    PlanStatus
	AppliedAt *string // nil if not applied
	Checksum  string
}

// GetStatus returns the status of all migrations.
func GetStatus(all []Migration, applied []AppliedMigration) []MigrationStatus {
	// Build maps for quick lookup
	appliedMap := ToMap(applied, func(a AppliedMigration) string { return a.Revision })
	migrationMap := ToMap(all, func(m Migration) string { return m.Revision })

	// Collect all unique revisions
	revisionSet := make(map[string]bool, len(all)+len(applied))
	for _, m := range all {
		revisionSet[m.Revision] = true
	}
	for _, a := range applied {
		revisionSet[a.Revision] = true
	}

	// Sort revisions
	revisions := make([]string, 0, len(revisionSet))
	for r := range revisionSet {
		revisions = append(revisions, r)
	}
	sort.Strings(revisions)

	// Build status list
	statuses := make([]MigrationStatus, 0, len(revisions))
	for _, rev := range revisions {
		status := MigrationStatus{
			Revision: rev,
		}

		m, hasMigration := migrationMap[rev]
		a, wasApplied := appliedMap[rev]

		if hasMigration {
			status.Name = m.Name
			status.Checksum = m.Checksum
		}

		if wasApplied {
			appliedStr := a.AppliedAt.Format("2006-01-02 15:04:05")
			status.AppliedAt = &appliedStr

			if !hasMigration {
				status.Status = StatusMissing
			} else if a.Checksum != "" && m.Checksum != "" && a.Checksum != m.Checksum {
				status.Status = StatusModified
			} else {
				status.Status = StatusApplied
			}
		} else {
			status.Status = StatusPending
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// GenerateDownOps generates down (rollback) operations for operations.
// This is used for auto-generating rollback operations.
func GenerateDownOps(ops []ast.Operation) []ast.Operation {
	// Process in reverse order
	downOps := make([]ast.Operation, 0, len(ops))

	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		downOp := generateDownOp(op)
		if downOp != nil {
			downOps = append(downOps, downOp)
		}
	}

	return downOps
}

// generateDownOp generates the reverse operation for a single operation.
func generateDownOp(op ast.Operation) ast.Operation {
	switch o := op.(type) {
	case *ast.CreateTable:
		// Reverse of CREATE TABLE is DROP TABLE
		return &ast.DropTable{
			TableOp:  o.TableOp,
			IfExists: true,
		}

	case *ast.DropTable:
		// Cannot auto-generate CREATE TABLE from DROP
		// Requires schema information
		return nil

	case *ast.AddColumn:
		// Reverse of ADD COLUMN is DROP COLUMN
		return &ast.DropColumn{
			TableRef: o.TableRef,
			Name:     o.Column.Name,
		}

	case *ast.DropColumn:
		// Cannot auto-generate ADD COLUMN from DROP
		// Requires column definition
		return nil

	case *ast.RenameColumn:
		// Reverse is to rename back
		return &ast.RenameColumn{
			TableRef: o.TableRef,
			OldName:  o.NewName,
			NewName:  o.OldName,
		}

	case *ast.RenameTable:
		// Reverse is to rename back
		return &ast.RenameTable{
			Namespace: o.Namespace,
			OldName:   o.NewName,
			NewName:   o.OldName,
		}

	case *ast.CreateIndex:
		// Reverse of CREATE INDEX is DROP INDEX
		return &ast.DropIndex{
			TableRef: o.TableRef,
			Name:     o.Name,
			IfExists: true,
		}

	case *ast.DropIndex:
		// Cannot auto-generate CREATE INDEX from DROP
		return nil

	case *ast.AddForeignKey:
		// Reverse of ADD FK is DROP FK
		return &ast.DropForeignKey{
			TableRef: o.TableRef,
			Name:     o.Name,
		}

	case *ast.DropForeignKey:
		// Cannot auto-generate ADD FK from DROP
		return nil

	case *ast.AddCheck:
		// Reverse of ADD CHECK is DROP CHECK
		return &ast.DropCheck{
			TableRef: o.TableRef,
			Name:     o.Name,
		}

	case *ast.DropCheck:
		// Cannot auto-generate ADD CHECK from DROP
		// Requires the expression to be known
		return nil

	case *ast.AlterColumn:
		// If we have the old column definition, generate a reverse AlterColumn
		if o.OldColumn != nil {
			reverse := &ast.AlterColumn{
				TableRef: o.TableRef,
				Name:     o.Name,
			}
			// Restore old type if it was changed
			if o.NewType != "" {
				reverse.NewType = o.OldColumn.Type
				reverse.NewTypeArgs = o.OldColumn.TypeArgs
			}
			// Restore old nullable if it was changed
			if o.SetNullable != nil {
				oldNullable := o.OldColumn.Nullable
				reverse.SetNullable = &oldNullable
			}
			// Restore old default if it was changed
			if o.SetDefault != nil || o.DropDefault {
				if o.OldColumn.Default != nil {
					reverse.SetDefault = o.OldColumn.Default
				} else {
					reverse.DropDefault = true
				}
			}
			reverse.OldColumn = nil // no further reverse chain needed
			return reverse
		}
		return nil

	case *ast.RawSQL:
		// Raw SQL cannot be auto-reversed
		return nil

	default:
		return nil
	}
}

// HasIrreversibleOps checks if any operations cannot be auto-reversed.
func HasIrreversibleOps(ops []ast.Operation) bool {
	for _, op := range ops {
		if generateDownOp(op) == nil {
			return true
		}
	}
	return false
}
