// Package testutil provides test helpers for the Alab project.
// It includes database setup, SQL assertions, error assertions, and golden file testing.
package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// updateGolden is a flag to update golden files.
// Use -update-golden to update golden files during test runs.
var updateGolden = flag.Bool("update-golden", false, "update golden files")

// -----------------------------------------------------------------------------
// SQL Assertions
// -----------------------------------------------------------------------------

// NormalizeSQL normalizes a SQL string for comparison.
// It collapses multiple whitespace characters into a single space,
// trims leading/trailing whitespace, and converts to uppercase.
func NormalizeSQL(sql string) string {
	// Replace all whitespace sequences (including newlines) with single space
	ws := regexp.MustCompile(`\s+`)
	sql = ws.ReplaceAllString(sql, " ")

	// Trim leading/trailing whitespace
	sql = strings.TrimSpace(sql)

	// Convert to uppercase for case-insensitive comparison
	sql = strings.ToUpper(sql)

	return sql
}

// AssertSQL compares two SQL strings after normalizing them.
// Normalization includes collapsing whitespace and converting to uppercase.
func AssertSQL(t *testing.T, got, want string) {
	t.Helper()

	gotNorm := NormalizeSQL(got)
	wantNorm := NormalizeSQL(want)

	if gotNorm != wantNorm {
		t.Errorf("SQL mismatch:\ngot:  %s\nwant: %s\n\noriginal got:\n%s\n\noriginal want:\n%s",
			gotNorm, wantNorm, got, want)
	}
}

// AssertSQLContains checks if a SQL string contains a substring.
// Both strings are normalized before comparison.
func AssertSQLContains(t *testing.T, sql, substr string) {
	t.Helper()

	sqlNorm := NormalizeSQL(sql)
	substrNorm := NormalizeSQL(substr)

	if !strings.Contains(sqlNorm, substrNorm) {
		t.Errorf("SQL does not contain expected substring:\nsql:    %s\nsubstr: %s\n\noriginal sql:\n%s",
			sqlNorm, substrNorm, sql)
	}
}

// -----------------------------------------------------------------------------
// Error Assertions
// -----------------------------------------------------------------------------

// AssertError checks that an error has the expected error code.
// If err is nil or doesn't have the expected code, the test fails.
func AssertError(t *testing.T, err error, code alerr.Code) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error with code %s, got nil", code)
		return
	}

	gotCode := alerr.GetErrorCode(err)
	if gotCode != code {
		t.Errorf("expected error code %s, got %s\nerror: %v", code, gotCode, err)
	}
}

// AssertNoError checks that an error is nil.
// If err is not nil, the test fails with the error message.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// AssertErrorContains checks that an error message contains a substring.
// If err is nil, the test fails.
func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error containing %q, got nil", substr)
		return
	}

	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error message does not contain %q\ngot: %v", substr, err)
	}
}

// -----------------------------------------------------------------------------
// Golden File Testing
// -----------------------------------------------------------------------------

// goldenDir returns the path to the testdata directory for golden files.
// It uses the test's file location to determine the correct path.
func goldenDir(t *testing.T) string {
	t.Helper()

	// Use the current working directory combined with testdata
	// This should work in most testing scenarios
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	return filepath.Join(wd, "testdata")
}

// goldenPath returns the path to a golden file.
func goldenPath(t *testing.T, name, ext string) string {
	t.Helper()

	dir := goldenDir(t)
	return filepath.Join(dir, name+ext)
}

// Golden compares a string against a golden file.
// If -update-golden flag is passed, updates the golden file instead.
// Golden files are stored in testdata/ directory with .golden extension.
func Golden(t *testing.T, name string, got string) {
	t.Helper()

	path := goldenPath(t, name, ".golden")

	if *updateGolden {
		// Ensure testdata directory exists
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}

		// Write the new golden file
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	// Read the expected golden file
	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file does not exist: %s\nrun with -update-golden to create it\n\ngot:\n%s",
				path, got)
		}
		t.Fatalf("failed to read golden file: %v", err)
	}

	// Compare
	if got != string(want) {
		t.Errorf("golden file mismatch: %s\n\ngot:\n%s\n\nwant:\n%s\n\nrun with -update-golden to update",
			path, got, string(want))
	}
}

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// TempDir creates a temporary directory that is automatically cleaned up.
func TempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "alab-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// WriteFile writes content to a file, creating parent directories as needed.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent directories: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

// AssertEqual is a generic equality check for testing.
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()

	if got != want {
		t.Errorf("values not equal:\ngot:  %v\nwant: %v", got, want)
	}
}

// AssertTrue checks that a condition is true.
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()

	if !condition {
		t.Errorf("expected true: %s", msg)
	}
}

// AssertFalse checks that a condition is false.
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()

	if condition {
		t.Errorf("expected false: %s", msg)
	}
}

// AssertNil checks that a value is nil.
func AssertNil(t *testing.T, val any) {
	t.Helper()

	if val != nil {
		t.Errorf("expected nil, got: %v", val)
	}
}

// AssertNotNil checks that a value is not nil.
func AssertNotNil(t *testing.T, val any) {
	t.Helper()

	if val == nil {
		t.Error("expected non-nil value, got nil")
	}
}
