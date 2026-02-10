/**
 * TypeScript Contract Tests
 *
 * These tests ensure that the TypeScript type definitions match
 * the actual objects passed from Go to JavaScript.
 */

import { describe, it, expect } from 'vitest';
import type { GeneratorSchema, SchemaTable, SchemaColumn } from '@types/generator';
import { createTestSchema } from '../helpers';

describe('TypeScript Contract Tests', () => {
  describe('GeneratorSchema', () => {
    it('should have required properties', () => {
      const schema = createTestSchema();

      // TypeScript will fail at compile time if these don't exist
      expect(schema.models).toBeDefined();
      expect(schema.tables).toBeDefined();
    });

    it('models should be keyed by namespace', () => {
      const schema = createTestSchema();

      expect(typeof schema.models).toBe('object');
      expect(Array.isArray(schema.models.auth)).toBe(true);
    });

    it('tables should be an array', () => {
      const schema = createTestSchema();

      expect(Array.isArray(schema.tables)).toBe(true);
    });
  });

  describe('SchemaTable', () => {
    it('should have required properties', () => {
      const schema = createTestSchema();
      const table = schema.tables[0];

      expect(table.name).toBeDefined();
      expect(table.table).toBeDefined();
      expect(table.primary_key).toBeDefined();
      expect(Array.isArray(table.columns)).toBe(true);
    });

    it('columns should be SchemaColumn[]', () => {
      const schema = createTestSchema();
      const table = schema.tables[0];

      expect(table.columns.length).toBeGreaterThan(0);

      const column = table.columns[0];
      expect(column.name).toBeDefined();
      expect(column.type).toBeDefined();
    });
  });

  describe('SchemaColumn', () => {
    it('should have required properties', () => {
      const schema = createTestSchema();
      const column = schema.tables[0].columns[0];

      expect(column.name).toBeDefined();
      expect(column.type).toBeDefined();
    });

    it('should have optional properties', () => {
      const schema = createTestSchema();
      const column = schema.tables[0].columns[0];

      // These are optional, so they may be undefined
      // TypeScript should allow this
      const nullable = column.nullable;
      const unique = column.unique;
      const defaultValue = column.default;
      const enumValues = column.enum;

      // If defined, they should be the correct type
      if (nullable !== undefined) expect(typeof nullable).toBe('boolean');
      if (unique !== undefined) expect(typeof unique).toBe('boolean');
      if (enumValues !== undefined) expect(Array.isArray(enumValues)).toBe(true);
    });
  });

  describe('Type Safety', () => {
    it('should not allow invalid schema', () => {
      // @ts-expect-error - missing required properties
      const invalid1: GeneratorSchema = {};

      // @ts-expect-error - models should be object, not array
      const invalid2: GeneratorSchema = {
        models: [],
        tables: [],
      };

      // @ts-expect-error - tables should be array, not object
      const invalid3: GeneratorSchema = {
        models: {},
        tables: {},
      };

      // These checks ensure TypeScript compilation catches errors
      expect(true).toBe(true);
    });

    it('should enforce readonly properties', () => {
      const schema = createTestSchema();

      // TypeScript should prevent modification of readonly properties
      // @ts-expect-error - models is readonly
      schema.models = {};

      // @ts-expect-error - tables is readonly
      schema.tables = [];

      expect(true).toBe(true);
    });
  });
});
