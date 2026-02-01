/**
 * Alab Generator Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

/**
 * A column definition.
 */
export interface SchemaColumn {
  readonly name: string;
  readonly type: string;
  readonly nullable?: boolean;
  readonly unique?: boolean;
  readonly default?: any;
  readonly enum?: readonly string[];
}

/**
 * A table definition.
 */
export interface SchemaTable {
  readonly name: string;
  readonly table: string;
  readonly primary_key: string;
  readonly timestamps?: boolean;
  readonly columns: readonly SchemaColumn[];
  readonly example?: Record<string, any>;
}

/**
 * The schema object passed to generators.
 *
 * `schema.models` maps namespace names to their table arrays.
 * `schema.tables` is a flat array of all tables across namespaces.
 */
export interface GeneratorSchema {
  /** Tables grouped by namespace. */
  readonly models: { readonly [namespace: string]: readonly SchemaTable[] };
  /** Flat array of all tables. */
  readonly tables: readonly SchemaTable[];
}

/**
 * The output of render(). Maps relative file paths to string contents.
 */
export type RenderOutput = { [path: string]: string };

/**
 * Renders output files from a generator.
 *
 * Every key must be a relative path (no `..`, no absolute, no dotfiles).
 * Every value must be a string.
 *
 * @param files - Object mapping relative file paths to string contents
 * @returns The same files object
 *
 * @example
 * render({
 *   "models.py": "class User: ...",
 *   "routers/auth.py": "from fastapi import ...",
 * })
 */
declare function render(files: RenderOutput): RenderOutput;

/**
 * Entry point for a code generator.
 *
 * Receives a callback that gets the schema object.
 * The callback must call `render({...})` to produce output files.
 *
 * @param fn - Generator callback receiving the schema
 * @returns The render output
 *
 * @example
 * export default gen((schema) => {
 *   const files = {};
 *   for (const [ns, tables] of Object.entries(schema.models)) {
 *     files[`${ns}/models.py`] = buildModels(tables);
 *   }
 *   return render(files);
 * });
 */
declare function gen(
  fn: (schema: GeneratorSchema) => RenderOutput,
): RenderOutput;

/**
 * Iterates over schema.tables and merges results into a single render output.
 *
 * @param schema - The schema object
 * @param fn - Callback receiving each table, returns partial render output
 * @returns Merged render output from all tables
 *
 * @example
 * const files = perTable(schema, (table) => {
 *   return { [`${table.name}.txt`]: `Table: ${table.name}` };
 * });
 */
declare function perTable(
  schema: GeneratorSchema,
  fn: (table: SchemaTable) => RenderOutput,
): RenderOutput;

/**
 * Iterates over schema.models and merges results into a single render output.
 *
 * @param schema - The schema object
 * @param fn - Callback receiving namespace name and tables, returns partial render output
 * @returns Merged render output from all namespaces
 *
 * @example
 * const files = perNamespace(schema, (ns, tables) => {
 *   return { [`${ns}/index.py`]: `# Namespace: ${ns}` };
 * });
 */
declare function perNamespace(
  schema: GeneratorSchema,
  fn: (namespace: string, tables: readonly SchemaTable[]) => RenderOutput,
): RenderOutput;

/**
 * Converts a value to a JSON string.
 *
 * @param value - The value to serialize
 * @param indent - Optional indentation string (e.g. "  " for 2-space indent)
 * @returns JSON string
 *
 * @example
 * json({ key: "value" })         // '{"key":"value"}'
 * json({ key: "value" }, "  ")   // '{\n  "key": "value"\n}'
 */
declare function json(value: any, indent?: string): string;

/**
 * Removes common leading whitespace from a multi-line string.
 *
 * @param str - The string to dedent
 * @returns Dedented string
 *
 * @example
 * dedent("    line1\n    line2")  // "line1\nline2"
 */
declare function dedent(str: string): string;

export { gen, render, perTable, perNamespace, json, dedent };
