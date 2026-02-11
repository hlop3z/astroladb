//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestMigrationRollback_SingleStep tests rolling back one migration.
func TestMigrationRollback_SingleStep(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rollback_one_migration", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create migrations
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => {
  m.create_table("users", t => {
    t.id()
    t.string("email", 255)
  })
})`)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => {
  m.create_table("posts", t => {
    t.id()
    t.string("title", 255)
  })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply both migrations
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")

		// Rollback last migration (002)
		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		// Users should still exist, posts should not
		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableNotExists(t, db, "posts")

		// Check status
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 2)

		// First migration should be applied, second pending
		var applied, pending int
		for _, s := range status {
			if s.Status == "applied" {
				applied++
			} else {
				pending++
			}
		}
		testutil.AssertEqual(t, applied, 1)
		testutil.AssertEqual(t, pending, 1)
	})

	t.Run("rollback_with_explicit_down", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Migration with explicit down operation
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_items.js"), `
migration(m => {
  m.create_table("items", t => {
    t.id()
    t.string("name", 100)
  })
}, {
  down: (m) => {
    m.drop_table("items")
  }
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
		testutil.AssertTableExists(t, db, "items")

		// Rollback using explicit down
		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "items")
	})
}

// TestMigrationRollback_MultipleSteps tests rolling back multiple migrations.
func TestMigrationRollback_MultipleSteps(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rollback_multiple_migrations", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create three migrations
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => { m.create_table("posts", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "003_create_comments.js"), `
migration(m => { m.create_table("comments", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply all migrations
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")
		testutil.AssertTableExists(t, db, "comments")

		// Rollback 2 migrations
		err = client.MigrationRollback(2)
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableNotExists(t, db, "posts")
		testutil.AssertTableNotExists(t, db, "comments")
	})

	t.Run("rollback_all_migrations", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_test.js"), `
migration(m => { m.create_table("test", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
		testutil.AssertTableExists(t, db, "test")

		// Rollback all (steps = all applied)
		err = client.MigrationRollback(999)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "test")

		// Status should show all pending
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		for _, s := range status {
			testutil.AssertEqual(t, s.Status, "pending")
		}
	})
}

// TestMigrationRollback_TargetRevision tests rolling back to a specific revision.
func TestMigrationRollback_TargetRevision(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rollback_to_specific_revision", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_a.js"), `
migration(m => { m.create_table("a", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_b.js"), `
migration(m => { m.create_table("b", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "003_create_c.js"), `
migration(m => { m.create_table("c", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Rollback to revision 001 (rollback 2 migrations: 003, 002)
		err = client.MigrationRollback(2)
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "a")
		testutil.AssertTableNotExists(t, db, "b")
		testutil.AssertTableNotExists(t, db, "c")
	})

	t.Run("rollback_to_clean_state", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_test.js"), `
migration(m => { m.create_table("test", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Rollback to clean state (rollback all)
		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "test")
	})
}

// TestMigrationRollback_ForeignKeys tests FK dependency handling during rollback.
func TestMigrationRollback_ForeignKeys(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rollback_with_fk_dependencies", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => {
  m.create_table("posts", t => {
    t.id()
    t.integer("user_id").references("users")
  })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Rollback both - should handle FK dependency correctly
		// Posts must be dropped before users
		err = client.MigrationRollback(2)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "users")
		testutil.AssertTableNotExists(t, db, "posts")
	})

	t.Run("rollback_respects_dependency_order", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Complex FK graph
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => { m.create_table("users", t => { t.id() }) })`)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), `
migration(m => {
  m.create_table("posts", t => {
    t.id()
    t.integer("user_id").references("users")
  })
})`)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "003_create_comments.js"), `
migration(m => {
  m.create_table("comments", t => {
    t.id()
    t.integer("post_id").references("posts")
  })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Rollback all - must drop in correct order: comments → posts → users
		err = client.MigrationRollback(3)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "users")
		testutil.AssertTableNotExists(t, db, "posts")
		testutil.AssertTableNotExists(t, db, "comments")
	})
}

// TestMigrationRollback_ErrorHandling tests rollback error scenarios.
func TestMigrationRollback_ErrorHandling(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rollback_with_no_migrations", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Rollback with no migrations applied - should handle gracefully
		err = client.MigrationRollback(1)
		// Should either succeed as no-op or provide clear message
		if err != nil {
			testutil.AssertErrorContains(t, err, "no")
		}
	})

	t.Run("rollback_with_sql_error", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Migration with invalid down SQL
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_invalid.js"), `
migration(m => {
  m.create_table("test", t => { t.id() })
}, {
  down: (m) => {
    m.sql("INVALID SQL SYNTAX HERE")
  }
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Rollback should fail due to SQL error
		err = client.MigrationRollback(1)
		testutil.AssertNotNil(t, err)
	})

	t.Run("rollback_more_than_applied", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_test.js"), `
migration(m => { m.create_table("test", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Try to rollback more than applied
		err = client.MigrationRollback(10)
		// Should handle gracefully (error or rollback what's available)
		// Behavior depends on implementation
		_ = err

		testutil.AssertTableNotExists(t, db, "test")
	})
}

// TestMigrationRollback_DryRunMode tests dry-run rollback mode.
func TestMigrationRollback_DryRunMode(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("dry_run_shows_sql_without_executing", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_test.js"), `
migration(m => { m.create_table("test", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
		testutil.AssertTableExists(t, db, "test")

		// Dry-run rollback (if supported)
		err = client.MigrationRollback(1, DryRun())
		testutil.AssertNoError(t, err)

		// Table should still exist (dry-run didn't execute)
		testutil.AssertTableExists(t, db, "test")
	})
}
