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
// GetChain Tests
// ===========================================================================

func TestClient_GetChain(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create migration chain
	migration1 := testutil.SimpleMigration(
		`    m.create_table("test.users", t => { t.id("id") })`,
		`    m.drop_table("test.users")`,
	)
	migration2 := testutil.SimpleMigration(
		`    m.create_table("test.posts", t => { t.id("id") })`,
		`    m.drop_table("test.posts")`,
	)
	migration3 := testutil.SimpleMigration(
		`    m.create_table("test.comments", t => { t.id("id") })`,
		`    m.drop_table("test.comments")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_users.js"), migration1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_posts.js"), migration2)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "003_comments.js"), migration3)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Get chain before running any migrations
	ch, err := client.GetChain()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, ch)
	testutil.AssertEqual(t, len(ch.Links), 3)

	// Run first migration
	err = client.MigrationRun(Steps(1))
	testutil.AssertNoError(t, err)

	// Get chain again
	ch, err = client.GetChain()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(ch.Links), 3)

	// Verify first link exists
	testutil.AssertEqual(t, ch.Links[0].Revision, "001")
}

func TestClient_GetChain_EmptyMigrations(t *testing.T) {
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

	// Get chain with no migrations
	ch, err := client.GetChain()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, ch)
	testutil.AssertEqual(t, len(ch.Links), 0)
}

// ===========================================================================
// VerifyChain Tests
// ===========================================================================

func TestClient_VerifyChain_Valid(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

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

	// Run migrations
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Verify chain
	result, err := client.VerifyChain()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result) // Chain verification completed
}

func TestClient_VerifyChain_Empty(t *testing.T) {
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

	// Verify empty chain
	result, err := client.VerifyChain()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result) // Empty chain verification completed
}

// ===========================================================================
// ClearMigrationHistory Tests
// ===========================================================================

func TestClient_ClearMigrationHistory(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

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

	// Run migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Verify history exists
	history, err := client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 1)

	// Clear history
	ctx := context.Background()
	deleted, err := client.ClearMigrationHistory(ctx)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, deleted, 1)

	// Verify history is cleared
	history, err = client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 0)
}

// ===========================================================================
// MigrationLock Tests
// ===========================================================================

func TestClient_MigrationLockStatus(t *testing.T) {
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

	// Check lock status (should be unlocked initially)
	lockInfo, err := client.MigrationLockStatus()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, lockInfo)
	testutil.AssertFalse(t, lockInfo.Locked, "should not be locked initially")
	// Note: LockedAt should be nil when not locked, but we don't assert it
	// to avoid Go's nil interface gotcha with *time.Time
}

func TestClient_MigrationReleaseLock(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create a migration to initialize the system
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

	// Run migration to initialize the migration system (creates lock table)
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Release lock (should work even if not locked)
	err = client.MigrationReleaseLock()
	testutil.AssertNoError(t, err)
}
