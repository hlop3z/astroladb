//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestWorkflow_FullMigrationLifecycle tests the complete migration workflow.
func TestWorkflow_FullMigrationLifecycle(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("check_generate_run_diff_verify", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Step 1: Define schema
		usersSchema := `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
  name: col.string(100),
  created_at: col.timestamp(),
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

		// Step 2: SchemaCheck - Validate schemas
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)

		// Step 3: MigrationGenerate - Create migration
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		// Step 4: MigrationRun - Apply to database
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify table exists
		testutil.AssertTableExists(t, db, "users")

		// Step 5: SchemaDiff - Verify no drift
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(ops), 0) // No differences

		// Step 6: VerifyChain - Validate integrity
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, result)
	})

	t.Run("multi_table_workflow", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Define multiple tables
		usersSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		postsSchema := `
export default table({
  id: col.id(),
  title: col.string(200),
  user_id: col.uuid().references("users"),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), usersSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), postsSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Generate and run
		_, err = client.MigrationGenerate("create_tables")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify both tables
		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")

		// No drift
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(ops), 0)
	})
}

// TestWorkflow_SchemaEvolution tests schema evolution scenarios.
func TestWorkflow_SchemaEvolution(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("add_columns_incrementally", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Initial schema
		schema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		schemaPath := filepath.Join(schemasDir, "public", "users.js")
		testutil.WriteFile(t, schemaPath, schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Create initial table
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Evolution 1: Add name column
		updatedSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
  name: col.string(100),
})
`
		testutil.WriteFile(t, schemaPath, updatedSchema)

		_, err = client.MigrationGenerate("add_name")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Evolution 2: Add timestamps
		finalSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
  name: col.string(100),
  created_at: col.timestamp(),
  updated_at: col.timestamp(),
})
`
		testutil.WriteFile(t, schemaPath, finalSchema)

		_, err = client.MigrationGenerate("add_timestamps")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify final state
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(ops), 0)

		// Check migration status
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 3) // Three migrations
	})

	t.Run("add_indexes_and_constraints", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Initial schema without constraints
		schema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		schemaPath := filepath.Join(schemasDir, "public", "users.js")
		testutil.WriteFile(t, schemaPath, schema)

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

		// Add unique constraint
		updatedSchema := `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
})
`
		testutil.WriteFile(t, schemaPath, updatedSchema)

		_, err = client.MigrationGenerate("add_unique")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify no drift
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(ops), 0)
	})

	t.Run("add_and_remove_tables", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Start with one table
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

		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Add second table
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), schema)

		_, err = client.MigrationGenerate("add_posts")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableExists(t, db, "posts")

		// Remove posts table (delete schema file)
		os.Remove(filepath.Join(schemasDir, "public", "posts.js"))

		_, err = client.MigrationGenerate("remove_posts")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")
		testutil.AssertTableNotExists(t, db, "posts")
	})
}

// TestWorkflow_RollbackReapply tests rollback and reapply cycles.
func TestWorkflow_RollbackReapply(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rollback_then_reapply", func(t *testing.T) {
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

		// Generate and apply
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")

		// Rollback
		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		testutil.AssertTableNotExists(t, db, "users")

		// Reapply
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "users")

		// Verify determinism
		result, err := client.VerifyChain()
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, result)
	})

	t.Run("multiple_rollback_reapply_cycles", func(t *testing.T) {
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
  value: col.integer(),
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

		_, err = client.MigrationGenerate("create_test")
		testutil.AssertNoError(t, err)

		// Cycle 1
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		// Cycle 2
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		err = client.MigrationRollback(1)
		testutil.AssertNoError(t, err)

		// Final apply
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "test")
	})
}

// TestWorkflow_DriftDetectionAndRepair tests drift detection and repair.
func TestWorkflow_DriftDetectionAndRepair(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("detect_manual_changes", func(t *testing.T) {
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

		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Manual change: add column directly to DB
		_, err = db.Exec("ALTER TABLE users ADD COLUMN extra VARCHAR(50)")
		testutil.AssertNoError(t, err)

		// Detect drift
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect drift")

		// Generate repair migration
		_, err = client.MigrationGenerate("repair_drift")
		testutil.AssertNoError(t, err)

		// Apply repair
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify drift is fixed
		ops, err = client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(ops), 0)
	})
}

// TestWorkflow_CrossNamespace tests multi-namespace workflows.
func TestWorkflow_CrossNamespace(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("multiple_namespaces", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755)
		os.MkdirAll(filepath.Join(schemasDir, "billing"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		authSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		billingSchema := `
export default table({
  id: col.id(),
  amount: col.decimal(10, 2),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "users.js"), authSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "billing", "invoices.js"), billingSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)

		_, err = client.MigrationGenerate("create_multi_namespace")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Verify both namespace tables exist
		testutil.AssertTableExists(t, db, "auth_users")
		testutil.AssertTableExists(t, db, "billing_invoices")

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(ops), 0)
	})
}

// TestWorkflow_ErrorRecovery tests error recovery scenarios.
func TestWorkflow_ErrorRecovery(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("recover_from_failed_migration", func(t *testing.T) {
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

		// Create valid migration
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Create failing migration manually
		failingMigration := `
migration(m => {
  m.sql("INVALID SQL SYNTAX")
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_failing.js"), failingMigration)

		// Try to run - should fail
		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)

		// Fix the migration
		fixedMigration := `
migration(m => {
  m.create_table("posts", t => {
    t.id()
  })
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "002_failing.js"), fixedMigration)

		// Run again with force to skip verification
		err = client.MigrationRun(Force())
		testutil.AssertNoError(t, err)

		testutil.AssertTableExists(t, db, "posts")
	})
}

// TestWorkflow_StatusTracking tests migration status throughout lifecycle.
func TestWorkflow_StatusTracking(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("status_evolution", func(t *testing.T) {
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

		// Generate migrations
		for i := 1; i <= 3; i++ {
			_, err = client.MigrationGenerate("migration")
			testutil.AssertNoError(t, err)
		}

		// All should be pending
		status, err := client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(status), 3)
		for _, s := range status {
			testutil.AssertEqual(t, s.Status, "pending")
		}

		// Apply first two
		err = client.MigrationRun(Steps(2))
		testutil.AssertNoError(t, err)

		status, err = client.MigrationStatus()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, status[0].Status, "applied")
		testutil.AssertEqual(t, status[1].Status, "applied")
		testutil.AssertEqual(t, status[2].Status, "pending")

		// Apply remaining
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		status, err = client.MigrationStatus()
		testutil.AssertNoError(t, err)
		for _, s := range status {
			testutil.AssertEqual(t, s.Status, "applied")
		}
	})
}
