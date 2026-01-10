// Edge case: Polymorphic relationship (reaction can be on post OR comment)
export default table({
  id: col.id(),
  // Who reacted
  user: col.belongs_to("auth.user"),
  // Polymorphic: what was reacted to (post or comment)
  reactable: col.belongs_to_any(["content.post", "content.comment"]),
  kind: col.enum(["like", "love", "laugh", "sad", "angry"]).default("like"),
})
  .timestamps()
  .unique("user", "reactable_type", "reactable_id", "type");
