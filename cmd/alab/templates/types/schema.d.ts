/**
 * Alab Schema Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { ColColumnBuilder, ColRelationshipBuilder, PolymorphicOptions } from "./column";

/**
 * Column definitions object for the object-based table API.
 * Keys become column names, values are column builders from col.*.
 */
export type ColumnDefinitions = Record<string, ColColumnBuilder | ColRelationshipBuilder>;

/**
 * Chainable table operations — append after `table({...})`.
 *
 * | Method                         | Effect                           |
 * | ------------------------------ | -------------------------------- |
 * | `.timestamps()`                | Adds `created_at` + `updated_at` |
 * | `.soft_delete()`               | Adds nullable `deleted_at`       |
 * | `.sortable()`                  | Adds `position` integer          |
 * | `.index(...cols)`              | Non-unique index                 |
 * | `.unique(...cols)`             | Composite unique constraint      |
 * | `.many_to_many(ref)`           | Auto join table                  |
 * | `.belongs_to_any(refs, opts?)` | Polymorphic type+id columns      |
 * | `.docs(s)`                     | Table description                |
 * | `.deprecated(s)`               | Mark deprecated                  |
 *
 * @example
 * export default table({
 *   id: col.id(),
 *   email: col.email().unique(),
 * })
 *   .timestamps()
 *   .soft_delete()
 *   .index("email")
 *   .docs("User accounts")
 */
export interface TableChain {
  /** Adds created_at and updated_at timestamp columns.
   * @example
   * table({ id: col.id() }).timestamps()
   */
  timestamps(): TableChain;

  /** Adds a deleted_at nullable timestamp for soft deletion.
   * @example
   * table({ id: col.id() }).timestamps().soft_delete()
   */
  soft_delete(): TableChain;

  /** Adds a position integer column for manual ordering.
   * @example
   * table({ id: col.id(), name: col.string(100) }).sortable()
   */
  sortable(): TableChain;

  /** Adds a non-unique index on columns.
   * @example
   * table({ id: col.id(), email: col.email() }).index("email")
   */
  index(...columns: string[]): TableChain;

  /** Adds a composite unique constraint on multiple columns.
   * @example
   * table({ id: col.id(), org: col.belongs_to("core.org"), name: col.string(100) }).unique("org", "name")
   */
  unique(...columns: string[]): TableChain;

  /** Declares a many-to-many relationship. Creates an auto join table.
   * @example
   * table({ id: col.id(), name: col.string(100) }).many_to_many("blog.tag")
   */
  many_to_many(model: string): TableChain;

  /** Adds a polymorphic reference (type + id columns).
   * @example
   * table({ id: col.id(), body: col.text() }).belongs_to_any(["blog.post", "media.video"], { as: "commentable" })
   */
  belongs_to_any(models: string[], opts?: PolymorphicOptions): TableChain;

  /** Adds documentation for the table.
   * @example
   * table({ id: col.id() }).docs("User accounts")
   */
  docs(description: string): TableChain;

  /** Marks the table as deprecated.
   * @example
   * table({ id: col.id() }).deprecated("Use auth.user_v2 instead")
   */
  deprecated(reason: string): TableChain;
}

/**
 * Defines a single table using an object.
 *
 * @example
 * // schemas/auth/user.js (object API)
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

export { table };
