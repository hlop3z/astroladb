//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// MigrationPlan Tests
// ===========================================================================

func TestClient_MigrationPlan(t *testing.T) {
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

	// Get migration plan before running any migrations
	plan, err := client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, plan)

	// Verify plan has migrations to run
	testutil.AssertTrue(t, len(plan.Migrations) > 0, "should have migrations in plan")

	// Run first migration
	err = client.MigrationRun(Steps(1))
	testutil.AssertNoError(t, err)

	// Get plan again - should have one less migration
	plan2, err := client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, plan2)
	testutil.AssertTrue(t, len(plan2.Migrations) < len(plan.Migrations), "should have fewer migrations after running one")
}

func TestClient_MigrationPlan_NoMigrations(t *testing.T) {
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

	// Get migration plan with no migrations
	plan, err := client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, plan)
	testutil.AssertEqual(t, len(plan.Migrations), 0)
}

func TestClient_MigrationPlan_AllMigrationsRun(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create migration
	migration := testutil.SimpleMigration(
		`    m.create_table("test.items", t => { t.id("id") })`,
		`    m.drop_table("test.items")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_items.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run all migrations
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Get plan - should be empty
	plan, err := client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, plan)
	testutil.AssertEqual(t, len(plan.Migrations), 0)
}
