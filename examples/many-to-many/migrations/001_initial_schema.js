// Migration: initial_schema
// Generated at: 2026-01-07T02:22:55-06:00

migration(m => {
  m.create_table("auth.roles", t => {
    t.id()
    t.string("name", 50).unique()
    t.string("description", 255).optional()
    t.timestamps()
  })
  m.create_table("auth.users", t => {
    t.id()
    t.string("email", 255).unique()
    t.string("name", 100)
    t.string("password_hash", 255)
    t.boolean("is_active").default(true)
    t.timestamps()
  })
  m.create_table("auth_roles_auth_users", t => {
    t.uuid("user_id")
    t.uuid("role_id")
  })
  m.create_index("auth_roles_auth_users", ["user_id"], {name: "idx_auth_roles_auth_users_user_id"})
  m.create_index("auth_roles_auth_users", ["role_id"], {name: "idx_auth_roles_auth_users_role_id"})
  m.create_index("auth_roles_auth_users", ["user_id", "role_id"], {unique: true, name: "idx_auth_roles_auth_users_unique"})
})

// Rollback operations (for reference):
// migration(m => {
//   m.drop_table("auth_roles_auth_users")
//   m.drop_table("auth.users")
//   m.drop_table("auth.roles")
// })
