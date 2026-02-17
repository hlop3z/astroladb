# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## IMPORTANT

Cargo-style formatting is required for all DSL errors.

## Build & Test Commands

```bash
# Build
go build ./cmd/alab

# Run all tests
go test ./...

# Run tests with race detector (requires gcc)
CGO_ENABLED=1 go test -race ./...

# Run integration tests (requires Docker)
task db-up                    # Start test databases
go test ./... -tags=integration -count=1
task db-down                  # Stop test databases

# Run specific test
go test -run TestName ./internal/package -v

# Lint
golangci-lint run ./...

# Format
go fmt ./...
```

**Using Task (Taskfile.yml):**

```bash
task build          # Build to ./bin
task test           # Run unit tests
task test-race      # Run with race detector
task test-all       # Run all tests including e2e (manages Docker)
task lint           # Run golangci-lint
task check          # Run lint + vuln + test
```

## Architecture Overview

AstrolaDB is a **documentation-oriented meta-model system** that captures higher-order system metadata through a declarative JavaScript DSL. It's fundamentally about modeling system intent across multiple dimensions — structural, behavioral, and operational — making this metadata consumable by generators and documentation tooling.

### What is Higher-Order System Metadata?

AstrolaDB schemas are **not just database schemas**. They encode:

- **Structural metadata** — tables, columns, types, relationships (traditional schema)
- **Lifecycle metadata** — entity states, workflow transitions, state machines
- **Access metadata** — role-based or attribute-based policy intents
- **Event metadata** — event contracts, payloads, publish targets
- **Service metadata** — bounded contexts, ownership, domain partitions
- **Deployment metadata** — regions, replicas, storage, topology hints

**Migrations are just one built-in core** that consumes structural metadata to generate SQL. Generators consume the full metadata graph to produce application code, documentation, infrastructure templates, and more.

### Three-Layer Architecture

```
┌─────────────────────────────────────────┐
│  User Layer (JavaScript DSL)            │  ← Users write schemas/generators
│  • table(), col.*, fn.*                 │
│  • gen(), render()                      │
└──────────────┬──────────────────────────┘
               │ parsed by
               ▼
┌─────────────────────────────────────────┐
│  Runtime Layer (Goja VM)                │  ← Sandboxed JS execution
│  • internal/runtime/sandbox.go          │
│  • internal/runtime/builder/*.go        │
└──────────────┬──────────────────────────┘
               │ produces
               ▼
┌─────────────────────────────────────────┐
│  Core Layer (Go)                        │  ← Built-in metadata consumers
│  • internal/engine - Migration core     │  (structural → SQL)
│  • internal/dialect - SQL generation    │
│  • internal/types - Type exports        │
│  • [Future: Generator registry]         │  (full metadata → code/docs/IaC)
└─────────────────────────────────────────┘
```

### Key Directories

- `cmd/alab/` - CLI entry point, command implementations
- `internal/runtime/` - **Goja JavaScript VM and DSL bindings**
  - `sandbox.go` - Secure JS execution environment
  - `builder/` - JavaScript DSL implementation (col.\*, table(), metadata builders)
  - `jserror.go` - JavaScript error parsing and formatting
- `internal/engine/` - **Migration core** (one of multiple built-in cores)
  - Consumes structural metadata to generate SQL migrations
- `internal/dialect/` - SQL generation for PostgreSQL, SQLite
- `internal/alerr/` - Structured error system with error codes
- `internal/cli/` - CLI output formatting (Rust/Cargo-style errors)
- `internal/ast/` - Schema AST representation (structural + higher-order metadata)
- `pkg/astroladb/` - Public Go API

## Critical Implementation Details

### JavaScript DSL Conventions

#### Naming Convention: snake_case

ALL DSL methods and properties use snake_case, NOT camelCase.

Examples:

- ✅ `col.primary_key()`, `col.read_only()`, `table.sort_by()`
- ❌ `col.primaryKey()`, `col.readOnly()`, `table.sortBy()`

This applies to:

- Column methods: `.belongs_to()`, `.created_at`, `.updated_at`
- Table methods: `.sort_by()`, `.searchable()`, `.filterable()`
- Higher-order metadata methods: `.lifecycle()`, `.policy()`, `.events()`, `.meta()`, `.deploy()`
- All future DSL additions

### JavaScript Runtime & Error Handling

**Unified Error Pipeline — All DSL Features:**

All JavaScript DSL features (table definitions with higher-order metadata, manual migrations, custom generators) execute through the **same Sandbox** and must follow the **same error pipeline**:

1. JS code executes in Goja VM → validation errors `panic(vm.ToValue(string))`
2. Goja captures the JS call site (line:col) in the Exception stack
3. `ParseJSError()` extracts line, column, and message from the Exception
4. `wrapJSError()` adjusts line offset, reads source from file, builds `*alerr.Error`
5. `cli.FormatError()` renders the Cargo-style output

**This pipeline is the same regardless of file type.** Never create a separate error path for migrations or generators — always go through `wrapJSError` so every error gets file location, source context, and consistent formatting.

**Execution Flow (all file types):**

1. Load `.js` file from disk → `sandbox.currentFile` set
2. Strip "export default " → code modified
3. Optionally wrap in IIFE → `lineOffset = 2` if wrapped, `0` if not
4. Execute with Goja → reports line numbers in executed code
5. On error: adjust line with `jsErr.Line - lineOffset + 1` to get original file line

**Line Number Indexing Convention:**

- Goja: 1-indexed (first line is 1)
- `GetSourceLine()` / `GetSourceLineFromFile()`: 1-indexed (first line is 1)
- IIFE wrapper offset: `adjustedLine = jsErr.Line - lineOffset + 1`
- See `internal/runtime/jserror_test.go` and `error_pipeline_test.go` for validation

**Error Throwing Pattern (simple panic — same for all DSL builders):**

Validation errors panic with a simple string value. Goja wraps this in an Exception and captures the JS call site automatically. **Never** call a JS function from Go to throw — Goja will capture the wrong call site.

```go
// Structured error: "[XXX-NNN] cause|help" — used for all validation errors
// Use the panicStructured helper in dsl_helpers.go:
panicStructured(vm, alerr.ErrMissingReference, "belongs_to() requires a reference", "try `col.belongs_to('ns.table')`")

// BuilderError helper (col.* API) — same format via .String()
panic(cb.vm.ToValue(ErrMsgStringRequiresLength.String()))

// throwStructuredError helper (table.* API)
throwStructuredError(vm, string(alerr.ErrMissingReference), "belongs_to() requires a reference", "try `col.belongs_to('ns.table')`")

// Passthrough helper for JS callback errors (replaces 4-line pattern):
panicPassthrough(vm, err)
```

`extractStructuredError()` in `sandbox.go` parses the `[XXX-NNN] cause|help` format back into code, message, and help text.

### Error Display Format

**All errors** — from schemas, migrations, and generators — follow the same Rust/Cargo-style formatting (implemented in `internal/cli/error.go`):

```shell
error[VAL-009]: belongs_to() requires a table reference
  --> schemas/auth/role.js:5:18
   |
 5 |   owner: col.belongs_to()
   |          ^^^^^^^^^^^^^^^^
   |
help: try `col.belongs_to('namespace.table')` or `col.belongs_to('.table')` for same namespace
```

**Error Code Format: `XXX-NNN`** — 3-letter category tag, dash, 3-digit number:

- `SCH-xxx` — Schema errors
- `VAL-xxx` — Validation errors
- `MIG-xxx` — Migration errors
- `SQL-xxx` — SQL/database errors
- `GEN-xxx` — Generator/runtime errors
- `DBX-xxx` — Database examine/introspection errors
- `GIT-xxx` — Git operation errors
- `CHE-xxx` — Cache errors
- `SYS-xxx` — System/internal errors

**Key Components (must be present for ALL error types):**

- Error code with message (`error[XXX-NNN]: ...`)
- File location with line:column (`--> file:line:col`)
- Source code line with caret highlighting (`^^^`)
- Contextual notes and actionable help messages
- Clean cause messages (no redundant line numbers, no `[XXX-NNN]` codes in cause)

**Consistency rule:** If a new DSL feature (schema, migration, or generator) can produce an error, it MUST go through the same pipeline and produce this same format. Test with `CRITICAL` prefix tests in `error_pipeline_test.go`.

**Error preview rule:** Every new error MUST be added to `TestPreviewAllErrors` in `internal/runtime/preview_errors_test.go`. This visual test renders every structured error with `cli.FormatError()` so developers can inspect formatting. Run with `go test -run TestPreviewAllErrors ./internal/runtime/ -v -count=1` or `task preview-errors`.

### Help Text Style Guide (Cargo conventions)

All `help:` text follows Rust/Cargo conventions for consistency:

1. **Start with `try`** for code suggestions — never bare examples, "Use", or "Do"
2. **Wrap code in backticks** — `` try `col.string(255)` `` not `try col.string(255)`
3. **Lowercase** — help text never starts with a capital letter
4. **Actionable fixes only** — explanations go in `note:`, fixes go in `help:`
5. **Keep short** — minimal viable example, no full function signatures

```go
// GOOD:
"try `col.string(255)` for VARCHAR(255)"
"try `migration({ up(m) { ... } })`"
"use lowercase letters, digits, and underscores"

// BAD:
"col.string(255)"                              // bare example, no verb
"Use col.string(255) for VARCHAR(255)"         // wrong verb, no backticks
"Try col.string('name', 255) for VARCHAR(255)" // capitalized
"table names must be valid identifiers (letters, digits, underscores)"  // explanation, not fix
"try add_column('ns.table', function(col) { col.string('name', 255) })" // too long
```

**Where things go:**

- `note:` — explains what went wrong (context, not fix)
- `help:` — tells how to fix it (actionable code suggestion)
- `cause:` — the underlying error (auto-extracted, not manually set)

### CLI Output Styling

**From `internal/cli/` patterns:**

- Use `ui.SectionTitle()` for section headers (green+bold, no prefix)
- Use `ui.Success()` only for success messages (adds ✓ prefix)
- Don't add "Usage" section that repeats command name
- Don't add verbose `Long` descriptions - point to docs/ instead
- Add consistent spacing between sections with `fmt.Println()`
- Main help colors: Blue commands, Green titles, Cyan flags, Dim helpers

### Project Philosophy (from CONTRIBUTING.md)

**Five Core Principles:**

1. **Simple** - One way to do things. No config options where a default will do.
2. **Boring** - No magic. Predictable behavior. Convention over configuration.
   - **ALL validation should happen during JavaScript execution**, not post-parse
3. **Deterministic** - Same input = same output, always.
4. **JS-Friendly** - All types safe in JavaScript (no int64, float64, etc.)
5. **Documentation-Oriented** - Schemas are living documentation capturing system intent across structural, behavioral, and operational dimensions. All higher-order metadata is **declarative and generator-consumable only** — no runtime enforcement.

**Hard Constraints:**

- PostgreSQL and SQLite only (custom dialects allowed but contributor-maintained)
- UUID-only IDs (no auto-increment, ULID, Snowflake)
- No int64/bigint (JS loses precision above 2^53)
- No float64/double (precision issues in JS)

## Testing Strategy

Follow the testing pyramid from TESTING.md:

- **80%** - Go unit tests (fast, isolated)
- **15%** - Contract tests (Go↔JS boundary)
- **5%** - E2E/integration tests (complete workflows)

**Test file patterns:**

- `*_test.go` - Unit tests alongside code
- `*_integration_test.go` - Integration tests
- `*_e2e_test.go` - End-to-end tests (require `//go:build e2e`)

**Critical tests:**

- Line number accuracy tests prevent user-visible bugs in error messages
- Add `CRITICAL` prefix to test names that validate user-facing behavior

## Common Gotchas

1. **Line number off-by-one errors** - Always use 1-indexed conventions, apply wrapper offset, then +1 for file reading
2. **Validation timing** - Validate during JS execution (in builder/\*.go), not post-parse (in merge.go or similar)
3. **Error format consistency** - All DSL errors (schemas, migrations, generators) must go through the same `wrapJSError` → `alerr.Error` → `cli.FormatError` pipeline. Never create a separate error path.
4. **Error throwing** - Always `panic(vm.ToValue(string))` from Go callbacks. Never call a JS function from Go to throw — Goja captures the wrong call site.
5. **Cause message cleaning** - Strip error codes, Goja stack traces, and redundant line numbers from cause display
6. **Help text style** - Always start with `try`, wrap code in backticks, keep lowercase. See "Help Text Style Guide" above.
7. **`sql()` is always per-dialect** - Never split SQL helpers into separate dialect functions. Use `sql({ postgres: "...", sqlite: "..." })` everywhere. This matches `fn.sql()` and follows the "one way to do things" principle.

## Documentation

- **User docs:** `docs/` (Starlight site at hlop3z.github.io/astroladb)
- **Developer docs:** `docs-devs/` and `TESTING.md`
- **Examples:** `examples/generators/` (FastAPI, Chi, Axum, tRPC generators)
