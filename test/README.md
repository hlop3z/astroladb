# AstrolaDB Test Suite

This directory contains all tests for the AstrolaDB project, organized by type and language.

## Directory Structure

```
test/
├── javascript/              # JavaScript/TypeScript tests
│   ├── generators/          # Generator API tests
│   ├── schema-dsl/          # Schema DSL tests
│   ├── types/               # TypeScript contract tests
│   ├── package.json         # JS test dependencies
│   └── README.md            # JS test documentation
│
├── fixtures/                # Shared test data
│   ├── schemas/             # Example schemas for testing
│   └── expected-outputs/    # Expected generator outputs
│
└── integration/             # Cross-language integration tests
    ├── go-js-boundary/      # Go ↔ JS data passing tests
    └── e2e/                 # End-to-end workflow tests
```

## Running Tests

### Go Tests

```bash
# All tests
go test ./...

# Unit tests only (fast)
go test ./... -short

# With coverage
go test ./... -coverprofile=coverage.out

# Integration tests
go test ./... -tags=integration
```

### JavaScript Tests

```bash
# Navigate to JS test directory
cd test/javascript

# Install dependencies
npm install

# Run tests
npm test

# Watch mode
npm run test:watch

# Coverage
npm run test:coverage
```

### Run Everything

```bash
# From project root
task test          # or make test, depending on your setup

# Or manually
go test ./... && cd test/javascript && npm test
```

## Test Organization

### By Layer (Testing Pyramid)

1. **Unit Tests (80%)** - `internal/**/*_test.go`
   - Fast, isolated Go tests
   - No external dependencies
   - Test individual functions/components

2. **Contract Tests (15%)** - `test/javascript/types/`
   - Verify Go ↔ JS API contracts
   - Ensure TypeScript types match Go structs
   - Test data marshaling

3. **Integration Tests (5%)** - `test/integration/` & `cmd/**/*_test.go`
   - Test complete workflows
   - Real database connections
   - End-to-end scenarios

### By Language

#### Go Tests (in source tree)
```
internal/
├── runtime/
│   ├── sandbox_test.go      # Tests JS execution
│   └── executor_test.go     # Tests timeout/isolation
├── sqlgen/
│   └── builder_test.go      # Tests SQL generation
└── engine/
    └── migration_test.go    # Tests migration engine
```

#### JavaScript Tests (in test/)
```
test/javascript/
├── generators/              # User-facing generator tests
├── schema-dsl/              # Schema definition tests
└── types/                   # Type contract tests
```

## Test Fixtures

Shared test data is in `test/fixtures/`:

```
fixtures/
├── schemas/
│   ├── auth.js              # Authentication schema
│   ├── blog.js              # Blog schema
│   └── ecommerce.js         # E-commerce schema
└── expected-outputs/
    ├── fastapi/             # Expected FastAPI output
    ├── go-chi/              # Expected Go Chi output
    └── rust-axum/           # Expected Rust Axum output
```

**Usage in Go tests:**
```go
schema := loadFixture(t, "test/fixtures/schemas/auth.js")
```

**Usage in JS tests:**
```typescript
import authSchema from '@fixtures/schemas/auth.js';
```

## CI/CD

Tests run automatically in GitHub Actions:

```yaml
# .github/workflows/test.yml
jobs:
  go-tests:
    - run: go test ./... -race -coverprofile=coverage.txt

  js-tests:
    - working-directory: test/javascript
      run: npm ci && npm test

  integration-tests:
    - run: go test ./... -tags=integration
```

## Writing New Tests

### Go Unit Test
```go
// internal/mypackage/myfile_test.go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    result := MyFunction("input")
    if result != "expected" {
        t.Errorf("got %s, want %s", result, "expected")
    }
}
```

### JavaScript Test
```typescript
// test/javascript/generators/mytest.test.ts
import { describe, it, expect } from 'vitest';

describe('My Generator', () => {
  it('should generate correct output', () => {
    const result = myGenerator(testSchema);
    expect(result).toContain('expected');
  });
});
```

### Integration Test
```go
// test/integration/e2e/workflow_test.go
// +build integration

package e2e

import "testing"

func TestFullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    // Test complete workflow
}
```

## Best Practices

1. **Keep tests close to code** - Go tests stay with Go source
2. **Use fixtures** - Share test data between Go and JS tests
3. **Tag integration tests** - Use `// +build integration` or `if testing.Short()`
4. **Test contracts** - Verify Go<->JS API stability
5. **Snapshot testing** - Use golden files for generator outputs

## More Information

- [Go Testing Guide](../../TESTING.md)
- [JavaScript Tests](javascript/README.md)
- [Contributing](../../CONTRIBUTING.md)
