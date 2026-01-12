/*
 * Alab Migration Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { TableBuilder, ColumnBuilder, ColumnFactory, AlterColumnBuilder } from "./column";

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
 * Migrations use ES module exports with up() and down() functions.
 * The TableBuilder (t parameter in create_table) has ALL semantic types:
 * - Core: id(), string(), text(), integer(), float(), decimal(), boolean(), date(), time(), datetime(), uuid(), json(), base64(), enum()
 * - Identity: email(), username(), password_hash(), phone()
 * - Content: name(), title(), slug(), body(), summary(), url(), markdown(), html()
 * - Financial: money(), percentage(), counter(), quantity()
 * - Identifiers: token(), code(), country(), currency(), locale(), timezone()
 * - Metrics: rating(), duration(), color()
 * - Relationships: belongs_to(), one_to_one(), many_to_many(), belongs_to_any()
 * - Patterns: timestamps(), soft_delete(), sortable(), flag()
 *
 * @example
 * // migrations/001_create_users.js - Basic table with semantic types
 * export default migration({
 *   up(m) {
 *     m.create_table("auth.user", t => {
 *       t.id()                             // UUID primary key
 *       t.email("email").unique()          // Email with validation
 *       t.username("username").unique()    // Username with pattern
 *       t.password_hash("password")        // Hidden from API
 *       t.phone("phone").optional()        // E.164 format
 *       t.flag("is_active", true)          // Boolean with default
 *       t.datetime("last_login").optional()
 *       t.timestamps()                     // created_at, updated_at
 *     })
 *   },
 *   down(m) {
 *     m.drop_table("auth.user")
 *   }
 * })
 *
 * @example
 * // migrations/002_create_posts.js - Content table with relationships
 * export default migration({
 *   up(m) {
 *     m.create_table("blog.post", t => {
 *       t.id()
 *       t.title("title")                   // string(200)
 *       t.slug("slug")                     // Unique URL-friendly
 *       t.body("content")                  // Unlimited text
 *       t.summary("excerpt").optional()    // string(500)
 *       t.belongs_to("auth.user").as("author") // FK with alias
 *       t.enum("status", ["draft", "published", "archived"]).default("draft")
 *       t.counter("view_count")            // integer default 0
 *       t.rating("average_rating").optional()
 *       t.datetime("published_at").optional()
 *       t.timestamps()
 *       t.soft_delete()                    // deleted_at
 *     })
 *
 *     // Add indexes
 *     m.create_index("blog.post", ["author_id", "status"])
 *     m.create_index("blog.post", ["published_at"])
 *   },
 *   down(m) {
 *     m.drop_table("blog.post")
 *   }
 * })
 *
 * @example
 * // migrations/003_add_tags.js - Many-to-many relationship
 * export default migration({
 *   up(m) {
 *     m.create_table("blog.tag", t => {
 *       t.id()
 *       t.name("name").unique()
 *       t.slug("slug")
 *       t.color("color").optional()
 *       t.timestamps()
 *     })
 *
 *     // Join table (many-to-many)
 *     m.create_table("blog_post_blog_tag", t => {
 *       t.belongs_to("blog.post")
 *       t.belongs_to("blog.tag")
 *       t.unique("post", "tag")  // Composite unique constraint
 *     })
 *
 *     m.create_index("blog_post_blog_tag", ["post_id"])
 *     m.create_index("blog_post_blog_tag", ["tag_id"])
 *   },
 *   down(m) {
 *     m.drop_table("blog_post_blog_tag")
 *     m.drop_table("blog.tag")
 *   }
 * })
 *
 * @example
 * // migrations/004_create_products.js - E-commerce table
 * export default migration({
 *   up(m) {
 *     m.create_table("shop.product", t => {
 *       t.id()
 *       t.code("sku").unique()            // Uppercase pattern
 *       t.title("name")
 *       t.slug("slug")
 *       t.body("description")
 *       t.money("price")                  // decimal(19,4), min(0)
 *       t.currency("currency")            // ISO 4217
 *       t.quantity("stock")               // integer, min(0)
 *       t.percentage("discount").optional()
 *       t.url("image_url").optional()
 *       t.belongs_to("shop.category")
 *       t.flag("is_featured", false)
 *       t.sortable()                      // position column
 *       t.timestamps()
 *     })
 *   },
 *   down(m) {
 *     m.drop_table("shop.product")
 *   }
 * })
 *
 * @example
 * // migrations/005_polymorphic.js - Polymorphic relationships
 * export default migration({
 *   up(m) {
 *     m.create_table("core.comment", t => {
 *       t.id()
 *       t.belongs_to("auth.user").as("author")
 *       t.text("content")
 *       // Can comment on posts OR videos
 *       t.belongs_to_any(["blog.post", "media.video"], { as: "commentable" })
 *       t.timestamps()
 *     })
 *   },
 *   down(m) {
 *     m.drop_table("core.comment")
 *   }
 * })
 */
export interface MigrationBuilder {
  // ===========================================================================
  // TABLE OPERATIONS
  // ===========================================================================

  /**
   * Creates a new table with columns.
   *
   * The callback receives a TableBuilder with ALL semantic types available:
   * - Use semantic types (email, username, money, etc.) for common patterns
   * - Use core types (string, integer, etc.) for custom fields
   * - Chain modifiers: .unique(), .optional(), .default()
   * - Add relationships: .belongs_to(), .one_to_one(), .many_to_many()
   * - Use helpers: .timestamps(), .soft_delete(), .sortable()
   *
   * @param name - Table reference: "namespace.table" or just "table"
   * @param fn - Callback receiving a TableBuilder (t parameter)
   *
   * @example
   * // User authentication table
   * m.create_table("auth.user", t => {
   *   t.id()
   *   t.email("email").unique()
   *   t.username("username").unique()
   *   t.password_hash("password")
   *   t.flag("is_active", true)
   *   t.timestamps()
   * })
   *
   * @example
   * // Blog post with relationships
   * m.create_table("blog.post", t => {
   *   t.id()
   *   t.title("title")
   *   t.slug("slug")
   *   t.body("content")
   *   t.belongs_to("auth.user").as("author")
   *   t.belongs_to("blog.category")
   *   t.enum("status", ["draft", "published"]).default("draft")
   *   t.datetime("published_at").optional()
   *   t.counter("view_count")
   *   t.timestamps()
   *   t.soft_delete()
   * })
   *
   * @example
   * // Product catalog
   * m.create_table("shop.product", t => {
   *   t.id()
   *   t.code("sku").unique()
   *   t.title("name")
   *   t.body("description")
   *   t.money("price")
   *   t.currency("currency")
   *   t.quantity("stock")
   *   t.url("image_url").optional()
   *   t.flag("is_featured", false)
   *   t.timestamps()
   * })
   */
  create_table(name: string, fn: (t: TableBuilder) => void): void;

  /**
   * Drops an existing table permanently.
   *
   * Use in down() migrations to reverse create_table().
   * WARNING: All data in the table will be lost.
   *
   * @param name - Table reference to drop
   *
   * @example
   * // In down() function
   * down(m) {
   *   m.drop_table("blog.post")
   *   m.drop_table("blog.category")
   * }
   */
  drop_table(name: string): void;

  /**
   * Renames a table within its namespace.
   *
   * The namespace stays the same, only the table name changes.
   * Foreign keys and indexes are automatically updated.
   *
   * @param oldName - Current table reference (e.g., "blog.post")
   * @param newName - New table name WITHOUT namespace (e.g., "article")
   *
   * @example
   * // Rename blog.post to blog.article
   * m.rename_table("blog.post", "article")
   *
   * @example
   * // Reverse in down()
   * down(m) {
   *   m.rename_table("blog.article", "post")
   * }
   */
  rename_table(oldName: string, newName: string): void;

  // ===========================================================================
  // COLUMN OPERATIONS
  // ===========================================================================

  /**
   * Adds a new column to an existing table.
   *
   * The callback receives a ColumnFactory with all semantic and core types.
   * If adding a NOT NULL column to a table with existing data, use .backfill().
   *
   * @param table - Table reference
   * @param fn - Callback that creates and returns a ColumnBuilder
   *
   * @example
   * // Add semantic type column
   * m.add_column("auth.user", c => c.phone("phone").optional())
   * m.add_column("blog.post", c => c.slug("slug"))
   * m.add_column("shop.product", c => c.money("price"))
   *
   * @example
   * // Add custom column with validation
   * m.add_column("blog.post", c =>
   *   c.string("meta_title", 60).optional()
   * )
   *
   * @example
   * // Add NOT NULL column with backfill for existing rows
   * m.add_column("blog.post", c =>
   *   c.counter("view_count").backfill(0)
   * )
   *
   * @example
   * // Add enum column
   * m.add_column("auth.user", c =>
   *   c.enum("role", ["admin", "user", "guest"]).default("user")
   * )
   */
  add_column(table: string, fn: (c: ColumnFactory) => ColumnBuilder): void;

  /**
   * Removes a column from a table permanently.
   *
   * WARNING: All data in the column will be lost.
   * Use in down() to reverse add_column().
   *
   * @param table - Table reference
   * @param column - Column name to drop
   *
   * @example
   * // Remove a single column
   * m.drop_column("blog.post", "legacy_field")
   * m.drop_column("auth.user", "old_phone")
   *
   * @example
   * // In down() function
   * down(m) {
   *   m.drop_column("auth.user", "phone")
   * }
   */
  drop_column(table: string, column: string): void;

  /**
   * Renames a column within a table.
   *
   * The column type and constraints remain unchanged.
   *
   * @param table - Table reference
   * @param oldName - Current column name
   * @param newName - New column name
   *
   * @example
   * // Rename for clarity
   * m.rename_column("blog.post", "title", "headline")
   * m.rename_column("auth.user", "name", "display_name")
   *
   * @example
   * // Reverse in down()
   * down(m) {
   *   m.rename_column("blog.post", "headline", "title")
   * }
   */
  rename_column(table: string, oldName: string, newName: string): void;

  /**
   * Modifies a column's type or constraints.
   *
   * Use with caution - type changes may require data conversion.
   * PostgreSQL handles most conversions automatically, SQLite may require
   * table recreation.
   *
   * @param table - Table reference
   * @param column - Column name to alter
   * @param fn - Callback receiving an AlterColumnBuilder
   *
   * @example
   * // Expand string length
   * m.alter_column("blog.post", "title", c => {
   *   c.set_type("string", 500)  // Was 200, now 500
   * })
   *
   * @example
   * // Make column nullable
   * m.alter_column("blog.post", "category_id", c => {
   *   c.set_nullable()
   * })
   *
   * @example
   * // Change type from string to text
   * m.alter_column("auth.user", "bio", c => {
   *   c.set_type("text")  // Remove length limit
   * })
   *
   * @example
   * // Add or change default
   * m.alter_column("blog.post", "status", c => {
   *   c.set_default("draft")
   * })
   *
   * @example
   * // Make NOT NULL (ensure no nulls exist first!)
   * m.alter_column("auth.user", "email", c => {
   *   c.set_not_null()
   * })
   */
  alter_column(
    table: string,
    column: string,
    fn: (c: AlterColumnBuilder) => void
  ): void;

  // ===========================================================================
  // INDEXES
  // ===========================================================================

  /**
   * Creates an index on one or more columns.
   *
   * Use indexes to speed up queries on columns frequently used in:
   * - WHERE clauses
   * - JOIN conditions
   * - ORDER BY clauses
   *
   * NOTE: Foreign keys (belongs_to) are automatically indexed - don't duplicate!
   *
   * @param table - Table reference
   * @param columns - Column names to index
   * @param opts - Optional: { unique: boolean, name: string }
   *
   * @example
   * // Simple index on one column
   * m.create_index("blog.post", ["published_at"])
   * m.create_index("blog.post", ["status"])
   *
   * @example
   * // Composite index (order matters!)
   * m.create_index("blog.post", ["author_id", "status"])
   * m.create_index("blog.post", ["status", "published_at"])
   *
   * @example
   * // Unique index
   * m.create_index("blog.post", ["slug"], { unique: true })
   *
   * @example
   * // Custom name
   * m.create_index("blog.post", ["author_id", "published_at"], {
   *   name: "idx_post_author_published"
   * })
   *
   * @example
   * // Join table indexes (m2m)
   * m.create_index("blog_post_blog_tag", ["post_id"])
   * m.create_index("blog_post_blog_tag", ["tag_id"])
   * m.create_index("blog_post_blog_tag", ["post_id", "tag_id"], { unique: true })
   */
  create_index(table: string, columns: string[], opts?: IndexOptions): void;

  /**
   * Drops an index by name.
   *
   * Use in down() to reverse create_index().
   *
   * @param name - Index name to drop
   *
   * @example
   * // Drop custom named index
   * m.drop_index("idx_post_author_published")
   *
   * @example
   * // Drop auto-generated index (check database for exact name)
   * m.drop_index("blog_post_published_at_idx")
   */
  drop_index(name: string): void;

  // ===========================================================================
  // RAW SQL (Use Sparingly)
  // ===========================================================================

  /**
   * Executes raw SQL statement.
   *
   * Use sparingly - prefer typed methods when possible.
   * Useful for database extensions, custom functions, or advanced features.
   *
   * @param sql - Raw SQL statement to execute
   *
   * @example
   * // Enable PostgreSQL extension
   * m.sql("CREATE EXTENSION IF NOT EXISTS pg_trgm")
   *
   * @example
   * // Create custom function
   * m.sql(`
   *   CREATE OR REPLACE FUNCTION update_updated_at()
   *   RETURNS TRIGGER AS $$
   *   BEGIN
   *     NEW.updated_at = NOW();
   *     RETURN NEW;
   *   END;
   *   $$ LANGUAGE plpgsql
   * `)
   *
   * @example
   * // Add trigger
   * m.sql(`
   *   CREATE TRIGGER set_updated_at
   *   BEFORE UPDATE ON blog_post
   *   FOR EACH ROW
   *   EXECUTE FUNCTION update_updated_at()
   * `)
   */
  sql(sql: string): void;

  /**
   * Executes dialect-specific SQL.
   *
   * Use when PostgreSQL and SQLite require different syntax.
   * Empty string means "skip for this database".
   *
   * @param postgres - PostgreSQL SQL statement
   * @param sqlite - SQLite SQL statement
   *
   * @example
   * // PostgreSQL-only extension
   * m.sql_dialect(
   *   "CREATE EXTENSION IF NOT EXISTS pg_trgm",
   *   "" // Skip on SQLite
   * )
   *
   * @example
   * // Different syntax for full-text search
   * m.sql_dialect(
   *   // PostgreSQL: GIN index for full-text
   *   "CREATE INDEX idx_post_fts ON blog_post USING GIN (to_tsvector('english', content))",
   *   // SQLite: FTS5 virtual table
   *   "CREATE VIRTUAL TABLE post_fts USING fts5(content)"
   * )
   *
   * @example
   * // Database-specific optimizations
   * m.sql_dialect(
   *   "ALTER TABLE blog_post SET (autovacuum_enabled = true)",  // PostgreSQL
   *   "PRAGMA optimize"  // SQLite
   * )
   */
  sql_dialect(postgres: string, sqlite: string): void;
}

// ===========================================================================
// MIGRATION DEFINITION
// ===========================================================================

/**
 * Migration definition with up and down functions.
 *
 * Both functions receive a MigrationBuilder with full IntelliSense support.
 */
export interface MigrationDefinition {
  /**
   * Migration up function - applies schema changes forward.
   *
   * @param m - MigrationBuilder with methods for creating tables, columns, indexes, etc.
   */
  up(m: MigrationBuilder): void;

  /**
   * Migration down function - reverses schema changes backward.
   *
   * @param m - MigrationBuilder with methods for dropping tables, columns, indexes, etc.
   */
  down(m: MigrationBuilder): void;
}

/**
 * Defines a database migration with up and down functions.
 *
 * Use this function to wrap your migration definition. It provides full
 * IntelliSense for both up() and down() functions automatically.
 *
 * The up() function runs when migrating forward (applying changes).
 * The down() function runs when rolling back (undoing changes).
 *
 * @param definition - Object with up() and down() functions
 *
 * @example
 * // migrations/001_create_users.js - Complete user table
 * export default migration({
 *   up(m) {
 *     m.create_table("auth.user", t => {
 *       t.id()
 *       t.email("email").unique()
 *       t.username("username").unique()
 *       t.password_hash("password")
 *       t.name("first_name")
 *       t.name("last_name")
 *       t.phone("phone").optional()
 *       t.url("avatar_url").optional()
 *       t.flag("is_active", true)
 *       t.flag("is_verified", false)
 *       t.datetime("last_login").optional()
 *       t.timestamps()
 *     })
 *   },
 *   down(m) {
 *     m.drop_table("auth.user")
 *   }
 * })
 *
 * @example
 * // migrations/002_create_posts_and_tags.js - Multiple tables
 * export default migration({
 *   up(m) {
 *     // Categories first (no dependencies)
 *     m.create_table("blog.category", t => {
 *       t.id()
 *       t.name("name").unique()
 *       t.slug("slug")
 *       t.color("color").optional()
 *       t.timestamps()
 *     })
 *
 *     // Posts (depends on user and category)
 *     m.create_table("blog.post", t => {
 *       t.id()
 *       t.title("title")
 *       t.slug("slug")
 *       t.summary("excerpt").optional()
 *       t.markdown("content")
 *       t.belongs_to("auth.user").as("author")
 *       t.belongs_to("blog.category")
 *       t.enum("status", ["draft", "published", "archived"]).default("draft")
 *       t.counter("view_count")
 *       t.rating("average_rating").optional()
 *       t.datetime("published_at").optional()
 *       t.timestamps()
 *       t.soft_delete()
 *     })
 *
 *     // Tags (no dependencies)
 *     m.create_table("blog.tag", t => {
 *       t.id()
 *       t.name("name").unique()
 *       t.slug("slug")
 *       t.timestamps()
 *     })
 *
 *     // Join table for m2m
 *     m.create_table("blog_post_blog_tag", t => {
 *       t.belongs_to("blog.post")
 *       t.belongs_to("blog.tag")
 *       t.unique("post", "tag")
 *     })
 *
 *     // Indexes for performance
 *     m.create_index("blog.post", ["author_id", "status"])
 *     m.create_index("blog.post", ["published_at"])
 *     m.create_index("blog_post_blog_tag", ["post_id"])
 *     m.create_index("blog_post_blog_tag", ["tag_id"])
 *   },
 *   down(m) {
 *     // Drop in reverse order of creation
 *     m.drop_table("blog_post_blog_tag")
 *     m.drop_table("blog.tag")
 *     m.drop_table("blog.post")
 *     m.drop_table("blog.category")
 *   }
 * })
 *
 * @example
 * // migrations/003_add_user_bio.js - Adding columns
 * export default migration({
 *   up(m) {
 *     m.add_column("auth.user", c => c.text("bio").optional())
 *     m.add_column("auth.user", c => c.country("country").optional())
 *     m.add_column("auth.user", c => c.locale("locale").default("en-US"))
 *   },
 *   down(m) {
 *     m.drop_column("auth.user", "locale")
 *     m.drop_column("auth.user", "country")
 *     m.drop_column("auth.user", "bio")
 *   }
 * })
 *
 * @example
 * // migrations/004_add_products.js - E-commerce
 * export default migration({
 *   up(m) {
 *     m.create_table("shop.product", t => {
 *       t.id()
 *       t.code("sku").unique()
 *       t.title("name")
 *       t.slug("slug")
 *       t.body("description")
 *       t.money("price")
 *       t.currency("currency")
 *       t.percentage("discount").optional()
 *       t.quantity("stock")
 *       t.url("image_url").optional()
 *       t.flag("is_featured", false)
 *       t.flag("is_available", true)
 *       t.sortable()
 *       t.timestamps()
 *     })
 *
 *     m.create_index("shop.product", ["is_featured", "is_available"])
 *   },
 *   down(m) {
 *     m.drop_table("shop.product")
 *   }
 * })
 *
 * @example
 * // migrations/005_polymorphic.js - Polymorphic relationships
 * export default migration({
 *   up(m) {
 *     m.create_table("core.comment", t => {
 *       t.id()
 *       t.belongs_to("auth.user").as("author")
 *       t.text("content")
 *       // Can comment on posts OR videos
 *       t.belongs_to_any(["blog.post", "media.video"], { as: "commentable" })
 *       t.timestamps()
 *     })
 *   },
 *   down(m) {
 *     m.drop_table("core.comment")
 *   }
 * })
 */
declare function migration(definition: MigrationDefinition): void;

export { migration, MigrationBuilder, MigrationDefinition, IndexOptions };
