/**
 * Alab Global Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

import { ColBuilder, FnBuilder, FnExpr } from "./column";

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
