package engine

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

const (
	// DefaultLockTimeout is the maximum time to wait for table locks (2 seconds).
	// Hardcoded for simplicity - fail fast if lock is unavailable.
	DefaultLockTimeout = "2s"

	// DefaultStatementTimeout is the maximum execution time for a single statement (30 seconds).
	// Prevents runaway queries during DDL operations.
	DefaultStatementTimeout = "30s"

	// DefaultBatchSize is the number of rows to process per batch (5,000 rows).
	// Industry standard: Rails (1K), production migrations (2K-5K).
	// Optimal balance between throughput and lock duration.
	DefaultBatchSize = 5000

	// DefaultBatchSleep is the sleep duration between batches (1 second).
	// Allows AUTOVACUUM to clean up dead tuples and prevents table bloat.
	DefaultBatchSleep = 1
)

// InjectLockTimeout adds lock timeout settings to SQL statements.
// Only injects timeouts for PostgreSQL - SQLite doesn't support SET commands.
//
// For PostgreSQL transactions (DDL, Data phases):
//
//	BEGIN;
//	  SET lock_timeout = '2s';
//	  SET statement_timeout = '30s';
//	  ... user SQL ...
//	COMMIT;
//
// For PostgreSQL non-transactional (Index phase):
//
//	SET lock_timeout = '2s';
//	... user SQL ...
//
// For SQLite (all phases):
//
//	BEGIN;
//	  ... user SQL ...
//	COMMIT;
func InjectLockTimeout(sql string, inTransaction bool, dialect string) string {
	// SQLite doesn't support SET commands - just wrap in transaction if needed
	if dialect == "sqlite" {
		if inTransaction {
			return fmt.Sprintf("BEGIN;\n  %s\nCOMMIT;", strutil.Indent(sql, 2))
		}
		return sql
	}

	// PostgreSQL: inject lock timeout settings
	if inTransaction {
		// Wrap in transaction with timeout settings
		return fmt.Sprintf(
			"BEGIN;\n  SET lock_timeout = '%s';\n  SET statement_timeout = '%s';\n\n  %s\nCOMMIT;",
			DefaultLockTimeout,
			DefaultStatementTimeout,
			strutil.Indent(sql, 2),
		)
	}

	// Non-transactional (for concurrent index creation)
	return fmt.Sprintf(
		"SET lock_timeout = '%s';\n%s",
		DefaultLockTimeout,
		sql,
	)
}

// GenerateConcurrentIndex generates SQL for CREATE INDEX CONCURRENTLY with validation.
// PostgreSQL requires concurrent index creation to run outside transactions.
//
// Returns SQL that:
// 1. Creates the index with CONCURRENTLY keyword (zero downtime)
// 2. Validates the index was created successfully (indisvalid=true)
// 3. Raises an error if index creation failed
func GenerateConcurrentIndex(idx *ast.CreateIndex, dialect string) string {
	// Build index name (auto-generate if not provided)
	indexName := idx.Name
	if indexName == "" {
		// Auto-generate: idx_table_column1_column2
		indexName = fmt.Sprintf("idx_%s_%s", idx.Table_, strings.Join(idx.Columns, "_"))
	}

	// Build column list
	columnList := strings.Join(idx.Columns, ", ")

	// Build CREATE INDEX statement
	uniqueKeyword := ""
	if idx.Unique {
		uniqueKeyword = "UNIQUE "
	}

	// Full table name (with namespace if provided)
	tableName := idx.Table_
	if idx.Namespace != "" {
		tableName = fmt.Sprintf("%s_%s", idx.Namespace, idx.Table_)
	}

	createSQL := fmt.Sprintf(
		"CREATE %sINDEX CONCURRENTLY %s ON %s(%s);",
		uniqueKeyword,
		indexName,
		tableName,
		columnList,
	)

	// PostgreSQL-specific validation
	if dialect == "postgres" || dialect == "postgresql" {
		validationSQL := fmt.Sprintf(`
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_index
    WHERE indexrelid = '%s'::regclass
      AND indisvalid = true
  ) THEN
    RAISE EXCEPTION 'Index creation failed or is invalid: %s';
  END IF;
END $$;`,
			indexName,
			indexName,
		)
		return createSQL + "\n" + validationSQL
	}

	// For other dialects (SQLite doesn't support CONCURRENTLY)
	// Just return regular CREATE INDEX without validation
	return strings.ReplaceAll(createSQL, " CONCURRENTLY", "")
}

// GenerateBatchedUpdate generates SQL for batched UPDATE operations with AUTOVACUUM support.
// Prevents table locks on large datasets by processing in chunks.
//
// Features:
// - Batches updates into chunks of 5,000 rows
// - Commits after each batch (releases locks)
// - Sleeps 1 second between batches (allows AUTOVACUUM to clean dead tuples)
// - Progress logging with RAISE NOTICE
//
// Example output:
//
//	DO $$
//	DECLARE
//	  batch_size INT := 5000;
//	  processed INT := 0;
//	BEGIN
//	  LOOP
//	    UPDATE auth_user SET bio = 'default' WHERE bio IS NULL AND id IN (
//	      SELECT id FROM auth_user WHERE bio IS NULL LIMIT batch_size
//	    );
//	    EXIT WHEN NOT FOUND;
//	    processed := processed + batch_size;
//	    RAISE NOTICE 'Progress: % rows', processed;
//	    COMMIT;
//	    PERFORM pg_sleep(1);
//	  END LOOP;
//	END $$;
func GenerateBatchedUpdate(table, column, value, whereClause string) string {
	// Generates a batched UPDATE with pg_sleep to allow AUTOVACUUM between batches.
	// Called when extractBackfill() detects a column with .backfill() modifier.
	return fmt.Sprintf(`DO $$
DECLARE
  batch_size INT := %d;
  processed INT := 0;
BEGIN
  LOOP
    UPDATE %s
    SET %s = %s
    WHERE %s AND id IN (
      SELECT id FROM %s WHERE %s LIMIT batch_size
    );

    EXIT WHEN NOT FOUND;

    processed := processed + batch_size;
    RAISE NOTICE 'Progress: %% rows', processed;

    COMMIT;
    PERFORM pg_sleep(%d);
  END LOOP;
END $$;`,
		DefaultBatchSize,
		table,
		column,
		value,
		whereClause,
		table,
		whereClause,
		DefaultBatchSleep,
	)
}

// WrapInTransaction wraps SQL in a transaction block.
// Utility function for DDL and Data phases.
func WrapInTransaction(sql string) string {
	return fmt.Sprintf("BEGIN;\n%s\nCOMMIT;", strutil.Indent(sql, 2))
}
