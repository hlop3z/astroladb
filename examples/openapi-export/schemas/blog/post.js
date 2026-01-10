// Schema: blog.post
// Blog post with author relationship

export default table({
  id: col.id(),
  author: col.belongs_to("blog.user"), // author_id -> blog.user
  title: col.title(),
  slug: col.slug(),
  content: col.body(),
  status: col.enum(["draft", "published", "archived"]).default("draft"),
  featured: col.flag(),
}).timestamps()
