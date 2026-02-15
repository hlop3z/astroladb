// Comprehensive test schema for Phase 1 metadata export validation
// Tests: Min/Max, Pattern, Format, Docs, Deprecated, read_only, write_only

export default table({
  // Primary key (automatic read_only)
  id: col.id(),

  // Numeric constraints (Phase 0 fix - float64)
  age: col.integer().min(0).max(150).docs("User age in years"),
  price: col.decimal(10, 2).min(0).max(99999.99).docs("Product price"),
  rating: col.float().min(0.0).max(5.0).docs("User rating"),

  // String constraints
  username: col.string(50)
    .min(3)
    .max(50)
    .pattern("^[a-zA-Z0-9_]+$")
    .docs("Unique username, alphanumeric only")
    .unique(),

  email: col.string(255)
    .format("email")
    .pattern("^[\\w.-]+@[\\w.-]+\\.\\w+$")
    .docs("Primary email address")
    .unique(),

  website: col.string(500)
    .format("uri")
    .docs("Personal website URL")
    .optional(),

  // Deprecated fields
  old_username: col.string(50)
    .deprecated("Use username instead. Will be removed in v2.0")
    .optional(),

  legacy_field: col.integer()
    .deprecated("No longer used. Migrate to new_field")
    .optional(),

  // Explicit read_only
  server_timestamp: col.datetime()
    .read_only()
    .docs("Server-generated timestamp, cannot be modified by client"),

  computed_hash: col.string(64)
    .read_only()
    .docs("Server-computed hash value"),

  // Explicit write_only
  api_secret: col.string(128)
    .write_only()
    .docs("API secret key, never returned in responses"),

  encryption_key: col.string(256)
    .write_only()
    .docs("Encryption key, write-only for security"),

  // Automatic write_only detection
  password: col.string(255).docs("User password (automatic write-only)"),
  api_token: col.string(128).docs("API token (automatic write-only)"),
  access_key: col.string(64).docs("Access key (automatic write-only)"),

  // Automatic read_only detection
  created_at: col.datetime().docs("Creation timestamp (automatic read-only)"),
  updated_at: col.datetime().docs("Last update timestamp (automatic read-only)"),

  // All validation metadata together
  phone: col.string(20)
    .pattern("^\\+?[1-9]\\d{1,14}$")
    .format("phone")
    .min(10)
    .max(20)
    .docs("International phone number in E.164 format")
    .optional(),

  // Normal fields for comparison
  first_name: col.string(100).docs("First name"),
  last_name: col.string(100).docs("Last name"),
  bio: col.text().docs("User biography").optional(),
})
  .docs("Comprehensive test table for Phase 1 metadata validation")
  .deprecated("This is a test table. Use production tables instead.")
  .searchable(["username", "email", "first_name", "last_name"])
  .filterable(["age", "rating", "created_at"])
  .sort_by(["created_at", "-rating"]);
