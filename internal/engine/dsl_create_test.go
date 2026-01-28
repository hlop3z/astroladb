//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// ---------------------------------------------------------------------------
// CreateTable
// ---------------------------------------------------------------------------

func TestDSL_CreateTable_AllTypes(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.all_types", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("name", 100)
					tb.Text("bio")
					tb.Integer("age")
					tb.Float("score")
					tb.Decimal("price", 10, 2)
					tb.Boolean("active")
					tb.Date("born_on")
					tb.Time("starts_at")
					tb.DateTime("published_at")
					tb.UUID("external_id")
					tb.JSON("metadata")
					tb.Base64("avatar")
					tb.Enum("status", []string{"draft", "published", "archived"})
					tb.Timestamps()
				})
			})
			execOps(t, d.db, d.dialect, ops)

			table := "test_all_types"
			testutil.AssertTableExists(t, d.db, table)
			for _, col := range []string{
				"id", "name", "bio", "age", "score", "price", "active",
				"born_on", "starts_at", "published_at", "external_id",
				"metadata", "avatar", "status", "created_at", "updated_at",
			} {
				testutil.AssertColumnExists(t, d.db, table, col)
			}
		})
	}
}

func TestDSL_CreateTable_WithRelationships(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				// Parent table
				m.CreateTable("auth.user", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("email", 255)
				})
				// belongs_to with alias
				m.CreateTable("blog.post", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.BelongsTo("auth.user", dsl.As("author"))
					tb.String("title", 200)
				})
				// one_to_one
				m.CreateTable("auth.profile", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.OneToOne("auth.user")
					tb.Text("bio")
				})
			})
			execOps(t, d.db, d.dialect, ops)

			testutil.AssertTableExists(t, d.db, "auth_user")
			testutil.AssertTableExists(t, d.db, "blog_post")
			testutil.AssertColumnExists(t, d.db, "blog_post", "author_id")
			testutil.AssertTableExists(t, d.db, "auth_profile")
			testutil.AssertColumnExists(t, d.db, "auth_profile", "user_id")
		})
	}
}

func TestDSL_CreateTable_Helpers(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("cms.item", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("title", 200)
					tb.Timestamps()
					tb.SoftDelete()
					tb.Sortable()
				})
			})
			execOps(t, d.db, d.dialect, ops)

			table := "cms_item"
			testutil.AssertTableExists(t, d.db, table)
			testutil.AssertColumnExists(t, d.db, table, "created_at")
			testutil.AssertColumnExists(t, d.db, table, "updated_at")
			testutil.AssertColumnExists(t, d.db, table, "deleted_at")
			testutil.AssertColumnExists(t, d.db, table, "position")
		})
	}
}

// ---------------------------------------------------------------------------
// DropTable
// ---------------------------------------------------------------------------

func TestDSL_DropTable(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			// Create then drop
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.droppable", func(tb *dsl.TableBuilder) {
					addID(tb)
				})
			})
			execOps(t, d.db, d.dialect, create)
			testutil.AssertTableExists(t, d.db, "test_droppable")

			drop := buildOps(func(m *dsl.MigrationBuilder) {
				m.DropTable("test.droppable")
			})
			execOps(t, d.db, d.dialect, drop)
			testutil.AssertTableNotExists(t, d.db, "test_droppable")
		})
	}
}

// ---------------------------------------------------------------------------
// RenameTable
// ---------------------------------------------------------------------------

func TestDSL_RenameTable(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			create := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.old_name", func(tb *dsl.TableBuilder) {
					addID(tb)
				})
			})
			execOps(t, d.db, d.dialect, create)
			testutil.AssertTableExists(t, d.db, "test_old_name")

			rename := buildOps(func(m *dsl.MigrationBuilder) {
				m.RenameTable("test.old_name", "new_name")
			})
			execOps(t, d.db, d.dialect, rename)
			testutil.AssertTableExists(t, d.db, "test_new_name")
			testutil.AssertTableNotExists(t, d.db, "test_old_name")
		})
	}
}
