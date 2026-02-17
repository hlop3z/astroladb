package dialect

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestCRITICAL_SQLExpr_DialectSelection verifies that buildDefaultValueSQL
// selects the correct per-dialect expression from *ast.SQLExpr.
func TestCRITICAL_SQLExpr_DialectSelection(t *testing.T) {
	expr := &ast.SQLExpr{
		Postgres: "NOW()",
		SQLite:   "CURRENT_TIMESTAMP",
	}

	pgResult := buildDefaultValueSQL(expr, "postgres", PostgresBooleans)
	if pgResult != "NOW()" {
		t.Errorf("postgres dialect got %q, want %q", pgResult, "NOW()")
	}

	slResult := buildDefaultValueSQL(expr, "sqlite", SQLiteBooleans)
	if slResult != "CURRENT_TIMESTAMP" {
		t.Errorf("sqlite dialect got %q, want %q", slResult, "CURRENT_TIMESTAMP")
	}
}

// TestCRITICAL_SQLExpr_UUIDDefault verifies per-dialect UUID default expressions.
func TestCRITICAL_SQLExpr_UUIDDefault(t *testing.T) {
	expr := &ast.SQLExpr{
		Postgres: "gen_random_uuid()",
		SQLite:   "lower(hex(randomblob(4)))",
	}

	pgResult := buildDefaultValueSQL(expr, "postgres", PostgresBooleans)
	if pgResult != "gen_random_uuid()" {
		t.Errorf("postgres UUID default = %q, want %q", pgResult, "gen_random_uuid()")
	}

	slResult := buildDefaultValueSQL(expr, "sqlite", SQLiteBooleans)
	if slResult != "lower(hex(randomblob(4)))" {
		t.Errorf("sqlite UUID default = %q, want %q", slResult, "lower(hex(randomblob(4)))")
	}
}
