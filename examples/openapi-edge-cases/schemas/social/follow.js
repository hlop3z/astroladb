// Edge case: Self-referential M2M pattern with unique()
// User A follows User B (asymmetric relationship)
export default table({
  id: col.id(),
  // The user who is following
  follower: col.belongs_to("auth.user"),
  // The user being followed
  following: col.belongs_to("auth.user"),
  followed_at: col.datetime().default(sql("NOW()")),
  notifications_enabled: col.flag(true),
}).timestamps().unique("follower", "following")
