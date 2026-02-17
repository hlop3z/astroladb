/**
 * Alab Global Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { ColBuilder, FnBuilder, FnExpr } from "./column";

/**
 * Per-dialect SQL expression input.
 *
 * Both `postgres` and `sqlite` keys are required so that every schema
 * and migration works identically on both supported databases.
 */
export interface SQLDialects {
  readonly postgres: string;
  readonly sqlite: string;
}

/**
 * Represents a raw SQL expression (returned by `sql()`).
 *
 * Created by the sql() helper function. Used for database-level
 * defaults and computations that can't be expressed as static values.
 */
export interface SQLExpr {
  readonly __sql: true;
  readonly postgres: string;
  readonly sqlite: string;
}

/**
 * Wraps per-dialect strings as a raw SQL expression.
 *
 * Use when you need database-side computation for defaults or backfills.
 * Both `postgres` and `sqlite` keys are required.
 *
 * @param dialects - Object with `postgres` and `sqlite` SQL strings
 * @returns SQLExpr object recognized by Alab
 *
 * @example
 * // Timestamp defaults
 * { expires_at: col.datetime().default(sql({ postgres: "NOW()", sqlite: "CURRENT_TIMESTAMP" })) }
 *
 * @example
 * // UUID generation
 * { token: col.uuid().default(sql({ postgres: "gen_random_uuid()", sqlite: "lower(hex(randomblob(16)))" })) }
 */
declare function sql(dialects: SQLDialects): SQLExpr;

/**
 * Column factory for the object-based table API.
 *
 * Use `col.*` methods to define columns. Column names come from the object keys.
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

/**
 * Expression builder for computed/virtual columns.
 *
 * Use `fn.*` methods to build expressions that reference other columns,
 * perform calculations, and apply functions. Expressions are translated
 * to database-specific SQL by each dialect.
 *
 * @example
 * export default table({
 *   first_name: col.string(50),
 *   last_name: col.string(50),
 *   full_name: col.text().computed(fn.concat(fn.col("first_name"), " ", fn.col("last_name"))),
 *   age: col.integer().computed(fn.years_since(fn.col("birth_date"))).virtual(),
 * })
 */
declare const fn: FnBuilder;

export { sql, col, fn };
