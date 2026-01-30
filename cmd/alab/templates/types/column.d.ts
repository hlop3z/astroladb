/**
 * Alab Column Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { SQLExpr } from "./globals";

/**
 * Opaque expression type returned by fn.* methods.
 * Pass to .computed() to define generated columns.
 */
export interface FnExpr {
  readonly __fn: true;
}

/**
 * Expression builder for computed/virtual columns.
 *
 * All methods return FnExpr which can be passed to .computed().
 * Expressions are dialect-aware — they generate the correct SQL
 * for PostgreSQL, SQLite, etc.
 *
 * #### Available Functions
 *
 * | Category  | Methods                                                              |
 * | --------- | -------------------------------------------------------------------- |
 * | Reference | `col(name)`                                                          |
 * | String    | `concat()` `upper()` `lower()` `trim()` `length()` `substring()`     |
 * | Math      | `add()` `sub()` `mul()` `div()` `abs()` `round()` `floor()` `ceil()` |
 * | Date      | `year()` `month()` `day()` `now()` `years_since()` `days_since()`    |
 * | Null      | `coalesce()` `nullif()` `if_null()`                                  |
 * | Logic     | `if_then()` `gt()` `gte()` `lt()` `lte()` `eq()`                     |
 * | Raw       | `sql({ postgres: "...", sqlite: "..." })`                            |
 *
 * @example
 * // Stored computed column
 * { full_name: col.text().computed(fn.concat(fn.col("first"), " ", fn.col("last"))) }
 *
 * @example
 * // Virtual (query-time) computed column
 * { age: col.integer().computed(fn.years_since(fn.col("birth_date"))).virtual() }
 *
 * @example
 * // Conditional expression
 * { is_adult: col.boolean().computed(fn.gte(fn.col("age"), 18)) }
 */
export interface FnBuilder {
  // ── Column Reference ──

  /** Reference a column by name. */
  col(name: string): FnExpr;

  // ── String Functions ──

  /** Concatenate strings and expressions. */
  concat(...args: (FnExpr | string)[]): FnExpr;

  /** Convert to uppercase. */
  upper(arg: FnExpr | string): FnExpr;

  /** Convert to lowercase. */
  lower(arg: FnExpr | string): FnExpr;

  /** Remove leading/trailing whitespace. */
  trim(arg: FnExpr | string): FnExpr;

  /** Get string length. */
  length(arg: FnExpr | string): FnExpr;

  /** Extract substring. */
  substring(str: FnExpr | string, start: number, length: number): FnExpr;

  // ── Math Functions ──

  /** Addition: a + b. */
  add(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Subtraction: a - b. */
  sub(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Multiplication: a * b. */
  mul(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Division: a / b. */
  div(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Absolute value. */
  abs(arg: FnExpr | number): FnExpr;

  /** Round to nearest integer. */
  round(arg: FnExpr | number): FnExpr;

  /** Round down. */
  floor(arg: FnExpr | number): FnExpr;

  /** Round up. */
  ceil(arg: FnExpr | number): FnExpr;

  // ── Date/Time Functions ──

  /** Extract year from date/datetime. */
  year(arg: FnExpr): FnExpr;

  /** Extract month from date/datetime. */
  month(arg: FnExpr): FnExpr;

  /** Extract day from date/datetime. */
  day(arg: FnExpr): FnExpr;

  /** Current timestamp (NOW()). */
  now(): FnExpr;

  /** Years elapsed since a date column. */
  years_since(date: FnExpr): FnExpr;

  /** Days elapsed since a date column. */
  days_since(date: FnExpr): FnExpr;

  // ── Null Handling ──

  /** Return first non-null value. */
  coalesce(...args: (FnExpr | any)[]): FnExpr;

  /** Return NULL if a equals b. */
  nullif(a: FnExpr | any, b: FnExpr | any): FnExpr;

  /** Return default if arg is NULL. */
  if_null(arg: FnExpr, defaultValue: FnExpr | any): FnExpr;

  // ── Conditional ──

  /** CASE WHEN condition THEN thenVal ELSE elseVal END. */
  if_then(condition: FnExpr, thenVal: FnExpr | any, elseVal: FnExpr | any): FnExpr;

  // ── Comparison ──

  /** Greater than: a > b. */
  gt(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Greater than or equal: a >= b. */
  gte(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Less than: a < b. */
  lt(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Less than or equal: a <= b. */
  lte(a: FnExpr | number, b: FnExpr | number): FnExpr;

  /** Equal: a = b. */
  eq(a: FnExpr | any, b: FnExpr | any): FnExpr;

  // ── Raw SQL Escape Hatch ──

  /**
   * Raw SQL expression per dialect.
   * @param dialects - Object with dialect-specific SQL strings
   * @example
   * fn.sql({ postgres: "AGE(birth_date)", sqlite: "CAST((julianday('now') - julianday(birth_date)) / 365.25 AS INTEGER)" })
   */
  sql(dialects: { postgres?: string; sqlite?: string }): FnExpr;
}

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

  /**
   * Marks this column as a computed/generated column.
   *
   * The expression is evaluated by the database (STORED) or at query time (VIRTUAL).
   * Use fn.* helpers to build expressions.
   *
   * @param expr - An fn.* expression or raw SQL via fn.sql()
   *
   * @example
   * t.text("full_name").computed(fn.concat(fn.col("first_name"), " ", fn.col("last_name")))
   * t.boolean("is_adult").computed(fn.gte(fn.col("age"), 18))
   */
  computed(expr: FnExpr): ColumnBuilder;

  /**
   * Marks this column as VIRTUAL (not stored) or app-only (no DB column).
   *
   * Virtual columns are not physically stored in the database. If combined
   * with computed(), the value is computed at query time. Without computed(),
   * the column exists only in the application layer.
   *
   * @example
   * t.integer("age").computed(fn.years_since(fn.col("birth_date"))).virtual()
   */
  virtual(): ColumnBuilder;
}

/**
 * Builder for creating tables in migrations.
 *
 * IMPORTANT: This interface only includes LOW-LEVEL types.
 * Semantic types (email, money, flag, etc.) are NOT available in migrations.
 * Use explicit types like string(255), decimal(19,4), boolean().default(false).
 *
 * Migrations should be explicit and stable. If semantic type definitions
 * change in the future, existing migrations will still work correctly.
 *
 * For semantic types, define your schema in schema files (using ColBuilder)
 * and let Alab generate migrations automatically.
 */
export interface MigrationTableBuilder {
  // ── Primary Key ──

  /**
   * Adds a UUID primary key column named "id".
   */
  id(): ColumnBuilder;

  // ── Low-Level Types ──

  /**
   * Variable-length string with maximum length.
   * @param name - Column name in snake_case
   * @param length - Maximum character length
   */
  string(name: string, length: number): ColumnBuilder;

  /**
   * Unlimited text content.
   * @param name - Column name in snake_case
   */
  text(name: string): ColumnBuilder;

  /**
   * 32-bit signed integer.
   * @param name - Column name in snake_case
   */
  integer(name: string): ColumnBuilder;

  /**
   * 32-bit floating-point number.
   * @param name - Column name in snake_case
   */
  float(name: string): ColumnBuilder;

  /**
   * Exact decimal number with fixed precision.
   * @param name - Column name in snake_case
   * @param precision - Total number of digits
   * @param scale - Digits after decimal point
   */
  decimal(name: string, precision: number, scale: number): ColumnBuilder;

  /**
   * True/false boolean value.
   * @param name - Column name in snake_case
   */
  boolean(name: string): ColumnBuilder;

  /**
   * Date without time (YYYY-MM-DD).
   * @param name - Column name in snake_case
   */
  date(name: string): ColumnBuilder;

  /**
   * Time without date (HH:MM:SS).
   * @param name - Column name in snake_case
   */
  time(name: string): ColumnBuilder;

  /**
   * Timestamp with timezone.
   * @param name - Column name in snake_case
   */
  datetime(name: string): ColumnBuilder;

  /**
   * UUID column (non-primary key).
   * @param name - Column name in snake_case
   */
  uuid(name: string): ColumnBuilder;

  /**
   * JSON/JSONB column for structured data.
   * @param name - Column name in snake_case
   */
  json(name: string): ColumnBuilder;

  /**
   * Binary data stored as base64.
   * @param name - Column name in snake_case
   */
  base64(name: string): ColumnBuilder;

  /**
   * Enumerated type with fixed allowed values.
   * @param name - Column name in snake_case
   * @param values - Array of allowed string values
   */
  enum(name: string, values: string[]): ColumnBuilder;

  // ── Convenience Patterns ──

  /**
   * Adds created_at and updated_at timestamp columns.
   */
  timestamps(): void;

  /**
   * Adds a deleted_at nullable timestamp for soft deletion.
   */
  soft_delete(): void;

  /**
   * Adds a position integer column for manual ordering.
   */
  sortable(): void;

  // ── Relationships ──

  /**
   * Adds a foreign key column to another table.
   * @param model - Reference in format "namespace.table"
   */
  belongs_to(model: string): RelationshipBuilder;

  /**
   * Adds a unique foreign key (one-to-one relationship).
   * @param model - Reference in format "namespace.table"
   */
  one_to_one(model: string): RelationshipBuilder;

  /**
   * Declares a many-to-many relationship.
   * @param model - Reference to the other table
   * @param opts - Optional: { through: "custom_table_name" }
   */
  many_to_many(model: string, opts?: M2MOptions): void;

  /**
   * Adds a polymorphic reference (type + id columns).
   * @param models - Array of possible model references
   * @param opts - Optional: { as: "base_name" }
   */
  belongs_to_any(models: string[], opts?: PolymorphicOptions): void;

  // ── Indexes & Constraints ──

  /**
   * Adds a non-unique index on columns.
   * @param columns - Column names to index
   */
  index(...columns: string[]): void;

  /**
   * Adds a composite unique constraint on multiple columns.
   * @param columns - Column names or relationship aliases
   */
  unique(...columns: string[]): void;

  // ── Documentation ──

  /**
   * Adds documentation for the table.
   * @param description - Human-readable table description
   */
  docs(description: string): void;

  /**
   * Marks the table as deprecated.
   * @param reason - Explanation and migration guidance
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
 * Only low-level types are available in migrations.
 * For semantic types, use schema definitions instead.
 */
export interface ColumnFactory {
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
}

// ── Object-Based API ──

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

  /**
   * Marks this column as a computed/generated column.
   * Use fn.* helpers to build expressions.
   * @param expr - An fn.* expression
   * @example
   * { full_name: col.text().computed(fn.concat(fn.col("first_name"), " ", fn.col("last_name"))) }
   */
  computed(expr: FnExpr): ColColumnBuilder;

  /**
   * Marks this column as VIRTUAL (not stored) or app-only.
   * @example
   * { age: col.integer().computed(fn.years_since(fn.col("birth_date"))).virtual() }
   */
  virtual(): ColColumnBuilder;
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
 * Column factory — use `col.*` in table({...}) definitions.
 *
 * #### Core Types
 *
 * | Method         | SQL Type    | Notes                                   |
 * | -------------- | ----------- | --------------------------------------- |
 * | `id()`         | UUID PK     | Required first column                   |
 * | `string(n)`    | VARCHAR(n)  | Length required                         |
 * | `text()`       | TEXT        | Unlimited length                        |
 * | `integer()`    | INTEGER     | 32-bit, JS-safe                         |
 * | `float()`      | REAL        | Approximate                             |
 * | `decimal(p,s)` | DECIMAL     | Exact; serialized as string             |
 * | `boolean()`    | BOOLEAN     | For flags, prefer `flag()`              |
 * | `date()`       | DATE        | YYYY-MM-DD                              |
 * | `time()`       | TIME        | HH:MM:SS                                |
 * | `datetime()`   | TIMESTAMPTZ | Use `.timestamps()` for created/updated |
 * | `uuid()`       | UUID        | Non-PK; for PK use `id()`               |
 * | `json()`       | JSONB       | Flexible structured data                |
 * | `base64()`     | BYTEA/BLOB  | Small binary data                       |
 * | `enum([...])`  | CHECK       | Fixed set of strings                    |
 *
 * #### Semantic Types (shortcuts with built-in validation)
 *
 * | Method           | Expands To    | Built-in                     |
 * | ---------------- | ------------- | ---------------------------- |
 * | `email()`        | string(255)   | format, RFC 5322 pattern     |
 * | `username()`     | string(50)    | min(3), alphanumeric pattern |
 * | `password_hash()`| string(255)   | hidden from API              |
 * | `phone()`        | string(50)    | E.164 pattern                |
 * | `name()`         | string(100)   | —                            |
 * | `title()`        | string(200)   | —                            |
 * | `slug()`         | string(255)   | unique, slug pattern         |
 * | `body()`         | text          | —                            |
 * | `summary()`      | string(500)   | —                            |
 * | `url()`          | string(2048)  | uri format                   |
 * | `money()`        | decimal(19,4) | min(0)                       |
 * | `percentage()`   | decimal(5,2)  | min(0), max(100)             |
 * | `counter()`      | integer       | default(0)                   |
 * | `quantity()`     | integer       | min(0)                       |
 * | `flag(def?)`     | boolean       | default(false)               |
 * | `token()`        | string(64)    | unique                       |
 * | `code()`         | string(20)    | uppercase pattern            |
 * | `country()`      | string(2)     | ISO 3166-1                   |
 * | `currency()`     | string(3)     | ISO 4217                     |
 * | `locale()`       | string(10)    | BCP 47                       |
 * | `timezone()`     | string(50)    | IANA                         |
 * | `rating()`       | decimal(2,1)  | min(0), max(5)               |
 * | `duration()`     | integer       | min(0), seconds              |
 * | `color()`        | string(7)     | #RRGGBB                      |
 * | `markdown()`     | text          | markdown format              |
 * | `html()`         | text          | html format                  |
 * | `ip()`           | string(45)    | IPv4 + IPv6                  |
 * | `ipv4()`         | string(15)    | dotted-decimal               |
 * | `ipv6()`         | string(45)    | full IPv6                    |
 * | `user_agent()`   | string(500)   | —                            |
 *
 * #### Relationships
 *
 * | Method            | Creates                |
 * | ----------------- | ---------------------- |
 * | `belongs_to(ref)` | `{key}_id` FK column   |
 * | `one_to_one(ref)` | `{key}_id` FK + unique |
 *
 * #### Column Modifiers (chainable)
 * `.optional()` `.unique()` `.default(v)` `.backfill(v)` `.min(n)` `.max(n)` `.pattern(re)` `.format(f)` `.docs(s)` `.deprecated(s)` `.computed(expr)` `.virtual()`
 *
 * @example
 * export default table({
 *   id: col.id(),
 *   email: col.email().unique(),
 *   author: col.belongs_to("auth.user"),
 * }).timestamps()
 */
export interface ColBuilder {
  // ── Primary Key ──

  /**
   * UUID primary key column.
   * Every table should have this as the first column.
   * @example
   * { id: col.id() }
   */
  id(): ColColumnBuilder;

  // ── Core Types ──

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

  // ── Semantic Types ──

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
   * IP address: string(45) for IPv4/IPv6 with validation.
   * Accepts both formats.
   * @example
   * { ip_address: col.ip() }
   */
  ip(): ColColumnBuilder;

  /**
   * IPv4 address only: string(15) with strict validation.
   * @example
   * { client_ip: col.ipv4() }
   */
  ipv4(): ColColumnBuilder;

  /**
   * IPv6 address only: string(45) with strict validation.
   * @example
   * { client_ip: col.ipv6() }
   */
  ipv6(): ColColumnBuilder;

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

  // ── Identifiers & Tokens ──

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

  // ── Geographic & I18N ──

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

  // ── Ratings & Measurements ──

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

  // ── Content Formats ──

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

  // ── Relationships ──
  //
  // belongs_to(ref)  → Many-to-one (post has one author)
  // one_to_one(ref)  → One-to-one  (user has one profile)
  //
  // For many-to-many, use .many_to_many() on TableChain.
  // For polymorphic, use .belongs_to_any() on TableChain.

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
