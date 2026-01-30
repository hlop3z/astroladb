//go:build integration

package engine_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// setupDialects returns database connections for all supported dialects.
// Extends setupAllDatabases with CockroachDB (optional).
func setupDialects(t *testing.T) []dialectDB {
	t.Helper()

	dialects := []dialectDB{
		{name: "sqlite", db: testutil.SetupSQLite(t), dialect: dialect.SQLite()},
		{name: "postgres", db: testutil.SetupPostgres(t), dialect: dialect.Postgres()},
	}

	// CockroachDB is optional — skip if not available.
	if crdb := trySetupCockroachDB(t); crdb != nil {
		dialects = append(dialects, dialectDB{
			name: "cockroachdb", db: crdb, dialect: dialect.Postgres(),
		})
	}

	return dialects
}

// trySetupCockroachDB attempts to connect to CockroachDB, returning nil if unavailable.
func trySetupCockroachDB(t *testing.T) *sql.DB {
	t.Helper()
	defer func() { recover() }() //nolint:errcheck // swallow panics from missing CockroachDB
	return testutil.SetupCockroachDB(t)
}

// execOps generates SQL from operations using the given dialect and executes them.
// Unlike execOperations, this handles multi-statement ALTER and AddCheck/DropCheck.
func execOps(t *testing.T, db *sql.DB, d dialect.Dialect, ops []ast.Operation) {
	t.Helper()
	for _, op := range ops {
		sqls, err := opToSQL(d, op)
		if err != nil {
			t.Fatalf("failed to generate SQL for %T: %v", op, err)
		}
		for _, s := range sqls {
			if _, err := db.Exec(s); err != nil {
				t.Fatalf("failed to execute SQL:\n%s\nerror: %v", s, err)
			}
		}
	}
}

// tryExecOps is like execOps but returns an error instead of failing the test.
func tryExecOps(t *testing.T, db *sql.DB, d dialect.Dialect, ops []ast.Operation) error {
	t.Helper()
	for _, op := range ops {
		sqls, err := opToSQL(d, op)
		if err != nil {
			return err
		}
		for _, s := range sqls {
			if _, err := db.Exec(s); err != nil {
				return err
			}
		}
	}
	return nil
}

// opToSQL dispatches an AST operation to the appropriate dialect method.
// Extends operationToSQL with AddCheck, DropCheck, and multi-statement support.
func opToSQL(d dialect.Dialect, op ast.Operation) ([]string, error) {
	switch o := op.(type) {
	case *ast.CreateTable:
		s, err := d.CreateTableSQL(o)
		return []string{s}, err
	case *ast.DropTable:
		s, err := d.DropTableSQL(o)
		return []string{s}, err
	case *ast.RenameTable:
		s, err := d.RenameTableSQL(o)
		return []string{s}, err
	case *ast.AddColumn:
		s, err := d.AddColumnSQL(o)
		return []string{s}, err
	case *ast.DropColumn:
		s, err := d.DropColumnSQL(o)
		return []string{s}, err
	case *ast.RenameColumn:
		s, err := d.RenameColumnSQL(o)
		return []string{s}, err
	case *ast.AlterColumn:
		s, err := d.AlterColumnSQL(o)
		if err != nil {
			return nil, err
		}
		return splitStatements(s), nil
	case *ast.CreateIndex:
		s, err := d.CreateIndexSQL(o)
		return []string{s}, err
	case *ast.DropIndex:
		s, err := d.DropIndexSQL(o)
		return []string{s}, err
	case *ast.AddForeignKey:
		s, err := d.AddForeignKeySQL(o)
		return []string{s}, err
	case *ast.DropForeignKey:
		s, err := d.DropForeignKeySQL(o)
		return []string{s}, err
	case *ast.AddCheck:
		s, err := d.AddCheckSQL(o)
		return []string{s}, err
	case *ast.DropCheck:
		s, err := d.DropCheckSQL(o)
		return []string{s}, err
	case *ast.RawSQL:
		s, err := d.RawSQLFor(o)
		return []string{s}, err
	default:
		return nil, nil
	}
}

// splitStatements splits multi-statement SQL (separated by ";\n").
func splitStatements(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";\n") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// skipForSQLite skips the test when running on SQLite (for unsupported operations).
func skipForSQLite(t *testing.T, d dialectDB) {
	t.Helper()
	if d.name == "sqlite" {
		t.Skip("operation not supported on SQLite")
	}
}

// buildOps creates operations using the DSL MigrationBuilder.
func buildOps(fn func(m *dsl.MigrationBuilder)) []ast.Operation {
	m := dsl.NewMigrationBuilder()
	fn(m)
	return m.Operations()
}

// addID adds a portable ID primary key column.
// tb.ID() sets Default(gen_random_uuid()) which is Postgres-specific;
// this helper avoids that so all dialects work.
func addID(tb *dsl.TableBuilder) {
	col := dsl.NewColumnBuilder("id", "id")
	col.PrimaryKey()
	tb.AddColumn(col.Build())
}
