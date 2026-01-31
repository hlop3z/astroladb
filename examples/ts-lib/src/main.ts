// main.ts — Generator entry point
// This gets bundled by esbuild into generators/fastapi.js

import { buildModels, buildRouter, buildMain } from "./templates";

export default gen((schema) => {
  const api = schema.openapi;
  const endpoint = api.paths["/schemas"].get.responses["200"]
    .content["application/json"].example;
  const namespaces = Object.keys(endpoint.models);
  const files: Record<string, string> = {};

  for (const ns of namespaces) {
    const tables = endpoint.models[ns];
    files[ns + "/models.py"] = buildModels(tables);
    files[ns + "/router.py"] = buildRouter(ns, tables);
    files[ns + "/__init__.py"] = 'from .router import router  # noqa: F401\n';
  }

  files["main.py"] = buildMain(namespaces);

  return render(files);
});
