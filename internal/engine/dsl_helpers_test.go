//go:build integration

package engine_test

import (
	"database/sql"
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
func opToSQL(d dialect.Dialect, op ast.Operation) ([]string, error) {
	return dialect.OperationToSQL(d, op)
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
