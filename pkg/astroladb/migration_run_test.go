//go:build integration

package astroladb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestMigrationRun_Basic tests basic MigrationRun functionality.
func TestMigrationRun_Basic(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("no_migrations", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("single_migration", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create migration file
		migrationContent := `migration(m => {
  m.create_table("users", t => {
    t.id()
    t.string("email", 255).unique()
  })
})`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), migrationContent)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify table was created
		testutil.AssertTableExists(t, db, "users")

		// Verify migration was recorded
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 1)
		testutil.AssertEqual(t, status[0].Status, "applied")
	})

	t.Run("multiple_migrations_sequential", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create multiple migrations
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => {
  m.create_table("users", t => {
    t.id()
  })
})`)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => {
  m.create_table("posts", t => {
    t.id()
    t.integer("user_id").references("users")
  })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify both tables created
		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")

		// Verify both migrations recorded
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 2)
	})

	t.Run("already_applied", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => {
  m.create_table("users", t => { t.id() })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Run first time
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Run second time - should be no-op
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})
}

// TestMigrationRun_Options tests MigrationRun with various options.
func TestMigrationRun_Options(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("dry_run", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => {
  m.create_table("users", t => { t.id() })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Dry run should not apply
		err = client.MigrationRun(DryRun())
		testutil.AssertNoError(t, err)

		// Table should NOT exist
		testutil.AssertTableNotExists(t, db, "users")

		// No migrations should be recorded
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, status[0].Status, "pending")
	})

	t.Run("steps_limit", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create 3 migrations
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => { m.create_table("posts", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "003_create_comments.js"), `
migration(m => { m.create_table("comments", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply only 2 migrations
		err = client.MigrationRun(Steps(2))
		testutil.AssertNoError(t, err)

		// First two tables should exist
		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")

		// Third table should NOT exist
		testutil.AssertTableNotExists(t, db, "comments")

		// Verify status
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 3)
		testutil.AssertEqual(t, status[0].Status, "applied")
		testutil.AssertEqual(t, status[1].Status, "applied")
		testutil.AssertEqual(t, status[2].Status, "pending")
	})

	t.Run("target_revision", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => { m.create_table("posts", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "003_create_comments.js"), `
migration(m => { m.create_table("comments", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply up to revision 002
		err = client.MigrationRun(Target("002"))
		testutil.AssertNoError(t, err)

		// First two tables should exist
		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")
		testutil.AssertTableNotExists(t, db, "comments")
	})

	t.Run("combined_options", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Dry run with steps
		err = client.MigrationRun(DryRun(), Steps(1))
		testutil.AssertNoError(t, err)
	})
}

// TestMigrationRun_ChainVerification tests chain integrity checks before execution.
func TestMigrationRun_ChainVerification(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("valid_chain", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// First run - should succeed
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Add another migration
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => { m.create_table("posts", t => { t.id() }) })`)

		// Second run - should succeed (valid chain)
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("force_mode_skips_verification", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Force mode should work even with potential issues
		err = client.MigrationRun(Force())
		testutil.AssertNoError(t, err)
	})
}

// TestMigrationRun_ErrorHandling tests error scenarios.
func TestMigrationRun_ErrorHandling(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("invalid_migration_syntax", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Invalid JavaScript syntax
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_invalid.js"), `
migration(m => {
  INVALID SYNTAX HERE
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)
	})

	t.Run("sql_execution_error", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Migration with invalid SQL
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_invalid_sql.js"), `
migration(m => {
  m.sql("INVALID SQL STATEMENT")
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)
	})

	t.Run("partial_failure_rollback", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Migration that partially fails
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_partial_fail.js"), `
migration(m => {
  m.create_table("users", t => { t.id() })
  m.sql("INVALID SQL")  // This will fail
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)

		// Due to transaction, table should NOT exist
		testutil.AssertTableNotExists(t, db, "users")

		// Migration should NOT be recorded
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		if len(status) > 0 {
			testutil.AssertEqual(t, status[0].Status, "pending")
		}
	})
}

// TestMigrationRun_EmptyDirectory tests behavior with empty migrations directory.
func TestMigrationRun_EmptyDirectory(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("empty_migrations_dir", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err) // Should succeed with no-op
	})

	t.Run("nonexistent_migrations_dir", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir("/nonexistent/path/migrations"),
		)

		// Should fail during initialization or run
		if err == nil {
			err = client.MigrationRun()
			testutil.AssertNotNil(t, err)
			client.Close()
		}
	})
}

// TestMigrationRun_LockManagement tests lock acquisition and release.
func TestMigrationRun_LockManagement(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("lock_released_after_success", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Lock should be released
		ctx := context.Background()
		locked, err := client.runner.VersionManager().IsLocked(ctx)
		testutil.AssertNoError(t, err)
		testutil.AssertFalse(t, locked, "Lock should be released after migration")
	})

	t.Run("lock_released_after_error", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_fail.js"), `
migration(m => { m.sql("INVALID SQL") })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)

		// Lock should still be released
		ctx := context.Background()
		locked, err := client.runner.VersionManager().IsLocked(ctx)
		testutil.AssertNoError(t, err)
		testutil.AssertFalse(t, locked, "Lock should be released even after error")
	})
}

// TestMigrationStatus tests migration status retrieval.
func TestMigrationStatus(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("status_shows_applied_and_pending", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => { m.create_table("posts", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply first migration only
		err = client.MigrationRun(Steps(1))
		testutil.AssertNoError(t, err)

		// Check status
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 2)

		// First should be applied
		testutil.AssertEqual(t, status[0].Revision, "001")
		testutil.AssertEqual(t, status[0].Status, "applied")
		testutil.AssertNotNil(t, status[0].AppliedAt)

		// Second should be pending
		testutil.AssertEqual(t, status[1].Revision, "002")
		testutil.AssertEqual(t, status[1].Status, "pending")
		testutil.AssertNil(t, status[1].AppliedAt)
	})
}
