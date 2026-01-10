# Contributing to Astroladb

Thank you for your interest in contributing! Before you start, please read this guide carefully. Astroladb is an **opinionated tool by design** - this means we intentionally limit scope and reject certain types of contributions.

> If you're looking for a flexible, configurable migration tool, there are many great options out there. Astroladb is not that tool - and that's intentional.

---

## Project Philosophy

Astroladb follows four core principles:

| Principle         | What it means                                                           |
| ----------------- | ----------------------------------------------------------------------- |
| **Simple**        | One way to do things. No configuration options where a default will do. |
| **Boring**        | No magic. Predictable behavior. Convention over configuration.          |
| **Deterministic** | Same input = same output, always. Reproducible across environments.     |
| **JS-Friendly**   | All types must be safe in JavaScript frontends (no precision loss).     |

---

## What We Accept

**Bug fixes** - Always welcome. If something doesn't work as documented, please fix it.

**Documentation improvements** - Typos, clarifications, better examples.

**Performance improvements** - Faster is better, as long as behavior doesn't change.

**Test coverage** - More tests are always welcome.

**Better error messages** - Help users understand what went wrong.

**IDE/tooling support** - TypeScript definitions, editor integrations.

**Export format improvements** - Better OpenAPI, GraphQL, TypeScript, Go, Python, Rust output.

---

## What We Do NOT Accept

These are non-negotiable. PRs adding these will be closed without merge:

### Database Support

- **No MySQL, MariaDB, SQL Server, Oracle, etc.** - PostgreSQL and SQLite only. Forever.
- No "database abstraction layers" that add complexity for hypothetical databases.

### Type System

- **No `int64` or `bigint`** - JavaScript loses precision above 2^53.
- **No `float64` or `double`** - Precision issues in JS frontends.
- **No auto-increment IDs** - UUID only. No ULID, Snowflake, or alternatives.

### Configuration

- **No "options" for things that should have defaults** - Pick the right default.
- **No feature flags** - Either a feature exists or it doesn't.
- **No plugin systems** - Keep it simple.

### Scope Creep

- **No ORM features** - Astroladb is a migration tool, not an ORM.
- **No query builders** - Use your language's database library.
- **No "runtime" features** - Astroladb runs at dev time, not production.

### Common Requests We'll Reject

| Request                          | Why we reject it                  |
| -------------------------------- | --------------------------------- |
| "Add an option for..."           | Pick a sensible default instead   |
| "Some users might want..."       | We solve 98% of cases, not 100%   |
| "Can we support MySQL..."        | No. PostgreSQL and SQLite only.   |
| "Auto-increment is simpler..."   | UUID is the standard. Period.     |
| "For backwards compatibility..." | We design it right the first time |
| "What if they need..."           | They can use another tool         |

---

## Development Setup

### Requirements

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **Docker** (for integration tests) - [Install Docker](https://docker.com)

### Clone and Build

```bash
git clone https://github.com/hlop3z/astroladb.git
cd astroladb
go build ./cmd/alab
```

### Run Tests

```bash
# Unit tests only (no database needed)
go test ./...

# Integration tests (requires Docker)
docker-compose -f docker-compose.test.yml up -d
go test ./... -tags=integration

# Stop databases when done
docker-compose -f docker-compose.test.yml down
```

### Project Structure

```text
cmd/alab/          # CLI entry point
internal/          # Core implementation (not public API)
  ├── alerr/       # Error handling with codes
  ├── ast/         # Schema AST
  ├── cli/         # CLI utilities, colors, output
  ├── dialect/     # PostgreSQL/SQLite specifics
  ├── dsl/         # JavaScript DSL bindings
  ├── engine/      # Migration engine
  ├── runtime/     # Goja JS runtime
  └── ...
pkg/astroladb/     # Public Go API
```

---

## Coding Standards

### Formatting

```bash
# Format code
go fmt ./...

# Lint (install golangci-lint first)
golangci-lint run
```

### Style

- Use existing internal utilities (see `internal/` packages)
- Errors use `internal/alerr` with error codes
- SQL building uses `internal/sqlgen`
- String utilities in `internal/strutil`

### Testing

- Unit tests alongside code: `foo_test.go`
- Integration tests use build tags: `//go:build integration`
- Use `internal/testutil` for test helpers

---

## Commit Guidelines

### Format

```text
<type>: <short description>

<optional body>
```

### Types

- `fix:` - Bug fix
- `feat:` - New feature
- `docs:` - Documentation only
- `test:` - Adding tests
- `refactor:` - Code change that doesn't fix a bug or add a feature
- `perf:` - Performance improvement

### Examples

```text
fix: Handle NULL values in JSON export

feat: Add col.rating() semantic type

docs: Clarify belongs_to relationship examples

test: Add integration tests for rollback
```

---

## Pull Request Guidelines

### Before Submitting

1. **Read this document** - Especially "What We Do NOT Accept"
2. **Run tests** - `go test ./...`
3. **Format code** - `go fmt ./...`
4. **Keep it small** - One logical change per PR

### PR Process

1. Fork the repository
2. Create a branch from `main`: `git checkout -b fix/issue-description`
3. Make your changes
4. Push and open a PR against `main`
5. Wait for review

### What Makes a Good PR

- **Focused** - Does one thing well
- **Tested** - Includes tests for new functionality
- **Documented** - Updates README/docs if needed
- **Follows conventions** - Matches existing code style

---

## Issue Guidelines

### Bug Reports

Include:

- Astroladb version (`alab --version`)
- OS and architecture
- Minimal reproduction steps
- Expected vs actual behavior
- Schema file (if relevant)

### Feature Requests

Before requesting a feature, ask yourself:

1. Does this align with the four principles (Simple, Boring, Deterministic, JS-Friendly)?
2. Is there already a way to do this?
3. Would this add configuration options? (If yes, probably won't be accepted)

---

## Security

**Do not open public issues for security vulnerabilities.**

Email security issues to the maintainers privately. Include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact

---

## Code of Conduct

Be respectful. Be constructive. Remember that maintainers are volunteers.

We welcome contributors of all backgrounds and experience levels, but we expect:

- Professional and respectful communication
- Constructive feedback
- Patience during code review

---

## License

By contributing, you agree that your contributions will be licensed under the BSD-3-Clause license.

---

## Still Want to Contribute?

Great! Here's how to get started:

1. Look at [open issues](https://github.com/hlop3z/astroladb/issues) labeled `good first issue`
2. Read the codebase - start with `cmd/alab/main.go`
3. Run the tests to make sure everything works
4. Pick something small and submit a PR

Questions? Open a discussion or issue. We're happy to help contributors who respect the project's philosophy.
