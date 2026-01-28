//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// ---------------------------------------------------------------------------
// Raw SQL
// ---------------------------------------------------------------------------

func TestDSL_RawSQL(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			// Create table first so raw SQL has something to work with
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.raw_sql", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("name", 100)
				})
			})
			execOps(t, d.db, d.dialect, create)

			// Use raw SQL to create an index
			raw := buildOps(func(m *dsl.MigrationBuilder) {
				m.SQL("CREATE INDEX idx_raw_name ON test_raw_sql (name)")
			})
			execOps(t, d.db, d.dialect, raw)

			testutil.AssertIndexExists(t, d.db, "test_raw_sql", "idx_raw_name")
		})
	}
}

func TestDSL_SQLDialect(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			// Create table first
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.dialect_sql", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("label", 100)
				})
			})
			execOps(t, d.db, d.dialect, create)

			// Dialect-specific raw SQL
			op := &ast.RawSQL{
				Postgres: "CREATE INDEX idx_dial_pg ON test_dialect_sql (label)",
				SQLite:   "CREATE INDEX idx_dial_sq ON test_dialect_sql (label)",
			}
			sqls, err := opToSQL(d.dialect, op)
			if err != nil {
				t.Fatalf("opToSQL: %v", err)
			}
			for _, s := range sqls {
				if _, err := d.db.Exec(s); err != nil {
					t.Fatalf("exec: %v", err)
				}
			}

			// Verify the dialect-appropriate index was created
			switch d.name {
			case "sqlite":
				testutil.AssertIndexExists(t, d.db, "test_dialect_sql", "idx_dial_sq")
			default:
				testutil.AssertIndexExists(t, d.db, "test_dialect_sql", "idx_dial_pg")
			}
		})
	}
}
