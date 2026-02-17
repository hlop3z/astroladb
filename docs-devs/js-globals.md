# JavaScript Global Namespace

Every function and object injected into the Goja VM via `vm.Set()`. Grouped by execution context.

---

## Schema Context

Bound in `internal/runtime/sandbox.go`.

| Global       | Kind     | Description                          |
| ------------ | -------- | ------------------------------------ |
| `table`      | function | Schema entry point                   |
| `col`        | object   | Column builder (`col.string()`, etc) |
| `fn`         | object   | Computed column builder              |
| `sql({...})` | function | Per-dialect raw SQL expression       |

**Disabled:** `eval` is set to `undefined`.

---

## Migration Context

Bound in `internal/runtime/bindings.go`.

| Global       | Kind     | Description                    |
| ------------ | -------- | ------------------------------ |
| `migration`  | function | Migration entry point          |
| `sql({...})` | function | Per-dialect raw SQL expression |

Note: `m.sql()` inside migrations is a different function — it executes raw SQL as a migration operation.

---

## Generator Context

Bound in `internal/runtime/sandbox_gen.go`.

| Global               | Kind     | Description                   |
| -------------------- | -------- | ----------------------------- |
| `gen(fn)`            | function | Generator entry point         |
| `render(obj)`        | function | Output file map to disk       |
| `json(val, indent?)` | function | Safe `JSON.stringify` wrapper |
