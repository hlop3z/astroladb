# Generator Schema Structure

This document explains the two different views of schema data available to generators.

## Overview

When you write a generator using `gen()`, the schema object passed to your generator function provides **two views** of your tables:

1. **`schema.tables`** - A flat array of all tables
2. **`schema.models`** - Tables grouped by namespace

## 1. `schema.tables` - Flat Array

This provides a flat list of all tables across all namespaces.

```json
{
  "tables": [
    { "namespace": "auth", "name": "auth_user", "table": "auth_user", ... },
    { "namespace": "auth", "name": "auth_role", "table": "auth_role", ... },
    { "namespace": "blog", "name": "blog_post", "table": "blog_post", ... }
  ]
}
```

### Example Usage

```javascript
export default gen((schema) => {
  // Iterate over all tables
  const models = schema.tables
    .map((table) => {
      return `export interface ${table.name} { ... }`;
    })
    .join("\n\n");

  return render({
    "models.ts": models,
  });
});
```

### When to Use

- Single-file output (e.g., `models.ts` with all interfaces)
- Simple generators that don't need namespace organization
- When you want to process all tables together

## 2. `schema.models` - Grouped by Namespace

This organizes tables into an object keyed by namespace.

```json
{
  "models": {
    "auth": [
      { "name": "auth_user", "table": "auth_user", ... },
      { "name": "auth_role", "table": "auth_role", ... }
    ],
    "blog": [
      { "name": "blog_post", "table": "blog_post", ... }
    ]
  }
}
```

### Example Usage

```javascript
export default gen((schema) => {
  const files = {};

  // Create separate files per namespace
  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`${ns}/models.py`] = buildModels(tables);
    files[`${ns}/router.py`] = buildRouter(ns, tables);
    files[`${ns}/__init__.py`] = "from .router import router\n";
  }

  return render(files);
});
```

This generates:

```
auth/
  ├── models.py
  ├── router.py
  └── __init__.py
blog/
  ├── models.py
  ├── router.py
  └── __init__.py
```

### When to Use

- Multi-file output organized by domain
- Package/module structures (Python, Go, etc.)
- When namespaces map to directories or modules
- When you want to keep related models together

## Real-World Examples

### Flat Structure - TypeScript Generator

```javascript
// examples/generators/generator.js
export default gen((schema) =>
  render({
    "models.ts": buildModels(schema.tables), // Single file
    "router.ts": buildRouter(schema.tables), // Single file
  }),
);
```

Output:

```
models.ts   # All interfaces
router.ts   # All routes
```

### Namespace Structure - FastAPI Generator

```javascript
// examples/generators/generators/fastapi.js
export default gen((schema) => {
  const files = {};

  for (const [ns, tables] of Object.entries(schema.models)) {
    files[`${ns}/models.py`] = modelFile(tables);
    files[`${ns}/router.py`] = routerFile(ns, tables);
    files[`${ns}/__init__.py`] = "from .router import router\n";
  }

  files["main.py"] = mainFile(Object.keys(schema.models));

  return render(files);
});
```

Output:

```
auth/
  ├── models.py     # Auth models only
  ├── router.py     # Auth routes only
  └── __init__.py
blog/
  ├── models.py     # Blog models only
  ├── router.py     # Blog routes only
  └── __init__.py
main.py            # Imports all namespaces
```

## Summary

| Aspect     | `schema.tables`             | `schema.models`                 |
| ---------- | --------------------------- | ------------------------------- |
| Structure  | Flat array                  | Grouped by namespace            |
| Use Case   | Simple, single-file outputs | Multi-file, organized outputs   |
| Iteration  | `schema.tables.map(...)`    | `Object.entries(schema.models)` |
| Example    | TypeScript interfaces       | Python packages                 |
| Complexity | Lower                       | Higher                          |

Choose the structure that best fits your target language and project organization needs.
