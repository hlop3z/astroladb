/**
 * Test fixture: Blog schema
 * Used by both Go and JavaScript tests
 */

export const post = table({
  id: col.id(),
  title: col.title(),
  slug: col.slug().unique(),
  body: col.text(),
  author_id: col.belongs_to('auth.user'),
  status: col.enum(['draft', 'published', 'archived']).default('draft'),
  published_at: col.datetime().optional(),
}).timestamps().soft_delete();

export const comment = table({
  id: col.id(),
  body: col.text(),
  post_id: col.belongs_to('.post'),
  author_id: col.belongs_to('auth.user'),
}).timestamps();

export const tag = table({
  id: col.id(),
  name: col.string(50).unique(),
  slug: col.slug().unique(),
}).timestamps();
