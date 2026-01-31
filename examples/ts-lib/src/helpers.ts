// helpers.ts — Reusable type mapping and naming utilities

interface Column {
  name: string;
  type: string;
  nullable?: boolean;
  unique?: boolean;
  default?: any;
  enum?: readonly string[];
}

const typeMap: Record<string, string> = {
  uuid: "UUID",
  string: "str",
  text: "str",
  integer: "int",
  float: "float",
  boolean: "bool",
  date: "date",
  time: "time",
  datetime: "datetime",
  decimal: "Decimal",
  json: "dict",
  base64: "str",
  enum: "str",
};

/** Map an alab column type to a Python type string. */
export function mapType(col: Column): string {
  const base = typeMap[col.type] || "Any";
  if (col.type === "enum" && col.enum) {
    return capitalize(col.name) + "Enum";
  }
  return base;
}

/** Format a default value as a Python literal. */
export function formatDefault(col: Column): string | null {
  const def = col.default;
  if (def === undefined) return null;
  if (typeof def === "object") return null; // SQL expressions like {expr: "NOW()"}
  const pt = mapType(col);
  if (col.type === "enum" && col.enum) {
    return pt + "." + String(def).toUpperCase();
  }
  if (typeof def === "string") return '"' + def + '"';
  if (typeof def === "boolean") return def ? "True" : "False";
  if (col.type === "decimal") return 'Decimal("' + def + '")';
  return String(def);
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/** Convert snake_case to PascalCase. */
export function pascalCase(s: string): string {
  return s.split("_").map(capitalize).join("");
}

/** Convert a name to snake_case (lowercased, underscored). */
export function snakeCase(s: string): string {
  return s.replace(/([A-Z])/g, "_$1").toLowerCase().replace(/^_/, "");
}

/** Generate Python enum class lines for enum columns. */
export function buildEnumLines(columns: readonly Column[]): string[] {
  const lines: string[] = [];
  for (const col of columns) {
    if (col.type === "enum" && col.enum) {
      const enumName = capitalize(col.name) + "Enum";
      lines.push("class " + enumName + "(str, Enum):");
      for (const val of col.enum) {
        lines.push("    " + val.toUpperCase() + ' = "' + val + '"');
      }
      lines.push("");
      lines.push("");
    }
  }
  return lines;
}
