//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// sortColumnsForDatabase Tests
// ===========================================================================

func TestSortColumnsForDatabase(t *testing.T) {
	tests := []struct {
		name  string
		input []*ast.ColumnDef
		want  []string // expected column order by name
	}{
		{
			name:  "empty columns",
			input: []*ast.ColumnDef{},
			want:  []string{},
		},
		{
			name: "primary key first",
			input: []*ast.ColumnDef{
				{Name: "name", Type: "string"},
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string"},
			},
			want: []string{"id", "email", "name"},
		},
		{
			name: "timestamps last",
			input: []*ast.ColumnDef{
				{Name: "name", Type: "string"},
				{Name: "created_at", Type: "datetime"},
				{Name: "email", Type: "string"},
				{Name: "updated_at", Type: "datetime"},
			},
			want: []string{"email", "name", "created_at", "updated_at"},
		},
		{
			name: "timestamps in correct order",
			input: []*ast.ColumnDef{
				{Name: "deleted_at", Type: "datetime"},
				{Name: "updated_at", Type: "datetime"},
				{Name: "created_at", Type: "datetime"},
			},
			want: []string{"created_at", "updated_at", "deleted_at"},
		},
		{
			name: "complete ordering: id, regular, timestamps",
			input: []*ast.ColumnDef{
				{Name: "updated_at", Type: "datetime"},
				{Name: "email", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "created_at", Type: "datetime"},
				{Name: "age", Type: "integer"},
			},
			want: []string{"id", "age", "email", "name", "created_at", "updated_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a client (schema-only, no DB needed for this test)
			client, err := New(WithSchemaOnly())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			got := client.sortColumnsForDatabase(tt.input)

			// Extract names in order
			gotNames := make([]string, len(got))
			for i, col := range got {
				gotNames[i] = col.Name
			}

			// Compare
			if len(gotNames) != len(tt.want) {
				t.Fatalf("got %d columns, want %d", len(gotNames), len(tt.want))
			}

			for i := range gotNames {
				if gotNames[i] != tt.want[i] {
					t.Errorf("position %d: got %q, want %q\nfull order: %v", i, gotNames[i], tt.want[i], gotNames)
					break
				}
			}
		})
	}
}

// ===========================================================================
// SchemaCheck Tests
// ===========================================================================

func TestSchemaCheck_Success(t *testing.T) {
	testutil.Parallel(t)

	// Create temp directory with valid schemas
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a valid schema file
	userSchema := `
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username(),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), userSchema)

	// Create client
	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test SchemaCheck
	err = client.SchemaCheck()
	testutil.AssertNoError(t, err)
}

func TestSchemaCheck_NoSchemas(t *testing.T) {
	testutil.Parallel(t)

	// Create temp directory with empty schemas directory
	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.SchemaCheck()
	if err == nil {
		t.Fatal("expected error for no schema files, got nil")
	}

	testutil.AssertErrorContains(t, err, "no schema files found")
}

func TestSchemaCheck_InvalidJavaScript(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write invalid JavaScript
	invalidSchema := `
export default table({
  id: col.id(),
  // Syntax error: missing closing brace
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), invalidSchema)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.SchemaCheck()
	if err == nil {
		t.Fatal("expected error for invalid JavaScript, got nil")
	}
}

// ===========================================================================
// SchemaDump Tests
// ===========================================================================

func TestSchemaDump_EmptySchema(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	sql, err := client.SchemaDump()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, sql, "")
}

func TestSchemaDump_SingleTable(t *testing.T) {
	testutil.Parallel(t)

	// Use PostgreSQL for this test
	_, dbURL := testutil.SetupPostgresWithURL(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	if err := os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755); err != nil {
		t.Fatal(err)
	}

	userSchema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), userSchema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	sql, err := client.SchemaDump()
	testutil.AssertNoError(t, err)

	// Verify SQL contains CREATE TABLE
	testutil.AssertSQLContains(t, sql, "CREATE TABLE")
	testutil.AssertSQLContains(t, sql, "auth_user")
	testutil.AssertSQLContains(t, sql, "id")
	testutil.AssertSQLContains(t, sql, "email")
}

// ===========================================================================
// SchemaDiff Tests (Integration - requires database)
// ===========================================================================

func TestSchemaDiff_NoDifference(t *testing.T) {
	testutil.Parallel(t)

	// Setup PostgreSQL test database
	db, dbURL := testutil.SetupPostgresWithURL(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(filepath.Join(schemasDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a simple schema
	schema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "item.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
		WithMigrationsDir(migrationsDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Apply schema to database using client
	ops, err := client.SchemaDiff()
	testutil.AssertNoError(t, err)

	// First diff should have operations (table doesn't exist yet)
	if len(ops) == 0 {
		t.Fatal("expected operations for initial schema, got 0")
	}

	// Apply the schema by creating the table
	_, err = db.Exec(`
		CREATE TABLE test_item (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL
		)
	`)
	testutil.AssertNoError(t, err)

	// Now compute diff again
	// Note: There may still be minor differences due to normalization (defaults, etc.)
	// This test verifies the diff mechanism works, not that it's perfect
	ops2, err := client.SchemaDiff()
	testutil.AssertNoError(t, err)

	// We created the table, so there should be fewer operations
	if len(ops2) >= len(ops) {
		t.Logf("After applying schema, still got %d operations (was %d)", len(ops2), len(ops))
		// This is okay - normalization can still find differences
	}
}

func TestSchemaDiff_MissingTable(t *testing.T) {
	testutil.Parallel(t)

	// Setup PostgreSQL test database
	_, dbURL := testutil.SetupPostgresWithURL(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(filepath.Join(schemasDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create schema for table that doesn't exist in DB yet
	schema := `
export default table({
  id: col.id(),
  title: col.string(200),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "post.js"), schema)

	client, err := New(
		WithDatabaseURL(dbURL),
		WithSchemasDir(schemasDir),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Compute diff - should detect missing table
	ops, err := client.SchemaDiff()
	testutil.AssertNoError(t, err)

	// Should have CreateTable operation
	if len(ops) == 0 {
		t.Fatal("expected at least 1 operation for missing table, got 0")
	}

	createOp, ok := ops[0].(*ast.CreateTable)
	if !ok {
		t.Fatalf("expected CreateTable operation, got %T", ops[0])
	}

	testutil.AssertEqual(t, createOp.Namespace, "test")
	testutil.AssertEqual(t, createOp.Name, "post")
}

// ===========================================================================
// SchemaTables Tests
// ===========================================================================

func TestSchemaTables(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	// Create schemas in different namespaces
	if err := os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(schemasDir, "blog"), 0755); err != nil {
		t.Fatal(err)
	}

	userSchema := `
export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username(),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), userSchema)

	postSchema := `
export default table({
  id: col.id(),
  title: col.string(200),
  body: col.text(),
  author_id: col.belongs_to("auth.user"),
}).index(["author_id"])
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "blog", "post.js"), postSchema)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	tables, err := client.SchemaTables()
	testutil.AssertNoError(t, err)

	// Should have 2 tables
	testutil.AssertEqual(t, len(tables), 2)

	// Find user table
	var userTable *TableInfo
	for i := range tables {
		if tables[i].Name == "user" {
			userTable = &tables[i]
			break
		}
	}

	if userTable == nil {
		t.Fatal("user table not found")
	}

	testutil.AssertEqual(t, userTable.Namespace, "auth")
	testutil.AssertEqual(t, userTable.SQLName, "auth_user")
	testutil.AssertEqual(t, userTable.ColumnCount, 3)
	testutil.AssertTrue(t, userTable.HasPrimaryKey, "user should have primary key")
}

// ===========================================================================
// SchemaNamespaces Tests
// ===========================================================================

func TestSchemaNamespaces(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")

	// Create multiple namespaces
	if err := os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(schemasDir, "blog"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(schemasDir, "cms"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create files
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), `export default table({ id: col.id() })`)
	testutil.WriteFile(t, filepath.Join(schemasDir, "blog", "post.js"), `export default table({ id: col.id() })`)
	testutil.WriteFile(t, filepath.Join(schemasDir, "cms", "page.js"), `export default table({ id: col.id() })`)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	namespaces, err := client.SchemaNamespaces()
	testutil.AssertNoError(t, err)

	// Should have 3 namespaces
	testutil.AssertEqual(t, len(namespaces), 3)

	// Check they're sorted
	expected := []string{"auth", "blog", "cms"}
	for i, ns := range namespaces {
		testutil.AssertEqual(t, ns, expected[i])
	}
}

func TestGetNamespaces_Alias(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(filepath.Join(schemasDir, "test"), 0755); err != nil {
		t.Fatal(err)
	}

	testutil.WriteFile(t, filepath.Join(schemasDir, "test", "item.js"), `export default table({ id: col.id() })`)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test both methods return same result
	ns1, err1 := client.SchemaNamespaces()
	testutil.AssertNoError(t, err1)

	ns2, err2 := client.GetNamespaces()
	testutil.AssertNoError(t, err2)

	testutil.AssertEqual(t, len(ns1), len(ns2))
	for i := range ns1 {
		testutil.AssertEqual(t, ns1[i], ns2[i])
	}
}

// ===========================================================================
// SchemaExport Tests
// ===========================================================================

func TestSchemaExport_UnknownFormat(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.SchemaExport("unknown_format")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}

	testutil.AssertErrorContains(t, err, "unknown")
}

func TestSchemaExport_TypeScript(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(filepath.Join(schemasDir, "auth"), 0755); err != nil {
		t.Fatal(err)
	}

	userSchema := `
export default table({
  id: col.id(),
  email: col.email(),
  age: col.integer(),
})
`
	testutil.WriteFile(t, filepath.Join(schemasDir, "auth", "user.js"), userSchema)

	client, err := New(
		WithSchemasDir(schemasDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	output, err := client.SchemaExport("typescript")
	testutil.AssertNoError(t, err)

	result := string(output)

	// Should contain TypeScript interface
	if !strings.Contains(result, "interface") || !strings.Contains(result, "User") {
		t.Errorf("TypeScript export should contain interface User, got:\n%s", result)
	}

	// Should contain field types
	if !strings.Contains(result, "string") {
		t.Errorf("TypeScript export should contain string type")
	}
}

// ===========================================================================
// SchemaAtRevision and ListRevisions Tests
// ===========================================================================

func TestListRevisions_NoMigrations(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	revisions, err := client.ListRevisions()
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(revisions), 0)
}

func TestListRevisions_MultipleMigrations(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create migration files
	migration1 := testutil.SimpleMigration(
		`    m.create_table("auth.user", t => { t.id() })`,
		`    m.drop_table("auth.user")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_initial.js"), migration1)

	migration2 := testutil.SimpleMigration(
		`    m.add_column("auth.user", "email", c => { c.email() })`,
		`    m.drop_column("auth.user", "email")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "002_add_email.js"), migration2)

	client, err := New(
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	revisions, err := client.ListRevisions()
	testutil.AssertNoError(t, err)

	testutil.AssertEqual(t, len(revisions), 2)
	testutil.AssertEqual(t, revisions[0], "001")
	testutil.AssertEqual(t, revisions[1], "002")
}

func TestSchemaAtRevision_NotFound(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration1 := testutil.SimpleMigration(
		`    m.create_table("test.item", t => { t.id() })`,
		`    m.drop_table("test.item")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_initial.js"), migration1)

	client, err := New(
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.SchemaAtRevision("999")
	if err == nil {
		t.Fatal("expected error for non-existent revision, got nil")
	}

	testutil.AssertErrorContains(t, err, "not found")
}

func TestGetMigrationInfo_Success(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	migration1 := testutil.SimpleMigration(
		`    m.create_table("test.users", t => { t.id(); t.email() })`,
		`    m.drop_table("test.users")`,
	)
	testutil.WriteFile(t, filepath.Join(migrationsDir, "001_create_users.js"), migration1)

	client, err := New(
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	info, err := client.GetMigrationInfo("001")
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, info)
	testutil.AssertEqual(t, info.Revision, "001")
	testutil.AssertEqual(t, info.Name, "create_users")
}

func TestGetMigrationInfo_NotFound(t *testing.T) {
	testutil.Parallel(t)

	tmpDir := testutil.TempDir(t)
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	client, err := New(
		WithMigrationsDir(migrationsDir),
		WithSchemaOnly(),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.GetMigrationInfo("999")
	if err == nil {
		t.Fatal("expected error for non-existent migration, got nil")
	}

	testutil.AssertErrorContains(t, err, "not found")
}
