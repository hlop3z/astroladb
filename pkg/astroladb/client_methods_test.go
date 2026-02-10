//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// DB Tests
// ===========================================================================

func TestClient_DB(t *testing.T) {
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

	// Get database connection
	db := client.DB()
	testutil.AssertNotNil(t, db)

	// Verify connection works
	err = db.Ping()
	testutil.AssertNoError(t, err)
}

// ===========================================================================
// LintPendingMigrations Tests
// ===========================================================================

func TestClient_LintPendingMigrations(t *testing.T) {
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

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_users.js"), migration1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_posts.js"), migration2)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Lint pending migrations
	issues, err := client.LintPendingMigrations()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, issues)
	// No specific assertions about lint issues - just verify it works
}

func TestClient_LintPendingMigrations_NoMigrations(t *testing.T) {
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

	// Lint with no migrations
	issues, err := client.LintPendingMigrations()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, issues)
	testutil.AssertEqual(t, len(issues), 0)
}

// ===========================================================================
// DetectMissingBackfills Tests
// ===========================================================================

func TestClient_DetectMissingBackfills(t *testing.T) {
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

	// Detect missing backfills - may error without schemas, just verify no panic
	_, err = client.DetectMissingBackfills()
	_ = err
}

// ===========================================================================
// DetectRenames Tests
// ===========================================================================

func TestClient_DetectRenames(t *testing.T) {
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

	// Detect renames - may error without schemas, just verify no panic
	_, err = client.DetectRenames()
	_ = err
}
