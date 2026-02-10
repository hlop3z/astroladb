# JavaScript/TypeScript Tests

This directory contains tests for JavaScript/TypeScript code including:
- User-facing generator API
- Schema DSL (table, col, etc.)
- TypeScript type contracts
- JavaScript runtime behavior

## Directory Structure

```
test/javascript/
├── generators/          # Tests for code generators
├── schema-dsl/          # Tests for schema definition API
├── types/               # TypeScript contract tests
└── helpers.ts           # Shared test utilities
```

## Running Tests

```bash
# Install dependencies
npm install

# Run all tests
npm test

# Watch mode (for development)
npm run test:watch

# Coverage report
npm run test:coverage

# Type check only
npm run type-check
```

## Writing Tests

### Example: Testing a Generator

```typescript
// generators/fastapi.test.ts
import { describe, it, expect } from 'vitest';
import type { GeneratorSchema } from '@types/generator';

describe('FastAPI Generator', () => {
  const testSchema: GeneratorSchema = {
    models: {
      auth: [
        {
          name: 'user',
          table: 'auth_users',
          primary_key: 'id',
          columns: [
            { name: 'id', type: 'uuid' },
            { name: 'email', type: 'string', unique: true },
          ],
        },
      ],
    },
    tables: [/* ... */],
  };

  it('should generate valid Pydantic models', () => {
    // Test implementation
  });
});
```

## Integration with Go Tests

These tests complement the Go tests in the main codebase:

- **Go tests** (`internal/**/*_test.go`): Test Go engine and Goja integration
- **JS tests** (here): Test JavaScript API and user-facing behavior
- **Integration tests** (`test/integration/`): Test Go ↔ JS boundary

## CI/CD

These tests run automatically in CI via:
```yaml
# .github/workflows/test.yml
- name: Run JavaScript tests
  working-directory: test/javascript
  run: |
    npm ci
    npm test
```
