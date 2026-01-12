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
