//go:build integration

// Package engine_test contains E2E cross-database consistency tests.
// These tests verify that the same schema produces equivalent results
// across PostgreSQL and SQLite.
package engine_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// -----------------------------------------------------------------------------
// Test Helper Types
// -----------------------------------------------------------------------------

// dbPair holds PostgreSQL and SQLite database connections for cross-db testing.
type dbPair struct {
	pg *sql.DB
	sq *sql.DB
}

// setupAllDBs creates connections to PostgreSQL and SQLite databases.
func setupAllDBs(t *testing.T) dbPair {
	t.Helper()
	return dbPair{
		pg: testutil.SetupPostgres(t),
		sq: testutil.SetupSQLite(t),
	}
}

// forEachDB runs a function against each database with its name.
func (d dbPair) forEachDB(t *testing.T, fn func(t *testing.T, db *sql.DB, dbName string)) {
	t.Helper()

	dbs := []struct {
		name string
		db   *sql.DB
	}{
		{"postgres", d.pg},
		{"sqlite", d.sq},
	}

	for _, db := range dbs {
		t.Run(db.name, func(t *testing.T) {
			fn(t, db.db, db.name)
		})
	}
}

// -----------------------------------------------------------------------------
// Schema Equivalence Tests
// -----------------------------------------------------------------------------

func TestCrossDB_SchemaEquivalence_CreateTable(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create same table on both databases using dialect-appropriate SQL
	pgSQL := `CREATE TABLE users (
		id UUID PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		name VARCHAR(100) NOT NULL,
		bio TEXT
	)`
	sqSQL := `CREATE TABLE users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		name TEXT NOT NULL,
		bio TEXT
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Verify all have the table
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertTableExists(t, db, "users")
	})

	// Verify all have the columns
	columns := []string{"id", "email", "name", "bio"}
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		for _, col := range columns {
			testutil.AssertColumnExists(t, db, "users", col)
		}
	})
}

func TestCrossDB_SchemaEquivalence_MultipleTablesWithFK(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create parent table
	pgUsersSQL := `CREATE TABLE auth_users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) NOT NULL UNIQUE
	)`
	sqUsersSQL := `CREATE TABLE auth_users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE
	)`

	testutil.ExecSQL(t, dbs.pg, pgUsersSQL)
	testutil.ExecSQL(t, dbs.sq, sqUsersSQL)

	// Create child table with FK
	pgPostsSQL := `CREATE TABLE blog_posts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		author_id UUID NOT NULL REFERENCES auth_users(id),
		title VARCHAR(200) NOT NULL,
		body TEXT
	)`
	sqPostsSQL := `CREATE TABLE blog_posts (
		id TEXT PRIMARY KEY,
		author_id TEXT NOT NULL REFERENCES auth_users(id),
		title TEXT NOT NULL,
		body TEXT
	)`

	testutil.ExecSQL(t, dbs.pg, pgPostsSQL)
	testutil.ExecSQL(t, dbs.sq, sqPostsSQL)

	// Verify both tables exist on all DBs
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertTableExists(t, db, "auth_users")
		testutil.AssertTableExists(t, db, "blog_posts")
	})
}

// -----------------------------------------------------------------------------
// Type Mapping Consistency Tests
// -----------------------------------------------------------------------------

func TestCrossDB_TypeMapping_StringTypes(t *testing.T) {
	dbs := setupAllDBs(t)

	// Test VARCHAR and TEXT types
	pgSQL := `CREATE TABLE string_test (
		id UUID PRIMARY KEY,
		short_str VARCHAR(50) NOT NULL,
		long_str VARCHAR(255) NOT NULL,
		unlimited_text TEXT
	)`
	sqSQL := `CREATE TABLE string_test (
		id TEXT PRIMARY KEY,
		short_str TEXT NOT NULL,
		long_str TEXT NOT NULL,
		unlimited_text TEXT
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Insert same data
	testID := "550e8400-e29b-41d4-a716-446655440000"
	insertPg := "INSERT INTO string_test (id, short_str, long_str, unlimited_text) VALUES ($1, $2, $3, $4)"
	insertSq := "INSERT INTO string_test (id, short_str, long_str, unlimited_text) VALUES (?, ?, ?, ?)"

	testutil.ExecSQL(t, dbs.pg, insertPg, testID, "short", "longer string here", "very long text content")
	testutil.ExecSQL(t, dbs.sq, insertSq, testID, "short", "longer string here", "very long text content")

	// Verify data is retrievable on all DBs
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertRowCount(t, db, "string_test", 1)

		var shortStr, longStr, text string
		var id string
		query := "SELECT id, short_str, long_str, unlimited_text FROM string_test"
		err := db.QueryRow(query).Scan(&id, &shortStr, &longStr, &text)
		if err != nil {
			t.Fatalf("failed to query %s: %v", name, err)
		}

		if shortStr != "short" {
			t.Errorf("%s: short_str = %q, want %q", name, shortStr, "short")
		}
		if longStr != "longer string here" {
			t.Errorf("%s: long_str = %q, want %q", name, longStr, "longer string here")
		}
		if text != "very long text content" {
			t.Errorf("%s: unlimited_text = %q, want %q", name, text, "very long text content")
		}
	})
}

func TestCrossDB_TypeMapping_NumericTypes(t *testing.T) {
	dbs := setupAllDBs(t)

	// Test INTEGER, FLOAT, DECIMAL
	pgSQL := `CREATE TABLE numeric_test (
		id UUID PRIMARY KEY,
		int_val INTEGER NOT NULL,
		float_val REAL NOT NULL,
		decimal_val DECIMAL(19, 4) NOT NULL
	)`
	sqSQL := `CREATE TABLE numeric_test (
		id TEXT PRIMARY KEY,
		int_val INTEGER NOT NULL,
		float_val REAL NOT NULL,
		decimal_val TEXT NOT NULL
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Insert same data
	testID := "550e8400-e29b-41d4-a716-446655440001"

	testutil.ExecSQL(t, dbs.pg, "INSERT INTO numeric_test VALUES ($1, $2, $3, $4)", testID, 42, 3.14, "9999.9999")
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO numeric_test VALUES (?, ?, ?, ?)", testID, 42, 3.14, "9999.9999")

	// Verify integer values
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var intVal int
		err := db.QueryRow("SELECT int_val FROM numeric_test").Scan(&intVal)
		if err != nil {
			t.Fatalf("%s: failed to query int_val: %v", name, err)
		}
		if intVal != 42 {
			t.Errorf("%s: int_val = %d, want 42", name, intVal)
		}
	})
}

func TestCrossDB_TypeMapping_BooleanType(t *testing.T) {
	dbs := setupAllDBs(t)

	// Boolean has different representations across DBs
	pgSQL := `CREATE TABLE bool_test (
		id UUID PRIMARY KEY,
		is_active BOOLEAN NOT NULL,
		is_verified BOOLEAN NOT NULL DEFAULT FALSE
	)`
	sqSQL := `CREATE TABLE bool_test (
		id TEXT PRIMARY KEY,
		is_active INTEGER NOT NULL,
		is_verified INTEGER NOT NULL DEFAULT 0
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Insert with explicit boolean values
	testID := "550e8400-e29b-41d4-a716-446655440002"

	testutil.ExecSQL(t, dbs.pg, "INSERT INTO bool_test (id, is_active, is_verified) VALUES ($1, TRUE, FALSE)", testID)
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO bool_test (id, is_active, is_verified) VALUES (?, 1, 0)", testID)

	// Verify boolean semantics work consistently
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var isActive, isVerified bool
		// Use CASE to normalize booleans for scanning
		query := "SELECT is_active, is_verified FROM bool_test"
		err := db.QueryRow(query).Scan(&isActive, &isVerified)
		if err != nil {
			t.Fatalf("%s: failed to query booleans: %v", name, err)
		}

		if !isActive {
			t.Errorf("%s: is_active should be true", name)
		}
		if isVerified {
			t.Errorf("%s: is_verified should be false", name)
		}
	})
}

func TestCrossDB_TypeMapping_DateTimeTypes(t *testing.T) {
	dbs := setupAllDBs(t)

	// Date/time handling
	pgSQL := `CREATE TABLE datetime_test (
		id UUID PRIMARY KEY,
		date_col DATE NOT NULL,
		time_col TIME NOT NULL,
		datetime_col TIMESTAMPTZ NOT NULL
	)`
	sqSQL := `CREATE TABLE datetime_test (
		id TEXT PRIMARY KEY,
		date_col TEXT NOT NULL,
		time_col TEXT NOT NULL,
		datetime_col TEXT NOT NULL
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Verify tables are created with correct columns
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertColumnExists(t, db, "datetime_test", "date_col")
		testutil.AssertColumnExists(t, db, "datetime_test", "time_col")
		testutil.AssertColumnExists(t, db, "datetime_test", "datetime_col")
	})
}

func TestCrossDB_TypeMapping_JSONType(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE json_test (
		id UUID PRIMARY KEY,
		data JSONB NOT NULL
	)`
	sqSQL := `CREATE TABLE json_test (
		id TEXT PRIMARY KEY,
		data TEXT NOT NULL
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Insert JSON data
	testID := "550e8400-e29b-41d4-a716-446655440003"
	jsonData := `{"name":"John","age":30}`

	testutil.ExecSQL(t, dbs.pg, "INSERT INTO json_test (id, data) VALUES ($1, $2::jsonb)", testID, jsonData)
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO json_test (id, data) VALUES (?, ?)", testID, jsonData)

	// Verify JSON can be retrieved as string
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var data string
		err := db.QueryRow("SELECT data FROM json_test").Scan(&data)
		if err != nil {
			t.Fatalf("%s: failed to query JSON: %v", name, err)
		}

		// JSON format may vary, but should contain expected keys
		if !strings.Contains(data, "name") || !strings.Contains(data, "John") {
			t.Errorf("%s: JSON data = %q, expected to contain name/John", name, data)
		}
	})
}

// -----------------------------------------------------------------------------
// Constraint Behavior Tests
// -----------------------------------------------------------------------------

func TestCrossDB_UniqueConstraint(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE unique_test (
		id UUID PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		code VARCHAR(20) NOT NULL UNIQUE
	)`
	sqSQL := `CREATE TABLE unique_test (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		code TEXT NOT NULL UNIQUE
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Insert first record
	id1 := "550e8400-e29b-41d4-a716-446655440010"
	testutil.ExecSQL(t, dbs.pg, "INSERT INTO unique_test VALUES ($1, $2, $3)", id1, "user@test.com", "CODE1")
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO unique_test VALUES (?, ?, ?)", id1, "user@test.com", "CODE1")

	// Try to insert duplicate email - should fail on all DBs
	id2 := "550e8400-e29b-41d4-a716-446655440011"
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO unique_test VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO unique_test VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, id2, "user@test.com", "CODE2")
		if err == nil {
			t.Errorf("%s: expected unique constraint violation for duplicate email", name)
		}
	})

	// Try to insert duplicate code - should fail on all DBs
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO unique_test VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO unique_test VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, id2, "other@test.com", "CODE1")
		if err == nil {
			t.Errorf("%s: expected unique constraint violation for duplicate code", name)
		}
	})

	// Insert valid different record - should succeed
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO unique_test VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO unique_test VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, id2, "other@test.com", "CODE2")
		if err != nil {
			t.Errorf("%s: unexpected error inserting valid record: %v", name, err)
		}
	})
}

func TestCrossDB_ForeignKeyConstraint(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create parent table
	pgParent := `CREATE TABLE fk_parent (id UUID PRIMARY KEY, name VARCHAR(100) NOT NULL)`
	sqParent := `CREATE TABLE fk_parent (id TEXT PRIMARY KEY, name TEXT NOT NULL)`

	testutil.ExecSQL(t, dbs.pg, pgParent)
	testutil.ExecSQL(t, dbs.sq, sqParent)

	// Create child table with FK
	pgChild := `CREATE TABLE fk_child (
		id UUID PRIMARY KEY,
		parent_id UUID NOT NULL REFERENCES fk_parent(id),
		value VARCHAR(100)
	)`
	sqChild := `CREATE TABLE fk_child (
		id TEXT PRIMARY KEY,
		parent_id TEXT NOT NULL REFERENCES fk_parent(id),
		value TEXT
	)`

	testutil.ExecSQL(t, dbs.pg, pgChild)
	testutil.ExecSQL(t, dbs.sq, sqChild)

	// Insert parent record
	parentID := "550e8400-e29b-41d4-a716-446655440020"
	testutil.ExecSQL(t, dbs.pg, "INSERT INTO fk_parent VALUES ($1, $2)", parentID, "Parent 1")
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO fk_parent VALUES (?, ?)", parentID, "Parent 1")

	// Try to insert child with non-existent parent - should fail on all DBs
	childID := "550e8400-e29b-41d4-a716-446655440021"
	nonExistentParent := "550e8400-e29b-41d4-a716-446655440099"

	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO fk_child VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO fk_child VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, childID, nonExistentParent, "orphan")
		if err == nil {
			t.Errorf("%s: expected FK constraint violation for non-existent parent", name)
		}
	})

	// Insert child with valid parent - should succeed
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO fk_child VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO fk_child VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, childID, parentID, "valid child")
		if err != nil {
			t.Errorf("%s: unexpected error inserting valid child: %v", name, err)
		}
	})
}

func TestCrossDB_ForeignKeyOnDeleteCascade(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create parent table
	pgParent := `CREATE TABLE cascade_parent (id UUID PRIMARY KEY, name VARCHAR(100) NOT NULL)`
	sqParent := `CREATE TABLE cascade_parent (id TEXT PRIMARY KEY, name TEXT NOT NULL)`

	testutil.ExecSQL(t, dbs.pg, pgParent)
	testutil.ExecSQL(t, dbs.sq, sqParent)

	// Create child table with ON DELETE CASCADE
	pgChild := `CREATE TABLE cascade_child (
		id UUID PRIMARY KEY,
		parent_id UUID NOT NULL REFERENCES cascade_parent(id) ON DELETE CASCADE,
		value VARCHAR(100)
	)`
	sqChild := `CREATE TABLE cascade_child (
		id TEXT PRIMARY KEY,
		parent_id TEXT NOT NULL REFERENCES cascade_parent(id) ON DELETE CASCADE,
		value TEXT
	)`

	testutil.ExecSQL(t, dbs.pg, pgChild)
	testutil.ExecSQL(t, dbs.sq, sqChild)

	// Insert parent and children
	parentID := "550e8400-e29b-41d4-a716-446655440030"
	childID1 := "550e8400-e29b-41d4-a716-446655440031"
	childID2 := "550e8400-e29b-41d4-a716-446655440032"

	testutil.ExecSQL(t, dbs.pg, "INSERT INTO cascade_parent VALUES ($1, $2)", parentID, "Parent")
	testutil.ExecSQL(t, dbs.pg, "INSERT INTO cascade_child VALUES ($1, $2, $3)", childID1, parentID, "Child 1")
	testutil.ExecSQL(t, dbs.pg, "INSERT INTO cascade_child VALUES ($1, $2, $3)", childID2, parentID, "Child 2")

	testutil.ExecSQL(t, dbs.sq, "INSERT INTO cascade_parent VALUES (?, ?)", parentID, "Parent")
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO cascade_child VALUES (?, ?, ?)", childID1, parentID, "Child 1")
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO cascade_child VALUES (?, ?, ?)", childID2, parentID, "Child 2")

	// Verify children exist
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertRowCount(t, db, "cascade_child", 2)
	})

	// Delete parent - children should be deleted too
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "DELETE FROM cascade_parent WHERE id = $1"
		} else {
			query = "DELETE FROM cascade_parent WHERE id = ?"
		}
		_, err := db.Exec(query, parentID)
		if err != nil {
			t.Fatalf("%s: failed to delete parent: %v", name, err)
		}

		// Children should be gone
		testutil.AssertRowCount(t, db, "cascade_child", 0)
	})
}

// -----------------------------------------------------------------------------
// Index Behavior Tests
// -----------------------------------------------------------------------------

func TestCrossDB_IndexCreation(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create table
	pgSQL := `CREATE TABLE idx_test (
		id UUID PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		name VARCHAR(100) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	)`
	sqSQL := `CREATE TABLE idx_test (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Create indexes
	pgIdx := `CREATE INDEX idx_test_email ON idx_test (email)`
	sqIdx := `CREATE INDEX idx_test_email ON idx_test (email)`

	testutil.ExecSQL(t, dbs.pg, pgIdx)
	testutil.ExecSQL(t, dbs.sq, sqIdx)

	// Create unique index
	pgUniq := `CREATE UNIQUE INDEX idx_test_name_unique ON idx_test (name)`
	sqUniq := `CREATE UNIQUE INDEX idx_test_name_unique ON idx_test (name)`

	testutil.ExecSQL(t, dbs.pg, pgUniq)
	testutil.ExecSQL(t, dbs.sq, sqUniq)

	// Verify indexes exist
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertIndexExists(t, db, "idx_test", "idx_test_email")
		testutil.AssertIndexExists(t, db, "idx_test", "idx_test_name_unique")
	})
}

func TestCrossDB_CompositeIndex(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create table
	pgSQL := `CREATE TABLE composite_idx_test (
		id UUID PRIMARY KEY,
		tenant_id UUID NOT NULL,
		user_id UUID NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	)`
	sqSQL := `CREATE TABLE composite_idx_test (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Create composite index
	pgIdx := `CREATE INDEX idx_composite_tenant_user ON composite_idx_test (tenant_id, user_id)`
	sqIdx := `CREATE INDEX idx_composite_tenant_user ON composite_idx_test (tenant_id, user_id)`

	testutil.ExecSQL(t, dbs.pg, pgIdx)
	testutil.ExecSQL(t, dbs.sq, sqIdx)

	// Verify composite index exists
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		testutil.AssertIndexExists(t, db, "composite_idx_test", "idx_composite_tenant_user")
	})
}

// -----------------------------------------------------------------------------
// NULL Handling Tests
// -----------------------------------------------------------------------------

func TestCrossDB_NotNullConstraint(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE notnull_test (
		id UUID PRIMARY KEY,
		required_field VARCHAR(100) NOT NULL,
		optional_field VARCHAR(100)
	)`
	sqSQL := `CREATE TABLE notnull_test (
		id TEXT PRIMARY KEY,
		required_field TEXT NOT NULL,
		optional_field TEXT
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Try to insert NULL into NOT NULL column - should fail
	testID := "550e8400-e29b-41d4-a716-446655440040"
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO notnull_test (id, required_field) VALUES ($1, NULL)"
		} else {
			query = "INSERT INTO notnull_test (id, required_field) VALUES (?, NULL)"
		}
		_, err := db.Exec(query, testID)
		if err == nil {
			t.Errorf("%s: expected NOT NULL constraint violation", name)
		}
	})

	// Insert with NULL in optional field - should succeed
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO notnull_test (id, required_field, optional_field) VALUES ($1, $2, NULL)"
		} else {
			query = "INSERT INTO notnull_test (id, required_field, optional_field) VALUES (?, ?, NULL)"
		}
		_, err := db.Exec(query, testID, "required value")
		if err != nil {
			t.Errorf("%s: unexpected error with NULL in optional field: %v", name, err)
		}
	})

	// Verify NULL is stored correctly
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var optionalField sql.NullString
		query := "SELECT optional_field FROM notnull_test"
		err := db.QueryRow(query).Scan(&optionalField)
		if err != nil {
			t.Fatalf("%s: failed to query: %v", name, err)
		}
		if optionalField.Valid {
			t.Errorf("%s: expected NULL for optional_field, got %q", name, optionalField.String)
		}
	})
}

func TestCrossDB_NullableWithDefault(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE nullable_default (
		id UUID PRIMARY KEY,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		priority INTEGER DEFAULT 0
	)`
	sqSQL := `CREATE TABLE nullable_default (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL DEFAULT 'pending',
		priority INTEGER DEFAULT 0
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Insert without specifying default columns
	testID := "550e8400-e29b-41d4-a716-446655440050"
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO nullable_default (id) VALUES ($1)"
		} else {
			query = "INSERT INTO nullable_default (id) VALUES (?)"
		}
		_, err := db.Exec(query, testID)
		if err != nil {
			t.Fatalf("%s: failed to insert: %v", name, err)
		}
	})

	// Verify defaults are applied
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var status string
		var priority int
		query := "SELECT status, priority FROM nullable_default"
		err := db.QueryRow(query).Scan(&status, &priority)
		if err != nil {
			t.Fatalf("%s: failed to query: %v", name, err)
		}

		if status != "pending" {
			t.Errorf("%s: status = %q, want 'pending'", name, status)
		}
		if priority != 0 {
			t.Errorf("%s: priority = %d, want 0", name, priority)
		}
	})
}

// -----------------------------------------------------------------------------
// Data Integrity Tests
// -----------------------------------------------------------------------------

func TestCrossDB_TransactionIsolation(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create table
	pgSQL := `CREATE TABLE tx_test (id UUID PRIMARY KEY, counter INTEGER NOT NULL)`
	sqSQL := `CREATE TABLE tx_test (id TEXT PRIMARY KEY, counter INTEGER NOT NULL)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	testID := "550e8400-e29b-41d4-a716-446655440060"

	// Test transaction rollback works correctly on all DBs
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		// Insert initial row
		var insertQuery string
		if name == "postgres" {
			insertQuery = "INSERT INTO tx_test VALUES ($1, $2)"
		} else {
			insertQuery = "INSERT INTO tx_test VALUES (?, ?)"
		}
		_, err := db.Exec(insertQuery, testID, 0)
		if err != nil {
			t.Fatalf("%s: failed to insert: %v", name, err)
		}

		// Start transaction and update
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("%s: failed to begin tx: %v", name, err)
		}

		var updateQuery string
		if name == "postgres" {
			updateQuery = "UPDATE tx_test SET counter = 999 WHERE id = $1"
		} else {
			updateQuery = "UPDATE tx_test SET counter = 999 WHERE id = ?"
		}
		_, err = tx.Exec(updateQuery, testID)
		if err != nil {
			tx.Rollback()
			t.Fatalf("%s: failed to update in tx: %v", name, err)
		}

		// Rollback instead of commit
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("%s: failed to rollback: %v", name, err)
		}

		// Verify original value is preserved
		var counter int
		err = db.QueryRow("SELECT counter FROM tx_test").Scan(&counter)
		if err != nil {
			t.Fatalf("%s: failed to query: %v", name, err)
		}

		if counter != 0 {
			t.Errorf("%s: counter = %d after rollback, want 0", name, counter)
		}
	})
}

func TestCrossDB_CompositeUniqueConstraint(t *testing.T) {
	dbs := setupAllDBs(t)

	// Create table with composite unique constraint
	pgSQL := `CREATE TABLE composite_unique (
		id UUID PRIMARY KEY,
		tenant_id UUID NOT NULL,
		email VARCHAR(255) NOT NULL,
		UNIQUE(tenant_id, email)
	)`
	sqSQL := `CREATE TABLE composite_unique (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		email TEXT NOT NULL,
		UNIQUE(tenant_id, email)
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	tenant1 := "550e8400-e29b-41d4-a716-446655440070"
	tenant2 := "550e8400-e29b-41d4-a716-446655440071"
	id1 := "550e8400-e29b-41d4-a716-446655440072"
	id2 := "550e8400-e29b-41d4-a716-446655440073"
	id3 := "550e8400-e29b-41d4-a716-446655440074"

	// Insert first record
	testutil.ExecSQL(t, dbs.pg, "INSERT INTO composite_unique VALUES ($1, $2, $3)", id1, tenant1, "user@test.com")
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO composite_unique VALUES (?, ?, ?)", id1, tenant1, "user@test.com")

	// Same email in different tenant - should succeed
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO composite_unique VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO composite_unique VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, id2, tenant2, "user@test.com")
		if err != nil {
			t.Errorf("%s: should allow same email in different tenant: %v", name, err)
		}
	})

	// Same tenant + email - should fail
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var query string
		if name == "postgres" {
			query = "INSERT INTO composite_unique VALUES ($1, $2, $3)"
		} else {
			query = "INSERT INTO composite_unique VALUES (?, ?, ?)"
		}
		_, err := db.Exec(query, id3, tenant1, "user@test.com")
		if err == nil {
			t.Errorf("%s: expected composite unique constraint violation", name)
		}
	})
}

// -----------------------------------------------------------------------------
// Edge Cases Tests
// -----------------------------------------------------------------------------

func TestCrossDB_EmptyStringVsNull(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE empty_vs_null (
		id UUID PRIMARY KEY,
		nullable_col VARCHAR(100),
		empty_col VARCHAR(100) NOT NULL
	)`
	sqSQL := `CREATE TABLE empty_vs_null (
		id TEXT PRIMARY KEY,
		nullable_col TEXT,
		empty_col TEXT NOT NULL
	)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	testID := "550e8400-e29b-41d4-a716-446655440080"

	// Insert with empty string (not NULL)
	testutil.ExecSQL(t, dbs.pg, "INSERT INTO empty_vs_null VALUES ($1, NULL, '')", testID)
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO empty_vs_null VALUES (?, NULL, '')", testID)

	// Verify empty string vs NULL distinction
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var nullable sql.NullString
		var empty string
		err := db.QueryRow("SELECT nullable_col, empty_col FROM empty_vs_null").Scan(&nullable, &empty)
		if err != nil {
			t.Fatalf("%s: failed to query: %v", name, err)
		}

		if nullable.Valid {
			t.Errorf("%s: nullable_col should be NULL, got %q", name, nullable.String)
		}
		if empty != "" {
			t.Errorf("%s: empty_col should be empty string, got %q", name, empty)
		}
	})
}

func TestCrossDB_UUIDFormat(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE uuid_test (id UUID PRIMARY KEY, ref_id UUID NOT NULL)`
	sqSQL := `CREATE TABLE uuid_test (id TEXT PRIMARY KEY, ref_id TEXT NOT NULL)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Test standard UUID format
	id := "550e8400-e29b-41d4-a716-446655440090"
	refID := "550e8400-e29b-41d4-a716-446655440091"

	testutil.ExecSQL(t, dbs.pg, "INSERT INTO uuid_test VALUES ($1, $2)", id, refID)
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO uuid_test VALUES (?, ?)", id, refID)

	// Verify UUID is stored and retrieved correctly
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var storedID, storedRefID string
		err := db.QueryRow("SELECT id, ref_id FROM uuid_test").Scan(&storedID, &storedRefID)
		if err != nil {
			t.Fatalf("%s: failed to query: %v", name, err)
		}

		if storedID != id {
			t.Errorf("%s: id = %q, want %q", name, storedID, id)
		}
		if storedRefID != refID {
			t.Errorf("%s: ref_id = %q, want %q", name, storedRefID, refID)
		}
	})
}

func TestCrossDB_LargeTextField(t *testing.T) {
	dbs := setupAllDBs(t)

	pgSQL := `CREATE TABLE large_text (id UUID PRIMARY KEY, content TEXT NOT NULL)`
	sqSQL := `CREATE TABLE large_text (id TEXT PRIMARY KEY, content TEXT NOT NULL)`

	testutil.ExecSQL(t, dbs.pg, pgSQL)
	testutil.ExecSQL(t, dbs.sq, sqSQL)

	// Create large text content (10KB)
	largeContent := strings.Repeat("Lorem ipsum dolor sit amet. ", 400)
	testID := "550e8400-e29b-41d4-a716-446655440100"

	testutil.ExecSQL(t, dbs.pg, "INSERT INTO large_text VALUES ($1, $2)", testID, largeContent)
	testutil.ExecSQL(t, dbs.sq, "INSERT INTO large_text VALUES (?, ?)", testID, largeContent)

	// Verify large content is stored correctly
	dbs.forEachDB(t, func(t *testing.T, db *sql.DB, name string) {
		var content string
		err := db.QueryRow("SELECT content FROM large_text").Scan(&content)
		if err != nil {
			t.Fatalf("%s: failed to query: %v", name, err)
		}

		if len(content) != len(largeContent) {
			t.Errorf("%s: content length = %d, want %d", name, len(content), len(largeContent))
		}
		if content != largeContent {
			t.Errorf("%s: content mismatch", name)
		}
	})
}
