/*
 * Alab Migration Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { MigrationTableBuilder, ColumnBuilder, ColumnFactory, AlterColumnBuilder } from "./column";

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
 * IMPORTANT: Migrations use LOW-LEVEL TYPES ONLY.
 * Semantic types (email, money, flag, etc.) are NOT available in migrations.
 * This ensures migrations are explicit and stable even if semantic definitions change.
 *
 * Available types in create_table:
 * - Core: id(), string(n), text(), integer(), float(), decimal(p,s), boolean(), date(), time(), datetime(), uuid(), json(), base64(), enum()
 * - Relationships: belongs_to(), one_to_one(), many_to_many(), belongs_to_any()
 * - Patterns: timestamps(), soft_delete(), sortable()
 * - Constraints: index(), unique()
 *
 * For semantic types, define schemas in schema files and let Alab generate migrations.
 *
 * #### Operations
 *
 * | Method                           | Purpose                          |
 * | -------------------------------- | -------------------------------- |
 * | `create_table(ref, fn)`          | New table (low-level types only) |
 * | `drop_table(ref)`                | Remove table                     |
 * | `rename_table(ref, new)`         | Rename table                     |
 * | `add_column(ref, fn)`            | Add column                       |
 * | `drop_column(ref, col)`          | Remove column                    |
 * | `rename_column(ref, old, new)`   | Rename column                    |
 * | `alter_column(ref, col, fn)`     | Modify type/null/default         |
 * | `create_index(ref, cols, opts?)` | Add index                        |
 * | `drop_index(name)`               | Remove index                     |
 * | `add_foreign_key(...)`           | Add FK constraint                |
 * | `drop_foreign_key(ref, name)`    | Remove FK                        |
 * | `add_check(ref, name, expr)`     | Add CHECK                        |
 * | `drop_check(ref, name)`          | Remove CHECK                     |
 * | `sql(up, down?)`                 | Raw SQL escape hatch             |
 *
 * @example
 * // migrations/001_create_users.js - Using low-level types
 * export default migration({
 *   up(m) {
 *     m.create_table("auth.user", t => {
 *       t.id()                                    // UUID primary key
 *       t.string("email", 255).unique()           // Use string, not email()
 *       t.string("username", 50).unique()         // Use string, not username()
 *       t.string("password", 255)                 // Use string, not password_hash()
 *       t.string("phone", 50).optional()          // Use string, not phone()
 *       t.boolean("is_active").default(true)      // Use boolean, not flag()
 *       t.datetime("last_login").optional()
 *       t.timestamps()                            // created_at, updated_at
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
 *       t.string("title", 200)
 *       t.string("slug", 255).unique()
 *       t.text("content")
 *       t.string("excerpt", 500).optional()
 *       t.belongs_to("auth.user").as("author")
 *       t.enum("status", ["draft", "published", "archived"]).default("draft")
 *       t.integer("view_count").default(0)
 *       t.decimal("average_rating", 2, 1).optional()
 *       t.datetime("published_at").optional()
 *       t.timestamps()
 *       t.soft_delete()
 *     })
 *
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
 *       t.string("name", 100).unique()
 *       t.string("slug", 255).unique()
 *       t.string("color", 7).optional()
 *       t.timestamps()
 *     })
 *
 *     // Join table (many-to-many)
 *     m.create_table("blog_post_blog_tag", t => {
 *       t.belongs_to("blog.post")
 *       t.belongs_to("blog.tag")
 *       t.unique("post", "tag")
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
 *       t.string("sku", 20).unique()
 *       t.string("name", 200)
 *       t.string("slug", 255).unique()
 *       t.text("description")
 *       t.decimal("price", 19, 4)
 *       t.string("currency", 3)
 *       t.integer("stock").default(0)
 *       t.decimal("discount", 5, 2).optional()
 *       t.string("image_url", 2048).optional()
 *       t.belongs_to("shop.category")
 *       t.boolean("is_featured").default(false)
 *       t.sortable()
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
  // ── Table Operations ──

  /**
   * Creates a new table with columns.
   *
   * The callback receives a MigrationTableBuilder with LOW-LEVEL types only:
   * - Use core types: string(n), text(), integer(), decimal(p,s), boolean(), etc.
   * - Chain modifiers: .unique(), .optional(), .default()
   * - Add relationships: .belongs_to(), .one_to_one(), .many_to_many()
   * - Use helpers: .timestamps(), .soft_delete(), .sortable()
   *
   * NOTE: Semantic types (email, money, flag, etc.) are NOT available.
   *
   * @param name - Table reference: "namespace.table" or just "table"
   * @param fn - Callback receiving a MigrationTableBuilder (t parameter)
   *
   * @example
   * // User authentication table
   * m.create_table("auth.user", t => {
   *   t.id()
   *   t.string("email", 255).unique()
   *   t.string("username", 50).unique()
   *   t.string("password", 255)
   *   t.boolean("is_active").default(true)
   *   t.timestamps()
   * })
   *
   * @example
   * // Blog post with relationships
   * m.create_table("blog.post", t => {
   *   t.id()
   *   t.string("title", 200)
   *   t.string("slug", 255).unique()
   *   t.text("content")
   *   t.belongs_to("auth.user").as("author")
   *   t.belongs_to("blog.category")
   *   t.enum("status", ["draft", "published"]).default("draft")
   *   t.datetime("published_at").optional()
   *   t.integer("view_count").default(0)
   *   t.timestamps()
   *   t.soft_delete()
   * })
   *
   * @example
   * // Product catalog
   * m.create_table("shop.product", t => {
   *   t.id()
   *   t.string("sku", 20).unique()
   *   t.string("name", 200)
   *   t.text("description")
   *   t.decimal("price", 19, 4)
   *   t.string("currency", 3)
   *   t.integer("stock").default(0)
   *   t.string("image_url", 2048).optional()
   *   t.boolean("is_featured").default(false)
   *   t.timestamps()
   * })
   */
  create_table(name: string, fn: (t: MigrationTableBuilder) => void): void;

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

  // ── Column Operations ──

  /**
   * Adds a new column to an existing table.
   *
   * NOTE: Only low-level types are available (string, integer, text, etc.).
   * Semantic types (email, phone, money, etc.) are only available in create_table().
   * If adding a NOT NULL column to a table with existing data, use .backfill().
   *
   * @param table - Table reference
   * @param fn - Callback that creates and returns a ColumnBuilder
   *
   * @example
   * // Add string column
   * m.add_column("auth.user", c => c.string("phone", 50).optional())
   *
   * @example
   * // Add text column
   * m.add_column("blog.post", c => c.text("meta_description").optional())
   *
   * @example
   * // Add NOT NULL column with backfill for existing rows
   * m.add_column("blog.post", c =>
   *   c.integer("view_count").default(0).backfill(0)
   * )
   *
   * @example
   * // Add enum column
   * m.add_column("auth.user", c =>
   *   c.enum("role", ["admin", "user", "guest"]).default("user")
   * )
   *
   * @example
   * // Add decimal for money (semantic equivalent)
   * m.add_column("shop.product", c =>
   *   c.decimal("price", 19, 4).min(0)
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

  // ── Indexes ──

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

  // ── Raw SQL ──

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

  // ── Foreign Keys (PostgreSQL / CockroachDB only) ──

  /**
   * Adds a foreign key constraint to an existing table.
   *
   * NOTE: Not supported on SQLite. Use belongs_to() in create_table for
   * automatic foreign keys instead.
   *
   * @param table - Table reference containing the foreign key columns
   * @param columns - Column names in the source table
   * @param refTable - Referenced table reference
   * @param refColumns - Referenced column names
   * @param opts - Optional: { name, on_delete, on_update }
   *
   * @example
   * m.add_foreign_key(
   *   "blog.comment", ["post_id"],
   *   "blog.post", ["id"],
   *   { name: "fk_comment_post", on_delete: "CASCADE" }
   * )
   */
  add_foreign_key(
    table: string,
    columns: string[],
    refTable: string,
    refColumns: string[],
    opts?: { name?: string; on_delete?: string; on_update?: string },
  ): void;

  /**
   * Drops a foreign key constraint by name.
   *
   * @param table - Table reference
   * @param name - Constraint name to drop
   *
   * @example
   * m.drop_foreign_key("blog.comment", "fk_comment_post")
   */
  drop_foreign_key(table: string, name: string): void;

  // ── Check Constraints (PostgreSQL / CockroachDB only) ──

  /**
   * Adds a CHECK constraint to a table.
   *
   * NOTE: Not supported on SQLite.
   *
   * @param table - Table reference
   * @param name - Constraint name
   * @param expression - SQL expression that must evaluate to true
   *
   * @example
   * m.add_check("auth.user", "chk_email_length", "length(email) > 0")
   * m.add_check("shop.product", "chk_price_positive", "price >= 0")
   */
  add_check(table: string, name: string, expression: string): void;

  /**
   * Drops a CHECK constraint by name.
   *
   * @param table - Table reference
   * @param name - Constraint name to drop
   *
   * @example
   * m.drop_check("auth.user", "chk_email_length")
   */
  drop_check(table: string, name: string): void;
}

// ── Migration Definition ──

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
 * // migrations/001_create_users.js - Complete user table (low-level types)
 * export default migration({
 *   up(m) {
 *     m.create_table("auth.user", t => {
 *       t.id()
 *       t.string("email", 255).unique()
 *       t.string("username", 50).unique()
 *       t.string("password", 255)
 *       t.string("first_name", 100)
 *       t.string("last_name", 100)
 *       t.string("phone", 50).optional()
 *       t.string("avatar_url", 2048).optional()
 *       t.boolean("is_active").default(true)
 *       t.boolean("is_verified").default(false)
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
 *       t.string("name", 100).unique()
 *       t.string("slug", 255).unique()
 *       t.string("color", 7).optional()
 *       t.timestamps()
 *     })
 *
 *     // Posts (depends on user and category)
 *     m.create_table("blog.post", t => {
 *       t.id()
 *       t.string("title", 200)
 *       t.string("slug", 255).unique()
 *       t.string("excerpt", 500).optional()
 *       t.text("content")
 *       t.belongs_to("auth.user").as("author")
 *       t.belongs_to("blog.category")
 *       t.enum("status", ["draft", "published", "archived"]).default("draft")
 *       t.integer("view_count").default(0)
 *       t.decimal("average_rating", 2, 1).optional()
 *       t.datetime("published_at").optional()
 *       t.timestamps()
 *       t.soft_delete()
 *     })
 *
 *     // Tags (no dependencies)
 *     m.create_table("blog.tag", t => {
 *       t.id()
 *       t.string("name", 100).unique()
 *       t.string("slug", 255).unique()
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
 * // migrations/003_add_user_bio.js - Adding columns (low-level types only)
 * export default migration({
 *   up(m) {
 *     m.add_column("auth.user", c => c.text("bio").optional())
 *     m.add_column("auth.user", c => c.string("country", 2).optional())
 *     m.add_column("auth.user", c => c.string("locale", 10).default("en-US"))
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
 *       t.string("sku", 20).unique()
 *       t.string("name", 200)
 *       t.string("slug", 255).unique()
 *       t.text("description")
 *       t.decimal("price", 19, 4)
 *       t.string("currency", 3)
 *       t.decimal("discount", 5, 2).optional()
 *       t.integer("stock").default(0)
 *       t.string("image_url", 2048).optional()
 *       t.boolean("is_featured").default(false)
 *       t.boolean("is_available").default(true)
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
