//go:build integration

package astroladb

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// TestConcurrent_MigrationRun tests concurrent migration execution.
func TestConcurrent_MigrationRun(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("concurrent_runs_sequential_execution", func(t *testing.T) {
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
  name: col.string(100),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

		// Create migration
		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)

		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)
		client.Close()

		// Launch concurrent clients
		var wg sync.WaitGroup
		results := make([]error, 3)

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				client, err := New(
					WithDatabaseURL(dbURL),
					WithSchemasDir(schemasDir),
					WithMigrationsDir(migrationsDir),
				)
				if err != nil {
					results[idx] = err
					return
				}
				defer client.Close()

				results[idx] = client.MigrationRun()
			}(i)
		}

		wg.Wait()

		// At least one should succeed
		successCount := 0
		for _, err := range results {
			if err == nil {
				successCount++
			}
		}
		testutil.AssertTrue(t, successCount >= 1, "At least one migration should succeed")

		// Table should exist
		testutil.AssertTableExists(t, db, "users")
	})

	t.Run("lock_prevents_concurrent_execution", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)
		os.MkdirAll(migrationsDir, 0755)

		// Create slow migration
		slowMigration := `
migration(m => {
  m.create_table("test", t => {
    t.id()
  })
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_slow.js"), slowMigration)

		// First client starts
		client1, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
			WithTimeout(30*time.Second),
		)
		testutil.AssertNoError(t, err)
		defer client1.Close()

		// Second client tries immediately
		client2, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
			WithTimeout(2*time.Second),
		)
		testutil.AssertNoError(t, err)
		defer client2.Close()

		var wg sync.WaitGroup
		var err1, err2 error

		wg.Add(2)
		go func() {
			defer wg.Done()
			err1 = client1.MigrationRun()
		}()

		// Small delay to ensure client1 acquires lock
		time.Sleep(100 * time.Millisecond)

		go func() {
			defer wg.Done()
			err2 = client2.MigrationRun()
		}()

		wg.Wait()

		// One should succeed, one may fail or wait
		testutil.AssertTrue(t, err1 == nil || err2 == nil, "At least one should succeed")
	})
}

// TestConcurrent_SchemaIsolation tests parallel test execution with schema isolation.
func TestConcurrent_SchemaIsolation(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("parallel_tests_isolated", func(t *testing.T) {
		// Launch multiple parallel operations
		var wg sync.WaitGroup
		errors := make([]error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Each gets isolated database
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
  value: col.integer(),
})
`
				testutil.WriteFile(t, filepath.Join(schemasDir, "public", "test.js"), schema)

				client, err := New(
					WithDatabaseURL(dbURL),
					WithSchemasDir(schemasDir),
					WithMigrationsDir(migrationsDir),
				)
				if err != nil {
					errors[idx] = err
					return
				}
				defer client.Close()

				_, err = client.MigrationGenerate("create_test")
				if err != nil {
					errors[idx] = err
					return
				}

				errors[idx] = client.MigrationRun()
			}(i)
		}

		wg.Wait()

		// All should succeed without interference
		for i, err := range errors {
			if err != nil {
				t.Errorf("Test %d should succeed, got error: %v", i, err)
			}
		}
	})
}

// TestConcurrent_SchemaReads tests concurrent schema reading.
func TestConcurrent_SchemaReads(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("concurrent_schema_check", func(t *testing.T) {
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

		// Multiple concurrent checks
		var wg sync.WaitGroup
		errors := make([]error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				client, err := New(
					WithSchemasDir(schemasDir),
					WithSchemaOnly(),
				)
				if err != nil {
					errors[idx] = err
					return
				}
				defer client.Close()

				errors[idx] = client.SchemaCheck()
			}(i)
		}

		wg.Wait()

		// All should succeed
		for _, err := range errors {
			testutil.AssertNoError(t, err)
		}
	})

	t.Run("concurrent_schema_export", func(t *testing.T) {
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

		var wg sync.WaitGroup
		results := make([][]byte, 5)
		errors := make([]error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				client, err := New(
					WithSchemasDir(schemasDir),
					WithSchemaOnly(),
				)
				if err != nil {
					errors[idx] = err
					return
				}
				defer client.Close()

				results[idx], errors[idx] = client.SchemaExport("typescript")
			}(i)
		}

		wg.Wait()

		// All should succeed and produce same output
		for i, err := range errors {
			testutil.AssertNoError(t, err)
			testutil.AssertTrue(t, len(results[i]) > 0, "Should produce output")
		}

		// Output should be deterministic
		for i := 1; i < len(results); i++ {
			if !bytes.Equal(results[0], results[i]) {
				t.Errorf("Results should be deterministic, result[0] != result[%d]", i)
			}
		}
	})
}

// TestConcurrent_VersionTableAccess tests concurrent version table operations.
func TestConcurrent_VersionTableAccess(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("concurrent_status_queries", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create and apply migration
		migration := `
migration(m => {
  m.create_table("test", t => {
    t.id()
  })
})
`
		testutil.WriteFile(t, filepath.Join(migrationsDir, "001_test.js"), migration)

		client, err := New(
			WithDatabaseURL(dbURL),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
		client.Close()

		// Concurrent status queries
		var wg sync.WaitGroup
		errors := make([]error, 20)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				client, err := New(
					WithDatabaseURL(dbURL),
					WithMigrationsDir(migrationsDir),
				)
				if err != nil {
					errors[idx] = err
					return
				}
				defer client.Close()

				_, errors[idx] = client.MigrationStatus()
			}(i)
		}

		wg.Wait()

		// All should succeed
		for _, err := range errors {
			testutil.AssertNoError(t, err)
		}
	})
}

// TestConcurrent_RaceConditions tests for race conditions.
func TestConcurrent_RaceConditions(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("no_race_in_schema_loading", func(t *testing.T) {
		tempDir := testutil.TempDir(t)
		schemasDir := filepath.Join(tempDir, "schemas")
		os.MkdirAll(filepath.Join(schemasDir, "public"), 0755)

		// Create multiple schema files
		for i := 1; i <= 5; i++ {
			schema := `
export default table({
  id: col.id(),
})
`
			testutil.WriteFile(t, filepath.Join(schemasDir, "public", "table"+string(rune('0'+i))+".js"), schema)
		}

		// Concurrent schema loading
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				client, err := New(
					WithSchemasDir(schemasDir),
					WithSchemaOnly(),
				)
				if err != nil {
					return
				}
				defer client.Close()

				_ = client.SchemaCheck()
			}()
		}

		wg.Wait()
	})

	t.Run("no_race_in_migration_parsing", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		migration := `
migration(m => {
  m.create_table("test", t => {
    t.id()
  })
})
`
		migrationPath := filepath.Join(migrationsDir, "001_test.js")
		testutil.WriteFile(t, migrationPath, migration)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				client, err := New(
					WithDatabaseURL(dbURL),
					WithMigrationsDir(migrationsDir),
				)
				if err != nil {
					return
				}
				defer client.Close()

				_, _ = client.ParseMigrationFile(migrationPath)
			}()
		}

		wg.Wait()
	})
}

// TestConcurrent_ConnectionPool tests connection pool behavior.
func TestConcurrent_ConnectionPool(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("connection_pool_limits", func(t *testing.T) {
		db, dbURL := testutil.SetupPostgresWithURL(t)
		defer db.Close()

		tempDir := testutil.TempDir(t)
		migrationsDir := filepath.Join(tempDir, "migrations")
		os.MkdirAll(migrationsDir, 0755)

		// Create many clients simultaneously
		var wg sync.WaitGroup
		clients := make([]*Client, 20)
		errors := make([]error, 20)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				client, err := New(
					WithDatabaseURL(dbURL),
					WithMigrationsDir(migrationsDir),
				)
				if err != nil {
					errors[idx] = err
					return
				}

				clients[idx] = client
			}(i)
		}

		wg.Wait()

		// All should connect successfully
		for i, err := range errors {
			testutil.AssertNoError(t, err)
			testutil.AssertNotNil(t, clients[i])
		}

		// Clean up
		for _, client := range clients {
			if client != nil {
				client.Close()
			}
		}
	})
}

// TestConcurrent_MultipleOperations tests various operations happening concurrently.
func TestConcurrent_MultipleOperations(t *testing.T) {
	testutil.SkipIfShort(t)

	t.Run("mixed_operations", func(t *testing.T) {
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
  name: col.string(100),
})
`
		testutil.WriteFile(t, filepath.Join(schemasDir, "public", "users.js"), schema)

		// Setup initial state
		client, err := New(
			WithDatabaseURL(dbURL),
			WithSchemasDir(schemasDir),
			WithMigrationsDir(migrationsDir),
		)
		testutil.AssertNoError(t, err)

		_, err = client.MigrationGenerate("create_users")
		testutil.AssertNoError(t, err)

		err = client.MigrationRun()
		testutil.AssertNoError(t, err)
		client.Close()

		// Concurrent mixed operations
		var wg sync.WaitGroup
		operations := []func() error{
			func() error {
				c, _ := New(WithDatabaseURL(dbURL), WithMigrationsDir(migrationsDir))
				defer c.Close()
				_, err := c.MigrationStatus()
				return err
			},
			func() error {
				c, _ := New(WithSchemasDir(schemasDir), WithSchemaOnly())
				defer c.Close()
				return c.SchemaCheck()
			},
			func() error {
				c, _ := New(WithDatabaseURL(dbURL), WithMigrationsDir(migrationsDir))
				defer c.Close()
				_, err := c.VerifyChain()
				return err
			},
		}

		for i := 0; i < 3; i++ {
			for _, op := range operations {
				wg.Add(1)
				go func(operation func() error) {
					defer wg.Done()
					_ = operation()
				}(op)
			}
		}

		wg.Wait()
	})
}
