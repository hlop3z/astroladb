package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// LoadJSFixture loads a JavaScript fixture file as a string.
// The path should be relative to the project root.
//
// Example:
//
//	jsCode := testutil.LoadJSFixture(t, "test/fixtures/schemas/auth/user.js")
func LoadJSFixture(t *testing.T, relativePath string) string {
	t.Helper()

	rootDir := findProjectRoot(t)
	fullPath := filepath.Join(rootDir, relativePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to load fixture %s: %v", relativePath, err)
	}

	return string(content)
}

// LoadJSFixtureOrDefault loads a fixture file, or returns defaultCode if the file doesn't exist.
// This is useful for gradual migration where some tests still use inline code.
func LoadJSFixtureOrDefault(t *testing.T, relativePath, defaultCode string) string {
	t.Helper()

	rootDir := findProjectRoot(t)
	fullPath := filepath.Join(rootDir, relativePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		// File doesn't exist, return default
		return defaultCode
	}

	return string(content)
}

// MustLoadJSFixture is like LoadJSFixture but panics on error.
// Useful for test setup code.
func MustLoadJSFixture(relativePath string) string {
	rootDir := mustFindProjectRoot()
	fullPath := filepath.Join(rootDir, relativePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		panic("Failed to load fixture " + relativePath + ": " + err.Error())
	}

	return string(content)
}

// findProjectRoot walks up the directory tree to find the project root (go.mod location).
func findProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return findProjectRootFrom(t, dir)
}

// findProjectRootFrom finds the project root starting from the given directory.
func findProjectRootFrom(t *testing.T, startDir string) string {
	t.Helper()

	dir := startDir

	// Walk up until we find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod)")
		}
		dir = parent
	}
}

// mustFindProjectRoot is like findProjectRoot but panics on error.
func mustFindProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("Failed to get working directory: " + err.Error())
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			panic("Could not find project root (go.mod)")
		}
		dir = parent
	}
}

// FixtureExists checks if a fixture file exists.
func FixtureExists(t *testing.T, relativePath string) bool {
	t.Helper()

	rootDir := findProjectRoot(t)
	fullPath := filepath.Join(rootDir, relativePath)

	_, err := os.Stat(fullPath)
	return err == nil
}
