// schemas/auth/user.js
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  is_active: col.flag(true),
  phone: col.phone().optional(),
  locale: col.string(10).default("en"),
}).timestamps();
