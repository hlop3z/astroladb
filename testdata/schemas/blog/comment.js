/// <reference types="../../../types" />

export default table({
  id: col.id(),
  user: col.belongs_to("auth.user"),
  post: col.belongs_to("blog.post"),
  body: col.text(),
}).timestamps()
