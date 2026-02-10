//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// GenerateBaselineOps Tests
// ===========================================================================

func TestClient_GenerateBaselineOps(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	// Create schema with a table (using correct JavaScript syntax)
	schema := `export default table({
	id: col.id(),
	name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "users.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate baseline operations
	ops, err := client.GenerateBaselineOps()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, ops)
	testutil.AssertTrue(t, len(ops) > 0, "should have at least one operation")

	// Verify the operation includes the users table
	foundTable := false
	for _, op := range ops {
		if strings.Contains(op.Table(), "users") {
			foundTable = true
			break
		}
	}
	testutil.AssertTrue(t, foundTable, "should include users table in baseline")
}

func TestClient_GenerateBaselineOps_NoSchemas(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate baseline with no schemas
	ops, err := client.GenerateBaselineOps()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, ops)
	testutil.AssertEqual(t, len(ops), 0)
}

// ===========================================================================
// GenerateBaselineContent Tests
// ===========================================================================

func TestClient_GenerateBaselineContent(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	// Create schema with correct JavaScript syntax
	schema := `export default table({
	id: col.id()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "items.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate baseline content
	content, err := client.GenerateBaselineContent("", 1)
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, len(content) > 0, "should generate content")
	testutil.AssertTrue(t, strings.Contains(content, "items"), "should include items table")
}

func TestClient_GenerateBaselineContent_Empty(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate baseline with no schemas
	content, err := client.GenerateBaselineContent("", 1)
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, len(content) > 0, "should still generate migration skeleton")
}

// ===========================================================================
// ParseMigrationFile Tests
// ===========================================================================

func TestClient_ParseMigrationFile(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create migration file
	migration := testutil.SimpleMigration(
		`    m.create_table("test.orders", t => { t.id("id") })`,
		`    m.drop_table("test.orders")`,
	)
	migrationPath := filepath.Join(migrationsDir, "001_orders.js")
	testutil.WriteFile(t, migrationPath, migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Parse migration file
	ops, err := client.ParseMigrationFile(migrationPath)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, ops)
	testutil.AssertEqual(t, len(ops), 1) // One create_table operation
}

func TestClient_ParseMigrationFile_InvalidPath(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Parse non-existent file
	_, err = client.ParseMigrationFile(filepath.Join(migrationsDir, "nonexistent.js"))
	testutil.AssertTrue(t, err != nil, "should error on non-existent file")
}

// ===========================================================================
// SaveMetadata Tests
// ===========================================================================

func TestClient_SaveMetadata(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	// Create schema with correct JavaScript syntax
	schema := `export default table({
	id: col.id()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "posts.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Save metadata (should work even with simple schema)
	err = client.SaveMetadata()
	// May not create file if there's no metadata to save, just verify no error
	testutil.AssertNoError(t, err)
}

func TestClient_SaveMetadataToFile(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	// Create schema with correct JavaScript syntax
	schema := `export default table({
	id: col.id()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "comments.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Save metadata to custom file
	customPath := filepath.Join(tmpDir, "custom-metadata.json")
	err = client.SaveMetadataToFile(customPath)
	testutil.AssertNoError(t, err)

	// Verify custom metadata file was created
	_, err = os.Stat(customPath)
	testutil.AssertNoError(t, err)
}
