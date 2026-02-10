/**
 * Test fixture: Authentication schema
 * Used by both Go and JavaScript tests
 */

export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  role: col.enum(['admin', 'editor', 'viewer']).default('viewer'),
  is_active: col.flag(true),
}).timestamps();
