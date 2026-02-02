package astroladb

import (
	"io"
	"time"
)

// Config holds all configuration options for the Client.
type Config struct {
	// DatabaseURL is the connection string for the database.
	// Format depends on the dialect:
	//   - PostgreSQL: postgres://user:pass@host:port/dbname
	//   - SQLite: ./path/to/db.db or /absolute/path/to/db.db
	DatabaseURL string

	// SchemasDir is the path to the directory containing schema files.
	// Default: ./schemas
	SchemasDir string

	// MigrationsDir is the path to the directory containing migration files.
	// Default: ./migrations
	MigrationsDir string

	// Dialect specifies the database dialect to use.
	// If empty, it will be auto-detected from the DatabaseURL.
	// Valid values: "postgres", "sqlite"
	Dialect string

	// Timeout is the maximum duration for database operations.
	// Default: 30s
	Timeout time.Duration

	// Logger is used for logging operations.
	// If nil, no logging is performed.
	Logger Logger

	// SchemaOnly when true, skips database connection.
	// Use for operations that only read schema files (export, check).
	SchemaOnly bool
}

// Logger is the interface for logging operations.
// It's compatible with the standard library's log.Logger.
type Logger interface {
	// Printf writes a formatted message to the log.
	Printf(format string, v ...any)
}

// Option is a functional option for configuring the Client.
type Option func(*Config)

// WithDatabaseURL sets the database connection URL.
//
// Examples:
//   - PostgreSQL: postgres://user:pass@localhost:5432/mydb
//   - SQLite: ./mydb.db or /absolute/path/to/mydb.db
func WithDatabaseURL(url string) Option {
	return func(c *Config) {
		c.DatabaseURL = url
	}
}

// WithSchemasDir sets the path to the schemas directory.
// Default: ./schemas
func WithSchemasDir(dir string) Option {
	return func(c *Config) {
		c.SchemasDir = dir
	}
}

// WithMigrationsDir sets the path to the migrations directory.
// Default: ./migrations
func WithMigrationsDir(dir string) Option {
	return func(c *Config) {
		c.MigrationsDir = dir
	}
}

// WithDialect explicitly sets the database dialect.
// If not set, it will be auto-detected from the database URL.
// Valid values: "postgres", "sqlite"
func WithDialect(dialect string) Option {
	return func(c *Config) {
		c.Dialect = dialect
	}
}

// WithLogger sets the logger for the client.
// If not set, no logging is performed.
func WithLogger(l Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}

// WithTimeout sets the timeout for database operations.
// Default: 30s
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

// WithSchemaOnly enables schema-only mode, which skips database connection.
// Use this for operations that only read schema files, such as:
//   - SchemaExport (export to OpenAPI, TypeScript, etc.)
//   - SchemaCheck (validate schema files)
//
// Operations that require a database connection will return an error.
func WithSchemaOnly() Option {
	return func(c *Config) {
		c.SchemaOnly = true
	}
}

// MigrationConfig holds options for migration operations.
type MigrationConfig struct {
	// DryRun if true, shows SQL without executing it.
	DryRun bool

	// Target is the target revision to migrate to.
	// Empty means latest (for up) or none (for down).
	Target string

	// Steps is the number of migrations to apply/rollback.
	// 0 means all pending (for up) or 1 (for down).
	Steps int

	// Output is where to write dry-run SQL output.
	// Defaults to io.Discard if nil.
	Output io.Writer

	// Force skips chain integrity verification.
	// Use with caution - this can lead to inconsistent state.
	Force bool

	// SkipLock disables distributed locking for migrations.
	// Use in CI environments where you're certain only one runner exists.
	// Default: false (locking is enabled by default)
	SkipLock bool

	// LockTimeout is the maximum time to wait for the migration lock.
	// Default: 30 seconds
	LockTimeout time.Duration
}

// MigrationOption is a functional option for migration operations.
type MigrationOption func(*MigrationConfig)

// DryRun enables dry-run mode, which shows the SQL that would be
// executed without actually executing it.
func DryRun() MigrationOption {
	return func(c *MigrationConfig) {
		c.DryRun = true
	}
}

// DryRunTo enables dry-run mode and writes output to the given writer.
func DryRunTo(w io.Writer) MigrationOption {
	return func(c *MigrationConfig) {
		c.DryRun = true
		c.Output = w
	}
}

// Target sets the target revision to migrate to.
// For MigrationRun: stops at this revision.
// For MigrationRollback: rolls back to this revision (exclusive).
func Target(revision string) MigrationOption {
	return func(c *MigrationConfig) {
		c.Target = revision
	}
}

// Steps sets the number of migrations to apply or rollback.
// For MigrationRun: applies up to n migrations.
// For MigrationRollback: rolls back n migrations.
func Steps(n int) MigrationOption {
	return func(c *MigrationConfig) {
		c.Steps = n
	}
}

// Force skips chain integrity verification.
// Use with caution - this can lead to inconsistent migration state.
func Force() MigrationOption {
	return func(c *MigrationConfig) {
		c.Force = true
	}
}

// SkipLock disables distributed locking for migrations.
// Use in CI environments where you're certain only one migration runner exists.
//
// WARNING: Without locking, concurrent migration attempts may cause conflicts.
// Only use this when you control the execution environment (e.g., CI pipelines).
func SkipLock() MigrationOption {
	return func(c *MigrationConfig) {
		c.SkipLock = true
	}
}

// LockTimeout sets the maximum time to wait for the migration lock.
// Default is 30 seconds if not specified.
//
// If the lock is not acquired within this timeout, the migration fails with
// an error indicating which process currently holds the lock.
func LockTimeout(d time.Duration) MigrationOption {
	return func(c *MigrationConfig) {
		c.LockTimeout = d
	}
}

// applyMigrationOptions applies all migration options to a config.
func applyMigrationOptions(opts []MigrationOption) *MigrationConfig {
	cfg := &MigrationConfig{
		Steps: 0, // 0 = all for up, 1 for down
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// ExportConfig holds options for schema export operations.
type ExportConfig struct {
	// Namespace filters output to a specific namespace.
	// Empty means all namespaces.
	Namespace string

	// UseChrono enables chrono types for Rust date/time fields.
	// When false (default), date/time fields use String.
	UseChrono bool

	// UseMik enables mik_sdk style for Rust exports.
	// Uses #[derive(Type)] and imports mik_sdk::prelude::*.
	UseMik bool

	// Relations enables generation of WithRelations type variants
	// that include relationship fields (foreign key references, many-to-many).
	Relations bool
}

// ExportOption is a functional option for export operations.
type ExportOption func(*ExportConfig)

// WithNamespace filters output to a specific namespace.
func WithNamespace(ns string) ExportOption {
	return func(c *ExportConfig) {
		c.Namespace = ns
	}
}

// WithChrono enables chrono types for Rust date/time fields.
func WithChrono() ExportOption {
	return func(c *ExportConfig) {
		c.UseChrono = true
	}
}

// WithMik enables mik_sdk style for Rust exports.
// Uses #[derive(Type)] and imports mik_sdk::prelude::*.
func WithMik() ExportOption {
	return func(c *ExportConfig) {
		c.UseMik = true
	}
}

// WithRelations enables generation of WithRelations type variants.
func WithRelations() ExportOption {
	return func(c *ExportConfig) {
		c.Relations = true
	}
}

// applyExportOptions applies all export options to a config.
func applyExportOptions(opts []ExportOption) *ExportConfig {
	cfg := &ExportConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
