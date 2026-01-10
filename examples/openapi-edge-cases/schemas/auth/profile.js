// Edge case: One-to-one relationship
export default table({
  id: col.id(),
  // One-to-one: each user has exactly one profile
  user: col.one_to_one("auth.user"),
  display_name: col.name().optional(),
  bio: col.text().optional(),
  avatar_url: col.url().optional(),
  website: col.url().optional(),
  birth_date: col.date().optional(),
  visibility: col.enum(["public", "private", "friends"]).default("public"),
}).timestamps()
