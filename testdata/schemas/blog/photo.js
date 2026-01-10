/// <reference types="../../../types" />

export default table({
  id: col.id(),
  user: col.belongs_to("auth.user"),
  url: col.url(),
  caption: col.summary().optional(),
}).timestamps()
