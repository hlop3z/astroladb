# Integration Tests

This directory is reserved for **cross-package integration tests** that don't fit neatly into a single package.

## Current Organization

Integration tests in AstrolaDB follow **Go conventions** and are organized as follows:

### ✅ Current Structure (Recommended)

```
├── cmd/alab/                      # CLI integration & E2E tests
│   ├── e2e_test.go               # Multi-phase workflows
│   ├── cli_integration_test.go    # CLI command tests
│   ├── cli_e2e_test.go           # End-to-end CLI tests
│   ├── postgres_types_e2e_test.go # PostgreSQL type tests
│   ├── sqlite_types_e2e_test.go   # SQLite type tests
│   └── utc_timezone_e2e_test.go   # Timezone handling tests
│
├── internal/*/                    # Package integration tests (next to source)
│   ├── engine/*_integration_test.go
│   ├── dialect/*_integration_test.go
│   ├── introspect/*_integration_test.go
│   └── pkg/astroladb/*_integration_test.go
│
└── test/integration/              # Cross-package integration (this directory)
    ├── e2e/                       # Reserved for future e2e tests
    └── go-js-boundary/            # Reserved for Go↔JS bridge tests
```

## Running Integration Tests

### All Integration Tests
```bash
go test ./... -tags=integration -v
```

### CLI Integration Tests Only
```bash
go test ./cmd/alab/... -tags=integration -v
```

### Package Integration Tests
```bash
go test ./internal/engine/... -tags=integration -v
go test ./internal/dialect/... -tags=integration -v
```

### Specific Test
```bash
go test ./cmd/alab -tags=integration -run TestE2E_FullBlogPlatform -v
```

## Test Categories

### 🔵 Unit Tests
- **Location**: Next to source (Go convention)
- **Build Tag**: None
- **Scope**: Single function/struct
- **Example**: `internal/engine/diff_test.go`

### 🟢 Integration Tests
- **Location**: Next to source or cross-package
- **Build Tag**: `//go:build integration`
- **Scope**: Multiple packages, database required
- **Example**: `internal/engine/migration_integration_test.go`

### 🟡 E2E Tests
- **Location**: `cmd/alab/*_e2e_test.go`
- **Build Tag**: `//go:build integration`
- **Scope**: Full user workflows, CLI testing
- **Example**: `cmd/alab/e2e_test.go`

## Design Principles

### ✅ DO
- Keep tests next to source code (Go convention)
- Use `//go:build integration` tag for tests requiring database
- Use descriptive test names: `TestE2E_FullBlogPlatform`
- Group related tests with subtests: `t.Run("subtest name", ...)`
- Use fixtures from `test/fixtures/` for reusable data

### ❌ DON'T
- Move unit tests away from source
- Mix unit and integration tests in same file
- Hardcode test data (use fixtures instead)
- Skip cleanup in integration tests

## Future Organization

When this directory grows, organize by concern:

```
test/integration/
├── e2e/
│   ├── workflows/          # Multi-step user workflows
│   ├── cli/                # CLI end-to-end tests
│   └── api/                # API end-to-end tests
├── go-js-boundary/         # Go↔JavaScript bridge tests
├── cross-dialect/          # Cross-database compatibility
└── performance/            # Performance benchmarks
```

## See Also

- [test/fixtures/](../fixtures/) - Reusable test data (JS migrations, schemas)
- [test/javascript/](../javascript/) - TypeScript contract tests
- [TESTING.md](../../TESTING.md) - Comprehensive testing guide
- [PLAN.md](../../PLAN.md) - Testing refactor plan
