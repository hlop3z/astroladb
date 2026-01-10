// Example: Blog post with relationships
// Notice how belongs_to, timestamps, and other methods autocomplete!

export default table({
  id: col.id(),
  title: col.title(),
  slug: col.slug(),
  content: col.body(),
  excerpt: col.summary().optional(),
  is_published: col.flag(),
  published_at: col.datetime().optional(),
  author: col.belongs_to("auth.user"),
}).timestamps().index("published_at")
