// Example: Comment with multiple relationships

export default table({
  id: col.id(),
  content: col.body(),
  post: col.belongs_to("blog.post"),
  author: col.belongs_to("auth.user"),
  parent: col.belongs_to("blog.comment").optional(),
  is_approved: col.flag(),
}).timestamps()
