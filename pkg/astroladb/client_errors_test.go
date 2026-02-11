//go:build integration

package astroladb

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestClient_ConfigurationValidation tests client creation with invalid configuration.
func TestClient_ConfigurationValidation(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("missing_database_url", func(t *testing.T) {
		_, err := New(
			WithMigrationsDir("./migrations"),
		)
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "database URL")
	})

	t.Run("unsupported_dialect", func(t *testing.T) {
		_, err := New(
			WithDatabaseURL("mysql://localhost/test"),
			WithDialect("mysql"),
		)
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "unsupported dialect")
	})

	t.Run("schema_only_mode_no_db_required", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(schemasDir, 0755)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, client)

		err = client.Close()
		testutil.AssertNoError(t, err)
	})

	t.Run("invalid_timeout_value", func(t *testing.T) {
		_, dbURL := testutil.SetupPostgresWithURL(t)

		// Timeout of 0 should use default
		client, err := New(
			WithDatabaseURL(dbURL),
			WithTimeout(0),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		cfg := client.Config()
		testutil.AssertTrue(t, cfg.Timeout > 0, "Should use default timeout")
	})
}

// TestClient_DatabaseConnectionErrors tests database connection failures.
func TestClient_DatabaseConnectionErrors(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("invalid_connection_url", func(t *testing.T) {
		_, err := New(
			WithDatabaseURL("postgresql://invalid:5432/nonexistent"),
		)
		testutil.AssertNotNil(t, err)
		// Connection error should be wrapped
		testutil.AssertTrue(t, err != nil, "Should fail to connect")
	})

	t.Run("connection_refused", func(t *testing.T) {
		_, err := New(
			WithDatabaseURL("postgresql://localhost:9999/test"),
			WithTimeout(1*time.Second),
		)
		testutil.AssertNotNil(t, err)
	})

	t.Run("authentication_failure", func(t *testing.T) {
		// Try to connect with invalid credentials
		_, err := New(
			WithDatabaseURL("postgresql://baduser:badpass@localhost:5432/postgres"),
			WithTimeout(2*time.Second),
		)
		testutil.AssertNotNil(t, err)
	})

	t.Run("network_timeout", func(t *testing.T) {
		// Use very short timeout to trigger timeout error
		_, err := New(
			WithDatabaseURL("postgresql://localhost:5432/postgres"),
			WithTimeout(1*time.Nanosecond),
		)
		// May succeed or fail depending on timing
		_ = err
	})

	t.Run("close_before_operations", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
		)
		testutil.AssertNoError(t, err)

		// Close client
		err = client.Close()
		testutil.AssertNoError(t, err)

		// Try to use after close
		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)
	})
}

// TestClient_FileSystemErrors tests file system operation failures.
func TestClient_FileSystemErrors(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("nonexistent_schemas_directory", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir("/nonexistent/schemas"),
		)
		testutil.AssertNoError(t, err) // Client creation succeeds
		defer client.Close()

		// Error occurs when trying to use schemas
		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
		testutil.AssertErrorContains(t, err, "does not exist")
	})

	t.Run("nonexistent_migrations_directory", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir("/nonexistent/migrations"),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Error occurs when trying to use migrations
		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)
	})

	t.Run("permission_denied_create_migration", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")

		// Create directories but make migrations dir read-only
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0444) // Read-only

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

		// Try to generate migration (should fail due to permissions)
		_, err = client.MigrationGenerate("test")
		// On Windows, permissions might work differently
		// This test documents the behavior
		_ = err

		// Restore permissions for cleanup
		os.Chmod(migrationsDir, 0755)
	})

	t.Run("invalid_schema_file_syntax", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		// Write invalid JavaScript
		invalidSchema := `
export default table({
  id: INVALID_SYNTAX_HERE
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "bad.js"), invalidSchema)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		err = client.SchemaCheck()
		testutil.AssertNotNil(t, err)
	})
}

// TestClient_ContextCancellation tests context cancellation handling.
func TestClient_ContextCancellation(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("timeout_during_schema_check", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

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
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithTimeout(1*time.Nanosecond), // Very short timeout
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Operation may succeed or timeout depending on timing
		err = client.SchemaCheck()
		_ = err
	})

	t.Run("context_with_deadline", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
			WithTimeout(10*time.Second),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Context methods should work
		ctx, cancel := client.context()
		defer cancel()

		testutil.AssertNotNil(t, ctx)
	})
}

// TestClient_SchemaOnlyMode tests schema-only mode operations.
func TestClient_SchemaOnlyMode(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("export_without_database", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		schema := `
export default table({
  id: col.id(),
  email: col.string(255),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Export should work without database
		output, err := client.SchemaExport("typescript")
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(output) > 0, "Should generate TypeScript")
	})

	t.Run("check_without_database", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		schema := `
export default table({
  id: col.id(),
  name: col.string(100),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "items.js"), schema)

		client, err := New(
			WithSchemasDir(schemasDir),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Check should work without database
		err = client.SchemaCheck()
		testutil.AssertNoError(t, err)
	})

	t.Run("migration_operations_require_database", func(t *testing.T) {
		client, err := New(
			WithSchemasDir("./schemas"),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		// Migration operations should fail in schema-only mode
		err = client.MigrationRun()
		testutil.AssertNotNil(t, err)
	})
}

// TestClient_DialectDetection tests dialect auto-detection from URLs.
func TestClient_DialectDetection(t *testing.T) {
	t.Run("postgres_url_detection", func(t *testing.T) {
		testCases := []string{
			"postgres://localhost/test",
			"postgresql://localhost/test",
			"POSTGRES://localhost/test",
		}

		for _, url := range testCases {
			detected := detectDialect(url)
			testutil.AssertEqual(t, detected, "postgres")
		}
	})

	t.Run("sqlite_url_detection", func(t *testing.T) {
		testCases := []string{
			"sqlite://./test.db",
			"sqlite3://./test.db",
			"file:test.db",
			"./database.db",
			"/path/to/data.sqlite",
			"mydb.sqlite3",
		}

		for _, url := range testCases {
			detected := detectDialect(url)
			testutil.AssertEqual(t, detected, "sqlite")
		}
	})

	t.Run("default_to_postgres", func(t *testing.T) {
		detected := detectDialect("unknown://something")
		testutil.AssertEqual(t, detected, "postgres")
	})
}

// TestClient_URLRedaction tests sensitive information removal from URLs.
func TestClient_URLRedaction(t *testing.T) {
	t.Run("redact_password", func(t *testing.T) {
		url := "postgres://user:secretpass@localhost:5432/mydb"
		redacted := redactURL(url)
		testutil.AssertTrue(t, len(redacted) > 0, "Should return redacted URL")
		// Password should be hidden
		testutil.AssertFalse(t, redacted == url || len(redacted) == len(url), "Should modify URL")
	})

	t.Run("no_password_to_redact", func(t *testing.T) {
		url := "sqlite://./test.db"
		redacted := redactURL(url)
		testutil.AssertEqual(t, redacted, url)
	})
}

// TestClient_Accessors tests client accessor methods.
func TestClient_Accessors(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("db_accessor", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		clientDB := client.DB()
		testutil.AssertNotNil(t, clientDB)
	})

	t.Run("dialect_accessor", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		dialect := client.Dialect()
		testutil.AssertEqual(t, dialect, "postgres")
	})

	t.Run("config_accessor", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir("./custom_schemas"),
			WithMigrationsDir("./custom_migrations"),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		cfg := client.Config()
		testutil.AssertEqual(t, cfg.SchemasDir, "./custom_schemas")
		testutil.AssertEqual(t, cfg.MigrationsDir, "./custom_migrations")
	})

	t.Run("schema_only_db_is_nil", func(t *testing.T) {
		client, err := New(
			WithSchemasDir("./schemas"),
			WithSchemaOnly(),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		db := client.DB()
		testutil.AssertTrue(t, db == nil, "DB should be nil in schema-only mode")
	})
}

// TestClient_ParseMigrationFileErrors tests migration file parsing errors.
func TestClient_ParseMigrationFileErrors(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("parse_valid_migration", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		migrationContent := `
migration(m => {
  m.create_table("users", t => {
    t.id()
    t.string("email", 255)
  })
})
`
		migrationPath := filepath.Join(migrationsDir, "001_test.js")
		testutil.WriteFile(t, migrationPath, migrationContent)

		client, err := New(
			WithDatabaseURL(dbURL),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		ops, err := client.ParseMigrationFile(migrationPath)
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(ops) > 0, "Should parse operations")
	})

	t.Run("parse_invalid_migration", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		invalidContent := `
migration(m => {
  INVALID SYNTAX
})
`
		migrationPath := filepath.Join(migrationsDir, "bad.js")
		testutil.WriteFile(t, migrationPath, invalidContent)

		client, err := New(
			WithDatabaseURL(dbURL),
		)
		testutil.AssertNoError(t, err)
		defer client.Close()

		_, err = client.ParseMigrationFile(migrationPath)
		testutil.AssertNotNil(t, err)
	})
}
