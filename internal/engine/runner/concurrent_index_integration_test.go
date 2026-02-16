//go:build integration
// +build integration

package runner

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestConcurrentIndexCreation_NoWriteLocks tests that CONCURRENT indexes
// don't block writes to the table during index creation.
//
// This test verifies:
// - CREATE INDEX CONCURRENTLY allows writes during index creation
// - Writers don't get blocked by long-running index creation
// - Index is created successfully while writes continue
//
// Requirements:
// - PostgreSQL database (SQLite doesn't support CONCURRENT indexes)
// - Run with: go test -tags=integration -run TestConcurrentIndexCreation -v
func TestConcurrentIndexCreation_NoWriteLocks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent index test in short mode")
	}

	dbURL := getTestDatabaseURL()
	if dbURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create test table with 100K rows (large enough for slow index creation)
	t.Log("Creating test table with 100,000 rows...")
	startSetup := time.Now()

	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS concurrent_test CASCADE;
		CREATE TABLE concurrent_test (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert 100K rows in batches
	batchSize := 10000
	totalRows := 100000

	for i := 0; i < totalRows; i += batchSize {
		values := make([]string, 0, batchSize)
		for j := 0; j < batchSize && i+j < totalRows; j++ {
			email := fmt.Sprintf("user%d@example.com", i+j)
			name := fmt.Sprintf("User %d", i+j)
			values = append(values, fmt.Sprintf("('%s', '%s')", email, name))
		}

		insertSQL := fmt.Sprintf("INSERT INTO concurrent_test (email, name) VALUES %s",
			joinStrings(values, ","))

		_, err = db.ExecContext(ctx, insertSQL)
		if err != nil {
			t.Fatalf("Failed to insert batch at row %d: %v", i, err)
		}
	}

	setupTime := time.Since(startSetup)
	t.Logf("✓ Created table with 100,000 rows in %v", setupTime)

	// Verify row count
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM concurrent_test").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if count != totalRows {
		t.Fatalf("Expected %d rows, got %d", totalRows, count)
	}

	// Test concurrent index creation
	t.Log("Testing concurrent index creation while writes continue...")

	// For this test, we use raw SQL to create a CONCURRENT index
	// The production code should generate CREATE INDEX CONCURRENTLY automatically
	indexSQL := "CREATE UNIQUE INDEX CONCURRENTLY idx_concurrent_test_email ON concurrent_test(email)"

	t.Logf("Using SQL: %s", indexSQL)

	// Start concurrent writes in background
	var wg sync.WaitGroup
	writeErrors := make(chan error, 1)
	writesDone := make(chan bool)
	writesBlocked := false

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Try to insert rows every 100ms for up to 10 seconds
		// If we get blocked, this will timeout
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		insertCount := 0
		for {
			select {
			case <-timeout:
				t.Logf("Writer completed: %d inserts", insertCount)
				return
			case <-ticker.C:
				// Try to insert a row with 2-second timeout
				insertCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				email := fmt.Sprintf("concurrent_user_%d@example.com", time.Now().UnixNano())
				_, err := db.ExecContext(insertCtx, "INSERT INTO concurrent_test (email, name) VALUES ($1, $2)",
					email, "Concurrent User")
				cancel()

				if err != nil {
					// Check if it's a timeout (means we got blocked)
					if err == context.DeadlineExceeded {
						t.Logf("WARNING: Write blocked by index creation (timeout after 2s)")
						writesBlocked = true
						writeErrors <- fmt.Errorf("writes blocked by index creation")
						return
					}
					// Ignore unique constraint violations (expected during concurrent writes)
					if !containsIgnoreCase(err.Error(), "unique constraint") &&
						!containsIgnoreCase(err.Error(), "duplicate key") {
						writeErrors <- fmt.Errorf("write failed: %v", err)
						return
					}
				} else {
					insertCount++
				}
			}
		}
	}()

	// Wait a bit for writer to start
	time.Sleep(200 * time.Millisecond)

	// Create index (should not block writes)
	t.Log("Creating CONCURRENT index...")
	startIndex := time.Now()

	_, err = db.ExecContext(ctx, indexSQL)
	if err != nil {
		t.Fatalf("Failed to create concurrent index: %v", err)
	}

	indexTime := time.Since(startIndex)
	t.Logf("✓ Index created in %v", indexTime)

	// Signal writer to stop
	close(writesDone)

	// Wait for writer to finish
	wg.Wait()

	// Check for write errors
	select {
	case err := <-writeErrors:
		t.Errorf("Writer encountered error: %v", err)
	default:
		// No errors
	}

	// Verify writes were NOT blocked
	if writesBlocked {
		t.Error("FAIL: Writes were blocked by index creation (should use CONCURRENTLY)")
	} else {
		t.Log("✓ Writes continued successfully during index creation")
	}

	// Verify index was created
	var indexExists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'concurrent_test'
			AND indexname = 'idx_concurrent_test_email'
		)
	`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check index existence: %v", err)
	}

	if !indexExists {
		t.Error("Index was not created")
	} else {
		t.Log("✓ Index exists and is usable")
	}

	// Cleanup
	t.Log("Cleaning up...")
	_, err = db.ExecContext(ctx, "DROP TABLE concurrent_test CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to cleanup table: %v", err)
	}

	t.Log("✓ Concurrent index test completed successfully")
}

// TestConcurrentIndexCreation_Performance benchmarks index creation time
// for different table sizes.
func TestConcurrentIndexCreation_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	dbURL := getTestDatabaseURL()
	if dbURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	testCases := []struct {
		name     string
		rowCount int
	}{
		{"10K rows", 10000},
		{"50K rows", 50000},
		{"100K rows", 100000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tableName := fmt.Sprintf("perf_test_%d", tc.rowCount)

			// Create table
			_, err := db.ExecContext(ctx, fmt.Sprintf(`
				DROP TABLE IF EXISTS %s CASCADE;
				CREATE TABLE %s (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email TEXT NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				);
			`, tableName, tableName))
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Insert rows
			batchSize := 10000
			for i := 0; i < tc.rowCount; i += batchSize {
				values := make([]string, 0, batchSize)
				for j := 0; j < batchSize && i+j < tc.rowCount; j++ {
					email := fmt.Sprintf("user%d@example.com", i+j)
					values = append(values, fmt.Sprintf("('%s')", email))
				}

				insertSQL := fmt.Sprintf("INSERT INTO %s (email) VALUES %s", tableName, joinStrings(values, ","))
				_, err = db.ExecContext(ctx, insertSQL)
				if err != nil {
					t.Fatalf("Failed to insert batch: %v", err)
				}
			}

			// Create concurrent index
			indexSQL := fmt.Sprintf("CREATE INDEX CONCURRENTLY idx_%s_email ON %s(email)", tableName, tableName)
			start := time.Now()

			_, err = db.ExecContext(ctx, indexSQL)
			if err != nil {
				t.Fatalf("Failed to create index: %v", err)
			}

			duration := time.Since(start)
			t.Logf("Index creation time: %v (%.0f rows/sec)", duration, float64(tc.rowCount)/duration.Seconds())

			// Cleanup
			_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s CASCADE", tableName))
		})
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsIgnoreCaseHelper(s, substr)))
}

func containsIgnoreCaseHelper(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
