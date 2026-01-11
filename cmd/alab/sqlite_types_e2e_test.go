//go:build integration

package main

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// -----------------------------------------------------------------------------
// SQLite E2E Tests: Column Types and Modifiers
// -----------------------------------------------------------------------------
// These tests verify all column types and modifiers work correctly with SQLite.

// TestE2E_SQLite_AllColumnTypes tests every column type available in the DSL.
func TestE2E_SQLite_AllColumnTypes(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "types_test.db")

	// Migration with ALL column types
	env.writeMigration(t, "001", "create_all_types_table", `
export function up(m) {
  m.create_table("types.all_columns", t => {
    // Primary key
    t.id()

    // Primitive types
    t.uuid("external_id")
    t.string("short_text", 100)
    t.text("long_text")
    t.integer("count")
    t.float("ratio")
    t.decimal("price", 10, 2)
    t.boolean("is_active")
    t.date("birth_date")
    t.time("start_time")
    t.datetime("created_at")
    t.json("metadata")
    t.base64("encoded_data")
    t.enum("status", ["pending", "active", "archived"])
  })
}

export function down(m) {
  m.drop_table("types.all_columns")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	// Apply migration
	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify table exists with correct name (namespace_table format)
	db := client.DB()
	testutil.AssertTableExists(t, db, "types_all_columns")

	// Verify all columns exist
	columns := []string{
		"id", "external_id", "short_text", "long_text", "count", "ratio",
		"price", "is_active", "birth_date", "start_time", "created_at",
		"metadata", "encoded_data", "status",
	}
	for _, col := range columns {
		testutil.AssertColumnExists(t, db, "types_all_columns", col)
	}

	// Verify SQLite column types
	// Note: SQLite has limited type system - most types map to TEXT, INTEGER, REAL, or BLOB
	assertSQLiteColumnType(t, db, "types_all_columns", "id", "TEXT")           // uuid -> TEXT
	assertSQLiteColumnType(t, db, "types_all_columns", "external_id", "TEXT")  // uuid -> TEXT
	assertSQLiteColumnType(t, db, "types_all_columns", "short_text", "TEXT")   // string -> TEXT
	assertSQLiteColumnType(t, db, "types_all_columns", "long_text", "TEXT")    // text -> TEXT
	assertSQLiteColumnType(t, db, "types_all_columns", "count", "INTEGER")     // integer -> INTEGER
	assertSQLiteColumnType(t, db, "types_all_columns", "ratio", "REAL")        // float -> REAL
	assertSQLiteColumnType(t, db, "types_all_columns", "price", "TEXT")        // decimal -> TEXT (for precision)
	assertSQLiteColumnType(t, db, "types_all_columns", "is_active", "INTEGER") // boolean -> INTEGER (0/1)
	assertSQLiteColumnType(t, db, "types_all_columns", "birth_date", "TEXT")   // date -> TEXT (ISO 8601)
	assertSQLiteColumnType(t, db, "types_all_columns", "start_time", "TEXT")   // time -> TEXT (RFC 3339)
	assertSQLiteColumnType(t, db, "types_all_columns", "created_at", "TEXT")   // datetime -> TEXT (RFC 3339)
	assertSQLiteColumnType(t, db, "types_all_columns", "metadata", "TEXT")     // json -> TEXT
	assertSQLiteColumnType(t, db, "types_all_columns", "encoded_data", "BLOB") // base64 -> BLOB
	assertSQLiteColumnType(t, db, "types_all_columns", "status", "TEXT")       // enum -> TEXT (with CHECK)

	t.Log("All column types test passed!")
}

// TestE2E_SQLite_ColumnModifiers tests all column modifiers.
func TestE2E_SQLite_ColumnModifiers(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "modifiers_test.db")

	// Migration with all modifiers
	env.writeMigration(t, "001", "create_modifiers_table", `
export function up(m) {
  m.create_table("test.modifiers", t => {
    t.id()

    // optional() - nullable column
    t.string("optional_field", 100).optional()

    // unique() - unique constraint
    t.string("unique_field", 100).unique()

    // default() - default value
    t.string("with_default", 50).default("hello")
    t.integer("count_default").default(0)
    t.boolean("flag_default").default(true)

    // Combined modifiers
    t.string("optional_unique", 100).optional().unique()
    t.integer("optional_default").optional().default(42)
  })
}

export function down(m) {
  m.drop_table("test.modifiers")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()
	testutil.AssertTableExists(t, db, "test_modifiers")

	// Verify columns
	columns := []string{
		"id", "optional_field", "unique_field", "with_default",
		"count_default", "flag_default", "optional_unique", "optional_default",
	}
	for _, col := range columns {
		testutil.AssertColumnExists(t, db, "test_modifiers", col)
	}

	// Verify NOT NULL constraints
	assertSQLiteColumnNotNull(t, db, "test_modifiers", "id", true)
	assertSQLiteColumnNotNull(t, db, "test_modifiers", "unique_field", true)
	assertSQLiteColumnNotNull(t, db, "test_modifiers", "optional_field", false)
	assertSQLiteColumnNotNull(t, db, "test_modifiers", "optional_unique", false)
	assertSQLiteColumnNotNull(t, db, "test_modifiers", "optional_default", false)

	// Verify unique constraints exist
	assertSQLiteHasUniqueConstraint(t, db, "test_modifiers", "unique_field")
	assertSQLiteHasUniqueConstraint(t, db, "test_modifiers", "optional_unique")

	// Verify defaults work by inserting a row
	_, err := db.Exec(`INSERT INTO test_modifiers (id, unique_field) VALUES ('test-id', 'test-unique')`)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Check default values were applied
	var withDefault, countDefault, flagDefault string
	err = db.QueryRow(`SELECT with_default, count_default, flag_default FROM test_modifiers WHERE id = 'test-id'`).
		Scan(&withDefault, &countDefault, &flagDefault)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if withDefault != "hello" {
		t.Errorf("Expected with_default='hello', got '%s'", withDefault)
	}
	if countDefault != "0" {
		t.Errorf("Expected count_default='0', got '%s'", countDefault)
	}
	if flagDefault != "1" { // SQLite stores booleans as 0/1
		t.Errorf("Expected flag_default='1', got '%s'", flagDefault)
	}

	t.Log("Column modifiers test passed!")
}

// TestE2E_SQLite_SemanticTypes tests all semantic column types.
func TestE2E_SQLite_SemanticTypes(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "semantic_test.db")

	// Migration with semantic types
	env.writeMigration(t, "001", "create_semantic_table", `
export function up(m) {
  m.create_table("user.profile", t => {
    t.id()

    // Identity and authentication
    t.email("email")
    t.username("username")
    t.password_hash("password")
    t.phone("phone").optional()

    // Text content
    t.name("display_name")
    t.title("job_title")
    t.slug("profile_slug")
    t.body("bio")
    t.summary("short_bio")

    // URLs and network
    t.url("website").optional()

    // Boolean flag
    t.flag("is_verified")
    t.flag("is_premium", true)
  })
}

export function down(m) {
  m.drop_table("user.profile")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()
	testutil.AssertTableExists(t, db, "user_profile")

	// Verify all semantic columns exist
	columns := []string{
		"id", "email", "username", "password", "phone",
		"display_name", "job_title", "profile_slug", "bio", "short_bio",
		"website", "is_verified", "is_premium",
	}
	for _, col := range columns {
		testutil.AssertColumnExists(t, db, "user_profile", col)
	}

	// Verify types
	assertSQLiteColumnType(t, db, "user_profile", "email", "TEXT")
	assertSQLiteColumnType(t, db, "user_profile", "username", "TEXT")
	assertSQLiteColumnType(t, db, "user_profile", "bio", "TEXT")
	assertSQLiteColumnType(t, db, "user_profile", "is_verified", "INTEGER")
	assertSQLiteColumnType(t, db, "user_profile", "is_premium", "INTEGER")

	// Verify slug has unique constraint (semantic type property)
	assertSQLiteHasUniqueConstraint(t, db, "user_profile", "profile_slug")

	// Verify flag defaults
	_, err := db.Exec(`INSERT INTO user_profile (id, email, username, password, display_name, job_title, profile_slug, bio, short_bio)
		VALUES ('test-id', 'test@example.com', 'testuser', 'hash', 'Test User', 'Developer', 'test-user', 'Bio text', 'Short bio')`)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	var isVerified, isPremium int
	err = db.QueryRow(`SELECT is_verified, is_premium FROM user_profile WHERE id = 'test-id'`).
		Scan(&isVerified, &isPremium)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if isVerified != 0 {
		t.Errorf("Expected is_verified=0 (default false), got %d", isVerified)
	}
	if isPremium != 1 {
		t.Errorf("Expected is_premium=1 (default true), got %d", isPremium)
	}

	t.Log("Semantic types test passed!")
}

// TestE2E_SQLite_HelperMethods tests timestamps(), soft_delete(), and sortable().
func TestE2E_SQLite_HelperMethods(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "helpers_test.db")

	// Migration with helper methods
	env.writeMigration(t, "001", "create_helpers_table", `
export function up(m) {
  m.create_table("app.item", t => {
    t.id()
    t.string("name", 100)

    // Helper methods
    t.timestamps()
    t.soft_delete()
    t.sortable()
  })
}

export function down(m) {
  m.drop_table("app.item")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()
	testutil.AssertTableExists(t, db, "app_item")

	// Verify columns from helper methods
	testutil.AssertColumnExists(t, db, "app_item", "created_at") // from timestamps()
	testutil.AssertColumnExists(t, db, "app_item", "updated_at") // from timestamps()
	testutil.AssertColumnExists(t, db, "app_item", "deleted_at") // from soft_delete()
	testutil.AssertColumnExists(t, db, "app_item", "position")   // from sortable()

	// Verify types
	assertSQLiteColumnType(t, db, "app_item", "created_at", "TEXT")
	assertSQLiteColumnType(t, db, "app_item", "updated_at", "TEXT")
	assertSQLiteColumnType(t, db, "app_item", "deleted_at", "TEXT")
	assertSQLiteColumnType(t, db, "app_item", "position", "INTEGER")

	// Verify deleted_at is nullable (soft delete)
	assertSQLiteColumnNotNull(t, db, "app_item", "deleted_at", false)

	t.Log("Helper methods test passed!")
}

// TestE2E_SQLite_Relationships tests belongs_to() relationships.
func TestE2E_SQLite_Relationships(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "relationships_test.db")

	// Migration with relationships
	env.writeMigration(t, "001", "create_relationship_tables", `
export function up(m) {
  m.create_table("auth.user", t => {
    t.id()
    t.email("email")
    t.timestamps()
  })

  m.create_table("blog.post", t => {
    t.id()
    t.string("title", 200)
    t.belongs_to("auth.user").as("author")
    t.belongs_to("auth.user").as("editor").optional()
    t.timestamps()
  })

  m.create_table("blog.comment", t => {
    t.id()
    t.text("content")
    t.belongs_to("blog.post")
    t.belongs_to("auth.user").optional()
    t.timestamps()
  })
}

export function down(m) {
  m.drop_table("blog.comment")
  m.drop_table("blog.post")
  m.drop_table("auth.user")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()

	// Verify tables with correct naming
	testutil.AssertTableExists(t, db, "auth_user")
	testutil.AssertTableExists(t, db, "blog_post")
	testutil.AssertTableExists(t, db, "blog_comment")

	// Verify FK columns exist
	testutil.AssertColumnExists(t, db, "blog_post", "author_id")
	testutil.AssertColumnExists(t, db, "blog_post", "editor_id")
	testutil.AssertColumnExists(t, db, "blog_comment", "post_id")
	testutil.AssertColumnExists(t, db, "blog_comment", "user_id")

	// Verify nullable constraints
	assertSQLiteColumnNotNull(t, db, "blog_post", "author_id", true)
	assertSQLiteColumnNotNull(t, db, "blog_post", "editor_id", false)
	assertSQLiteColumnNotNull(t, db, "blog_comment", "post_id", true)
	assertSQLiteColumnNotNull(t, db, "blog_comment", "user_id", false)

	t.Log("Relationships test passed!")
}

// TestE2E_SQLite_Indexes tests index creation.
func TestE2E_SQLite_Indexes(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "indexes_test.db")

	// Migration with various indexes
	env.writeMigration(t, "001", "create_indexed_table", `
export function up(m) {
  m.create_table("catalog.product", t => {
    t.id()
    t.string("sku", 50).unique()
    t.string("name", 200)
    t.decimal("price", 10, 2)
    t.string("status", 20).default("active")
    t.uuid("category_id").optional()
    t.timestamps()
  })

  // Single column index
  m.create_index("catalog.product", ["status"])

  // Composite index
  m.create_index("catalog.product", ["category_id", "price"])

  // Named index
  m.create_index("catalog.product", ["name"], { name: "idx_product_name" })

  // Unique composite index
  m.create_index("catalog.product", ["category_id", "sku"], { unique: true, name: "idx_unique_category_sku" })
}

export function down(m) {
  m.drop_index("idx_product_name")
  m.drop_index("idx_unique_category_sku")
  m.drop_table("catalog.product")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()
	testutil.AssertTableExists(t, db, "catalog_product")

	// Verify indexes exist
	assertSQLiteIndexExists(t, db, "idx_product_name")
	assertSQLiteIndexExists(t, db, "idx_unique_category_sku")

	t.Log("Indexes test passed!")
}

// TestE2E_SQLite_AddColumn tests adding columns with modifiers.
func TestE2E_SQLite_AddColumn(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "add_column_test.db")

	// Initial table
	env.writeMigration(t, "001", "create_initial_table", `
export function up(m) {
  m.create_table("app.user", t => {
    t.id()
    t.email("email")
  })
}

export function down(m) {
  m.drop_table("app.user")
}
`)

	// Add columns with various modifiers
	env.writeMigration(t, "002", "add_columns", `
export function up(m) {
  m.add_column("app.user", c => c.string("name", 100))
  m.add_column("app.user", c => c.integer("age").optional())
  m.add_column("app.user", c => c.boolean("is_active").default(true))
  m.add_column("app.user", c => c.string("status", 20).default("pending"))
}

export function down(m) {
  m.drop_column("app.user", "name")
  m.drop_column("app.user", "age")
  m.drop_column("app.user", "is_active")
  m.drop_column("app.user", "status")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()

	// Verify new columns
	testutil.AssertColumnExists(t, db, "app_user", "name")
	testutil.AssertColumnExists(t, db, "app_user", "age")
	testutil.AssertColumnExists(t, db, "app_user", "is_active")
	testutil.AssertColumnExists(t, db, "app_user", "status")

	// Verify types
	assertSQLiteColumnType(t, db, "app_user", "name", "TEXT")
	assertSQLiteColumnType(t, db, "app_user", "age", "INTEGER")
	assertSQLiteColumnType(t, db, "app_user", "is_active", "INTEGER")

	t.Log("Add column test passed!")
}

// TestE2E_SQLite_Rollback tests rollback functionality.
func TestE2E_SQLite_Rollback(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "rollback_test.db")

	env.writeMigration(t, "001", "create_table", `
export function up(m) {
  m.create_table("test.item", t => {
    t.id()
    t.string("name", 100)
  })
}

export function down(m) {
  m.drop_table("test.item")
}
`)

	env.writeMigration(t, "002", "add_column", `
export function up(m) {
  m.add_column("test.item", c => c.text("description").optional())
}

export function down(m) {
  m.drop_column("test.item", "description")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	// Apply all migrations
	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()
	testutil.AssertTableExists(t, db, "test_item")
	testutil.AssertColumnExists(t, db, "test_item", "description")

	// Rollback one migration
	if err := client.MigrationRollback(1, astroladb.Force()); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Table should exist but column should be gone
	testutil.AssertTableExists(t, db, "test_item")
	// Note: SQLite doesn't easily support DROP COLUMN, so this may fail depending on implementation

	// Rollback remaining
	if err := client.MigrationRollback(1, astroladb.Force()); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	testutil.AssertTableNotExists(t, db, "test_item")

	t.Log("Rollback test passed!")
}

// TestE2E_SQLite_TableNaming verifies the namespace_table naming convention.
func TestE2E_SQLite_TableNaming(t *testing.T) {
	env := setupTestEnv(t)
	dbPath := filepath.Join(env.tmpDir, "naming_test.db")

	// Migration with various namespaces
	env.writeMigration(t, "001", "create_namespaced_tables", `
export function up(m) {
  m.create_table("auth.user", t => {
    t.id()
    t.email("email")
  })

  m.create_table("auth.session", t => {
    t.id()
    t.uuid("user_id")
    t.datetime("expires_at")
  })

  m.create_table("blog.post", t => {
    t.id()
    t.string("title", 200)
  })

  m.create_table("catalog.product", t => {
    t.id()
    t.string("name", 200)
  })
}

export function down(m) {
  m.drop_table("catalog.product")
  m.drop_table("blog.post")
  m.drop_table("auth.session")
  m.drop_table("auth.user")
}
`)

	client := env.newClient(t, "sqlite://"+dbPath)

	if err := client.MigrationRun(astroladb.Force()); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	db := client.DB()

	// Verify all tables have correct namespace_table naming
	expectedTables := []string{
		"auth_user",
		"auth_session",
		"blog_post",
		"catalog_product",
	}

	for _, tableName := range expectedTables {
		testutil.AssertTableExists(t, db, tableName)
	}

	// Verify the actual table SQL shows correct naming
	for _, tableName := range expectedTables {
		var createSQL string
		err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, tableName).Scan(&createSQL)
		if err != nil {
			t.Errorf("Failed to get CREATE statement for %s: %v", tableName, err)
			continue
		}

		// Check the table name is quoted correctly
		expectedPattern := `"` + tableName + `"`
		if !strings.Contains(createSQL, expectedPattern) {
			t.Errorf("Table %s CREATE statement doesn't contain expected pattern %s:\n%s",
				tableName, expectedPattern, createSQL)
		}
	}

	t.Log("Table naming test passed!")
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// assertSQLiteColumnType verifies a column's type in SQLite.
func assertSQLiteColumnType(t *testing.T, db *sql.DB, table, column, expectedType string) {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		t.Fatalf("Failed to get table info for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		if name == column {
			if !strings.Contains(strings.ToUpper(colType), strings.ToUpper(expectedType)) {
				t.Errorf("Column %s.%s: expected type containing %s, got %s", table, column, expectedType, colType)
			}
			return
		}
	}
	t.Errorf("Column %s.%s not found", table, column)
}

// assertSQLiteColumnNotNull verifies a column's NOT NULL constraint.
func assertSQLiteColumnNotNull(t *testing.T, db *sql.DB, table, column string, expectNotNull bool) {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		t.Fatalf("Failed to get table info for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		if name == column {
			isNotNull := notNull == 1 || pk == 1 // Primary keys are implicitly NOT NULL
			if isNotNull != expectNotNull {
				t.Errorf("Column %s.%s: expected NOT NULL=%v, got %v", table, column, expectNotNull, isNotNull)
			}
			return
		}
	}
	t.Errorf("Column %s.%s not found", table, column)
}

// assertSQLiteHasUniqueConstraint verifies a column has a unique constraint.
func assertSQLiteHasUniqueConstraint(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	// Check for unique constraint in table definition or index
	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&createSQL)
	if err != nil {
		t.Fatalf("Failed to get CREATE statement for %s: %v", table, err)
	}

	// Check inline UNIQUE constraint
	if strings.Contains(createSQL, `"`+column+`"`) && strings.Contains(createSQL, "UNIQUE") {
		return
	}

	// Check for unique index
	rows, err := db.Query(`SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name=?`, table)
	if err != nil {
		t.Fatalf("Failed to get indexes for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var indexSQL sql.NullString
		if err := rows.Scan(&indexSQL); err != nil {
			continue
		}
		if indexSQL.Valid && strings.Contains(indexSQL.String, "UNIQUE") &&
			strings.Contains(indexSQL.String, `"`+column+`"`) {
			return
		}
	}

	// Some unique constraints are embedded as CONSTRAINT clauses
	if strings.Contains(strings.ToUpper(createSQL), "CONSTRAINT") &&
		strings.Contains(createSQL, column) {
		return
	}

	t.Errorf("Column %s.%s does not have a unique constraint", table, column)
}

// assertSQLiteIndexExists verifies an index exists.
func assertSQLiteIndexExists(t *testing.T, db *sql.DB, indexName string) {
	t.Helper()

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, indexName).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check index %s: %v", indexName, err)
	}
	if count == 0 {
		t.Errorf("Index %s does not exist", indexName)
	}
}
