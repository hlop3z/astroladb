// schemas/auth/role.js
export default table({
  id: col.id(),
  name: col.name().unique(),
}).timestamps().many_to_many("auth.user");
