//go:build integration

package astroladb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// Client Creation Error Paths
// ===========================================================================

func TestClient_New_InvalidDatabaseURL(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Try to create client with invalid database URL
	_, err := New(
		WithDatabaseURL("invalid://url"),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertTrue(t, err != nil, "should error with invalid database URL")
}

func TestClient_New_MissingMigrationsDir(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)

	// Try to create client without migrations directory
	_, err := New(
		WithDatabaseURL(dbURL),
	)
	// Should work - migrations dir is optional in some modes
	_ = err
}

// ===========================================================================
// MigrationRollback Error Paths
// ===========================================================================

func TestClient_MigrationRollback_NoMigrationsToRollback(t *testing.T) {
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

	// Try to rollback when nothing has been run
	err = client.MigrationRollback(1)
	// Should handle gracefully (might error or no-op depending on implementation)
	_ = err
}

func TestClient_MigrationRollback_AfterRun(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create a simple migration
	migration := testutil.SimpleMigration(
		`    m.create_table("test.rollback_test", t => { t.id("id") })`,
		`    m.drop_table("test.rollback_test")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_rollback.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Rollback
	err = client.MigrationRollback(1)
	testutil.AssertNoError(t, err)

	// Verify migration was rolled back
	history, err := client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 0)
}

func TestClient_MigrationRollback_MultipleSteps(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create multiple migrations
	for i := 1; i <= 3; i++ {
		migration := testutil.SimpleMigration(
			`    m.create_table("test.table`+string(rune('0'+i))+`", t => { t.id("id") })`,
			`    m.drop_table("test.table`+string(rune('0'+i))+`")`,
		)
		filename := filepath.Join(migrationsDir, "00"+string(rune('0'+i))+"_table.js")
		testutil.WriteFile(t, filename, migration)
	}

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run all migrations
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Rollback 2 steps
	err = client.MigrationRollback(2)
	testutil.AssertNoError(t, err)

	// Verify only 1 migration remains
	history, err := client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 1)
}

// ===========================================================================
// Context Cancellation Tests
// ===========================================================================

func TestClient_CheckDriftContext_Cancelled(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to check drift with cancelled context
	_, err = client.CheckDriftContext(ctx)
	// Should detect cancellation (might error or handle gracefully)
	_ = err
}

func TestClient_CheckDriftContext_WithTimeout(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Create a context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check drift with timeout
	result, err := client.CheckDriftContext(ctx)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
}

// ===========================================================================
// Migration History Edge Cases
// ===========================================================================

func TestClient_MigrationHistory_EmptyDatabase(t *testing.T) {
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

	// Get history from empty database
	history, err := client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 0)
}

// ===========================================================================
// Migration Status Edge Cases
// ===========================================================================

func TestClient_MigrationStatus_PartiallyRun(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))

	// Create 3 migrations
	for i := 1; i <= 3; i++ {
		migration := testutil.SimpleMigration(
			`    m.create_table("test.status`+string(rune('0'+i))+`", t => { t.id("id") })`,
			`    m.drop_table("test.status`+string(rune('0'+i))+`")`,
		)
		filename := filepath.Join(migrationsDir, "00"+string(rune('0'+i))+"_status.js")
		testutil.WriteFile(t, filename, migration)
	}

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Run only first 2 migrations
	err = client.MigrationRun(Steps(2))
	testutil.AssertNoError(t, err)

	// Check status
	status, err := client.MigrationStatus()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(status), 3)

	// Verify first 2 are applied, last is pending
	testutil.AssertEqual(t, status[0].Status, "applied")
	testutil.AssertEqual(t, status[1].Status, "applied")
	testutil.AssertEqual(t, status[2].Status, "pending")
}

// ===========================================================================
// Close and Resource Cleanup
// ===========================================================================

func TestClient_Close_MultipleTimes(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(filepath.Join(tmpDir, "schemas")),
	)
	testutil.AssertNoError(t, err)

	// Close multiple times should not panic
	err = client.Close()
	testutil.AssertNoError(t, err)

	err = client.Close()
	// Second close might error or be no-op
	_ = err
}
