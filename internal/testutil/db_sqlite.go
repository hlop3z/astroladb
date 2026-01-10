//go:build sqlite

// Package testutil provides SQLite-specific test helpers.
// Use build tag 'sqlite' to run only SQLite tests:
//
//	go test ./... -tags=sqlite
package testutil

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

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

// AssertTableExists checks that a table exists in the SQLite database.
func AssertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`, table).Scan(&name)
	if err == sql.ErrNoRows {
		t.Errorf("expected table %q to exist, but it does not", table)
		return
	}
	if err != nil {
		t.Fatalf("failed to check if table exists: %v", err)
	}
}

// AssertTableNotExists checks that a table does not exist in the SQLite database.
func AssertTableNotExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`, table).Scan(&name)
	if err == sql.ErrNoRows {
		return // Table does not exist, as expected
	}
	if err != nil {
		t.Fatalf("failed to check if table exists: %v", err)
	}
	t.Errorf("expected table %q to not exist, but it does", table)
}

// AssertColumnExists checks that a column exists in a SQLite table.
func AssertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("failed to get table info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("failed to scan column info: %v", err)
		}
		if name == column {
			return // Column exists
		}
	}

	t.Errorf("expected column %q to exist in table %q, but it does not", column, table)
}

// AssertColumnNotExists checks that a column does not exist in a SQLite table.
func AssertColumnNotExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("failed to get table info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("failed to scan column info: %v", err)
		}
		if name == column {
			t.Errorf("expected column %q to not exist in table %q, but it does", column, table)
			return
		}
	}
}

// AssertIndexExists checks that an index exists on a SQLite table.
func AssertIndexExists(t *testing.T, db *sql.DB, table, index string) {
	t.Helper()

	var name string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND tbl_name = ? AND name = ?
	`, table, index).Scan(&name)
	if err == sql.ErrNoRows {
		t.Errorf("expected index %q to exist on table %q, but it does not", index, table)
		return
	}
	if err != nil {
		t.Fatalf("failed to check if index exists: %v", err)
	}
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
