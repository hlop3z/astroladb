// Schema: core.post
// Test post table with foreign key relationship

export default table({
  id: col.id(),
  author: col.belongs_to("core.user"),
  title: col.title(),
  content: col.body(),
  published: col.flag(),
}).timestamps()
