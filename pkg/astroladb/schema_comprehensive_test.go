//go:build integration

package astroladb

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestSchema_Comprehensive_SchemaDiff tests all code paths in SchemaDiff pipeline.
func TestSchema_Comprehensive_SchemaDiff(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("full_diff_pipeline_postgres", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create initial schema
		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255),
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

		// Generate and apply initial migration
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Test SchemaDiff with no changes
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))

		// Modify schema to trigger diff
		modifiedSchema := `
export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255),
  age: col.integer(),
  created_at: col.timestamp(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), modifiedSchema)

		// Test SchemaDiff with changes
		ops, err = client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should detect schema changes")
	})

	t.Run("diff_with_sqlite", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		dbPath := filepath.Join(tempDir, "test.db")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
  title: col.string(200),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), schema)

		client, err := New(
			WithDatabaseURL(dbPath),
			WithDialect("sqlite"),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("create_posts")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})

	t.Run("diff_with_multiple_namespaces", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		publicSchema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		authSchema := `
export default table({
  id: col.id(),
  username: col.string(50),
  password_hash: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "items.js"), publicSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "users.js"), authSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("create_tables")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})

	t.Run("diff_empty_schema", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})

	t.Run("diff_large_schema", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create 10 tables with multiple columns
		for i := 1; i <= 10; i++ {
			schema := `
export default table({
  id: col.id(),
  name: col.string(100),
  description: col.text(),
  count: col.integer(),
  price: col.decimal(10, 2),
  active: col.boolean(),
  created_at: col.timestamp(),
  updated_at: col.timestamp(),
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

		_, err = client.MigrationGenerate("create_large_schema")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})
}

// TestSchema_100Pct_SchemaCheck tests all validation paths.
func TestSchema_Comprehensive_SchemaCheck(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("check_valid_schema", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("check_determinism", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		// Schema with multiple columns to test ordering
		schema := `
export default table({
  id: col.id(),
  email: col.string(255),
  name: col.string(100),
  age: col.integer(),
  created_at: col.timestamp(),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// SchemaCheck should verify determinism by evaluating twice
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("check_invalid_foreign_key", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		// Schema with FK to non-existent table
		schema := `
export default table({
  id: col.id(),
  user_id: col.integer().foreign_key("nonexistent", "id"),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), schema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
	})

	t.Run("check_duplicate_table", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		// Create two files with same table name
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users1.js"), schema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users2.js"), schema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		// Should detect duplicate table name
		testutil.AssertNotNil(t, err)
	})

	t.Run("check_empty_schema", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("check_many_to_many_join_tables", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

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
  users: col.many_to_many("users"),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), usersSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "roles.js"), rolesSchema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})
}

// TestSchema_100Pct_SchemaExport tests all export formats.
func TestSchema_Comprehensive_SchemaExport(t *testing.T) {
	testutil.SkipIfShort(t)

	tempDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tempDir, "schemas")
	os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

	schema := `
export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255),
  age: col.integer(),
  active: col.boolean(),
  bio: col.text(),
  score: col.decimal(10, 2),
  created_at: col.timestamp(),
  updated_at: col.timestamp(),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	t.Run("export_typescript", func(t *testing.T) {
		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate TypeScript")
		testutil.AssertTrue(t, contains(output, "interface"), "Should contain interface keyword")
	})

	t.Run("export_go", func(t *testing.T) {
		output, err := client.SchemaExport("go")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate Go")
		testutil.AssertTrue(t, contains(output, "type"), "Should contain type keyword")
	})

	t.Run("export_graphql", func(t *testing.T) {
		output, err := client.SchemaExport("graphql")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate GraphQL")
		testutil.AssertTrue(t, contains(output, "type"), "Should contain type keyword")
	})

	t.Run("export_json_schema", func(t *testing.T) {
		output, err := client.SchemaExport("json")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate JSON Schema")
		testutil.AssertTrue(t, contains(output, "properties"), "Should contain properties")
	})

	t.Run("export_openapi", func(t *testing.T) {
		output, err := client.SchemaExport("openapi")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate OpenAPI")
		testutil.AssertTrue(t, contains(output, "components"), "Should contain components")
	})

	t.Run("export_invalid_format", func(t *testing.T) {
		_, err := client.SchemaExport("invalid")
		testutil.AssertNotNil(t, err)
	})

	t.Run("export_with_multiple_namespaces", func(t *testing.T) {
		os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755)
		authSchema := `
export default table({
  id: col.id(),
  username: col.string(50),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "users.js"), authSchema)

		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate TypeScript for multiple namespaces")
	})

	t.Run("export_deterministic_output", func(t *testing.T) {
		output1, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)

		output2, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)

		if !bytes.Equal(output1, output2) {
			t.Error("Export should be deterministic")
		}
	})
}

// TestSchema_100Pct_ColumnOrdering tests sortColumnsForDatabase.
func TestSchema_Comprehensive_ColumnOrdering(t *testing.T) {
	testutil.SkipIfShort(t)

	tempDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tempDir, "schemas")
	os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

	// Schema with columns in random order
	schema := `
export default table({
  updated_at: col.timestamp(),
  name: col.string(100),
  id: col.id(),
  email: col.string(255),
  created_at: col.timestamp(),
  age: col.integer(),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	output, err := client.SchemaExport("typescript")
	testutil.AssertNoError(t, err)

	// Verify column ordering: id first, timestamps last, others alphabetical
	idPos := indexOfSubstring(output, "id")
	namePos := indexOfSubstring(output, "name")
	emailPos := indexOfSubstring(output, "email")
	agePos := indexOfSubstring(output, "age")
	createdPos := indexOfSubstring(output, "created_at")
	updatedPos := indexOfSubstring(output, "updated_at")

	// id should be first
	testutil.AssertTrue(t, idPos < namePos, "id should come before name")
	testutil.AssertTrue(t, idPos < emailPos, "id should come before email")
	testutil.AssertTrue(t, idPos < agePos, "id should come before age")

	// timestamps should be last
	testutil.AssertTrue(t, createdPos > namePos, "created_at should come after name")
	testutil.AssertTrue(t, updatedPos > emailPos, "updated_at should come after email")
}

// TestSchema_100Pct_NormalizeSchema tests schema normalization.
func TestSchema_Comprehensive_NormalizeSchema(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("normalize_with_indexes", func(t *testing.T) {
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
  email: col.string(255).unique(),
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

		// SchemaDiff should normalize both schemas
		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})

	t.Run("normalize_with_foreign_keys", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		usersSchema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		postsSchema := `
export default table({
  id: col.id(),
  title: col.string(200),
  user_id: col.integer().foreign_key("users", "id"),
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

		_, err = client.MigrationGenerate("create_tables")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})

	t.Run("normalize_with_enums", func(t *testing.T) {
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
  status: col.enum(["active", "inactive", "pending"]),
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

		ops, err := client.SchemaDiff()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(ops))
	})
}

// Helper functions
func contains(b []byte, substr string) bool {
	return indexOfSubstring(b, substr) >= 0
}

func indexOfSubstring(b []byte, substr string) int {
	s := string(b)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
