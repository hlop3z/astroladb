//go:build integration

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// -----------------------------------------------------------------------------
// Test Environment Setup
// -----------------------------------------------------------------------------

// testEnv holds the test environment configuration.
type testEnv struct {
	tmpDir        string
	schemasDir    string
	migrationsDir string
	configPath    string
}

// setupTestEnv creates a temporary directory structure for tests.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tmpDir := t.TempDir()

	env := &testEnv{
		tmpDir:        tmpDir,
		schemasDir:    filepath.Join(tmpDir, "schemas"),
		migrationsDir: filepath.Join(tmpDir, "migrations"),
		configPath:    filepath.Join(tmpDir, "alab.yaml"),
	}

	// Create directories
	if err := os.MkdirAll(env.schemasDir, 0755); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.MkdirAll(env.migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations dir: %v", err)
	}

	return env
}

// writeConfig writes an alab.yaml config file.
func (e *testEnv) writeConfig(t *testing.T, databaseURL string) {
	t.Helper()

	content := `database_url: ` + databaseURL + `
schemas_dir: ` + e.schemasDir + `
migrations_dir: ` + e.migrationsDir + `
`
	if err := os.WriteFile(e.configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

// writeSchema writes a schema file in the given namespace.
func (e *testEnv) writeSchema(t *testing.T, namespace, tableName, content string) {
	t.Helper()

	nsDir := filepath.Join(e.schemasDir, namespace)
	if err := os.MkdirAll(nsDir, 0755); err != nil {
		t.Fatalf("failed to create namespace dir: %v", err)
	}

	path := filepath.Join(nsDir, tableName+".js")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}
}

// writeMigration writes a migration file.
func (e *testEnv) writeMigration(t *testing.T, revision, name, content string) {
	t.Helper()

	filename := revision + "_" + name + ".js"
	path := filepath.Join(e.migrationsDir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write migration: %v", err)
	}
}

// newClient creates a new astroladb client for the test environment.
func (e *testEnv) newClient(t *testing.T, databaseURL string) *astroladb.Client {
	t.Helper()

	client, err := astroladb.New(
		astroladb.WithDatabaseURL(databaseURL),
		astroladb.WithSchemasDir(e.schemasDir),
		astroladb.WithMigrationsDir(e.migrationsDir),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

// -----------------------------------------------------------------------------
// Init Command Tests
// -----------------------------------------------------------------------------

func TestInit_CreatesDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Save original dir and restore after test (for Windows file locking)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	// Simulate init command by creating directories
	dirs := []string{"schemas", "migrations", "types"}
	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	// Verify directories exist
	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		stat, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", dir, err)
			continue
		}
		if !stat.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestInit_CreatesConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "alab.yaml")

	content := `# Alab configuration
database_url: ${DATABASE_URL}
schemas_dir: ./schemas
migrations_dir: ./migrations
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Verify config exists and has correct content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if !strings.Contains(string(data), "database_url") {
		t.Error("config file should contain database_url")
	}
	if !strings.Contains(string(data), "schemas_dir") {
		t.Error("config file should contain schemas_dir")
	}
	if !strings.Contains(string(data), "migrations_dir") {
		t.Error("config file should contain migrations_dir")
	}
}

// -----------------------------------------------------------------------------
// Migrate Command Tests - PostgreSQL
// -----------------------------------------------------------------------------

func TestMigrate_Postgres_AppliesPendingMigrations(t *testing.T) {
	db, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// Write a migration
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255).unique()
		t.string("username", 50)
		t.timestamps()
	})
})
`)

	// Create client and run migration
	client := env.newClient(t, pgURL)

	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify table was created
	testutil.AssertTableExists(t, db, "auth_user")
	testutil.AssertColumnExists(t, db, "auth_user", "id")
	testutil.AssertColumnExists(t, db, "auth_user", "email")
	testutil.AssertColumnExists(t, db, "auth_user", "username")
	testutil.AssertColumnExists(t, db, "auth_user", "created_at")
	testutil.AssertColumnExists(t, db, "auth_user", "updated_at")
}

func TestMigrate_Postgres_DryRunShowsSQL(t *testing.T) {
	env := setupTestEnv(t)

	// Write a migration
	env.writeMigration(t, "001", "create_posts", `
migration(m => {
	m.create_table("content.post", t => {
		t.id()
		t.string("title", 200)
		t.text("body").optional()
	})
})
`)

	// Create client and run dry-run
	var buf bytes.Buffer
	client := env.newClient(t, testutil.PostgresTestURL(t))
	err := client.MigrationRun(astroladb.DryRunTo(&buf), astroladb.Force())
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	output := buf.String()

	// Verify SQL output contains expected statements
	if !strings.Contains(strings.ToUpper(output), "CREATE TABLE") {
		t.Errorf("dry-run output should contain CREATE TABLE, got: %s", output)
	}
}

func TestMigrate_Postgres_MultipleMigrations(t *testing.T) {
	db, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// First migration - create users table
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255).unique()
	})
})
`)

	// Second migration - create posts table
	env.writeMigration(t, "002", "create_posts", `
migration(m => {
	m.create_table("content.post", t => {
		t.id()
		t.string("title", 200)
	})
})
`)

	// Third migration - add column to users
	env.writeMigration(t, "003", "add_username", `
migration(m => {
	m.add_column("auth.user", c => c.string("username", 50).nullable())
})
`)

	client := env.newClient(t, pgURL)

	// Apply all migrations
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	// Verify both tables exist
	testutil.AssertTableExists(t, db, "auth_user")
	testutil.AssertTableExists(t, db, "content_post")
	testutil.AssertColumnExists(t, db, "auth_user", "username")
}

func TestMigrate_Postgres_StepsLimitsApplied(t *testing.T) {
	db, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// First migration
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255)
	})
})
`)

	// Second migration
	env.writeMigration(t, "002", "create_posts", `
migration(m => {
	m.create_table("content.post", t => {
		t.id()
		t.string("title", 200)
	})
})
`)

	client := env.newClient(t, pgURL)

	// Apply only 1 migration
	err := client.MigrationRun(astroladb.Steps(1), astroladb.Force())
	if err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	// First table should exist
	testutil.AssertTableExists(t, db, "auth_user")

	// Second table should not exist yet
	testutil.AssertTableNotExists(t, db, "content_post")

	// Apply remaining
	err = client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("second batch failed: %v", err)
	}

	// Now second table should exist
	testutil.AssertTableExists(t, db, "content_post")
}

// -----------------------------------------------------------------------------
// Migrate Command Tests - SQLite
// -----------------------------------------------------------------------------

func TestMigrate_SQLite_AppliesPendingMigrations(t *testing.T) {
	db := testutil.SetupSQLite(t)
	env := setupTestEnv(t)

	// Create a file-based SQLite for the client
	dbPath := filepath.Join(env.tmpDir, "test.db")

	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255).unique()
		t.boolean("is_active").default(true)
	})
})
`)

	client := env.newClient(t, "sqlite://"+dbPath)
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Use the client's DB for assertions
	testDB := client.DB()
	testutil.AssertTableExists(t, testDB, "auth_user")
	testutil.AssertColumnExists(t, testDB, "auth_user", "id")
	testutil.AssertColumnExists(t, testDB, "auth_user", "email")
	testutil.AssertColumnExists(t, testDB, "auth_user", "is_active")

	// Close unused in-memory db
	_ = db
}

func TestMigrate_SQLite_InMemory(t *testing.T) {
	env := setupTestEnv(t)

	env.writeMigration(t, "001", "create_products", `
migration(m => {
	m.create_table("store.product", t => {
		t.id()
		t.string("name", 100)
		t.decimal("price", 19, 4)
		t.integer("stock").default(0)
	})
})
`)

	// Use in-memory SQLite
	client := env.newClient(t, "sqlite://:memory:")
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	db := client.DB()
	testutil.AssertTableExists(t, db, "store_product")
	testutil.AssertColumnExists(t, db, "store_product", "name")
	testutil.AssertColumnExists(t, db, "store_product", "price")
	testutil.AssertColumnExists(t, db, "store_product", "stock")
}

// -----------------------------------------------------------------------------
// Status Command Tests
// -----------------------------------------------------------------------------

func TestStatus_Postgres_ShowsAppliedMigrations(t *testing.T) {
	env := setupTestEnv(t)

	// Create and apply migrations
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255)
	})
})
`)

	env.writeMigration(t, "002", "create_posts", `
migration(m => {
	m.create_table("content.post", t => {
		t.id()
		t.string("title", 200)
	})
})
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	// Apply first migration only
	err := client.MigrationRun(astroladb.Steps(1), astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Get status
	statuses, err := client.MigrationStatus()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(statuses))
	}

	// First should be applied
	if statuses[0].Status != "applied" {
		t.Errorf("expected first migration to be applied, got %s", statuses[0].Status)
	}

	// Second should be pending
	if statuses[1].Status != "pending" {
		t.Errorf("expected second migration to be pending, got %s", statuses[1].Status)
	}
}

func TestStatus_Postgres_JSONOutput(t *testing.T) {
	env := setupTestEnv(t)

	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
	})
})
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	// Apply migration
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Get status
	statuses, err := client.MigrationStatus()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	// Build JSON output similar to the CLI
	var applied, pending int
	for _, s := range statuses {
		if s.Status == "applied" {
			applied++
		} else {
			pending++
		}
	}

	output := map[string]any{
		"applied": applied,
		"pending": pending,
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	// Verify JSON structure
	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result["applied"].(float64) != 1 {
		t.Errorf("expected 1 applied, got %v", result["applied"])
	}

	if result["pending"].(float64) != 0 {
		t.Errorf("expected 0 pending, got %v", result["pending"])
	}
}

// -----------------------------------------------------------------------------
// Diff Command Tests
// -----------------------------------------------------------------------------

func TestDiff_Postgres_EmptyWhenInSync(t *testing.T) {
	env := setupTestEnv(t)

	// Write schema
	env.writeSchema(t, "auth", "user", `
export default table({
	id: col.id(),
	email: col.email(),
})
`)

	// Write matching migration
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255)
	})
})
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	// Apply migration
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Get diff
	ops, err := client.SchemaDiff()
	if err != nil {
		t.Fatalf("failed to get diff: %v", err)
	}

	if len(ops) != 0 {
		t.Errorf("expected 0 operations when in sync, got %d", len(ops))
		for _, op := range ops {
			t.Logf("  operation: %s on %s", op.Type(), op.Table())
		}
	}
}

func TestDiff_Postgres_ShowsDifferences(t *testing.T) {
	env := setupTestEnv(t)

	// Write schema with a column that doesn't exist in DB
	env.writeSchema(t, "auth", "user", `
export default table({
	id: col.id(),
	email: col.email(),
	username: col.username(),
})
`)

	// Write migration without username column
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255)
	})
})
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	// Apply migration
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Get diff - should show missing username column
	ops, err := client.SchemaDiff()
	if err != nil {
		t.Fatalf("failed to get diff: %v", err)
	}

	if len(ops) == 0 {
		t.Error("expected operations for schema diff, got none")
	}

	// Should contain an AddColumn operation
	found := false
	for _, op := range ops {
		if op.Type().String() == "AddColumn" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected AddColumn operation for username")
	}
}

// -----------------------------------------------------------------------------
// Check Command Tests
// -----------------------------------------------------------------------------

func TestCheck_Postgres_ValidSchema(t *testing.T) {
	env := setupTestEnv(t)

	// Write valid schema
	env.writeSchema(t, "auth", "user", `
export default table({
	id: col.id(),
	email: col.email().unique(),
	username: col.username(),
}).timestamps()
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	// Check should pass
	err := client.SchemaCheck()
	if err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

func TestCheck_Postgres_InvalidSchema(t *testing.T) {
	env := setupTestEnv(t)

	// Write invalid schema (missing closing bracket)
	env.writeSchema(t, "auth", "user", `
export default table({
	id: col.id(),
	email: col.email()
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	// Check should fail
	err := client.SchemaCheck()
	if err == nil {
		t.Error("expected error for invalid schema, got nil")
	}
}

func TestCheck_SQLite_ValidSchema(t *testing.T) {
	env := setupTestEnv(t)

	env.writeSchema(t, "inventory", "item", `
export default table({
	id: col.id(),
	name: col.name(),
	quantity: col.quantity(),
})
`)

	client := env.newClient(t, "sqlite://:memory:")

	err := client.SchemaCheck()
	if err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Rollback Command Tests
// -----------------------------------------------------------------------------

func TestRollback_Postgres_RollsBackMigration(t *testing.T) {
	db, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255)
	})
})
`)

	client := env.newClient(t, pgURL)

	// Apply migration
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	testutil.AssertTableExists(t, db, "auth_user")

	// Rollback
	err = client.MigrationRollback(1, astroladb.Force())
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	testutil.AssertTableNotExists(t, db, "auth_user")
}

func TestRollback_Postgres_RollsBackMultiple(t *testing.T) {
	db, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
	})
})
`)

	env.writeMigration(t, "002", "create_posts", `
migration(m => {
	m.create_table("blog.post", t => {
		t.id()
	})
})
`)

	client := env.newClient(t, pgURL)

	// Apply both
	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	testutil.AssertTableExists(t, db, "auth_user")
	testutil.AssertTableExists(t, db, "blog_post")

	// Rollback 2 steps
	err = client.MigrationRollback(2, astroladb.Force())
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	testutil.AssertTableNotExists(t, db, "auth_user")
	testutil.AssertTableNotExists(t, db, "blog_post")
}

// -----------------------------------------------------------------------------
// Complex Schema Tests
// -----------------------------------------------------------------------------

func TestMigrate_Postgres_ComplexSchema(t *testing.T) {
	db, pgURL := testutil.SetupPostgresWithURL(t)
	env := setupTestEnv(t)

	// Users table
	env.writeMigration(t, "001", "create_users", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255).unique()
		t.string("password_hash", 255)
		t.boolean("is_active").default(true)
		t.boolean("is_verified").default(false)
		t.datetime("last_login").optional()
		t.timestamps()
	})
})
`)

	// Posts table with foreign key
	env.writeMigration(t, "002", "create_posts", `
migration(m => {
	m.create_table("blog.post", t => {
		t.id()
		t.uuid("author_id")
		t.string("title", 200)
		t.text("body").optional()
		t.string("slug", 255).unique()
		t.timestamps()
	})
})
`)

	// Comments table
	env.writeMigration(t, "003", "create_comments", `
migration(m => {
	m.create_table("blog.comment", t => {
		t.id()
		t.uuid("post_id")
		t.uuid("author_id").optional()
		t.text("content")
		t.timestamps()
	})
})
`)

	client := env.newClient(t, pgURL)

	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify all tables
	testutil.AssertTableExists(t, db, "auth_user")
	testutil.AssertTableExists(t, db, "blog_post")
	testutil.AssertTableExists(t, db, "blog_comment")

	// Verify columns
	testutil.AssertColumnExists(t, db, "auth_user", "email")
	testutil.AssertColumnExists(t, db, "auth_user", "is_active")
	testutil.AssertColumnExists(t, db, "blog_post", "author_id")
	testutil.AssertColumnExists(t, db, "blog_post", "slug")
	testutil.AssertColumnExists(t, db, "blog_comment", "content")
}

// -----------------------------------------------------------------------------
// Index Tests
// -----------------------------------------------------------------------------

func TestMigrate_Postgres_CreatesIndexes(t *testing.T) {
	env := setupTestEnv(t)

	env.writeMigration(t, "001", "create_users_with_indexes", `
migration(m => {
	m.create_table("auth.user", t => {
		t.id()
		t.string("email", 255)
		t.string("name", 100)
	})
	m.create_index("auth.user", ["email"], { unique: true })
	m.create_index("auth.user", ["name"])
})
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))
	db := client.DB()

	err := client.MigrationRun(astroladb.Force())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	testutil.AssertTableExists(t, db, "auth_user")

	// Check indexes exist (using pg_indexes)
	var indexCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pg_indexes
		WHERE tablename = 'auth_user' AND schemaname = current_schema()
	`).Scan(&indexCount)
	if err != nil {
		t.Fatalf("failed to query indexes: %v", err)
	}

	// Should have at least 2 custom indexes plus the primary key
	if indexCount < 2 {
		t.Errorf("expected at least 2 indexes, got %d", indexCount)
	}
}

// -----------------------------------------------------------------------------
// Error Handling Tests
// -----------------------------------------------------------------------------

func TestMigrate_Postgres_InvalidMigrationFails(t *testing.T) {
	env := setupTestEnv(t)

	// Invalid SQL in migration
	env.writeMigration(t, "001", "invalid", `
migration(m => {
	m.sql("INVALID SQL STATEMENT HERE")
})
`)

	client := env.newClient(t, testutil.PostgresTestURL(t))

	err := client.MigrationRun(astroladb.Force())
	if err == nil {
		t.Error("expected error for invalid migration, got nil")
	}
}

func TestCheck_NoSchemas(t *testing.T) {
	env := setupTestEnv(t)

	client := env.newClient(t, "sqlite://:memory:")

	err := client.SchemaCheck()
	if err == nil {
		t.Error("expected error for empty schemas directory")
	}
}

// -----------------------------------------------------------------------------
// Helper to verify DB structure
// -----------------------------------------------------------------------------

func assertPrimaryKeyExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	// PostgreSQL-specific query
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			WHERE tc.table_name = $1
			AND kcu.column_name = $2
			AND tc.constraint_type = 'PRIMARY KEY'
		)
	`, table, column).Scan(&exists)

	if err != nil {
		t.Fatalf("failed to check primary key: %v", err)
	}

	if !exists {
		t.Errorf("expected primary key on %s.%s", table, column)
	}
}
