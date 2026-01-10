// Schema: auth.user
// User table (singular naming convention)

export default table({
  id: col.id(),
  email: col.string(255).unique(),
  name: col.string(100),
}).timestamps()
