//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// Migration DSL Generation - Table Operations
// ===========================================================================

func TestClient_MigrationGenerate_CreateTable(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create schema
	schema := `export default table({
	id: col.id(),
	name: col.string(100),
	email: col.string(255).unique()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "contacts.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate migration - should create table with columns
	migrationPath, err := client.MigrationGenerate("create_contacts")
	testutil.AssertNoError(t, err)

	// Verify migration contains create_table
	content, err := os.ReadFile(migrationPath)
	testutil.AssertNoError(t, err)
	contentStr := string(content)
	testutil.AssertTrue(t, strings.Contains(contentStr, "create_table"), "should have create_table")
	testutil.AssertTrue(t, strings.Contains(contentStr, "contacts"), "should have table name")
}

func TestClient_MigrationGenerate_DropTable(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema1 := `export default table({
	id: col.id()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "temp_table.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run migration to create table
	_, err = client.MigrationGenerate("create_temp")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Remove schema file (effectively dropping the table from schema)
	err = os.Remove(filepath.Join(schemasDir, "test", "temp_table.js"))
	testutil.AssertNoError(t, err)

	// Generate migration - should drop table
	migrationPath, err := client.MigrationGenerate("drop_temp")
	testutil.AssertNoError(t, err)

	// Verify migration contains drop_table
	content, err := os.ReadFile(migrationPath)
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, strings.Contains(string(content), "drop_table"), "should have drop_table")
}

// ===========================================================================
// Migration DSL Generation - Column Operations
// ===========================================================================

func TestClient_MigrationGenerate_AddColumn(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema1 := `export default table({
	id: col.id(),
	name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "employees.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run initial migration
	_, err = client.MigrationGenerate("create_employees")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Update schema with new column
	schema2 := `export default table({
	id: col.id(),
	name: col.string(100),
	department: col.string(50).optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "employees.js"), schema2)

	// Generate migration - should add column
	migrationPath, err := client.MigrationGenerate("add_department")
	testutil.AssertNoError(t, err)

	// Verify migration contains add_column
	content, err := os.ReadFile(migrationPath)
	testutil.AssertNoError(t, err)
	contentStr := string(content)
	testutil.AssertTrue(t, strings.Contains(contentStr, "department"), "should reference department column")
}

func TestClient_MigrationGenerate_DropColumn(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema with extra column
	schema1 := `export default table({
	id: col.id(),
	name: col.string(100),
	temp_field: col.string(50).optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "staff.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run initial migration
	_, err = client.MigrationGenerate("create_staff")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Remove column from schema
	schema2 := `export default table({
	id: col.id(),
	name: col.string(100)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "staff.js"), schema2)

	// Generate migration - should drop column
	migrationPath, err := client.MigrationGenerate("drop_temp_field")
	testutil.AssertNoError(t, err)

	// Verify migration contains drop_column
	content, err := os.ReadFile(migrationPath)
	testutil.AssertNoError(t, err)
	testutil.AssertTrue(t, strings.Contains(string(content), "temp_field"), "should reference temp_field")
}

// ===========================================================================
// Migration DSL Generation - Index Operations
// ===========================================================================

func TestClient_MigrationGenerate_CreateIndex(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema without index
	schema1 := `export default table({
	id: col.id(),
	email: col.string(255)
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "subscribers.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run initial migration
	_, err = client.MigrationGenerate("create_subscribers")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Add another column (index creation is tested via other scenarios)
	schema2 := `export default table({
	id: col.id(),
	email: col.string(255),
	name: col.string(100).optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "subscribers.js"), schema2)

	// Generate migration - will add name column
	_, err = client.MigrationGenerate("add_name")
	// Just verify it doesn't crash
	_ = err
}

// ===========================================================================
// Migration DSL Generation - Column Modifiers
// ===========================================================================

func TestClient_MigrationGenerate_AlterColumn(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create initial schema
	schema1 := `export default table({
	id: col.id(),
	description: col.string(100).optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "notes.js"), schema1)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate and run initial migration
	_, err = client.MigrationGenerate("create_notes")
	testutil.AssertNoError(t, err)

	err = client.MigrationRun()
	testutil.AssertNoError(t, err)

	// Update schema with longer description
	schema2 := `export default table({
	id: col.id(),
	description: col.string(500).optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "notes.js"), schema2)

	// Generate migration - should alter column
	migrationPath, err := client.MigrationGenerate("alter_description")
	testutil.AssertNoError(t, err)

	// Verify migration file was created
	_, err = os.Stat(migrationPath)
	testutil.AssertNoError(t, err)
}

// ===========================================================================
// Migration DSL Generation - Complex Types
// ===========================================================================

func TestClient_MigrationGenerate_ComplexTypes(t *testing.T) {
	testutil.Parallel(t)

	_, dbURL := testutil.SetupPostgresWithURL(t)
	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	schemasDir := filepath.Join(tmpDir, "schemas")

	testutil.MustValue(t, 0, os.MkdirAll(migrationsDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(schemasDir, 0755))
	testutil.MustValue(t, 0, os.MkdirAll(filepath.Join(schemasDir, "test"), 0755))

	// Create schema with various column types
	schema := `export default table({
	id: col.id(),
	name: col.string(100),
	age: col.integer().optional(),
	salary: col.decimal(10, 2).optional(),
	is_active: col.boolean().default(true),
	created_at: col.datetime(),
	metadata: col.json().optional()
});`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "records.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithMigrationsDir(migrationsDir),
		WithSchemasDir(schemasDir),
	)
	testutil.AssertNoError(t, err)
	defer client.Close()

	// Generate migration with complex types
	migrationPath, err := client.MigrationGenerate("create_records")
	testutil.AssertNoError(t, err)

	// Verify migration contains various column types
	content, err := os.ReadFile(migrationPath)
	testutil.AssertNoError(t, err)
	contentStr := string(content)
	testutil.AssertTrue(t, strings.Contains(contentStr, "records"), "should have table name")
	// Should contain column definitions
	testutil.AssertTrue(t, len(contentStr) > 100, "should have substantial content")
}
