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

// Performance benchmark results are logged but don't fail tests.
// These provide baseline performance data for different operations.

// BenchmarkBackfillPerformance_VariousTableSizes benchmarks backfill
// performance across different table sizes.
//
// Run with: go test -tags=integration -run BenchmarkBackfillPerformance -v
func TestBenchmarkBackfillPerformance_VariousTableSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark in short mode")
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
		timeout  time.Duration
	}{
		{"10K rows", 10000, 2 * time.Minute},
		{"50K rows", 50000, 5 * time.Minute},
		{"100K rows", 100000, 10 * time.Minute},
		{"500K rows", 500000, 30 * time.Minute},
	}

	results := make(map[string]BenchmarkResult)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tableName := fmt.Sprintf("bench_backfill_%d", tc.rowCount)

			// Create table
			t.Logf("Creating table with %d rows...", tc.rowCount)
			startSetup := time.Now()

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

			// Insert rows in batches
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

			setupTime := time.Since(startSetup)

			// Add column
			_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN bio TEXT", tableName))
			if err != nil {
				t.Fatalf("Failed to add column: %v", err)
			}

			// Benchmark backfill
			t.Log("Running backfill benchmark...")
			backfillSQL := engine.GenerateBatchedUpdate(
				tableName,
				"bio",
				"'Benchmark bio'",
				"bio IS NULL",
			)

			startBackfill := time.Now()
			_, err = db.ExecContext(ctx, backfillSQL)
			if err != nil {
				t.Fatalf("Backfill failed: %v", err)
			}
			backfillTime := time.Since(startBackfill)

			// Calculate throughput
			rowsPerSec := float64(tc.rowCount) / backfillTime.Seconds()

			result := BenchmarkResult{
				RowCount:     tc.rowCount,
				SetupTime:    setupTime,
				BackfillTime: backfillTime,
				RowsPerSec:   rowsPerSec,
			}

			results[tc.name] = result

			// Log result
			t.Logf("\nBenchmark Results for %s:", tc.name)
			t.Logf("  Setup time: %v", setupTime)
			t.Logf("  Backfill time: %v", backfillTime)
			t.Logf("  Throughput: %.0f rows/sec", rowsPerSec)

			// Cleanup
			_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s CASCADE", tableName))
		})
	}

	// Print summary
	t.Log("\n========================================")
	t.Log("Backfill Performance Summary")
	t.Log("========================================")
	for name, result := range results {
		t.Logf("%s:", name)
		t.Logf("  Backfill: %v (%.0f rows/sec)", result.BackfillTime, result.RowsPerSec)
	}
	t.Log("========================================")
}

// BenchmarkIndexCreation_VariousTableSizes benchmarks concurrent index
// creation across different table sizes.
func TestBenchmarkIndexCreation_VariousTableSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark in short mode")
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
		{"500K rows", 500000},
	}

	results := make(map[string]IndexBenchmarkResult)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tableName := fmt.Sprintf("bench_index_%d", tc.rowCount)

			// Create table
			t.Logf("Creating table with %d rows...", tc.rowCount)
			_, err := db.ExecContext(ctx, fmt.Sprintf(`
				DROP TABLE IF EXISTS %s CASCADE;
				CREATE TABLE %s (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email TEXT NOT NULL,
					name TEXT NOT NULL,
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
					name := fmt.Sprintf("User %d", i+j)
					values = append(values, fmt.Sprintf("('%s', '%s')", email, name))
				}

				insertSQL := fmt.Sprintf("INSERT INTO %s (email, name) VALUES %s", tableName, joinStrings(values, ","))
				_, err = db.ExecContext(ctx, insertSQL)
				if err != nil {
					t.Fatalf("Failed to insert batch: %v", err)
				}
			}

			// Benchmark index creation (concurrent)
			t.Log("Benchmarking CONCURRENT index creation...")
			indexSQL := fmt.Sprintf("CREATE INDEX CONCURRENTLY idx_%s_email ON %s(email)", tableName, tableName)

			start := time.Now()
			_, err = db.ExecContext(ctx, indexSQL)
			if err != nil {
				t.Fatalf("Index creation failed: %v", err)
			}
			indexTime := time.Since(start)

			// Benchmark unique index creation
			t.Log("Benchmarking UNIQUE index creation...")
			uniqueIndexSQL := fmt.Sprintf("CREATE UNIQUE INDEX CONCURRENTLY idx_%s_name ON %s(name)", tableName, tableName)

			startUnique := time.Now()
			_, err = db.ExecContext(ctx, uniqueIndexSQL)
			if err != nil {
				t.Fatalf("Unique index creation failed: %v", err)
			}
			uniqueIndexTime := time.Since(startUnique)

			result := IndexBenchmarkResult{
				RowCount:        tc.rowCount,
				IndexTime:       indexTime,
				UniqueIndexTime: uniqueIndexTime,
			}

			results[tc.name] = result

			// Log result
			t.Logf("\nBenchmark Results for %s:", tc.name)
			t.Logf("  Index creation: %v", indexTime)
			t.Logf("  Unique index creation: %v", uniqueIndexTime)

			// Cleanup
			_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s CASCADE", tableName))
		})
	}

	// Print summary
	t.Log("\n========================================")
	t.Log("Index Creation Performance Summary")
	t.Log("========================================")
	for name, result := range results {
		t.Logf("%s:", name)
		t.Logf("  Index: %v", result.IndexTime)
		t.Logf("  Unique: %v", result.UniqueIndexTime)
	}
	t.Log("========================================")
}

// BenchmarkDDLOperations benchmarks various DDL operations.
func TestBenchmarkDDLOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark in short mode")
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

	// Create test table with 100K rows
	tableName := "bench_ddl"
	rowCount := 100000

	t.Logf("Creating table with %d rows...", rowCount)
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		DROP TABLE IF EXISTS %s CASCADE;
		CREATE TABLE %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL
		);
	`, tableName, tableName))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert rows
	batchSize := 10000
	for i := 0; i < rowCount; i += batchSize {
		values := make([]string, 0, batchSize)
		for j := 0; j < batchSize && i+j < rowCount; j++ {
			email := fmt.Sprintf("user%d@example.com", i+j)
			values = append(values, fmt.Sprintf("('%s')", email))
		}

		insertSQL := fmt.Sprintf("INSERT INTO %s (email) VALUES %s", tableName, joinStrings(values, ","))
		_, err = db.ExecContext(ctx, insertSQL)
		if err != nil {
			t.Fatalf("Failed to insert batch: %v", err)
		}
	}

	t.Log("\nBenchmarking DDL operations...")

	// Benchmark: ADD COLUMN (nullable)
	t.Run("ADD_COLUMN_nullable", func(t *testing.T) {
		start := time.Now()
		_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN name TEXT", tableName))
		if err != nil {
			t.Fatalf("ADD COLUMN failed: %v", err)
		}
		duration := time.Since(start)
		t.Logf("ADD COLUMN (nullable): %v", duration)
	})

	// Benchmark: ADD COLUMN with DEFAULT
	t.Run("ADD_COLUMN_with_default", func(t *testing.T) {
		start := time.Now()
		_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN status TEXT DEFAULT 'active'", tableName))
		if err != nil {
			t.Fatalf("ADD COLUMN with DEFAULT failed: %v", err)
		}
		duration := time.Since(start)
		t.Logf("ADD COLUMN with DEFAULT: %v", duration)
	})

	// Benchmark: DROP COLUMN
	t.Run("DROP_COLUMN", func(t *testing.T) {
		start := time.Now()
		_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s DROP COLUMN status", tableName))
		if err != nil {
			t.Fatalf("DROP COLUMN failed: %v", err)
		}
		duration := time.Since(start)
		t.Logf("DROP COLUMN: %v", duration)
	})

	// Benchmark: RENAME COLUMN
	t.Run("RENAME_COLUMN", func(t *testing.T) {
		start := time.Now()
		_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s RENAME COLUMN name TO full_name", tableName))
		if err != nil {
			t.Fatalf("RENAME COLUMN failed: %v", err)
		}
		duration := time.Since(start)
		t.Logf("RENAME COLUMN: %v", duration)
	})

	// Cleanup
	_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s CASCADE", tableName))

	t.Log("\n✓ DDL operations benchmark completed")
}

// BenchmarkResult holds backfill benchmark results.
type BenchmarkResult struct {
	RowCount     int
	SetupTime    time.Duration
	BackfillTime time.Duration
	RowsPerSec   float64
}

// IndexBenchmarkResult holds index creation benchmark results.
type IndexBenchmarkResult struct {
	RowCount        int
	IndexTime       time.Duration
	UniqueIndexTime time.Duration
}
