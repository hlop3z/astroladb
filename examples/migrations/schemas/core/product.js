// Schema: core.product
// Product table demonstrating all column types

export default table({
  id: col.id(),
  name: col.string(200),
  description: col.text().optional(),
  price: col.decimal(10, 2),
  stock_count: col.integer().default(0),
  is_active: col.boolean().default(true),
  status: col.enum(["draft", "published", "archived"]).default("draft"),
  metadata: col.json().optional(),
}).timestamps()
