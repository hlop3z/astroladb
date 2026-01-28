//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// ---------------------------------------------------------------------------
// AddColumn
// ---------------------------------------------------------------------------

func TestDSL_AddColumn(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			// Create base table
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.add_col", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("name", 100)
				})
			})
			execOps(t, d.db, d.dialect, create)

			// Add a new column
			add := buildOps(func(m *dsl.MigrationBuilder) {
				m.AddColumn("test.add_col", func(cb *dsl.ColumnBuilder) *dsl.ColumnBuilder {
					return dsl.NewColumnBuilder("phone", "string", 20).Nullable()
				})
			})
			execOps(t, d.db, d.dialect, add)
			testutil.AssertColumnExists(t, d.db, "test_add_col", "phone")
		})
	}
}

// ---------------------------------------------------------------------------
// DropColumn
// ---------------------------------------------------------------------------

func TestDSL_DropColumn(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.drop_col", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("removable", 50)
				})
			})
			execOps(t, d.db, d.dialect, create)
			testutil.AssertColumnExists(t, d.db, "test_drop_col", "removable")

			drop := buildOps(func(m *dsl.MigrationBuilder) {
				m.DropColumn("test.drop_col", "removable")
			})
			execOps(t, d.db, d.dialect, drop)
			testutil.AssertColumnNotExists(t, d.db, "test_drop_col", "removable")
		})
	}
}

// ---------------------------------------------------------------------------
// RenameColumn
// ---------------------------------------------------------------------------

func TestDSL_RenameColumn(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.ren_col", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("old_col", 100)
				})
			})
			execOps(t, d.db, d.dialect, create)

			rename := buildOps(func(m *dsl.MigrationBuilder) {
				m.RenameColumn("test.ren_col", "old_col", "new_col")
			})
			execOps(t, d.db, d.dialect, rename)
			testutil.AssertColumnExists(t, d.db, "test_ren_col", "new_col")
			testutil.AssertColumnNotExists(t, d.db, "test_ren_col", "old_col")
		})
	}
}

// ---------------------------------------------------------------------------
// AlterColumn (PostgreSQL / CockroachDB only)
// ---------------------------------------------------------------------------

func TestDSL_AlterColumn(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			skipForSQLite(t, d)

			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.alter_col", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("label", 50)
					tb.Integer("count")
				})
			})
			execOps(t, d.db, d.dialect, create)

			// SetNullable
			alter1 := buildOps(func(m *dsl.MigrationBuilder) {
				m.AlterColumn("test.alter_col", "label", func(a *dsl.AlterColumnBuilder) {
					a.SetNullable()
				})
			})
			execOps(t, d.db, d.dialect, alter1)

			// SetNotNull
			alter2 := buildOps(func(m *dsl.MigrationBuilder) {
				m.AlterColumn("test.alter_col", "label", func(a *dsl.AlterColumnBuilder) {
					a.SetNotNull()
				})
			})
			execOps(t, d.db, d.dialect, alter2)

			// SetDefault / DropDefault
			alter3 := buildOps(func(m *dsl.MigrationBuilder) {
				m.AlterColumn("test.alter_col", "count", func(a *dsl.AlterColumnBuilder) {
					a.SetDefault(0)
				})
			})
			execOps(t, d.db, d.dialect, alter3)

			alter4 := buildOps(func(m *dsl.MigrationBuilder) {
				m.AlterColumn("test.alter_col", "count", func(a *dsl.AlterColumnBuilder) {
					a.DropDefault()
				})
			})
			execOps(t, d.db, d.dialect, alter4)
		})
	}
}

// ---------------------------------------------------------------------------
// Column Modifiers (.optional, .unique, .default)
// ---------------------------------------------------------------------------

func TestDSL_ColumnModifiers(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.modifiers", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("email", 255).Unique()
					tb.String("nickname", 100).Nullable()
					tb.Boolean("active").Default(true)
					tb.Integer("score").Default(0).Nullable()
				})
			})
			execOps(t, d.db, d.dialect, ops)

			testutil.AssertTableExists(t, d.db, "test_modifiers")
			testutil.AssertColumnExists(t, d.db, "test_modifiers", "email")
			testutil.AssertColumnExists(t, d.db, "test_modifiers", "nickname")
			testutil.AssertColumnExists(t, d.db, "test_modifiers", "active")
			testutil.AssertColumnExists(t, d.db, "test_modifiers", "score")
		})
	}
}
