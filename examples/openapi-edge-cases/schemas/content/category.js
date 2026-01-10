// Edge case: Self-referential hierarchy (parent category)
export default table({
  id: col.id(),
  name: col.name(),
  slug: col.slug(),
  // Self-referential: parent category (nullable for root categories)
  parent: col.belongs_to("content.category").optional().on_delete("cascade"),
  sort_order: col.counter(),
  is_active: col.flag(true),
}).timestamps()
