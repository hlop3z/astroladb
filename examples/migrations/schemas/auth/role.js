// Schema: auth.role
// Role table with many-to-many to user

export default table({
  id: col.id(),
  name: col.string(50).unique(),
}).timestamps().many_to_many("auth.user")
