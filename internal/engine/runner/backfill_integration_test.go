package runner

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// TestBackfillIntegration verifies the complete backfill flow:
// 1. AddColumn with backfill modifier creates correct operations
// 2. SplitIntoPhases separates DDL and Data phases
// 3. GenerateBatchedUpdate produces correct SQL
// 4. Phase execution handles batched updates
func TestBackfillIntegration(t *testing.T) {
	// Simulate a migration that adds a column with backfill
	migration := engine.Migration{
		Revision: "001",
		Name:     "add_bio_with_backfill",
		Operations: []ast.Operation{
			&ast.AddColumn{
				TableRef: ast.TableRef{
					Namespace: "auth",
					Table_:    "user",
				},
				Column: &ast.ColumnDef{
					Name:        "bio",
					Type:        "string",
					TypeArgs:    []any{500},
					Backfill:    "No bio provided",
					BackfillSet: true,
				},
			},
			&ast.AddColumn{
				TableRef: ast.TableRef{
					Namespace: "auth",
					Table_:    "user",
				},
				Column: &ast.ColumnDef{
					Name:        "notifications_enabled",
					Type:        "boolean",
					Backfill:    true,
					BackfillSet: true,
				},
			},
		},
	}

	// Split into phases
	phases := engine.SplitIntoPhases(migration.Operations)

	// Verify we have 2 phases: DDL and Data
	if len(phases) != 2 {
		t.Fatalf("Expected 2 phases (DDL + Data), got %d", len(phases))
	}

	// Phase 1: DDL
	ddlPhase := phases[0]
	if ddlPhase.Type != engine.DDL {
		t.Errorf("Phase 0 type = %v, want DDL", ddlPhase.Type)
	}
	if len(ddlPhase.Ops) != 2 {
		t.Errorf("Phase 0 has %d operations, want 2", len(ddlPhase.Ops))
	}
	if !ddlPhase.InTransaction {
		t.Error("DDL phase should run in transaction")
	}
	if ddlPhase.Batched {
		t.Error("DDL phase should not be batched")
	}

	// Verify both operations are AddColumn
	for i, op := range ddlPhase.Ops {
		if _, ok := op.(*ast.AddColumn); !ok {
			t.Errorf("DDL operation %d is %T, want *ast.AddColumn", i, op)
		}
	}

	// Phase 2: Data
	dataPhase := phases[1]
	if dataPhase.Type != engine.Data {
		t.Errorf("Phase 1 type = %v, want Data", dataPhase.Type)
	}
	if len(dataPhase.Ops) != 2 {
		t.Errorf("Phase 1 has %d operations, want 2 (one backfill per column)", len(dataPhase.Ops))
	}
	if !dataPhase.InTransaction {
		t.Error("Data phase should run in transaction")
	}
	if !dataPhase.Batched {
		t.Error("Data phase should be batched")
	}

	// Verify backfill operations are RawSQL
	for i, op := range dataPhase.Ops {
		rawSQL, ok := op.(*ast.RawSQL)
		if !ok {
			t.Fatalf("Data operation %d is %T, want *ast.RawSQL", i, op)
		}

		// Verify SQL structure
		sql := rawSQL.SQL
		if !strings.Contains(sql, "DO $$") {
			t.Errorf("Backfill SQL %d missing DO $$ block", i)
		}
		if !strings.Contains(sql, "batch_size INT := 5000") {
			t.Errorf("Backfill SQL %d missing batch size", i)
		}
		if !strings.Contains(sql, "pg_sleep(1)") {
			t.Errorf("Backfill SQL %d missing sleep", i)
		}
		if !strings.Contains(sql, "auth_user") {
			t.Errorf("Backfill SQL %d missing table name", i)
		}
	}

	// Verify first backfill (bio column)
	bioSQL := dataPhase.Ops[0].(*ast.RawSQL).SQL
	if !strings.Contains(bioSQL, "bio") {
		t.Error("First backfill SQL missing 'bio' column")
	}
	if !strings.Contains(bioSQL, "'No bio provided'") {
		t.Error("First backfill SQL missing value")
	}
	if !strings.Contains(bioSQL, "bio IS NULL") {
		t.Error("First backfill SQL missing NULL check")
	}

	// Verify second backfill (notifications_enabled column)
	notifSQL := dataPhase.Ops[1].(*ast.RawSQL).SQL
	if !strings.Contains(notifSQL, "notifications_enabled") {
		t.Error("Second backfill SQL missing 'notifications_enabled' column")
	}
	if !strings.Contains(notifSQL, "TRUE") {
		t.Error("Second backfill SQL missing boolean value")
	}
	if !strings.Contains(notifSQL, "notifications_enabled IS NULL") {
		t.Error("Second backfill SQL missing NULL check")
	}
}

// TestBackfillWithSQLExpr verifies that sql() expressions are passed through correctly
func TestBackfillWithSQLExpr(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{
				Table_: "events",
			},
			Column: &ast.ColumnDef{
				Name:        "created_at",
				Type:        "timestamp",
				Backfill:    &ast.SQLExpr{Expr: "NOW()"},
				BackfillSet: true,
			},
		},
	}

	phases := engine.SplitIntoPhases(ops)

	// Should have DDL + Data phases
	if len(phases) != 2 {
		t.Fatalf("Expected 2 phases, got %d", len(phases))
	}

	dataPhase := phases[1]
	if dataPhase.Type != engine.Data {
		t.Fatalf("Phase 1 type = %v, want Data", dataPhase.Type)
	}

	rawSQL := dataPhase.Ops[0].(*ast.RawSQL)
	sql := rawSQL.SQL

	// Verify NOW() is used directly (not quoted)
	if !strings.Contains(sql, "SET created_at = NOW()") {
		t.Errorf("SQL should use NOW() directly, got: %s", sql)
	}
	if strings.Contains(sql, "'NOW()'") {
		t.Error("SQL should not quote NOW() expression")
	}
}

// TestBackfillSkipsNullableColumns verifies nullable columns don't generate backfill
func TestBackfillSkipsNullableColumns(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{
				Table_: "users",
			},
			Column: &ast.ColumnDef{
				Name:     "phone",
				Type:     "string",
				Nullable: true, // Nullable - no backfill needed
			},
		},
	}

	phases := engine.SplitIntoPhases(ops)

	// Should only have DDL phase (no Data phase)
	if len(phases) != 1 {
		t.Fatalf("Expected 1 phase (DDL only), got %d", len(phases))
	}

	if phases[0].Type != engine.DDL {
		t.Errorf("Phase 0 type = %v, want DDL", phases[0].Type)
	}
}

// TestBackfillMultipleColumns verifies multiple backfills in same migration
func TestBackfillMultipleColumns(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:        "bio",
				Type:        "string",
				Backfill:    "Default bio",
				BackfillSet: true,
			},
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:        "country",
				Type:        "string",
				Backfill:    "US",
				BackfillSet: true,
			},
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:     "phone",
				Type:     "string",
				Nullable: true,
				// No backfill
			},
		},
	}

	phases := engine.SplitIntoPhases(ops)

	// Should have DDL + Data phases
	if len(phases) != 2 {
		t.Fatalf("Expected 2 phases, got %d", len(phases))
	}

	// DDL phase should have all 3 AddColumn operations
	if len(phases[0].Ops) != 3 {
		t.Errorf("DDL phase has %d operations, want 3", len(phases[0].Ops))
	}

	// Data phase should have 2 backfill operations (bio + country)
	if len(phases[1].Ops) != 2 {
		t.Errorf("Data phase has %d operations, want 2", len(phases[1].Ops))
	}
}
