// Migration: initial
// Generated at: 2026-01-07T02:40:17-06:00

migration(m => {
  m.create_table("auth.role", t => {
    t.id()
    t.string("name", 50).unique()
    t.timestamps()
  })
  m.create_table("auth.user", t => {
    t.id()
    t.string("email", 255).unique()
    t.string("name", 100)
    t.timestamps()
  })
  m.create_table("auth_role_auth_user", t => {
    t.uuid("role_id")
    t.uuid("user_id")
  })
  m.create_index("auth_role_auth_user", ["role_id"], {name: "idx_auth_role_auth_user_role_id"})
  m.create_index("auth_role_auth_user", ["user_id"], {name: "idx_auth_role_auth_user_user_id"})
  m.create_index("auth_role_auth_user", ["role_id", "user_id"], {unique: true, name: "idx_auth_role_auth_user_unique"})
  m.create_table("core.order", t => {
    t.id()
    t.uuid("user_id")
    t.decimal("total", 10, 2)
    t.timestamps()
  })
  m.create_index("core.order", ["user_id"])
  m.create_table("core.product", t => {
    t.id()
    t.string("name", 200)
    t.text("description").optional()
    t.decimal("price", 10, 2)
    t.integer("stock_count").default(0)
    t.boolean("is_active").default(true)
    t.enum("status", ["draft", "published", "archived"]).default("draft")
    t.json("metadata").optional()
    t.timestamps()
  })
  m.create_table("social.follow", t => {
    t.id()
    t.uuid("follower_id")
    t.uuid("follows_id")
    t.timestamps()
  })
  m.create_index("social.follow", ["follower_id"])
  m.create_index("social.follow", ["follows_id"])
  m.create_index("social.follow", ["follower_id", "follows_id"], {unique: true})
})

// Rollback operations (for reference):
// migration(m => {
//   m.drop_table("social.follow")
//   m.drop_table("core.product")
//   m.drop_table("core.order")
//   m.drop_table("auth_role_auth_user")
//   m.drop_table("auth.user")
//   m.drop_table("auth.role")
// })
