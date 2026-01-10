/// <reference types="../../../types" />

export default table({
  id: col.id(),
  author: col.belongs_to("auth.user"),
  title: col.title(),
  slug: col.slug(),
  body: col.body(),
  status: col.enum(["draft", "published", "archived"]).default("draft"),
  published_at: col.datetime().optional(),
}).soft_delete().timestamps()
