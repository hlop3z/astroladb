// Package astroladb provides the public API for the Alab database migration tool.
// It offers a clean, ergonomic interface for schema management, migration operations,
// and schema export in various formats (OpenAPI, JSON Schema, TypeScript).
package astroladb

import (
	"errors"
	"fmt"

	"github.com/hlop3z/astroladb/internal/chain"
)

// Sentinel errors for common error conditions.
// Use errors.Is() to check for these errors.
var (
	// ErrMissingDatabaseURL is returned when no database URL is provided.
	ErrMissingDatabaseURL = errors.New("astroladb: database URL required")

	// ErrConnectionFailed is returned when the database connection fails.
	ErrConnectionFailed = errors.New("astroladb: connection failed")

	// ErrMigrationFailed is returned when a migration fails to execute.
	ErrMigrationFailed = errors.New("astroladb: migration failed")

	// ErrSchemaInvalid is returned when a schema file is malformed or invalid.
	ErrSchemaInvalid = errors.New("astroladb: schema invalid")

	// ErrUnsupportedDialect is returned when the database dialect is not supported.
	ErrUnsupportedDialect = errors.New("astroladb: unsupported dialect")

	// ErrSchemaNotFound is returned when a referenced schema file does not exist.
	ErrSchemaNotFound = errors.New("astroladb: schema not found")

	// ErrExportFormatUnknown is returned when an unknown export format is requested.
	ErrExportFormatUnknown = errors.New("astroladb: unknown export format")
)

// MigrationError provides detailed information about a failed migration.
// It includes the revision, operation, SQL statement, and underlying cause.
type MigrationError struct {
	// Revision is the migration revision that failed (e.g., "001", "20240101120000").
	Revision string

	// Operation describes the operation that failed (e.g., "CREATE TABLE", "ADD COLUMN").
	Operation string

	// SQL is the SQL statement that caused the failure.
	SQL string

	// Cause is the underlying error from the database driver.
	Cause error
}

// Error returns a formatted error message.
func (e *MigrationError) Error() string {
	if e.SQL != "" {
		return fmt.Sprintf("astroladb: migration %s failed during %s: %v\nSQL: %s",
			e.Revision, e.Operation, e.Cause, e.SQL)
	}
	return fmt.Sprintf("astroladb: migration %s failed during %s: %v",
		e.Revision, e.Operation, e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *MigrationError) Unwrap() error {
	return e.Cause
}

// Is reports whether this error matches the target error.
func (e *MigrationError) Is(target error) bool {
	return target == ErrMigrationFailed
}

// SchemaError provides detailed information about a schema validation error.
type SchemaError struct {
	// Namespace is the schema namespace (e.g., "auth", "billing").
	Namespace string

	// Table is the table name where the error occurred.
	Table string

	// Column is the column name where the error occurred (if applicable).
	Column string

	// File is the path to the schema file.
	File string

	// Line is the line number where the error occurred (if known).
	Line int

	// Message describes what went wrong.
	Message string

	// Cause is the underlying error.
	Cause error
}

// Error returns a formatted error message.
func (e *SchemaError) Error() string {
	location := ""
	if e.File != "" {
		if e.Line > 0 {
			location = fmt.Sprintf("%s:%d: ", e.File, e.Line)
		} else {
			location = e.File + ": "
		}
	}

	target := ""
	if e.Namespace != "" && e.Table != "" {
		if e.Column != "" {
			target = fmt.Sprintf("%s.%s.%s: ", e.Namespace, e.Table, e.Column)
		} else {
			target = fmt.Sprintf("%s.%s: ", e.Namespace, e.Table)
		}
	} else if e.Table != "" {
		if e.Column != "" {
			target = fmt.Sprintf("%s.%s: ", e.Table, e.Column)
		} else {
			target = e.Table + ": "
		}
	}

	if e.Cause != nil {
		return fmt.Sprintf("astroladb: %s%s%s: %v", location, target, e.Message, e.Cause)
	}
	return fmt.Sprintf("astroladb: %s%s%s", location, target, e.Message)
}

// Unwrap returns the underlying cause error.
func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// Is reports whether this error matches the target error.
func (e *SchemaError) Is(target error) bool {
	return target == ErrSchemaInvalid
}

// ConnectionError provides detailed information about a database connection error.
type ConnectionError struct {
	// URL is the database URL (with password redacted).
	URL string

	// Dialect is the database dialect (postgres, sqlite).
	Dialect string

	// Cause is the underlying error from the database driver.
	Cause error
}

// Error returns a formatted error message.
func (e *ConnectionError) Error() string {
	return fmt.Sprintf("astroladb: failed to connect to %s database: %v", e.Dialect, e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// Is reports whether this error matches the target error.
func (e *ConnectionError) Is(target error) bool {
	return target == ErrConnectionFailed
}

// ErrChainIntegrity is returned when the migration chain integrity check fails.
var ErrChainIntegrity = errors.New("astroladb: chain integrity broken")

// ChainError provides detailed information about chain integrity failures.
type ChainError struct {
	// Result contains the detailed verification result.
	Result *chain.VerificationResult
}

// Error returns a formatted error message.
func (e *ChainError) Error() string {
	if e.Result == nil {
		return "astroladb: chain integrity broken"
	}

	return "astroladb: chain integrity broken\n" + chain.FormatVerificationResult(e.Result)
}

// Is reports whether this error matches the target error.
func (e *ChainError) Is(target error) bool {
	return target == ErrChainIntegrity
}
