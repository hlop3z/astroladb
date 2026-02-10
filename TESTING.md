# Testing Guide for AstrolaDB

> **Architecture Context**: AstrolaDB is a polyglot system with a Go engine that embeds a JavaScript runtime (Goja) to execute user-written schemas and generators. This guide covers testing strategies for this embedded runtime architecture.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Testing Philosophy](#testing-philosophy)
- [Testing Layers](#testing-layers)
- [Setting Up Your Test Environment](#setting-up-your-test-environment)
- [Running Tests](#running-tests)
- [Testing Patterns](#testing-patterns)
- [Tools & Dependencies](#tools--dependencies)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)
- [Resources & References](#resources--references)

---

## Architecture Overview

Understanding the architecture is crucial for effective testing:

```
┌─────────────────────────────────────────────────┐
│           User Layer (JavaScript/TypeScript)     │
│  • Schema definitions (table(), col.*)          │
│  • Custom generators (gen(), render())          │
└─────────────────────┬───────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│         Embedded Runtime Layer (Goja)           │
│  • JavaScript VM running inside Go              │
│  • Executes user code in sandboxed environment  │
└─────────────────────┬───────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│            Core Engine (Go)                     │
│  • Schema processing & validation               │
│  • Migration generation                         │
│  • SQL generation (PostgreSQL, SQLite)          │
│  • Type exports (Rust, Go, Python, TypeScript)  │
└─────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│              Database Layer                     │
│  • PostgreSQL, SQLite, MySQL                    │
└─────────────────────────────────────────────────┘
```

**Key Testing Challenges:**

1. Testing across language boundaries (Go ↔ JavaScript)
2. Testing embedded runtime behavior (Goja)
3. Testing user-extensible generators (JavaScript)
4. Testing database interactions (SQL)
5. Ensuring type safety across languages

---

## Testing Philosophy

We follow a **layered testing strategy** based on the testing pyramid:

```
         ┌─────────────────┐
         │   E2E Tests     │  ~5%  (Slow, brittle, high value)
         │ (Integration)   │       1-2 per major feature
         ├─────────────────┤
         │  Contract Tests │  ~15% (Medium speed, API stability)
         │  (Go runs JS)   │       Test Go<->JS boundary
         ├─────────────────┤
         │   Go Unit Tests │  ~80% (Fast, isolated, comprehensive)
         │                 │       Core business logic
         └─────────────────┘
```

**Principles:**

- **Fast feedback loops**: Prioritize fast unit tests
- **Isolated testing**: Test components independently when possible
- **Integration confidence**: Use integration tests for critical paths
- **Contract stability**: Ensure user-facing APIs don't break
- **Real-world validation**: E2E tests verify complete workflows

---

## Testing Layers

### Layer 1: Go Unit Tests (80%)

**Purpose**: Test Go components in isolation without JavaScript involvement.

**When to use:**

- Testing SQL generation logic
- Testing schema validation
- Testing utility functions
- Testing error handling

**Example:**

```go
// internal/sqlgen/builder_test.go
package sqlgen

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreateTableSQL_PostgreSQL(t *testing.T) {
    builder := NewBuilder("postgres")
    columns := []Column{
        {Name: "id", Type: "uuid", PrimaryKey: true},
        {Name: "email", Type: "string", Unique: true},
    }

    sql := builder.CreateTable("users", columns)

    assert.Contains(t, sql, "CREATE TABLE")
    assert.Contains(t, sql, "users")
    assert.Contains(t, sql, "PRIMARY KEY")
    assert.Contains(t, sql, "UNIQUE")
}

func TestCreateTableSQL_SQLite(t *testing.T) {
    builder := NewBuilder("sqlite")
    columns := []Column{
        {Name: "id", Type: "integer", PrimaryKey: true},
    }

    sql := builder.CreateTable("users", columns)

    assert.Contains(t, sql, "INTEGER PRIMARY KEY")
}
```

**Running:**

```bash
go test ./internal/sqlgen/... -v
go test ./internal/sqlgen/... -cover
```

---

### Layer 2: Go-Driven JavaScript Execution Tests (15%)

**Purpose**: Test the integration between Go and the embedded Goja runtime.

**When to use:**

- Testing schema evaluation
- Testing generator execution
- Testing JavaScript error handling
- Testing Go<->JS data marshaling
- Testing runtime security/sandboxing

**Example:**

```go
// internal/runtime/sandbox_test.go
package runtime

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEvalSchema_BasicTable(t *testing.T) {
    sb := NewSandbox(nil)

    jsCode := `
    table({
      id: col.id(),
      email: col.email().unique(),
      username: col.username().unique(),
    }).timestamps()
    `

    tableDef, err := sb.EvalSchema(jsCode, "auth", "user")
    require.NoError(t, err)

    assert.Equal(t, "user", tableDef.Name)
    assert.Equal(t, "auth", tableDef.Namespace)

    // Should have 5 columns: id, email, username, created_at, updated_at
    assert.Len(t, tableDef.Columns, 5)

    // Verify column details
    idCol := findColumn(tableDef.Columns, "id")
    require.NotNil(t, idCol)
    assert.True(t, idCol.PrimaryKey)

    emailCol := findColumn(tableDef.Columns, "email")
    require.NotNil(t, emailCol)
    assert.True(t, emailCol.Unique)
}

func TestEvalSchema_SyntaxError(t *testing.T) {
    sb := NewSandbox(nil)

    jsCode := `table({ id: col.id(` // Missing closing parentheses

    _, err := sb.EvalSchema(jsCode, "auth", "user")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "syntax")
}

func TestEvalSchema_RuntimeError(t *testing.T) {
    sb := NewSandbox(nil)

    jsCode := `table({ id: col.nonexistent() })` // Invalid column type

    _, err := sb.EvalSchema(jsCode, "auth", "user")
    assert.Error(t, err)
}

func TestGeneratorSandbox_PathTraversal(t *testing.T) {
    sb := NewSandbox(nil)

    tests := []struct {
        name      string
        code      string
        shouldErr bool
        errMsg    string
    }{
        {
            name:      "path traversal blocked",
            code:      `render({ "../etc/passwd": "hack" })`,
            shouldErr: true,
            errMsg:    "relative paths not allowed",
        },
        {
            name:      "absolute path blocked",
            code:      `render({ "/etc/passwd": "hack" })`,
            shouldErr: true,
            errMsg:    "absolute paths not allowed",
        },
        {
            name:      "dotfile blocked",
            code:      `render({ ".env": "SECRET=hack" })`,
            shouldErr: true,
            errMsg:    "dotfiles not allowed",
        },
        {
            name:      "valid relative path allowed",
            code:      `render({ "auth/models.py": "class User: pass" })`,
            shouldErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := sb.ExecuteGenerator(tt.code, testSchema)

            if tt.shouldErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// Helper function
func findColumn(columns []Column, name string) *Column {
    for _, col := range columns {
        if col.Name == name {
            return &col
        }
    }
    return nil
}
```

**Running:**

```bash
go test ./internal/runtime/... -v
```

---

### Layer 3: Contract Tests (API Stability)

**Purpose**: Ensure the API contract between Go and JavaScript remains stable.

**When to use:**

- Testing schema object structure matches TypeScript definitions
- Testing that generators receive correct data
- Preventing breaking changes to user-facing APIs

**Example:**

```go
// internal/runtime/generator_contract_test.go
package runtime

import (
    "encoding/json"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGeneratorSchemaContract(t *testing.T) {
    // Create a schema object from Go
    schema := &GeneratorSchema{
        Models: map[string][]SchemaTable{
            "auth": {
                {
                    Name:       "user",
                    Table:      "auth_users",
                    PrimaryKey: "id",
                    Timestamps: true,
                    Columns: []SchemaColumn{
                        {Name: "id", Type: "uuid", Nullable: false},
                        {Name: "email", Type: "string", Unique: true},
                        {Name: "username", Type: "string", Unique: true},
                    },
                },
            },
        },
        Tables: []SchemaTable{/* ... */},
    }

    // Marshal to JSON
    jsonBytes, err := json.Marshal(schema)
    require.NoError(t, err)

    // Execute TypeScript validator
    sb := NewSandbox(nil)
    jsCode := `
    const schema = ` + string(jsonBytes) + `;

    // Validate contract matches TypeScript interface
    if (!schema.models) {
        throw new Error("Missing 'models' property");
    }
    if (!schema.tables) {
        throw new Error("Missing 'tables' property");
    }
    if (!Array.isArray(schema.tables)) {
        throw new Error("'tables' must be an array");
    }
    if (typeof schema.models !== 'object') {
        throw new Error("'models' must be an object");
    }

    // Validate table structure
    const firstTable = schema.tables[0];
    if (!firstTable.name || !firstTable.table || !firstTable.primary_key) {
        throw new Error("Table missing required properties");
    }
    if (!Array.isArray(firstTable.columns)) {
        throw new Error("Table columns must be an array");
    }

    "contract_valid"
    `

    result, err := sb.ExecuteScript(jsCode)
    require.NoError(t, err)
    assert.Equal(t, "contract_valid", result)
}

func TestRenderOutputContract(t *testing.T) {
    sb := NewSandbox(nil)

    jsCode := `
    export default gen((schema) => {
        return render({
            "models.py": "# Generated",
            "routers.py": "# Generated",
        });
    });
    `

    files, err := sb.ExecuteGenerator(jsCode, testSchema)
    require.NoError(t, err)

    // Verify render output structure
    assert.IsType(t, map[string]string{}, files)
    assert.Contains(t, files, "models.py")
    assert.Contains(t, files, "routers.py")
}
```

**Running:**

```bash
go test ./internal/runtime/... -run Contract -v
```

---

### Layer 4: Integration Tests (5%)

**Purpose**: Test complete workflows end-to-end.

**When to use:**

- Testing full migration lifecycle
- Testing generator execution with real schemas
- Testing cross-database compatibility
- Testing CLI commands

**Example:**

```go
// cmd/alab/migration_integration_test.go
package main

import (
    "database/sql"
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMigrationLifecycle_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup temp environment
    tmpDir := t.TempDir()
    schemasDir := filepath.Join(tmpDir, "schemas")
    migrationsDir := filepath.Join(tmpDir, "migrations")

    require.NoError(t, os.MkdirAll(schemasDir, 0755))
    require.NoError(t, os.MkdirAll(migrationsDir, 0755))

    // Create config
    configPath := filepath.Join(tmpDir, "alab.yaml")
    configContent := `
database:
  dialect: sqlite
  url: ./test.db

schemas: ./schemas
migrations: ./migrations
`
    require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

    // Change to temp directory
    oldWd, _ := os.Getwd()
    defer os.Chdir(oldWd)
    require.NoError(t, os.Chdir(tmpDir))

    // Step 1: Create schema files
    userSchema := `
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
}).timestamps();
`
    schemaPath := filepath.Join(schemasDir, "auth", "user.js")
    require.NoError(t, os.MkdirAll(filepath.Dir(schemaPath), 0755))
    require.NoError(t, os.WriteFile(schemaPath, []byte(userSchema), 0644))

    // Step 2: Generate migration
    client, err := newClient()
    require.NoError(t, err)
    defer client.Close()

    migrationPath, err := client.MigrationGenerate("create_users")
    require.NoError(t, err)
    assert.FileExists(t, migrationPath)

    // Step 3: Apply migration
    require.NoError(t, client.MigrationRun())

    // Step 4: Verify database
    db, err := sql.Open("sqlite", "test.db")
    require.NoError(t, err)
    defer db.Close()

    assertTableExists(t, db, "auth_users")
    assertColumnExists(t, db, "auth_users", "id")
    assertColumnExists(t, db, "auth_users", "email")
    assertColumnExists(t, db, "auth_users", "username")
    assertColumnExists(t, db, "auth_users", "created_at")
    assertColumnExists(t, db, "auth_users", "updated_at")

    // Step 5: Rollback
    require.NoError(t, client.MigrationRollback(1))

    assertTableNotExists(t, db, "auth_users")
}

func TestGenerator_E2E_FastAPI(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    tmpDir := t.TempDir()

    // Setup schema
    // ... (similar setup)

    client, err := newClient()
    require.NoError(t, err)
    defer client.Close()

    // Run FastAPI generator
    outputDir := filepath.Join(tmpDir, "output")
    err = client.GeneratorRun("examples/generators/generators/fastapi.js", outputDir)
    require.NoError(t, err)

    // Verify generated files
    assert.FileExists(t, filepath.Join(outputDir, "auth", "models.py"))
    assert.FileExists(t, filepath.Join(outputDir, "auth", "router.py"))
    assert.FileExists(t, filepath.Join(outputDir, "main.py"))

    // Verify file contents
    modelsContent, err := os.ReadFile(filepath.Join(outputDir, "auth", "models.py"))
    require.NoError(t, err)
    assert.Contains(t, string(modelsContent), "class User")
    assert.Contains(t, string(modelsContent), "BaseModel")
}

// Helper functions
func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
    t.Helper()
    var name string
    err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
    require.NoError(t, err, "Table %s should exist", tableName)
}

func assertTableNotExists(t *testing.T, db *sql.DB, tableName string) {
    t.Helper()
    var name string
    err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
    assert.Error(t, err, "Table %s should not exist", tableName)
}

func assertColumnExists(t *testing.T, db *sql.DB, tableName, columnName string) {
    t.Helper()
    rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
    require.NoError(t, err)
    defer rows.Close()

    found := false
    for rows.Next() {
        var cid int
        var name, typ string
        var notNull, pk int
        var dflt interface{}
        require.NoError(t, rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk))
        if name == columnName {
            found = true
            break
        }
    }
    assert.True(t, found, "Column %s should exist in table %s", columnName, tableName)
}
```

**Running:**

```bash
# Run all integration tests
go test ./... -tags=integration -v

# Run specific integration test
go test ./cmd/alab/... -run TestMigrationLifecycle -v

# Skip integration tests (fast mode)
go test ./... -short
```

---

### Layer 5: Snapshot/Golden Tests

**Purpose**: Test that generators produce expected output.

**When to use:**

- Testing generator outputs
- Regression testing
- Verifying code generation correctness

**Setup:**

```bash
go get github.com/sebdah/goldie/v2
```

**Example:**

```go
// internal/generators/snapshot_test.go
package generators

import (
    "testing"
    "github.com/sebdah/goldie/v2"
)

func TestFastAPIGenerator_Snapshot(t *testing.T) {
    schema := loadTestSchema("testdata/schemas/auth.js")

    generator := loadGenerator("examples/generators/generators/fastapi.js")
    files, err := executeGenerator(generator, schema)
    require.NoError(t, err)

    // Compare with golden files
    g := goldie.New(t)

    for filename, content := range files {
        g.Assert(t, filename, []byte(content))
    }
}

// Update snapshots with:
// go test ./... -update
```

**Directory structure:**

```
testdata/
├── schemas/
│   ├── auth.js
│   └── blog.js
└── golden/
    ├── fastapi-auth-models.py.golden
    ├── fastapi-auth-router.py.golden
    └── go-chi-handlers.go.golden
```

**Running:**

```bash
# Test against snapshots
go test ./internal/generators/... -v

# Update snapshots
go test ./internal/generators/... -update
```

---

## Setting Up Your Test Environment

### 1. Install Go Testing Dependencies

```bash
# Testing framework
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require

# Snapshot testing
go get github.com/sebdah/goldie/v2

# Database testing with Docker
go get github.com/ory/dockertest/v3

# Mock generation (optional)
go install github.com/vektra/mockery/v2@latest
```

### 2. Install Database Dependencies (for integration tests)

**Using Docker (Recommended):**

```bash
# PostgreSQL
docker run --name alab-test-postgres -e POSTGRES_PASSWORD=test -p 5432:5432 -d postgres:15

# MySQL
docker run --name alab-test-mysql -e MYSQL_ROOT_PASSWORD=test -p 3306:3306 -d mysql:8
```

**Or install locally:**

- PostgreSQL 15+
- MySQL 8+
- SQLite (included in Go)

### 3. Configure Test Database

Create a `.env.test` file:

```bash
DATABASE_URL_POSTGRES=postgresql://postgres:test@localhost:5432/alab_test
DATABASE_URL_MYSQL=root:test@tcp(localhost:3306)/alab_test
DATABASE_URL_SQLITE=file::memory:?cache=shared
```

### 4. Setup Test Fixtures

```bash
# Create test directories
mkdir -p testdata/{schemas,migrations,generators,golden}

# Create example schema for testing
cat > testdata/schemas/auth.js <<'EOF'
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
}).timestamps();
EOF
```

---

## Running Tests

### Quick Reference

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run only unit tests (skip integration)
go test ./... -short

# Run only integration tests
go test ./... -tags=integration

# Run specific test
go test ./internal/runtime/... -run TestEvalSchema

# Run tests in parallel
go test ./... -parallel 4

# Run with race detection
go test ./... -race

# Update golden snapshots
go test ./... -update

# Clean test cache
go clean -testcache
```

### Test Organization

```bash
# Test by package
go test ./internal/runtime/...
go test ./internal/sqlgen/...
go test ./cmd/alab/...

# Test by pattern
go test ./... -run Unit
go test ./... -run Integration
go test ./... -run Contract
go test ./... -run E2E
```

### Debugging Tests

```bash
# Run with debugging output
go test -v ./internal/runtime/... -run TestEvalSchema

# Print test logs even for passing tests
go test -v ./... -test.v

# Run single test with full stack traces
go test -v ./internal/runtime/... -run TestEvalSchema -failfast

# Use Delve debugger
dlv test ./internal/runtime/... -- -test.run TestEvalSchema
```

---

## Testing Patterns

### Pattern 1: Table-Driven Tests

**Use for:** Testing multiple scenarios with similar logic.

```go
func TestColumnTypeMapping(t *testing.T) {
    tests := []struct {
        name     string
        colType  string
        dialect  string
        expected string
    }{
        {"uuid-postgres", "uuid", "postgres", "UUID"},
        {"uuid-sqlite", "uuid", "sqlite", "TEXT"},
        {"integer-postgres", "integer", "postgres", "INTEGER"},
        {"integer-mysql", "integer", "mysql", "INT"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mapper := NewTypeMapper(tt.dialect)
            result := mapper.MapType(tt.colType)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Pattern 2: Subtests

**Use for:** Grouping related tests.

```go
func TestSchemaValidation(t *testing.T) {
    t.Run("valid schemas", func(t *testing.T) {
        t.Run("simple table", func(t *testing.T) { /* ... */ })
        t.Run("with relationships", func(t *testing.T) { /* ... */ })
        t.Run("with indexes", func(t *testing.T) { /* ... */ })
    })

    t.Run("invalid schemas", func(t *testing.T) {
        t.Run("missing primary key", func(t *testing.T) { /* ... */ })
        t.Run("duplicate column", func(t *testing.T) { /* ... */ })
        t.Run("invalid reference", func(t *testing.T) { /* ... */ })
    })
}
```

### Pattern 3: Test Helpers

**Use for:** Reducing boilerplate in tests.

```go
// testutil/helpers.go
package testutil

func NewTestSandbox(t *testing.T) *runtime.Sandbox {
    t.Helper()
    sb := runtime.NewSandbox(nil)
    t.Cleanup(func() {
        sb.Close()
    })
    return sb
}

func MustEvalSchema(t *testing.T, sb *runtime.Sandbox, code string) *runtime.TableDef {
    t.Helper()
    tableDef, err := sb.EvalSchema(code, "test", "table")
    require.NoError(t, err)
    return tableDef
}

func CreateTestDB(t *testing.T, dialect string) *sql.DB {
    t.Helper()
    db, err := sql.Open(dialect, testDatabaseURL(dialect))
    require.NoError(t, err)
    t.Cleanup(func() {
        db.Close()
    })
    return db
}
```

**Usage:**

```go
func TestWithHelpers(t *testing.T) {
    sb := testutil.NewTestSandbox(t)
    tableDef := testutil.MustEvalSchema(t, sb, `table({ id: col.id() })`)
    assert.NotNil(t, tableDef)
}
```

### Pattern 4: Mock External Dependencies

**Use for:** Isolating units under test.

```go
// Using mockery (auto-generated mocks)
//go:generate mockery --name=DatabaseClient --output=mocks

type DatabaseClient interface {
    Execute(query string) error
    Query(query string) (*sql.Rows, error)
}

func TestMigrationRunner_WithMock(t *testing.T) {
    mockDB := new(mocks.DatabaseClient)
    mockDB.On("Execute", mock.MatchedBy(func(query string) bool {
        return strings.Contains(query, "CREATE TABLE")
    })).Return(nil)

    runner := NewMigrationRunner(mockDB)
    err := runner.ApplyMigration(testMigration)

    assert.NoError(t, err)
    mockDB.AssertExpectations(t)
}
```

### Pattern 5: Testing Error Cases

**Use for:** Ensuring proper error handling.

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectError bool
        errorMsg    string
    }{
        {
            name:        "syntax error",
            input:       `table({ id: col.id(`,
            expectError: true,
            errorMsg:    "syntax",
        },
        {
            name:        "runtime error",
            input:       `table({ id: col.invalid() })`,
            expectError: true,
            errorMsg:    "invalid",
        },
        {
            name:        "valid input",
            input:       `table({ id: col.id() })`,
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sb := NewSandbox(nil)
            _, err := sb.EvalSchema(tt.input, "test", "table")

            if tt.expectError {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errorMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Pattern 6: Testing with Real Databases

**Use for:** Integration tests that need real DB interactions.

```go
func TestPostgresMigrations(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping database integration test")
    }

    // Setup
    db := testutil.CreateTestDB(t, "postgres")
    defer db.Close()

    // Clean state
    _, err := db.Exec("DROP TABLE IF EXISTS auth_users CASCADE")
    require.NoError(t, err)

    // Test
    migration := generateMigration(testSchema)
    _, err = db.Exec(migration.UpSQL)
    require.NoError(t, err)

    // Verify
    assertTableExists(t, db, "auth_users")

    // Cleanup (rollback)
    _, err = db.Exec(migration.DownSQL)
    require.NoError(t, err)
}
```

### Pattern 7: Parallel Tests

**Use for:** Speeding up independent tests.

```go
func TestParallelExecution(t *testing.T) {
    tests := []struct {
        name string
        fn   func(t *testing.T)
    }{
        {"test1", testFunc1},
        {"test2", testFunc2},
        {"test3", testFunc3},
    }

    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Run in parallel
            tt.fn(t)
        })
    }
}
```

---

## Tools & Dependencies

### Core Testing Tools

| Tool           | Purpose              | Installation                                     |
| -------------- | -------------------- | ------------------------------------------------ |
| **testing**    | Go standard testing  | Built-in                                         |
| **testify**    | Assertions & mocking | `go get github.com/stretchr/testify`             |
| **goldie**     | Snapshot testing     | `go get github.com/sebdah/goldie/v2`             |
| **dockertest** | DB integration tests | `go get github.com/ory/dockertest/v3`            |
| **mockery**    | Mock generation      | `go install github.com/vektra/mockery/v2@latest` |

### Optional Tools

| Tool          | Purpose            | Installation                                          |
| ------------- | ------------------ | ----------------------------------------------------- |
| **go-fuzz**   | Fuzzing            | `go get github.com/dvyukov/go-fuzz`                   |
| **gotestsum** | Pretty test output | `go install gotest.tools/gotestsum@latest`            |
| **gocov**     | Coverage reports   | `go install github.com/axw/gocov/gocov@latest`        |
| **delve**     | Debugger           | `go install github.com/go-delve/delve/cmd/dlv@latest` |

### Running with gotestsum

```bash
# Install
go install gotest.tools/gotestsum@latest

# Run tests with nice output
gotestsum --format testname

# With coverage
gotestsum --format testname -- -coverprofile=coverage.out ./...
```

---

## CI/CD Integration

### GitHub Actions

Create `.github/workflows/test.yml`:

```yaml
name: Test Suite

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.21"

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Run unit tests
        run: go test -short -race -coverprofile=coverage.txt ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.txt
          flags: unittests

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: alab_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      mysql:
        image: mysql:8
        env:
          MYSQL_ROOT_PASSWORD: test
          MYSQL_DATABASE: alab_test
        options: >-
          --health-cmd "mysqladmin ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 3306:3306

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.21"

      - name: Run integration tests
        run: go test -tags=integration -race ./...
        env:
          DATABASE_URL_POSTGRES: postgresql://postgres:test@localhost:5432/alab_test?sslmode=disable
          DATABASE_URL_MYSQL: root:test@tcp(localhost:3306)/alab_test

  e2e-tests:
    name: E2E Tests
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.21"

      - name: Build binary
        run: go build -o alab ./cmd/alab

      - name: Run E2E tests
        run: |
          ./alab init
          ./alab table auth user
          ./alab new create_users
          ./alab migrate --dry

      - name: Test generator execution
        run: |
          ./alab gen run examples/generators/generators/fastapi.js -o ./output
          test -f output/auth/models.py
          test -f output/main.py

  lint:
    name: Lint
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.21"

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
```

---

## Best Practices

### 1. Test Naming Conventions

```go
// ✅ Good: Descriptive test names
func TestEvalSchema_WithTimestamps_AddsCreatedAndUpdatedColumns(t *testing.T)
func TestMigrationRollback_WithForeignKeys_PreservesIntegrity(t *testing.T)
func TestGenerator_FastAPI_GeneratesValidPydanticModels(t *testing.T)

// ❌ Bad: Vague test names
func TestEval(t *testing.T)
func TestMigration(t *testing.T)
func TestGenerator(t *testing.T)
```

### 2. Test Organization

```go
// ✅ Good: Use subtests for related scenarios
func TestSchemaValidation(t *testing.T) {
    t.Run("valid schemas", func(t *testing.T) {
        t.Run("basic table", func(t *testing.T) { /* ... */ })
        t.Run("with relationships", func(t *testing.T) { /* ... */ })
    })

    t.Run("invalid schemas", func(t *testing.T) {
        t.Run("missing primary key", func(t *testing.T) { /* ... */ })
    })
}

// ❌ Bad: Flat, hard to navigate
func TestSchemaValidation1(t *testing.T) { /* ... */ }
func TestSchemaValidation2(t *testing.T) { /* ... */ }
func TestSchemaValidation3(t *testing.T) { /* ... */ }
```

### 3. Use Table-Driven Tests

```go
// ✅ Good: Easy to add new test cases
func TestTypeMapping(t *testing.T) {
    tests := []struct {
        input    string
        dialect  string
        expected string
    }{
        {"uuid", "postgres", "UUID"},
        {"uuid", "sqlite", "TEXT"},
        {"integer", "postgres", "INTEGER"},
    }

    for _, tt := range tests {
        t.Run(tt.input+"-"+tt.dialect, func(t *testing.T) {
            result := MapType(tt.input, tt.dialect)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 4. Use Testify for Cleaner Assertions

```go
// ✅ Good: Clear, readable assertions
func TestSchema(t *testing.T) {
    schema := loadSchema()

    assert.NotNil(t, schema)
    assert.Equal(t, "user", schema.Name)
    assert.Len(t, schema.Columns, 5)
    assert.Contains(t, schema.Columns, "email")
}

// ❌ Bad: Manual error handling
func TestSchema(t *testing.T) {
    schema := loadSchema()

    if schema == nil {
        t.Fatal("schema is nil")
    }
    if schema.Name != "user" {
        t.Errorf("expected name 'user', got '%s'", schema.Name)
    }
    // ... more manual checks
}
```

### 5. Use Test Fixtures

```go
// ✅ Good: Reusable test data
func TestMigrationGeneration(t *testing.T) {
    schema := testutil.LoadFixture(t, "testdata/schemas/auth.js")
    migration := GenerateMigration(schema)
    assert.NotEmpty(t, migration.UpSQL)
}

// testdata/schemas/auth.js contains the test schema
```

### 6. Clean Up Resources

```go
// ✅ Good: Use t.Cleanup or defer
func TestWithDatabase(t *testing.T) {
    db := openTestDB()
    t.Cleanup(func() {
        db.Close()
    })

    // Or use defer
    defer db.Close()

    // Test code...
}
```

### 7. Skip Slow Tests in Short Mode

```go
func TestSlowIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Slow test code...
}
```

Run fast tests only:

```bash
go test ./... -short
```

### 8. Use Parallel Tests Wisely

```go
// ✅ Good: Parallel for independent tests
func TestIndependentOperations(t *testing.T) {
    t.Parallel() // OK - no shared state

    result := PureFunction(input)
    assert.Equal(t, expected, result)
}

// ❌ Bad: Parallel with shared state
func TestWithSharedDatabase(t *testing.T) {
    t.Parallel() // AVOID - shared DB can cause race conditions

    db.Insert(data)
    result := db.Query(query)
    // ...
}
```

### 9. Test Error Messages

```go
// ✅ Good: Verify error messages are helpful
func TestValidation_MissingPrimaryKey(t *testing.T) {
    schema := &Schema{Columns: []Column{{Name: "email"}}}

    err := ValidateSchema(schema)

    require.Error(t, err)
    assert.Contains(t, err.Error(), "primary key")
    assert.Contains(t, err.Error(), "required")
}
```

### 10. Document Complex Test Setup

```go
// ✅ Good: Explain non-obvious test setup
func TestMigrationRollback_WithCircularReferences(t *testing.T) {
    // Setup: Create a schema with circular foreign key references
    // (users -> posts -> comments -> users)
    // This tests that the rollback handles dependency order correctly

    schema := createCircularReferenceSchema()

    // Apply migrations
    applyMigrations(schema)

    // Rollback should drop tables in reverse dependency order
    err := rollbackMigrations(schema)
    assert.NoError(t, err)
}
```

---

## Resources & References

### Documentation & Guides

1. **Go Testing**
   - [Official Go Testing Package](https://pkg.go.dev/testing)
   - [Table Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
   - [Advanced Testing with Go](https://www.youtube.com/watch?v=8hQG7QlcLBk) (Mitchell Hashimoto)

2. **Embedded Runtime Testing**
   - [Testing in Go - Polyglot Programming](https://frontendmasters.com/courses/typescript-go-rust/testing-in-go/)
   - [Goja Documentation](https://github.com/dop251/goja)

3. **Integration Testing**
   - [dockertest Documentation](https://github.com/ory/dockertest)
   - [Testing with Real Databases](https://www.ardanlabs.com/blog/2019/03/integration-testing-in-go-executing-tests-with-docker.html)

4. **Contract Testing**
   - [Pact Contract Testing with Go](https://github.com/pact-foundation/pact-go)
   - [Contract Testing in TypeScript and Go](https://www.gerbenvanadrichem.com/quality-assurance/contract-testing-with-pact-in-typescript-and-golang/)

5. **Full Stack Testing**
   - [Testing in 2026: Full Stack Strategies](https://www.nucamp.co/blog/testing-in-2026-jest-react-testing-library-and-full-stack-testing-strategies)
   - [Go vs TypeScript Backend Testing](https://dev.to/encore/typescript-vs-go-choosing-your-backend-language-2bc5)

### Tools & Libraries

- **testify**: <https://github.com/stretchr/testify>
- **goldie**: <https://github.com/sebdah/goldie>
- **dockertest**: <https://github.com/ory/dockertest>
- **mockery**: <https://github.com/vektra/mockery>
- **gotestsum**: <https://github.com/gotestyourself/gotestsum>
- **goja**: <https://github.com/dop251/goja>

### Community Resources

- [Go Testing Tips](https://github.com/golang/go/wiki/TestComments)
- [Awesome Go Testing](https://github.com/shomali11/go-interview#testing)
- [Go Proverbs](https://go-proverbs.github.io/) - "A little copying is better than a little dependency"

---

## Troubleshooting

### Common Issues

#### Issue: Tests fail with "JavaScript execution timeout"

**Solution:**

```go
// Increase timeout for slow operations
executor := NewJSExecutor(vm, 30*time.Second) // Default is 10s
```

#### Issue: Race conditions in parallel tests

**Solution:**

```bash
# Run with race detector
go test ./... -race

# If race is detected, remove t.Parallel() or fix shared state
```

#### Issue: Flaky integration tests

**Solution:**

```go
// Add retry logic for database connections
func connectWithRetry(t *testing.T, url string) *sql.DB {
    var db *sql.DB
    var err error

    for i := 0; i < 5; i++ {
        db, err = sql.Open("postgres", url)
        if err == nil {
            return db
        }
        time.Sleep(time.Second)
    }

    t.Fatalf("Failed to connect after retries: %v", err)
    return nil
}
```

#### Issue: Snapshot tests always failing

**Solution:**

```bash
# Regenerate snapshots if intentional change
go test ./... -update

# Or delete golden files and regenerate
rm -rf testdata/golden/*.golden
go test ./... -update
```

---

## Contributing Tests

When contributing to AstrolaDB, please ensure:

1. **All new code has tests** (aim for 80%+ coverage)
2. **Tests are in the appropriate layer** (unit vs integration)
3. **Integration tests use `-short` skip** for CI speed
4. **Test names are descriptive** and follow conventions
5. **Snapshots are committed** to version control
6. **CI passes** before submitting PR

**Pre-commit checklist:**

```bash
# Run all tests
go test ./...

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# Run with race detector
go test ./... -race

# Run linter
golangci-lint run

# Format code
gofmt -s -w .
```

---

## License

See [LICENSE](LICENSE) for details.

---

## Questions?

- **GitHub Issues**: <https://github.com/hlop3z/astroladb/issues>
- **Discussions**: <https://github.com/hlop3z/astroladb/discussions>
- **Documentation**: <https://hlop3z.github.io/astroladb/>
