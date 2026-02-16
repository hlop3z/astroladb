//go:build integration
// +build integration

package runner

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/hlop3z/astroladb/internal/engine"
)

// TestPhaseFailureRecovery_BackfillResumability tests that backfill operations
// can be safely resumed after interruption.
//
// This test verifies:
// - Backfill WHERE clause (IS NULL) makes operation idempotent
// - Partially completed backfills can be resumed
// - Re-running backfill doesn't duplicate or corrupt data
//
// Requirements:
// - PostgreSQL database
// - Run with: go test -tags=integration -run TestPhaseFailureRecovery -v
func TestPhaseFailureRecovery_BackfillResumability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failure recovery test in short mode")
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

	// Create test table with 50K rows
	t.Log("Creating test table with 50,000 rows...")
	startSetup := time.Now()

	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS recovery_test CASCADE;
		CREATE TABLE recovery_test (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert 50K rows
	batchSize := 10000
	totalRows := 50000

	for i := 0; i < totalRows; i += batchSize {
		values := make([]string, 0, batchSize)
		for j := 0; j < batchSize && i+j < totalRows; j++ {
			email := fmt.Sprintf("user%d@example.com", i+j)
			values = append(values, fmt.Sprintf("('%s')", email))
		}

		insertSQL := fmt.Sprintf("INSERT INTO recovery_test (email) VALUES %s",
			joinStrings(values, ","))

		_, err = db.ExecContext(ctx, insertSQL)
		if err != nil {
			t.Fatalf("Failed to insert batch at row %d: %v", i, err)
		}
	}

	setupTime := time.Since(startSetup)
	t.Logf("✓ Created table with 50,000 rows in %v", setupTime)

	// Add column (nullable first)
	t.Log("Adding bio column...")
	_, err = db.ExecContext(ctx, "ALTER TABLE recovery_test ADD COLUMN bio TEXT")
	if err != nil {
		t.Fatalf("Failed to add column: %v", err)
	}

	// Test Scenario 1: Partial backfill then resume
	t.Log("Scenario 1: Partial backfill then resume...")

	// Backfill 50% of rows (25K out of 50K)
	t.Log("  Step 1: Backfilling first 25,000 rows...")
	partialBackfillSQL := `
		UPDATE recovery_test
		SET bio = 'Partial bio'
		WHERE bio IS NULL
		AND id IN (
			SELECT id FROM recovery_test WHERE bio IS NULL LIMIT 25000
		)
	`
	result, err := db.ExecContext(ctx, partialBackfillSQL)
	if err != nil {
		t.Fatalf("Partial backfill failed: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	t.Logf("  ✓ Backfilled %d rows", rowsAffected)

	// Verify partial state
	var nullCount, filledCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio IS NULL").Scan(&nullCount)
	if err != nil {
		t.Fatalf("Failed to count NULL rows: %v", err)
	}
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio IS NOT NULL").Scan(&filledCount)
	if err != nil {
		t.Fatalf("Failed to count filled rows: %v", err)
	}

	t.Logf("  State after partial backfill: %d NULL, %d filled", nullCount, filledCount)

	if nullCount == 0 {
		t.Error("Expected some NULL rows remaining after partial backfill")
	}
	if filledCount == 0 {
		t.Error("Expected some filled rows after partial backfill")
	}

	// Now resume backfill (complete the remaining rows)
	t.Log("  Step 2: Resuming backfill (idempotent operation)...")

	// Generate full batched update (same as production)
	fullBackfillSQL := engine.GenerateBatchedUpdate(
		"recovery_test",
		"bio",
		"'Resumed bio'",
		"bio IS NULL",
	)

	startResume := time.Now()
	_, err = db.ExecContext(ctx, fullBackfillSQL)
	if err != nil {
		t.Fatalf("Resume backfill failed: %v", err)
	}
	resumeTime := time.Since(startResume)

	t.Logf("  ✓ Resume completed in %v", resumeTime)

	// Verify all rows are backfilled
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio IS NULL").Scan(&nullCount)
	if err != nil {
		t.Fatalf("Failed to count NULL rows: %v", err)
	}

	if nullCount != 0 {
		t.Errorf("Expected 0 NULL rows after resume, got %d", nullCount)
	} else {
		t.Log("  ✓ All rows backfilled after resume")
	}

	// Verify no duplicates (all rows should have bio)
	var totalCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test").Scan(&totalCount)
	if err != nil {
		t.Fatalf("Failed to count total rows: %v", err)
	}

	if totalCount != totalRows {
		t.Errorf("Expected %d total rows, got %d (possible duplicates)", totalRows, totalCount)
	}

	// Test Scenario 2: Re-run backfill (should be no-op)
	t.Log("Scenario 2: Re-run backfill (idempotency test)...")

	startRerun := time.Now()
	result, err = db.ExecContext(ctx, fullBackfillSQL)
	if err != nil {
		t.Fatalf("Re-run backfill failed: %v", err)
	}
	rerunTime := time.Since(startRerun)

	rowsAffected, _ = result.RowsAffected()
	t.Logf("  ✓ Re-run completed in %v, affected %d rows", rerunTime, rowsAffected)

	// Should affect 0 rows (all already backfilled)
	// Note: In batched update, we can't easily get rows affected, but we can verify state

	// Verify state unchanged
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio IS NULL").Scan(&nullCount)
	if err != nil {
		t.Fatalf("Failed to count NULL rows: %v", err)
	}

	if nullCount != 0 {
		t.Errorf("Expected 0 NULL rows after re-run, got %d", nullCount)
	}

	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test").Scan(&totalCount)
	if err != nil {
		t.Fatalf("Failed to count total rows: %v", err)
	}

	if totalCount != totalRows {
		t.Errorf("Expected %d total rows after re-run, got %d (possible duplicates)", totalRows, totalCount)
	} else {
		t.Log("  ✓ Re-run is idempotent (no duplicates, no changes)")
	}

	// Test Scenario 3: Verify different backfill values don't conflict
	t.Log("Scenario 3: Testing backfill value updates...")

	// Reset table
	_, err = db.ExecContext(ctx, "UPDATE recovery_test SET bio = NULL")
	if err != nil {
		t.Fatalf("Failed to reset bio column: %v", err)
	}

	// Backfill with value A
	backfillA := engine.GenerateBatchedUpdate(
		"recovery_test",
		"bio",
		"'Value A'",
		"bio IS NULL",
	)
	_, err = db.ExecContext(ctx, backfillA)
	if err != nil {
		t.Fatalf("Backfill A failed: %v", err)
	}

	// Verify all have Value A
	var valueACount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio = 'Value A'").Scan(&valueACount)
	if err != nil {
		t.Fatalf("Failed to count Value A rows: %v", err)
	}

	if valueACount != totalRows {
		t.Errorf("Expected %d rows with 'Value A', got %d", totalRows, valueACount)
	} else {
		t.Log("  ✓ All rows have Value A")
	}

	// Try to backfill with value B (should be no-op due to WHERE bio IS NULL)
	backfillB := engine.GenerateBatchedUpdate(
		"recovery_test",
		"bio",
		"'Value B'",
		"bio IS NULL",
	)
	_, err = db.ExecContext(ctx, backfillB)
	if err != nil {
		t.Fatalf("Backfill B failed: %v", err)
	}

	// Verify still all have Value A (Value B didn't overwrite)
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio = 'Value A'").Scan(&valueACount)
	if err != nil {
		t.Fatalf("Failed to count Value A rows: %v", err)
	}

	var valueBCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_test WHERE bio = 'Value B'").Scan(&valueBCount)
	if err != nil {
		t.Fatalf("Failed to count Value B rows: %v", err)
	}

	if valueACount != totalRows {
		t.Errorf("Expected %d rows with 'Value A', got %d", totalRows, valueACount)
	}
	if valueBCount != 0 {
		t.Errorf("Expected 0 rows with 'Value B', got %d (WHERE clause not working)", valueBCount)
	} else {
		t.Log("  ✓ WHERE bio IS NULL prevents overwrites (idempotent)")
	}

	// Cleanup
	t.Log("Cleaning up...")
	_, err = db.ExecContext(ctx, "DROP TABLE recovery_test CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to cleanup table: %v", err)
	}

	t.Log("✓ Phase failure recovery test completed successfully")
}

// TestPhaseFailureRecovery_IndexCreation tests index creation failure scenarios.
func TestPhaseFailureRecovery_IndexCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping index recovery test in short mode")
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

	// Create test table
	t.Log("Creating test table...")

	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS index_recovery_test CASCADE;
		CREATE TABLE index_recovery_test (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert some rows
	_, err = db.ExecContext(ctx, `
		INSERT INTO index_recovery_test (email)
		SELECT 'user' || generate_series || '@example.com'
		FROM generate_series(1, 1000)
	`)
	if err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	// Test: Create index, drop it, create again (recovery scenario)
	t.Log("Scenario: Index creation after previous failure...")

	// Create index
	indexSQL := "CREATE INDEX CONCURRENTLY idx_index_recovery_email ON index_recovery_test(email)"
	_, err = db.ExecContext(ctx, indexSQL)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	t.Log("  ✓ Index created")

	// Simulate failure: drop index
	_, err = db.ExecContext(ctx, "DROP INDEX idx_index_recovery_email")
	if err != nil {
		t.Fatalf("Failed to drop index: %v", err)
	}
	t.Log("  ✓ Index dropped (simulating failure)")

	// Recreate index (recovery)
	_, err = db.ExecContext(ctx, indexSQL)
	if err != nil {
		t.Fatalf("Failed to recreate index: %v", err)
	}
	t.Log("  ✓ Index recreated successfully")

	// Verify index exists
	var indexExists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'index_recovery_test'
			AND indexname = 'idx_index_recovery_email'
		)
	`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check index existence: %v", err)
	}

	if !indexExists {
		t.Error("Index was not recreated")
	} else {
		t.Log("  ✓ Index exists and is usable")
	}

	// Cleanup
	t.Log("Cleaning up...")
	_, err = db.ExecContext(ctx, "DROP TABLE index_recovery_test CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to cleanup table: %v", err)
	}

	t.Log("✓ Index recovery test completed successfully")
}
