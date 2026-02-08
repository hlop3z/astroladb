// Rust Axum Code Generator
// Generates Serde models and Axum handlers from your alab schema.
//
// Usage:
//   alab gen run generators/rust-axum -o ./generated

export default gen((schema) => {
  const files = {};

  // ─── Helpers ───────────────────────────────────────────────

  const TYPE_MAP = {
    uuid: "Uuid",
    string: "String",
    text: "String",
    integer: "i64",
    float: "f64",
    boolean: "bool",
    date: "NaiveDate",
    time: "NaiveTime",
    datetime: "DateTime<Utc>",
    decimal: "Decimal",
    json: "serde_json::Value",
    base64: "String",
    enum: "enum",
  };

  const AUTO_FIELDS = new Set(["id", "created_at", "updated_at"]);

  const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
  const pascalCase = (s) => s.split("_").map(capitalize).join("");
  const snakeCase = (s) => s.toLowerCase();

  const rustType = (col) => {
    if (col.type === "enum" && col.enum) {
      return capitalize(col.name);
    }
    return TYPE_MAP[col.type] || "String";
  };

  const rustField = (col) => {
    const typ = rustType(col);
    const wrapper = col.nullable ? `Option<${typ}>` : typ;
    return `    pub ${col.name}: ${wrapper},`;
  };

  // ─── Models File ───────────────────────────────────────────

  const MODELS_IMPORTS = [
    "use chrono::{DateTime, NaiveDate, NaiveTime, Utc};",
    "use rust_decimal::Decimal;",
    "use serde::{Deserialize, Serialize};",
    "use uuid::Uuid;",
    "",
  ].join("\n");

  function modelsFile(tables) {
    let out = MODELS_IMPORTS;

    for (const table of tables) {
      const cls = pascalCase(table.name);

      // Enums
      for (const col of table.columns) {
        if (col.type !== "enum" || !col.enum) continue;
        const enumName = capitalize(col.name);
        out += `\n#[derive(Debug, Clone, Serialize, Deserialize)]\n`;
        out += `pub enum ${enumName} {\n`;
        for (const val of col.enum) {
          out += `    ${capitalize(val)},\n`;
        }
        out += `}\n\n`;
      }

      // Base struct (create/update — no id, no timestamps)
      out += `#[derive(Debug, Clone, Serialize, Deserialize)]\n`;
      out += `pub struct ${cls}Base {\n`;
      const fields = table.columns.filter((c) => !AUTO_FIELDS.has(c.name));
      if (fields.length === 0) {
        out += `    // No user-defined fields\n`;
      } else {
        for (const col of fields) {
          out += `${rustField(col)}\n`;
        }
      }
      out += `}\n\n`;

      // Full struct (adds id + timestamps)
      out += `#[derive(Debug, Clone, Serialize, Deserialize)]\n`;
      out += `pub struct ${cls} {\n`;
      out += `    pub id: Uuid,\n`;
      for (const col of fields) {
        out += `${rustField(col)}\n`;
      }
      if (table.timestamps) {
        out += `    pub created_at: DateTime<Utc>,\n`;
        out += `    pub updated_at: DateTime<Utc>,\n`;
      }
      out += `}\n\n`;
    }

    return out;
  }

  // ─── Handlers File ─────────────────────────────────────────

  function handlersFile(ns, tables) {
    const imports = tables
      .map((t) => `${pascalCase(t.name)}, ${pascalCase(t.name)}Base`)
      .join(", ");

    let out =
      "use axum::{\n" +
      "    extract::{Path, State},\n" +
      "    http::StatusCode,\n" +
      "    response::IntoResponse,\n" +
      "    Json,\n" +
      "};\n" +
      "use uuid::Uuid;\n\n" +
      `use crate::models::{${imports}};\n\n` +
      "// TODO: Add your database/state type\n" +
      "type AppState = ();\n\n";

    for (const table of tables) {
      const cls = pascalCase(table.name);
      const name = table.name;

      out += `// ${name} handlers\n\n`;

      // List
      out += `pub async fn list_${name}(\n`;
      out += `    State(_state): State<AppState>,\n`;
      out += `) -> impl IntoResponse {\n`;
      out += `    // TODO: implement\n`;
      out += `    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))\n`;
      out += `}\n\n`;

      // Get by ID
      out += `pub async fn get_${name}(\n`;
      out += `    State(_state): State<AppState>,\n`;
      out += `    Path(id): Path<Uuid>,\n`;
      out += `) -> impl IntoResponse {\n`;
      out += `    // TODO: implement\n`;
      out += `    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))\n`;
      out += `}\n\n`;

      // Create
      out += `pub async fn create_${name}(\n`;
      out += `    State(_state): State<AppState>,\n`;
      out += `    Json(_body): Json<${cls}Base>,\n`;
      out += `) -> impl IntoResponse {\n`;
      out += `    // TODO: implement\n`;
      out += `    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))\n`;
      out += `}\n\n`;

      // Update
      out += `pub async fn update_${name}(\n`;
      out += `    State(_state): State<AppState>,\n`;
      out += `    Path(id): Path<Uuid>,\n`;
      out += `    Json(_body): Json<${cls}Base>,\n`;
      out += `) -> impl IntoResponse {\n`;
      out += `    // TODO: implement\n`;
      out += `    (StatusCode::NOT_IMPLEMENTED, Json("not implemented"))\n`;
      out += `}\n\n`;

      // Delete
      out += `pub async fn delete_${name}(\n`;
      out += `    State(_state): State<AppState>,\n`;
      out += `    Path(id): Path<Uuid>,\n`;
      out += `) -> impl IntoResponse {\n`;
      out += `    // TODO: implement\n`;
      out += `    StatusCode::NOT_IMPLEMENTED\n`;
      out += `}\n\n`;
    }

    return out;
  }

  // ─── Routes File ───────────────────────────────────────────

  function routesFile(ns, tables) {
    let out = "use axum::{routing::{delete, get, patch, post}, Router};\n\n";
    out += `use crate::${ns}::handlers::*;\n\n`;
    out += `pub fn ${ns}_routes() -> Router {\n`;
    out += `    Router::new()\n`;

    for (const table of tables) {
      const name = table.name;
      const slug = name.replace(/_/g, "-");

      out += `        .route("/${slug}", get(list_${name}).post(create_${name}))\n`;
      out += `        .route("/${slug}/:id", get(get_${name}).patch(update_${name}).delete(delete_${name}))\n`;
    }

    out += `}\n`;
    return out;
  }

  // ─── Main File ─────────────────────────────────────────────

  function mainFile(namespaces) {
    const mods = namespaces.map((ns) => `mod ${ns};`).join("\n");
    const uses = namespaces.map((ns) => `use ${ns}::routes::${ns}_routes;`).join("\n");
    const routes = namespaces.map((ns) => `        .nest("/${ns}", ${ns}_routes())`).join("\n");

    return (
      "//! Axum API (generated by alab)\n" +
      "//!\n" +
      "//! Run:\n" +
      "//!   cargo add axum tokio serde serde_json uuid chrono rust_decimal\n" +
      "//!   cargo add -F tokio/full -F uuid/v4 -F uuid/serde\n" +
      "//!   cargo run\n\n" +
      "mod models;\n" +
      mods + "\n\n" +
      uses + "\n\n" +
      "#[tokio::main]\n" +
      "async fn main() {\n" +
      "    let app = axum::Router::new()\n" +
      routes + "\n" +
      '        .route("/", axum::routing::get(|| async { "Alab Generated API" }));\n\n' +
      '    let listener = tokio::net::TcpListener::bind("127.0.0.1:3000")\n' +
      '        .await\n' +
      '        .unwrap();\n\n' +
      '    println!("✓ Axum server running on http://127.0.0.1:3000");\n' +
      "    axum::serve(listener, app).await.unwrap();\n" +
      "}\n"
    );
  }

  // ─── Generate ──────────────────────────────────────────────

  files["src/models.rs"] = modelsFile(
    Object.values(schema.models).flat()
  );

  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`src/${ns}/mod.rs`] = "pub mod handlers;\npub mod routes;\n";
    files[`src/${ns}/handlers.rs`] = handlersFile(ns, tables);
    files[`src/${ns}/routes.rs`] = routesFile(ns, tables);
  }

  files["src/main.rs"] = mainFile(Object.keys(schema.models));

  // Cargo.toml
  files["Cargo.toml"] = [
    '[package]',
    'name = "alab-api"',
    'version = "0.1.0"',
    'edition = "2021"',
    '',
    '[dependencies]',
    'axum = "0.7"',
    'tokio = { version = "1", features = ["full"] }',
    'serde = { version = "1", features = ["derive"] }',
    'serde_json = "1"',
    'uuid = { version = "1", features = ["v4", "serde"] }',
    'chrono = { version = "0.4", features = ["serde"] }',
    'rust_decimal = "1"',
    '',
  ].join("\n");

  return render(files);
});
