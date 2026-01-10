/// <reference types="../../types" />

export default migration(m => {
  m.create_table("auth.users", t => {
    t.id()
    t.string("email", 255).unique()
    t.string("password_hash", 255)
    t.boolean("is_active").default(true)
    t.timestamps()
  })

  m.create_table("auth.roles", t => {
    t.id()
    t.string("name", 100).unique()
    t.timestamps()
  })
})
