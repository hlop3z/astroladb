// Schema: social.follow
// Follower relationship table demonstrating:
// - belongs_to with object keys for self-referential relationships
// - unique() for composite uniqueness

export default table({
  id: col.id(),
  // Self-referential: both point to auth.user but with different names
  follower: col.belongs_to("auth.user"),   // Creates follower_id
  following: col.belongs_to("auth.user"),  // Creates following_id
}).timestamps().unique("follower", "following")
