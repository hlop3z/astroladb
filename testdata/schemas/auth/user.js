/// <reference types="../../../types" />

export default table({
  id: col.id(),
  email: col.email().unique(),
  password_hash: col.password_hash(),
  username: col.username().unique(),
  is_active: col.flag(true),
  preferences: col.json().optional(),
}).timestamps()
