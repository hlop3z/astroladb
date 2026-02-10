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
// MigrationGenerate Tests
// ===========================================================================

func TestClient_MigrationGenerate(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema := `export default table({
	id: col.id(),
	name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "products.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate migration for the initial schema
	migrationPath, err := client.MigrationGenerate("create_products")
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, len(migrationPath) > 0, "should return migration path")
	testutil.AssertTrue(t, strings.Contains(migrationPath, "create_products"), "should include migration name")

	// Verify migration file was created
	_, err = os.Stat(migrationPath)
	testutil.AssertNoError(t, err)
}

func TestClient_MigrationGenerate_NoChanges(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create schema
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

	// Generate and run first migration
	_, err = client.MigrationGenerate("initial")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Try to generate another migration with no schema changes
	_, err = client.MigrationGenerate("no_changes")
	// Should either error or create empty migration
	_ = err
}

func TestClient_MigrationGenerate_WithChanges(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema1 := `export default table({
	id: col.id(),
	name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "orders.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run first migration
	_, err = client.MigrationGenerate("create_orders")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Update schema with new column
	schema2 := `export default table({
	id: col.id(),
	name: col.string(100),
	status: col.string(50).optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "orders.js"), schema2)

	// Generate migration for the changes
	migrationPath, err := client.MigrationGenerate("add_status")
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, len(migrationPath) > 0, "should generate migration for changes")

	// Verify migration file contains the new column
	contentBytes, err := os.ReadFile(migrationPath)
	testutil.AssertNoError(t, err)
	content := string(contentBytes)
	testutil.AssertTrue(t, strings.Contains(content, "status"), "should include status column")
}

// ===========================================================================
// MigrationGenerateWithRenames Tests
// ===========================================================================

func TestClient_MigrationGenerateWithRenames(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema1 := `export default table({
	id: col.id(),
	username: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "accounts.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run first migration
	_, err = client.MigrationGenerate("create_accounts")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Update schema with renamed column (username -> user_name)
	schema2 := `export default table({
	id: col.id(),
	user_name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "accounts.js"), schema2)

	// Generate migration with empty renames list (will treat as add/drop)
	migrationPath, err := client.MigrationGenerateWithRenames("rename_username", nil)
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, len(migrationPath) > 0, "should generate migration")
}

// ===========================================================================
// MigrationGenerateInteractive Tests
// ===========================================================================

func TestClient_MigrationGenerateInteractive(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema := `export default table({
	id: col.id(),
	name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "categories.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate migration interactively with empty rename/backfill lists
	migrationPath, err := client.MigrationGenerateInteractive("create_categories", nil, nil)
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, len(migrationPath) > 0, "should generate migration")

	// Verify migration file was created
	_, err = os.Stat(migrationPath)
	testutil.AssertNoError(t, err)
}
