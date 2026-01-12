package astroladb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/drift"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/registry"
	"github.com/hlop3z/astroladb/internal/runtime"
)

// Client is the main entry point for the Alab database migration tool.
// It provides methods for schema validation, migration management, and schema export.
//
// Create a new client with New() and close it with Close() when done.
//
// Example:
//
//	client, err := astroladb.New(
//	    astroladb.WithDatabaseURL("postgres://localhost/mydb"),
//	    astroladb.WithSchemasDir("./schemas"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	if err := client.MigrationRun(); err != nil {
//	    log.Fatal(err)
//	}
type Client struct {
	db       *sql.DB
	dialect  dialect.Dialect
	config   *Config
	sandbox  *runtime.Sandbox
	registry *registry.ModelRegistry
	runner   *engine.Runner
	eval     *engine.Evaluator
}

// New creates a new Client with the given options.
//
// At minimum, WithDatabaseURL must be provided.
// The dialect will be auto-detected from the URL if not explicitly set.
//
// Example:
//
//	client, err := astroladb.New(
//	    astroladb.WithDatabaseURL("postgres://user:pass@localhost:5432/mydb"),
//	    astroladb.WithSchemasDir("./schemas"),
//	    astroladb.WithMigrationsDir("./migrations"),
//	)
func New(opts ...Option) (*Client, error) {
	// Apply default configuration
	cfg := &Config{
		SchemasDir:    "./schemas",
		MigrationsDir: "./migrations",
		Timeout:       30 * time.Second,
	}

	// Apply user options
	for _, opt := range opts {
		opt(cfg)
	}

	// Initialize internal components (always needed)
	reg := registry.NewModelRegistry()
	sandbox := runtime.NewSandbox(reg)
	evaluator := engine.NewEvaluator(reg)

	// Schema-only mode: skip database connection
	// Used for operations like export and check that only read schema files
	if cfg.SchemaOnly {
		return &Client{
			db:       nil,
			dialect:  nil,
			config:   cfg,
			sandbox:  sandbox,
			registry: reg,
			runner:   nil,
			eval:     evaluator,
		}, nil
	}

	// Validate required configuration for database mode
	if cfg.DatabaseURL == "" {
		return nil, ErrMissingDatabaseURL
	}

	// Auto-detect dialect from URL if not specified
	if cfg.Dialect == "" {
		cfg.Dialect = detectDialect(cfg.DatabaseURL)
	}

	// Get the dialect implementation
	d := dialect.Get(cfg.Dialect)
	if d == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDialect, cfg.Dialect)
	}

	// Connect to the database
	db, err := openDatabase(cfg.DatabaseURL, cfg.Dialect)
	if err != nil {
		return nil, &ConnectionError{
			URL:     redactURL(cfg.DatabaseURL),
			Dialect: cfg.Dialect,
			Cause:   err,
		}
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, &ConnectionError{
			URL:     redactURL(cfg.DatabaseURL),
			Dialect: cfg.Dialect,
			Cause:   err,
		}
	}

	// Set timezone to UTC for PostgreSQL
	if cfg.Dialect == "postgres" {
		if _, err := db.ExecContext(ctx, "SET timezone = 'UTC'"); err != nil {
			db.Close()
			return nil, &ConnectionError{
				URL:     redactURL(cfg.DatabaseURL),
				Dialect: cfg.Dialect,
				Cause:   fmt.Errorf("failed to set UTC timezone: %w", err),
			}
		}
	}

	// Initialize runner (needs db and dialect)
	runner := engine.NewRunner(db, d)

	return &Client{
		db:       db,
		dialect:  d,
		config:   cfg,
		sandbox:  sandbox,
		registry: reg,
		runner:   runner,
		eval:     evaluator,
	}, nil
}

// Close closes the database connection and releases resources.
// It should be called when the client is no longer needed.
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// DB returns the underlying database connection.
// Use with caution - prefer the high-level methods when possible.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Dialect returns the database dialect name.
func (c *Client) Dialect() string {
	return c.dialect.Name()
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return *c.config
}

// log logs a message if a logger is configured.
func (c *Client) log(format string, v ...any) {
	if c.config.Logger != nil {
		c.config.Logger.Printf(format, v...)
	}
}

// context returns a context with the configured timeout.
func (c *Client) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.config.Timeout)
}

// detectDialect auto-detects the database dialect from the connection URL.
//
// Detection rules:
//   - postgres:// or postgresql:// -> postgres
//   - sqlite:// or file: or path ending with .db/.sqlite/.sqlite3 -> sqlite
func detectDialect(url string) string {
	url = strings.ToLower(url)

	// Check for explicit scheme prefixes
	switch {
	case strings.HasPrefix(url, "postgres://"),
		strings.HasPrefix(url, "postgresql://"):
		return "postgres"

	case strings.HasPrefix(url, "sqlite://"),
		strings.HasPrefix(url, "sqlite3://"),
		strings.HasPrefix(url, "file:"):
		return "sqlite"

	case strings.HasSuffix(url, ".db"),
		strings.HasSuffix(url, ".sqlite"),
		strings.HasSuffix(url, ".sqlite3"):
		return "sqlite"
	}

	// Default to postgres if no match
	return "postgres"
}

// openDatabase opens a database connection based on the dialect.
func openDatabase(url, dialectName string) (*sql.DB, error) {
	var driverName string
	var dsn string

	switch dialectName {
	case "postgres":
		driverName = "postgres"
		dsn = url

	case "sqlite":
		driverName = "sqlite"
		// Convert sqlite:// URL to file path, or use path directly
		dsn = convertSQLiteURL(url)

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialectName)
	}

	return sql.Open(driverName, dsn)
}

// convertSQLiteURL converts a sqlite:// URL to a file path, or returns the path as-is.
// Accepts both URL formats (sqlite://path) and direct file paths (./path or /path).
func convertSQLiteURL(url string) string {
	url = strings.TrimPrefix(url, "sqlite://")
	url = strings.TrimPrefix(url, "sqlite3://")
	url = strings.TrimPrefix(url, "file:")

	// Handle query parameters (e.g., ?mode=memory)
	// SQLite3 driver accepts these directly
	return url
}

// redactURL removes sensitive information from a database URL for logging.
func redactURL(url string) string {
	// Find password in URL and replace it
	// Pattern: ://user:password@ or ://password@
	start := strings.Index(url, "://")
	if start == -1 {
		return url
	}
	start += 3

	end := strings.Index(url[start:], "@")
	if end == -1 {
		return url
	}
	end += start

	credentials := url[start:end]
	if colonIdx := strings.Index(credentials, ":"); colonIdx != -1 {
		// Has password - redact it
		user := credentials[:colonIdx]
		return url[:start] + user + ":***@" + url[end+1:]
	}

	return url
}

// loadTables loads and evaluates all schema files.
// It also includes auto-generated join tables from many_to_many relationships.
func (c *Client) loadTables() ([]*ast.TableDef, error) {
	tables, err := c.eval.EvalDirStrict(c.config.SchemasDir)
	if err != nil {
		return nil, &SchemaError{
			Message: "failed to load schemas",
			File:    c.config.SchemasDir,
			Cause:   err,
		}
	}

	// Append auto-generated join tables from many_to_many relationships
	joinTables := c.eval.GetJoinTables()
	tables = append(tables, joinTables...)

	return tables, nil
}

// loadMigrations loads all migration files from the migrations directory.
func (c *Client) loadMigrations() ([]engine.Migration, error) {
	// This would load migrations from files
	// For now, we return empty since the migration loading logic
	// is typically handled by the evaluator
	return nil, nil
}

// getSchema loads the schema from schema files.
func (c *Client) getSchema() (*engine.Schema, error) {
	tables, err := c.loadTables()
	if err != nil {
		return nil, err
	}
	return engine.SchemaFromTables(tables)
}

// SaveMetadata saves schema metadata to .alab/metadata.json.
// This includes many_to_many relationships, polymorphic mappings, and join table definitions.
// Call this after evaluating schemas to persist metadata for external tools.
func (c *Client) SaveMetadata() error {
	// Get the project directory (parent of schemas dir)
	projectDir := c.config.SchemasDir
	if projectDir == "./schemas" || projectDir == "schemas" {
		projectDir = "."
	} else if len(projectDir) > 8 && projectDir[len(projectDir)-8:] == "/schemas" {
		projectDir = projectDir[:len(projectDir)-8]
	}
	return c.eval.SaveMetadata(projectDir)
}

// SaveMetadataToFile saves schema metadata to a custom file path.
// This includes many_to_many relationships, polymorphic mappings, and join table definitions.
// Useful for exporting metadata to a specific location for external tools.
func (c *Client) SaveMetadataToFile(filePath string) error {
	return c.eval.SaveMetadataToFile(filePath)
}

// DriftResult represents the result of a drift detection operation.
// It includes whether drift was detected, merkle tree hashes for fast comparison,
// and detailed information about any differences found.
type DriftResult struct {
	// HasDrift is true if any differences were found between expected and actual schema.
	HasDrift bool

	// ExpectedHash is the merkle root hash of the expected schema (from migrations).
	ExpectedHash string

	// ActualHash is the merkle root hash of the actual database schema.
	ActualHash string

	// Summary provides counts of missing, extra, and modified tables.
	Summary *DriftSummary

	// TableDiffs contains detailed differences for each modified table.
	TableDiffs map[string]*TableDiff

	// MissingTables lists tables that should exist (per migrations) but don't.
	MissingTables []string

	// ExtraTables lists tables that exist in DB but aren't defined in migrations.
	ExtraTables []string
}

// DriftSummary provides a summary of drift detection results.
type DriftSummary struct {
	Tables         int // Total expected tables
	MissingTables  int // Tables missing from database
	ExtraTables    int // Extra tables in database
	ModifiedTables int // Tables with differences
}

// TableDiff represents differences within a single table.
type TableDiff struct {
	Name            string
	MissingColumns  []string
	ExtraColumns    []string
	ModifiedColumns []string
	MissingIndexes  []string
	ExtraIndexes    []string
	ModifiedIndexes []string
	MissingFKs      []string
	ExtraFKs        []string
	ModifiedFKs     []string
}

// CheckDrift detects schema drift by comparing the expected schema (from migrations)
// against the actual database schema.
//
// Uses merkle trees for fast comparison - if root hashes match, schemas are identical.
// When drift is detected, provides detailed breakdown of differences.
//
// Example:
//
//	result, err := client.CheckDrift()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if result.HasDrift {
//	    fmt.Println("Schema drift detected!")
//	    for _, table := range result.MissingTables {
//	        fmt.Printf("  Missing: %s\n", table)
//	    }
//	}
func (c *Client) CheckDrift() (*DriftResult, error) {
	ctx, cancel := c.context()
	defer cancel()

	return c.CheckDriftContext(ctx)
}

// CheckDriftContext detects schema drift with a custom context.
func (c *Client) CheckDriftContext(ctx context.Context) (*DriftResult, error) {
	// Load expected schema from migrations
	expected, err := c.getExpectedSchema()
	if err != nil {
		return nil, err
	}

	// Create drift detector and run detection
	detector := drift.NewDetector(c.db, c.dialect)
	result, err := detector.Detect(ctx, expected)
	if err != nil {
		return nil, err
	}

	// Convert internal result to public API result
	return convertDriftResult(result), nil
}

// QuickDriftCheck performs a fast drift check by comparing only root hashes.
// Returns true if schemas match (no drift), false if drift exists.
//
// Use this when you just need to know if drift exists, not the details.
// For full details, use CheckDrift().
func (c *Client) QuickDriftCheck() (bool, error) {
	ctx, cancel := c.context()
	defer cancel()

	expected, err := c.getExpectedSchema()
	if err != nil {
		return false, err
	}

	detector := drift.NewDetector(c.db, c.dialect)
	return detector.QuickCheck(ctx, expected)
}

// getExpectedSchema returns the expected schema state in canonical/normalized form.
// This uses the industry-standard approach combining Liquibase and Atlas patterns:
//
// 1. Liquibase/Flyway Pattern: Replay migration history to get expected schema
// 2. Atlas Pattern: Apply to dev database for normalization to canonical form
//
// Why both are needed:
// - Migration replay gives us "what should be in DB" (natural form)
// - Dev database normalization converts to canonical form for accurate comparison
// - Without normalization, representations don't match (e.g., "NOW()" vs "CURRENT_TIMESTAMP")
func (c *Client) getExpectedSchema() (*engine.Schema, error) {
	// Step 1: Replay migrations to get expected schema (Liquibase pattern)
	schema, err := c.getSchemaFromMigrations()
	if err != nil {
		return nil, err
	}

	// Step 2: Normalize via dev database (Atlas pattern)
	return c.normalizeSchema(schema)
}

// convertDriftResult converts internal drift.Result to public DriftResult.
func convertDriftResult(r *drift.Result) *DriftResult {
	if r == nil {
		return &DriftResult{}
	}

	result := &DriftResult{
		HasDrift:      r.HasDrift,
		ExpectedHash:  r.ExpectedHash,
		ActualHash:    r.ActualHash,
		MissingTables: r.Comparison.MissingTables,
		ExtraTables:   r.Comparison.ExtraTables,
		TableDiffs:    make(map[string]*TableDiff),
	}

	// Convert summary
	summary := drift.Summarize(r)
	result.Summary = &DriftSummary{
		Tables:         summary.Tables,
		MissingTables:  summary.MissingTables,
		ExtraTables:    summary.ExtraTables,
		ModifiedTables: summary.ModifiedTables,
	}

	// Convert table diffs
	for name, diff := range r.Comparison.TableDiffs {
		result.TableDiffs[name] = &TableDiff{
			Name:            diff.Name,
			MissingColumns:  diff.MissingColumns,
			ExtraColumns:    diff.ExtraColumns,
			ModifiedColumns: diff.ModifiedColumns,
			MissingIndexes:  diff.MissingIndexes,
			ExtraIndexes:    diff.ExtraIndexes,
			ModifiedIndexes: diff.ModifiedIndexes,
			MissingFKs:      diff.MissingFKs,
			ExtraFKs:        diff.ExtraFKs,
			ModifiedFKs:     diff.ModifiedFKs,
		}
	}

	return result
}

// FormatDriftResult returns a user-friendly string representation of drift results.
func FormatDriftResult(result *DriftResult) string {
	if result == nil {
		return "No drift detection result available."
	}

	// Convert back to internal format for formatting
	internalResult := &drift.Result{
		HasDrift:     result.HasDrift,
		ExpectedHash: result.ExpectedHash,
		ActualHash:   result.ActualHash,
		Comparison: &drift.HashComparison{
			Match:         !result.HasDrift,
			ExpectedRoot:  result.ExpectedHash,
			ActualRoot:    result.ActualHash,
			MissingTables: result.MissingTables,
			ExtraTables:   result.ExtraTables,
			TableDiffs:    make(map[string]*drift.TableDiff),
		},
		ExpectedSchema: engine.NewSchema(),
		ActualSchema:   engine.NewSchema(),
	}

	// Convert table diffs back
	for name, td := range result.TableDiffs {
		internalResult.Comparison.TableDiffs[name] = &drift.TableDiff{
			Name:            td.Name,
			MissingColumns:  td.MissingColumns,
			ExtraColumns:    td.ExtraColumns,
			ModifiedColumns: td.ModifiedColumns,
			MissingIndexes:  td.MissingIndexes,
			ExtraIndexes:    td.ExtraIndexes,
			ModifiedIndexes: td.ModifiedIndexes,
			MissingFKs:      td.MissingFKs,
			ExtraFKs:        td.ExtraFKs,
			ModifiedFKs:     td.ModifiedFKs,
		}
	}

	// Set table count for summary
	if result.Summary != nil {
		for i := 0; i < result.Summary.Tables; i++ {
			internalResult.ExpectedSchema.Tables[fmt.Sprintf("table_%d", i)] = nil
		}
	}

	return drift.FormatResult(internalResult)
}

// computeMerkleHash computes the merkle hash for a schema.
func (c *Client) computeMerkleHash(schema *engine.Schema) (*drift.SchemaHash, error) {
	return drift.ComputeSchemaHash(schema)
}
