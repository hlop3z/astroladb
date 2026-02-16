package engine

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestExtractBackfill(t *testing.T) {
	tests := []struct {
		name      string
		op        ast.Operation
		wantSQL   bool
		wantTable string
		wantCol   string
		wantValue string
	}{
		{
			name: "AddColumn with string backfill",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{
					Namespace: "auth",
					Table_:    "user",
				},
				Column: &ast.ColumnDef{
					Name:        "bio",
					Type:        "string",
					Backfill:    "No bio",
					BackfillSet: true,
				},
			},
			wantSQL:   true,
			wantTable: "auth_user",
			wantCol:   "bio",
			wantValue: "'No bio'",
		},
		{
			name: "AddColumn with boolean backfill",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{
					Table_: "settings",
				},
				Column: &ast.ColumnDef{
					Name:        "notifications_enabled",
					Type:        "boolean",
					Backfill:    true,
					BackfillSet: true,
				},
			},
			wantSQL:   true,
			wantTable: "settings",
			wantCol:   "notifications_enabled",
			wantValue: "TRUE",
		},
		{
			name: "AddColumn with integer backfill",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{
					Table_: "products",
				},
				Column: &ast.ColumnDef{
					Name:        "stock",
					Type:        "integer",
					Backfill:    0,
					BackfillSet: true,
				},
			},
			wantSQL:   true,
			wantTable: "products",
			wantCol:   "stock",
			wantValue: "0",
		},
		{
			name: "AddColumn with SQLExpr backfill",
			op: &ast.AddColumn{
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
			wantSQL:   true,
			wantTable: "events",
			wantCol:   "created_at",
			wantValue: "NOW()",
		},
		{
			name: "AddColumn without backfill",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{
					Table_: "users",
				},
				Column: &ast.ColumnDef{
					Name: "email",
					Type: "string",
				},
			},
			wantSQL: false,
		},
		{
			name: "CreateTable operation (not AddColumn)",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id"},
				},
			},
			wantSQL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBackfill(tt.op)

			if !tt.wantSQL {
				if result != nil {
					t.Errorf("extractBackfill() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("extractBackfill() = nil, want RawSQL operation")
			}

			rawSQL, ok := result.(*ast.RawSQL)
			if !ok {
				t.Fatalf("extractBackfill() = %T, want *ast.RawSQL", result)
			}

			sql := rawSQL.SQL

			// Verify SQL contains expected elements
			if !strings.Contains(sql, "DO $$") {
				t.Errorf("SQL missing DO $$ block")
			}
			if !strings.Contains(sql, tt.wantTable) {
				t.Errorf("SQL missing table name %q", tt.wantTable)
			}
			if !strings.Contains(sql, tt.wantCol) {
				t.Errorf("SQL missing column name %q", tt.wantCol)
			}
			if !strings.Contains(sql, tt.wantValue) {
				t.Errorf("SQL missing backfill value %q", tt.wantValue)
			}
			if !strings.Contains(sql, tt.wantCol+" IS NULL") {
				t.Errorf("SQL missing WHERE clause with NULL check")
			}
			if !strings.Contains(sql, "batch_size INT := 5000") {
				t.Errorf("SQL missing batch size declaration")
			}
			if !strings.Contains(sql, "pg_sleep(1)") {
				t.Errorf("SQL missing pg_sleep call")
			}
		})
	}
}

func TestSplitIntoPhases_WithBackfill(t *testing.T) {
	ops := []ast.Operation{
		// DDL: Add a column with backfill
		&ast.AddColumn{
			TableRef: ast.TableRef{
				Namespace: "auth",
				Table_:    "user",
			},
			Column: &ast.ColumnDef{
				Name:        "bio",
				Type:        "string",
				Backfill:    "Default bio",
				BackfillSet: true,
			},
		},
		// DDL: Add a column without backfill
		&ast.AddColumn{
			TableRef: ast.TableRef{
				Table_: "users",
			},
			Column: &ast.ColumnDef{
				Name: "email",
				Type: "string",
			},
		},
	}

	phases := SplitIntoPhases(ops)

	// Should have DDL phase and Data phase
	if len(phases) != 2 {
		t.Fatalf("SplitIntoPhases() returned %d phases, want 2", len(phases))
	}

	// Phase 1: DDL (both AddColumn operations)
	if phases[0].Type != DDL {
		t.Errorf("phases[0].Type = %v, want DDL", phases[0].Type)
	}
	if len(phases[0].Ops) != 2 {
		t.Errorf("phases[0].Ops length = %d, want 2", len(phases[0].Ops))
	}
	if !phases[0].InTransaction {
		t.Errorf("phases[0].InTransaction = false, want true")
	}

	// Phase 2: Data (backfill operation)
	if phases[1].Type != Data {
		t.Errorf("phases[1].Type = %v, want Data", phases[1].Type)
	}
	if len(phases[1].Ops) != 1 {
		t.Errorf("phases[1].Ops length = %d, want 1", len(phases[1].Ops))
	}
	if !phases[1].InTransaction {
		t.Errorf("phases[1].InTransaction = false, want true")
	}
	if !phases[1].Batched {
		t.Errorf("phases[1].Batched = false, want true")
	}

	// Verify the backfill operation is RawSQL
	rawSQL, ok := phases[1].Ops[0].(*ast.RawSQL)
	if !ok {
		t.Fatalf("phases[1].Ops[0] = %T, want *ast.RawSQL", phases[1].Ops[0])
	}
	if !strings.Contains(rawSQL.SQL, "DO $$") {
		t.Errorf("Backfill SQL missing DO $$ block")
	}
	if !strings.Contains(rawSQL.SQL, "auth_user") {
		t.Errorf("Backfill SQL missing table name 'auth_user'")
	}
	if !strings.Contains(rawSQL.SQL, "'Default bio'") {
		t.Errorf("Backfill SQL missing backfill value")
	}
}

func TestFormatBackfillValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"string", "hello", "'hello'"},
		{"string with quotes", "it's", "'it''s'"},
		{"boolean true", true, "TRUE"},
		{"boolean false", false, "FALSE"},
		{"integer", 42, "42"},
		{"int64", int64(100), "100"},
		{"float", 3.14, "3.14"},
		{"nil", nil, "NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBackfillValue(tt.value)
			if got != tt.want {
				t.Errorf("formatBackfillValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
