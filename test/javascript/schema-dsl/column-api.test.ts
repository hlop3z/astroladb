/**
 * Column API Contract Tests
 *
 * These tests ensure that column definitions have consistent
 * structure and properties across the DSL.
 */

import { describe, it, expect } from "vitest";
import { createTestColumn, createTestTable } from "../helpers";

describe("Column API Contract", () => {
  describe("Basic Properties", () => {
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

    it("should allow minimal column definition", () => {
      const column = createTestColumn("name", "string");

      expect(column.name).toBe("name");
      expect(column.type).toBe("string");
      // Optional properties should be undefined
      expect(column.unique).toBeUndefined();
      expect(column.nullable).toBeUndefined();
      expect(column.default).toBeUndefined();
    });
  });

  describe("Column Types", () => {
    it("should support string types", () => {
      const stringCol = createTestColumn("name", "string");
      const textCol = createTestColumn("description", "text");

      expect(stringCol.type).toBe("string");
      expect(textCol.type).toBe("text");
    });

    it("should support numeric types", () => {
      const intCol = createTestColumn("age", "integer");
      const floatCol = createTestColumn("rating", "float");
      const decimalCol = createTestColumn("price", "decimal");

      expect(intCol.type).toBe("integer");
      expect(floatCol.type).toBe("float");
      expect(decimalCol.type).toBe("decimal");
    });

    it("should support date/time types", () => {
      const dateCol = createTestColumn("birth_date", "date");
      const timeCol = createTestColumn("start_time", "time");
      const datetimeCol = createTestColumn("created_at", "datetime");

      expect(dateCol.type).toBe("date");
      expect(timeCol.type).toBe("time");
      expect(datetimeCol.type).toBe("datetime");
    });

    it("should support special types", () => {
      const uuidCol = createTestColumn("id", "uuid");
      const boolCol = createTestColumn("is_active", "boolean");
      const jsonCol = createTestColumn("metadata", "json");
      const base64Col = createTestColumn("avatar", "base64");

      expect(uuidCol.type).toBe("uuid");
      expect(boolCol.type).toBe("boolean");
      expect(jsonCol.type).toBe("json");
      expect(base64Col.type).toBe("base64");
    });
  });

  describe("Constraints", () => {
    it("should handle unique constraint", () => {
      const column = createTestColumn("email", "string", { unique: true });

      expect(column.unique).toBe(true);
    });

    it("should handle nullable constraint", () => {
      const required = createTestColumn("email", "string", { nullable: false });
      const optional = createTestColumn("phone", "string", { nullable: true });

      expect(required.nullable).toBe(false);
      expect(optional.nullable).toBe(true);
    });

    it("should combine constraints", () => {
      const column = createTestColumn("username", "string", {
        unique: true,
        nullable: false,
      });

      expect(column.unique).toBe(true);
      expect(column.nullable).toBe(false);
    });
  });

  describe("Default Values", () => {
    it("should handle boolean defaults", () => {
      const column = createTestColumn("is_active", "boolean", { default: true });

      expect(column.default).toBe(true);
    });

    it("should handle string defaults", () => {
      const column = createTestColumn("role", "string", { default: "user" });

      expect(column.default).toBe("user");
    });

    it("should handle numeric defaults", () => {
      const intCol = createTestColumn("count", "integer", { default: 0 });
      const floatCol = createTestColumn("rating", "float", { default: 0.0 });

      expect(intCol.default).toBe(0);
      expect(floatCol.default).toBe(0.0);
    });

    it("should handle null as default", () => {
      const column = createTestColumn("optional_field", "string", {
        nullable: true,
        default: null,
      });

      expect(column.nullable).toBe(true);
      expect(column.default).toBeNull();
    });
  });

  describe("Enum Values", () => {
    it("should store enum values", () => {
      const column = createTestColumn("status", "string", {
        enum: ["pending", "active", "archived"],
      });

      expect(column.enum).toBeDefined();
      expect(column.enum).toHaveLength(3);
      expect(column.enum).toContain("pending");
      expect(column.enum).toContain("active");
      expect(column.enum).toContain("archived");
    });

    it("should combine enum with default", () => {
      const column = createTestColumn("status", "string", {
        enum: ["draft", "published"],
        default: "draft",
      });

      expect(column.enum).toContain("draft");
      expect(column.enum).toContain("published");
      expect(column.default).toBe("draft");
    });

    it("should handle numeric enums", () => {
      const column = createTestColumn("priority", "integer", {
        enum: [1, 2, 3, 4, 5],
        default: 3,
      });

      expect(column.enum).toHaveLength(5);
      expect(column.enum).toContain(1);
      expect(column.enum).toContain(5);
      expect(column.default).toBe(3);
    });
  });

  describe("Integration with Tables", () => {
    it("should work within table context", () => {
      const table = createTestTable("users", [
        createTestColumn("id", "uuid", { nullable: false }),
        createTestColumn("email", "string", { unique: true }),
        createTestColumn("name", "string"),
        createTestColumn("is_active", "boolean", { default: true }),
      ]);

      expect(table.columns).toHaveLength(4);

      const idCol = table.columns[0];
      const emailCol = table.columns[1];
      const isActiveCol = table.columns[3];

      expect(idCol.nullable).toBe(false);
      expect(emailCol.unique).toBe(true);
      expect(isActiveCol.default).toBe(true);
    });

    it("should maintain column order", () => {
      const columns = [
        createTestColumn("first", "string"),
        createTestColumn("second", "string"),
        createTestColumn("third", "string"),
      ];

      const table = createTestTable("ordered", columns);

      expect(table.columns[0].name).toBe("first");
      expect(table.columns[1].name).toBe("second");
      expect(table.columns[2].name).toBe("third");
    });
  });

  describe("Edge Cases", () => {
    it("should handle empty string as name", () => {
      const column = createTestColumn("", "string");

      expect(column.name).toBe("");
      expect(column.type).toBe("string");
    });

    it("should handle special characters in names", () => {
      const column = createTestColumn("user_email_address", "string");

      expect(column.name).toBe("user_email_address");
    });

    it("should handle very long column names", () => {
      const longName = "a".repeat(100);
      const column = createTestColumn(longName, "string");

      expect(column.name).toBe(longName);
      expect(column.name.length).toBe(100);
    });
  });
});
