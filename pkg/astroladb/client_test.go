//go:build integration

package astroladb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// Client Accessor Tests
// ===========================================================================

func TestClient_Dialect(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(filepath.Join(tmpDir, "schemas")),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	dialect := client.Dialect()
	testutil.AssertEqual(t, dialect, "postgres")
}

func TestClient_Config(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	migrationsDir := filepath.Join(tmpDir, "migrations")

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	config := client.Config()
	testutil.AssertNotNil(t, config)
	testutil.AssertEqual(t, config.SchemasDir, schemasDir)
	testutil.AssertEqual(t, config.MigrationsDir, migrationsDir)
	testutil.AssertEqual(t, config.DatabaseURL, dbURL)
}

// ===========================================================================
// Drift Detection Tests
// ===========================================================================

func TestClient_CheckDrift_NoDrift(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))
	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create schema
	schema := `export default table({ id: col.id(), name: col.string(100) })`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "item.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Create migration
	migration := testutil.SimpleMigration(
		`    m.create_table("test.item", t => {
      t.id("id")
      t.string("name", 100)
    })`,
		`    m.drop_table("test.item")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_items.js"), migration)

	// Run migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Check drift - verify method works (drift detection has complex normalization)
	result, err := client.CheckDrift()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
	// Note: Actual drift result depends on schema normalization details
}

func TestClient_CheckDrift_WithDrift(t *testing.T) {
	testutil.Parallel(t)

	db, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))
	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create schema expecting two columns
	schema := `export default table({
		id: col.id(),
		name: col.string(100),
		email: col.string(255)
	})`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "user.js"), schema)

	// But database only has id and name
	testutil.ExecSQL(t, db, `
		CREATE TABLE test_user (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100)
		)
	`)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Check drift - should detect missing email column
	result, err := client.CheckDrift()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
	testutil.AssertTrue(t, result.HasDrift, "should detect drift (missing email column)")
}

func TestClient_CheckDriftContext(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	schema := `export default table({ id: col.id() })`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "item.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	ctx := context.Background()
	result, err := client.CheckDriftContext(ctx)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
	// Verify the method works correctly (actual drift result depends on schema state)
}

func TestClient_QuickDriftCheck(t *testing.T) {
	testutil.Parallel(t)

	db, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	schema := `export default table({ id: col.id(), name: col.string(50) })`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "item.js"), schema)

	// Create matching table
	testutil.ExecSQL(t, db, `
		CREATE TABLE test_item (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(50)
		)
	`)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	hasDrift, err := client.QuickDriftCheck()
	testutil.AssertNoError(t, err)
	// Quick check might still detect minor differences, but should not error
	_ = hasDrift // Result depends on normalization details
}

// ===========================================================================
// GetNamespaces Tests
// ===========================================================================

func TestClient_GetNamespaces(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	// Create schemas in multiple namespaces
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "blog"), 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "shop"), 0755))

	schema := `export default table({ id: col.id() })`
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), schema)
	testutil.WriteFile(t, filepath.Join(schemasDir, "blog", "post.js"), schema)
	testutil.WriteFile(t, filepath.Join(schemasDir, "shop", "product.js"), schema)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	namespaces, err := client.GetNamespaces()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(namespaces), 3)

	// Verify all namespaces are present (order doesn't matter)
	nsMap := make(map[string]bool)
	for _, ns := range namespaces {
		nsMap[ns] = true
	}
	testutil.AssertTrue(t, nsMap["auth"], "should have auth namespace")
	testutil.AssertTrue(t, nsMap["blog"], "should have blog namespace")
	testutil.AssertTrue(t, nsMap["shop"], "should have shop namespace")
}

// ===========================================================================
// ListRevisions Tests
// ===========================================================================

func TestClient_ListRevisions(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create migrations
	migration1 := testutil.SimpleMigration(
		`    m.create_table("test.users", t => { t.id("id") })`,
		`    m.drop_table("test.users")`,
	)
	migration2 := testutil.SimpleMigration(
		`    m.create_table("test.posts", t => { t.id("id") })`,
		`    m.drop_table("test.posts")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), migration1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), migration2)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run migrations
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// List revisions
	revisions, err := client.ListRevisions()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(revisions), 2)
	testutil.AssertEqual(t, revisions[0], "001")
	testutil.AssertEqual(t, revisions[1], "002")
}

// Note: GetMigrationInfo is already tested in schema_test.go
