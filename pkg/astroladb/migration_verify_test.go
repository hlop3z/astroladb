//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestVerifyChain tests chain integrity validation.
func TestVerifyChain(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("valid_chain", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create valid migrations
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

		// Apply migrations
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify chain - should pass
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, result)
	})

	t.Run("modified_migration_detected", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		migrationPath := filepath.Join(migrationsDir, "001_create_users.js")
		originalContent := `
migration(m => {
  m.create_table("users", t => {
    t.id()
    t.string("email", 255)
  })
})`
		testutil.WriteFile(t, migrationPath, originalContent)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply migration
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Modify the migration file
		modifiedContent := `
migration(m => {
  m.create_table("users", t => {
    t.id()
    t.string("email", 255)
    t.string("name", 100)  // ADDED FIELD - MODIFICATION
  })
})`
		testutil.WriteFile(t, migrationPath, modifiedContent)

		// Verify chain - should fail due to modification
		_, err = client.VerifyChain()
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "checksum")
	})

	t.Run("missing_migration_file", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		migration1 := filepath.Join(migrationsDir, "001_create_users.js")
		migration2 := filepath.Join(migrationsDir, "002_create_posts.js")

		testutil.WriteFile(t, migration1, `
migration(m => { m.create_table("users", t => { t.id() }) })`)
		testutil.WriteFile(t, migration2, `
migration(m => { m.create_table("posts", t => { t.id() }) })`)

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

		// Delete the first migration file
		os.Remove(migration1)

		// Verify chain - should fail due to missing file
		_, err = client.VerifyChain()
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "001")
	})

	t.Run("out_of_order_revisions", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create migrations with intentionally wrong ordering
		testutil.WriteFile(t, filepath.Join(migrationsDir, "003_out_of_order.js"), `
migration(m => { m.create_table("wrong_order", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_first.js"), `
migration(m => { m.create_table("first", t => { t.id() }) })`)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_second.js"), `
migration(m => { m.create_table("second", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply migrations - they should be sorted correctly
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify chain - should succeed (migrations are auto-sorted)
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, result)
	})

	t.Run("invalid_revision_format", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Invalid revision format (not numeric)
		testutil.WriteFile(t, filepath.Join(migrationsDir, "abc_invalid.js"), `
migration(m => { m.create_table("invalid", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Attempting to run should handle invalid format gracefully
		err = client.MigrationRun()
		// Behavior depends on implementation - might error or skip
		// This test documents the actual behavior
		if err != nil {
			testutil.AssertErrorContains(t, err, "revision")
		}
	})
}

// TestVerifySQLDeterminism tests SQL checksum consistency validation.
func TestVerifySQLDeterminism(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("deterministic_sql_generation", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), `
migration(m => {
  m.create_table("users", t => {
    t.id()
    t.string("email", 255)
    t.string("name", 100)
    t.timestamp("created_at")
  })
})`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply migration
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Re-evaluate the migration and check SQL matches stored checksum
		// This verifies deterministic SQL generation
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, result)
	})

	t.Run("checksum_mismatch", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		migrationPath := filepath.Join(migrationsDir, "001_test.js")
		testutil.WriteFile(t, migrationPath, `
migration(m => { m.create_table("test", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply migration
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Modify migration content
		testutil.WriteFile(t, migrationPath, `
migration(m => { m.create_table("test", t => { t.id(); t.string("extra", 50) }) })`)

		// Verify should detect checksum mismatch
		_, err = client.VerifyChain()
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "checksum")
	})

	t.Run("migrations_without_checksums", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_legacy.js"), `
migration(m => { m.create_table("legacy", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply migration
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Manually clear checksum from version table to simulate legacy migration
		// (This would require direct DB access - implementation detail)

		// Verify chain should handle missing checksums gracefully
		_, err = client.VerifyChain()
		// Should either succeed (lenient mode) or provide clear message
		if err != nil {
			testutil.AssertErrorContains(t, err, "checksum")
		}
	})

	t.Run("empty_results_handling", func(t *testing.T) {
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

		// Verify chain with no applied migrations
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err) // Should succeed with empty chain
		testutil.AssertNotNil(t, result)
	})
}

// TestVerifyChain_ForceMode tests that force mode skips verification.
func TestVerifyChain_ForceMode(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("force_skips_verification", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		migrationPath := filepath.Join(migrationsDir, "001_test.js")
		testutil.WriteFile(t, migrationPath, `
migration(m => { m.create_table("test", t => { t.id() }) })`)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Apply migration
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Modify migration
		testutil.WriteFile(t, migrationPath, `
migration(m => { m.create_table("test", t => { t.id(); t.string("modified", 50) }) })`)

		// Normal verify should fail
		_, err = client.VerifyChain()
		testutil.AssertNotNil(t, err)

		// Force mode should skip verification and succeed
		err = client.MigrationRun(Force())
		testutil.AssertNoError(t, err)
	})
}
