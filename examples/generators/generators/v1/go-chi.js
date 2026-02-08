// Go Chi Code Generator
// Generates Go structs and Chi handlers from your alab schema.
//
// Usage:
//   alab gen run generators/go-chi -o ./generated

export default gen((schema) => {
  const files = {};

  // ─── Helpers ───────────────────────────────────────────────

  const TYPE_MAP = {
    uuid: "uuid.UUID",
    string: "string",
    text: "string",
    integer: "int64",
    float: "float64",
    boolean: "bool",
    date: "time.Time",
    time: "time.Time",
    datetime: "time.Time",
    decimal: "decimal.Decimal",
    json: "json.RawMessage",
    base64: "string",
    enum: "string",
  };

  const AUTO_FIELDS = new Set(["id", "created_at", "updated_at"]);

  const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
  const pascalCase = (s) => s.split("_").map(capitalize).join("");
  const lowerFirst = (s) => s.charAt(0).toLowerCase() + s.slice(1);

  const goType = (col) => {
    return TYPE_MAP[col.type] || "interface{}";
  };

  const goField = (col) => {
    const typ = goType(col);
    const name = pascalCase(col.name);
    const jsonTag = `\`json:"${col.name}"\``;

    if (col.nullable) {
      return `\t${name} *${typ} ${jsonTag}`;
    }
    return `\t${name} ${typ} ${jsonTag}`;
  };

  // ─── Models File ───────────────────────────────────────────

  const MODELS_IMPORTS = [
    'package models',
    '',
    'import (',
    '\t"encoding/json"',
    '\t"time"',
    '',
    '\t"github.com/google/uuid"',
    '\t"github.com/shopspring/decimal"',
    ')',
    '',
  ].join("\n");

  function modelsFile(tables) {
    let out = MODELS_IMPORTS;

    for (const table of tables) {
      const cls = pascalCase(table.name);

      // Base struct (create/update — no id, no timestamps)
      out += `// ${cls}Base is used for creating and updating ${table.name}\n`;
      out += `type ${cls}Base struct {\n`;

      const fields = table.columns.filter((c) => !AUTO_FIELDS.has(c.name));
      if (fields.length === 0) {
        out += `\t// No user-defined fields\n`;
      } else {
        for (const col of fields) {
          out += `${goField(col)}\n`;
        }
      }
      out += `}\n\n`;

      // Full struct (adds id + timestamps)
      out += `// ${cls} represents a ${table.name} record\n`;
      out += `type ${cls} struct {\n`;
      out += `\tID uuid.UUID \`json:"id"\`\n`;

      for (const col of fields) {
        out += `${goField(col)}\n`;
      }

      if (table.timestamps) {
        out += `\tCreatedAt time.Time \`json:"created_at"\`\n`;
        out += `\tUpdatedAt time.Time \`json:"updated_at"\`\n`;
      }
      out += `}\n\n`;
    }

    return out;
  }

  // ─── Handlers File ─────────────────────────────────────────

  function handlersFile(ns, tables) {
    const imports = tables
      .map((t) => pascalCase(t.name))
      .join(", ");

    let out =
      `package ${ns}\n\n` +
      'import (\n' +
      '\t"encoding/json"\n' +
      '\t"net/http"\n' +
      '\n' +
      '\t"github.com/go-chi/chi/v5"\n' +
      '\t"github.com/google/uuid"\n' +
      '\n' +
      '\t"your-project/models"\n' +
      ')\n\n';

    for (const table of tables) {
      const cls = pascalCase(table.name);
      const name = table.name;

      out += `// ${cls} handlers\n\n`;

      // List
      out += `func List${cls}(w http.ResponseWriter, r *http.Request) {\n`;
      out += `\t// TODO: implement\n`;
      out += `\tw.WriteHeader(http.StatusNotImplemented)\n`;
      out += `\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})\n`;
      out += `}\n\n`;

      // Get by ID
      out += `func Get${cls}(w http.ResponseWriter, r *http.Request) {\n`;
      out += `\tidStr := chi.URLParam(r, "id")\n`;
      out += `\tid, err := uuid.Parse(idStr)\n`;
      out += `\tif err != nil {\n`;
      out += `\t\tw.WriteHeader(http.StatusBadRequest)\n`;
      out += `\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid uuid"})\n`;
      out += `\t\treturn\n`;
      out += `\t}\n`;
      out += `\t_ = id // TODO: use id to fetch record\n`;
      out += `\tw.WriteHeader(http.StatusNotImplemented)\n`;
      out += `\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})\n`;
      out += `}\n\n`;

      // Create
      out += `func Create${cls}(w http.ResponseWriter, r *http.Request) {\n`;
      out += `\tvar body models.${cls}Base\n`;
      out += `\tif err := json.NewDecoder(r.Body).Decode(&body); err != nil {\n`;
      out += `\t\tw.WriteHeader(http.StatusBadRequest)\n`;
      out += `\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})\n`;
      out += `\t\treturn\n`;
      out += `\t}\n`;
      out += `\t_ = body // TODO: create record\n`;
      out += `\tw.WriteHeader(http.StatusNotImplemented)\n`;
      out += `\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})\n`;
      out += `}\n\n`;

      // Update
      out += `func Update${cls}(w http.ResponseWriter, r *http.Request) {\n`;
      out += `\tidStr := chi.URLParam(r, "id")\n`;
      out += `\tid, err := uuid.Parse(idStr)\n`;
      out += `\tif err != nil {\n`;
      out += `\t\tw.WriteHeader(http.StatusBadRequest)\n`;
      out += `\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid uuid"})\n`;
      out += `\t\treturn\n`;
      out += `\t}\n`;
      out += `\tvar body models.${cls}Base\n`;
      out += `\tif err := json.NewDecoder(r.Body).Decode(&body); err != nil {\n`;
      out += `\t\tw.WriteHeader(http.StatusBadRequest)\n`;
      out += `\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})\n`;
      out += `\t\treturn\n`;
      out += `\t}\n`;
      out += `\t_, _ = id, body // TODO: update record\n`;
      out += `\tw.WriteHeader(http.StatusNotImplemented)\n`;
      out += `\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})\n`;
      out += `}\n\n`;

      // Delete
      out += `func Delete${cls}(w http.ResponseWriter, r *http.Request) {\n`;
      out += `\tidStr := chi.URLParam(r, "id")\n`;
      out += `\tid, err := uuid.Parse(idStr)\n`;
      out += `\tif err != nil {\n`;
      out += `\t\tw.WriteHeader(http.StatusBadRequest)\n`;
      out += `\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid uuid"})\n`;
      out += `\t\treturn\n`;
      out += `\t}\n`;
      out += `\t_ = id // TODO: delete record\n`;
      out += `\tw.WriteHeader(http.StatusNotImplemented)\n`;
      out += `}\n\n`;
    }

    return out;
  }

  // ─── Routes File ───────────────────────────────────────────

  function routesFile(ns, tables) {
    let out =
      `package ${ns}\n\n` +
      'import (\n' +
      '\t"github.com/go-chi/chi/v5"\n' +
      ')\n\n' +
      `func Routes() chi.Router {\n` +
      `\tr := chi.NewRouter()\n\n`;

    for (const table of tables) {
      const cls = pascalCase(table.name);
      const slug = table.name.replace(/_/g, "-");

      out += `\t// ${table.name} routes\n`;
      out += `\tr.Get("/${slug}", List${cls})\n`;
      out += `\tr.Post("/${slug}", Create${cls})\n`;
      out += `\tr.Get("/${slug}/{id}", Get${cls})\n`;
      out += `\tr.Patch("/${slug}/{id}", Update${cls})\n`;
      out += `\tr.Delete("/${slug}/{id}", Delete${cls})\n\n`;
    }

    out += `\treturn r\n`;
    out += `}\n`;
    return out;
  }

  // ─── Main File ─────────────────────────────────────────────

  function mainFile(namespaces) {
    const imports = namespaces
      .map((ns) => `\t"your-project/${ns}"`)
      .join("\n");
    const mounts = namespaces
      .map((ns) => `\tr.Mount("/${ns}", ${ns}.Routes())`)
      .join("\n");

    return (
      '// Alab Generated Chi API\n' +
      '//\n' +
      '// Run:\n' +
      '//   go get github.com/go-chi/chi/v5\n' +
      '//   go get github.com/google/uuid\n' +
      '//   go get github.com/shopspring/decimal\n' +
      '//   go run main.go\n' +
      '\n' +
      'package main\n\n' +
      'import (\n' +
      '\t"fmt"\n' +
      '\t"net/http"\n\n' +
      '\t"github.com/go-chi/chi/v5"\n' +
      '\t"github.com/go-chi/chi/v5/middleware"\n\n' +
      imports + '\n' +
      ')\n\n' +
      'func main() {\n' +
      '\tr := chi.NewRouter()\n\n' +
      '\t// Middleware\n' +
      '\tr.Use(middleware.Logger)\n' +
      '\tr.Use(middleware.Recoverer)\n' +
      '\tr.Use(middleware.SetHeader("Content-Type", "application/json"))\n\n' +
      '\t// Routes\n' +
      '\tr.Get("/", func(w http.ResponseWriter, r *http.Request) {\n' +
      '\t\tw.Write([]byte(`{"message": "Alab Generated API", "docs": "/docs"}`))\n' +
      '\t})\n\n' +
      mounts + '\n\n' +
      '\tfmt.Println("✓ Chi server running on http://localhost:3000")\n' +
      '\thttp.ListenAndServe(":3000", r)\n' +
      '}\n'
    );
  }

  // ─── Generate ──────────────────────────────────────────────

  files["models/models.go"] = modelsFile(
    Object.values(schema.models).flat()
  );

  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`${ns}/handlers.go`] = handlersFile(ns, tables);
    files[`${ns}/routes.go`] = routesFile(ns, tables);
  }

  files["main.go"] = mainFile(Object.keys(schema.models));

  // go.mod
  files["go.mod"] = [
    'module your-project',
    '',
    'go 1.24',
    '',
    'require (',
    '\tgithub.com/go-chi/chi/v5 v5.2.1',
    '\tgithub.com/google/uuid v1.6.0',
    '\tgithub.com/shopspring/decimal v1.4.0',
    ')',
    '',
  ].join("\n");

  return render(files);
});
