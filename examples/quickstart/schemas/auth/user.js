// Example: User table with semantic types
// Try typing "col." to see autocomplete suggestions!

export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  is_active: col.flag(true),
  is_verified: col.flag(),
  last_login: col.datetime().optional(),
}).timestamps();
