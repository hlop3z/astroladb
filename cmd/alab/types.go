package main

import (
	"os"
	"path/filepath"
)

// writeTypeDefinitions creates the TypeScript definition files for IDE autocomplete.
// These are always overwritten to ensure they stay in sync.
func writeTypeDefinitions() error {
	typesDir := "types"
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		return err
	}

	files := map[string]string{
		"index.d.ts":     typeIndexDTS,
		"globals.d.ts":   typeGlobalsDTS,
		"column.d.ts":    typeColumnDTS,
		"schema.d.ts":    typeSchemaDTS,
		"migration.d.ts": typeMigrationDTS,
	}

	for name, content := range files {
		path := filepath.Join(typesDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Write jsconfig.json in project root for IDE support
	if err := os.WriteFile("jsconfig.json", []byte(typeJSConfig), 0644); err != nil {
		return err
	}

	return nil
}

const typeJSConfig = `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ES2020",
    "moduleResolution": "node",
    "checkJs": false,
    "strict": false,
    "noEmit": true
  },
  "include": [
    "types/**/*.d.ts",
    "schemas/**/*.js",
    "migrations/**/*.js"
  ],
  "exclude": [
    "node_modules",
    ".alab"
  ]
}
`

// Type definition file contents (auto-generated, do not edit)

const typeIndexDTS = `/**
 * Alab Type Definitions
 *
 * TypeScript definitions for Alab - a language-agnostic database migration tool.
 * These types provide IDE autocomplete and type checking without runtime cost.
 *
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

// Re-export all types
export * from "./globals";
export * from "./column";
export * from "./schema";
export * from "./migration";

// Declare globals for ambient usage in .js files
import { SQLExpr } from "./globals";
import { TableBuilder, ColBuilder } from "./column";
import { SchemaBuilder, TableDefinition, TableChain, ColumnDefinitions } from "./schema";
import { MigrationBuilder, MigrationOptions } from "./migration";

declare global {
  /** Wraps a string as a raw SQL expression. */
  function sql(expr: string): SQLExpr;

  /**
   * Column factory for the object-based table API.
   * Use ` + "`" + `col.*` + "`" + ` methods to define columns. Column names come from the object keys.
   *
   * @example
   * export default table({
   *   id: col.id(),
   *   email: col.email().unique(),
   *   author: col.belongs_to("auth.user"),
   * }).timestamps()
   */
  const col: ColBuilder;

  /** Defines a schema namespace with multiple tables. */
  function schema(namespace: string, fn: (s: SchemaBuilder) => void): void;

  /** Defines a single table using a callback (legacy API). */
  function table(fn: (t: TableBuilder) => void): TableDefinition;

  /** Defines a single table using an object (preferred API). */
  function table(columns: ColumnDefinitions): TableChain;

  /** Defines a migration with schema change operations. */
  function migration(fn: (m: MigrationBuilder) => void, opts?: MigrationOptions): void;
}
`

const typeGlobalsDTS = `/**
 * Alab Global Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { ColBuilder } from "./column";

/**
 * Represents a raw SQL expression.
 *
 * Created by the sql() helper function. Used for database-level
 * defaults and computations that can't be expressed as static values.
 */
export interface SQLExpr {
  readonly __sql: true;
  readonly expr: string;
}

/**
 * Wraps a string as a raw SQL expression.
 *
 * Use when you need database-side computation for defaults or backfills.
 * The expression is passed directly to the database without escaping.
 *
 * Common use cases:
 * - Current timestamp: sql("NOW()") or sql("CURRENT_TIMESTAMP")
 * - UUID generation: sql("gen_random_uuid()")
 * - Computed backfills: sql("LOWER(existing_column)")
 *
 * @param expr - Raw SQL expression string
 * @returns SQLExpr object recognized by Alab
 *
 * @example
 * // Auto-generate UUID for new column (object API)
 * { public_id: col.uuid().default(sql("gen_random_uuid()")) }
 *
 * @example
 * // Timestamp defaults
 * { expires_at: col.datetime().default(sql("NOW() + INTERVAL '30 days'")) }
 *
 * @example
 * // Backfill from existing data
 * { slug: col.string(200).backfill(sql("LOWER(REPLACE(title, ' ', '-'))")) }
 */
declare function sql(expr: string): SQLExpr;

/**
 * Column factory for the object-based table API.
 *
 * Use ` + "`" + `col.*` + "`" + ` methods to define columns. Column names come from the object keys.
 *
 * @example
 * export default table({
 *   id: col.id(),
 *   email: col.email().unique(),
 *   username: col.username().unique(),
 *   password: col.password_hash(),
 *   author: col.belongs_to("auth.user"),
 *   is_active: col.flag(true),
 * }).timestamps()
 */
declare const col: ColBuilder;

export { sql, col };
`

const typeColumnDTS = `/**
 * Alab Column Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { SQLExpr } from "./globals";

/**
 * Fluent builder for relationship definitions.
 * Returned by belongs_to() and one_to_one() for chaining options.
 */
export interface RelationshipBuilder {
  /**
   * Sets a custom alias for the foreign key column.
   *
   * By default, belongs_to("auth.user") creates a column named "user_id".
   * Use .as() to customize this when you have multiple FKs to the same table.
   *
   * @param alias - The alias name (without _id suffix)
   *
   * @example
   * // Creates "author_id" instead of "user_id"
   * t.belongs_to("auth.user").as("author")
   *
   * @example
   * // Self-referential: follower_id and following_id
   * t.belongs_to("auth.user").as("follower")
   * t.belongs_to("auth.user").as("following")
   */
  as(alias: string): RelationshipBuilder;

  /**
   * Makes the foreign key nullable (allows NULL).
   *
   * By default, belongs_to() creates a NOT NULL column. Use .optional()
   * when the relationship is not required.
   *
   * @example
   * // User may or may not have a manager
   * t.belongs_to("auth.user").as("manager").optional()
   *
   * @example
   * // Category can be unassigned
   * t.belongs_to("catalog.category").optional()
   */
  optional(): RelationshipBuilder;

  /**
   * Makes the foreign key nullable. Alias for optional().
   */
  nullable(): RelationshipBuilder;

  /**
   * Sets the ON DELETE action for referential integrity.
   *
   * Controls what happens to this row when the referenced row is deleted.
   *
   * @param action - One of:
   *   - "cascade" - Delete this row too
   *   - "set null" - Set FK to NULL (requires .optional())
   *   - "restrict" - Prevent deletion (default)
   *   - "no action" - Check at end of transaction
   *   - "set default" - Set FK to default value
   *
   * @example
   * // Delete all posts when user is deleted
   * t.belongs_to("auth.user").as("author").on_delete("cascade")
   *
   * @example
   * // Keep posts but clear the author
   * t.belongs_to("auth.user").as("author").optional().on_delete("set null")
   */
  on_delete(action: string): RelationshipBuilder;

  /**
   * Sets the ON UPDATE action for referential integrity.
   *
   * Controls what happens when the referenced row's primary key changes.
   * Rarely needed since UUIDs don't change.
   *
   * @param action - Same options as on_delete()
   */
  on_update(action: string): RelationshipBuilder;

  /**
   * Adds documentation for the relationship.
   *
   * Becomes a SQL COMMENT and appears in generated OpenAPI schemas.
   *
   * @example
   * t.belongs_to("auth.user").as("reviewer").docs("User who approved this item")
   */
  docs(description: string): RelationshipBuilder;

  /**
   * Marks the relationship as deprecated.
   *
   * Shows deprecation warning in OpenAPI schemas.
   *
   * @example
   * t.belongs_to("legacy.account").deprecated("Use auth.user instead")
   */
  deprecated(reason: string): RelationshipBuilder;
}

/**
 * Options for many-to-many relationships.
 */
export interface M2MOptions {
  /**
   * Custom name for the join table.
   *
   * By default, Alab generates a name like "post_tag".
   * Use this to specify your own join table name.
   */
  through?: string;
}

/**
 * Options for polymorphic (belongs_to_any) relationships.
 */
export interface PolymorphicOptions {
  /**
   * Base name for the type/id column pair.
   *
   * Creates {as}_type (string) and {as}_id (uuid) columns.
   * Defaults to "parent" if not specified.
   */
  as?: string;
}

/**
 * Fluent builder for column definitions.
 * All methods return the builder for chaining.
 */
export interface ColumnBuilder {
  /**
   * Allows NULL values in this column.
   *
   * By default, all columns are NOT NULL. Use .optional() when a value
   * isn't always required. Prefer .optional() over .nullable() for clarity.
   *
   * @example
   * t.datetime("deleted_at").optional()
   * t.string("middle_name", 50).optional()
   */
  nullable(): ColumnBuilder;

  /**
   * Allows NULL values in this column. Preferred alias for nullable().
   *
   * Use when a column value isn't always required.
   *
   * @example
   * t.datetime("last_login").optional()
   * t.text("bio").optional()
   */
  optional(): ColumnBuilder;

  /**
   * Adds a unique constraint to this column.
   *
   * Ensures no two rows can have the same value. Unique columns are
   * automatically indexed for fast lookups.
   *
   * @example
   * t.email("email").unique()
   * t.string("api_key", 64).unique()
   */
  unique(): ColumnBuilder;

  /**
   * Sets the default value for new rows.
   *
   * This is a permanent default that applies to all future INSERT statements.
   * For SQL expressions, use sql() helper.
   *
   * @param value - Static value or sql() expression
   *
   * @example
   * t.integer("view_count").default(0)
   * t.string("status", 20).default("pending")
   * t.datetime("created_at").default(sql("NOW()"))
   * t.uuid("public_id").default(sql("gen_random_uuid()"))
   */
  default(value: any): ColumnBuilder;

  /**
   * Sets a one-time value for existing rows during migration.
   *
   * Use when adding a NOT NULL column to a table that already has data.
   * Unlike default(), this only runs once during migration execution.
   *
   * @param value - Static value or sql() expression
   *
   * @example
   * // Fill existing rows with 0, new rows get the column default
   * t.integer("score").default(0).backfill(0)
   *
   * @example
   * // Compute from another column
   * t.string("slug", 200).unique().backfill(sql("LOWER(REPLACE(title, ' ', '-'))"))
   */
  backfill(value: any): ColumnBuilder;

  /**
   * Sets the minimum constraint for the column.
   *
   * For strings: minimum character length.
   * For numbers: minimum value.
   * Generates CHECK constraint and OpenAPI minimum/minLength.
   *
   * @param n - Minimum value
   *
   * @example
   * t.string("username", 50).min(3)  // At least 3 characters
   * t.integer("quantity").min(1)      // At least 1
   * t.decimal("price", 10, 2).min(0)  // No negative prices
   */
  min(n: number): ColumnBuilder;

  /**
   * Sets the maximum constraint for the column.
   *
   * For strings: maximum character length (usually set by type).
   * For numbers: maximum value.
   * Generates CHECK constraint and OpenAPI maximum/maxLength.
   *
   * @param n - Maximum value
   *
   * @example
   * t.integer("rating").min(1).max(5)  // 1-5 star rating
   * t.decimal("discount", 5, 2).max(100)  // Up to 100%
   */
  max(n: number): ColumnBuilder;

  /**
   * Sets a regex pattern for validation.
   *
   * Generates a CHECK constraint (if supported) and OpenAPI pattern.
   * Use for format validation that isn't covered by semantic types.
   *
   * @param regex - JavaScript regex pattern (without delimiters)
   *
   * @example
   * t.string("zip_code", 10).pattern("^\\d{5}(-\\d{4})?$")
   * t.string("sku", 20).pattern("^[A-Z]{2}-\\d{6}$")
   */
  pattern(regex: string): ColumnBuilder;

  /**
   * Sets the OpenAPI format hint for the column.
   *
   * Standard formats: "email", "uri", "uuid", "date", "time", "date-time",
   * "hostname", "ipv4", "ipv6", "password".
   *
   * Note: Prefer semantic types (t.email, t.url) which set format automatically.
   *
   * @param format - OpenAPI format string
   *
   * @example
   * t.string("callback", 2048).format("uri")
   */
  format(format: string): ColumnBuilder;

  /**
   * Adds documentation for the column.
   *
   * Becomes a SQL COMMENT on the column and the "description" field
   * in generated OpenAPI schemas.
   *
   * @param description - Human-readable description
   *
   * @example
   * t.integer("retry_count").default(0).docs("Number of failed delivery attempts")
   * t.datetime("embargo_until").optional().docs("Content hidden until this date")
   */
  docs(description: string): ColumnBuilder;

  /**
   * Marks the column as deprecated.
   *
   * Adds "deprecated: true" to OpenAPI schema. Use when phasing out a column.
   *
   * @param reason - Explanation and migration guidance
   *
   * @example
   * t.string("old_status", 20).deprecated("Use status_id FK instead")
   */
  deprecated(reason: string): ColumnBuilder;

  /**
   * Sets a foreign key reference. Usually used internally by relationships.
   *
   * Prefer belongs_to() for declaring relationships.
   *
   * @param ref - Reference in format "namespace.table"
   */
  references(ref: string): ColumnBuilder;
}

/**
 * Fluent builder for table definitions.
 * Use inside table() or migration create_table() callbacks.
 */
export interface TableBuilder {
  // ===========================================================================
  // PRIMARY KEY
  // ===========================================================================

  /**
   * Adds a UUID primary key column named "id".
   *
   * This is the ONLY way to define a primary key in Alab. Uses UUID v4 for
   * globally unique, non-sequential IDs that are safe in JavaScript.
   *
   * Every table should start with t.id().
   *
   * @example
   * table(t => {
   *   t.id()  // Required first line
   *   t.email("email").unique()
   *   // ...
   * })
   */
  id(): ColumnBuilder;

  // ===========================================================================
  // CORE TYPES - Use when semantic types don't fit your needs
  // ===========================================================================

  /**
   * Variable-length string with maximum length.
   *
   * Maps to VARCHAR(n). Use when you need a specific max length that isn't
   * covered by semantic types. Length is required to prevent unbounded columns.
   *
   * @param name - Column name in snake_case
   * @param length - Maximum character length
   *
   * @example
   * t.string("country_code", 2)   // ISO country code
   * t.string("license_plate", 10)
   * t.string("custom_field", 255)
   */
  string(name: string, length: number): ColumnBuilder;

  /**
   * Unlimited text content.
   *
   * Maps to TEXT. Use for user-generated content without length limits.
   * For bodies/content, prefer t.body() semantic type.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.text("notes")
   * t.text("raw_response")
   */
  text(name: string): ColumnBuilder;

  /**
   * 32-bit signed integer (-2B to +2B).
   *
   * Safe for JavaScript (no precision loss). Use for counts, quantities,
   * and numeric IDs from external systems.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.integer("view_count")
   * t.integer("external_id")
   * t.integer("sort_order")
   */
  integer(name: string): ColumnBuilder;

  /**
   * 32-bit floating-point number.
   *
   * Use for approximate values where precision isn't critical.
   * For money/financial data, use t.decimal() or t.money() instead.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.float("latitude")
   * t.float("longitude")
   * t.float("temperature")
   */
  float(name: string): ColumnBuilder;

  /**
   * Exact decimal number with fixed precision.
   *
   * Maps to DECIMAL(precision, scale). Use for money, percentages, and
   * anywhere exact decimal arithmetic matters. Serialized as string in JSON
   * to preserve precision.
   *
   * @param name - Column name in snake_case
   * @param precision - Total number of digits
   * @param scale - Digits after decimal point
   *
   * @example
   * t.decimal("price", 10, 2)      // Up to 99,999,999.99
   * t.decimal("tax_rate", 5, 4)    // e.g., 0.0825 for 8.25%
   * t.decimal("exchange_rate", 12, 6)
   */
  decimal(name: string, precision: number, scale: number): ColumnBuilder;

  /**
   * True/false boolean value.
   *
   * For flags with defaults, prefer t.flag() which includes a default value.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.boolean("agreed_to_terms")
   */
  boolean(name: string): ColumnBuilder;

  /**
   * Date without time (YYYY-MM-DD).
   *
   * Maps to DATE. Use for birthdays, deadlines, and calendar dates.
   * Serialized as ISO 8601 string.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.date("birth_date")
   * t.date("due_date")
   * t.date("start_date")
   */
  date(name: string): ColumnBuilder;

  /**
   * Time without date (HH:MM:SS).
   *
   * Maps to TIME. Use for recurring times, schedules.
   * Serialized as ISO 8601 string.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.time("opens_at")
   * t.time("reminder_time")
   */
  time(name: string): ColumnBuilder;

  /**
   * Timestamp with timezone.
   *
   * Maps to TIMESTAMP WITH TIME ZONE. Use for event times, created/updated.
   * Serialized as ISO 8601 string. For created_at/updated_at, use t.timestamps().
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.datetime("published_at").optional()
   * t.datetime("expires_at")
   * t.datetime("last_login").optional()
   */
  datetime(name: string): ColumnBuilder;

  /**
   * UUID column (non-primary key).
   *
   * Use for external references, public IDs, or correlation IDs.
   * For the primary key, use t.id() instead.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.uuid("correlation_id")
   * t.uuid("public_token").unique().default(sql("gen_random_uuid()"))
   */
  uuid(name: string): ColumnBuilder;

  /**
   * JSON/JSONB column for structured data.
   *
   * Maps to JSONB (PostgreSQL) or JSON (SQLite). Use for flexible
   * schemas, preferences, metadata. Stored and retrieved as native JS objects.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.json("preferences").default({})
   * t.json("metadata").optional()
   * t.json("address")  // { street, city, zip, ... }
   */
  json(name: string): ColumnBuilder;

  /**
   * Binary data stored as base64.
   *
   * Maps to BYTEA/BLOB. Use for small binary data like hashes, thumbnails.
   * For files, use external storage with a URL reference instead.
   *
   * @param name - Column name in snake_case
   *
   * @example
   * t.base64("avatar_thumbnail")
   * t.base64("checksum")
   */
  base64(name: string): ColumnBuilder;

  /**
   * Enumerated type with fixed allowed values.
   *
   * Creates a check constraint ensuring only listed values are allowed.
   * Use for status fields, types, categories with known values.
   *
   * @param name - Column name in snake_case
   * @param values - Array of allowed string values
   *
   * @example
   * t.enum("status", ["draft", "published", "archived"])
   * t.enum("priority", ["low", "medium", "high", "urgent"])
   * t.enum("role", ["admin", "moderator", "member"])
   */
  enum(name: string, values: string[]): ColumnBuilder;

  // ===========================================================================
  // SEMANTIC TYPES - Preferred for common patterns
  // ===========================================================================

  /**
   * Email address: string(255) + email format + RFC 5322 pattern.
   *
   * Use for user emails, contact emails. Includes validation pattern
   * and OpenAPI format hint for proper UI rendering.
   *
   * @param name - Column name (typically "email")
   *
   * @example
   * t.email("email").unique()
   * t.email("contact_email").optional()
   */
  email(name: string): ColumnBuilder;

  /**
   * Username: string(50) + alphanumeric pattern + min(3).
   *
   * Use for user handles, login names. Enforces reasonable length
   * and character restrictions.
   *
   * @param name - Column name (typically "username")
   *
   * @example
   * t.username("username").unique()
   * t.username("handle").unique()
   */
  username(name: string): ColumnBuilder;

  /**
   * Password hash: string(255) + hidden from OpenAPI.
   *
   * Use for storing bcrypt/argon2 hashes. Excluded from API responses
   * by default for security.
   *
   * @param name - Column name (typically "password" or "password_hash")
   *
   * @example
   * t.password_hash("password")
   */
  password_hash(name: string): ColumnBuilder;

  /**
   * Phone number: string(50) + E.164 pattern.
   *
   * Stores phone numbers in international format. Length accommodates
   * country codes and extensions.
   *
   * @param name - Column name
   *
   * @example
   * t.phone("phone").optional()
   * t.phone("mobile")
   */
  phone(name: string): ColumnBuilder;

  /**
   * Person/entity name: string(100).
   *
   * Use for first name, last name, or full name fields.
   *
   * @param name - Column name
   *
   * @example
   * t.name("first_name")
   * t.name("last_name")
   * t.name("display_name")
   */
  name(name: string): ColumnBuilder;

  /**
   * Title/headline: string(200).
   *
   * Use for post titles, product names, page headings.
   *
   * @param name - Column name (typically "title")
   *
   * @example
   * t.title("title")
   * t.title("headline")
   */
  title(name: string): ColumnBuilder;

  /**
   * URL slug: string(255) + unique + slug pattern.
   *
   * Use for URL-friendly identifiers. Automatically unique.
   * Pattern allows lowercase letters, numbers, and hyphens.
   *
   * @param name - Column name (typically "slug")
   *
   * @example
   * t.slug("slug")  // "my-awesome-post"
   */
  slug(name: string): ColumnBuilder;

  /**
   * Long-form content: text (unlimited).
   *
   * Use for article bodies, descriptions, comments. Maps to TEXT
   * for unlimited length.
   *
   * @param name - Column name (typically "content" or "body")
   *
   * @example
   * t.body("content")
   * t.body("description")
   */
  body(name: string): ColumnBuilder;

  /**
   * Short summary: string(500).
   *
   * Use for excerpts, meta descriptions, preview text.
   *
   * @param name - Column name
   *
   * @example
   * t.summary("excerpt").optional()
   * t.summary("meta_description").optional()
   */
  summary(name: string): ColumnBuilder;

  /**
   * URL/URI: string(2048) + uri format.
   *
   * Use for links, images, callbacks. 2048 chars accommodates
   * long query strings.
   *
   * @param name - Column name
   *
   * @example
   * t.url("website").optional()
   * t.url("avatar_url").optional()
   * t.url("callback_url")
   */
  url(name: string): ColumnBuilder;

  /**
   * IP address: string(45) for IPv4/IPv6.
   *
   * 45 chars fits the longest IPv6 representation.
   *
   * @param name - Column name
   *
   * @example
   * t.ip("ip_address")
   * t.ip("last_known_ip").optional()
   */
  ip(name: string): ColumnBuilder;

  /**
   * User agent string: string(500).
   *
   * Use for storing browser/client user agents for analytics.
   *
   * @param name - Column name
   *
   * @example
   * t.user_agent("user_agent").optional()
   */
  user_agent(name: string): ColumnBuilder;

  /**
   * Money amount: decimal(19,4) + min(0).
   *
   * Use for prices, totals, balances. 4 decimal places handle
   * currency subdivisions. Serialized as string to preserve precision.
   *
   * @param name - Column name
   *
   * @example
   * t.money("price")
   * t.money("total")
   * t.money("balance")
   */
  money(name: string): ColumnBuilder;

  /**
   * Percentage: decimal(5,2) + min(0) + max(100).
   *
   * Stores percentages as 0-100 values with 2 decimal precision.
   *
   * @param name - Column name
   *
   * @example
   * t.percentage("discount")
   * t.percentage("completion")
   * t.percentage("tax_rate")
   */
  percentage(name: string): ColumnBuilder;

  /**
   * Counter: integer + default(0).
   *
   * Use for incrementing counts. Starts at 0.
   *
   * @param name - Column name
   *
   * @example
   * t.counter("view_count")
   * t.counter("login_count")
   * t.counter("failed_attempts")
   */
  counter(name: string): ColumnBuilder;

  /**
   * Quantity: integer + min(0).
   *
   * Use for stock levels, cart quantities. Cannot be negative.
   *
   * @param name - Column name
   *
   * @example
   * t.quantity("stock")
   * t.quantity("quantity")
   */
  quantity(name: string): ColumnBuilder;

  /**
   * Boolean flag with default value.
   *
   * Use for feature flags, status booleans. Default is false unless specified.
   *
   * @param name - Column name (typically "is_*" or "has_*")
   * @param defaultValue - Default value (defaults to false)
   *
   * @example
   * t.flag("is_active", true)   // Default true
   * t.flag("is_verified")       // Default false
   * t.flag("has_accepted_terms")
   */
  flag(name: string, defaultValue?: boolean): ColumnBuilder;

  // ===========================================================================
  // IDENTIFIERS & TOKENS
  // ===========================================================================

  /**
   * Secure token: string(64) + unique.
   *
   * Use for API keys, session tokens, password reset tokens.
   * 64 chars fits a 256-bit hex-encoded token.
   *
   * @param name - Column name
   *
   * @example
   * t.token("api_key").unique()
   * t.token("reset_token").optional()
   * t.token("session_token")
   */
  token(name: string): ColumnBuilder;

  /**
   * Short code: string(20) + uppercase pattern.
   *
   * Use for SKUs, promo codes, reference numbers, vouchers.
   * Pattern allows uppercase letters, numbers, and hyphens.
   *
   * @param name - Column name
   *
   * @example
   * t.code("sku").unique()
   * t.code("promo_code")
   * t.code("reference_number")
   */
  code(name: string): ColumnBuilder;

  // ===========================================================================
  // GEOGRAPHIC & I18N
  // ===========================================================================

  /**
   * ISO 3166-1 alpha-2 country code: string(2).
   *
   * Two-letter country codes (US, GB, DE, JP, etc.).
   *
   * @param name - Column name
   *
   * @example
   * t.country("country")
   * t.country("shipping_country")
   * t.country("billing_country")
   */
  country(name: string): ColumnBuilder;

  /**
   * ISO 4217 currency code: string(3).
   *
   * Three-letter currency codes (USD, EUR, GBP, JPY, etc.).
   *
   * @param name - Column name
   *
   * @example
   * t.currency("currency")
   * t.currency("payout_currency")
   */
  currency(name: string): ColumnBuilder;

  /**
   * BCP 47 locale code: string(10).
   *
   * Language and region codes (en-US, es-MX, zh-CN, etc.).
   * Use for user language preferences and i18n.
   *
   * @param name - Column name
   *
   * @example
   * t.locale("locale")
   * t.locale("preferred_language")
   */
  locale(name: string): ColumnBuilder;

  /**
   * IANA timezone identifier: string(50).
   *
   * Timezone names like America/New_York, Europe/London, Asia/Tokyo.
   *
   * @param name - Column name
   *
   * @example
   * t.timezone("timezone")
   * t.timezone("user_timezone")
   */
  timezone(name: string): ColumnBuilder;

  // ===========================================================================
  // RATINGS & MEASUREMENTS
  // ===========================================================================

  /**
   * Star rating: decimal(2,1) + min(0) + max(5).
   *
   * Use for review ratings, scores. Allows half-stars (4.5).
   * Range is 0 to 5.
   *
   * @param name - Column name
   *
   * @example
   * t.rating("rating")
   * t.rating("average_rating")
   * t.rating("user_score")
   */
  rating(name: string): ColumnBuilder;

  /**
   * Duration in seconds: integer + min(0).
   *
   * Use for video length, session duration, time intervals.
   * Store in seconds, convert to minutes/hours in app.
   *
   * @param name - Column name
   *
   * @example
   * t.duration("duration")         // Video length
   * t.duration("session_length")   // Session time
   * t.duration("cooldown_period")  // Wait time
   */
  duration(name: string): ColumnBuilder;

  /**
   * Hex color code: string(7).
   *
   * Stores colors as #RRGGBB format (e.g., #FF5733).
   * Use for theme colors, user preferences.
   *
   * @param name - Column name
   *
   * @example
   * t.color("primary_color")
   * t.color("background_color")
   * t.color("brand_color")
   */
  color(name: string): ColumnBuilder;

  // ===========================================================================
  // CONTENT FORMATS
  // ===========================================================================

  /**
   * Markdown content: text + markdown format hint.
   *
   * Use for rich text that will be rendered as Markdown.
   * Maps to TEXT. OpenAPI format indicates markdown rendering.
   *
   * @param name - Column name
   *
   * @example
   * t.markdown("content")
   * t.markdown("description")
   * t.markdown("readme")
   */
  markdown(name: string): ColumnBuilder;

  /**
   * HTML content: text + html format hint.
   *
   * Use for pre-rendered HTML content.
   * Maps to TEXT. Be careful with XSS - sanitize on input.
   *
   * @param name - Column name
   *
   * @example
   * t.html("rendered_content")
   * t.html("email_body")
   */
  html(name: string): ColumnBuilder;

  // ===========================================================================
  // CONVENIENCE PATTERNS
  // ===========================================================================

  /**
   * Adds created_at and updated_at timestamp columns.
   *
   * Both columns default to NOW() and are NOT NULL.
   * Put near the end of your table definition.
   *
   * @example
   * table(t => {
   *   t.id()
   *   t.email("email").unique()
   *   t.timestamps()  // created_at, updated_at
   * })
   */
  timestamps(): void;

  /**
   * Adds a deleted_at nullable timestamp for soft deletion.
   *
   * NULL means not deleted. Set to NOW() to soft-delete.
   * Query with WHERE deleted_at IS NULL for active records.
   *
   * @example
   * table(t => {
   *   t.id()
   *   t.title("title")
   *   t.timestamps()
   *   t.soft_delete()  // deleted_at
   * })
   */
  soft_delete(): void;

  /**
   * Adds a position integer column for manual ordering.
   *
   * Defaults to 0. Use for drag-and-drop reordering.
   *
   * @example
   * table(t => {
   *   t.id()
   *   t.title("title")
   *   t.sortable()  // position column
   *   t.timestamps()
   * })
   */
  sortable(): void;

  /**
   * Adds a unique slug column derived from another column.
   *
   * Shorthand for t.slug("slug"). The sourceColumn is documentation
   * only - actual slug generation happens in your application.
   *
   * @param sourceColumn - Column that slugs are derived from
   *
   * @example
   * table(t => {
   *   t.id()
   *   t.title("title")
   *   t.slugged("title")  // Creates slug column
   * })
   */
  slugged(sourceColumn: string): void;

  // ===========================================================================
  // RELATIONSHIPS
  // ===========================================================================

  /**
   * Adds a foreign key column to another table.
   *
   * Creates a NOT NULL {table}_id column (or {alias}_id with .as()).
   * Automatically creates an index on the FK. Returns RelationshipBuilder
   * for chaining options.
   *
   * @param model - Reference in format "namespace.table"
   *
   * @example
   * // Simple FK - creates user_id column
   * t.belongs_to("auth.user")
   *
   * @example
   * // With alias - creates author_id column
   * t.belongs_to("auth.user").as("author")
   *
   * @example
   * // Optional with cascade delete
   * t.belongs_to("catalog.category").optional().on_delete("cascade")
   *
   * @example
   * // Self-referential for hierarchies
   * t.belongs_to("catalog.category").as("parent").optional()
   */
  belongs_to(model: string): RelationshipBuilder;

  /**
   * Adds a unique foreign key (one-to-one relationship).
   *
   * Like belongs_to() but with a unique constraint. Use when each
   * parent has at most one child (e.g., user has one profile).
   *
   * @param model - Reference in format "namespace.table"
   *
   * @example
   * // User has one profile
   * t.one_to_one("auth.user")
   *
   * @example
   * // Optional one-to-one with alias
   * t.one_to_one("billing.account").as("billing").optional()
   */
  one_to_one(model: string): RelationshipBuilder;

  /**
   * Declares a many-to-many relationship.
   *
   * Alab generates a join table with foreign keys to both tables.
   * The join table is created automatically during migration.
   *
   * @param model - Reference to the other table
   * @param opts - Optional: { through: "custom_table_name" }
   *
   * @example
   * // Post has many tags, Tag has many posts
   * // Creates post_tag join table
   * t.many_to_many("blog.tag")
   *
   * @example
   * // User has many roles with custom join table
   * t.many_to_many("auth.role", { through: "user_roles" })
   */
  many_to_many(model: string, opts?: M2MOptions): void;

  /**
   * Adds a polymorphic reference (type + id columns).
   *
   * Use when a row can belong to multiple different table types.
   * Creates {as}_type (string) and {as}_id (uuid) columns.
   *
   * @param models - Array of possible model references
   * @param opts - Optional: { as: "base_name" } (defaults to "parent")
   *
   * @example
   * // Comment can belong to Post or Video
   * // Creates commentable_type and commentable_id
   * t.belongs_to_any(["blog.post", "media.video"], { as: "commentable" })
   *
   * @example
   * // Reaction on posts or comments
   * t.belongs_to_any(["blog.post", "blog.comment"], { as: "reactable" })
   */
  belongs_to_any(models: string[], opts?: PolymorphicOptions): void;

  // ===========================================================================
  // INDEXES & CONSTRAINTS
  // ===========================================================================

  /**
   * Adds a non-unique index on columns.
   *
   * Use for columns frequently used in WHERE clauses or JOINs.
   * Foreign keys are automatically indexed; don't duplicate them.
   *
   * @param columns - Column names to index
   *
   * @example
   * t.index("created_at")
   * t.index("status", "created_at")  // Composite index
   */
  index(...columns: string[]): void;

  /**
   * Adds a composite unique constraint on multiple columns.
   *
   * Ensures the combination of values is unique across all rows.
   * Relationship aliases are automatically expanded to {alias}_id.
   *
   * @param columns - Column names or relationship aliases
   *
   * @example
   * // User can only follow someone once
   * t.belongs_to("auth.user").as("follower")
   * t.belongs_to("auth.user").as("following")
   * t.unique("follower", "following")  // Expands to follower_id, following_id
   *
   * @example
   * // Unique slug per author
   * t.unique("author", "slug")
   */
  unique(...columns: string[]): void;

  // ===========================================================================
  // DOCUMENTATION
  // ===========================================================================

  /**
   * Adds documentation for the table.
   *
   * Becomes a SQL COMMENT on the table and appears in OpenAPI schemas.
   *
   * @param description - Human-readable table description
   *
   * @example
   * table(t => {
   *   t.docs("User accounts for authentication and profile data")
   *   t.id()
   *   // ...
   * })
   */
  docs(description: string): void;

  /**
   * Marks the table as deprecated.
   *
   * Shows deprecation warning in OpenAPI schemas.
   *
   * @param reason - Explanation and migration guidance
   *
   * @example
   * table(t => {
   *   t.deprecated("Use auth.user instead, will be removed in v2")
   *   t.id()
   *   // ...
   * })
   */
  deprecated(reason: string): void;
}

/**
 * Builder for altering columns in migrations.
 */
export interface AlterColumnBuilder {
  /**
   * Changes the column's data type.
   *
   * @param typeName - New type name (string, integer, etc.)
   * @param args - Type arguments (e.g., length for string)
   *
   * @example
   * m.alter_column("auth.user", "bio", c => {
   *   c.set_type("text")  // Expand from string to text
   * })
   */
  set_type(typeName: string, ...args: any[]): AlterColumnBuilder;

  /**
   * Makes the column nullable (allow NULL).
   *
   * @example
   * m.alter_column("blog.post", "category_id", c => {
   *   c.set_nullable()
   * })
   */
  set_nullable(): AlterColumnBuilder;

  /**
   * Makes the column NOT NULL.
   *
   * Ensure all existing rows have a value first.
   *
   * @example
   * m.alter_column("auth.user", "email", c => {
   *   c.set_not_null()
   * })
   */
  set_not_null(): AlterColumnBuilder;

  /**
   * Sets or changes the default value.
   *
   * @param value - New default value
   *
   * @example
   * m.alter_column("blog.post", "status", c => {
   *   c.set_default("draft")
   * })
   */
  set_default(value: any): AlterColumnBuilder;

  /**
   * Removes the default value.
   *
   * @example
   * m.alter_column("blog.post", "views", c => {
   *   c.drop_default()
   * })
   */
  drop_default(): AlterColumnBuilder;

  /**
   * Sets a raw SQL expression as the default.
   *
   * @param expr - SQL expression
   *
   * @example
   * m.alter_column("audit.log", "created_at", c => {
   *   c.set_server_default("NOW()")
   * })
   */
  set_server_default(expr: string): AlterColumnBuilder;
}

/**
 * Factory for creating columns in add_column migrations.
 * Has the same type methods as TableBuilder.
 */
export interface ColumnFactory {
  // Core types
  string(name: string, length: number): ColumnBuilder;
  text(name: string): ColumnBuilder;
  integer(name: string): ColumnBuilder;
  float(name: string): ColumnBuilder;
  decimal(name: string, precision: number, scale: number): ColumnBuilder;
  boolean(name: string): ColumnBuilder;
  date(name: string): ColumnBuilder;
  time(name: string): ColumnBuilder;
  datetime(name: string): ColumnBuilder;
  uuid(name: string): ColumnBuilder;
  json(name: string): ColumnBuilder;
  base64(name: string): ColumnBuilder;
  enum(name: string, values: string[]): ColumnBuilder;
  // Semantic types
  email(name: string): ColumnBuilder;
  username(name: string): ColumnBuilder;
  password_hash(name: string): ColumnBuilder;
  phone(name: string): ColumnBuilder;
  name(name: string): ColumnBuilder;
  title(name: string): ColumnBuilder;
  slug(name: string): ColumnBuilder;
  body(name: string): ColumnBuilder;
  summary(name: string): ColumnBuilder;
  url(name: string): ColumnBuilder;
  ip(name: string): ColumnBuilder;
  user_agent(name: string): ColumnBuilder;
  money(name: string): ColumnBuilder;
  percentage(name: string): ColumnBuilder;
  counter(name: string): ColumnBuilder;
  quantity(name: string): ColumnBuilder;
  flag(name: string, defaultValue?: boolean): ColumnBuilder;
  // Identifiers & tokens
  token(name: string): ColumnBuilder;
  code(name: string): ColumnBuilder;
  // Geographic & i18n
  country(name: string): ColumnBuilder;
  currency(name: string): ColumnBuilder;
  locale(name: string): ColumnBuilder;
  timezone(name: string): ColumnBuilder;
  // Ratings & measurements
  rating(name: string): ColumnBuilder;
  duration(name: string): ColumnBuilder;
  color(name: string): ColumnBuilder;
  // Content formats
  markdown(name: string): ColumnBuilder;
  html(name: string): ColumnBuilder;
}

// =============================================================================
// OBJECT-BASED API - Column factory for table({...}) syntax
// =============================================================================

/**
 * Fluent builder for column definitions in object-based API.
 * Returned by col.* methods. The column name comes from the object key.
 */
export interface ColColumnBuilder {
  /**
   * Allows NULL values in this column.
   * By default, all columns are NOT NULL.
   * @example
   * { middle_name: col.string(50).optional() }
   */
  optional(): ColColumnBuilder;

  /**
   * Adds a unique constraint to this column.
   * Unique columns are automatically indexed.
   * @example
   * { email: col.email().unique() }
   */
  unique(): ColColumnBuilder;

  /**
   * Sets the default value for new rows.
   * @param value - Static value or sql() expression
   * @example
   * { status: col.string(20).default("pending") }
   * { created_at: col.datetime().default(sql("NOW()")) }
   */
  default(value: any): ColColumnBuilder;

  /**
   * Sets a one-time value for existing rows during migration.
   * Use when adding a NOT NULL column to a table with existing data.
   * @param value - Static value or sql() expression
   * @example
   * { score: col.integer().backfill(0) }
   */
  backfill(value: any): ColColumnBuilder;

  /**
   * Sets the minimum constraint.
   * For strings: minimum character length. For numbers: minimum value.
   * @param n - Minimum value
   * @example
   * { username: col.string(50).min(3) }
   * { quantity: col.integer().min(1) }
   */
  min(n: number): ColColumnBuilder;

  /**
   * Sets the maximum constraint.
   * For strings: maximum character length. For numbers: maximum value.
   * @param n - Maximum value
   * @example
   * { rating: col.integer().min(1).max(5) }
   */
  max(n: number): ColColumnBuilder;

  /**
   * Sets a regex pattern for validation.
   * Generates CHECK constraint and OpenAPI pattern.
   * @param regex - JavaScript regex pattern (without delimiters)
   * @example
   * { zip_code: col.string(10).pattern("^\\d{5}(-\\d{4})?$") }
   */
  pattern(regex: string): ColColumnBuilder;

  /**
   * Sets the OpenAPI format hint.
   * Standard formats: "email", "uri", "uuid", "date", "time", "date-time", etc.
   * @param format - OpenAPI format string
   * @example
   * { callback: col.string(2048).format("uri") }
   */
  format(format: string): ColColumnBuilder;

  /**
   * Adds documentation for the column.
   * Becomes a SQL COMMENT and OpenAPI description.
   * @param description - Human-readable description
   * @example
   * { retry_count: col.integer().default(0).docs("Number of failed attempts") }
   */
  docs(description: string): ColColumnBuilder;

  /**
   * Marks the column as deprecated.
   * Shows deprecation warning in OpenAPI schemas.
   * @param reason - Explanation and migration guidance
   * @example
   * { old_status: col.string(20).deprecated("Use status_id instead") }
   */
  deprecated(reason: string): ColColumnBuilder;
}

/**
 * Fluent builder for relationship definitions in object-based API.
 * Returned by col.belongs_to() and col.one_to_one().
 */
export interface ColRelationshipBuilder {
  /**
   * Makes the foreign key nullable (allows NULL).
   * By default, belongs_to() creates a NOT NULL column.
   * @example
   * { manager: col.belongs_to("auth.user").optional() }
   */
  optional(): ColRelationshipBuilder;

  /**
   * Sets the ON DELETE action for referential integrity.
   * @param action - "cascade" | "set null" | "restrict" | "no action" | "set default"
   * @example
   * { author: col.belongs_to("auth.user").on_delete("cascade") }
   */
  on_delete(action: string): ColRelationshipBuilder;

  /**
   * Sets the ON UPDATE action for referential integrity.
   * Rarely needed since UUIDs don't change.
   * @param action - "cascade" | "set null" | "restrict" | "no action" | "set default"
   */
  on_update(action: string): ColRelationshipBuilder;

  /**
   * Adds documentation for the relationship.
   * Becomes a SQL COMMENT and OpenAPI description.
   * @param description - Human-readable description
   * @example
   * { reviewer: col.belongs_to("auth.user").docs("User who approved this") }
   */
  docs(description: string): ColRelationshipBuilder;

  /**
   * Marks the relationship as deprecated.
   * Shows deprecation warning in OpenAPI schemas.
   * @param reason - Explanation and migration guidance
   */
  deprecated(reason: string): ColRelationshipBuilder;
}

/**
 * Column factory for the object-based table API.
 * Column names come from the object keys, not method arguments.
 *
 * @example
 * export default table({
 *   id: col.id(),
 *   email: col.email().unique(),
 *   author: col.belongs_to("auth.user"),
 * }).timestamps()
 */
export interface ColBuilder {
  // ===========================================================================
  // PRIMARY KEY
  // ===========================================================================

  /**
   * UUID primary key column.
   * Every table should have this as the first column.
   * @example
   * { id: col.id() }
   */
  id(): ColColumnBuilder;

  // ===========================================================================
  // CORE TYPES
  // ===========================================================================

  /**
   * Variable-length string with maximum length.
   * @param length - Maximum character length
   * @example
   * { country_code: col.string(2) }
   */
  string(length: number): ColColumnBuilder;

  /** Unlimited text content. Maps to TEXT. */
  text(): ColColumnBuilder;

  /**
   * 32-bit signed integer (-2B to +2B). Safe for JavaScript.
   * @example
   * { view_count: col.integer() }
   */
  integer(): ColColumnBuilder;

  /** 32-bit floating-point number. For money, use decimal() instead. */
  float(): ColColumnBuilder;

  /**
   * Exact decimal number with fixed precision.
   * @param precision - Total number of digits
   * @param scale - Digits after decimal point
   * @example
   * { price: col.decimal(10, 2) }
   */
  decimal(precision: number, scale: number): ColColumnBuilder;

  /** True/false boolean. For flags with defaults, use flag() instead. */
  boolean(): ColColumnBuilder;

  /** Date without time (YYYY-MM-DD). */
  date(): ColColumnBuilder;

  /** Time without date (HH:MM:SS). */
  time(): ColColumnBuilder;

  /** Timestamp with timezone. For created_at/updated_at, use .timestamps(). */
  datetime(): ColColumnBuilder;

  /** UUID column (non-primary key). For primary key, use id(). */
  uuid(): ColColumnBuilder;

  /**
   * JSON/JSONB column for structured data.
   * @example
   * { preferences: col.json().default({}) }
   */
  json(): ColColumnBuilder;

  /** Binary data stored as base64. Maps to BYTEA/BLOB. */
  base64(): ColColumnBuilder;

  /**
   * Enumerated type with fixed allowed values.
   * @param values - Array of allowed string values
   * @example
   * { status: col.enum(["draft", "published", "archived"]) }
   */
  enum(values: string[]): ColColumnBuilder;

  // ===========================================================================
  // SEMANTIC TYPES
  // ===========================================================================

  /**
   * Email address: string(255) + email format + RFC 5322 pattern.
   * @example
   * { email: col.email().unique() }
   */
  email(): ColColumnBuilder;

  /**
   * Username: string(50) + alphanumeric pattern + min(3).
   * @example
   * { username: col.username().unique() }
   */
  username(): ColColumnBuilder;

  /**
   * Password hash: string(255) + hidden from OpenAPI.
   * @example
   * { password: col.password_hash() }
   */
  password_hash(): ColColumnBuilder;

  /**
   * Phone number: string(50) + E.164 pattern.
   * @example
   * { phone: col.phone().optional() }
   */
  phone(): ColColumnBuilder;

  /**
   * Person/entity name: string(100).
   * @example
   * { first_name: col.name() }
   */
  name(): ColColumnBuilder;

  /**
   * Title/headline: string(200).
   * @example
   * { title: col.title() }
   */
  title(): ColColumnBuilder;

  /**
   * URL slug: string(255) + unique + slug pattern.
   * @example
   * { slug: col.slug() }
   */
  slug(): ColColumnBuilder;

  /**
   * Long-form content: text (unlimited).
   * @example
   * { content: col.body() }
   */
  body(): ColColumnBuilder;

  /**
   * Short summary: string(500).
   * @example
   * { excerpt: col.summary().optional() }
   */
  summary(): ColColumnBuilder;

  /**
   * URL/URI: string(2048) + uri format.
   * @example
   * { website: col.url().optional() }
   */
  url(): ColColumnBuilder;

  /**
   * IP address: string(45) for IPv4/IPv6.
   * @example
   * { ip_address: col.ip() }
   */
  ip(): ColColumnBuilder;

  /**
   * User agent string: string(500).
   * @example
   * { user_agent: col.user_agent().optional() }
   */
  user_agent(): ColColumnBuilder;

  /**
   * Money amount: decimal(19,4) + min(0).
   * Serialized as string to preserve precision.
   * @example
   * { price: col.money() }
   */
  money(): ColColumnBuilder;

  /**
   * Percentage: decimal(5,2) + min(0) + max(100).
   * @example
   * { discount: col.percentage() }
   */
  percentage(): ColColumnBuilder;

  /**
   * Counter: integer + default(0).
   * @example
   * { view_count: col.counter() }
   */
  counter(): ColColumnBuilder;

  /**
   * Quantity: integer + min(0). Cannot be negative.
   * @example
   * { stock: col.quantity() }
   */
  quantity(): ColColumnBuilder;

  /**
   * Boolean flag with default value.
   * @param defaultValue - Default value (defaults to false)
   * @example
   * { is_active: col.flag(true) }
   * { is_verified: col.flag() }
   */
  flag(defaultValue?: boolean): ColColumnBuilder;

  // ===========================================================================
  // IDENTIFIERS & TOKENS
  // ===========================================================================

  /**
   * Secure token: string(64) + unique.
   * For API keys, session tokens, reset tokens.
   * @example
   * { api_key: col.token().unique() }
   */
  token(): ColColumnBuilder;

  /**
   * Short code: string(20) + uppercase pattern.
   * For SKUs, promo codes, reference numbers.
   * @example
   * { sku: col.code().unique() }
   */
  code(): ColColumnBuilder;

  // ===========================================================================
  // GEOGRAPHIC & I18N
  // ===========================================================================

  /**
   * ISO 3166-1 alpha-2 country code: string(2).
   * Two-letter codes like US, GB, DE, JP.
   * @example
   * { country: col.country() }
   */
  country(): ColColumnBuilder;

  /**
   * ISO 4217 currency code: string(3).
   * Three-letter codes like USD, EUR, GBP.
   * @example
   * { currency: col.currency() }
   */
  currency(): ColColumnBuilder;

  /**
   * BCP 47 locale code: string(10).
   * Language/region codes like en-US, es-MX, zh-CN.
   * @example
   * { locale: col.locale() }
   */
  locale(): ColColumnBuilder;

  /**
   * IANA timezone identifier: string(50).
   * Names like America/New_York, Europe/London.
   * @example
   * { timezone: col.timezone() }
   */
  timezone(): ColColumnBuilder;

  // ===========================================================================
  // RATINGS & MEASUREMENTS
  // ===========================================================================

  /**
   * Star rating: decimal(2,1) + min(0) + max(5).
   * Allows half-stars (e.g., 4.5).
   * @example
   * { rating: col.rating() }
   */
  rating(): ColColumnBuilder;

  /**
   * Duration in seconds: integer + min(0).
   * @example
   * { duration: col.duration() }
   */
  duration(): ColColumnBuilder;

  /**
   * Hex color code: string(7).
   * Format: #RRGGBB (e.g., #FF5733).
   * @example
   * { primary_color: col.color() }
   */
  color(): ColColumnBuilder;

  // ===========================================================================
  // CONTENT FORMATS
  // ===========================================================================

  /**
   * Markdown content: text + markdown format hint.
   * @example
   * { content: col.markdown() }
   */
  markdown(): ColColumnBuilder;

  /**
   * HTML content: text + html format hint.
   * Be careful with XSS - sanitize on input.
   * @example
   * { rendered_content: col.html() }
   */
  html(): ColColumnBuilder;

  // ===========================================================================
  // RELATIONSHIPS
  // ===========================================================================

  /**
   * Foreign key to another table.
   * Creates a NOT NULL {key}_id column. Use .optional() for nullable.
   * @param model - Reference in format "namespace.table"
   * @example
   * { author: col.belongs_to("auth.user") }
   * { category: col.belongs_to("catalog.category").optional() }
   */
  belongs_to(model: string): ColRelationshipBuilder;

  /**
   * Unique foreign key (one-to-one relationship).
   * Like belongs_to() but with a unique constraint.
   * @param model - Reference in format "namespace.table"
   * @example
   * { user: col.one_to_one("auth.user") }
   */
  one_to_one(model: string): ColRelationshipBuilder;
}
`

const typeSchemaDTS = `/**
 * Alab Schema Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { TableBuilder, ColColumnBuilder, ColRelationshipBuilder, PolymorphicOptions } from "./column";

/**
 * Represents a complete table definition.
 * Returned by the table() function.
 */
export interface TableDefinition {
  readonly namespace: string;
  readonly name: string;
}

/**
 * Column definitions object for the object-based table API.
 * Keys become column names, values are column builders from col.*.
 */
export type ColumnDefinitions = Record<string, ColColumnBuilder | ColRelationshipBuilder>;

/**
 * Chainable table operations for the object-based API.
 * Returned by table({...}) for adding table-level configuration.
 *
 * @example
 * export default table({
 *   id: col.id(),
 *   email: col.email().unique(),
 * })
 * .timestamps()
 * .index("email")
 * .docs("User accounts")
 */
export interface TableChain {
  /** Adds created_at and updated_at timestamp columns. */
  timestamps(): TableChain;

  /** Adds a deleted_at nullable timestamp for soft deletion. */
  soft_delete(): TableChain;

  /** Adds a position integer column for manual ordering. */
  sortable(): TableChain;

  /** Adds a non-unique index on columns. */
  index(...columns: string[]): TableChain;

  /** Adds a composite unique constraint on multiple columns. */
  unique(...columns: string[]): TableChain;

  /** Declares a many-to-many relationship. */
  many_to_many(model: string): TableChain;

  /** Adds a polymorphic reference (type + id columns). */
  belongs_to_any(models: string[], opts?: PolymorphicOptions): TableChain;

  /** Adds documentation for the table. */
  docs(description: string): TableChain;

  /** Marks the table as deprecated. */
  deprecated(reason: string): TableChain;
}

/**
 * Builder for defining multiple tables within a namespace.
 */
export interface SchemaBuilder {
  table(name: string, fn: (t: TableBuilder) => void): void;
}

/**
 * Defines a schema namespace with multiple tables in one file.
 */
declare function schema(
  namespace: string,
  fn: (s: SchemaBuilder) => void
): void;

/**
 * Defines a single table using a callback (legacy API).
 */
declare function table(fn: (t: TableBuilder) => void): TableDefinition;

/**
 * Defines a single table using an object (preferred API).
 *
 * @example
 * // schemas/auth/user.js (object API - preferred)
 * export default table({
 *   id: col.id(),
 *   email: col.email().unique(),
 *   username: col.username().unique(),
 *   password: col.password_hash(),
 *   is_active: col.flag(true),
 *   last_login: col.datetime().optional(),
 * }).timestamps()
 *
 * @example
 * // schemas/blog/post.js
 * export default table({
 *   id: col.id(),
 *   title: col.title(),
 *   slug: col.slug(),
 *   content: col.body(),
 *   author: col.belongs_to("auth.user"),
 *   status: col.enum(["draft", "published", "archived"]).default("draft"),
 * })
 * .timestamps()
 * .soft_delete()
 */
declare function table(columns: ColumnDefinitions): TableChain;

export { schema, table, SchemaBuilder, TableDefinition, TableChain, ColumnDefinitions };
`

const typeMigrationDTS = `/**
 * Alab Migration Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { TableBuilder, ColumnBuilder, ColumnFactory, AlterColumnBuilder } from "./column";

/**
 * Options for the migration function.
 */
export interface MigrationOptions {
  /** Human-readable description of the migration */
  description?: string;
  /** Whether this migration is reversible (default: true) */
  reversible?: boolean;
}

/**
 * Options for creating indexes.
 */
export interface IndexOptions {
  /** Make the index unique */
  unique?: boolean;
  /** Custom index name */
  name?: string;
}

/**
 * Builder for defining migration operations.
 * Provides imperative methods for schema changes.
 *
 * @example
 * migration(m => {
 *   m.create_table("blog.post", t => {
 *     t.id()
 *     t.string("title", 200)
 *     t.timestamps()
 *   })
 * })
 */
export interface MigrationBuilder {
  /**
   * Creates a new table.
   *
   * @param name - Table reference (e.g., "blog.post" or "post")
   * @param fn - Callback receiving a TableBuilder
   *
   * @example
   * m.create_table("blog.post", t => {
   *   t.id()
   *   t.string("title", 200)
   *   t.text("content")
   *   t.timestamps()
   * })
   */
  create_table(name: string, fn: (t: TableBuilder) => void): void;

  /**
   * Drops an existing table.
   *
   * @param name - Table reference to drop
   *
   * @example
   * m.drop_table("blog.post")
   */
  drop_table(name: string): void;

  /**
   * Renames a table.
   *
   * @param oldName - Current table reference
   * @param newName - New table name (without namespace)
   *
   * @example
   * m.rename_table("blog.post", "article")
   */
  rename_table(oldName: string, newName: string): void;

  /**
   * Adds a column to an existing table.
   *
   * @param table - Table reference
   * @param fn - Callback that creates and returns a ColumnBuilder
   *
   * @example
   * m.add_column("blog.post", c => c.string("slug", 200).unique())
   * m.add_column("blog.post", c => c.integer("view_count").default(0))
   */
  add_column(table: string, fn: (c: ColumnFactory) => ColumnBuilder): void;

  /**
   * Removes a column from a table.
   *
   * @param table - Table reference
   * @param column - Column name to drop
   *
   * @example
   * m.drop_column("blog.post", "legacy_field")
   */
  drop_column(table: string, column: string): void;

  /**
   * Renames a column.
   *
   * @param table - Table reference
   * @param oldName - Current column name
   * @param newName - New column name
   *
   * @example
   * m.rename_column("blog.post", "title", "headline")
   */
  rename_column(table: string, oldName: string, newName: string): void;

  /**
   * Modifies a column's properties.
   *
   * @param table - Table reference
   * @param column - Column name to alter
   * @param fn - Callback receiving an AlterColumnBuilder
   *
   * @example
   * m.alter_column("blog.post", "title", c => {
   *   c.set_type("string", 500)
   *   c.set_nullable()
   * })
   */
  alter_column(
    table: string,
    column: string,
    fn: (c: AlterColumnBuilder) => void
  ): void;

  /**
   * Creates an index on a table.
   *
   * @param table - Table reference
   * @param columns - Column names to index
   * @param opts - Optional index configuration
   *
   * @example
   * m.create_index("blog.post", ["author_id", "created_at"])
   * m.create_index("blog.post", ["slug"], { unique: true, name: "post_slug_idx" })
   */
  create_index(table: string, columns: string[], opts?: IndexOptions): void;

  /**
   * Drops an index by name.
   *
   * @param name - Index name to drop
   *
   * @example
   * m.drop_index("post_slug_idx")
   */
  drop_index(name: string): void;

  /**
   * Executes raw SQL.
   * Use sparingly - prefer the typed methods when possible.
   *
   * @param sql - Raw SQL statement to execute
   *
   * @example
   * m.sql("CREATE EXTENSION IF NOT EXISTS pg_trgm")
   */
  sql(sql: string): void;

  /**
   * Executes raw SQL with dialect-specific variants.
   * Useful when different databases need different syntax.
   *
   * @param postgres - PostgreSQL variant
   * @param sqlite - SQLite variant
   *
   * @example
   * m.sql_dialect(
   *   "CREATE EXTENSION IF NOT EXISTS pg_trgm",
   *   "" // SQLite doesn't support this extension
   * )
   */
  sql_dialect(postgres: string, sqlite: string): void;
}

/**
 * Defines a migration with schema change operations.
 *
 * @param fn - Callback receiving a MigrationBuilder
 * @param opts - Optional migration configuration
 *
 * @example
 * // migrations/001_create_users.js
 * migration(m => {
 *   m.create_table("auth.user", t => {
 *     t.id()
 *     t.email("email").unique()
 *     t.password_hash("password_hash")
 *     t.timestamps()
 *   })
 * }, { description: "Create users table" })
 *
 * @example
 * // migrations/002_add_user_profile.js
 * migration(m => {
 *   m.add_column("auth.user", c =>
 *     c.string("display_name", 100).optional()
 *   )
 *   m.add_column("auth.user", c =>
 *     c.text("bio").optional()
 *   )
 * })
 */
declare function migration(
  fn: (m: MigrationBuilder) => void,
  opts?: MigrationOptions
): void;

export { migration, MigrationBuilder, MigrationOptions, IndexOptions };
`
