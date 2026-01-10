package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestLintOperations_Empty(t *testing.T) {
	warnings := LintOperations(nil)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestLintOperations_DropTable(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropTable{
			TableOp: ast.TableOp{Namespace: "core", Name: "user"},
		},
	}

	warnings := LintOperations(ops)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	w := warnings[0]
	if w.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", w.Severity)
	}
	if w.Type != "drop_table" {
		t.Errorf("expected type 'drop_table', got %q", w.Type)
	}
	if w.Table != "core_user" {
		t.Errorf("expected table 'core_user', got %q", w.Table)
	}
}

func TestLintOperations_DropColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "core", Table_: "user"},
			Name:     "email",
		},
	}

	warnings := LintOperations(ops)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	w := warnings[0]
	if w.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", w.Severity)
	}
	if w.Type != "drop_column" {
		t.Errorf("expected type 'drop_column', got %q", w.Type)
	}
	if w.Table != "core_user" {
		t.Errorf("expected table 'core_user', got %q", w.Table)
	}
	if w.Column != "email" {
		t.Errorf("expected column 'email', got %q", w.Column)
	}
}

func TestLintOperations_MultipleDestructive(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropTable{TableOp: ast.TableOp{Namespace: "core", Name: "user"}},
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "core", Table_: "order"}, Name: "status"},
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "core", Name: "product"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}},
	}

	warnings := LintOperations(ops)
	if len(warnings) != 2 {
		t.Errorf("expected 2 warnings (drop_table + drop_column), got %d", len(warnings))
	}
}

func TestLintOperations_NonDestructive(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "core", Name: "user"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "core", Table_: "user"}, Column: &ast.ColumnDef{Name: "email", Type: "string"}},
		&ast.CreateIndex{TableRef: ast.TableRef{Namespace: "core", Table_: "user"}, Columns: []string{"email"}},
	}

	warnings := LintOperations(ops)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for non-destructive ops, got %d", len(warnings))
	}
}

func TestFormatWarnings_Empty(t *testing.T) {
	result := FormatWarnings(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormatWarnings_WithWarnings(t *testing.T) {
	warnings := []Warning{
		{Message: "Will DELETE ALL DATA in table 'core_user'"},
		{Message: "Will DELETE DATA in column 'core_order.status'"},
	}

	result := FormatWarnings(warnings)

	// Check that it contains expected content
	if result == "" {
		t.Error("expected non-empty result")
	}
	if len(result) < 50 {
		t.Error("result seems too short")
	}
}
