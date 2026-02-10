package testutil

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

// CreateTestDB creates a test database connection and ensures it's cleaned up.
// The database connection is automatically closed when the test finishes.
//
// Example:
//
//	db := testutil.CreateTestDB(t, "sqlite", ":memory:")
//	defer db.Close() // Optional, already handled by t.Cleanup
func CreateTestDB(t *testing.T, dialect, url string) *sql.DB {
	t.Helper()

	db, err := sql.Open(dialect, url)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// RetryWithTimeout retries a function until it succeeds or times out.
// Useful for flaky operations like database connections.
//
// Example:
//
//	err := testutil.RetryWithTimeout(t, 5*time.Second, func() error {
//	    return db.Ping()
//	})
func RetryWithTimeout(t *testing.T, timeout time.Duration, fn func() error) error {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
		}
	}

	return lastErr
}

// SkipIfShort skips the test if running in short mode.
// Use this for integration tests.
//
// Example:
//
//	func TestIntegration(t *testing.T) {
//	    testutil.SkipIfShort(t)
//	    // ... integration test code
//	}
func SkipIfShort(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// SkipInCI skips the test if running in CI environment.
// Useful for tests that require local resources.
func SkipInCI(t *testing.T) {
	t.Helper()

	if isCI() {
		t.Skip("Skipping test in CI environment")
	}
}

// isCI checks if running in a CI environment.
func isCI() bool {
	// Check common CI environment variables
	ciEnvVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
	}

	for _, envVar := range ciEnvVars {
		if value := os.Getenv(envVar); value != "" {
			return true
		}
	}

	return false
}

// RequireEnv ensures an environment variable is set, or skips the test.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Required environment variable %s not set", key)
	}

	return value
}

// Must asserts that err is nil, or fails the test immediately.
// Useful for test setup code.
//
// Example:
//
//	testutil.Must(t, os.WriteFile(path, data, 0644))
func Must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// MustValue asserts that err is nil, or fails the test immediately.
// Returns the value on success.
//
// Example:
//
//	content := testutil.MustValue(t, os.ReadFile(path))
func MustValue[T any](t *testing.T, value T, err error) T {
	t.Helper()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return value
}

// Parallel marks the test to run in parallel with other parallel tests.
// Only use this for tests that don't share state.
func Parallel(t *testing.T) {
	t.Helper()
	t.Parallel()
}
