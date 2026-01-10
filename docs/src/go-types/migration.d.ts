/**
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
