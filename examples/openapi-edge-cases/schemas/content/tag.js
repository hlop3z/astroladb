// Simple tag for M2M with posts
export default table({
  id: col.id(),
  name: col.name().unique(),
  slug: col.slug(),
  color: col.string(7).optional().docs("Hex color code like #FF5733"),
}).timestamps()
