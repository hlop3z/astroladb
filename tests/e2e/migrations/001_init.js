// Migration: init
// Generated at: 2026-01-07T16:23:30-06:00

migration(m => {
  m.create_table("core.user", t => {
    t.id()
    t.username("username").unique()
    t.email("email").unique()
    t.text("bio").optional()
    t.integer("age")
    t.float("score")
    t.decimal("balance", 10, 2)
    t.flag("active", true)
    t.date("birth_date")
    t.time("login_time")
    t.datetime("last_seen")
    t.json("settings").optional()
    t.enum("role", ["admin", "user", "guest"]).default("user")
    t.timestamps()
  })
  m.create_table("core.post", t => {
    t.id()
    t.uuid("author_id")
    t.title("title")
    t.body("content")
    t.flag("published")
    t.timestamps()
  })
  m.create_index("core.post", ["author_id"])
})

// Rollback operations (for reference):
// migration(m => {
//   m.drop_table("core.post")
//   m.drop_table("core.user")
// })
