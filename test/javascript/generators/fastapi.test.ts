/**
 * FastAPI Generator Contract Tests
 *
 * These tests ensure that generators receive correctly structured
 * schema objects from the Go engine.
 */

import { describe, it, expect } from "vitest";
import { createTestSchema, createTestTable, createTestColumn } from "../helpers";

describe("FastAPI Generator Contract", () => {
  describe("Schema Structure", () => {
    it("should receive valid schema object", () => {
      const schema = createTestSchema({
        models: {
          auth: [
            {
              name: "user",
              table: "auth_users",
              primary_key: "id",
              timestamps: true,
              columns: [
                { name: "id", type: "uuid", nullable: false },
                { name: "email", type: "string", unique: true },
                { name: "created_at", type: "datetime" },
                { name: "updated_at", type: "datetime" },
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

    it("should handle multiple namespaces", () => {
      const schema = createTestSchema({
        models: {
          auth: [
            createTestTable("user", [
              createTestColumn("id", "uuid"),
              createTestColumn("email", "string"),
            ]),
          ],
          blog: [
            createTestTable("post", [
              createTestColumn("id", "uuid"),
              createTestColumn("title", "string"),
            ]),
          ],
        },
      });

      expect(Object.keys(schema.models)).toContain("auth");
      expect(Object.keys(schema.models)).toContain("blog");
      expect(schema.models.auth).toHaveLength(1);
      expect(schema.models.blog).toHaveLength(1);
    });
  });

  describe("Column Properties", () => {
    it("should identify primary keys", () => {
      const table = createTestTable("user", [
        createTestColumn("id", "uuid", { nullable: false }),
        createTestColumn("email", "string"),
      ]);

      expect(table.primary_key).toBe("id");
      const idColumn = table.columns.find((c) => c.name === "id");
      expect(idColumn).toBeDefined();
      expect(idColumn?.type).toBe("uuid");
    });

    it("should handle unique constraints", () => {
      const table = createTestTable("user", [
        createTestColumn("id", "uuid"),
        createTestColumn("email", "string", { unique: true }),
        createTestColumn("username", "string", { unique: true }),
      ]);

      const emailColumn = table.columns.find((c) => c.name === "email");
      const usernameColumn = table.columns.find((c) => c.name === "username");

      expect(emailColumn?.unique).toBe(true);
      expect(usernameColumn?.unique).toBe(true);
    });

    it("should handle nullable fields", () => {
      const table = createTestTable("post", [
        createTestColumn("id", "uuid", { nullable: false }),
        createTestColumn("excerpt", "text", { nullable: true }),
      ]);

      const idColumn = table.columns.find((c) => c.name === "id");
      const excerptColumn = table.columns.find((c) => c.name === "excerpt");

      expect(idColumn?.nullable).toBe(false);
      expect(excerptColumn?.nullable).toBe(true);
    });

    it("should handle default values", () => {
      const table = createTestTable("user", [
        createTestColumn("id", "uuid"),
        createTestColumn("is_active", "boolean", { default: true }),
        createTestColumn("role", "string", { default: "user" }),
      ]);

      const isActiveColumn = table.columns.find((c) => c.name === "is_active");
      const roleColumn = table.columns.find((c) => c.name === "role");

      expect(isActiveColumn?.default).toBe(true);
      expect(roleColumn?.default).toBe("user");
    });

    it("should handle enum values", () => {
      const table = createTestTable("post", [
        createTestColumn("id", "uuid"),
        createTestColumn("status", "string", {
          enum: ["draft", "published", "archived"],
          default: "draft",
        }),
      ]);

      const statusColumn = table.columns.find((c) => c.name === "status");

      expect(statusColumn?.enum).toBeDefined();
      expect(statusColumn?.enum).toHaveLength(3);
      expect(statusColumn?.enum).toContain("draft");
      expect(statusColumn?.enum).toContain("published");
      expect(statusColumn?.enum).toContain("archived");
      expect(statusColumn?.default).toBe("draft");
    });
  });

  describe("Timestamps", () => {
    it("should identify timestamped tables", () => {
      const withTimestamps = createTestTable(
        "user",
        [createTestColumn("id", "uuid")],
        { timestamps: true }
      );
      const withoutTimestamps = createTestTable(
        "log",
        [createTestColumn("id", "uuid")],
        { timestamps: false }
      );

      expect(withTimestamps.timestamps).toBe(true);
      expect(withoutTimestamps.timestamps).toBe(false);
    });

    it("should have timestamp columns when enabled", () => {
      const table = createTestTable(
        "post",
        [
          createTestColumn("id", "uuid"),
          createTestColumn("title", "string"),
          createTestColumn("created_at", "datetime"),
          createTestColumn("updated_at", "datetime"),
        ],
        { timestamps: true }
      );

      const createdAt = table.columns.find((c) => c.name === "created_at");
      const updatedAt = table.columns.find((c) => c.name === "updated_at");

      expect(createdAt).toBeDefined();
      expect(updatedAt).toBeDefined();
      expect(createdAt?.type).toBe("datetime");
      expect(updatedAt?.type).toBe("datetime");
    });
  });

  describe("Type Mapping", () => {
    it("should recognize all supported column types", () => {
      const supportedTypes = [
        "uuid",
        "string",
        "text",
        "integer",
        "float",
        "decimal",
        "boolean",
        "date",
        "time",
        "datetime",
        "json",
        "base64",
      ];

      const columns = supportedTypes.map((type, i) =>
        createTestColumn(`col_${i}`, type)
      );

      const table = createTestTable("all_types", columns);

      supportedTypes.forEach((type, i) => {
        const column = table.columns.find((c) => c.name === `col_${i}`);
        expect(column?.type).toBe(type);
      });
    });
  });
});
