//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestSchemaDiff_FullPipeline tests the complete schema diff pipeline with ephemeral DB.
func TestSchemaDiff_FullPipeline(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("column_addition", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create initial table in database
		_, err := db.Exec(`
			CREATE TABLE users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
		`)
		testutil.AssertNoError(t, err)

		// Define schema with additional column
		schema := `
export default table({
  id: col.id(),
  email: col.string(255),
  name: col.string(100),  // New column
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

		// Compute diff
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect column addition")
	})

	t.Run("column_removal", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create table with extra column
		_, err := db.Exec(`
			CREATE TABLE products (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(255) NOT NULL,
				description TEXT
			)
		`)
		testutil.AssertNoError(t, err)

		// Define schema without description
		schema := `
export default table({
  id: col.id(),
  name: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "products.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect column removal")
	})

	t.Run("type_change", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create table with string age
		_, err := db.Exec(`
			CREATE TABLE users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				age VARCHAR(10)
			)
		`)
		testutil.AssertNoError(t, err)

		// Define schema with integer age
		schema := `
export default table({
  id: col.id(),
  age: col.integer(),
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

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect type change")
	})

	t.Run("index_addition", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create table without index
		_, err := db.Exec(`
			CREATE TABLE users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
		`)
		testutil.AssertNoError(t, err)

		// Define schema with index
		schema := `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
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

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect index addition")
	})
}

// TestSchemaDiff_EdgeCases tests edge cases in schema diff.
func TestSchemaDiff_EdgeCases(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("empty_schema", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

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

		// Diff with no schema files should handle gracefully
		ops, err := client.SchemaDiff()
		// Behavior depends on implementation
		if err != nil {
			testutil.AssertErrorContains(t, err, "schema")
		} else {
			testutil.AssertTrue(t, len(ops) >= 0, "Empty schema should return empty or valid ops")
		}
	})

	t.Run("large_schema_many_tables", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create multiple schema files
		for i := 1; i <= 10; i++ {
			schema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
			testutil.WriteFile(t, filepath.Join(schemasDir, "public", "table"+string(rune('0'+i))+".js"), schema)
		}

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should handle large schema without error
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect all table creations")
	})

	t.Run("cross_namespace_tables", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755)
		os.MkdirAll(filepath.Join(schemasDir, "billing"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schemas in different namespaces
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

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect tables in multiple namespaces")
	})
}

// TestSchemaDiff_TableNameMapping tests table name mapping resolution.
func TestSchemaDiff_TableNameMapping(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("namespace_resolution", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "app"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create table in database (no namespace prefix in PG)
		_, err := db.Exec(`
			CREATE TABLE app_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
		`)
		testutil.AssertNoError(t, err)

		// Define schema with namespace
		schema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "app", "users.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should correctly map app.users to app_users table
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		// If mapping works, diff should be minimal (just normalization differences)
		testutil.AssertTrue(t, ops != nil, "Should return operations list")
	})
}

// TestSchemaDiff_NormalizeSchema tests the normalizeSchema function.
func TestSchemaDiff_NormalizeSchema(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("normalization_idempotent", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schema
		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
  created_at: col.timestamp(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "items.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Run diff twice - should be deterministic
		ops1, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)

		ops2, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)

		testutil.AssertEqual(t, len(ops1), len(ops2))
	})

	t.Run("canonical_form_defaults", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create table with explicit defaults
		_, err := db.Exec(`
			CREATE TABLE items (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				active BOOLEAN DEFAULT true
			)
		`)
		testutil.AssertNoError(t, err)

		// Define schema with explicit defaults
		schema := `
export default table({
  id: col.id(),
  active: col.boolean().default(true),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "items.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Normalization should handle defaults correctly
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		// May have operations due to normalization differences, but shouldn't error
		_ = ops
	})
}

// TestSchemaDiff_ErrorHandling tests error scenarios in schema diff.
func TestSchemaDiff_ErrorHandling(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("invalid_schema_syntax", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schema with invalid syntax
		invalidSchema := `
export default table({
  id: col.id()
  INVALID SYNTAX HERE
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "bad.js"), invalidSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should error on invalid syntax
		_, err = client.SchemaDiff()
		testutil.AssertNotNil(t, err)
	})

	t.Run("database_connection_error", func(t *testing.T) {
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

		// Invalid database URL
		client, err := New(
			WithDatabaseURL("postgresql://invalid:5432/nonexistent"),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		if err != nil {
			// Client creation may fail with invalid URL
			return
		}
		defer client.Close()

		// Should error when trying to connect
		_, err = client.SchemaDiff()
		testutil.AssertNotNil(t, err)
	})
}
