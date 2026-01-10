// Example: Initial migration
// In real usage, run 'alab new <name>' to auto-generate migrations from schema changes

migration(m => {
  // Create users table
  m.create_table("auth.user", t => {
    t.id()
    t.email("email").unique()
    t.username("username").unique()
    t.password_hash("password")
    t.flag("is_active", true)
    t.flag("is_verified")
    t.datetime("last_login").optional()
    t.timestamps()
  })

  // Create posts table
  m.create_table("blog.post", t => {
    t.id()
    t.title("title")
    t.slug("slug")
    t.body("content")
    t.summary("excerpt").optional()
    t.flag("is_published")
    t.datetime("published_at").optional()
    t.belongs_to("auth.user").as("author")
    t.timestamps()
    t.index("published_at")
  })

  // Create comments table
  m.create_table("blog.comment", t => {
    t.id()
    t.body("content")
    t.belongs_to("blog.post")
    t.belongs_to("auth.user").as("author")
    t.belongs_to("blog.comment").as("parent").optional()
    t.flag("is_approved")
    t.timestamps()
  })
}, { description: "Initial schema setup" })
