/**
 * Alab Generator Type Definitions
 * AUTO-GENERATED - Do not edit. Run 'alab types' to regenerate.
 */

/**
 * OpenAPI x-db column metadata.
 */
export interface XDBColumn {
  readonly sql_type?: { readonly postgres?: string; readonly sqlite?: string };
  readonly semantic?: string;
  readonly default?: any;
  readonly auto_managed?: boolean;
  readonly generated?: boolean;
  readonly ref?: string;
  readonly fk?: string;
  readonly on_delete?: string;
  readonly on_update?: string;
  readonly relation?: string;
  readonly inverse_of?: string;
  readonly virtual?: boolean;
  readonly computed?: any;
  readonly polymorphic?: any;
}

/**
 * OpenAPI x-db table metadata.
 */
export interface XDBTable {
  readonly table: string;
  readonly namespace: string;
  readonly primary_key: readonly string[];
  readonly timestamps?: boolean;
  readonly soft_delete?: boolean;
  readonly auditable?: boolean;
  readonly sort_by?: readonly string[];
  readonly searchable?: readonly string[];
  readonly filterable?: readonly string[];
  readonly indexes?: readonly any[];
  readonly relationships?: any;
  readonly join_table?: any;
}

/**
 * A column definition from the /schemas endpoint example.
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
 * A table definition from the /schemas endpoint example.
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
 * The /schemas endpoint example payload.
 */
export interface SchemaEndpoint {
  readonly models: { readonly [namespace: string]: readonly SchemaTable[] };
}

/**
 * OpenAPI 3.0 schema property.
 */
export interface OpenAPIProperty {
  readonly type?: string;
  readonly format?: string;
  readonly nullable?: boolean;
  readonly readOnly?: boolean;
  readonly enum?: readonly string[];
  readonly maxLength?: number;
  readonly pattern?: string;
  readonly additionalProperties?: boolean;
  readonly "x-db"?: XDBColumn;
}

/**
 * OpenAPI 3.0 component schema (a table).
 */
export interface OpenAPISchema {
  readonly type: string;
  readonly required?: readonly string[];
  readonly properties: { readonly [column: string]: OpenAPIProperty };
  readonly "x-db"?: XDBTable;
}

/**
 * The schema object passed to generators.
 *
 * `schema.openapi` is a full OpenAPI 3.0 spec with `x-db` extensions.
 * `schema.graphql` is the GraphQL SDL string (optional).
 */
export interface GeneratorSchema {
  readonly openapi: {
    readonly openapi: string;
    readonly info: any;
    readonly paths: {
      readonly "/schemas": {
        readonly get: {
          readonly responses: {
            readonly "200": {
              readonly content: {
                readonly "application/json": {
                  readonly example: SchemaEndpoint;
                };
              };
            };
          };
        };
      };
    };
    readonly components: {
      readonly schemas: { readonly [name: string]: OpenAPISchema };
    };
  };
  readonly graphql?: string;
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
 * Receives a callback that gets the frozen schema object.
 * The callback must call `render({...})` to produce output files.
 *
 * @param fn - Generator callback receiving the schema
 * @returns The render output
 *
 * @example
 * export default gen((schema) => {
 *   var api = schema.openapi;
 *   var endpoint = api.paths["/schemas"].get.responses["200"]
 *     .content["application/json"].example;
 *   var namespaces = Object.keys(endpoint.models);
 *   var files = {};
 *   // ... build files ...
 *   return render(files);
 * });
 */
declare function gen(fn: (schema: GeneratorSchema) => RenderOutput): RenderOutput;

/**
 * Iterates over schema.tables and merges results into a single render output.
 *
 * @param schema - The schema object
 * @param fn - Callback receiving each table, returns partial render output
 * @returns Merged render output from all tables
 *
 * @example
 * var files = perTable(schema, function(table) {
 *   return { [table.name + ".txt"]: "Table: " + table.name };
 * });
 */
declare function perTable(
  schema: GeneratorSchema,
  fn: (table: any) => RenderOutput,
): RenderOutput;

/**
 * Iterates over schema.namespaces and merges results into a single render output.
 *
 * @param schema - The schema object
 * @param fn - Callback receiving namespace name and tables, returns partial render output
 * @returns Merged render output from all namespaces
 *
 * @example
 * var files = perNamespace(schema, function(ns, tables) {
 *   return { [ns + "/index.py"]: "# Namespace: " + ns };
 * });
 */
declare function perNamespace(
  schema: GeneratorSchema,
  fn: (namespace: string, tables: any) => RenderOutput,
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
