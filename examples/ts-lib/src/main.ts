// main.ts — Generator entry point
// This gets bundled by esbuild into generators/fastapi.js

import { buildModels, buildRouter, buildMain } from "./templates";

export default gen((schema) => {
  const namespaces = Object.keys(schema.models);
  const files: Record<string, string> = {};

  for (const ns of namespaces) {
    const tables = schema.models[ns];
    files[ns + "/models.py"] = buildModels(tables);
    files[ns + "/router.py"] = buildRouter(ns, tables);
    files[ns + "/__init__.py"] = 'from .router import router  # noqa: F401\n';
  }

  files["main.py"] = buildMain(namespaces);

  return render(files);
});
