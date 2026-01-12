/**
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
import { MigrationBuilder, MigrationDefinition } from "./migration";

declare global {
  /** Wraps a string as a raw SQL expression. */
  function sql(expr: string): SQLExpr;

  /**
   * Column factory for the object-based table API.
   * Use `col.*` methods to define columns. Column names come from the object keys.
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

  /**
   * Defines a database migration with up and down functions.
   *
   * Provides full IntelliSense for both up() and down() functions.
   * The up() function applies changes, down() reverses them.
   *
   * @param definition - Object with up() and down() methods
   *
   * @example
   * // migrations/001_create_users.js
   * export default migration({
   *   up(m) {
   *     m.create_table("auth.user", t => {
   *       t.id()
   *       t.email("email").unique()
   *       t.username("username").unique()
   *       t.password_hash("password")
   *       t.timestamps()
   *     })
   *   },
   *   down(m) {
   *     m.drop_table("auth.user")
   *   }
   * })
   */
  function migration(definition: MigrationDefinition): void;
}
