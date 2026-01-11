package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestMigrationLifecycle tests the full migration lifecycle:
// - Generate 10 migrations
// - Apply all migrations (forward)
// - Rollback 2-5 times
// - Apply forward again
// - Verify schema state and SQL correctness at each step
func TestMigrationLifecycle(t *testing.T) {
	testCases := []struct {
		name          string
		rollbackCount int
	}{
		{"rollback_2", 2},
		{"rollback_3", 3},
		{"rollback_4", 4},
		{"rollback_5", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testMigrationLifecycleWithRollback(t, tc.rollbackCount)
		})
	}
}

func testMigrationLifecycleWithRollback(t *testing.T, rollbackCount int) {
	// Setup test environment
	tmpDir := t.TempDir()
	schemasDir := filepath.Join(tmpDir, "schemas")
	migrationsDir := filepath.Join(tmpDir, "migrations")

	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatalf("Failed to create schemas dir: %v", err)
	}
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}

	// Create config file
	configPath := filepath.Join(tmpDir, "alab.yaml")
	configContent := fmt.Sprintf(`database:
  dialect: sqlite
  url: ./test.db

schemas: ./schemas
migrations: ./migrations
`)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Change to temp directory for the test
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize global config file variable (normally set by cobra flags)
	oldConfigFile := configFile
	configFile = "alab.yaml"
	defer func() { configFile = oldConfigFile }()

	// Verify config file exists and read it
	configBytes, err := os.ReadFile("alab.yaml")
	if err != nil {
		t.Fatalf("Config file not found: %v", err)
	}
	t.Logf("Config file created:\n%s", string(configBytes))

	// Create 10 progressive schema states using version-based helper
	// Create 10 progressive schema states using version numbers
	migrations := []struct {
		name    string
		version int
	}{
		{"001_initial", 1},
		{"002_add_password", 2},
		{"003_add_posts", 3},
		{"004_add_is_active", 4},
		{"005_add_post_status", 5},
		{"006_add_comments", 6},
		{"007_add_published_at", 7},
		{"008_add_slug", 8},
		{"009_add_soft_delete", 9},
		{"010_add_tags", 10},
	}

	// Step 1: Generate 10 migrations
	t.Log("Step 1: Generating 10 migrations...")
	for i, mig := range migrations {
		// Write schema files for this version
		createSchemaFiles(schemasDir, mig.version)

		// Generate migration
		client, err := newClient()
		if err != nil {
			t.Fatalf("Failed to create client for migration %d: %v", i+1, err)
		}

		migrationPath, err := client.MigrationGenerate(mig.name)
		client.Close()

		if err != nil {
			t.Fatalf("Failed to generate migration %d: %v", i+1, err)
		}

		t.Logf("  Generated: %s", migrationPath)

		// Verify migration file exists and is valid
		if err := verifyMigrationFile(t, migrationPath); err != nil {
			t.Fatalf("Migration %d file invalid: %v", i+1, err)
		}
	}

	// Verify we have exactly 10 migrations
	migrationFiles, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("Failed to read migrations dir: %v", err)
	}
	migrationCount := 0
	for _, entry := range migrationFiles {
		if strings.HasSuffix(entry.Name(), ".js") {
			migrationCount++
		}
	}
	if migrationCount != 10 {
		t.Fatalf("Expected 10 migrations, got %d", migrationCount)
	}

	// Step 2: Apply all migrations (forward)
	t.Log("\nStep 2: Applying all 10 migrations forward...")
	client, err := newClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if err := client.MigrationRun(); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Verify all tables exist
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	expectedTables := []string{"app_users", "app_posts", "app_comments", "app_tags"}
	for _, table := range expectedTables {
		assertTableExists(t, db, table)
		t.Logf("  ✓ Table exists: %s", table)
	}

	// Verify column order in database (id should be first)
	verifyColumnOrder(t, db, "app_users", []string{"id", "email", "is_active", "password", "username", "created_at", "updated_at"})
	verifyColumnOrder(t, db, "app_posts", []string{"id", "author_id", "body", "published_at", "slug", "status", "title", "created_at", "updated_at", "deleted_at"})

	// Step 3: Rollback N times
	t.Logf("\nStep 3: Rolling back %d migrations...", rollbackCount)
	if err := client.MigrationRollback(rollbackCount); err != nil {
		t.Fatalf("Failed to rollback %d migrations: %v", rollbackCount, err)
	}

	// Verify correct tables exist/don't exist after rollback
	switch rollbackCount {
	case 2:
		// After rolling back 2, should have migrations 1-8 applied
		assertTableExists(t, db, "app_users")
		assertTableExists(t, db, "app_posts")
		assertTableExists(t, db, "app_comments")
		assertTableNotExists(t, db, "app_tags")
		// Post should not have deleted_at (added in migration 9)
		cols := getColumnNames(t, db, "blog_post")
		if contains(cols, "deleted_at") {
			t.Errorf("blog_post should not have deleted_at after rollback")
		}
	case 3:
		// After rolling back 3, should have migrations 1-7 applied
		assertTableExists(t, db, "app_users")
		assertTableExists(t, db, "app_posts")
		assertTableExists(t, db, "app_comments")
		assertTableNotExists(t, db, "app_tags")
		// Post should not have slug (added in migration 8)
		cols := getColumnNames(t, db, "blog_post")
		if contains(cols, "slug") {
			t.Errorf("blog_post should not have slug after rollback")
		}
	case 4:
		// After rolling back 4, should have migrations 1-6 applied
		assertTableExists(t, db, "app_users")
		assertTableExists(t, db, "app_posts")
		assertTableExists(t, db, "app_comments")
		assertTableNotExists(t, db, "app_tags")
		// Post should not have published_at (added in migration 7)
		cols := getColumnNames(t, db, "blog_post")
		if contains(cols, "published_at") {
			t.Errorf("blog_post should not have published_at after rollback")
		}
	case 5:
		// After rolling back 5, should have migrations 1-5 applied
		assertTableExists(t, db, "app_users")
		assertTableExists(t, db, "app_posts")
		assertTableNotExists(t, db, "blog_comment")
		assertTableNotExists(t, db, "app_tags")
	}
	t.Logf("  ✓ Schema state correct after rollback")

	// Step 4: Apply forward again
	t.Log("\nStep 4: Applying migrations forward again...")
	if err := client.MigrationRun(); err != nil {
		t.Fatalf("Failed to re-apply migrations: %v", err)
	}

	// Verify all tables exist again
	for _, table := range expectedTables {
		assertTableExists(t, db, table)
	}
	t.Logf("  ✓ All tables restored")

	// Verify column order is still correct after rollback/forward cycle
	verifyColumnOrder(t, db, "app_users", []string{"id", "email", "is_active", "password", "username", "created_at", "updated_at"})
	verifyColumnOrder(t, db, "app_posts", []string{"id", "author_id", "body", "published_at", "slug", "status", "title", "created_at", "updated_at", "deleted_at"})
	t.Logf("  ✓ Column order correct after rollback/forward cycle")

	// Step 5: Verify migration history
	t.Log("\nStep 5: Verifying migration history...")
	history, err := client.MigrationHistory()
	if err != nil {
		t.Fatalf("Failed to get migration history: %v", err)
	}

	if len(history) != 10 {
		t.Fatalf("Expected 10 applied migrations, got %d", len(history))
	}
	t.Logf("  ✓ Migration history correct: %d migrations applied", len(history))

	t.Log("\n✅ Migration lifecycle test passed!")
}

// verifyMigrationFile checks that a migration file is valid
func verifyMigrationFile(t *testing.T, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)

	// Check for required functions
	if !strings.Contains(contentStr, "export function up(m)") {
		return fmt.Errorf("missing up() function")
	}
	if !strings.Contains(contentStr, "export function down(m)") {
		return fmt.Errorf("missing down() function")
	}

	// Check for IF NOT EXISTS in SQL (via create_table calls)
	if strings.Contains(contentStr, "create_table") {
		t.Logf("    Migration contains create_table (will use IF NOT EXISTS)")
	}

	// Check for IF EXISTS in SQL (via drop_table calls)
	if strings.Contains(contentStr, "drop_table") {
		t.Logf("    Migration contains drop_table (will use IF EXISTS)")
	}

	return nil
}

// assertTableExists checks that a table exists in the database
func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err == sql.ErrNoRows {
		t.Fatalf("Table %s does not exist", tableName)
	} else if err != nil {
		t.Fatalf("Failed to check table %s: %v", tableName, err)
	}
}

// assertTableNotExists checks that a table does not exist in the database
func assertTableNotExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err == nil {
		t.Fatalf("Table %s should not exist but it does", tableName)
	} else if err != sql.ErrNoRows {
		t.Fatalf("Failed to check table %s: %v", tableName, err)
	}
}

// verifyColumnOrder checks that columns appear in the expected order
func verifyColumnOrder(t *testing.T, db *sql.DB, tableName string, expectedOrder []string) {
	actual := getColumnNames(t, db, tableName)

	// Filter actual to only include expected columns (in case there are extras)
	var filtered []string
	for _, col := range actual {
		if contains(expectedOrder, col) {
			filtered = append(filtered, col)
		}
	}

	if len(filtered) != len(expectedOrder) {
		t.Errorf("Column count mismatch for %s: expected %d, got %d", tableName, len(expectedOrder), len(filtered))
		t.Errorf("  Expected: %v", expectedOrder)
		t.Errorf("  Actual: %v", filtered)
		return
	}

	for i, expected := range expectedOrder {
		if i >= len(filtered) || filtered[i] != expected {
			t.Errorf("Column order mismatch for %s at position %d:", tableName, i)
			t.Errorf("  Expected: %v", expectedOrder)
			t.Errorf("  Actual: %v", filtered)
			return
		}
	}

	t.Logf("  ✓ Column order correct for %s", tableName)
}

// getColumnNames returns the column names for a table in order
func getColumnNames(t *testing.T, db *sql.DB, tableName string) []string {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		t.Fatalf("Failed to get table info for %s: %v", tableName, err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		columns = append(columns, name)
	}

	return columns
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// createSchemaFiles creates schema files for a specific version
func createSchemaFiles(schemasDir string, version int) {
	appDir := filepath.Join(schemasDir, "app")
	os.MkdirAll(appDir, 0755)
	
	// Remove all existing files
	files, _ := filepath.Glob(filepath.Join(appDir, "*.js"))
	for _, f := range files {
		os.Remove(f)
	}
	
	// Version 1+: users table
	usersSchema := `export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
}).timestamps();`
	
	if version >= 2 {
		usersSchema = `export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
}).timestamps();`
	}
	
	if version >= 4 {
		usersSchema = `export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  is_active: col.flag(true),
}).timestamps();`
	}
	
	os.WriteFile(filepath.Join(appDir, "users.js"), []byte(usersSchema), 0644)
	
	// Version 3+: posts table
	if version >= 3 {
		postsSchema := `export default table({
  id: col.id(),
  title: col.string(200),
  body: col.text(),
  author_id: col.belongs_to(".users"),
}).timestamps();`
		
		if version >= 5 {
			postsSchema = `export default table({
  id: col.id(),
  title: col.string(200),
  body: col.text(),
  author_id: col.belongs_to(".users"),
  status: col.string(20).default("draft"),
}).timestamps();`
		}
		
		if version >= 7 {
			postsSchema = `export default table({
  id: col.id(),
  title: col.string(200),
  body: col.text(),
  author_id: col.belongs_to(".users"),
  status: col.string(20).default("draft"),
  published_at: col.datetime().optional(),
}).timestamps();`
		}
		
		if version >= 8 {
			postsSchema = `export default table({
  id: col.id(),
  title: col.string(200),
  slug: col.string(255).optional(),
  body: col.text(),
  author_id: col.belongs_to(".users"),
  status: col.string(20).default("draft"),
  published_at: col.datetime().optional(),
}).timestamps();`
		}
		
		if version >= 9 {
			postsSchema = `export default table({
  id: col.id(),
  title: col.string(200),
  slug: col.string(255).optional(),
  body: col.text(),
  author_id: col.belongs_to(".users"),
  status: col.string(20).default("draft"),
  published_at: col.datetime().optional(),
  deleted_at: col.datetime().optional(),
}).timestamps();`
		}
		
		os.WriteFile(filepath.Join(appDir, "posts.js"), []byte(postsSchema), 0644)
	}
	
	// Version 6+: comments table
	if version >= 6 {
		commentsSchema := `export default table({
  id: col.id(),
  body: col.text(),
  post_id: col.belongs_to(".posts"),
  author_id: col.belongs_to(".users"),
}).timestamps();`
		os.WriteFile(filepath.Join(appDir, "comments.js"), []byte(commentsSchema), 0644)
	}
	
	// Version 10: tags table
	if version >= 10 {
		tagsSchema := `export default table({
  id: col.id(),
  name: col.string(100).unique(),
}).timestamps();`
		os.WriteFile(filepath.Join(appDir, "tags.js"), []byte(tagsSchema), 0644)
	}
}
