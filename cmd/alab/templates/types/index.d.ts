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
export * from "./generator";

// Declare globals for ambient usage in .js files
import { SQLExpr } from "./globals";
import { ColBuilder, FnBuilder } from "./column";
import { TableChain, ColumnDefinitions } from "./schema";
import { MigrationBuilder, MigrationDefinition } from "./migration";
import { GeneratorSchema, SchemaTable, RenderOutput } from "./generator";

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

  /**
   * Expression builder for computed/virtual columns.
   * Use `fn.*` methods to build expressions that reference other columns,
   * perform calculations, and apply functions.
   *
   * @example
   * export default table({
   *   first_name: col.string(50),
   *   last_name: col.string(50),
   *   full_name: col.text().computed(fn.concat(fn.col("first_name"), " ", fn.col("last_name"))),
   * })
   */
  const fn: FnBuilder;

  /** Defines a single table using an object. */
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

  /**
   * Entry point for a code generator.
   * Receives a callback with the schema object.
   * The callback must call render() to produce output files.
   *
   * @example
   * export default gen((schema) => {
   *   const files = {};
   *   for (const [ns, tables] of Object.entries(schema.models)) {
   *     files[`${ns}/output.txt`] = "hello";
   *   }
   *   return render(files);
   * });
   */
  function gen(fn: (schema: GeneratorSchema) => RenderOutput): RenderOutput;

  /** Renders output files. Maps relative paths to string contents. */
  function render(files: RenderOutput): RenderOutput;

  /** Iterates schema.tables, merges results into render output. */
  function perTable(schema: GeneratorSchema, fn: (table: SchemaTable) => RenderOutput): RenderOutput;

  /** Iterates schema.models by namespace, merges results into render output. */
  function perNamespace(schema: GeneratorSchema, fn: (namespace: string, tables: readonly SchemaTable[]) => RenderOutput): RenderOutput;

  /** Converts a value to JSON string. */
  function json(value: any, indent?: string): string;

  /** Removes common leading whitespace from a multi-line string. */
  function dedent(str: string): string;
}
