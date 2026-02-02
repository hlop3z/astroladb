// Package engine provides the core migration execution functionality.
package engine

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/dialect"
)

// Version tracking table schema:
// CREATE TABLE alab_migrations (
//     revision          VARCHAR(20) PRIMARY KEY,
//     applied_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
//     checksum          VARCHAR(64),
//     exec_time_ms      INTEGER,
//     description       TEXT,
//     sql_checksum      VARCHAR(64),
//     squashed_through  VARCHAR(20),
//     applied_order     INTEGER
// )

const (
	// MigrationTableName is the name of the version tracking table.
	MigrationTableName = "alab_migrations"

	// LockTableName is the name of the migration lock table.
	// Used to prevent concurrent migration execution (Liquibase-style).
	LockTableName = "alab_migrations_lock"

	// DefaultLockTimeout is the default timeout for acquiring a migration lock.
	DefaultLockTimeout = 30 * time.Second

	// lockRetryInterval is the interval between lock acquisition retries.
	lockRetryInterval = 500 * time.Millisecond
)

// AppliedMigration represents a migration that has been applied to the database.
type AppliedMigration struct {
	Revision        string
	AppliedAt       time.Time
	Checksum        string
	ExecTimeMs      int
	Description     string
	SQLChecksum     string
	SquashedThrough string
	AppliedOrder    int
}

// VersionManager handles version tracking for migrations.
// It manages the alab_migrations table that tracks which migrations have been applied.
type VersionManager struct {
	db      *sql.DB
	dialect dialect.Dialect
}

// NewVersionManager creates a new VersionManager.
func NewVersionManager(db *sql.DB, d dialect.Dialect) *VersionManager {
	return &VersionManager{
		db:      db,
		dialect: d,
	}
}

// EnsureTable creates the migration tracking table if it doesn't exist.
// Also adds new columns to existing tables for backward compatibility.
func (v *VersionManager) EnsureTable(ctx context.Context) error {
	sql := v.createTableSQL()
	_, err := v.db.ExecContext(ctx, sql)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to create migrations table").
			WithSQL(sql)
	}

	// Add new columns if they don't exist (safe for existing installations)
	for _, col := range v.newColumns() {
		// Ignore errors — column may already exist
		_, _ = v.db.ExecContext(ctx, col)
	}

	return nil
}

// newColumns returns ALTER TABLE statements to add columns introduced after the initial schema.
func (v *VersionManager) newColumns() []string {
	qt := v.dialect.QuoteIdent(MigrationTableName)
	switch v.dialect.Name() {
	case "sqlite":
		return []string{
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN description TEXT", qt),
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN sql_checksum TEXT", qt),
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN squashed_through TEXT", qt),
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN applied_order INTEGER", qt),
		}
	default:
		return []string{
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS description TEXT", qt),
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS sql_checksum VARCHAR(64)", qt),
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS squashed_through VARCHAR(20)", qt),
			fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS applied_order INTEGER", qt),
		}
	}
}

// createTableSQL returns the CREATE TABLE statement for the migrations table.
func (v *VersionManager) createTableSQL() string {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)

	switch v.dialect.Name() {
	case "postgres":
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    revision          VARCHAR(20) PRIMARY KEY,
    applied_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum          VARCHAR(64),
    exec_time_ms      INTEGER,
    description       TEXT,
    sql_checksum      VARCHAR(64),
    squashed_through  VARCHAR(20),
    applied_order     INTEGER
)`, quotedTable)

	case "sqlite":
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    revision          TEXT PRIMARY KEY,
    applied_at        TEXT NOT NULL DEFAULT (datetime('now')),
    checksum          TEXT,
    exec_time_ms      INTEGER,
    description       TEXT,
    sql_checksum      TEXT,
    squashed_through  TEXT,
    applied_order     INTEGER
)`, quotedTable)

	default:
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    revision          VARCHAR(20) PRIMARY KEY,
    applied_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum          VARCHAR(64),
    exec_time_ms      INTEGER,
    description       TEXT,
    sql_checksum      VARCHAR(64),
    squashed_through  VARCHAR(20),
    applied_order     INTEGER
)`, quotedTable)
	}
}

// GetApplied returns all applied migrations ordered by revision.
func (v *VersionManager) GetApplied(ctx context.Context) ([]AppliedMigration, error) {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)
	query := fmt.Sprintf(
		"SELECT revision, applied_at, checksum, exec_time_ms, description, sql_checksum, squashed_through, applied_order FROM %s ORDER BY revision ASC",
		quotedTable,
	)

	rows, err := v.db.QueryContext(ctx, query)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to query applied migrations").
			WithSQL(query)
	}
	defer rows.Close()

	var migrations []AppliedMigration
	for rows.Next() {
		var m AppliedMigration
		var checksum, description, sqlChecksum, squashedThrough sql.NullString
		var execTime, appliedOrder sql.NullInt64
		var appliedAt interface{}

		err := rows.Scan(&m.Revision, &appliedAt, &checksum, &execTime, &description, &sqlChecksum, &squashedThrough, &appliedOrder)
		if err != nil {
			return nil, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to scan migration row")
		}

		m.AppliedAt = v.parseAppliedAt(appliedAt)

		if checksum.Valid {
			m.Checksum = checksum.String
		}
		if execTime.Valid {
			m.ExecTimeMs = int(execTime.Int64)
		}
		if description.Valid {
			m.Description = description.String
		}
		if sqlChecksum.Valid {
			m.SQLChecksum = sqlChecksum.String
		}
		if squashedThrough.Valid {
			m.SquashedThrough = squashedThrough.String
		}
		if appliedOrder.Valid {
			m.AppliedOrder = int(appliedOrder.Int64)
		}

		migrations = append(migrations, m)
	}

	if err := rows.Err(); err != nil {
		return nil, alerr.Wrap(alerr.ErrSQLExecution, err, "error iterating migration rows")
	}

	return migrations, nil
}

// parseAppliedAt converts the database timestamp to time.Time.
func (v *VersionManager) parseAppliedAt(val interface{}) time.Time {
	switch t := val.(type) {
	case time.Time:
		return t
	case string:
		// SQLite stores timestamps as strings
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, t); err == nil {
				return parsed
			}
		}
		// Fallback to current time if parsing fails
		return time.Now()
	case []byte:
		return v.parseAppliedAt(string(t))
	default:
		return time.Now()
	}
}

// ApplyRecord holds the fields to record when a migration is applied.
type ApplyRecord struct {
	Revision        string
	Checksum        string
	ExecTime        time.Duration
	Description     string
	SQLChecksum     string
	SquashedThrough string
}

// RecordApplied records that a migration has been applied.
// applied_order is auto-assigned as MAX(applied_order)+1.
func (v *VersionManager) RecordApplied(ctx context.Context, rec ApplyRecord) error {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)
	execTimeMs := int(rec.ExecTime.Milliseconds())

	// Compute next applied_order
	nextOrder, err := v.nextAppliedOrder(ctx)
	if err != nil {
		return err
	}

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		query = fmt.Sprintf(
			"INSERT INTO %s (revision, checksum, exec_time_ms, description, sql_checksum, squashed_through, applied_order) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			quotedTable,
		)
		args = []interface{}{rec.Revision, rec.Checksum, execTimeMs, nullIfEmpty(rec.Description), nullIfEmpty(rec.SQLChecksum), nullIfEmpty(rec.SquashedThrough), nextOrder}

	case "sqlite":
		query = fmt.Sprintf(
			"INSERT INTO %s (revision, checksum, exec_time_ms, description, sql_checksum, squashed_through, applied_order) VALUES (?, ?, ?, ?, ?, ?, ?)",
			quotedTable,
		)
		args = []interface{}{rec.Revision, rec.Checksum, execTimeMs, nullIfEmpty(rec.Description), nullIfEmpty(rec.SQLChecksum), nullIfEmpty(rec.SquashedThrough), nextOrder}

	default:
		query = fmt.Sprintf(
			"INSERT INTO %s (revision, checksum, exec_time_ms, description, sql_checksum, squashed_through, applied_order) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			quotedTable,
		)
		args = []interface{}{rec.Revision, rec.Checksum, execTimeMs, nullIfEmpty(rec.Description), nullIfEmpty(rec.SQLChecksum), nullIfEmpty(rec.SquashedThrough), nextOrder}
	}

	_, err = v.db.ExecContext(ctx, query, args...)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to record applied migration").
			With("revision", rec.Revision).
			WithSQL(query)
	}

	return nil
}

// nextAppliedOrder returns the next applied_order value (MAX+1, starting at 1).
func (v *VersionManager) nextAppliedOrder(ctx context.Context) (int, error) {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)
	query := fmt.Sprintf("SELECT COALESCE(MAX(applied_order), 0) FROM %s", quotedTable)

	var maxOrder int
	err := v.db.QueryRowContext(ctx, query).Scan(&maxOrder)
	if err != nil {
		return 0, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to get max applied_order").
			WithSQL(query)
	}
	return maxOrder + 1, nil
}

// nullIfEmpty returns nil if the string is empty, otherwise the string pointer.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// RecordRollback removes a migration record (used during rollback).
func (v *VersionManager) RecordRollback(ctx context.Context, revision string) error {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		query = fmt.Sprintf("DELETE FROM %s WHERE revision = $1", quotedTable)
		args = []interface{}{revision}

	case "sqlite":
		query = fmt.Sprintf("DELETE FROM %s WHERE revision = ?", quotedTable)
		args = []interface{}{revision}

	default:
		query = fmt.Sprintf("DELETE FROM %s WHERE revision = $1", quotedTable)
		args = []interface{}{revision}
	}

	result, err := v.db.ExecContext(ctx, query, args...)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to remove migration record").
			With("revision", revision).
			WithSQL(query)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return alerr.New(alerr.ErrMigrationNotFound, "migration not found in version table").
			With("revision", revision)
	}

	return nil
}

// IsApplied checks if a migration has been applied.
func (v *VersionManager) IsApplied(ctx context.Context, revision string) (bool, error) {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		query = fmt.Sprintf("SELECT 1 FROM %s WHERE revision = $1 LIMIT 1", quotedTable)
		args = []interface{}{revision}

	case "sqlite":
		query = fmt.Sprintf("SELECT 1 FROM %s WHERE revision = ? LIMIT 1", quotedTable)
		args = []interface{}{revision}

	default:
		query = fmt.Sprintf("SELECT 1 FROM %s WHERE revision = $1 LIMIT 1", quotedTable)
		args = []interface{}{revision}
	}

	var exists int
	err := v.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to check if migration is applied").
			With("revision", revision).
			WithSQL(query)
	}

	return true, nil
}

// GetChecksum returns the checksum of an applied migration.
func (v *VersionManager) GetChecksum(ctx context.Context, revision string) (string, error) {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		query = fmt.Sprintf("SELECT checksum FROM %s WHERE revision = $1", quotedTable)
		args = []interface{}{revision}

	case "sqlite":
		query = fmt.Sprintf("SELECT checksum FROM %s WHERE revision = ?", quotedTable)
		args = []interface{}{revision}

	default:
		query = fmt.Sprintf("SELECT checksum FROM %s WHERE revision = $1", quotedTable)
		args = []interface{}{revision}
	}

	var checksum sql.NullString
	err := v.db.QueryRowContext(ctx, query, args...).Scan(&checksum)
	if err == sql.ErrNoRows {
		return "", alerr.New(alerr.ErrMigrationNotFound, "migration not found").
			With("revision", revision)
	}
	if err != nil {
		return "", alerr.Wrap(alerr.ErrSQLExecution, err, "failed to get migration checksum").
			With("revision", revision).
			WithSQL(query)
	}

	return checksum.String, nil
}

// VerifyChecksum checks if the checksum matches for an applied migration.
// Returns an error if the migration was applied with a different checksum.
func (v *VersionManager) VerifyChecksum(ctx context.Context, revision, expectedChecksum string) error {
	actualChecksum, err := v.GetChecksum(ctx, revision)
	if err != nil {
		return err
	}

	if actualChecksum != "" && expectedChecksum != "" && actualChecksum != expectedChecksum {
		return alerr.New(alerr.ErrMigrationChecksum, "migration checksum mismatch").
			With("revision", revision).
			With("expected", expectedChecksum).
			With("actual", actualChecksum)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Migration Lock (Liquibase-style)
// -----------------------------------------------------------------------------

// LockInfo contains information about the current migration lock.
type LockInfo struct {
	Locked   bool
	LockedAt *time.Time
	LockedBy string
}

// EnsureLockTable creates the migration lock table if it doesn't exist.
// The lock table has a single row with id=1 that is used for locking.
func (v *VersionManager) EnsureLockTable(ctx context.Context) error {
	createSQL := v.createLockTableSQL()
	_, err := v.db.ExecContext(ctx, createSQL)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to create migrations lock table").
			WithSQL(createSQL)
	}

	// Ensure the single lock row exists
	insertSQL := v.insertLockRowSQL()
	_, err = v.db.ExecContext(ctx, insertSQL)
	if err != nil {
		// Ignore duplicate key errors - row already exists
		// This is expected on subsequent calls
	}

	return nil
}

// createLockTableSQL returns the CREATE TABLE statement for the lock table.
func (v *VersionManager) createLockTableSQL() string {
	quotedTable := v.dialect.QuoteIdent(LockTableName)

	switch v.dialect.Name() {
	case "postgres":
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    id          INTEGER PRIMARY KEY,
    locked      BOOLEAN NOT NULL DEFAULT FALSE,
    locked_at   TIMESTAMPTZ,
    locked_by   VARCHAR(255)
)`, quotedTable)

	case "sqlite":
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    id          INTEGER PRIMARY KEY,
    locked      INTEGER NOT NULL DEFAULT 0,
    locked_at   TEXT,
    locked_by   TEXT
)`, quotedTable)

	default:
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    id          INTEGER PRIMARY KEY,
    locked      BOOLEAN NOT NULL DEFAULT FALSE,
    locked_at   TIMESTAMP,
    locked_by   VARCHAR(255)
)`, quotedTable)
	}
}

// insertLockRowSQL returns the INSERT statement for the initial lock row.
func (v *VersionManager) insertLockRowSQL() string {
	quotedTable := v.dialect.QuoteIdent(LockTableName)

	switch v.dialect.Name() {
	case "postgres":
		return fmt.Sprintf(
			"INSERT INTO %s (id, locked) VALUES (1, FALSE) ON CONFLICT (id) DO NOTHING",
			quotedTable,
		)
	case "sqlite":
		return fmt.Sprintf(
			"INSERT OR IGNORE INTO %s (id, locked) VALUES (1, 0)",
			quotedTable,
		)
	default:
		return fmt.Sprintf(
			"INSERT INTO %s (id, locked) VALUES (1, FALSE) ON CONFLICT (id) DO NOTHING",
			quotedTable,
		)
	}
}

// AcquireLock attempts to acquire the migration lock with a timeout.
// Returns nil if lock was acquired, error otherwise.
// The lock is identified by the hostname and process ID for debugging.
func (v *VersionManager) AcquireLock(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}

	// Ensure lock table exists
	if err := v.EnsureLockTable(ctx); err != nil {
		return err
	}

	// Get lock identifier (hostname + pid)
	lockID := v.getLockIdentifier()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to acquire the lock
		acquired, err := v.tryAcquireLock(ctx, lockID)
		if err != nil {
			lastErr = err
			time.Sleep(lockRetryInterval)
			continue
		}

		if acquired {
			return nil
		}

		// Lock is held by someone else, wait and retry
		time.Sleep(lockRetryInterval)
	}

	// Timeout - get info about who holds the lock
	info, _ := v.GetLockInfo(ctx)
	if info != nil && info.Locked {
		return alerr.New(alerr.ErrMigrationConflict, "migration lock timeout: another process is running migrations").
			With("locked_by", info.LockedBy).
			With("locked_at", info.LockedAt).
			With("timeout", timeout.String())
	}

	if lastErr != nil {
		return alerr.Wrap(alerr.ErrMigrationConflict, lastErr, "failed to acquire migration lock")
	}

	return alerr.New(alerr.ErrMigrationConflict, "migration lock timeout")
}

// tryAcquireLock attempts a single lock acquisition.
// Returns (true, nil) if lock was acquired, (false, nil) if lock is held by another.
func (v *VersionManager) tryAcquireLock(ctx context.Context, lockID string) (bool, error) {
	quotedTable := v.dialect.QuoteIdent(LockTableName)

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		// Atomic compare-and-swap: only update if not locked
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = TRUE, locked_at = CURRENT_TIMESTAMP, locked_by = $1
			WHERE id = 1 AND locked = FALSE`,
			quotedTable,
		)
		args = []interface{}{lockID}

	case "sqlite":
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = 1, locked_at = datetime('now'), locked_by = ?
			WHERE id = 1 AND locked = 0`,
			quotedTable,
		)
		args = []interface{}{lockID}

	default:
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = TRUE, locked_at = CURRENT_TIMESTAMP, locked_by = $1
			WHERE id = 1 AND locked = FALSE`,
			quotedTable,
		)
		args = []interface{}{lockID}
	}

	result, err := v.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to acquire lock").
			WithSQL(query)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to check lock acquisition")
	}

	// If 1 row was updated, we got the lock
	return rowsAffected == 1, nil
}

// ReleaseLock releases the migration lock.
// Only releases the lock if it was acquired by this process (same lock ID).
func (v *VersionManager) ReleaseLock(ctx context.Context) error {
	quotedTable := v.dialect.QuoteIdent(LockTableName)
	lockID := v.getLockIdentifier()

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		// Only release if we hold the lock
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = FALSE, locked_at = NULL, locked_by = NULL
			WHERE id = 1 AND locked_by = $1`,
			quotedTable,
		)
		args = []interface{}{lockID}

	case "sqlite":
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = 0, locked_at = NULL, locked_by = NULL
			WHERE id = 1 AND locked_by = ?`,
			quotedTable,
		)
		args = []interface{}{lockID}

	default:
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = FALSE, locked_at = NULL, locked_by = NULL
			WHERE id = 1 AND locked_by = $1`,
			quotedTable,
		)
		args = []interface{}{lockID}
	}

	_, err := v.db.ExecContext(ctx, query, args...)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to release lock").
			WithSQL(query)
	}

	return nil
}

// ForceReleaseLock forcefully releases the migration lock regardless of who holds it.
// Use with caution - only for recovering from stuck locks.
func (v *VersionManager) ForceReleaseLock(ctx context.Context) error {
	quotedTable := v.dialect.QuoteIdent(LockTableName)

	var query string

	switch v.dialect.Name() {
	case "postgres":
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = FALSE, locked_at = NULL, locked_by = NULL
			WHERE id = 1`,
			quotedTable,
		)

	case "sqlite":
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = 0, locked_at = NULL, locked_by = NULL
			WHERE id = 1`,
			quotedTable,
		)

	default:
		query = fmt.Sprintf(`
			UPDATE %s
			SET locked = FALSE, locked_at = NULL, locked_by = NULL
			WHERE id = 1`,
			quotedTable,
		)
	}

	_, err := v.db.ExecContext(ctx, query)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to force release lock").
			WithSQL(query)
	}

	return nil
}

// GetLockInfo returns information about the current lock state.
func (v *VersionManager) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	quotedTable := v.dialect.QuoteIdent(LockTableName)
	query := fmt.Sprintf(
		"SELECT locked, locked_at, locked_by FROM %s WHERE id = 1",
		quotedTable,
	)

	var info LockInfo
	var locked interface{}
	var lockedAt sql.NullString
	var lockedBy sql.NullString

	err := v.db.QueryRowContext(ctx, query).Scan(&locked, &lockedAt, &lockedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to get lock info").
			WithSQL(query)
	}

	// Parse locked (handles both boolean and integer)
	switch l := locked.(type) {
	case bool:
		info.Locked = l
	case int64:
		info.Locked = l != 0
	case int:
		info.Locked = l != 0
	}

	if lockedAt.Valid && lockedAt.String != "" {
		t := v.parseAppliedAt(lockedAt.String)
		info.LockedAt = &t
	}

	if lockedBy.Valid {
		info.LockedBy = lockedBy.String
	}

	return &info, nil
}

// IsLocked returns true if the migration lock is currently held.
func (v *VersionManager) IsLocked(ctx context.Context) (bool, error) {
	info, err := v.GetLockInfo(ctx)
	if err != nil {
		return false, err
	}
	if info == nil {
		return false, nil
	}
	return info.Locked, nil
}

// getLockIdentifier returns a unique identifier for this process.
func (v *VersionManager) getLockIdentifier() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s:%d", hostname, os.Getpid())
}
