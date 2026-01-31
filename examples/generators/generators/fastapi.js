// FastAPI Code Generator
// Generates Pydantic models and CRUD routers from your alab schema.
//
// Usage:
//   alab gen run generators/fastapi -o ./generated

export default gen((schema) => {
  var api = schema.openapi;
  var endpoint = api.paths["/schemas"].get.responses["200"].content["application/json"].example;
  var namespaces = Object.keys(endpoint.models);
  var files = {};

  // --- Type mapping ---
  var typeMap = {
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

  function pyType(col) {
    var base = typeMap[col.type] || "Any";
    if (col.type === "enum" && col["enum"]) {
      return capitalize(col.name) + "Enum";
    }
    return base;
  }

  function capitalize(s) {
    return s.charAt(0).toUpperCase() + s.slice(1);
  }

  function pascalCase(s) {
    var parts = s.split("_");
    var result = "";
    for (var i = 0; i < parts.length; i++) {
      result += capitalize(parts[i]);
    }
    return result;
  }

  // --- Generate per namespace ---
  for (var n = 0; n < namespaces.length; n++) {
    var ns = namespaces[n];
    var tables = endpoint.models[ns];

    // --- Models file ---
    var modelLines = [];
    modelLines.push("from __future__ import annotations");
    modelLines.push("");
    modelLines.push("from datetime import date, time, datetime");
    modelLines.push("from decimal import Decimal");
    modelLines.push("from enum import Enum");
    modelLines.push("from typing import Optional");
    modelLines.push("from uuid import UUID");
    modelLines.push("");
    modelLines.push("from pydantic import BaseModel");
    modelLines.push("");
    modelLines.push("");

    for (var t = 0; t < tables.length; t++) {
      var table = tables[t];
      var className = pascalCase(table.name);
      var columns = table.columns;

      // Generate enums first
      for (var c = 0; c < columns.length; c++) {
        var col = columns[c];
        if (col.type === "enum" && col["enum"]) {
          var enumName = capitalize(col.name) + "Enum";
          modelLines.push("class " + enumName + "(str, Enum):");
          var vals = col["enum"];
          for (var v = 0; v < vals.length; v++) {
            modelLines.push("    " + vals[v].toUpperCase() + ' = "' + vals[v] + '"');
          }
          modelLines.push("");
          modelLines.push("");
        }
      }

      // Base model (for create/update — no id, no timestamps)
      modelLines.push("class " + className + "Base(BaseModel):");
      var baseFields = [];
      for (var c = 0; c < columns.length; c++) {
        var col = columns[c];
        if (col.name === "id" || col.name === "created_at" || col.name === "updated_at") {
          continue;
        }
        var pt = pyType(col);
        var line = "    " + col.name + ": ";
        if (col.nullable) {
          line += "Optional[" + pt + "] = None";
        } else if (col["default"] !== undefined) {
          var def = col["default"];
          if (typeof def === "object") {
            // Skip SQL expressions like {expr: "NOW()"}
            line += pt;
          } else if (col.type === "enum" && col["enum"]) {
            // Use enum member as default
            line += pt + " = " + pt + "." + String(def).toUpperCase();
          } else if (typeof def === "string") {
            line += pt + ' = "' + def + '"';
          } else if (typeof def === "boolean") {
            line += pt + " = " + (def ? "True" : "False");
          } else if (col.type === "decimal") {
            line += pt + ' = Decimal("' + def + '")';
          } else {
            line += pt + " = " + def;
          }
        } else {
          line += pt;
        }
        modelLines.push(line);
        baseFields.push(col.name);
      }
      if (baseFields.length === 0) {
        modelLines.push("    pass");
      }
      modelLines.push("");
      modelLines.push("");

      // Read model (includes id + timestamps)
      modelLines.push("class " + className + "(" + className + "Base):");
      modelLines.push("    id: UUID");
      if (table.timestamps) {
        modelLines.push("    created_at: datetime");
        modelLines.push("    updated_at: datetime");
      }
      modelLines.push("");
      modelLines.push("    class Config:");
      modelLines.push("        from_attributes = True");
      modelLines.push("");
      modelLines.push("");
    }

    files[ns + "/models.py"] = modelLines.join("\n");

    // --- Router file ---
    var routerLines = [];
    routerLines.push("from __future__ import annotations");
    routerLines.push("");
    routerLines.push("from uuid import UUID");
    routerLines.push("");
    routerLines.push("from fastapi import APIRouter, HTTPException");
    routerLines.push("");
    routerLines.push("from .models import (");
    for (var t = 0; t < tables.length; t++) {
      var className = pascalCase(tables[t].name);
      routerLines.push("    " + className + ",");
      routerLines.push("    " + className + "Base,");
    }
    routerLines.push(")");
    routerLines.push("");
    routerLines.push("");
    routerLines.push('router = APIRouter(prefix="/' + ns + '", tags=["' + ns + '"])');
    routerLines.push("");
    routerLines.push("");

    for (var t = 0; t < tables.length; t++) {
      var table = tables[t];
      var className = pascalCase(table.name);
      var slug = table.name.replace(/_/g, "-");

      routerLines.push('@router.get("/' + slug + '", response_model=list[' + className + "])");
      routerLines.push("async def list_" + table.name + "():");
      routerLines.push("    # TODO: implement query");
      routerLines.push('    raise HTTPException(501, "not implemented")');
      routerLines.push("");
      routerLines.push("");

      routerLines.push('@router.get("/' + slug + '/{item_id}", response_model=' + className + ")");
      routerLines.push("async def get_" + table.name + "(item_id: UUID):");
      routerLines.push("    # TODO: implement query");
      routerLines.push('    raise HTTPException(501, "not implemented")');
      routerLines.push("");
      routerLines.push("");

      routerLines.push('@router.post("/' + slug + '", response_model=' + className + ", status_code=201)");
      routerLines.push("async def create_" + table.name + "(body: " + className + "Base):");
      routerLines.push("    # TODO: implement insert");
      routerLines.push('    raise HTTPException(501, "not implemented")');
      routerLines.push("");
      routerLines.push("");

      routerLines.push('@router.patch("/' + slug + '/{item_id}", response_model=' + className + ")");
      routerLines.push("async def update_" + table.name + "(item_id: UUID, body: " + className + "Base):");
      routerLines.push("    # TODO: implement update");
      routerLines.push('    raise HTTPException(501, "not implemented")');
      routerLines.push("");
      routerLines.push("");

      routerLines.push('@router.delete("/' + slug + '/{item_id}", status_code=204)');
      routerLines.push("async def delete_" + table.name + "(item_id: UUID):");
      routerLines.push("    # TODO: implement delete");
      routerLines.push('    raise HTTPException(501, "not implemented")');
      routerLines.push("");
      routerLines.push("");
    }

    files[ns + "/router.py"] = routerLines.join("\n");

    // --- __init__.py ---
    files[ns + "/__init__.py"] = 'from .router import router  # noqa: F401\n';
  }

  // --- main.py ---
  var mainLines = [];
  mainLines.push('"""');
  mainLines.push("FastAPI app (generated by alab).");
  mainLines.push("");
  mainLines.push("Run:");
  mainLines.push("    uv run fastapi dev main.py");
  mainLines.push("");
  mainLines.push("Then open http://localhost:8000/docs");
  mainLines.push('"""');
  mainLines.push("");
  mainLines.push("from fastapi import FastAPI");
  mainLines.push("");

  // Import all namespace routers
  for (var n = 0; n < namespaces.length; n++) {
    mainLines.push("from " + namespaces[n] + " import router as " + namespaces[n] + "_router");
  }

  mainLines.push("");
  mainLines.push('app = FastAPI(title="Alab Generated API")');
  mainLines.push("");

  for (var n = 0; n < namespaces.length; n++) {
    mainLines.push("app.include_router(" + namespaces[n] + "_router)");
  }

  mainLines.push("");
  mainLines.push("");
  mainLines.push('@app.get("/")');
  mainLines.push("async def root():");
  mainLines.push('    return {"message": "Alab Generated API", "docs": "/docs"}');
  mainLines.push("");

  files["main.py"] = mainLines.join("\n");

  return render(files);
});
