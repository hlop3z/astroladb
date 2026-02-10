/**
 * Test fixture: Blog post schema
 * Used by both Go and JavaScript tests
 */

export default table({
  id: col.id(),
  title: col.title(),
  slug: col.slug().unique(),
  body: col.text(),
  author_id: col.belongs_to('auth.user'),
  status: col.enum(['draft', 'published', 'archived']).default('draft'),
  published_at: col.datetime().optional(),
}).timestamps().soft_delete();
