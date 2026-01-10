//go:build integration

// Package testutil provides database integration test helpers.
// This file contains database setup and assertion functions that require
// running database containers (see docker-compose.test.yml).
package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	// Import database drivers
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// Default database connection strings (from docker-compose.test.yml)
const (
	defaultPostgresURL = "postgres://alab:alab@localhost:5432/alab_test?sslmode=disable"
)

// SetupPostgres connects to a PostgreSQL test database.
// It reads the connection string from POSTGRES_URL environment variable,
// or uses the default from docker-compose.test.yml.
// The connection is automatically closed when the test completes.
//
// Each test gets its own PostgreSQL schema to enable parallel test execution
// across packages without interference. The schema name is appended to the
// connection string's search_path so all connections use it.
//
// Returns the database connection. Use PostgresURL(t) if you need a URL
// that uses the test schema.
func SetupPostgres(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		url = defaultPostgresURL
	}

	// First, connect to create the schema
	setupDB, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("failed to open postgres connection: %v", err)
	}

	// Verify connection works
	if err := setupDB.Ping(); err != nil {
		setupDB.Close()
		t.Fatalf("failed to ping postgres: %v\n\nIs the database running? Start it with:\n  docker-compose -f docker-compose.test.yml up -d", err)
	}

	// Generate a unique schema name
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		setupDB.Close()
		t.Fatalf("failed to generate random schema name: %v", err)
	}
	schemaName := "test_" + hex.EncodeToString(randomBytes)

	// Create the schema
	_, err = setupDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
	if err != nil {
		setupDB.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}
	setupDB.Close()

	// Now connect with ONLY the test schema in the search_path
	// IMPORTANT: Do NOT include 'public' - this ensures complete isolation
	separator := "&"
	if !strings.Contains(url, "?") {
		separator = "?"
	}
	urlWithSchema := fmt.Sprintf("%s%ssearch_path=%s", url, separator, schemaName)

	db, err := sql.Open("postgres", urlWithSchema)
	if err != nil {
		t.Fatalf("failed to open postgres connection with schema: %v", err)
	}

	// Limit to single connection to ensure schema consistency
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping postgres with schema: %v", err)
	}

	// Clean up when test completes
	t.Cleanup(func() {
		db.Close()
		// Reconnect to drop the schema
		cleanupDB, err := sql.Open("postgres", url)
		if err == nil {
			_, _ = cleanupDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
			cleanupDB.Close()
		}
	})

	return db
}

// SetupPostgresWithURL connects to a PostgreSQL test database and returns both
// the database connection and a URL that includes the test schema.
// Use this when you need to pass a database URL to an external client that
// should operate in the same test schema.
func SetupPostgresWithURL(t *testing.T) (*sql.DB, string) {
	t.Helper()

	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		url = defaultPostgresURL
	}

	// First, connect to create the schema
	setupDB, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("failed to open postgres connection: %v", err)
	}

	// Verify connection works
	if err := setupDB.Ping(); err != nil {
		setupDB.Close()
		t.Fatalf("failed to ping postgres: %v\n\nIs the database running? Start it with:\n  docker-compose -f docker-compose.test.yml up -d", err)
	}

	// Generate a unique schema name
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		setupDB.Close()
		t.Fatalf("failed to generate random schema name: %v", err)
	}
	schemaName := "test_" + hex.EncodeToString(randomBytes)

	// Create the schema
	_, err = setupDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
	if err != nil {
		setupDB.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}
	setupDB.Close()

	// Build URL with ONLY the test schema in search_path
	// IMPORTANT: Do NOT include 'public' - this prevents interference from
	// leftover tables in public schema and ensures complete isolation
	separator := "&"
	if !strings.Contains(url, "?") {
		separator = "?"
	}
	urlWithSchema := fmt.Sprintf("%s%ssearch_path=%s", url, separator, schemaName)

	db, err := sql.Open("postgres", urlWithSchema)
	if err != nil {
		t.Fatalf("failed to open postgres connection with schema: %v", err)
	}

	// Limit to single connection to ensure schema consistency
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping postgres with schema: %v", err)
	}

	// Clean up when test completes
	t.Cleanup(func() {
		db.Close()
		// Reconnect to drop the schema
		cleanupDB, err := sql.Open("postgres", url)
		if err == nil {
			_, _ = cleanupDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
			cleanupDB.Close()
		}
	})

	return db, urlWithSchema
}

// PostgresTestURL returns a PostgreSQL connection URL that uses an isolated test schema.
// The schema is automatically created and cleaned up when the test completes.
// Use this when you only need a URL (not a direct db connection) to pass to a client.
func PostgresTestURL(t *testing.T) string {
	t.Helper()
	_, url := SetupPostgresWithURL(t)
	return url
}

// SetupSQLite creates an in-memory SQLite database for testing.
// The connection is automatically closed when the test completes.
func SetupSQLite(t *testing.T) *sql.DB {
	t.Helper()

	// Use :memory: for in-memory database
	// Add mode=memory&cache=shared to allow multiple connections to same db
	db, err := sql.Open("sqlite", ":memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open sqlite connection: %v", err)
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping sqlite: %v", err)
	}

	// Enable foreign keys (disabled by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Clean up when test completes
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// SetupSQLiteFile creates a file-based SQLite database for testing.
// The file is automatically removed when the test completes.
func SetupSQLiteFile(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("failed to open sqlite file: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping sqlite: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	// Clean up when test completes
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})

	return db
}

// AssertTableExists checks that a table exists in the database.
// Works with PostgreSQL and SQLite.
func AssertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	exists, err := tableExists(db, table)
	if err != nil {
		t.Fatalf("failed to check if table exists: %v", err)
	}

	if !exists {
		t.Errorf("expected table %q to exist, but it does not", table)
	}
}

// AssertTableNotExists checks that a table does not exist in the database.
func AssertTableNotExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	exists, err := tableExists(db, table)
	if err != nil {
		t.Fatalf("failed to check if table exists: %v", err)
	}

	if exists {
		t.Errorf("expected table %q to not exist, but it does", table)
	}
}

// tableExists checks if a table exists in the database.
func tableExists(db *sql.DB, table string) (bool, error) {
	// Try a simple query that works across all databases
	// This is more reliable than querying information_schema
	query := "SELECT 1 FROM " + table + " LIMIT 0"
	_, err := db.Exec(query)
	if err != nil {
		// Check if the error is "table doesn't exist"
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "does not exist") ||
			strings.Contains(errStr, "doesn't exist") ||
			strings.Contains(errStr, "no such table") {
			return false, nil
		}
		// Some other error
		return false, err
	}
	return true, nil
}

// AssertColumnExists checks that a column exists in a table.
// Works with PostgreSQL and SQLite.
func AssertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	exists, err := columnExists(db, table, column)
	if err != nil {
		t.Fatalf("failed to check if column exists: %v", err)
	}

	if !exists {
		t.Errorf("expected column %q to exist in table %q, but it does not", column, table)
	}
}

// AssertColumnNotExists checks that a column does not exist in a table.
func AssertColumnNotExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	exists, err := columnExists(db, table, column)
	if err != nil {
		t.Fatalf("failed to check if column exists: %v", err)
	}

	if exists {
		t.Errorf("expected column %q to not exist in table %q, but it does", column, table)
	}
}

// columnExists checks if a column exists in a table.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	// Try selecting the column - this works across all databases
	query := "SELECT " + column + " FROM " + table + " LIMIT 0"
	_, err := db.Exec(query)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "column") ||
			strings.Contains(errStr, "no such column") ||
			strings.Contains(errStr, "unknown column") {
			return false, nil
		}
		// If table doesn't exist, treat column as not existing
		if strings.Contains(errStr, "does not exist") ||
			strings.Contains(errStr, "doesn't exist") ||
			strings.Contains(errStr, "no such table") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AssertIndexExists checks that an index exists on a table.
// Works with PostgreSQL and SQLite.
func AssertIndexExists(t *testing.T, db *sql.DB, table, index string) {
	t.Helper()

	exists, err := indexExists(db, table, index)
	if err != nil {
		t.Fatalf("failed to check if index exists: %v", err)
	}

	if !exists {
		t.Errorf("expected index %q to exist on table %q, but it does not", index, table)
	}
}

// indexExists checks if an index exists on a table.
// Detects the database type and uses appropriate query.
func indexExists(db *sql.DB, table, index string) (bool, error) {
	// Try PostgreSQL query first
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = $1 AND indexname = $2
		)
	`, table, index).Scan(&exists)
	if err == nil {
		return exists, nil
	}

	// Try SQLite query
	var name string
	err = db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND tbl_name = ? AND name = ?
	`, table, index).Scan(&name)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}

	return false, err
}

// ExecSQL executes a SQL statement and fails the test on error.
func ExecSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()

	_, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("failed to execute SQL:\n%s\nerror: %v", query, err)
	}
}

// QuerySQL executes a query and returns the result rows.
// The rows are automatically closed when the test completes.
func QuerySQL(t *testing.T, db *sql.DB, query string, args ...any) *sql.Rows {
	t.Helper()

	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatalf("failed to query SQL:\n%s\nerror: %v", query, err)
	}

	t.Cleanup(func() {
		rows.Close()
	})

	return rows
}

// AssertRowCount checks that a table has the expected number of rows.
func AssertRowCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows in %s: %v", table, err)
	}

	if count != expected {
		t.Errorf("expected %d rows in %s, got %d", expected, table, count)
	}
}
