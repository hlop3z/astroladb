// schemas/auth/role.js
export default table({
  id: col.id(),
  name: col.name().unique(),
  description: col.text().optional(),
})
  .timestamps()
  .many_to_many("auth.user");
