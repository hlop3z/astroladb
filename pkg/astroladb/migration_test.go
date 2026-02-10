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
// MigrationRun Tests
// ===========================================================================

func TestMigrationRun_NoMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Should succeed with no migrations
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)
}

func TestMigrationRun_SingleMigration(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a migration file
	migration := testutil.SimpleMigration(
		`    m.create_table("test.users", t => {
      t.id()
      t.string("email", 255)
    })`,
		`    m.drop_table("test.users")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Run migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Verify table was created
	testutil.AssertTableExists(t, client.DB(), "test_users")
}

func TestMigrationRun_DryRun(t *testing.T) {
	testutil.Parallel(t)

	db, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration := testutil.SimpleMigration(
		`    m.create_table("test.items", t => { t.id() })`,
		`    m.drop_table("test.items")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_items.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Dry run - should not create table
	err = client.MigrationRun(DryRun())
	testutil.AssertNoError(t, err)

	// Verify table was NOT created
	testutil.AssertTableNotExists(t, db, "test_items")
}

func TestMigrationRun_Steps(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 3 migrations
	mig1 := testutil.SimpleMigration(
		`    m.create_table("test.table1", t => { t.id() })`,
		`    m.drop_table("test.table1")`,
	)
	mig2 := testutil.SimpleMigration(
		`    m.create_table("test.table2", t => { t.id() })`,
		`    m.drop_table("test.table2")`,
	)
	mig3 := testutil.SimpleMigration(
		`    m.create_table("test.table3", t => { t.id() })`,
		`    m.drop_table("test.table3")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_table1.js"), mig1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_table2.js"), mig2)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "003_table3.js"), mig3)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply only 2 migrations
	err = client.MigrationRun(Steps(2))
	testutil.AssertNoError(t, err)

	// Should have table1 and table2, but not table3
	testutil.AssertTableExists(t, client.DB(), "test_table1")
	testutil.AssertTableExists(t, client.DB(), "test_table2")
	testutil.AssertTableNotExists(t, client.DB(), "test_table3")
}

// ===========================================================================
// MigrationStatus Tests
// ===========================================================================

func TestMigrationStatus_NoMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	statuses, err := client.MigrationStatus()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(statuses), 0)
}

func TestMigrationStatus_PendingMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration := testutil.SimpleMigration(
		`    m.create_table("test.pending", t => { t.id() })`,
		`    m.drop_table("test.pending")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_pending.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	statuses, err := client.MigrationStatus()
	testutil.AssertNoError(t, err)

	testutil.AssertEqual(t, len(statuses), 1)
	testutil.AssertEqual(t, statuses[0].Revision, "001")
	testutil.AssertEqual(t, statuses[0].Name, "pending")
	testutil.AssertEqual(t, statuses[0].Status, "pending")
	// AppliedAt should be nil for pending migrations
	if statuses[0].AppliedAt != nil {
		t.Errorf("expected AppliedAt to be nil for pending migration, got %v", statuses[0].AppliedAt)
	}
}

func TestMigrationStatus_AppliedMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration := testutil.SimpleMigration(
		`    m.create_table("test.applied", t => { t.id() })`,
		`    m.drop_table("test.applied")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_applied.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Check status
	statuses, err := client.MigrationStatus()
	testutil.AssertNoError(t, err)

	testutil.AssertEqual(t, len(statuses), 1)
	testutil.AssertEqual(t, statuses[0].Status, "applied")
	testutil.AssertNotNil(t, statuses[0].AppliedAt)
}

// ===========================================================================
// MigrationHistory Tests
// ===========================================================================

func TestMigrationHistory_Empty(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(filepath.Join(tmpDir, "migrations")),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	history, err := client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 0)
}

func TestMigrationHistory_WithApplied(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create and apply migrations
	mig1 := testutil.SimpleMigration(
		`    m.create_table("test.hist1", t => { t.id() })`,
		`    m.drop_table("test.hist1")`,
	)
	mig2 := testutil.SimpleMigration(
		`    m.create_table("test.hist2", t => { t.id() })`,
		`    m.drop_table("test.hist2")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_hist1.js"), mig1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_hist2.js"), mig2)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply migrations
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Get history
	history, err := client.MigrationHistory()
	testutil.AssertNoError(t, err)

	testutil.AssertEqual(t, len(history), 2)

	// Verify first migration
	testutil.AssertEqual(t, history[0].Revision, "001")
	testutil.AssertEqual(t, history[0].Name, "hist1")
	testutil.AssertTrue(t, !history[0].AppliedAt.IsZero(), "AppliedAt should be set")

	// Verify second migration
	testutil.AssertEqual(t, history[1].Revision, "002")
	testutil.AssertEqual(t, history[1].Name, "hist2")
}

// ===========================================================================
// MigrationPlan Tests
// ===========================================================================

func TestMigrationPlan_NoPending(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	plan, err := client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, plan)
	testutil.AssertTrue(t, plan.IsEmpty(), "Plan should be empty")
}

func TestMigrationPlan_PendingMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create migrations
	mig1 := testutil.SimpleMigration(
		`    m.create_table("test.plan1", t => { t.id() })`,
		`    m.drop_table("test.plan1")`,
	)
	mig2 := testutil.SimpleMigration(
		`    m.create_table("test.plan2", t => { t.id() })`,
		`    m.drop_table("test.plan2")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_plan1.js"), mig1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_plan2.js"), mig2)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get plan - should show 2 pending migrations
	plan, err := client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, plan)
	testutil.AssertEqual(t, len(plan.Migrations), 2)

	// Apply first migration
	err = client.MigrationRun(Steps(1))
	testutil.AssertNoError(t, err)

	// Plan should now show only 1 pending
	plan, err = client.MigrationPlan()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(plan.Migrations), 1)
	testutil.AssertEqual(t, plan.Migrations[0].Revision, "002")
}

// ===========================================================================
// GetChain Tests
// ===========================================================================

func TestGetChain_NoMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	chain, err := client.GetChain()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, chain)
	testutil.AssertEqual(t, len(chain.Links), 0)
}

func TestGetChain_WithMigrations(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create migration chain
	mig1 := testutil.SimpleMigration(
		`    m.create_table("test.chain1", t => { t.id() })`,
		`    m.drop_table("test.chain1")`,
	)
	mig2 := testutil.SimpleMigration(
		`    m.create_table("test.chain2", t => { t.id() })`,
		`    m.drop_table("test.chain2")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_chain1.js"), mig1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_chain2.js"), mig2)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	chain, err := client.GetChain()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(chain.Links), 2)

	// Verify chain structure
	testutil.AssertEqual(t, chain.Links[0].Revision, "001")
	testutil.AssertEqual(t, chain.Links[1].Revision, "002")

	// Second link should reference first
	testutil.AssertEqual(t, chain.Links[1].PrevChecksum, chain.Links[0].Checksum)
}

// ===========================================================================
// VerifyChain Tests
// ===========================================================================

func TestVerifyChain_ValidChain(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	mig := testutil.SimpleMigration(
		`    m.create_table("test.verify", t => { t.id() })`,
		`    m.drop_table("test.verify")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_verify.js"), mig)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Verify chain - should be valid
	result, err := client.VerifyChain()
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
	testutil.AssertTrue(t, result.Valid, "Chain should be valid")
}

// ===========================================================================
// MigrationRollback Tests
// ===========================================================================

func TestMigrationRollback_Single(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration := testutil.SimpleMigration(
		`    m.create_table("test.rollback", t => { t.id() })`,
		`    m.drop_table("test.rollback")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_rollback.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)
	testutil.AssertTableExists(t, client.DB(), "test_rollback")

	// Rollback (1 migration)
	err = client.MigrationRollback(1)
	testutil.AssertNoError(t, err)

	// Table should be gone
	testutil.AssertTableNotExists(t, client.DB(), "test_rollback")
}

func TestMigrationRollback_DryRun(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration := testutil.SimpleMigration(
		`    m.create_table("test.dryback", t => { t.id() })`,
		`    m.drop_table("test.dryback")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_dryback.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply migration
	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Dry-run rollback - should not drop table
	err = client.MigrationRollback(1, DryRun())
	testutil.AssertNoError(t, err)

	// Table should still exist
	testutil.AssertTableExists(t, client.DB(), "test_dryback")
}

// ===========================================================================
// ClearMigrationHistory Tests
// ===========================================================================

func TestClearMigrationHistory(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration := testutil.SimpleMigration(
		`    m.create_table("test.clear", t => { t.id() })`,
		`    m.drop_table("test.clear")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_clear.js"), migration)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply migration
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

	// History should be empty
	history, err = client.MigrationHistory()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(history), 0)

	// But table should still exist
	testutil.AssertTableExists(t, client.DB(), "test_clear")
}

// ===========================================================================
// ListRevisions Tests (from schema_test.go, duplicated here for completeness)
// ===========================================================================

func TestListRevisions_WithMigrations(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create migrations
	mig1 := testutil.SimpleMigration(
		`    m.create_table("test.rev1", t => { t.id() })`,
		`    m.drop_table("test.rev1")`,
	)
	mig2 := testutil.SimpleMigration(
		`    m.create_table("test.rev2", t => { t.id() })`,
		`    m.drop_table("test.rev2")`,
	)

	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_rev1.js"), mig1)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_rev2.js"), mig2)

	client, err := New(
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	revisions, err := client.ListRevisions()
	testutil.AssertNoError(t, err)

	testutil.AssertEqual(t, len(revisions), 2)
	testutil.AssertEqual(t, revisions[0], "001")
	testutil.AssertEqual(t, revisions[1], "002")
}
