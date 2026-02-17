package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// ===========================================================================
// CRITICAL: sql() validation errors — schema context
// ===========================================================================

func TestCRITICAL_SQLError_Schema_NoArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.js")
	code := `export default table({
  id: col.id(),
  ts: sql(),
})`
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "schema")
	assertStructuredError(t, err, alerr.ErrSchemaInvalid,
		"error[SCH-001]", "sql() requires an object", "-->", "help:")
}

func TestCRITICAL_SQLError_Schema_StringArg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.js")
	code := `export default table({
  id: col.id(),
  ts: sql("NOW()"),
})`
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "schema")
	assertStructuredError(t, err, alerr.ErrSchemaInvalid,
		"error[SCH-001]", "sql() argument must be an object", "-->", "help:")
}

func TestCRITICAL_SQLError_Schema_MissingKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.js")
	code := `export default table({
  id: col.id(),
  ts: sql({ postgres: "NOW()" }),
})`
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "schema")
	assertStructuredError(t, err, alerr.ErrSchemaInvalid,
		"error[SCH-001]", "sql() requires both", "-->", "help:")
}

// ===========================================================================
// CRITICAL: sql() validation errors — migration context
// ===========================================================================

func TestCRITICAL_SQLError_Migration_NoArgs(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) {
  m.add_column("auth.user", function(col) {
    col.datetime("ts").backfill(sql())
  })
} })`)
	assertStructuredError(t, err, alerr.ErrSchemaInvalid,
		"error[SCH-001]", "sql() requires an object", "-->", "help:")
}

func TestCRITICAL_SQLError_Migration_StringArg(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) {
  m.add_column("auth.user", function(col) {
    col.datetime("ts").backfill(sql("NOW()"))
  })
} })`)
	assertStructuredError(t, err, alerr.ErrSchemaInvalid,
		"error[SCH-001]", "sql() argument must be an object", "-->", "help:")
}

func TestCRITICAL_SQLError_Migration_MissingKeys(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) {
  m.add_column("auth.user", function(col) {
    col.datetime("ts").backfill(sql({ sqlite: "CURRENT_TIMESTAMP" }))
  })
} })`)
	assertStructuredError(t, err, alerr.ErrSchemaInvalid,
		"error[SCH-001]", "sql() requires both", "-->", "help:")
}

// ===========================================================================
// CRITICAL: sql() happy path — correct values flow through
// ===========================================================================

func TestCRITICAL_SQL_HappyPath_Migration_AddColumnWithBackfill(t *testing.T) {
	code := `migration({ up: function(m) {
  m.add_column("auth.user", function(col) {
    col.datetime("last_active").backfill(sql({ postgres: "NOW()", sqlite: "CURRENT_TIMESTAMP" }))
  })
} })`
	dir := t.TempDir()
	path := filepath.Join(dir, "migration.js")
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	sb := NewSandbox(nil)
	ops, err := sb.RunFile(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(ops) == 0 {
		t.Fatal("expected at least one operation")
	}

	addCol, ok := ops[0].(*ast.AddColumn)
	if !ok {
		t.Fatalf("expected *ast.AddColumn, got %T", ops[0])
	}

	if addCol.Column.Name != "last_active" {
		t.Errorf("column name = %q, want %q", addCol.Column.Name, "last_active")
	}

	// Backfill value should be a *ast.SQLExpr
	sqlExpr, ok := addCol.Column.Backfill.(*ast.SQLExpr)
	if !ok {
		t.Fatalf("backfill value = %T, want *ast.SQLExpr", addCol.Column.Backfill)
	}
	if sqlExpr.Postgres != "NOW()" {
		t.Errorf("backfill postgres = %q, want %q", sqlExpr.Postgres, "NOW()")
	}
	if sqlExpr.SQLite != "CURRENT_TIMESTAMP" {
		t.Errorf("backfill sqlite = %q, want %q", sqlExpr.SQLite, "CURRENT_TIMESTAMP")
	}
}

func TestCRITICAL_SQL_HappyPath_Migration_CreateTable(t *testing.T) {
	code := `migration({ up: function(m) {
  m.create_table("auth.session", function(col) {
    col.uuid("id")
    col.datetime("expires_at")
  })
} })`
	dir := t.TempDir()
	path := filepath.Join(dir, "migration.js")
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	sb := NewSandbox(nil)
	ops, err := sb.RunFile(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(ops) == 0 {
		t.Fatal("expected at least one operation")
	}

	ct, ok := ops[0].(*ast.CreateTable)
	if !ok {
		t.Fatalf("expected *ast.CreateTable, got %T", ops[0])
	}

	if ct.Name != "session" {
		t.Errorf("table name = %q, want %q", ct.Name, "session")
	}
	if ct.Namespace != "auth" {
		t.Errorf("namespace = %q, want %q", ct.Namespace, "auth")
	}
	if len(ct.Columns) != 2 {
		t.Errorf("column count = %d, want 2", len(ct.Columns))
	}
}

// ===========================================================================
// ConvertSQLExprValue — unit test for new per-dialect format
// ===========================================================================

func TestConvertSQLExprValue_PerDialect(t *testing.T) {
	input := map[string]any{
		"_type":    "sql_expr",
		"postgres": "NOW()",
		"sqlite":   "CURRENT_TIMESTAMP",
	}

	result := ast.ConvertSQLExprValue(input)
	sqlExpr, ok := result.(*ast.SQLExpr)
	if !ok {
		t.Fatalf("expected *ast.SQLExpr, got %T", result)
	}
	if sqlExpr.Postgres != "NOW()" {
		t.Errorf("Postgres = %q, want %q", sqlExpr.Postgres, "NOW()")
	}
	if sqlExpr.SQLite != "CURRENT_TIMESTAMP" {
		t.Errorf("SQLite = %q, want %q", sqlExpr.SQLite, "CURRENT_TIMESTAMP")
	}
}

func TestConvertSQLExprValue_NonSQLExpr(t *testing.T) {
	// Non-sql_expr maps pass through unchanged
	input := map[string]any{"foo": "bar"}
	result := ast.ConvertSQLExprValue(input)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["foo"] != "bar" {
		t.Error("map should pass through unchanged")
	}
}

func TestConvertSQLExprValue_NonMap(t *testing.T) {
	// Non-map values pass through unchanged
	result := ast.ConvertSQLExprValue("hello")
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "hello" {
		t.Errorf("got %q, want %q", s, "hello")
	}
}
