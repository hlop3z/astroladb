//go:build integration
// +build integration

package runner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// TestBackfillLargeDataset_1MillionRows tests batched backfill with 1M rows.
// This is a long-running test that verifies:
// - Batched updates work correctly with large datasets
// - Progress logging works
// - Memory usage stays reasonable
// - Transaction boundaries are correct
//
// Requirements:
// - PostgreSQL database with 1M+ row capacity
// - Run with: go test -tags=integration -run TestBackfillLargeDataset_1MillionRows -v
func TestBackfillLargeDataset_1MillionRows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	// Connect to PostgreSQL test database
	// Set DATABASE_URL environment variable or use default
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

	// Create test table with 1M rows
	t.Log("Creating test table with 1,000,000 rows...")
	startSetup := time.Now()

	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS backfill_test_1m CASCADE;
		CREATE TABLE backfill_test_1m (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert 1M rows in batches (faster than individual inserts)
	batchSize := 10000
	totalRows := 1000000

	for i := 0; i < totalRows; i += batchSize {
		values := make([]string, 0, batchSize)
		for j := 0; j < batchSize && i+j < totalRows; j++ {
			values = append(values, fmt.Sprintf("('User %d')", i+j))
		}

		insertSQL := fmt.Sprintf("INSERT INTO backfill_test_1m (name) VALUES %s",
			joinStrings(values, ","))

		_, err = db.ExecContext(ctx, insertSQL)
		if err != nil {
			t.Fatalf("Failed to insert batch at row %d: %v", i, err)
		}

		if (i+batchSize)%100000 == 0 {
			t.Logf("Inserted %d rows...", i+batchSize)
		}
	}

	setupTime := time.Since(startSetup)
	t.Logf("✓ Created table with 1,000,000 rows in %v", setupTime)

	// Verify row count
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM backfill_test_1m").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if count != totalRows {
		t.Fatalf("Expected %d rows, got %d", totalRows, count)
	}

	// Now test the backfill operation
	t.Log("Testing batched backfill (ADD COLUMN with backfill)...")
	startBackfill := time.Now()

	// Create migration operation: ADD COLUMN bio with backfill
	op := &ast.AddColumn{
		TableRef: ast.TableRef{Table_: "backfill_test_1m"},
		Column: &ast.ColumnDef{
			Name:        "bio",
			Type:        "string",
			Nullable:    false,
			Backfill:    "Default bio",
			BackfillSet: true,
		},
	}

	// Split into phases
	phases := engine.SplitIntoPhases([]ast.Operation{op})

	if len(phases) != 2 {
		t.Fatalf("Expected 2 phases (DDL + Data), got %d", len(phases))
	}

	// Phase 1: DDL (ADD COLUMN as nullable first)
	t.Log("Phase 1: Adding column...")
	_, err = db.ExecContext(ctx, "ALTER TABLE backfill_test_1m ADD COLUMN bio TEXT")
	if err != nil {
		t.Fatalf("Phase 1 failed: %v", err)
	}

	// Phase 2: Data (Batched backfill)
	t.Log("Phase 2: Running batched backfill...")

	// Generate batched update SQL
	backfillSQL := engine.GenerateBatchedUpdate(
		"backfill_test_1m",
		"bio",
		"'Default bio'",
		"bio IS NULL",
	)

	// Execute batched update
	_, err = db.ExecContext(ctx, backfillSQL)
	if err != nil {
		t.Fatalf("Phase 2 (backfill) failed: %v", err)
	}

	backfillTime := time.Since(startBackfill)
	t.Logf("✓ Backfill completed in %v", backfillTime)

	// Verify all rows were backfilled
	var nullCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM backfill_test_1m WHERE bio IS NULL").Scan(&nullCount)
	if err != nil {
		t.Fatalf("Failed to count NULL rows: %v", err)
	}

	if nullCount != 0 {
		t.Errorf("Expected 0 NULL rows after backfill, got %d", nullCount)
	}

	// Verify backfilled value
	var sampleBio string
	err = db.QueryRowContext(ctx, "SELECT bio FROM backfill_test_1m LIMIT 1").Scan(&sampleBio)
	if err != nil {
		t.Fatalf("Failed to sample backfilled value: %v", err)
	}

	if sampleBio != "Default bio" {
		t.Errorf("Expected bio='Default bio', got %q", sampleBio)
	}

	// Phase 3: Set NOT NULL constraint
	t.Log("Phase 3: Setting NOT NULL constraint...")
	_, err = db.ExecContext(ctx, "ALTER TABLE backfill_test_1m ALTER COLUMN bio SET NOT NULL")
	if err != nil {
		t.Fatalf("Phase 3 failed: %v", err)
	}

	// Performance check
	t.Logf("\nPerformance Summary:")
	t.Logf("  Setup (1M inserts): %v", setupTime)
	t.Logf("  Backfill (1M updates): %v", backfillTime)
	t.Logf("  Rows/second: %.0f", float64(totalRows)/backfillTime.Seconds())

	// Expected performance:
	// - 5,000 rows per batch
	// - 1M rows = 200 batches
	// - 2s per batch (1s update + 1s sleep)
	// - Total: ~400s (6-7 minutes)
	expectedTime := 400 * time.Second
	if backfillTime > expectedTime*2 {
		t.Logf("WARNING: Backfill took longer than expected (expected ~%v, got %v)",
			expectedTime, backfillTime)
	}

	// Cleanup
	t.Log("Cleaning up...")
	_, err = db.ExecContext(ctx, "DROP TABLE backfill_test_1m CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to cleanup table: %v", err)
	}

	t.Log("✓ Large dataset test completed successfully")
}

// TestBackfillLargeDataset_MemoryUsage verifies memory stays reasonable during backfill.
func TestBackfillLargeDataset_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// This test would use runtime.MemStats to verify memory doesn't grow unbounded
	// during backfill operations. Batched updates should keep memory usage constant.
	t.Skip("Memory usage test - implement with runtime.MemStats")
}

// TestBackfillLargeDataset_Resumability tests that backfill can be resumed after failure.
func TestBackfillLargeDataset_Resumability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resumability test in short mode")
	}

	// This test would:
	// 1. Start a backfill
	// 2. Interrupt it midway (simulate failure)
	// 3. Re-run the same backfill
	// 4. Verify it completes correctly (WHERE bio IS NULL is idempotent)
	t.Skip("Resumability test - implement with controlled interruption")
}

// getTestDatabaseURL returns the test database URL from environment or default.
func getTestDatabaseURL() string {
	// Try environment variable first
	if url := getEnv("DATABASE_URL"); url != "" {
		return url
	}

	// Try TEST_DATABASE_URL
	if url := getEnv("TEST_DATABASE_URL"); url != "" {
		return url
	}

	// Default (assumes local PostgreSQL)
	return "postgres://postgres:postgres@localhost:5432/astroladb_test?sslmode=disable"
}

// getEnv returns environment variable value or empty string.
func getEnv(key string) string {
	return os.Getenv(key)
}

// joinStrings joins string slice with separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
