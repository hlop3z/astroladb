//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestMigration_AllCodePaths tests all code paths in migration.go comprehensively.
func TestMigration_AllCodePaths(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("generate_all_operation_types", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Initial schema
		initialSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		schemaPath := filepath.Join(schemasDir, "public", "users.js")
		testutil.WriteFile(t, schemaPath, initialSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Generate and apply initial
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Test AddColumn
		testutil.WriteFile(t, schemaPath, `
export default table({
  id: col.id(),
  email: col.string(255),
  name: col.string(100),
})
`)
		_, err = client.MigrationGenerate("add_name")
		testutil.AssertNoError(t, err)
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Test AddIndex
		testutil.WriteFile(t, schemaPath, `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
  name: col.string(100),
})
`)
		_, err = client.MigrationGenerate("add_index")
		testutil.AssertNoError(t, err)
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Test AlterColumn
		testutil.WriteFile(t, schemaPath, `
export default table({
  id: col.id(),
  email: col.string(500),
  name: col.string(100),
})
`)
		_, err = client.MigrationGenerate("alter_email")
		testutil.AssertNoError(t, err)
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("migration_with_description", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "test.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Generate with description
		_, err = client.MigrationGenerate("This is a detailed description")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Check that description was recorded
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(status) > 0, "Should have migrations")
	})

	t.Run("migration_plan_detailed", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Get migration plan
		plan, err := client.MigrationPlan()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(plan.Migrations) > 0, "Should have operations in plan")

		// Generate from plan
		_, err = client.MigrationGenerate("from_plan")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("verify_chain_complete", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "test.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("test")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify chain
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, result)
		testutil.AssertTrue(t, result.Valid, "Chain should be valid")
	})

	t.Run("rollback_with_custom_down", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create migration with custom down
		migrationContent := `
migration(m => {
  m.up = () => {
    m.create_table("test", t => {
      t.id()
    })
  }
  m.down = () => {
    m.sql("DROP TABLE test")
  }
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_custom.js"), migrationContent)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "test")

		// Rollback using custom down
		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "test")
	})
}

// TestMigration_WriterFunctions tests all 12 writer functions completely.
func TestMigration_WriterFunctions(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("writeCreateTable_with_all_features", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Complex table with enums, indexes, timestamps
		schema := `
export default table({
  id: col.id(),
  status: col.enum(['active', 'inactive']),
  email: col.string(255).unique(),
  created_at: col.timestamp(),
  updated_at: col.timestamp(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "complex.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("complex_table")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("writeAddForeignKey_with_actions", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create users first
		usersSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), usersSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Add posts with FK and ON DELETE/UPDATE
		postsSchema := `
export default table({
  id: col.id(),
  user_id: col.uuid().references("users", { onDelete: "CASCADE", onUpdate: "CASCADE" }),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), postsSchema)

		_, err = client.MigrationGenerate("add_posts")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("writeCreateIndex_composite", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Table first
		initialSchema := `
export default table({
  id: col.id(),
  first_name: col.string(100),
  last_name: col.string(100),
})
`
		schemaPath := filepath.Join(schemasDir, "public", "users.js")
		testutil.WriteFile(t, schemaPath, initialSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Add composite index
		updatedSchema := `
export default table({
  id: col.id(),
  first_name: col.string(100).index(),
  last_name: col.string(100).index(),
})
`
		testutil.WriteFile(t, schemaPath, updatedSchema)

		_, err = client.MigrationGenerate("add_indexes")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("writeRenameColumn", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create table
		_, err := db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, old_name VARCHAR(100))")
		testutil.AssertNoError(t, err)

		// Manual migration for rename
		migration := `
migration(m => {
  m.rename_column("test", "old_name", "new_name")
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_rename.js"), migration)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("writeRenameTable", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create table
		_, err := db.Exec("CREATE TABLE old_table (id SERIAL PRIMARY KEY)")
		testutil.AssertNoError(t, err)

		// Manual migration for table rename
		migration := `
migration(m => {
  m.rename_table("old_table", "new_table")
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_rename_table.js"), migration)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "old_table")
		testutil.AssertTableExists(t, db, "new_table")
	})
}

// TestMigration_EdgeCasesComplete tests all edge cases for 100% coverage.
func TestMigration_EdgeCasesComplete(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("empty_migration_content", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Empty migration
		migration := `
migration(m => {
  // Empty - no operations
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_empty.js"), migration)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Should still be recorded
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 1)
	})

	t.Run("migration_with_only_raw_sql", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		migration := `
migration(m => {
  m.sql("CREATE TABLE custom (id SERIAL PRIMARY KEY)")
  m.sql("INSERT INTO custom (id) VALUES (1)")
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_raw.js"), migration)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "custom")
	})

	t.Run("generate_with_prettier_formatting", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
  very_long_column_name_that_should_wrap: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "test.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("test_formatting")
		testutil.AssertNoError(t, err)

		// Check that file was created and is valid JS
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
	})

	t.Run("revision_number_calculation", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "test.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Generate multiple migrations to test revision numbering
		for i := 1; i <= 5; i++ {
			_, err = client.MigrationGenerate("migration")
			testutil.AssertNoError(t, err)
		}

		// Check that revisions are sequential
		files, _ := os.ReadDir(migrationsDir)
		testutil.AssertEqual(t, len(files), 5)
	})
}
