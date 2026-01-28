// schemas/blog/tag.js
export default table({
  id: col.id(),
  name: col.name().unique(),
  url_slug: col.slug().unique(),
})
  .timestamps()
  .sortable()
  .many_to_many("blog.post");
