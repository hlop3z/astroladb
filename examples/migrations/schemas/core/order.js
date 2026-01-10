// Schema: core.order
// Order table with belongs_to relationship

export default table({
  id: col.id(),
  user: col.belongs_to("auth.user"), // Creates user_id column with auto-index
  total: col.decimal(10, 2),
}).timestamps()
