// schemas/auth/profile.js
export default table({
  id: col.id(),
  user: col.one_to_one("auth.user"),
  display_name: col.name().optional(),
  bio: col.body().optional(),
  avatar_url: col.url().optional(),
  birth_date: col.date().optional(),
}).timestamps();
