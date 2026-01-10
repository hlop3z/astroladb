// Schema: blog.comment
// Comment on a post by a user

export default table({
  id: col.id(),
  post: col.belongs_to("blog.post"),    // post_id -> blog.post
  author: col.belongs_to("blog.user"),  // author_id -> blog.user
  body: col.body(),
}).timestamps()
