// Schema: auth.role
// Role table for RBAC

export default table({
  id: col.id(),
  name: col.string(50).unique(),
  description: col.string(255).optional(),
}).timestamps()
