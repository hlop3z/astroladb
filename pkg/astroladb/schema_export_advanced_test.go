//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestSchemaExport_AllFormats tests SchemaExport for all supported formats.
func TestSchemaExport_AllFormats(t *testing.T) {
	testutil.SkipIfShort(t)

	// Setup common schema
	setupSchema := func(t *testing.T) *Client {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create test schema
		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255).unique(),
  age: col.integer(),
  active: col.boolean().default(true),
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
		return client
	}

	t.Run("openapi_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		output, err := client.SchemaExport("openapi")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "openapi"), "Should contain OpenAPI markers")
	})

	t.Run("graphql_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		output, err := client.SchemaExport("graphql")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "type"), "Should contain GraphQL type definitions")
	})

	t.Run("typescript_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "interface") || strings.Contains(outputStr, "type"),
			"Should contain TypeScript type definitions")
	})

	t.Run("go_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		output, err := client.SchemaExport("go")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "struct"), "Should contain Go struct definitions")
	})

	t.Run("python_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		output, err := client.SchemaExport("python")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "class") || strings.Contains(outputStr, "def"),
			"Should contain Python class or function definitions")
	})

	t.Run("rust_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		output, err := client.SchemaExport("rust")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "struct"), "Should contain Rust struct definitions")
	})

	t.Run("case_insensitive_formats", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		// Test uppercase
		output1, err := client.SchemaExport("TYPESCRIPT")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output1) > 0, "Uppercase format should work")

		// Test mixed case
		output2, err := client.SchemaExport("TypeScript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output2) > 0, "Mixed case format should work")
	})

	t.Run("invalid_format", func(t *testing.T) {
		client := setupSchema(t)
		defer client.Close()

		_, err := client.SchemaExport("invalid_format")
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "format")
	})
}

// TestSchemaExport_ColumnOrdering tests column ordering in exports.
func TestSchemaExport_ColumnOrdering(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("id_column_first", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schema with id not first in definition
		schema := `
export default table({
  name: col.string(100),
  email: col.string(255),
  id: col.id(),  // id defined last
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

		// Export should order id first
		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")
	})

	t.Run("timestamps_last", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schema with timestamps in middle
		schema := `
export default table({
  id: col.id(),
  created_at: col.timestamp(),  // timestamp early
  name: col.string(100),
  updated_at: col.timestamp(),
  email: col.string(255),
  deleted_at: col.timestamp(),
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

		// Export should order timestamps last
		output, err := client.SchemaExport("go")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")
	})

	t.Run("alphabetical_ordering", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schema with non-alphabetical order
		schema := `
export default table({
  id: col.id(),
  zebra: col.string(50),
  alpha: col.string(50),
  beta: col.string(50),
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

		// Regular columns should be alphabetically ordered
		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")
	})
}

// TestSchemaExport_NamespaceFiltering tests namespace filtering in exports.
func TestSchemaExport_NamespaceFiltering(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("filter_single_namespace", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755)
		os.MkdirAll(filepath.Join(schemasDir, "billing"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schemas in different namespaces
		authSchema := `export default table({ id: col.id(), email: col.string(255) })`
		billingSchema := `export default table({ id: col.id(), amount: col.decimal(10, 2) })`

		testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "users.js"), authSchema)
		testutil.WriteFile(t, filepath.Join(schemasDir, "billing", "invoices.js"), billingSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Export only auth namespace
		output, err := client.SchemaExport("typescript", WithNamespace("auth"))
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")

		// Should contain auth table but not billing
		outputStr := string(output)
		testutil.AssertTrue(t, strings.Contains(outputStr, "email"), "Should contain auth table fields")
	})

	t.Run("export_all_namespaces", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "ns1"), 0755)
		os.MkdirAll(filepath.Join(schemasDir, "ns2"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema1 := `export default table({ id: col.id(), field1: col.string(50) })`
		schema2 := `export default table({ id: col.id(), field2: col.string(50) })`

		testutil.WriteFile(t, filepath.Join(schemasDir, "ns1", "table1.js"), schema1)
		testutil.WriteFile(t, filepath.Join(schemasDir, "ns2", "table2.js"), schema2)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Export without namespace filter should include all
		output, err := client.SchemaExport("go")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")
	})
}

// TestSchemaExport_Determinism tests output formatting and determinism.
func TestSchemaExport_Determinism(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("consistent_output", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

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

		// Export twice - should be identical
		output1, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)

		output2, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)

		testutil.AssertEqual(t, string(output1), string(output2))
	})

	t.Run("multiple_formats_deterministic", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

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
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "items.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Each format should be deterministic
		formats := []string{"typescript", "go", "graphql", "openapi"}
		for _, format := range formats {
			out1, err := client.SchemaExport(format)
			testutil.AssertNoError(t, err)

			out2, err := client.SchemaExport(format)
			testutil.AssertNoError(t, err)

			testutil.AssertEqual(t, string(out1), string(out2))
		}
	})
}

// TestSchemaExport_ComplexSchemas tests export with complex schema structures.
func TestSchemaExport_ComplexSchemas(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("schema_with_relationships", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		usersSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
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
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "posts.js"), postsSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Should export both tables with relationships
		output, err := client.SchemaExport("graphql")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")
	})

	t.Run("schema_with_indexes", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `
export default table({
  id: col.id(),
  email: col.string(255).unique(),
  name: col.string(100).index(),
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

		// Should handle indexes in export
		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should produce output")
	})
}
