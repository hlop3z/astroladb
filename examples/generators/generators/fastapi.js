// FastAPI Code Generator
// Generates Pydantic models and CRUD routers from your alab schema.
//
// Usage:
//   alab gen run generators/fastapi -o ./generated

export default gen((schema) => {
  const files = {};

  // ─── Helpers ───────────────────────────────────────────────

  const TYPE_MAP = {
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

  const AUTO_FIELDS = new Set(["id", "created_at", "updated_at"]);

  const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);
  const pascalCase = (s) => s.split("_").map(capitalize).join("");

  const pyType = (col) => {
    if (col.type === "enum" && col.enum) return `${capitalize(col.name)}Enum`;
    return TYPE_MAP[col.type] || "Any";
  };

  // Format a Python default value, or return "" if none.
  const pyDefault = (col) => {
    const def = col.default;
    if (def === undefined) return "";
    if (typeof def === "object") return ""; // SQL expression
    const pt = pyType(col);
    if (col.type === "enum") return ` = ${pt}.${String(def).toUpperCase()}`;
    if (typeof def === "boolean") return ` = ${def ? "True" : "False"}`;
    if (typeof def === "string") return ` = "${def}"`;
    if (col.type === "decimal") return ` = Decimal("${def}")`;
    return ` = ${def}`;
  };

  // Format a single Pydantic field line.
  const pyField = (col) => {
    const pt = pyType(col);
    if (col.nullable) return `    ${col.name}: Optional[${pt}] = None`;
    return `    ${col.name}: ${pt}${pyDefault(col)}`;
  };

  // ─── Models ────────────────────────────────────────────────

  const MODEL_IMPORTS = [
    "from __future__ import annotations",
    "",
    "from datetime import date, time, datetime",
    "from decimal import Decimal",
    "from enum import Enum",
    "from typing import Optional",
    "from uuid import UUID",
    "",
    "from pydantic import BaseModel",
    "",
  ].join("\n");

  function modelFile(tables) {
    let out = MODEL_IMPORTS + "\n";

    for (const table of tables) {
      const cls = pascalCase(table.name);

      // Enums
      for (const col of table.columns) {
        if (col.type !== "enum" || !col.enum) continue;
        const enumName = `${capitalize(col.name)}Enum`;
        out += `\nclass ${enumName}(str, Enum):\n`;
        for (const val of col.enum) {
          out += `    ${val.toUpperCase()} = "${val}"\n`;
        }
        out += "\n";
      }

      // Base model (create/update — no id, no timestamps)
      out += `\nclass ${cls}Base(BaseModel):\n`;
      const fields = table.columns.filter((c) => !AUTO_FIELDS.has(c.name));
      if (fields.length === 0) {
        out += "    pass\n";
      } else {
        for (const col of fields) out += `${pyField(col)}\n`;
      }

      // Read model (adds id + timestamps)
      out += `\n\nclass ${cls}(${cls}Base):\n`;
      out += "    id: UUID\n";
      if (table.timestamps) {
        out += "    created_at: datetime\n";
        out += "    updated_at: datetime\n";
      }
      out += "\n    class Config:\n        from_attributes = True\n\n";
    }

    return out;
  }

  // ─── Router ────────────────────────────────────────────────

  // Each CRUD operation as a data entry — no copy-paste per endpoint.
  const CRUD = [
    {
      method: "get",
      path: "",
      fn: "list",
      args: "",
      resp: "list[$CLS]",
      status: "",
    },
    {
      method: "get",
      path: "/{item_id}",
      fn: "get",
      args: "item_id: UUID",
      resp: "$CLS",
      status: "",
    },
    {
      method: "post",
      path: "",
      fn: "create",
      args: "body: $CLSBase",
      resp: "$CLS",
      status: ", status_code=201",
    },
    {
      method: "patch",
      path: "/{item_id}",
      fn: "update",
      args: "item_id: UUID, body: $CLSBase",
      resp: "$CLS",
      status: "",
    },
    {
      method: "delete",
      path: "/{item_id}",
      fn: "delete",
      args: "item_id: UUID",
      resp: "",
      status: ", status_code=204",
    },
  ];

  function crudBlock(slug, name, cls, op) {
    const resp = op.resp
      ? `, response_model=${op.resp.replace(/\$CLS/g, cls)}`
      : "";
    const args = op.args.replace(/\$CLS/g, cls);
    return (
      `@router.${op.method}("/${slug}${op.path}"${resp}${op.status})\n` +
      `async def ${op.fn}_${name}(${args}):\n` +
      `    # TODO: implement\n` +
      `    raise HTTPException(501, "not implemented")\n\n\n`
    );
  }

  function routerFile(ns, tables) {
    const imports = tables
      .map((t) => pascalCase(t.name))
      .map((cls) => `    ${cls},\n    ${cls}Base,`)
      .join("\n");

    let out =
      "from __future__ import annotations\n\n" +
      "from uuid import UUID\n\n" +
      "from fastapi import APIRouter, HTTPException\n\n" +
      `from .models import (\n${imports}\n)\n\n\n` +
      `router = APIRouter(prefix="/${ns}", tags=["${ns}"])\n\n\n`;

    for (const table of tables) {
      const cls = pascalCase(table.name);
      const slug = table.name.replace(/_/g, "-");
      for (const op of CRUD) out += crudBlock(slug, table.name, cls, op);
    }

    return out;
  }

  // ─── Main app ──────────────────────────────────────────────

  function mainFile(namespaces) {
    const imports = namespaces
      .map((ns) => `from ${ns} import router as ${ns}_router`)
      .join("\n");
    const routers = namespaces
      .map((ns) => `app.include_router(${ns}_router)`)
      .join("\n");

    return (
      '"""\n' +
      "FastAPI app (generated by alab).\n\n" +
      "Run:\n    uv run fastapi dev main.py\n\n" +
      "Then open http://localhost:8000/docs\n" +
      '"""\n\n' +
      "from fastapi import FastAPI\n\n" +
      `${imports}\n\n` +
      'app = FastAPI(title="Alab Generated API")\n\n' +
      `${routers}\n\n\n` +
      '@app.get("/")\n' +
      "async def root():\n" +
      '    return {"message": "Alab Generated API", "docs": "/docs"}\n'
    );
  }

  // ─── Generate ──────────────────────────────────────────────

  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`${ns}/models.py`] = modelFile(tables);
    files[`${ns}/router.py`] = routerFile(ns, tables);
    files[`${ns}/__init__.py`] = "from .router import router  # noqa: F401\n";
  }

  files["main.py"] = mainFile(Object.keys(schema.models));

  return render(files);
});
