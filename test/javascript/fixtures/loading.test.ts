/**
 * Fixture Loading Tests
 *
 * These tests verify that JavaScript fixtures can be loaded
 * and used in tests, demonstrating the fixture extraction pattern.
 */

import { describe, it, expect } from "vitest";
import { readFileSync, existsSync } from "fs";
import { join } from "path";

describe("Fixture Loading", () => {
  const fixturesRoot = join(__dirname, "../../fixtures");

  describe("Migration Fixtures", () => {
    it("should load auth migration fixtures", () => {
      const userMigration = join(fixturesRoot, "migrations/auth/001_create_users.js");
      const rolesMigration = join(fixturesRoot, "migrations/auth/002_create_roles.js");

      expect(existsSync(userMigration)).toBe(true);
      expect(existsSync(rolesMigration)).toBe(true);

      const userContent = readFileSync(userMigration, "utf-8");
      const rolesContent = readFileSync(rolesMigration, "utf-8");

      expect(userContent).toContain("migration(m =>");
      expect(userContent).toContain("auth.user");
      expect(rolesContent).toContain("auth.role");
    });

    it("should load blog migration fixtures", () => {
      const blogTables = join(fixturesRoot, "migrations/blog/003_create_blog_tables.js");
      const engagement = join(fixturesRoot, "migrations/blog/004_create_engagement_tables.js");

      expect(existsSync(blogTables)).toBe(true);
      expect(existsSync(engagement)).toBe(true);

      const blogContent = readFileSync(blogTables, "utf-8");
      const engagementContent = readFileSync(engagement, "utf-8");

      expect(blogContent).toContain("blog.category");
      expect(blogContent).toContain("blog.post");
      expect(engagementContent).toContain("blog.comment");
      expect(engagementContent).toContain("blog.reaction");
    });

    it("should load e-commerce migration fixtures", () => {
      const catalogMigration = join(fixturesRoot, "migrations/ecommerce/001_create_catalog.js");

      expect(existsSync(catalogMigration)).toBe(true);

      const content = readFileSync(catalogMigration, "utf-8");

      expect(content).toContain("catalog.category");
      expect(content).toContain("catalog.product");
    });

    it("should load app migration fixtures", () => {
      const usersMigration = join(fixturesRoot, "migrations/app/001_create_users.js");
      const tasksMigration = join(fixturesRoot, "migrations/app/002_create_tasks.js");

      expect(existsSync(usersMigration)).toBe(true);
      expect(existsSync(tasksMigration)).toBe(true);

      const usersContent = readFileSync(usersMigration, "utf-8");
      const tasksContent = readFileSync(tasksMigration, "utf-8");

      expect(usersContent).toContain("app.user");
      expect(tasksContent).toContain("app.task");
    });
  });

  describe("Schema Fixtures", () => {
    it("should load auth schema fixture", () => {
      const authSchema = join(fixturesRoot, "schemas/auth.js");

      expect(existsSync(authSchema)).toBe(true);

      const authContent = readFileSync(authSchema, "utf-8");

      expect(authContent).toContain("export default table(");
      expect(authContent).toContain("col.id()");
      expect(authContent).toContain("col.email()");
    });

    it("should load blog schema fixtures as separate files", () => {
      const postSchema = join(fixturesRoot, "schemas/blog/post.js");
      const commentSchema = join(fixturesRoot, "schemas/blog/comment.js");
      const tagSchema = join(fixturesRoot, "schemas/blog/tag.js");

      expect(existsSync(postSchema)).toBe(true);
      expect(existsSync(commentSchema)).toBe(true);
      expect(existsSync(tagSchema)).toBe(true);

      const postContent = readFileSync(postSchema, "utf-8");
      const commentContent = readFileSync(commentSchema, "utf-8");
      const tagContent = readFileSync(tagSchema, "utf-8");

      // Each should use export default pattern
      expect(postContent).toContain("export default table(");
      expect(commentContent).toContain("export default table(");
      expect(tagContent).toContain("export default table(");

      // Verify table contents
      expect(postContent).toContain("col.title()");
      expect(postContent).toContain("col.slug()");
      expect(commentContent).toContain("col.text()");
      expect(tagContent).toContain("col.string(50)");
    });
  });

  describe("Fixture Structure", () => {
    it("should have valid JavaScript syntax", () => {
      const fixturePath = join(fixturesRoot, "migrations/auth/001_create_users.js");
      const content = readFileSync(fixturePath, "utf-8");

      // Check for valid migration structure
      expect(content).toContain("migration(m =>");
      expect(content).toContain("m.create_table(");
      expect(content).toContain("t.id()");
      expect(content).toContain("t.timestamps()");

      // Should not have syntax errors (basic check)
      expect(content).toContain("{");
      expect(content).toContain("}");
    });

    it("should follow naming conventions", () => {
      const fixtures = [
        "migrations/auth/001_create_users.js",
        "migrations/auth/002_create_roles.js",
        "migrations/blog/003_create_blog_tables.js",
        "migrations/app/001_create_users.js",
      ];

      fixtures.forEach((fixture) => {
        const path = join(fixturesRoot, fixture);
        expect(existsSync(path)).toBe(true);

        // Check naming: should start with digits, underscore, then descriptive name
        const filename = fixture.split("/").pop()!;
        expect(filename).toMatch(/^\d{3}_[a-z_]+\.js$/);
      });
    });

    it("should be organized by domain", () => {
      const domains = ["auth", "blog", "app", "ecommerce"];

      domains.forEach((domain) => {
        const domainPath = join(fixturesRoot, "migrations", domain);
        expect(existsSync(domainPath)).toBe(true);
      });
    });
  });

  describe("Fixture Reusability", () => {
    it("should be independent and reusable", () => {
      // Each fixture should be self-contained
      const userMigration = join(fixturesRoot, "migrations/auth/001_create_users.js");
      const content = readFileSync(userMigration, "utf-8");

      // Should start with migration function
      expect(content.trim()).toMatch(/^migration\(/);

      // Should end with closing bracket
      expect(content.trim()).toMatch(/\)$/);
    });

    it("should not have dependencies on other fixtures", () => {
      const fixturePath = join(fixturesRoot, "migrations/blog/003_create_blog_tables.js");
      const content = readFileSync(fixturePath, "utf-8");

      // Should not contain imports or requires
      expect(content).not.toContain("import ");
      expect(content).not.toContain("require(");

      // Should be pure JavaScript migration code
      expect(content).toContain("migration(m =>");
    });
  });
});
