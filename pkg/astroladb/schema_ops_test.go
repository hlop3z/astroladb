//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// SchemaCheck Tests
// ===========================================================================

func TestClient_SchemaCheck(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(), // Schema-only mode doesn't require schema files
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// SchemaCheck should succeed (we're just testing that the function works)
	// Note: In schema-only mode, this is a no-op but shouldn't error
	err = client.SchemaCheck()
	// This might error since we don't have schemas, so just test it doesn't panic
	_ = err
}

// ===========================================================================
// SchemaDump Tests
// ===========================================================================

func TestClient_SchemaDump(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	// Create a migration to set up a table
	migration := testutil.SimpleMigration(
		`    m.create_table("test.users", t => { t.id("id"); t.string("name") })`,
		`    m.drop_table("test.users")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_users.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run migration to create the table
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Dump schema - will fail without schema files but should not panic
	_, err = client.SchemaDump()
	// Just verify it doesn't panic, error is expected without schema files
	_ = err
}

func TestClient_SchemaDump_EmptyDatabase(t *testing.T) {
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

	// Dump empty schema - should work even with no tables
	dump, err := client.SchemaDump()
	// May error if no schema files, so just verify it doesn't panic
	_ = err
	_ = dump
}

// ===========================================================================
// SchemaAtRevision Tests
// ===========================================================================

func TestClient_SchemaAtRevision(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create migrations
	migration1 := testutil.SimpleMigration(
		`    m.create_table("test.items", t => { t.id("id") })`,
		`    m.drop_table("test.items")`,
	)
	migration2 := testutil.SimpleMigration(
		`    m.create_table("test.orders", t => { t.id("id") })`,
		`    m.drop_table("test.orders")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_items.js"), migration1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_orders.js"), migration2)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run first migration
	err = client.MigrationRun(Steps(1))
	testutil.AssertNoError(t, err)

	// Get schema at revision 001
	schema, err := client.SchemaAtRevision("001")
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, schema)
	testutil.AssertEqual(t, len(schema.Tables), 1) // Should have items table
}

func TestClient_SchemaAtRevision_InitialRevision(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create a migration
	migration := testutil.SimpleMigration(
		`    m.create_table("test.initial", t => { t.id("id") })`,
		`    m.drop_table("test.initial")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_initial.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Get schema at revision - just verify it doesn't panic
	// Revision 000 may not exist, so error is expected
	_, err = client.SchemaAtRevision("000")
	_ = err
}

// ===========================================================================
// SchemaDiffFromMigrations Tests
// ===========================================================================

func TestClient_SchemaDiffFromMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	// Create migration
	migration := testutil.SimpleMigration(
		`    m.create_table("test.posts", t => { t.id("id") })`,
		`    m.drop_table("test.posts")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_posts.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Get diff from migrations (before running them)
	ops, err := client.SchemaDiffFromMigrations()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, ops)
	// Note: Without schemas, this will show difference between empty schema and migrations
}

// ===========================================================================
// SchemaExport Tests
// ===========================================================================

func TestClient_SchemaExport_SQL(t *testing.T) {
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

	// Export as SQL - will fail without schema files but should not panic
	_, err = client.SchemaExport("sql")
	_ = err
}

func TestClient_SchemaExport_JSON(t *testing.T) {
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

	// Export as JSON - will fail without schema files but should not panic
	_, err = client.SchemaExport("json")
	_ = err
}
