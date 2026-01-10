// Package engine provides the core migration execution functionality.
package engine

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/dialect"
)

// Version tracking table schema:
// CREATE TABLE alab_migrations (
//     revision     VARCHAR(20) PRIMARY KEY,
//     applied_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
//     checksum     VARCHAR(64),
//     exec_time_ms INTEGER
// )

const (
	// MigrationTableName is the name of the version tracking table.
	MigrationTableName = "alab_migrations"
)

// AppliedMigration represents a migration that has been applied to the database.
type AppliedMigration struct {
	Revision   string
	AppliedAt  time.Time
	Checksum   string
	ExecTimeMs int
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
func (v *VersionManager) EnsureTable(ctx context.Context) error {
	sql := v.createTableSQL()
	_, err := v.db.ExecContext(ctx, sql)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to create migrations table").
			WithSQL(sql)
	}
	return nil
}

// createTableSQL returns the CREATE TABLE statement for the migrations table.
func (v *VersionManager) createTableSQL() string {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)

	switch v.dialect.Name() {
	case "postgres":
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    revision     VARCHAR(20) PRIMARY KEY,
    applied_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum     VARCHAR(64),
    exec_time_ms INTEGER
)`, quotedTable)

	case "sqlite":
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    revision     TEXT PRIMARY KEY,
    applied_at   TEXT NOT NULL DEFAULT (datetime('now')),
    checksum     TEXT,
    exec_time_ms INTEGER
)`, quotedTable)

	default:
		// Fallback to PostgreSQL syntax
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    revision     VARCHAR(20) PRIMARY KEY,
    applied_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum     VARCHAR(64),
    exec_time_ms INTEGER
)`, quotedTable)
	}
}

// GetApplied returns all applied migrations ordered by revision.
func (v *VersionManager) GetApplied(ctx context.Context) ([]AppliedMigration, error) {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)
	query := fmt.Sprintf(
		"SELECT revision, applied_at, checksum, exec_time_ms FROM %s ORDER BY revision ASC",
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
		var checksum sql.NullString
		var execTime sql.NullInt64
		var appliedAt interface{}

		err := rows.Scan(&m.Revision, &appliedAt, &checksum, &execTime)
		if err != nil {
			return nil, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to scan migration row")
		}

		// Handle applied_at based on dialect
		m.AppliedAt = v.parseAppliedAt(appliedAt)

		if checksum.Valid {
			m.Checksum = checksum.String
		}
		if execTime.Valid {
			m.ExecTimeMs = int(execTime.Int64)
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

// RecordApplied records that a migration has been applied.
func (v *VersionManager) RecordApplied(ctx context.Context, revision, checksum string, execTime time.Duration) error {
	quotedTable := v.dialect.QuoteIdent(MigrationTableName)
	execTimeMs := int(execTime.Milliseconds())

	var query string
	var args []interface{}

	switch v.dialect.Name() {
	case "postgres":
		query = fmt.Sprintf(
			"INSERT INTO %s (revision, checksum, exec_time_ms) VALUES ($1, $2, $3)",
			quotedTable,
		)
		args = []interface{}{revision, checksum, execTimeMs}

	case "sqlite":
		query = fmt.Sprintf(
			"INSERT INTO %s (revision, checksum, exec_time_ms) VALUES (?, ?, ?)",
			quotedTable,
		)
		args = []interface{}{revision, checksum, execTimeMs}

	default:
		query = fmt.Sprintf(
			"INSERT INTO %s (revision, checksum, exec_time_ms) VALUES ($1, $2, $3)",
			quotedTable,
		)
		args = []interface{}{revision, checksum, execTimeMs}
	}

	_, err := v.db.ExecContext(ctx, query, args...)
	if err != nil {
		return alerr.Wrap(alerr.ErrSQLExecution, err, "failed to record applied migration").
			With("revision", revision).
			WithSQL(query)
	}

	return nil
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
