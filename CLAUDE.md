# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

AstrolaDB is a **polyglot code generator** with a Go core that embeds a JavaScript runtime (Goja) to execute user-written schemas and generators.

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
│  Engine Layer (Go)                      │  ← Core processing
│  • internal/engine - Migration engine   │
│  • internal/dialect - SQL generation    │
│  • internal/types - Type exports        │
└─────────────────────────────────────────┘
```

### Key Directories

- `cmd/alab/` - CLI entry point, command implementations
- `internal/runtime/` - **Goja JavaScript VM and DSL bindings**
  - `sandbox.go` - Secure JS execution environment
  - `builder/` - JavaScript DSL implementation (col.\*, table())
  - `jserror.go` - JavaScript error parsing and formatting
- `internal/engine/` - Migration generation and planning
- `internal/dialect/` - SQL generation for PostgreSQL, SQLite
- `internal/alerr/` - Structured error system with error codes
- `internal/cli/` - CLI output formatting (Rust/Cargo-style errors)
- `internal/ast/` - Schema AST representation
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
- All future DSL additions

### JavaScript Runtime & Error Handling

**Schema Execution Flow:**

1. Load `.js` file from disk → `sandbox.currentFile` set
2. Strip "export default " → code modified
3. Optionally wrap in IIFE → `lineOffset = 2` if wrapped, `0` if not
4. Execute with Goja → reports line numbers in executed code
5. On error: adjust line with `jsErr.Line - lineOffset` to get original file line

**Line Number Indexing Convention:**

- Goja: 1-indexed (first line is 1)
- `GetSourceLine()` / `GetSourceLineFromFile()`: 1-indexed (first line is 1)
- After wrapper adjustment, need +1 correction to fetch correct source line from file
- See `internal/runtime/jserror_test.go` test case `goja_error_line_accuracy` for validation

**Error Handling Patterns:**

Use **different error formats** for different APIs:

1. **Column API (col.\*)** - Use `BuilderError` (pipe format):

   ```go
   // internal/runtime/builder/errors.go
   ErrMsgStringRequiresLength = BuilderError{
       Cause: "string() requires a length argument",
       Help:  "try col.string(255) for VARCHAR(255)",
   }

   // internal/runtime/builder/column.go
   panic(cb.vm.ToValue(ErrMsgStringRequiresLength.String()))
   ```

2. **Table API (t.\*)** - Use `alerr.NewXXXError()` functions:

   ```go
   // internal/alerr/suggestions.go
   func NewMissingReferenceError(method string) *Error { ... }

   // internal/runtime/builder/table.go
   panic(tb.vm.ToValue(alerr.NewMissingReferenceError("t.belongs_to").Error()))
   ```

**IMPORTANT - TODO:** All validation errors for schemas and migrations should use Goja's structured error types (`CompilerSyntaxError`, `Exception`) instead of string parsing. Use `syntaxErr.Message`, `syntaxErr.File.Position(syntaxErr.Offset)` to get clean messages and accurate line numbers. See `internal/runtime/jserror.go:37-44` for the correct pattern.

### Error Display Format

Follows Rust/Cargo-style formatting (implemented in `internal/cli/error.go`):

```
error[E2003]: belongs_to() requires a table reference
  --> schemas/auth/role.js:5:18
   |
 5 |   owner: col.belongs_to()
   |          ^^^^^^^^^^^^^^^^
   |
note: relationships need a target table
help: try col.belongs_to('namespace.table') or col.belongs_to('.table') for same namespace
```

**Key Components:**

- File location with line:column (`--> file:line:col`)
- Source code line with caret highlighting (`^^^`)
- Notes and help messages
- Clean cause messages (no redundant line numbers, no `[E####]` codes in cause)

### CLI Output Styling

**From `internal/cli/` patterns:**

- Use `ui.SectionTitle()` for section headers (green+bold, no prefix)
- Use `ui.Success()` only for success messages (adds ✓ prefix)
- Don't add "Usage" section that repeats command name
- Don't add verbose `Long` descriptions - point to docs/ instead
- Add consistent spacing between sections with `fmt.Println()`
- Main help colors: Blue commands, Green titles, Cyan flags, Dim helpers

### Project Philosophy (from CONTRIBUTING.md)

**Four Core Principles:**

1. **Simple** - One way to do things. No config options where a default will do.
2. **Boring** - No magic. Predictable behavior. Convention over configuration.
   - **ALL validation should happen during JavaScript execution**, not post-parse
3. **Deterministic** - Same input = same output, always.
4. **JS-Friendly** - All types safe in JavaScript (no int64, float64, etc.)

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
3. **Error format consistency** - BuilderError for col._, alerr functions for t._
4. **Cause message cleaning** - Strip error codes, Goja stack traces, and redundant line numbers from cause display
5. **Case-insensitive regex** - Goja uses "Line X:Y" (capital L) in error messages - use `(?i)` flag

## Documentation

- **User docs:** `docs/` (Starlight site at hlop3z.github.io/astroladb)
- **Developer docs:** `docs-devs/` and `TESTING.md`
- **Examples:** `examples/generators/` (FastAPI, Chi, Axum, tRPC generators)
