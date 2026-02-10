/**
 * Test fixture: Blog comment schema
 * Used by both Go and JavaScript tests
 */

export default table({
  id: col.id(),
  body: col.text(),
  post_id: col.belongs_to('blog.post'),
  author_id: col.belongs_to('auth.user'),
}).timestamps();
