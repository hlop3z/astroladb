package testutil

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

func TestNormalizeSQL(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "simple",
			sql:  "SELECT * FROM users",
			want: "SELECT * FROM USERS",
		},
		{
			name: "with extra spaces",
			sql:  "SELECT  *   FROM  users",
			want: "SELECT * FROM USERS",
		},
		{
			name: "with newlines",
			sql: `SELECT *
				FROM users
				WHERE id = 1`,
			want: "SELECT * FROM USERS WHERE ID = 1",
		},
		{
			name: "with tabs",
			sql:  "SELECT\t*\tFROM\tusers",
			want: "SELECT * FROM USERS",
		},
		{
			name: "complex",
			sql: `
				CREATE TABLE users (
					id UUID PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				)
			`,
			want: "CREATE TABLE USERS ( ID UUID PRIMARY KEY, NAME VARCHAR(255) NOT NULL )",
		},
		{
			name: "leading and trailing whitespace",
			sql:  "   SELECT * FROM users   ",
			want: "SELECT * FROM USERS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSQL(tt.sql)
			if got != tt.want {
				t.Errorf("NormalizeSQL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAssertSQL(t *testing.T) {
	// Create a mock test
	mockT := &testing.T{}

	// Test matching SQL
	AssertSQL(mockT, "SELECT * FROM users", "select * from users")
	if mockT.Failed() {
		t.Error("AssertSQL should not fail for equivalent SQL")
	}

	// Test with different whitespace
	mockT = &testing.T{}
	AssertSQL(mockT, "SELECT  *  FROM  users", "SELECT * FROM users")
	if mockT.Failed() {
		t.Error("AssertSQL should not fail for SQL with different whitespace")
	}
}

func TestAssertSQLContains(t *testing.T) {
	mockT := &testing.T{}

	// Test containing substring
	AssertSQLContains(mockT, "SELECT * FROM users WHERE id = 1", "FROM users")
	if mockT.Failed() {
		t.Error("AssertSQLContains should not fail for SQL containing substring")
	}

	// Test case insensitive
	mockT = &testing.T{}
	AssertSQLContains(mockT, "SELECT * FROM users", "from USERS")
	if mockT.Failed() {
		t.Error("AssertSQLContains should not fail for case-insensitive match")
	}
}

func TestAssertError(t *testing.T) {
	mockT := &testing.T{}

	// Test with correct error code
	err := alerr.New(alerr.ErrInvalidIdentifier, "test error")
	AssertError(mockT, err, alerr.ErrInvalidIdentifier)
	if mockT.Failed() {
		t.Error("AssertError should not fail for matching error code")
	}

	// Test nil error (should fail)
	mockT = &testing.T{}
	AssertError(mockT, nil, alerr.ErrInvalidIdentifier)
	if !mockT.Failed() {
		t.Error("AssertError should fail for nil error")
	}
}

func TestAssertNoError(t *testing.T) {
	mockT := &testing.T{}

	// Test with nil error
	AssertNoError(mockT, nil)
	if mockT.Failed() {
		t.Error("AssertNoError should not fail for nil error")
	}

	// Test with error (should fail)
	mockT = &testing.T{}
	err := alerr.New(alerr.ErrInvalidIdentifier, "test error")
	AssertNoError(mockT, err)
	if !mockT.Failed() {
		t.Error("AssertNoError should fail for non-nil error")
	}
}

func TestAssertErrorContains(t *testing.T) {
	mockT := &testing.T{}

	// Test with matching substring
	err := alerr.New(alerr.ErrInvalidIdentifier, "invalid column name")
	AssertErrorContains(mockT, err, "invalid")
	if mockT.Failed() {
		t.Error("AssertErrorContains should not fail for matching substring")
	}

	// Test nil error (should fail)
	mockT = &testing.T{}
	AssertErrorContains(mockT, nil, "any")
	if !mockT.Failed() {
		t.Error("AssertErrorContains should fail for nil error")
	}
}

func TestAssertEqual(t *testing.T) {
	mockT := &testing.T{}

	// Test equal values
	AssertEqual(mockT, 42, 42)
	if mockT.Failed() {
		t.Error("AssertEqual should not fail for equal values")
	}

	// Test equal strings
	mockT = &testing.T{}
	AssertEqual(mockT, "hello", "hello")
	if mockT.Failed() {
		t.Error("AssertEqual should not fail for equal strings")
	}
}

func TestAssertTrue(t *testing.T) {
	mockT := &testing.T{}

	AssertTrue(mockT, true, "should be true")
	if mockT.Failed() {
		t.Error("AssertTrue should not fail for true condition")
	}

	mockT = &testing.T{}
	AssertTrue(mockT, false, "should be true")
	if !mockT.Failed() {
		t.Error("AssertTrue should fail for false condition")
	}
}

func TestAssertFalse(t *testing.T) {
	mockT := &testing.T{}

	AssertFalse(mockT, false, "should be false")
	if mockT.Failed() {
		t.Error("AssertFalse should not fail for false condition")
	}

	mockT = &testing.T{}
	AssertFalse(mockT, true, "should be false")
	if !mockT.Failed() {
		t.Error("AssertFalse should fail for true condition")
	}
}

func TestTempDir(t *testing.T) {
	dir := TempDir(t)

	// Verify directory exists
	if dir == "" {
		t.Error("TempDir should return a non-empty path")
	}
}

func TestTempFile(t *testing.T) {
	path := TempFile(t, "test.txt", "hello world")

	// Verify file was created
	if path == "" {
		t.Error("TempFile should return a non-empty path")
	}
}

func TestWriteFile(t *testing.T) {
	dir := TempDir(t)
	path := dir + "/subdir/test.txt"

	WriteFile(t, path, "test content")

	// File should be created (tested indirectly through no panic)
}
