/// <reference types="../../../types" />

export default table({
  id: col.id(),
  name: col.string(100).unique(),
  description: col.string(500).optional(),
}).timestamps()
