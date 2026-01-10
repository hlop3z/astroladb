/// <reference types="../../../types" />

export default table({
  id: col.id(),
  user: col.one_to_one("auth.user"),
  bio: col.text().optional(),
  avatar_url: col.url().optional(),
  birth_date: col.date().optional(),
}).timestamps()
