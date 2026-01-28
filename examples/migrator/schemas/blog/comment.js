// schemas/blog/comment.js
export default table({
  id: col.id(),
  post: col.belongs_to("blog.post"),
  commenter: col.belongs_to("auth.user"),
  body: col.body(),
  is_approved: col.flag(false),
})
  .timestamps()
  .soft_delete();
