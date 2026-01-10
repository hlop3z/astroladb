// Simple lookup table for M2M relationship
export default table({
  id: col.id(),
  name: col.name().unique(),
  description: col.summary().optional(),
  permissions: col.json().default({}),
  demo: col.string(100),
}).timestamps();
