# Integration Tests for Large Dataset Backfill

This directory contains integration tests that verify the multi-phase migration system works correctly with large datasets (1M+ rows).

## Requirements

### Database
- **PostgreSQL 15+** with enough disk space for test data
- At least **2GB** free disk space (for 1M row table)
- Connection string via environment variable

### Go Build Tags
Tests use the `integration` build tag to prevent running during normal test execution:
```go
//go:build integration
// +build integration
```

## Running the Tests

### 1. Set Up PostgreSQL Test Database

```bash
# Create test database
createdb astroladb_test

# Or use Docker
docker run -d \
  --name astroladb-test-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=astroladb_test \
  -p 5432:5432 \
  postgres:15
```

### 2. Set Database URL

```bash
# Option 1: Use DATABASE_URL
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/astroladb_test?sslmode=disable"

# Option 2: Use TEST_DATABASE_URL (higher priority)
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/astroladb_test?sslmode=disable"
```

### 3. Run Integration Tests

```bash
# Run all integration tests
go test -tags=integration -v ./internal/engine/runner

# Run specific test
go test -tags=integration -run TestBackfillLargeDataset_1MillionRows -v ./internal/engine/runner

# Run with timeout (large dataset tests can take 10+ minutes)
go test -tags=integration -timeout 30m -v ./internal/engine/runner
```

## Test Cases

### TestBackfillLargeDataset_1MillionRows

**What it tests:**
- Batched backfill with 1,000,000 rows
- Progress logging during batched updates
- Memory usage stays reasonable
- Transaction boundaries are correct

**Expected duration:** ~6-10 minutes

**Expected output:**
```
=== RUN   TestBackfillLargeDataset_1MillionRows
    backfill_large_integration_test.go:38: Creating test table with 1,000,000 rows...
    backfill_large_integration_test.go:62: Inserted 100000 rows...
    backfill_large_integration_test.go:62: Inserted 200000 rows...
    ...
    backfill_large_integration_test.go:70: ✓ Created table with 1,000,000 rows in 45s
    backfill_large_integration_test.go:83: Testing batched backfill (ADD COLUMN with backfill)...
    backfill_large_integration_test.go:104: Phase 1: Adding column...
    backfill_large_integration_test.go:110: Phase 2: Running batched backfill...
    backfill_large_integration_test.go:122: ✓ Backfill completed in 6m40s
    backfill_large_integration_test.go:151:
Performance Summary:
    backfill_large_integration_test.go:152:   Setup (1M inserts): 45s
    backfill_large_integration_test.go:153:   Backfill (1M updates): 6m40s
    backfill_large_integration_test.go:154:   Rows/second: 2500
--- PASS: TestBackfillLargeDataset_1MillionRows (7.5m)
```

**Performance expectations:**
- **Batch size:** 5,000 rows
- **Batches:** 200 (1M ÷ 5K)
- **Time per batch:** ~2 seconds (1s UPDATE + 1s pg_sleep)
- **Total time:** ~400 seconds (~6-7 minutes)
- **Throughput:** ~2,500 rows/second

### 2. TestConcurrentIndexCreation_NoWriteLocks

**What it tests:**

- CREATE INDEX CONCURRENTLY allows writes during index creation
- Writers don't get blocked by long-running index creation
- Index is created successfully while writes continue

**Expected duration:** ~2-3 minutes

**What it verifies:**

- Concurrent writer can INSERT rows during index creation
- No 2-second timeouts (means no write locks)
- Index exists after creation
- Generated SQL contains CONCURRENTLY keyword

**Test file:** `concurrent_index_integration_test.go`

### 3. TestPhaseFailureRecovery_BackfillResumability

**What it tests:**

- Backfill operations can be safely resumed after interruption
- WHERE clause (IS NULL) makes operation idempotent
- Partially completed backfills can be resumed without data corruption

**Expected duration:** ~1-2 minutes

**Test scenarios:**

1. **Partial backfill then resume:** Backfill 50% of rows, then resume to complete
2. **Re-run backfill (idempotency):** Re-running completed backfill is a no-op
3. **Backfill value updates:** WHERE bio IS NULL prevents overwrites

**Test file:** `failure_recovery_integration_test.go`

### 4. TestPhaseFailureRecovery_IndexCreation

**What it tests:**

- Index creation can be retried after failure
- DROP INDEX + CREATE INDEX works correctly
- Index is usable after recovery

**Expected duration:** < 1 minute

**Test file:** `failure_recovery_integration_test.go`

### 5. TestBenchmarkBackfillPerformance_VariousTableSizes

**What it benchmarks:**

- Backfill performance for 10K, 50K, 100K, 500K rows
- Throughput (rows/second) for each size
- Setup time vs backfill time

**Expected duration:** ~10-15 minutes

**Test file:** `performance_benchmark_test.go`

### 6. TestBenchmarkIndexCreation_VariousTableSizes

**What it benchmarks:**

- Concurrent index creation for various table sizes
- Unique index creation performance
- Index creation time scaling

**Expected duration:** ~10-15 minutes

**Test file:** `performance_benchmark_test.go`

### 7. TestBenchmarkDDLOperations

**What it benchmarks:**

- ADD COLUMN (nullable)
- ADD COLUMN with DEFAULT
- DROP COLUMN
- RENAME COLUMN

**Expected duration:** ~2-3 minutes

**Test file:** `performance_benchmark_test.go`

## Interpreting Results

### Success Criteria

✅ **Test passes if:**
- All 1M rows are backfilled (0 NULL values remaining)
- Backfilled values match expected value
- Total time is within 2x of expected time (< 800s for 1M rows)
- No database errors or constraint violations

❌ **Test fails if:**
- NULL rows remain after backfill
- Wrong backfill value
- Database errors (deadlock, timeout, constraint violation)
- Excessive time (> 20 minutes for 1M rows)

### Performance Analysis

**If backfill is slower than expected:**

1. **Check batch size:**
   ```sql
   -- Ensure batches are 5,000 rows (from GenerateBatchedUpdate)
   SELECT COUNT(*) FROM pg_stat_activity WHERE query LIKE '%LIMIT 5000%';
   ```

2. **Check pg_sleep:**
   ```sql
   -- Ensure 1s sleep between batches
   SELECT query FROM pg_stat_activity WHERE query LIKE '%pg_sleep(1)%';
   ```

3. **Check table bloat:**
   ```sql
   -- Dead tuples indicate AUTOVACUUM isn't keeping up
   SELECT n_dead_tup FROM pg_stat_user_tables WHERE relname = 'backfill_test_1m';
   ```

4. **Check concurrent load:**
   ```sql
   -- Other queries competing for resources
   SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active';
   ```

## Cleanup

Tests automatically clean up after themselves, but if interrupted:

```bash
# Connect to test database
psql astroladb_test

# Drop test tables
DROP TABLE IF EXISTS backfill_test_1m CASCADE;
```

## Continuous Integration

These tests are **not run in CI** by default due to:
- Long execution time (10+ minutes)
- Database dependency
- Disk space requirements

To run in CI, add:
```yaml
# .github/workflows/integration.yml
- name: Run Integration Tests
  run: go test -tags=integration -timeout 30m -v ./internal/engine/runner
  env:
    DATABASE_URL: ${{ secrets.TEST_DATABASE_URL }}
```

## Troubleshooting

### "DATABASE_URL not set - skipping integration test"

**Solution:** Set the `DATABASE_URL` or `TEST_DATABASE_URL` environment variable.

### "connection refused"

**Solution:** Ensure PostgreSQL is running and accessible:
```bash
pg_isready -h localhost -p 5432
```

### "disk full" errors

**Solution:** Free up disk space or use a different database with more capacity:
```bash
df -h /var/lib/postgresql/data
```

### Slow performance

**Possible causes:**
- Slow disk I/O (use SSD if possible)
- Insufficient PostgreSQL resources (increase `shared_buffers`, `work_mem`)
- Other processes competing for resources
- Network latency (if database is remote)

**Solutions:**
```sql
-- Check PostgreSQL settings
SHOW shared_buffers;
SHOW work_mem;

-- Increase if needed (requires restart)
ALTER SYSTEM SET shared_buffers = '2GB';
ALTER SYSTEM SET work_mem = '64MB';
```

## Running All Tests

Use the comprehensive test runner to run all integration tests:

```bash
# Make script executable
chmod +x run_all_integration_tests.sh

# Run all tests (total duration: ~40-50 minutes)
./run_all_integration_tests.sh

# Or run individual tests
go test -tags=integration -run TestConcurrentIndexCreation_NoWriteLocks -v ./internal/engine/runner
go test -tags=integration -run TestPhaseFailureRecovery_BackfillResumability -v ./internal/engine/runner
go test -tags=integration -run TestBenchmarkBackfillPerformance -v ./internal/engine/runner
```

The test runner will:

1. Detect PostgreSQL container
2. Verify database connection
3. Run all 7 integration tests
4. Report results summary

## Next Steps

After all integration tests pass:

1. **Week 8-9 Completion:**
   - ✅ Large dataset test (1M rows) - DONE
   - ✅ Concurrent index creation - DONE
   - ✅ Phase failure recovery - DONE
   - ✅ Performance benchmarks - DONE

2. **Week 10:**
   - Update user documentation
   - Tag v0.0.9 release
