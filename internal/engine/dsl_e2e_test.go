//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestDSL_FullMigration exercises a realistic migration lifecycle:
//
//	create tables with relationships → add columns → create indexes → verify all.
func TestDSL_FullMigration(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			// --- Migration 001: initial schema ---
			m1 := buildOps(func(m *dsl.MigrationBuilder) {
				// auth.role
				m.CreateTable("auth.role", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("name", 100).Unique()
					tb.Timestamps()
				})

				// auth.user
				m.CreateTable("auth.user", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.String("email", 255).Unique()
					tb.Boolean("is_active").Default(true)
					tb.String("password", 255)
					tb.String("username", 50).Unique()
					tb.Timestamps()
				})

				// junction table
				m.CreateTable("auth_role_auth_user", func(tb *dsl.TableBuilder) {
					tb.BelongsTo("auth.role")
					tb.BelongsTo("auth.user")
				})
				m.CreateIndex("auth_role_auth_user", []string{"role_id"},
					dsl.IndexName("idx_arau_role"),
				)
				m.CreateIndex("auth_role_auth_user", []string{"user_id"},
					dsl.IndexName("idx_arau_user"),
				)
				m.CreateIndex("auth_role_auth_user", []string{"role_id", "user_id"},
					dsl.IndexName("idx_arau_unique"),
					dsl.IndexUnique(),
				)

				// blog.post
				m.CreateTable("blog.post", func(tb *dsl.TableBuilder) {
					addID(tb)
					tb.BelongsTo("auth.user", dsl.As("author"))
					tb.Text("body")
					tb.DateTime("published_at").Nullable()
					tb.String("slug", 255).Unique()
					tb.Enum("status", []string{"draft", "published", "archived"}).Default("draft")
					tb.String("title", 200)
					tb.SoftDelete()
					tb.Timestamps()
				})
				m.CreateIndex("blog.post", []string{"author_id"},
					dsl.IndexName("idx_blog_post_author"),
				)
			})
			execOps(t, d.db, d.dialect, m1)

			// Verify migration 001
			testutil.AssertTableExists(t, d.db, "auth_role")
			testutil.AssertTableExists(t, d.db, "auth_user")
			testutil.AssertTableExists(t, d.db, "auth_role_auth_user")
			testutil.AssertTableExists(t, d.db, "blog_post")
			testutil.AssertColumnExists(t, d.db, "blog_post", "author_id")
			testutil.AssertColumnExists(t, d.db, "blog_post", "deleted_at")
			testutil.AssertIndexExists(t, d.db, "auth_role_auth_user", "idx_arau_unique")
			testutil.AssertIndexExists(t, d.db, "blog_post", "idx_blog_post_author")

			// --- Migration 002: evolve schema ---
			m2 := buildOps(func(m *dsl.MigrationBuilder) {
				// Add a column to auth.user
				m.AddColumn("auth.user", func(cb *dsl.ColumnBuilder) *dsl.ColumnBuilder {
					return dsl.NewColumnBuilder("phone", "string", 20).Nullable()
				})

				// New index
				m.CreateIndex("auth.user", []string{"email"},
					dsl.IndexName("idx_user_email"),
				)
			})
			execOps(t, d.db, d.dialect, m2)

			testutil.AssertColumnExists(t, d.db, "auth_user", "phone")
			testutil.AssertIndexExists(t, d.db, "auth_user", "idx_user_email")

			// --- Migration 003: PostgreSQL-only ALTER ---
			if d.name != "sqlite" {
				m3 := buildOps(func(m *dsl.MigrationBuilder) {
					m.AlterColumn("auth.user", "phone", func(a *dsl.AlterColumnBuilder) {
						a.SetType("string", 30)
					})
				})
				// AddCheck is JS-only; use AST directly
				m3 = append(m3, &ast.AddCheck{
					TableRef:   ast.TableRef{Namespace: "auth", Table_: "user"},
					Name:       "chk_email_len",
					Expression: "length(email) > 0",
				})
				execOps(t, d.db, d.dialect, m3)
			}
		})
	}
}
