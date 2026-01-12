//go:build integration

package main

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// -----------------------------------------------------------------------------
// PostgreSQL & CockroachDB E2E Tests: Column Types and Modifiers
// -----------------------------------------------------------------------------
// These tests verify all column types and modifiers work correctly with
// PostgreSQL and CockroachDB. Both databases share the same postgres wire protocol.

// dbSetup holds the database setup function and name for test parameterization.
type dbSetup struct {
	name      string
	setupFunc func(t *testing.T) (*sql.DB, string)
}

// getDatabaseSetups returns all database setups to test against.
func getDatabaseSetups() []dbSetup {
	return []dbSetup{
		{name: "PostgreSQL", setupFunc: testutil.SetupPostgresWithURL},
		{name: "CockroachDB", setupFunc: testutil.SetupCockroachDBWithURL},
	}
}

// TestE2E_Postgres_AllColumnTypes tests every column type available in the DSL.
func TestE2E_Postgres_AllColumnTypes(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with ALL column types
			env.writeMigration(t, "001", "create_all_types_table", testutil.SimpleMigration(
				`    m.create_table("types.all_columns", t => {
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
    })`,
				`    m.drop_table("types.all_columns")`,
			))

			client := env.newClient(t, dbURL)

			// Apply migration
			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

			// Verify table exists with correct name (namespace_table format)
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

			// Verify PostgreSQL/CockroachDB column types
			// Note: CockroachDB uses bigint for integer, and real for float in some cases
			assertPgColumnType(t, db, "types_all_columns", "id", "uuid")
			assertPgColumnType(t, db, "types_all_columns", "external_id", "uuid")
			assertPgColumnType(t, db, "types_all_columns", "short_text", "character varying")
			assertPgColumnType(t, db, "types_all_columns", "long_text", "text")
			assertPgColumnTypeContains(t, db, "types_all_columns", "count", "integer", "bigint")
			assertPgColumnTypeContains(t, db, "types_all_columns", "ratio", "double precision", "float", "real")
			assertPgColumnType(t, db, "types_all_columns", "price", "numeric")
			assertPgColumnType(t, db, "types_all_columns", "is_active", "boolean")
			assertPgColumnType(t, db, "types_all_columns", "birth_date", "date")
			assertPgColumnTypeContains(t, db, "types_all_columns", "start_time", "time")
			assertPgColumnTypeContains(t, db, "types_all_columns", "created_at", "timestamp")
			assertPgColumnTypeContains(t, db, "types_all_columns", "metadata", "json")
			assertPgColumnType(t, db, "types_all_columns", "encoded_data", "bytea")
			assertPgColumnTypeContains(t, db, "types_all_columns", "status", "character varying", "text")

			t.Log("All column types test passed!")
		})
	}
}

// TestE2E_Postgres_ColumnModifiers tests all column modifiers.
func TestE2E_Postgres_ColumnModifiers(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with all modifiers
			env.writeMigration(t, "001", "create_modifiers_table", testutil.SimpleMigration(
				`    m.create_table("test.modifiers", t => {
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
    })`,
				`    m.drop_table("test.modifiers")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

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
			assertPgColumnNotNull(t, db, "test_modifiers", "id", true)
			assertPgColumnNotNull(t, db, "test_modifiers", "unique_field", true)
			assertPgColumnNotNull(t, db, "test_modifiers", "optional_field", false)
			assertPgColumnNotNull(t, db, "test_modifiers", "optional_unique", false)
			assertPgColumnNotNull(t, db, "test_modifiers", "optional_default", false)

			// Verify unique constraints exist
			assertPgHasUniqueConstraint(t, db, "test_modifiers", "unique_field")
			assertPgHasUniqueConstraint(t, db, "test_modifiers", "optional_unique")

			// Verify defaults work by inserting a row
			_, err := db.Exec(`INSERT INTO test_modifiers (id, unique_field) VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'test-unique')`)
			if err != nil {
				t.Fatalf("Insert failed: %v", err)
			}

			// Check default values were applied
			var withDefault string
			var countDefault int
			var flagDefault bool
			err = db.QueryRow(`SELECT with_default, count_default, flag_default FROM test_modifiers WHERE id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'`).
				Scan(&withDefault, &countDefault, &flagDefault)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if withDefault != "hello" {
				t.Errorf("Expected with_default='hello', got '%s'", withDefault)
			}
			if countDefault != 0 {
				t.Errorf("Expected count_default=0, got %d", countDefault)
			}
			if !flagDefault {
				t.Errorf("Expected flag_default=true, got %v", flagDefault)
			}

			t.Log("Column modifiers test passed!")
		})
	}
}

// TestE2E_Postgres_SemanticTypes tests all semantic column types.
func TestE2E_Postgres_SemanticTypes(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with semantic types
			env.writeMigration(t, "001", "create_semantic_table", testutil.SimpleMigration(
				`    m.create_table("user.profile", t => {
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
    })`,
				`    m.drop_table("user.profile")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

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
			assertPgColumnTypeContains(t, db, "user_profile", "email", "character varying", "text")
			assertPgColumnTypeContains(t, db, "user_profile", "username", "character varying", "text")
			assertPgColumnType(t, db, "user_profile", "bio", "text")
			assertPgColumnType(t, db, "user_profile", "is_verified", "boolean")
			assertPgColumnType(t, db, "user_profile", "is_premium", "boolean")

			// Verify slug has unique constraint (semantic type property)
			assertPgHasUniqueConstraint(t, db, "user_profile", "profile_slug")

			// Verify flag defaults
			_, err := db.Exec(`INSERT INTO user_profile (id, email, username, password, display_name, job_title, profile_slug, bio, short_bio)
				VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'test@example.com', 'testuser', 'hash', 'Test User', 'Developer', 'test-user', 'Bio text', 'Short bio')`)
			if err != nil {
				t.Fatalf("Insert failed: %v", err)
			}

			var isVerified, isPremium bool
			err = db.QueryRow(`SELECT is_verified, is_premium FROM user_profile WHERE id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12'`).
				Scan(&isVerified, &isPremium)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if isVerified {
				t.Errorf("Expected is_verified=false (default false), got %v", isVerified)
			}
			if !isPremium {
				t.Errorf("Expected is_premium=true (default true), got %v", isPremium)
			}

			t.Log("Semantic types test passed!")
		})
	}
}

// TestE2E_Postgres_HelperMethods tests timestamps(), soft_delete(), and sortable().
func TestE2E_Postgres_HelperMethods(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with helper methods
			env.writeMigration(t, "001", "create_helpers_table", testutil.SimpleMigration(
				`    m.create_table("app.item", t => {
      t.id()
      t.string("name", 100)

      // Helper methods
      t.timestamps()
      t.soft_delete()
      t.sortable()
    })`,
				`    m.drop_table("app.item")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

			testutil.AssertTableExists(t, db, "app_item")

			// Verify columns from helper methods
			testutil.AssertColumnExists(t, db, "app_item", "created_at") // from timestamps()
			testutil.AssertColumnExists(t, db, "app_item", "updated_at") // from timestamps()
			testutil.AssertColumnExists(t, db, "app_item", "deleted_at") // from soft_delete()
			testutil.AssertColumnExists(t, db, "app_item", "position")   // from sortable()

			// Verify types
			// Note: CockroachDB uses bigint for integer in some cases
			assertPgColumnTypeContains(t, db, "app_item", "created_at", "timestamp")
			assertPgColumnTypeContains(t, db, "app_item", "updated_at", "timestamp")
			assertPgColumnTypeContains(t, db, "app_item", "deleted_at", "timestamp")
			assertPgColumnTypeContains(t, db, "app_item", "position", "integer", "bigint")

			// Verify deleted_at is nullable (soft delete)
			assertPgColumnNotNull(t, db, "app_item", "deleted_at", false)

			t.Log("Helper methods test passed!")
		})
	}
}

// TestE2E_Postgres_Relationships tests belongs_to() relationships.
func TestE2E_Postgres_Relationships(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with relationships
			env.writeMigration(t, "001", "create_relationship_tables", testutil.SimpleMigration(
				`    m.create_table("auth.user", t => {
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
    })`,
				`    m.drop_table("blog.comment")
    m.drop_table("blog.post")
    m.drop_table("auth.user")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

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
			assertPgColumnNotNull(t, db, "blog_post", "author_id", true)
			assertPgColumnNotNull(t, db, "blog_post", "editor_id", false)
			assertPgColumnNotNull(t, db, "blog_comment", "post_id", true)
			assertPgColumnNotNull(t, db, "blog_comment", "user_id", false)

			t.Log("Relationships test passed!")
		})
	}
}

// TestE2E_Postgres_Indexes tests index creation.
func TestE2E_Postgres_Indexes(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with various indexes
			env.writeMigration(t, "001", "create_indexed_table", testutil.SimpleMigration(
				`    m.create_table("catalog.product", t => {
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
    m.create_index("catalog.product", ["category_id", "sku"], { unique: true, name: "idx_unique_category_sku" })`,
				`    m.drop_index("idx_product_name")
    m.drop_index("idx_unique_category_sku")
    m.drop_table("catalog.product")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

			testutil.AssertTableExists(t, db, "catalog_product")

			// Verify indexes exist
			assertPgIndexExists(t, db, "catalog_product", "idx_product_name")
			assertPgIndexExists(t, db, "catalog_product", "idx_unique_category_sku")

			t.Log("Indexes test passed!")
		})
	}
}

// TestE2E_Postgres_AddColumn tests adding columns with modifiers.
func TestE2E_Postgres_AddColumn(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Initial table
			env.writeMigration(t, "001", "create_initial_table", testutil.SimpleMigration(
				`    m.create_table("app.user", t => {
      t.id()
      t.email("email")
    })`,
				`    m.drop_table("app.user")`,
			))

			// Add columns with various modifiers
			env.writeMigration(t, "002", "add_columns", testutil.SimpleMigration(
				`    m.add_column("app.user", c => c.string("name", 100))
    m.add_column("app.user", c => c.integer("age").optional())
    m.add_column("app.user", c => c.boolean("is_active").default(true))
    m.add_column("app.user", c => c.string("status", 20).default("pending"))`,
				`    m.drop_column("app.user", "name")
    m.drop_column("app.user", "age")
    m.drop_column("app.user", "is_active")
    m.drop_column("app.user", "status")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

			// Verify new columns
			testutil.AssertColumnExists(t, db, "app_user", "name")
			testutil.AssertColumnExists(t, db, "app_user", "age")
			testutil.AssertColumnExists(t, db, "app_user", "is_active")
			testutil.AssertColumnExists(t, db, "app_user", "status")

			// Verify types
			// Note: CockroachDB uses bigint for integer in some cases
			assertPgColumnTypeContains(t, db, "app_user", "name", "character varying", "text")
			assertPgColumnTypeContains(t, db, "app_user", "age", "integer", "bigint")
			assertPgColumnType(t, db, "app_user", "is_active", "boolean")

			t.Log("Add column test passed!")
		})
	}
}

// TestE2E_Postgres_Rollback tests rollback functionality.
func TestE2E_Postgres_Rollback(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			env.writeMigration(t, "001", "create_table", testutil.SimpleMigration(
				`    m.create_table("test.item", t => {
      t.id()
      t.string("name", 100)
    })`,
				`    m.drop_table("test.item")`,
			))

			env.writeMigration(t, "002", "add_column", testutil.SimpleMigration(
				`    m.add_column("test.item", c => c.text("description").optional())`,
				`    m.drop_column("test.item", "description")`,
			))

			client := env.newClient(t, dbURL)

			// Apply all migrations
			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

			testutil.AssertTableExists(t, db, "test_item")
			testutil.AssertColumnExists(t, db, "test_item", "description")

			// Rollback one migration
			if err := client.MigrationRollback(1, astroladb.Force()); err != nil {
				t.Fatalf("Rollback failed: %v", err)
			}

			// Table should exist but column should be gone
			testutil.AssertTableExists(t, db, "test_item")
			testutil.AssertColumnNotExists(t, db, "test_item", "description")

			// Rollback remaining
			if err := client.MigrationRollback(1, astroladb.Force()); err != nil {
				t.Fatalf("Rollback failed: %v", err)
			}

			testutil.AssertTableNotExists(t, db, "test_item")

			t.Log("Rollback test passed!")
		})
	}
}

// TestE2E_Postgres_TableNaming verifies the namespace_table naming convention.
func TestE2E_Postgres_TableNaming(t *testing.T) {
	for _, setup := range getDatabaseSetups() {
		t.Run(setup.name, func(t *testing.T) {
			db, dbURL := setup.setupFunc(t)
			env := setupTestEnv(t)

			// Migration with various namespaces
			env.writeMigration(t, "001", "create_namespaced_tables", testutil.SimpleMigration(
				`    m.create_table("auth.user", t => {
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
    })`,
				`    m.drop_table("catalog.product")
    m.drop_table("blog.post")
    m.drop_table("auth.session")
    m.drop_table("auth.user")`,
			))

			client := env.newClient(t, dbURL)

			if err := client.MigrationRun(astroladb.Force()); err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

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

			t.Log("Table naming test passed!")
		})
	}
}

// -----------------------------------------------------------------------------
// Helper Functions for PostgreSQL/CockroachDB
// -----------------------------------------------------------------------------

// assertPgColumnType verifies a column's type in PostgreSQL/CockroachDB.
func assertPgColumnType(t *testing.T, db *sql.DB, table, column, expectedType string) {
	t.Helper()

	var dataType string
	err := db.QueryRow(`
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2
	`, table, column).Scan(&dataType)
	if err != nil {
		t.Fatalf("Failed to get column type for %s.%s: %v", table, column, err)
	}

	if !strings.Contains(strings.ToLower(dataType), strings.ToLower(expectedType)) {
		t.Errorf("Column %s.%s: expected type containing %s, got %s", table, column, expectedType, dataType)
	}
}

// assertPgColumnTypeContains verifies a column's type contains one of the expected types.
// This is useful when PostgreSQL and CockroachDB have slight type differences.
func assertPgColumnTypeContains(t *testing.T, db *sql.DB, table, column string, expectedTypes ...string) {
	t.Helper()

	var dataType string
	err := db.QueryRow(`
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2
	`, table, column).Scan(&dataType)
	if err != nil {
		t.Fatalf("Failed to get column type for %s.%s: %v", table, column, err)
	}

	dataTypeLower := strings.ToLower(dataType)
	for _, expected := range expectedTypes {
		if strings.Contains(dataTypeLower, strings.ToLower(expected)) {
			return
		}
	}
	t.Errorf("Column %s.%s: expected type containing one of %v, got %s", table, column, expectedTypes, dataType)
}

// assertPgColumnNotNull verifies a column's NOT NULL constraint.
func assertPgColumnNotNull(t *testing.T, db *sql.DB, table, column string, expectNotNull bool) {
	t.Helper()

	var isNullable string
	err := db.QueryRow(`
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_name = $1 AND column_name = $2
	`, table, column).Scan(&isNullable)
	if err != nil {
		t.Fatalf("Failed to get nullable info for %s.%s: %v", table, column, err)
	}

	isNotNull := isNullable == "NO"
	if isNotNull != expectNotNull {
		t.Errorf("Column %s.%s: expected NOT NULL=%v, got %v (is_nullable=%s)", table, column, expectNotNull, isNotNull, isNullable)
	}
}

// assertPgHasUniqueConstraint verifies a column has a unique constraint.
func assertPgHasUniqueConstraint(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	// Check for unique constraint via information_schema
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.table_schema
		WHERE tc.table_name = $1
			AND ccu.column_name = $2
			AND tc.constraint_type = 'UNIQUE'
	`, table, column).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check unique constraint for %s.%s: %v", table, column, err)
	}

	if count == 0 {
		// Also check for unique index (some DBs use indexes instead of constraints)
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM pg_indexes
			WHERE tablename = $1
				AND indexdef LIKE '%UNIQUE%'
				AND indexdef LIKE '%' || $2 || '%'
		`, table, column).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("Column %s.%s does not have a unique constraint", table, column)
		}
	}
}

// assertPgIndexExists verifies an index exists on a table.
func assertPgIndexExists(t *testing.T, db *sql.DB, table, indexName string) {
	t.Helper()

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE tablename = $1 AND indexname = $2
	`, table, indexName).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check index %s: %v", indexName, err)
	}
	if count == 0 {
		t.Errorf("Index %s does not exist on table %s", indexName, table)
	}
}
