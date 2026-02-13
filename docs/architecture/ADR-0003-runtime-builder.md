# ADR-0003: Runtime Builder Pattern with Goja

**Status:** Accepted
**Date:** 2026-02-11
**Commit:** c37d263

## Context

AstrolaDB uses JavaScript (ES5.1) as the schema definition language, executed via the Goja VM. Users define schemas and migrations in `.js` files that get parsed into Go AST structs, then rendered to SQL. The runtime needs to:

- Provide a fluent, ergonomic API for both schemas and migrations
- Execute untrusted user code safely in a sandboxed environment
- Report accurate error locations (file + line number) on failure
- Support two distinct definition styles (object-based schemas, callback-based migrations)

## Decision

### Sandboxed Execution (`sandbox.go`)

Wrap `goja.Runtime` in a `Sandbox` struct that provides:
- **IIFE wrapping**: Schema code is wrapped in an IIFE to prevent global pollution
- **Line offset tracking**: Wrapper adds 2 lines; error line numbers are adjusted by subtracting the offset then adding 1 for file-based source lookup
- **Call stack limit**: Capped at 500 to prevent infinite recursion
- **Timeout**: Default 5-second execution timeout

### Two Builder Styles

**Object-based** (schemas):
```javascript
schema({ users: {
  name: col.string(255),
  age: col.integer()
}})
```

**Method-based** (migrations):
```javascript
migration({ up(m) {
  m.create_table("users", t => {
    t.string("name", 255)
    t.integer("age")
  })
}})
```

### Builder Package Structure

```
internal/runtime/builder/
  table.go     TableBuilder - fluent API for table definition
  column.go    ColBuilder - object-based column creation
  chain.go     Chainable modifiers (.optional(), .unique(), etc.)
  types.go     Type definitions (ColumnDef, IndexDef, ForeignKeyDef)
  semantic.go  Semantic types (email, username, slug, url, phone, flag)
  fn.go        Computed columns (FnExpr builder)
  errors.go    Error messages for CLI display
```

### Flow: JavaScript to SQL

1. `bindings.go` exposes `migration()` or `schema()` to the JS runtime
2. User code calls builder methods: `t.string("name", 255).unique().optional()`
3. `TableBuilder` accumulates `ColumnDef` structs in its `Columns` slice
4. `ToResult()` converts Go structs to JS-compatible objects (maps)
5. `sandbox.go` parses the JS objects back into `ast.TableDef` / `ast.ColumnDef`
6. Dialect layer renders AST to SQL

### Semantic Types (`semantic.go`)

Centralized registry of high-level types that auto-apply constraints:
- `email` - string with format validation
- `username`, `slug` - string with naming constraints
- `url`, `phone` - string with pattern validation
- `flag` - boolean with default value

Available in both builder styles (method-based and object-based).

### Relationships (`table.go`)

- `belongs_to(ref)` - FK with auto-naming (`"auth.user"` -> `user_id`)
- `one_to_one(ref)` - Unique FK
- `many_to_many(ref)` - Junction table metadata
- `belongs_to_any(refs)` - Polymorphic (adds `_type` + `_id` columns)
- Chainable: `.as("author").optional().on_delete("cascade")`

## Consequences

**Positive:**
- Type safety: Go structs validate schemas before SQL generation
- Fluent API: chainable methods improve developer experience
- Sandboxed: Goja isolation prevents JS from affecting the Go runtime
- Accurate errors: line offset tracking points users to exact source locations
- Extensible: new types/relationships added via builder pattern

**Negative:**
- ES5.1 only: Goja doesn't support ES6 modules, arrow functions in all contexts, or modern JS features
- Two-way conversion: JS objects -> Go structs -> AST adds complexity
- Line number math: wrapper offset adjustment requires careful +1 correction (see MEMORY.md for the off-by-one bug fix)

## Alternatives Considered

1. **YAML/TOML schemas** - Rejected; too verbose for complex schemas with relationships and computed columns
2. **Go-native DSL** - Rejected; would require recompilation on schema changes
3. **Full Node.js runtime** - Rejected; too heavy for a CLI tool. Goja embeds cleanly in Go
4. **Single builder style** - Rejected; object-based is more natural for schemas, method-based for migrations
