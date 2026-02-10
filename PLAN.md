# Testing Infrastructure Refactor - Implementation Plan

> **Goal**: Organize test infrastructure for polyglot system (Go + embedded JavaScript), extract fixtures, add TypeScript tests, and ensure 80%+ coverage.

**Status**: 🔵 Planning
**Start Date**: 2026-02-09
**Estimated Duration**: 2-3 weeks
**Risk Level**: Low (incremental changes, existing tests remain functional)

---

## Table of Contents

- [Current State](#current-state)
- [Objectives](#objectives)
- [Implementation Phases](#implementation-phases)
- [Verification Strategy](#verification-strategy)
- [Success Criteria](#success-criteria)
- [Rollback Plan](#rollback-plan)

---

## Current State

### Existing Test Structure

```
astroladb/
├── internal/
│   ├── runtime/
│   │   ├── sandbox_test.go          # ✅ 15+ tests with embedded JS
│   │   ├── executor_test.go         # ✅ 8+ tests
│   │   └── object_api_test.go       # ✅ 10+ tests with embedded JS
│   ├── sqlgen/
│   │   └── builder_test.go          # ✅ 12+ tests
│   ├── engine/
│   │   ├── migration_integration_test.go
│   │   └── crossdb_integration_test.go
│   └── cli/
│       ├── cli_test.go
│       └── color_test.go
├── cmd/alab/
│   ├── migration_lifecycle_test.go  # ✅ Integration test (complex)
│   └── utc_timezone_e2e_test.go     # ✅ E2E test
└── testdata/
    └── (currently empty or minimal)
```

### Known Issues

1. **Embedded JS strings** - 20+ tests have JavaScript code as strings
2. **No JS fixtures** - Test data duplicated across tests
3. **No JS testing** - No tests for TypeScript/JavaScript code
4. **Mixed test types** - Integration tests mixed with unit tests
5. **No test organization guide** - Unclear where new tests should go

### Current Test Coverage

```bash
# Baseline (before changes)
go test ./... -coverprofile=coverage.baseline.out
# Expected: ~60-70% coverage
```

---

## Objectives

### Primary Goals

1. ✅ **Extract JS fixtures** - Move embedded JS to `test/fixtures/`
2. ✅ **Add JS test suite** - TypeScript tests for generators and contracts
3. ✅ **Organize integration tests** - Move to `test/integration/`
4. ✅ **Create test helpers** - Utilities for loading fixtures
5. ✅ **Document testing strategy** - Clear guidelines for developers

### Non-Goals

- ❌ Don't move Go unit tests from `internal/`
- ❌ Don't break existing CI/CD
- ❌ Don't change test behavior (only organization)
- ❌ Don't introduce new dependencies without approval

---

## Implementation Phases

### Phase 0: Preparation (Day 1)

**Goal**: Set up infrastructure and baseline

#### Tasks

- [x] Create test directory structure
- [x] Write TESTING.md guide
- [x] Write PLAN.md (this file)
- [ ] Establish baseline metrics

#### Commands

```bash
# 1. Create baseline coverage report
go test ./... -coverprofile=coverage.baseline.out
go tool cover -func=coverage.baseline.out | tee coverage.baseline.txt

# 2. Count existing tests
go test ./... -v -dry-run | grep -c "^=== RUN" > test_count.baseline.txt

# 3. Document current test count per package
go test ./... -v -json | jq -r '.Package' | sort | uniq -c > packages.baseline.txt

# 4. Git checkpoint
git add -A
git commit -m "docs: Add testing infrastructure documentation"
git tag testing-refactor-start
```

#### Success Criteria

- ✅ Baseline coverage report generated
- ✅ Test counts documented
- ✅ Git checkpoint created
- ✅ All tests passing

---

### Phase 1: Create Test Utilities (Days 2-3)

**Goal**: Build helper functions for fixture loading

#### Tasks

- [ ] Create `internal/testutil/fixtures.go`
- [ ] Create `internal/testutil/helpers.go`
- [ ] Add tests for test utilities
- [ ] Document usage in TESTING.md

#### Implementation

**File: `internal/testutil/fixtures.go`**

```go
package testutil

import (
    "os"
    "path/filepath"
    "testing"
)

// LoadJSFixture loads a JavaScript fixture file
func LoadJSFixture(t *testing.T, relativePath string) string {
    t.Helper()

    rootDir := findProjectRoot(t)
    fullPath := filepath.Join(rootDir, relativePath)

    content, err := os.ReadFile(fullPath)
    if err != nil {
        t.Fatalf("Failed to load fixture %s: %v", relativePath, err)
    }

    return string(content)
}

// findProjectRoot walks up to find go.mod
func findProjectRoot(t *testing.T) string {
    t.Helper()

    dir, err := os.Getwd()
    if err != nil {
        t.Fatalf("Failed to get working directory: %v", err)
    }

    for {
        goModPath := filepath.Join(dir, "go.mod")
        if _, err := os.Stat(goModPath); err == nil {
            return dir
        }

        parent := filepath.Dir(dir)
        if parent == dir {
            t.Fatal("Could not find project root (go.mod)")
        }
        dir = parent
    }
}

// LoadJSFixtureOrDefault loads fixture or returns default code
func LoadJSFixtureOrDefault(t *testing.T, relativePath, defaultCode string) string {
    t.Helper()

    rootDir := findProjectRoot(t)
    fullPath := filepath.Join(rootDir, relativePath)

    content, err := os.ReadFile(fullPath)
    if err != nil {
        return defaultCode
    }

    return string(content)
}
```

**File: `internal/testutil/helpers.go`**

```go
package testutil

import (
    "database/sql"
    "testing"
    "time"
)

// NewTestSandbox creates a sandbox for testing
func NewTestSandbox(t *testing.T) *runtime.Sandbox {
    t.Helper()
    sb := runtime.NewSandbox(nil)
    t.Cleanup(func() {
        sb.Close()
    })
    return sb
}

// CreateTestDB creates a test database connection
func CreateTestDB(t *testing.T, dialect, url string) *sql.DB {
    t.Helper()
    db, err := sql.Open(dialect, url)
    if err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }
    t.Cleanup(func() {
        db.Close()
    })
    return db
}

// RetryWithTimeout retries a function until it succeeds or times out
func RetryWithTimeout(t *testing.T, timeout time.Duration, fn func() error) error {
    t.Helper()

    deadline := time.Now().Add(timeout)
    var lastErr error

    for time.Now().Before(deadline) {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
            time.Sleep(100 * time.Millisecond)
        }
    }

    return lastErr
}
```

**File: `internal/testutil/fixtures_test.go`**

```go
package testutil

import (
    "os"
    "path/filepath"
    "testing"
)

func TestLoadJSFixture_Success(t *testing.T) {
    // Create temp fixture
    rootDir := findProjectRoot(t)
    fixturePath := filepath.Join(rootDir, "test", "fixtures", "test.js")
    os.MkdirAll(filepath.Dir(fixturePath), 0755)
    os.WriteFile(fixturePath, []byte("test content"), 0644)
    defer os.Remove(fixturePath)

    // Load fixture
    content := LoadJSFixture(t, "test/fixtures/test.js")

    if content != "test content" {
        t.Errorf("Expected 'test content', got '%s'", content)
    }
}

func TestLoadJSFixture_NotFound(t *testing.T) {
    // This should fail - capture panic
    defer func() {
        if r := recover(); r == nil {
            t.Error("Expected panic for missing fixture")
        }
    }()

    LoadJSFixture(t, "nonexistent.js")
}
```

#### Verification

```bash
# Run new tests
go test ./internal/testutil/... -v

# Ensure all existing tests still pass
go test ./... -short

# Check coverage hasn't decreased
go test ./... -coverprofile=coverage.phase1.out
```

#### Git Checkpoint

```bash
git add internal/testutil/
git commit -m "test: Add fixture loading utilities"
git tag testing-refactor-phase1
```

---

### Phase 2: Extract JavaScript Fixtures (Days 4-6)

**Goal**: Move embedded JS from tests to fixture files

#### Audit Current Embedded JS

```bash
# Find all embedded JS in tests
grep -r "jsCode :=" internal/ cmd/ --include="*_test.go" -A 5 > embedded_js_audit.txt

# Count occurrences
grep -r "jsCode :=" internal/ cmd/ --include="*_test.go" | wc -l
```

#### Categorize Fixtures

**Extract these to fixtures:**

1. `internal/runtime/sandbox_test.go`
   - `TestObjectBasedTableAPI` → `test/fixtures/schemas/auth/profile.js`
   - `TestObjectBasedTableAPI_WithChainedMethods` → `test/fixtures/schemas/blog/post.js`

2. `cmd/alab/migration_lifecycle_test.go`
   - `createSchemaFiles()` versions → `test/fixtures/schemas/versions/v{1-10}.js`

**Keep inline:**

1. Error cases (invalid syntax)
2. Simple 1-3 line tests
3. Test-specific variations

#### Create Fixture Files

**File: `test/fixtures/schemas/auth/user.js`**

```javascript
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  role: col.enum(["admin", "editor", "viewer"]).default("viewer"),
  is_active: col.flag(true),
}).timestamps();
```

**File: `test/fixtures/schemas/blog/post.js`**

```javascript
export default table({
  id: col.id(),
  title: col.title(),
  slug: col.slug(),
  author: col.belongs_to("auth.user"),
  category: col.belongs_to("blog.category").optional(),
})
  .timestamps()
  .soft_delete()
  .index("title")
  .unique("author", "slug");
```

**File: `test/fixtures/schemas/versions/v1.js`** (for migration tests)

```javascript
// schemas/app/users.js (version 1)
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
}).timestamps();
```

#### Update Tests to Use Fixtures

**Before:**

```go
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
    // ...
}
```

**After:**

```go
func TestEvalSchema_BasicTable(t *testing.T) {
    sb := testutil.NewTestSandbox(t)

    jsCode := testutil.LoadJSFixture(t, "test/fixtures/schemas/auth/user.js")

    tableDef, err := sb.EvalSchema(jsCode, "auth", "user")
    require.NoError(t, err)
    assert.Equal(t, "user", tableDef.Name)
}
```

#### Migration Order

1. **Day 4**: Extract `internal/runtime/` tests (10-15 fixtures)
2. **Day 5**: Extract `cmd/alab/` tests (migration lifecycle fixtures)
3. **Day 6**: Review, test, and validate

#### Verification Checklist

```bash
# 1. All tests still pass
go test ./... -v

# 2. Coverage hasn't decreased
go test ./... -coverprofile=coverage.phase2.out
diff coverage.phase1.txt coverage.phase2.txt

# 3. Verify fixtures are loaded correctly
ls -la test/fixtures/schemas/

# 4. Check for leftover embedded JS
grep -r "jsCode :=" internal/ cmd/ --include="*_test.go" | grep -v "LoadJSFixture"
```

#### Git Checkpoint

```bash
git add test/fixtures/
git add internal/
git add cmd/
git commit -m "test: Extract JavaScript fixtures from tests"
git tag testing-refactor-phase2
```

---

### Phase 3: Add TypeScript Test Suite (Days 7-9)

**Goal**: Set up and populate TypeScript test directory

#### Setup TypeScript Testing

**Already done in `test/javascript/`:**

- ✅ `package.json` with Vitest
- ✅ `tsconfig.json`
- ✅ `vitest.config.ts`
- ✅ `helpers.ts`
- ✅ Example contract test

#### Add Generator Tests

**File: `test/javascript/generators/fastapi.test.ts`**

```typescript
import { describe, it, expect } from "vitest";
import { createTestSchema } from "../helpers";

describe("FastAPI Generator Contract", () => {
  it("should receive valid schema object", () => {
    const schema = createTestSchema({
      models: {
        auth: [
          {
            name: "user",
            table: "auth_users",
            primary_key: "id",
            columns: [
              { name: "id", type: "uuid" },
              { name: "email", type: "string", unique: true },
            ],
          },
        ],
      },
    });

    expect(schema.models).toBeDefined();
    expect(schema.models.auth).toHaveLength(1);
    expect(schema.tables).toBeDefined();
  });

  it("should have required table properties", () => {
    const schema = createTestSchema();
    const table = schema.tables[0];

    expect(table.name).toBeDefined();
    expect(table.table).toBeDefined();
    expect(table.primary_key).toBeDefined();
    expect(table.columns).toBeInstanceOf(Array);
  });
});
```

**File: `test/javascript/schema-dsl/column-api.test.ts`**

```typescript
import { describe, it, expect } from "vitest";
import { createTestColumn } from "../helpers";

describe("Column API Contract", () => {
  it("should have required properties", () => {
    const column = createTestColumn("email", "string", { unique: true });

    expect(column.name).toBe("email");
    expect(column.type).toBe("string");
    expect(column.unique).toBe(true);
  });

  it("should handle optional properties", () => {
    const column = createTestColumn("count", "integer", {
      nullable: true,
      default: 0,
    });

    expect(column.nullable).toBe(true);
    expect(column.default).toBe(0);
  });
});
```

#### Install Dependencies

```bash
cd test/javascript
npm install
npm test
```

#### Verification

```bash
# Run TypeScript tests
cd test/javascript
npm test

# Type check
npm run type-check

# Coverage
npm run test:coverage
```

#### Git Checkpoint

```bash
git add test/javascript/
git commit -m "test: Add TypeScript test suite for contracts"
git tag testing-refactor-phase3
```

---

### Phase 4: Move Integration Tests (Days 10-11)

**Goal**: Organize integration tests in `test/integration/`

#### Files to Move

```bash
# Create integration test directory
mkdir -p test/integration/e2e

# Move files
git mv cmd/alab/migration_lifecycle_test.go test/integration/
git mv cmd/alab/utc_timezone_e2e_test.go test/integration/e2e/
git mv internal/engine/crossdb_integration_test.go test/integration/
git mv internal/engine/roundtrip_integration_test.go test/integration/
```

#### Update Package Names

**Before:**

```go
// cmd/alab/migration_lifecycle_test.go
package main
```

**After:**

```go
// test/integration/migration_lifecycle_test.go
package integration_test

import (
    "testing"
    "github.com/hlop3z/astroladb/cmd/alab"
    "github.com/hlop3z/astroladb/internal/engine"
)
```

#### Add Integration Test Helper

**File: `test/integration/integration_test.go`**

```go
// +build integration

package integration_test

import (
    "database/sql"
    "os"
    "testing"
)

func setupIntegrationTest(t *testing.T) {
    t.Helper()

    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup code
}

func cleanupIntegrationTest(t *testing.T, db *sql.DB) {
    t.Helper()
    // Cleanup code
}
```

#### Update CI/CD

**File: `.github/workflows/test.yml`** (add integration job)

```yaml
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
      ports:
        - 5432:5432

  steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: "1.21"

    - name: Run integration tests
      run: go test ./test/integration/... -tags=integration -v
      env:
        DATABASE_URL_POSTGRES: postgresql://postgres:test@localhost:5432/alab_test?sslmode=disable
```

#### Verification

```bash
# Run integration tests
go test ./test/integration/... -tags=integration -v

# Ensure unit tests still work
go test ./internal/... -short -v

# Full test suite
go test ./... -v
```

#### Git Checkpoint

```bash
git add test/integration/
git add .github/workflows/test.yml
git commit -m "test: Move integration tests to test/integration"
git tag testing-refactor-phase4
```

---

### Phase 5: Add New Tests (Days 12-14)

**Goal**: Increase coverage to 80%+

#### Coverage Analysis

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View by package
go tool cover -func=coverage.out | sort -k3 -n

# Identify low coverage packages
go tool cover -func=coverage.out | awk '$3 < 80 {print}'
```

#### Priority Areas for New Tests

Based on typical coverage gaps:

1. **Error handling paths** - Test error conditions
2. **Edge cases** - Boundary conditions
3. **Security** - Input validation, path traversal
4. **Concurrency** - Race conditions
5. **Database interactions** - SQL generation

#### Example: Add Security Tests

**File: `internal/runtime/security_test.go`**

```go
package runtime

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSandbox_PathTraversalProtection(t *testing.T) {
    tests := []struct {
        name      string
        path      string
        shouldErr bool
    }{
        {"absolute path", "/etc/passwd", true},
        {"relative traversal", "../../../etc/passwd", true},
        {"dotfile", ".env", true},
        {"valid path", "models/user.py", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateOutputPath(tt.path)

            if tt.shouldErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestSandbox_ScriptTimeout(t *testing.T) {
    sb := NewSandbox(nil)

    jsCode := `while(true) {}`  // Infinite loop

    _, err := sb.ExecuteWithTimeout(jsCode, 100*time.Millisecond)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "timeout")
}
```

#### Example: Add Edge Case Tests

**File: `internal/sqlgen/edge_cases_test.go`**

```go
package sqlgen

import "testing"

func TestSQLBuilder_EmptyTable(t *testing.T) {
    builder := NewBuilder("postgres")

    sql := builder.CreateTable("empty", []Column{})

    // Should handle gracefully (error or default)
    if sql != "" {
        t.Log("Generated SQL:", sql)
    }
}

func TestSQLBuilder_LongTableName(t *testing.T) {
    builder := NewBuilder("postgres")

    longName := strings.Repeat("a", 100)
    sql := builder.CreateTable(longName, testColumns)

    // PostgreSQL has 63 char limit
    assert.Contains(t, sql, longName[:63])
}

func TestSQLBuilder_SpecialCharacters(t *testing.T) {
    builder := NewBuilder("postgres")

    sql := builder.CreateTable("user-table", testColumns)

    // Should escape or sanitize
    assert.NotContains(t, sql, "user-table")
    assert.Contains(t, sql, "user_table") // Or error
}
```

#### Target Coverage Goals

| Package            | Current | Target | Priority |
| ------------------ | ------- | ------ | -------- |
| `internal/runtime` | 65%     | 85%    | High     |
| `internal/sqlgen`  | 70%     | 85%    | High     |
| `internal/engine`  | 60%     | 80%    | Medium   |
| `internal/cli`     | 55%     | 75%    | Medium   |
| `cmd/alab`         | 45%     | 70%    | Low      |

#### Verification

```bash
# Check new coverage
go test ./... -coverprofile=coverage.phase5.out
go tool cover -func=coverage.phase5.out | grep total

# Compare with baseline
echo "Baseline:"
go tool cover -func=coverage.baseline.out | grep total
echo "Current:"
go tool cover -func=coverage.phase5.out | grep total
```

#### Git Checkpoint

```bash
git add internal/
git add test/
git commit -m "test: Add security, edge case, and error handling tests"
git tag testing-refactor-phase5
```

---

### Phase 6: Documentation & Cleanup (Days 15-16)

**Goal**: Finalize documentation and remove temporary files

#### Update Documentation

1. **Update TESTING.md** - Add decision tree for test placement
2. **Update README.md** - Link to TESTING.md
3. **Add CONTRIBUTING.md section** - Testing requirements
4. **Create test/README.md** - Overview of test structure

#### Add Decision Tree to TESTING.md

```markdown
## Where Should I Put My Test?

### Decision Tree

1. **Is this a JavaScript/TypeScript test?**
   → YES: `test/javascript/`
   → NO: Continue...

2. **Does it test a single Go package/function in isolation?**
   → YES: Next to source (`internal/pkg/file_test.go`)
   → NO: Continue...

3. **Does it require a real database or external services?**
   → YES: `test/integration/`
   → NO: Continue...

4. **Is it test data/fixtures shared across tests?**
   → YES: `test/fixtures/`
   → NO: Back to #2 (probably a unit test)
```

#### Clean Up

```bash
# Remove baseline files
rm coverage.baseline.out
rm test_count.baseline.txt
rm packages.baseline.txt
rm embedded_js_audit.txt

# Remove temp files
find . -name "*.tmp" -delete
find . -name ".DS_Store" -delete

# Format all code
gofmt -s -w .
cd test/javascript && npm run format
```

#### Update .gitignore

```bash
cat >> .gitignore <<'EOF'

# Test coverage
coverage.out
coverage.*.out
coverage.html

# Test artifacts
*.test
*.prof

# JavaScript tests
test/javascript/node_modules/
test/javascript/coverage/
test/javascript/dist/
EOF
```

#### Git Checkpoint

```bash
git add TESTING.md README.md CONTRIBUTING.md
git add test/README.md
git add .gitignore
git commit -m "docs: Update testing documentation and cleanup"
git tag testing-refactor-complete
```

---

## Verification Strategy

### Continuous Verification (After Each Phase)

```bash
#!/bin/bash
# verify.sh - Run after each phase

set -e

echo "🧪 Running verification..."

# 1. All tests pass
echo "1️⃣  Running all tests..."
go test ./... -v

# 2. Coverage hasn't decreased
echo "2️⃣  Checking coverage..."
go test ./... -coverprofile=coverage.current.out
COVERAGE=$(go tool cover -func=coverage.current.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Current coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE < 60" | bc -l) )); then
    echo "❌ Coverage below 60%!"
    exit 1
fi

# 3. No broken imports
echo "3️⃣  Checking imports..."
go build ./...

# 4. Linter passes
echo "4️⃣  Running linter..."
golangci-lint run

# 5. JavaScript tests (if applicable)
if [ -d "test/javascript" ]; then
    echo "5️⃣  Running JavaScript tests..."
    cd test/javascript
    npm test
    cd ../..
fi

echo "✅ All verifications passed!"
```

### Final Verification (After Phase 6)

```bash
#!/bin/bash
# final-verify.sh

set -e

echo "🎯 Final verification..."

# 1. Full test suite
go test ./... -race -coverprofile=coverage.final.out

# 2. Coverage comparison
BASELINE=$(go tool cover -func=coverage.baseline.out | grep total | awk '{print $3}')
FINAL=$(go tool cover -func=coverage.final.out | grep total | awk '{print $3}')

echo "Coverage: $BASELINE → $FINAL"

# 3. Test count comparison
BASELINE_COUNT=$(cat test_count.baseline.txt)
FINAL_COUNT=$(go test ./... -v -dry-run | grep -c "^=== RUN")

echo "Test count: $BASELINE_COUNT → $FINAL_COUNT"

# 4. All fixtures exist
echo "Checking fixtures..."
for fixture in test/fixtures/schemas/**/*.js; do
    if [ ! -f "$fixture" ]; then
        echo "❌ Missing fixture: $fixture"
        exit 1
    fi
done

# 5. TypeScript tests
cd test/javascript
npm test
npm run type-check
cd ../..

# 6. Build succeeds
go build ./cmd/alab

echo "✅ Final verification complete!"
echo "🎉 Testing refactor successful!"
```

---

## Success Criteria

### Must Have ✅

- [ ] All existing tests pass
- [ ] Coverage ≥ 60% (baseline maintained or improved)
- [ ] No broken imports or build failures
- [ ] CI/CD pipeline passes
- [ ] TESTING.md complete and accurate

### Should Have 🎯

- [ ] Coverage ≥ 80%
- [ ] TypeScript tests running in CI
- [ ] Integration tests separated
- [ ] All JS fixtures extracted
- [ ] Test count increased by 20%+

### Nice to Have 🌟

- [ ] Coverage ≥ 90%
- [ ] Property-based tests
- [ ] Benchmark tests
- [ ] Performance regression tests

---

## Rollback Plan

### If Phase Fails

```bash
# Rollback to previous phase
git checkout testing-refactor-phase{N-1}

# Or rollback individual changes
git revert <commit-hash>

# Restore baseline
git checkout testing-refactor-start -- .
```

### Emergency Rollback

```bash
# Nuclear option: revert all changes
git checkout testing-refactor-start
git branch testing-refactor-backup testing-refactor-complete
git reset --hard testing-refactor-start

# Keep documentation
git checkout testing-refactor-complete -- TESTING.md PLAN.md
```

---

## Risk Mitigation

### Known Risks

| Risk                    | Likelihood | Impact | Mitigation                    |
| ----------------------- | ---------- | ------ | ----------------------------- |
| Breaking existing tests | Low        | High   | Run tests after each change   |
| Coverage decrease       | Medium     | Medium | Monitor coverage continuously |
| CI/CD breakage          | Low        | High   | Test CI locally first         |
| Fixture path issues     | Medium     | Low    | Use helper functions          |
| Team confusion          | Medium     | Medium | Clear documentation           |

### Mitigation Strategies

1. **Incremental changes** - Small commits, frequent verification
2. **Git tags** - Checkpoint after each phase
3. **Parallel branches** - Keep main stable
4. **Code review** - Review each phase before proceeding
5. **Documentation** - Update docs as we go

---

## Timeline

```
Week 1: Phases 0-2 (Setup + Extract Fixtures)
├── Mon:  Phase 0 - Preparation
├── Tue:  Phase 1 - Test Utilities
├── Wed:  Phase 2 - Extract Fixtures (Day 1)
├── Thu:  Phase 2 - Extract Fixtures (Day 2)
└── Fri:  Phase 2 - Extract Fixtures (Day 3) + Review

Week 2: Phases 3-4 (TypeScript + Integration)
├── Mon:  Phase 3 - TypeScript Tests (Day 1)
├── Tue:  Phase 3 - TypeScript Tests (Day 2)
├── Wed:  Phase 3 - TypeScript Tests (Day 3)
├── Thu:  Phase 4 - Move Integration Tests (Day 1)
└── Fri:  Phase 4 - Move Integration Tests (Day 2)

Week 3: Phases 5-6 (New Tests + Documentation)
├── Mon:  Phase 5 - New Tests (Day 1)
├── Tue:  Phase 5 - New Tests (Day 2)
├── Wed:  Phase 5 - New Tests (Day 3)
├── Thu:  Phase 6 - Documentation
└── Fri:  Phase 6 - Final Review + Cleanup
```

---

## Progress Tracking

### Phase Completion Checklist

- [x] Phase 0: Preparation
- [ ] Phase 1: Create Test Utilities
- [ ] Phase 2: Extract JavaScript Fixtures
- [ ] Phase 3: Add TypeScript Test Suite
- [ ] Phase 4: Move Integration Tests
- [ ] Phase 5: Add New Tests
- [ ] Phase 6: Documentation & Cleanup

### Daily Standup Template

```markdown
## Testing Refactor - Daily Update

**Date**: YYYY-MM-DD
**Phase**: N - [Phase Name]
**Status**: 🟢 On Track / 🟡 At Risk / 🔴 Blocked

### Yesterday

- Completed: [tasks]
- Blockers: [issues]

### Today

- Plan: [tasks]
- Focus: [priority]

### Metrics

- Tests passing: X/Y
- Coverage: Z%
- Files changed: N
```

---

## Resources

### Documentation

- [TESTING.md](TESTING.md) - Testing guide
- [test/README.md](test/README.md) - Test structure overview
- [test/javascript/README.md](test/javascript/README.md) - JS test guide

### Tools

- Go testing: https://pkg.go.dev/testing
- Testify: https://github.com/stretchr/testify
- Vitest: https://vitest.dev/
- Goldie: https://github.com/sebdah/goldie

### References

- [Go Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
- [Testing Polyglot Systems](https://frontendmasters.com/courses/typescript-go-rust/testing/)
- [Contract Testing Guide](https://www.gerbenvanadrichem.com/quality-assurance/contract-testing-with-pact-in-typescript-and-golang/)

---

## Sign-off

**Plan Created**: 2026-02-09
**Last Updated**: 2026-02-09
**Status**: 🔵 Ready to Execute

**Next Steps**:

1. Review plan with team
2. Get approval for timeline
3. Start Phase 1 execution
4. Daily progress updates

---

**Questions or Issues?**

- GitHub Issues: https://github.com/hlop3z/astroladb/issues
- Discussions: https://github.com/hlop3z/astroladb/discussions
