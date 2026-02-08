// Go Chi Code Generator (v2 - Clean Templates)
// Generates Go structs and Chi handlers from your alab schema.
//
// Usage:
//   alab gen run generators/go-chi -o ./generated

export default gen((schema) => {
  const files = {};

  // ─── Helpers ───────────────────────────────────────────────

  const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
  const pascalCase = (s) => s.split("_").map(capitalize).join("");

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
  };

  const goType = (col) => {
    const typ = TYPE_MAP[col.type] || "interface{}";
    return col.nullable ? `*${typ}` : typ;
  };

  const goField = (col) =>
    `\t${pascalCase(col.name)} ${goType(col)} \`json:"${col.name}"\``;

  // ─── Templates ─────────────────────────────────────────────

  const modelsFile = (tables) => {
    const models = tables.map((table) => {
      const cls = pascalCase(table.name);
      const fields = table.columns
        .filter((c) => !["id", "created_at", "updated_at"].includes(c.name))
        .map(goField);

      const baseFields = fields.length > 0
        ? fields.join("\n")
        : "\t// No user-defined fields";

      const fullFields = [
        `\tID uuid.UUID \`json:"id"\``,
        ...fields,
        table.timestamps ? `\tCreatedAt time.Time \`json:"created_at"\`\n\tUpdatedAt time.Time \`json:"updated_at"\`` : ""
      ].filter(Boolean).join("\n");

      return `
// ${cls}Base is used for creating and updating ${table.name}
type ${cls}Base struct {
${baseFields}
}

// ${cls} represents a ${table.name} record
type ${cls} struct {
${fullFields}
}
`.trim();
    }).join("\n\n");

    return `
package models

import (
\t"encoding/json"
\t"time"

\t"github.com/google/uuid"
\t"github.com/shopspring/decimal"
)

${models}
`.trim() + "\n";
  };

  const handlersFile = (ns, tables) => {
    const handlers = tables.map((table) => {
      const cls = pascalCase(table.name);
      const name = table.name;

      const listHandler = `
func List${cls}(w http.ResponseWriter, r *http.Request) {
\tw.WriteHeader(http.StatusNotImplemented)
\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}`;

      const getHandler = `
func Get${cls}(w http.ResponseWriter, r *http.Request) {
\tidStr := chi.URLParam(r, "id")
\tid, err := uuid.Parse(idStr)
\tif err != nil {
\t\tw.WriteHeader(http.StatusBadRequest)
\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid uuid"})
\t\treturn
\t}
\t_ = id
\tw.WriteHeader(http.StatusNotImplemented)
\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}`;

      const createHandler = `
func Create${cls}(w http.ResponseWriter, r *http.Request) {
\tvar body models.${cls}Base
\tif err := json.NewDecoder(r.Body).Decode(&body); err != nil {
\t\tw.WriteHeader(http.StatusBadRequest)
\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
\t\treturn
\t}
\t_ = body
\tw.WriteHeader(http.StatusNotImplemented)
\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}`;

      const updateHandler = `
func Update${cls}(w http.ResponseWriter, r *http.Request) {
\tidStr := chi.URLParam(r, "id")
\tid, err := uuid.Parse(idStr)
\tif err != nil {
\t\tw.WriteHeader(http.StatusBadRequest)
\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid uuid"})
\t\treturn
\t}
\tvar body models.${cls}Base
\tif err := json.NewDecoder(r.Body).Decode(&body); err != nil {
\t\tw.WriteHeader(http.StatusBadRequest)
\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
\t\treturn
\t}
\t_, _ = id, body
\tw.WriteHeader(http.StatusNotImplemented)
\tjson.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}`;

      const deleteHandler = `
func Delete${cls}(w http.ResponseWriter, r *http.Request) {
\tidStr := chi.URLParam(r, "id")
\tid, err := uuid.Parse(idStr)
\tif err != nil {
\t\tw.WriteHeader(http.StatusBadRequest)
\t\tjson.NewEncoder(w).Encode(map[string]string{"error": "invalid uuid"})
\t\treturn
\t}
\t_ = id
\tw.WriteHeader(http.StatusNotImplemented)
}`;

      return `
// ${cls} handlers
${listHandler}

${getHandler}

${createHandler}

${updateHandler}

${deleteHandler}
`.trim();
    }).join("\n\n");

    return `
package ${ns}

import (
\t"encoding/json"
\t"net/http"

\t"github.com/go-chi/chi/v5"
\t"github.com/google/uuid"

\t"your-project/models"
)

${handlers}
`.trim() + "\n";
  };

  const routesFile = (ns, tables) => {
    const routes = tables.map((table) => {
      const cls = pascalCase(table.name);
      const slug = table.name.replace(/_/g, "-");

      return `
\t// ${table.name} routes
\tr.Get("/${slug}", List${cls})
\tr.Post("/${slug}", Create${cls})
\tr.Get("/${slug}/{id}", Get${cls})
\tr.Patch("/${slug}/{id}", Update${cls})
\tr.Delete("/${slug}/{id}", Delete${cls})
`.trim();
    }).join("\n\n");

    return `
package ${ns}

import (
\t"github.com/go-chi/chi/v5"
)

func Routes() chi.Router {
\tr := chi.NewRouter()

${routes}

\treturn r
}
`.trim() + "\n";
  };

  const mainFile = (namespaces) => {
    const imports = namespaces.map((ns) => `\t"your-project/${ns}"`).join("\n");
    const mounts = namespaces.map((ns) => `\tr.Mount("/${ns}", ${ns}.Routes())`).join("\n");

    return `
// Alab Generated Chi API
//
// Run:
//   go get github.com/go-chi/chi/v5
//   go get github.com/google/uuid
//   go get github.com/shopspring/decimal
//   go run main.go

package main

import (
\t"fmt"
\t"net/http"

\t"github.com/go-chi/chi/v5"
\t"github.com/go-chi/chi/v5/middleware"

${imports}
)

func main() {
\tr := chi.NewRouter()

\t// Middleware
\tr.Use(middleware.Logger)
\tr.Use(middleware.Recoverer)
\tr.Use(middleware.SetHeader("Content-Type", "application/json"))

\t// Routes
\tr.Get("/", func(w http.ResponseWriter, r *http.Request) {
\t\tw.Write([]byte(\`{"message": "Alab Generated API", "docs": "/docs"}\`))
\t})

${mounts}

\tfmt.Println("✓ Chi server running on http://localhost:3000")
\thttp.ListenAndServe(":3000", r)
}
`.trim() + "\n";
  };

  const goMod = () => `
module your-project

go 1.24

require (
\tgithub.com/go-chi/chi/v5 v5.2.1
\tgithub.com/google/uuid v1.6.0
\tgithub.com/shopspring/decimal v1.4.0
)
`.trim() + "\n";

  // ─── Generate ──────────────────────────────────────────────

  files["models/models.go"] = modelsFile(Object.values(schema.models).flat());

  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`${ns}/handlers.go`] = handlersFile(ns, tables);
    files[`${ns}/routes.go`] = routesFile(ns, tables);
  }

  files["main.go"] = mainFile(Object.keys(schema.models));
  files["go.mod"] = goMod();

  return render(files);
});
