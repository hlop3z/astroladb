// Edge case: Self-referential (threaded comments) + multiple FKs
export default table({
  id: col.id(),
  // Comment on a post
  post: col.belongs_to("content.post").on_delete("cascade"),
  // Author of the comment
  author: col.belongs_to("auth.user"),
  // Self-referential: reply to another comment (threaded)
  reply_to: col.belongs_to("content.comment").optional().on_delete("cascade"),
  body: col.body(),
  is_edited: col.flag(),
  is_hidden: col.flag(),
}).timestamps()
