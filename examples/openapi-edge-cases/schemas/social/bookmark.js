// Edge case: Another polymorphic + unique constraint
// Users can bookmark posts or comments
export default table({
  id: col.id(),
  user: col.belongs_to("auth.user"),
  // Polymorphic: what was bookmarked
  bookmarkable: col.belongs_to_any(["content.post", "content.comment"]),
  note: col.summary().optional().docs("Optional user note about the bookmark"),
}).timestamps().unique("user", "bookmarkable_type", "bookmarkable_id")
