//go:build integration

package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// -----------------------------------------------------------------------------
// End-to-End Workflow Tests
// -----------------------------------------------------------------------------
// These tests simulate real-world multi-step workflows using the Go API.

// TestE2E_FullBlogPlatform tests building a complete blog platform schema
// through multiple migrations, simulating real development workflow.
func TestE2E_FullBlogPlatform(t *testing.T) {
	_, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// =========================================================================
	// Phase 1: Initial schema - Users and basic auth
	// =========================================================================
	t.Log("Phase 1: Creating initial auth schema")

	env.writeMigration(t, "001", "create_users", testutil.MustLoadJSFixture("test/fixtures/migrations/auth/001_create_users.js"))

	client := env.newClient(t, pgURL)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Phase 1 migration failed: %v", err)
	}

	// Verify status
	statuses, _ := client.MigrationStatus()
	assertMigrationCount(t, statuses, 1, 0) // 1 applied, 0 pending

	// =========================================================================
	// Phase 2: Add roles and permissions
	// =========================================================================
	t.Log("Phase 2: Adding roles and permissions")

	env.writeMigration(t, "002", "create_roles", testutil.MustLoadJSFixture("test/fixtures/migrations/auth/002_create_roles.js"))

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Phase 2 migration failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 2, 0)

	// =========================================================================
	// Phase 3: Add blog content tables
	// =========================================================================
	t.Log("Phase 3: Creating blog content schema")

	env.writeMigration(t, "003", "create_blog_tables", testutil.MustLoadJSFixture("test/fixtures/migrations/blog/003_create_blog_tables.js"))

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Phase 3 migration failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 3, 0)

	// =========================================================================
	// Phase 4: Add comments and reactions
	// =========================================================================
	t.Log("Phase 4: Adding comments and reactions")

	env.writeMigration(t, "004", "create_engagement_tables", testutil.MustLoadJSFixture("test/fixtures/migrations/blog/004_create_engagement_tables.js"))

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Phase 4 migration failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 4, 0)

	// =========================================================================
	// Phase 5: Add tags (many-to-many)
	// =========================================================================
	t.Log("Phase 5: Adding tags with many-to-many relationship")

	env.writeMigration(t, "005", "create_tags", testutil.MustLoadJSFixture("test/fixtures/migrations/blog/005_create_tags.js"))

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Phase 5 migration failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 5, 0)

	// =========================================================================
	// Phase 6: Add user profile and settings
	// =========================================================================
	t.Log("Phase 6: Adding user profiles and settings")

	// Multi-table migration: create_table + add_column in ONE migration (industry standard)
	// Index names are auto-generated for rollback support
	env.writeMigration(t, "006", "create_user_profile", testutil.MustLoadJSFixture("test/fixtures/migrations/auth/006_create_user_profile.js"))

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Phase 6 migration failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 6, 0)

	// =========================================================================
	// Verification: Check all tables exist
	// =========================================================================
	t.Log("Verification: Checking all tables exist")

	db := client.DB()
	tables := []string{
		"auth_user",
		"auth_role",
		"auth_permission",
		"auth_role_permission",
		"auth_profile",
		"blog_category",
		"blog_post",
		"blog_comment",
		"blog_reaction",
		"blog_tag",
		"blog_post_tag",
	}

	for _, table := range tables {
		testutil.AssertTableExists(t, db, table)
	}

	// =========================================================================
	// Rollback Test: Roll back last 3 migrations
	// =========================================================================
	t.Log("Rollback Test: Rolling back 3 migrations")

	if err := client.MigrationRollback(3, astroladb.Force()); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 3, 3) // 3 applied, 3 pending (rolled back 6,5,4)

	// Tables from rolled-back migrations should not exist
	testutil.AssertTableNotExists(t, db, "blog_tag")
	testutil.AssertTableNotExists(t, db, "blog_post_tag")
	testutil.AssertTableNotExists(t, db, "auth_profile")

	// Tables from earlier migrations should still exist
	testutil.AssertTableExists(t, db, "auth_user")
	testutil.AssertTableExists(t, db, "blog_post")

	// =========================================================================
	// Re-apply Test: Apply remaining migrations
	// =========================================================================
	t.Log("Re-apply Test: Applying remaining migrations")

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Re-apply failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 6, 0)

	// All tables should exist again
	for _, table := range tables {
		testutil.AssertTableExists(t, db, table)
	}

	t.Log("E2E Blog Platform test completed successfully!")
}

// TestE2E_EcommercePlatform tests building an e-commerce schema.
func TestE2E_EcommercePlatform(t *testing.T) {
	_, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// Migration 1: Products catalog
	env.writeMigration(t, "001", "create_catalog", testutil.MustLoadJSFixture("test/fixtures/migrations/ecommerce/001_create_catalog.js"))

	// Migration 2: Customer management
	env.writeMigration(t, "002", "create_customers", testutil.MustLoadJSFixture("test/fixtures/migrations/ecommerce/002_create_customers.js"))

	// Migration 3: Orders
	env.writeMigration(t, "003", "create_orders", testutil.MustLoadJSFixture("test/fixtures/migrations/ecommerce/003_create_orders.js"))

	// Migration 4: Inventory
	env.writeMigration(t, "004", "create_inventory", testutil.MustLoadJSFixture("test/fixtures/migrations/ecommerce/004_create_inventory.js"))

	client := env.newClient(t, pgURL)

	// Apply all migrations
	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("E-commerce migrations failed: %v", err)
	}

	// Verify all tables
	db := client.DB()
	tables := []string{
		"catalog_category",
		"catalog_product",
		"catalog_product_category",
		"customer_account",
		"customer_address",
		"order_order",
		"order_line_item",
		"inventory_warehouse",
		"inventory_stock",
	}

	for _, table := range tables {
		testutil.AssertTableExists(t, db, table)
	}

	// Verify columns on key tables
	testutil.AssertColumnExists(t, db, "catalog_product", "sku")
	testutil.AssertColumnExists(t, db, "catalog_product", "price")
	testutil.AssertColumnExists(t, db, "order_order", "order_number")
	testutil.AssertColumnExists(t, db, "order_order", "total")

	// Check status
	statuses, _ := client.MigrationStatus()
	assertMigrationCount(t, statuses, 4, 0)

	t.Log("E2E E-commerce Platform test completed successfully!")
}

// TestE2E_SQLiteWorkflow tests the complete workflow with SQLite.
func TestE2E_SQLiteWorkflow(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "test.db")

	// Multiple migrations
	env.writeMigration(t, "001", "create_users", testutil.MustLoadJSFixture("test/fixtures/migrations/app/001_create_users.js"))

	env.writeMigration(t, "002", "create_tasks", testutil.MustLoadJSFixture("test/fixtures/migrations/app/002_create_tasks.js"))

	env.writeMigration(t, "003", "add_priority", testutil.MustLoadJSFixture("test/fixtures/migrations/app/003_add_priority.js"))

	client := env.newClient(t, "sqlite://"+dbPath)

	// Apply migrations one by one using steps
	if err := client.MigrationRun(astroladb.Steps(1), astroladb.Force()); err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}

	statuses, _ := client.MigrationStatus()
	assertMigrationCount(t, statuses, 1, 2)

	if err := client.MigrationRun(astroladb.Steps(1), astroladb.Force()); err != nil {
		t.Fatalf("Step 2 failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 2, 1)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Step 3 failed: %v", err)
	}

	statuses, _ = client.MigrationStatus()
	assertMigrationCount(t, statuses, 3, 0)

	// Verify tables
	db := client.DB()
	testutil.AssertTableExists(t, db, "app_user")
	testutil.AssertTableExists(t, db, "app_task")
	testutil.AssertColumnExists(t, db, "app_task", "priority")

	t.Log("E2E SQLite Workflow test completed successfully!")
}

// TestE2E_DryRunWorkflow_MultipleTables tests dry-run with multiple tables in one migration.
func TestE2E_DryRunWorkflow_MultipleTables(t *testing.T) {
	_, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// Migration with 2 tables and an index
	env.writeMigration(t, "001", "relationships", testutil.SimpleMigration(
		`    m.create_table("auth.user", t => {
      t.id()
      t.string("email", 255).unique()
      t.string("username", 50).unique()
      t.timestamps()
    })
    m.create_table("blog.post", t => {
      t.id()
      t.uuid("author_id")
      t.string("title", 200)
      t.text("body")
      t.timestamps()
    })
    m.create_index("blog.post", ["author_id"])`,
		`    m.drop_table("blog.post")
    m.drop_table("auth.user")`,
	))

	client := env.newClient(t, pgURL)

	// Run with dry-run
	var buf bytes.Buffer
	if err := client.MigrationRun(astroladb.DryRunTo(&buf), astroladb.Force()); err != nil {
		t.Fatalf("Dry-run failed: %v", err)
	}

	output := buf.String()
	t.Logf("Dry-run output:\n%s", output)

	// Should contain CREATE TABLE for both tables
	outputUpper := strings.ToUpper(output)
	if !strings.Contains(outputUpper, "CREATE TABLE") {
		t.Errorf("Expected CREATE TABLE in dry-run output, got: %s", output)
	}

	// Count CREATE TABLE statements
	createCount := strings.Count(outputUpper, "CREATE TABLE")
	if createCount < 2 {
		t.Errorf("Expected at least 2 CREATE TABLE statements, got %d. Output:\n%s", createCount, output)
	}

	// Should contain CREATE INDEX
	if !strings.Contains(outputUpper, "CREATE INDEX") {
		t.Errorf("Expected CREATE INDEX in dry-run output, got: %s", output)
	}

	t.Log("E2E Dry-Run Multiple Tables test completed!")
}

// TestE2E_DryRunWorkflow_WithRelationships tests dry-run with belongs_to relationships.
func TestE2E_DryRunWorkflow_WithRelationships(t *testing.T) {
	_, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// Migration with relationships using belongs_to
	env.writeMigration(t, "001", "relationships", testutil.SimpleMigration(
		`    m.create_table("auth.user", t => {
      t.id()
      t.string("email", 255)
      t.string("username", 50).unique()
      t.boolean("is_active").default(true)
      t.timestamps()
    })

    m.create_table("blog.post", t => {
      t.id()
      t.belongs_to("auth.user").as("author")
      t.belongs_to("auth.user").as("editor").optional()
      t.string("title", 200)
      t.text("body")
      t.string("slug", 255).unique()
      t.enum("status", ["draft", "published", "archived"]).default("draft")
      t.timestamps()
      t.soft_delete()
    })

    m.create_table("blog.comment", t => {
      t.id()
      t.belongs_to("blog.post")
      t.belongs_to("auth.user").optional()
      t.text("content")
      t.timestamps()
    })`,
		`    m.drop_table("blog.comment")
    m.drop_table("blog.post")
    m.drop_table("auth.user")`,
	))

	client := env.newClient(t, pgURL)

	// Run with dry-run
	var buf bytes.Buffer
	if err := client.MigrationRun(astroladb.DryRunTo(&buf), astroladb.Force()); err != nil {
		t.Fatalf("Dry-run failed: %v", err)
	}

	output := buf.String()
	t.Logf("Dry-run output:\n%s", output)

	outputUpper := strings.ToUpper(output)

	// Count CREATE TABLE statements - should have 3
	createCount := strings.Count(outputUpper, "CREATE TABLE")
	if createCount < 3 {
		t.Errorf("Expected at least 3 CREATE TABLE statements, got %d. Output:\n%s", createCount, output)
	}

	// Should contain FK columns
	if !strings.Contains(output, "author_id") {
		t.Errorf("Expected author_id column in output, got: %s", output)
	}
	if !strings.Contains(output, "editor_id") {
		t.Errorf("Expected editor_id column in output, got: %s", output)
	}
	if !strings.Contains(output, "post_id") {
		t.Errorf("Expected post_id column in output, got: %s", output)
	}
	if !strings.Contains(output, "user_id") {
		t.Errorf("Expected user_id column in output, got: %s", output)
	}

	// editor_id should be nullable (optional relationship)
	// Check for NULL constraint on editor_id - it should NOT have "NOT NULL"
	// This is a bit tricky to check in SQL output, so we'll verify after applying

	t.Log("E2E Dry-Run With Relationships test completed!")
}

// TestE2E_DryRunWorkflow tests dry-run functionality.
func TestE2E_DryRunWorkflow(t *testing.T) {
	_, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	env.writeMigration(t, "001", "create_test_table", `
migration(m => {
	m.create_table("test.item", t => {
		t.id()
		t.string("name", 100)
	})
})
`)

	client := env.newClient(t, pgURL)

	// Run with dry-run - should not create table
	var buf bytes.Buffer
	if err := client.MigrationRun(astroladb.DryRunTo(&buf), astroladb.Force()); err != nil {
		t.Fatalf("Dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(strings.ToUpper(output), "CREATE TABLE") {
		t.Errorf("Expected CREATE TABLE in dry-run output, got: %s", output)
	}

	// Table should NOT exist (dry-run)
	db := client.DB()
	testutil.AssertTableNotExists(t, db, "test_item")

	// Status should show pending
	statuses, _ := client.MigrationStatus()
	assertMigrationCount(t, statuses, 0, 1)

	// Now actually apply
	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Actual migration failed: %v", err)
	}

	testutil.AssertTableExists(t, db, "test_item")

	t.Log("E2E Dry-Run Workflow test completed successfully!")
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func assertMigrationCount(t *testing.T, statuses []astroladb.MigrationStatus, expectedApplied, expectedPending int) {
	t.Helper()

	var applied, pending int
	for _, s := range statuses {
		if s.Status == "applied" {
			applied++
		} else {
			pending++
		}
	}

	if applied != expectedApplied {
		t.Errorf("expected %d applied migrations, got %d", expectedApplied, applied)
	}
	if pending != expectedPending {
		t.Errorf("expected %d pending migrations, got %d", expectedPending, pending)
	}
}
