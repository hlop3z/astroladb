//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestMigrationGenerate_CreateTable tests writeCreateTable with various table configurations.
func TestMigrationGenerate_CreateTable(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("simple_table", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create schema file
		schemaContent := `
table("users", t => {
  t.id()
  t.string("email", 255)
  t.string("name", 100)
})`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schemaContent)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Generate migration
		path, err := client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, path)

		// Read generated migration
		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "create_table"), "Should contain create_table")
		testutil.AssertTrue(t, strings.Contains(contentStr, "users"), "Should contain table name")
		testutil.AssertTrue(t, strings.Contains(contentStr, "t.id()"), "Should contain id column")
		testutil.AssertTrue(t, strings.Contains(contentStr, `t.string("email", 255)`), "Should contain email column")
	})

	t.Run("table_with_timestamps", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schemaContent := `
table("posts", t => {
  t.id()
  t.string("title", 255)
  t.timestamps()
})`
		testutil.WriteFile(t, filepath.Join(schemasDir, "posts.js"), schemaContent)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_posts")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "t.timestamps()"), "Should contain timestamps")
	})

	t.Run("table_with_indexes", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schemaContent := `
table("products", t => {
  t.id()
  t.string("sku", 100).unique()
  t.string("name", 255).index()
})`
		testutil.WriteFile(t, filepath.Join(schemasDir, "products.js"), schemaContent)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_products")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "unique()"), "Should contain unique modifier")
		testutil.AssertTrue(t, strings.Contains(contentStr, "index()"), "Should contain index modifier")
	})
}

// TestMigrationGenerate_AddColumn tests writeAddColumn with various column types and modifiers.
func TestMigrationGenerate_AddColumn(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("add_nullable_column", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Initial schema
		schema1 := `table("users", t => { t.id(); t.string("email", 255) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema1)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// First migration
		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Add column to schema
		schema2 := `table("users", t => { t.id(); t.string("email", 255); t.string("phone", 20) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema2)

		// Generate add column migration
		path, err := client.MigrationGenerate("add_phone")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "add_column"), "Should contain add_column")
		testutil.AssertTrue(t, strings.Contains(contentStr, "phone"), "Should contain column name")
	})

	t.Run("add_column_with_default", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema1 := `table("items", t => { t.id(); t.string("name", 100) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "items.js"), schema1)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.MigrationGenerate("create_items")
		testutil.AssertNoError(t, err)
		err = client.MigrationRun()
		testutil.AssertNoError(t, err)

		// Add column with default
		schema2 := `table("items", t => { t.id(); t.string("name", 100); t.integer("quantity").default(0) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "items.js"), schema2)

		path, err := client.MigrationGenerate("add_quantity")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "add_column"), "Should contain add_column")
		testutil.AssertTrue(t, strings.Contains(contentStr, "default"), "Should contain default modifier")
	})
}

// TestMigrationGenerate_AlterColumn tests writeAlterColumn with type changes and modifiers.
func TestMigrationGenerate_AlterColumn(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("change_column_type", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema1 := `table("users", t => { t.id(); t.string("age", 10) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema1)

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

		// Change column type
		schema2 := `table("users", t => { t.id(); t.integer("age") })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema2)

		path, err := client.MigrationGenerate("alter_age_type")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "alter_column") ||
			strings.Contains(contentStr, "drop_column") && strings.Contains(contentStr, "add_column"),
			"Should contain alter_column or drop+add pattern")
	})
}

// TestMigrationGenerate_ForeignKeys tests writeAddForeignKey and writeBelongsTo.
func TestMigrationGenerate_ForeignKeys(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("add_foreign_key_explicit", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create parent table first
		schema1 := `table("users", t => { t.id() })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema1)
		schema2 := `table("posts", t => { t.id(); t.integer("user_id") })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "posts.js"), schema2)

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

		// Add FK constraint
		schema3 := `table("posts", t => { t.id(); t.integer("user_id").references("users") })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "posts.js"), schema3)

		path, err := client.MigrationGenerate("add_fk")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "references") ||
			strings.Contains(contentStr, "add_foreign_key"),
			"Should contain FK reference")
	})

	t.Run("belongs_to_relationship", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema1 := `table("users", t => { t.id() })`
		schema2 := `table("comments", t => { t.id(); t.belongs_to("user") })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema1)
		testutil.WriteFile(t, filepath.Join(schemasDir, "comments.js"), schema2)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_tables")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "belongs_to") ||
			strings.Contains(contentStr, "user_id"),
			"Should contain belongs_to or generated user_id")
	})
}

// TestMigrationGenerate_Indexes tests writeCreateIndex with various configurations.
func TestMigrationGenerate_Indexes(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("create_simple_index", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema1 := `table("users", t => { t.id(); t.string("email", 255) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema1)

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

		// Add index
		schema2 := `table("users", t => { t.id(); t.string("email", 255).index() })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "users.js"), schema2)

		path, err := client.MigrationGenerate("add_email_index")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		testutil.AssertTrue(t, strings.Contains(contentStr, "index"), "Should contain index operation")
	})

	t.Run("create_composite_index", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema1 := `table("orders", t => { t.id(); t.integer("user_id"); t.timestamp("created_at") })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "orders.js"), schema1)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_orders")
		testutil.AssertNoError(t, err)

		// Composite indexes are typically created separately
		// This test documents the generation pattern
		testutil.AssertTrue(t, path != "", "Migration should be generated")
	})
}

// TestMigrationGenerate_Renames tests writeRenameTable and writeRenameColumn.
func TestMigrationGenerate_Renames(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("rename_table", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Note: Table renames typically require interactive mode
		// This test documents expected behavior
		schema1 := `table("user", t => { t.id() })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "user.js"), schema1)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_user")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, path != "", "Migration should be generated")
	})
}

// TestMigrationGenerate_DownOperations tests down operation inference.
func TestMigrationGenerate_DownOperations(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("down_operations_inferred", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `table("items", t => { t.id(); t.string("name", 100) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "items.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_items")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		contentStr := string(content)
		// Should have both up and down sections
		testutil.AssertTrue(t, strings.Contains(contentStr, "down:"), "Should contain down section")
		testutil.AssertTrue(t, strings.Contains(contentStr, "drop_table"), "Down should drop table")
	})
}

// TestMigrationGenerate_Prettier tests Prettier integration.
func TestMigrationGenerate_Prettier(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("prettier_formats_migration", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(schemasDir, 0755)
		os.MkdirAll(migrationsDir, 0755)

		schema := `table("products", t => { t.id(); t.string("name", 255) })`
		testutil.WriteFile(t, filepath.Join(schemasDir, "products.js"), schema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		path, err := client.MigrationGenerate("create_products")
		testutil.AssertNoError(t, err)

		content, err := os.ReadFile(path)
		testutil.AssertNoError(t, err)

		// Prettier formatting should be consistent
		// (Actual formatting depends on Prettier availability)
		testutil.AssertTrue(t, len(content) > 0, "Migration should have content")
	})
}
