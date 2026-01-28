// schemas/blog/post.js
export default table({
  id: col.id(),
  author: col.belongs_to("auth.user"),
  title: col.title(),
  slug: col.slug().unique(),
  body: col.body(),
  status: col.enum(["draft", "published", "archived"]).default("draft"),
  published_at: col.datetime().optional(),
})
  .timestamps()
  .soft_delete()
  .searchable(["title", "body"]);