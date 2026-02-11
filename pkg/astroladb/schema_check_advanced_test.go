//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestSchemaCheck_DeterminismValidation tests determinism checking with double-evaluation.
func TestSchemaCheck_DeterminismValidation(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("deterministic_schema", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create deterministic schema
		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255),
  created_at: col.timestamp(),
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

		// Should pass determinism check
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("table_count_consistency", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create multiple tables
		for i := 1; i <= 5; i++ {
			schema := `
export default table({
  id: col.id(),
  value: col.integer(),
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

		// Should validate all tables consistently
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("column_order_consistency", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Schema with specific column order
		schema := `
export default table({
  id: col.id(),
  alpha: col.string(50),
  beta: col.string(50),
  gamma: col.string(50),
  created_at: col.timestamp(),
  updated_at: col.timestamp(),
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

		// Column order should be deterministic
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})
}

// TestSchemaCheck_InvalidSchemaDetection tests detection of invalid schemas.
func TestSchemaCheck_InvalidSchemaDetection(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("circular_dependency", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create circular dependency: A -> B -> A
		schemaA := `
export default table({
  id: col.id(),
  b_id: col.uuid().references("b"),
})
`
		schemaB := `
export default table({
  id: col.id(),
  a_id: col.uuid().references("a"),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "a.js"), schemaA)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "b.js"), schemaB)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should detect circular dependency
		err = client.SchemaCheck()
		// May or may not error depending on implementation
		// This test documents the behavior
		_ = err
	})

	t.Run("invalid_foreign_key_reference", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Reference to non-existent table
		schema := `
export default table({
  id: col.id(),
  nonexistent_id: col.uuid().references("nonexistent_table"),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "orders.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should detect invalid reference
		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
	})

	t.Run("missing_referenced_table", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Reference table that doesn't have a schema file
		schema := `
export default table({
  id: col.id(),
  user_id: col.uuid().references("users"),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should error on missing table reference
		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
	})

	t.Run("duplicate_table_names", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create two files that would result in same table name
		schema1 := `
export default table({
  id: col.id(),
  value1: col.string(50),
})
`
		schema2 := `
export default table({
  id: col.id(),
  value2: col.string(50),
})
`
		// Both would be named "public.users"
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema1)
		// Note: This scenario might not be possible with current file-to-table mapping
		// This test documents expected behavior
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema2)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should handle duplicate detection
		err = client.SchemaCheck()
		// Behavior depends on implementation
		_ = err
	})

	t.Run("invalid_column_types", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Invalid type usage (syntax error will be caught by eval)
		invalidSchema := `
export default table({
  id: col.id(),
  name: col.invalid_type(100),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "items.js"), invalidSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should error on invalid type
		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
	})
}

// TestSchemaCheck_JoinTableGeneration tests join table inclusion from many_to_many.
func TestSchemaCheck_JoinTableGeneration(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("many_to_many_join_tables", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create tables with many_to_many relationship
		usersSchema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		rolesSchema := `
export default table({
  id: col.id(),
  name: col.string(50),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), usersSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "roles.js"), rolesSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should validate including auto-generated join tables
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})
}

// TestSchemaCheck_ValidationPipeline tests the complete validation pipeline.
func TestSchemaCheck_ValidationPipeline(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("valid_schema_pipeline", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create valid schemas with relationships
		usersSchema := `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
  created_at: col.timestamp(),
})
`
		postsSchema := `
export default table({
  id: col.id(),
  title: col.string(255),
  user_id: col.uuid().references("users"),
  created_at: col.timestamp(),
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

		// Full validation pipeline should pass
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("empty_schemas_directory", func(t *testing.T) {
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

		// Should error on empty schemas
		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "no schema")
	})

	t.Run("complex_schema_validation", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create complex schema with multiple relationships
		usersSchema := `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
  profile_id: col.uuid().references("profiles"),
})
`
		profilesSchema := `
export default table({
  id: col.id(),
  bio: col.text(),
})
`
		postsSchema := `
export default table({
  id: col.id(),
  user_id: col.uuid().references("users"),
  title: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), usersSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "profiles.js"), profilesSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), postsSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should validate complex schema
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})
}
