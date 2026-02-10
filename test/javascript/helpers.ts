/**
 * Shared test utilities for JavaScript/TypeScript tests
 */

import type { GeneratorSchema, SchemaTable, SchemaColumn } from '@types/generator';

/**
 * Creates a minimal test schema with sensible defaults
 */
export function createTestSchema(overrides?: Partial<GeneratorSchema>): GeneratorSchema {
  const defaultSchema: GeneratorSchema = {
    models: {
      auth: [
        {
          name: 'user',
          table: 'auth_users',
          primary_key: 'id',
          timestamps: true,
          columns: [
            { name: 'id', type: 'uuid', nullable: false },
            { name: 'email', type: 'string', unique: true },
            { name: 'created_at', type: 'datetime' },
            { name: 'updated_at', type: 'datetime' },
          ],
        },
      ],
    },
    tables: [],
  };

  // Flatten models into tables
  if (!overrides?.tables) {
    defaultSchema.tables = Object.values(defaultSchema.models).flat();
  }

  return {
    ...defaultSchema,
    ...overrides,
  };
}

/**
 * Creates a test table definition
 */
export function createTestTable(
  name: string,
  columns: SchemaColumn[],
  overrides?: Partial<SchemaTable>
): SchemaTable {
  return {
    name,
    table: `test_${name}`,
    primary_key: 'id',
    timestamps: false,
    columns,
    ...overrides,
  };
}

/**
 * Creates a test column definition
 */
export function createTestColumn(
  name: string,
  type: string,
  overrides?: Partial<SchemaColumn>
): SchemaColumn {
  return {
    name,
    type,
    ...overrides,
  };
}

/**
 * Asserts that generated code contains expected patterns
 */
export function assertCodeContains(code: string, patterns: string[]): void {
  for (const pattern of patterns) {
    if (!code.includes(pattern)) {
      throw new Error(`Code does not contain expected pattern: "${pattern}"`);
    }
  }
}

/**
 * Asserts that generated code does NOT contain patterns
 */
export function assertCodeNotContains(code: string, patterns: string[]): void {
  for (const pattern of patterns) {
    if (code.includes(pattern)) {
      throw new Error(`Code should not contain pattern: "${pattern}"`);
    }
  }
}
