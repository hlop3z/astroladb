//go:build integration

package testutil

import (
	"testing"
)

func TestSetupPostgres(t *testing.T) {
	db := SetupPostgres(t)
	if db == nil {
		t.Fatal("SetupPostgres returned nil")
	}

	// Test basic query
	var result int
	err := db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %d", result)
	}
}

func TestSetupSQLite(t *testing.T) {
	db := SetupSQLite(t)
	if db == nil {
		t.Fatal("SetupSQLite returned nil")
	}

	// Test basic query
	var result int
	err := db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1, got %d", result)
	}
}

func TestAssertTableExists_Integration(t *testing.T) {
	db := SetupSQLite(t)

	// Create a test table
	_, err := db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Should not fail
	AssertTableExists(t, db, "test_table")

	// Should fail for non-existent table (but we can't easily test this)
}

func TestAssertColumnExists_Integration(t *testing.T) {
	db := SetupSQLite(t)

	// Create a test table
	_, err := db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Should not fail
	AssertColumnExists(t, db, "test_table", "id")
	AssertColumnExists(t, db, "test_table", "name")
}

func TestExecSQL_Integration(t *testing.T) {
	db := SetupSQLite(t)

	// Should not fail
	ExecSQL(t, db, "CREATE TABLE test_table (id INTEGER PRIMARY KEY)")

	// Verify table was created
	AssertTableExists(t, db, "test_table")
}

func TestAssertRowCount_Integration(t *testing.T) {
	db := SetupSQLite(t)

	// Create and populate table
	ExecSQL(t, db, "CREATE TABLE test_table (id INTEGER PRIMARY KEY)")
	ExecSQL(t, db, "INSERT INTO test_table (id) VALUES (1), (2), (3)")

	// Should not fail
	AssertRowCount(t, db, "test_table", 3)
}
