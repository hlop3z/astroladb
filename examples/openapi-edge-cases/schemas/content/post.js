// Edge case: Multiple FKs to same table + M2M
export default table({
  id: col.id(),
  // Author of the post
  author: col.belongs_to("auth.user"),
  // Editor who last modified (nullable, different from author)
  editor: col.belongs_to("auth.user").optional(),
  // Category (optional)
  category: col.belongs_to("content.category").optional(),
  title: col.title(),
  slug: col.slug(),
  content: col.body(),
  excerpt: col.summary().optional(),
  status: col.enum(["draft", "review", "published", "archived"]).default("draft"),
  is_featured: col.flag(),
  allow_comments: col.flag(true),
  view_count: col.counter(),
  published_at: col.datetime().optional(),
}).timestamps().many_to_many("content.tag")
