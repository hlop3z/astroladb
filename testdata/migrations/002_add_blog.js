/// <reference types="../../types" />

export default migration(m => {
  m.create_table("blog.posts", t => {
    t.id()
    t.belongs_to("auth.users")
    t.string("title", 200)
    t.text("body")
    t.timestamps()
  })
})
