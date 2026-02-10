/**
 * Test fixture: Blog tag schema
 * Used by both Go and JavaScript tests
 */

export default table({
  id: col.id(),
  name: col.string(50).unique(),
  slug: col.slug().unique(),
}).timestamps();
