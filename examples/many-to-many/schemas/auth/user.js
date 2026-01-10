// Schema: auth.user
// User table with many-to-many relationship to roles

export default table({
  id: col.id(),
  email: col.string(255).unique(),
  name: col.string(100),
  password_hash: col.string(255),
  is_active: col.boolean().default(true),
}).timestamps().many_to_many("auth.role")
