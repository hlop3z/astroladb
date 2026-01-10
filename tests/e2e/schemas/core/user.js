// Schema: core.user
// Test user table with all JS-safe types

export default table({
  id: col.id(),
  username: col.username().unique(),
  email: col.email().unique(),
  bio: col.text().optional(),
  age: col.integer(),
  score: col.float(),
  balance: col.decimal(10, 2),
  active: col.flag(true),
  birth_date: col.date(),
  login_time: col.time(),
  last_seen: col.datetime(),
  settings: col.json().optional(),
  role: col.enum(["admin", "user", "guest"]).default("user"),
}).timestamps()
