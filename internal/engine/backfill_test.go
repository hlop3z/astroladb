package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestDetectMissingBackfills_NotNullNoDefault(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:     "age",
				Type:     "integer",
				Nullable: false, // NOT NULL
				// No default or backfill
			},
		},
	}

	candidates := DetectMissingBackfills(ops)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]
	if c.Namespace != "auth" {
		t.Errorf("expected namespace 'auth', got %q", c.Namespace)
	}
	if c.Table != "user" {
		t.Errorf("expected table 'user', got %q", c.Table)
	}
	if c.Column != "age" {
		t.Errorf("expected column 'age', got %q", c.Column)
	}
	if c.ColType != "integer" {
		t.Errorf("expected type 'integer', got %q", c.ColType)
	}
}

func TestDetectMissingBackfills_NullableColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:     "nickname",
				Type:     "string",
				Nullable: true, // Nullable - no backfill needed
			},
		},
	}

	candidates := DetectMissingBackfills(ops)

	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for nullable column, got %d", len(candidates))
	}
}

func TestDetectMissingBackfills_HasDefault(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:       "status",
				Type:       "string",
				Nullable:   false,
				Default:    "active",
				DefaultSet: true, // Has default
			},
		},
	}

	candidates := DetectMissingBackfills(ops)

	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for column with default, got %d", len(candidates))
	}
}

func TestDetectMissingBackfills_HasBackfill(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:        "age",
				Type:        "integer",
				Nullable:    false,
				Backfill:    0,
				BackfillSet: true, // Has backfill
			},
		},
	}

	candidates := DetectMissingBackfills(ops)

	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for column with backfill, got %d", len(candidates))
	}
}

func TestDetectMissingBackfills_PrimaryKey(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:       "id",
				Type:       "uuid",
				Nullable:   false,
				PrimaryKey: true, // Primary key - auto-generated
			},
		},
	}

	candidates := DetectMissingBackfills(ops)

	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for primary key, got %d", len(candidates))
	}
}

func TestSuggestDefault(t *testing.T) {
	tests := []struct {
		colType  string
		expected string
	}{
		{"string", `""`},
		{"text", `""`},
		{"integer", "0"},
		{"float", "0"},
		{"decimal", "0"},
		{"boolean", "false"},
		{"date", `"1970-01-01"`},
		{"time", `"00:00:00"`},
		{"date_time", `"1970-01-01T00:00:00Z"`},
		{"json", `{}`},
		{"unknown", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.colType, func(t *testing.T) {
			got := SuggestDefault(tt.colType)
			if got != tt.expected {
				t.Errorf("SuggestDefault(%q) = %q, want %q", tt.colType, got, tt.expected)
			}
		})
	}
}

func TestApplyBackfills(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:     "age",
				Type:     "integer",
				Nullable: false,
			},
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name:     "name",
				Type:     "string",
				Nullable: false,
			},
		},
	}

	backfills := map[string]string{
		"auth.user.age": "0",
		// name not backfilled
	}

	result := ApplyBackfills(ops, backfills)

	if len(result) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(result))
	}

	// First column should have backfill
	addCol1 := result[0].(*ast.AddColumn)
	if !addCol1.Column.BackfillSet {
		t.Error("expected first column to have BackfillSet=true")
	}
	if addCol1.Column.Backfill != "0" {
		t.Errorf("expected backfill '0', got %v", addCol1.Column.Backfill)
	}

	// Second column should NOT have backfill
	addCol2 := result[1].(*ast.AddColumn)
	if addCol2.Column.BackfillSet {
		t.Error("expected second column to have BackfillSet=false")
	}
}

func TestApplyBackfills_Empty(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name: "age",
				Type: "integer",
			},
		},
	}

	// No backfills - should return original ops
	result := ApplyBackfills(ops, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(result))
	}

	// Should be the same operation
	if result[0] != ops[0] {
		t.Error("expected same operation reference when no backfills applied")
	}
}
