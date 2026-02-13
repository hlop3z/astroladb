package runner

import (
	"slices"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

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
func PlanMigrations(all []engine.Migration, applied []AppliedMigration, target string, dir engine.Direction) (*engine.Plan, error) {
	// Create a set of applied revisions for quick lookup
	appliedSet := make(map[string]bool, len(applied))
	appliedChecksums := make(map[string]string, len(applied))
	for _, a := range applied {
		appliedSet[a.Revision] = true
		appliedChecksums[a.Revision] = a.Checksum
	}

	switch dir {
	case engine.Up:
		// Verify checksums of applied migrations against current files
		if err := verifyChecksums(all, appliedChecksums); err != nil {
			return nil, err
		}
		return planUp(all, appliedSet, target)
	case engine.Down:
		return planDown(all, applied, target)
	default:
		return &engine.Plan{Direction: dir}, nil
	}
}

// verifyChecksums checks that all applied migrations still match their recorded checksums.
// Returns an error if any migration file has been modified after being applied.
func verifyChecksums(all []engine.Migration, appliedChecksums map[string]string) error {
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
func planUp(all []engine.Migration, appliedSet map[string]bool, target string) (*engine.Plan, error) {
	plan := &engine.Plan{
		Direction:  engine.Up,
		Migrations: make([]engine.Migration, 0),
	}

	// Sort all migrations by revision
	sorted := slices.Clone(all)
	slices.SortFunc(sorted, func(a, b engine.Migration) int {
		return strings.Compare(a.Revision, b.Revision)
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
func planDown(all []engine.Migration, applied []AppliedMigration, target string) (*engine.Plan, error) {
	plan := &engine.Plan{
		Direction:  engine.Down,
		Migrations: make([]engine.Migration, 0),
	}

	// Create a map of all migrations for lookup
	migrationMap := engine.ToMap(all, func(m engine.Migration) string { return m.Revision })

	// Sort applied migrations in reverse order (most recent first)
	sortedApplied := slices.Clone(applied)
	slices.SortFunc(sortedApplied, func(a, b AppliedMigration) int {
		return strings.Compare(b.Revision, a.Revision)
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
func PlanSingle(m engine.Migration, dir engine.Direction) *engine.Plan {
	return &engine.Plan{
		Direction:  dir,
		Migrations: []engine.Migration{m},
	}
}

// GetStatus returns the status of all migrations.
func GetStatus(all []engine.Migration, applied []AppliedMigration) []engine.MigrationStatus {
	// Build maps for quick lookup
	appliedMap := engine.ToMap(applied, func(a AppliedMigration) string { return a.Revision })
	migrationMap := engine.ToMap(all, func(m engine.Migration) string { return m.Revision })

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
	slices.Sort(revisions)

	// Build status list
	statuses := make([]engine.MigrationStatus, 0, len(revisions))
	for _, rev := range revisions {
		status := engine.MigrationStatus{
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
				status.Status = engine.StatusMissing
			} else if a.Checksum != "" && m.Checksum != "" && a.Checksum != m.Checksum {
				status.Status = engine.StatusModified
			} else {
				status.Status = engine.StatusApplied
			}
		} else {
			status.Status = engine.StatusPending
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
