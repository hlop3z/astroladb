//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// ---------------------------------------------------------------------------
// AddForeignKey (PostgreSQL / CockroachDB only)
// ---------------------------------------------------------------------------

func TestDSL_AddForeignKey(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			skipForSQLite(t, d)

			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.fk_parent", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("name", 100)
				})
				m.CreateTable("test.fk_child", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.UUID("parent_id")
				})
			})
			execOps(t, d.db, d.dialect, create)

			addFK := buildOps(func(m *dsl.MigrationBuilder) {
				m.AddForeignKey(
					"test.fk_child",
					[]string{"parent_id"},
					"test.fk_parent",
					[]string{"id"},
					dsl.FKName("fk_child_parent"),
					dsl.FKOnDelete("CASCADE"),
					dsl.FKOnUpdate("CASCADE"),
				)
			})
			execOps(t, d.db, d.dialect, addFK)

			// Verify parent table still exists (sanity)
			testutil.AssertTableExists(t, d.db, "test_fk_parent")
			testutil.AssertTableExists(t, d.db, "test_fk_child")
		})
	}
}

// ---------------------------------------------------------------------------
// DropForeignKey (PostgreSQL / CockroachDB only)
// ---------------------------------------------------------------------------

func TestDSL_DropForeignKey(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			skipForSQLite(t, d)

			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.dfk_parent", func(tb *dsl.TableBuilder) {
					addID(tb)
				})
				m.CreateTable("test.dfk_child", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.UUID("parent_id")
				})
			})
			execOps(t, d.db, d.dialect, create)

			addFK := buildOps(func(m *dsl.MigrationBuilder) {
				m.AddForeignKey(
					"test.dfk_child",
					[]string{"parent_id"},
					"test.dfk_parent",
					[]string{"id"},
					dsl.FKName("fk_dfk_child_parent"),
				)
			})
			execOps(t, d.db, d.dialect, addFK)

			dropFK := buildOps(func(m *dsl.MigrationBuilder) {
				m.DropForeignKey("test.dfk_child", "fk_dfk_child_parent")
			})
			execOps(t, d.db, d.dialect, dropFK)
		})
	}
}

// ---------------------------------------------------------------------------
// AddCheck / DropCheck (PostgreSQL / CockroachDB only)
// ---------------------------------------------------------------------------

func TestDSL_AddCheck(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			skipForSQLite(t, d)

			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.chk_tbl", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.Integer("age")
				})
			})
			execOps(t, d.db, d.dialect, create)

			// AddCheck is only in the JS bindings; use AST directly.
			addCheck := []ast.Operation{
				&ast.AddCheck{
					TableRef:   ast.TableRef{Namespace: "test", Table_: "chk_tbl"},
					Name:       "chk_age_positive",
					Expression: "age >= 0",
				},
			}
			execOps(t, d.db, d.dialect, addCheck)

			testutil.AssertTableExists(t, d.db, "test_chk_tbl")
		})
	}
}

func TestDSL_DropCheck(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			skipForSQLite(t, d)

			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.dchk_tbl", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.Integer("score")
				})
			})
			execOps(t, d.db, d.dialect, create)

			add := []ast.Operation{
				&ast.AddCheck{
					TableRef:   ast.TableRef{Namespace: "test", Table_: "dchk_tbl"},
					Name:       "chk_score",
					Expression: "score >= 0",
				},
			}
			execOps(t, d.db, d.dialect, add)

			drop := []ast.Operation{
				&ast.DropCheck{
					TableRef: ast.TableRef{Namespace: "test", Table_: "dchk_tbl"},
					Name:     "chk_score",
				},
			}
			execOps(t, d.db, d.dialect, drop)
		})
	}
}
