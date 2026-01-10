// Edge case: Self-referential FK (referral system)
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  // Self-referential: user who referred this user (nullable)
  referred_by: col.belongs_to("auth.user").optional().on_delete("set null"),
  is_active: col.flag(true),
  is_verified: col.flag(),
  last_login: col.datetime().optional(),
}).timestamps().many_to_many("auth.role")
