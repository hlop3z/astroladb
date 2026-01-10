//go:build integration

package engine_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// =============================================================================
// FK Constraint Violation Tests
// =============================================================================

func TestError_FKViolation_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	// Create parent and child tables with FK
	testutil.ExecSQL(t, db, `CREATE TABLE parent (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE child (
		id UUID PRIMARY KEY,
		parent_id UUID REFERENCES parent(id)
	)`)

	// Try to insert child with non-existent parent - should fail
	_, err := db.Exec(`INSERT INTO child (id, parent_id) VALUES ('11111111-1111-1111-1111-111111111111', '99999999-9999-9999-9999-999999999999')`)
	if err == nil {
		t.Fatal("expected FK violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "foreign key") && !strings.Contains(errStr, "violates") {
		t.Fatalf("expected FK violation error, got: %v", err)
	}
}

func TestError_FKViolation_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	// Create parent and child tables with FK
	testutil.ExecSQL(t, db, `CREATE TABLE parent (id TEXT PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE child (
		id TEXT PRIMARY KEY,
		parent_id TEXT REFERENCES parent(id)
	)`)

	// Try to insert child with non-existent parent - should fail
	_, err := db.Exec(`INSERT INTO child (id, parent_id) VALUES ('11111111-1111-1111-1111-111111111111', '99999999-9999-9999-9999-999999999999')`)
	if err == nil {
		t.Fatal("expected FK violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "foreign key") && !strings.Contains(errStr, "constraint") {
		t.Fatalf("expected FK violation error, got: %v", err)
	}
}

func TestError_DropTableWithDependents_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	// Create parent and child tables with FK
	testutil.ExecSQL(t, db, `CREATE TABLE parent (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE child (
		id UUID PRIMARY KEY,
		parent_id UUID REFERENCES parent(id)
	)`)

	// Try to drop parent table without CASCADE - should fail
	_, err := db.Exec(`DROP TABLE parent`)
	if err == nil {
		t.Fatal("expected dependency error when dropping table with dependents, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "depend") && !strings.Contains(errStr, "constraint") {
		t.Fatalf("expected dependency error, got: %v", err)
	}
}

// =============================================================================
// Unique Constraint Violation Tests
// =============================================================================

func TestError_UniqueViolation_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255) UNIQUE)`)
	testutil.ExecSQL(t, db, `INSERT INTO users (id, email) VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com')`)

	// Try to insert duplicate email - should fail
	_, err := db.Exec(`INSERT INTO users (id, email) VALUES ('22222222-2222-2222-2222-222222222222', 'test@example.com')`)
	if err == nil {
		t.Fatal("expected unique violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "unique") && !strings.Contains(errStr, "duplicate") {
		t.Fatalf("expected unique violation error, got: %v", err)
	}
}

func TestError_UniqueViolation_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT UNIQUE)`)
	testutil.ExecSQL(t, db, `INSERT INTO users (id, email) VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com')`)

	// Try to insert duplicate email - should fail
	_, err := db.Exec(`INSERT INTO users (id, email) VALUES ('22222222-2222-2222-2222-222222222222', 'test@example.com')`)
	if err == nil {
		t.Fatal("expected unique violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "unique") && !strings.Contains(errStr, "constraint") {
		t.Fatalf("expected unique violation error, got: %v", err)
	}
}

func TestError_CompositeUniqueViolation_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE followers (
		id UUID PRIMARY KEY,
		follower_id UUID NOT NULL,
		following_id UUID NOT NULL,
		UNIQUE(follower_id, following_id)
	)`)
	testutil.ExecSQL(t, db, `INSERT INTO followers (id, follower_id, following_id) VALUES
		('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')`)

	// Try to insert duplicate follower/following pair - should fail
	_, err := db.Exec(`INSERT INTO followers (id, follower_id, following_id) VALUES
		('22222222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')`)
	if err == nil {
		t.Fatal("expected unique violation error for composite key, got nil")
	}
}

// =============================================================================
// NOT NULL Violation Tests
// =============================================================================

func TestError_NotNullViolation_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255) NOT NULL)`)

	// Try to insert NULL email - should fail
	_, err := db.Exec(`INSERT INTO users (id, email) VALUES ('11111111-1111-1111-1111-111111111111', NULL)`)
	if err == nil {
		t.Fatal("expected NOT NULL violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "null") && !strings.Contains(errStr, "violates") {
		t.Fatalf("expected NOT NULL violation error, got: %v", err)
	}
}

func TestError_NotNullViolation_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT NOT NULL)`)

	// Try to insert NULL email - should fail
	_, err := db.Exec(`INSERT INTO users (id, email) VALUES ('11111111-1111-1111-1111-111111111111', NULL)`)
	if err == nil {
		t.Fatal("expected NOT NULL violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "null") && !strings.Contains(errStr, "constraint") && !strings.Contains(errStr, "not null") {
		t.Fatalf("expected NOT NULL violation error, got: %v", err)
	}
}

// =============================================================================
// Invalid Operations Tests
// =============================================================================

func TestError_DropNonExistentTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	// Try to drop non-existent table
	_, err := db.Exec(`DROP TABLE nonexistent_table`)
	if err == nil {
		t.Fatal("expected error when dropping non-existent table, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "does not exist") && !strings.Contains(errStr, "not exist") {
		t.Fatalf("expected 'does not exist' error, got: %v", err)
	}
}

func TestError_DropNonExistentTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	// Try to drop non-existent table
	_, err := db.Exec(`DROP TABLE nonexistent_table`)
	if err == nil {
		t.Fatal("expected error when dropping non-existent table, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "no such table") {
		t.Fatalf("expected 'no such table' error, got: %v", err)
	}
}

func TestError_DropNonExistentColumn_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY)`)

	// Try to drop non-existent column
	_, err := db.Exec(`ALTER TABLE users DROP COLUMN nonexistent_col`)
	if err == nil {
		t.Fatal("expected error when dropping non-existent column, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "does not exist") && !strings.Contains(errStr, "column") {
		t.Fatalf("expected column 'does not exist' error, got: %v", err)
	}
}

func TestError_AddColumnToNonExistentTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	// Try to add column to non-existent table
	_, err := db.Exec(`ALTER TABLE nonexistent_table ADD COLUMN email VARCHAR(255)`)
	if err == nil {
		t.Fatal("expected error when adding column to non-existent table, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "does not exist") && !strings.Contains(errStr, "relation") {
		t.Fatalf("expected 'does not exist' error, got: %v", err)
	}
}

func TestError_AddColumnToNonExistentTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	// Try to add column to non-existent table
	_, err := db.Exec(`ALTER TABLE nonexistent_table ADD COLUMN email TEXT`)
	if err == nil {
		t.Fatal("expected error when adding column to non-existent table, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "no such table") {
		t.Fatalf("expected 'no such table' error, got: %v", err)
	}
}

func TestError_CreateDuplicateTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY)`)

	// Try to create table that already exists
	_, err := db.Exec(`CREATE TABLE users (id UUID PRIMARY KEY)`)
	if err == nil {
		t.Fatal("expected error when creating duplicate table, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "already exists") && !strings.Contains(errStr, "relation") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestError_CreateDuplicateTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY)`)

	// Try to create table that already exists
	_, err := db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY)`)
	if err == nil {
		t.Fatal("expected error when creating duplicate table, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

// =============================================================================
// Type Mismatch Tests
// =============================================================================

func TestError_TypeMismatch_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE products (id UUID PRIMARY KEY, price INTEGER NOT NULL)`)

	// Try to insert string into integer column - should fail
	_, err := db.Exec(`INSERT INTO products (id, price) VALUES ('11111111-1111-1111-1111-111111111111', 'not_a_number')`)
	if err == nil {
		t.Fatal("expected type mismatch error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "integer") && !strings.Contains(errStr, "invalid input") && !strings.Contains(errStr, "syntax") {
		t.Fatalf("expected type mismatch error, got: %v", err)
	}
}

func TestError_InvalidUUID_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY)`)

	// Try to insert invalid UUID - should fail
	_, err := db.Exec(`INSERT INTO users (id) VALUES ('not-a-valid-uuid')`)
	if err == nil {
		t.Fatal("expected invalid UUID error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "uuid") && !strings.Contains(errStr, "invalid") && !strings.Contains(errStr, "syntax") {
		t.Fatalf("expected invalid UUID error, got: %v", err)
	}
}

func TestError_InvalidDate_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE events (id UUID PRIMARY KEY, event_date DATE NOT NULL)`)

	// Try to insert invalid date - should fail
	_, err := db.Exec(`INSERT INTO events (id, event_date) VALUES ('11111111-1111-1111-1111-111111111111', '2023-13-45')`)
	if err == nil {
		t.Fatal("expected invalid date error, got nil")
	}
}

// =============================================================================
// Large Schema Operations Tests
// =============================================================================

func TestLargeTable_ManyColumns_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	// Build a table with 50+ columns
	var columnDefs []string
	columnDefs = append(columnDefs, "id UUID PRIMARY KEY")
	for i := 1; i <= 60; i++ {
		columnDefs = append(columnDefs, fmt.Sprintf("col_%d VARCHAR(255)", i))
	}

	createSQL := fmt.Sprintf("CREATE TABLE large_table (%s)", strings.Join(columnDefs, ", "))
	testutil.ExecSQL(t, db, createSQL)

	// Verify table was created successfully
	testutil.AssertTableExists(t, db, "large_table")
	testutil.AssertColumnExists(t, db, "large_table", "id")
	testutil.AssertColumnExists(t, db, "large_table", "col_1")
	testutil.AssertColumnExists(t, db, "large_table", "col_60")

	// Insert a row to verify table works
	var insertValues []string
	insertValues = append(insertValues, "'11111111-1111-1111-1111-111111111111'")
	for i := 1; i <= 60; i++ {
		insertValues = append(insertValues, fmt.Sprintf("'value_%d'", i))
	}
	insertSQL := fmt.Sprintf("INSERT INTO large_table VALUES (%s)", strings.Join(insertValues, ", "))
	testutil.ExecSQL(t, db, insertSQL)

	testutil.AssertRowCount(t, db, "large_table", 1)
}

func TestLargeTable_ManyColumns_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	// Build a table with 50+ columns
	var columnDefs []string
	columnDefs = append(columnDefs, "id TEXT PRIMARY KEY")
	for i := 1; i <= 60; i++ {
		columnDefs = append(columnDefs, fmt.Sprintf("col_%d TEXT", i))
	}

	createSQL := fmt.Sprintf("CREATE TABLE large_table (%s)", strings.Join(columnDefs, ", "))
	testutil.ExecSQL(t, db, createSQL)

	// Verify table was created successfully
	testutil.AssertTableExists(t, db, "large_table")
	testutil.AssertColumnExists(t, db, "large_table", "id")
	testutil.AssertColumnExists(t, db, "large_table", "col_1")
	testutil.AssertColumnExists(t, db, "large_table", "col_60")

	// Insert a row to verify table works
	var insertValues []string
	insertValues = append(insertValues, "'11111111-1111-1111-1111-111111111111'")
	for i := 1; i <= 60; i++ {
		insertValues = append(insertValues, fmt.Sprintf("'value_%d'", i))
	}
	insertSQL := fmt.Sprintf("INSERT INTO large_table VALUES (%s)", strings.Join(insertValues, ", "))
	testutil.ExecSQL(t, db, insertSQL)

	testutil.AssertRowCount(t, db, "large_table", 1)
}

// =============================================================================
// Check Constraint Violation Tests (where supported)
// =============================================================================

func TestError_CheckConstraintViolation_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE products (
		id UUID PRIMARY KEY,
		price DECIMAL(10,2) NOT NULL CHECK (price >= 0)
	)`)

	// Try to insert negative price - should fail
	_, err := db.Exec(`INSERT INTO products (id, price) VALUES ('11111111-1111-1111-1111-111111111111', -10.00)`)
	if err == nil {
		t.Fatal("expected check constraint violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "check") && !strings.Contains(errStr, "violates") && !strings.Contains(errStr, "constraint") {
		t.Fatalf("expected check constraint violation error, got: %v", err)
	}
}

// =============================================================================
// Primary Key Violation Tests
// =============================================================================

func TestError_PrimaryKeyViolation_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `INSERT INTO users (id) VALUES ('11111111-1111-1111-1111-111111111111')`)

	// Try to insert duplicate primary key - should fail
	_, err := db.Exec(`INSERT INTO users (id) VALUES ('11111111-1111-1111-1111-111111111111')`)
	if err == nil {
		t.Fatal("expected primary key violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "duplicate") && !strings.Contains(errStr, "unique") && !strings.Contains(errStr, "primary") {
		t.Fatalf("expected primary key violation error, got: %v", err)
	}
}

func TestError_PrimaryKeyViolation_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `INSERT INTO users (id) VALUES ('11111111-1111-1111-1111-111111111111')`)

	// Try to insert duplicate primary key - should fail
	_, err := db.Exec(`INSERT INTO users (id) VALUES ('11111111-1111-1111-1111-111111111111')`)
	if err == nil {
		t.Fatal("expected primary key violation error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "unique") && !strings.Contains(errStr, "constraint") && !strings.Contains(errStr, "primary") {
		t.Fatalf("expected primary key violation error, got: %v", err)
	}
}

// =============================================================================
// Transaction Rollback Tests
// =============================================================================

func TestTransactionRollback_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255) NOT NULL)`)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Insert a row
	_, err = tx.Exec(`INSERT INTO users (id, email) VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com')`)
	if err != nil {
		tx.Rollback()
		t.Fatalf("failed to insert in transaction: %v", err)
	}

	// Rollback
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify row was not inserted
	testutil.AssertRowCount(t, db, "users", 0)
}

func TestTransactionRollback_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT NOT NULL)`)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Insert a row
	_, err = tx.Exec(`INSERT INTO users (id, email) VALUES ('11111111-1111-1111-1111-111111111111', 'test@example.com')`)
	if err != nil {
		tx.Rollback()
		t.Fatalf("failed to insert in transaction: %v", err)
	}

	// Rollback
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify row was not inserted
	testutil.AssertRowCount(t, db, "users", 0)
}

// =============================================================================
// String Length Violation Tests
// =============================================================================

func TestError_StringTooLong_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY, name VARCHAR(10) NOT NULL)`)

	// Try to insert string longer than column length - should fail
	_, err := db.Exec(`INSERT INTO users (id, name) VALUES ('11111111-1111-1111-1111-111111111111', 'this_is_a_very_long_name')`)
	if err == nil {
		t.Fatal("expected string too long error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "too long") && !strings.Contains(errStr, "varying") && !strings.Contains(errStr, "character") {
		t.Fatalf("expected string too long error, got: %v", err)
	}
}

// =============================================================================
// Index Operation Tests
// =============================================================================

func TestError_DuplicateIndex_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255))`)
	testutil.ExecSQL(t, db, `CREATE INDEX idx_users_email ON users(email)`)

	// Try to create duplicate index - should fail
	_, err := db.Exec(`CREATE INDEX idx_users_email ON users(email)`)
	if err == nil {
		t.Fatal("expected duplicate index error, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "already exists") && !strings.Contains(errStr, "relation") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestError_DropNonExistentIndex_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE users (id UUID PRIMARY KEY)`)

	// Try to drop non-existent index - should fail
	_, err := db.Exec(`DROP INDEX nonexistent_idx`)
	if err == nil {
		t.Fatal("expected error when dropping non-existent index, got nil")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "does not exist") {
		t.Fatalf("expected 'does not exist' error, got: %v", err)
	}
}

// =============================================================================
// Concurrent Modification Tests (Optimistic Locking Scenario)
// =============================================================================

func TestConcurrentModification_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE counters (id UUID PRIMARY KEY, value INTEGER NOT NULL, version INTEGER NOT NULL)`)
	testutil.ExecSQL(t, db, `INSERT INTO counters (id, value, version) VALUES ('11111111-1111-1111-1111-111111111111', 0, 1)`)

	// Simulate optimistic locking conflict
	// First update succeeds
	result, err := db.Exec(`UPDATE counters SET value = 1, version = 2 WHERE id = '11111111-1111-1111-1111-111111111111' AND version = 1`)
	if err != nil {
		t.Fatalf("first update failed: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		t.Fatalf("expected 1 row affected, got %d", rowsAffected)
	}

	// Second update with old version fails (0 rows affected)
	result, err = db.Exec(`UPDATE counters SET value = 2, version = 2 WHERE id = '11111111-1111-1111-1111-111111111111' AND version = 1`)
	if err != nil {
		t.Fatalf("second update query failed: %v", err)
	}
	rowsAffected, _ = result.RowsAffected()
	if rowsAffected != 0 {
		t.Fatalf("expected 0 rows affected (optimistic lock failure), got %d", rowsAffected)
	}
}

// =============================================================================
// Cascade Delete Tests
// =============================================================================

func TestCascadeDelete_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE parent (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE child (
		id UUID PRIMARY KEY,
		parent_id UUID REFERENCES parent(id) ON DELETE CASCADE
	)`)

	// Insert parent and child
	testutil.ExecSQL(t, db, `INSERT INTO parent (id) VALUES ('11111111-1111-1111-1111-111111111111')`)
	testutil.ExecSQL(t, db, `INSERT INTO child (id, parent_id) VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111')`)

	// Delete parent - child should be deleted too
	testutil.ExecSQL(t, db, `DELETE FROM parent WHERE id = '11111111-1111-1111-1111-111111111111'`)

	// Verify child was deleted
	testutil.AssertRowCount(t, db, "child", 0)
}

func TestCascadeDelete_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)

	testutil.ExecSQL(t, db, `CREATE TABLE parent (id TEXT PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE child (
		id TEXT PRIMARY KEY,
		parent_id TEXT REFERENCES parent(id) ON DELETE CASCADE
	)`)

	// Insert parent and child
	testutil.ExecSQL(t, db, `INSERT INTO parent (id) VALUES ('11111111-1111-1111-1111-111111111111')`)
	testutil.ExecSQL(t, db, `INSERT INTO child (id, parent_id) VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111')`)

	// Delete parent - child should be deleted too
	testutil.ExecSQL(t, db, `DELETE FROM parent WHERE id = '11111111-1111-1111-1111-111111111111'`)

	// Verify child was deleted
	testutil.AssertRowCount(t, db, "child", 0)
}

// =============================================================================
// Set Null on Delete Tests
// =============================================================================

func TestSetNullOnDelete_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)

	testutil.ExecSQL(t, db, `CREATE TABLE parent (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE child (
		id UUID PRIMARY KEY,
		parent_id UUID REFERENCES parent(id) ON DELETE SET NULL
	)`)

	// Insert parent and child
	testutil.ExecSQL(t, db, `INSERT INTO parent (id) VALUES ('11111111-1111-1111-1111-111111111111')`)
	testutil.ExecSQL(t, db, `INSERT INTO child (id, parent_id) VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111')`)

	// Delete parent - child's parent_id should become NULL
	testutil.ExecSQL(t, db, `DELETE FROM parent WHERE id = '11111111-1111-1111-1111-111111111111'`)

	// Verify child still exists but parent_id is NULL
	testutil.AssertRowCount(t, db, "child", 1)
	var parentID *string
	err := db.QueryRow(`SELECT parent_id FROM child WHERE id = '22222222-2222-2222-2222-222222222222'`).Scan(&parentID)
	if err != nil {
		t.Fatalf("failed to query child: %v", err)
	}
	if parentID != nil {
		t.Fatalf("expected parent_id to be NULL, got %v", *parentID)
	}
}
