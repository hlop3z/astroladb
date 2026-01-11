package astroladb

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/devdb"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/introspect"
)

// sortColumnsForDatabase sorts columns for database creation:
// 1. id/primary key first
// 2. Other columns alphabetically
// 3. timestamps (created_at, updated_at, deleted_at) last
func (c *Client) sortColumnsForDatabase(columns []*ast.ColumnDef) []*ast.ColumnDef {
	if len(columns) == 0 {
		return columns
	}

	// Make a copy to avoid modifying the original
	sorted := make([]*ast.ColumnDef, len(columns))
	copy(sorted, columns)

	// Sort using custom ordering
	sort.SliceStable(sorted, func(i, j int) bool {
		col1, col2 := sorted[i], sorted[j]

		// Priority 1: id/primary key columns first
		if col1.PrimaryKey && !col2.PrimaryKey {
			return true
		}
		if !col1.PrimaryKey && col2.PrimaryKey {
			return false
		}

		// Priority 2: timestamp columns last
		isTimestamp1 := col1.Name == "created_at" || col1.Name == "updated_at" || col1.Name == "deleted_at"
		isTimestamp2 := col2.Name == "created_at" || col2.Name == "updated_at" || col2.Name == "deleted_at"

		if isTimestamp1 && !isTimestamp2 {
			return false
		}
		if !isTimestamp1 && isTimestamp2 {
			return true
		}

		// Priority 3: created_at before updated_at before deleted_at
		if isTimestamp1 && isTimestamp2 {
			order := map[string]int{
				"created_at": 1,
				"updated_at": 2,
				"deleted_at": 3,
			}
			return order[col1.Name] < order[col2.Name]
		}

		// Priority 4: alphabetically by name
		return col1.Name < col2.Name
	})

	return sorted
}

// getCompiledSchema loads tables and compiles schema - used by multiple methods.
// This is a DRY helper that reduces code duplication across SchemaCheck, SchemaDump, and SchemaExport.
func (c *Client) getCompiledSchema() (*engine.Schema, []*ast.TableDef, error) {
	tables, err := c.loadTables()
	if err != nil {
		return nil, nil, err
	}
	schema, err := engine.SchemaFromTables(tables)
	if err != nil {
		return nil, nil, err
	}
	return schema, tables, nil
}

// SchemaCheck validates all schema files in the schemas directory.
// It checks for:
//   - Valid JavaScript syntax
//   - Valid table and column definitions
//   - Valid type usage (no forbidden types)
//   - Valid references between tables
//   - No circular dependencies
//
// Returns nil if all schemas are valid, or an error describing the first problem found.
func (c *Client) SchemaCheck() error {
	c.log("Checking schemas in %s", c.config.SchemasDir)

	// Load tables and compile schema
	schema, tables, err := c.getCompiledSchema()
	if err != nil {
		return err
	}

	if len(tables) == 0 {
		return &SchemaError{
			Message: "no schema files found",
			File:    c.config.SchemasDir,
		}
	}

	c.log("Found %d tables", len(tables))

	// Validate the schema (checks references, cycles, etc.)
	if err := schema.Validate(); err != nil {
		return &SchemaError{
			Message: "schema validation failed",
			Cause:   err,
		}
	}

	// Double-evaluate check for determinism
	if err := c.checkDeterminism(tables); err != nil {
		return err
	}

	c.log("Schema check passed: %d tables validated", len(tables))
	return nil
}

// checkDeterminism verifies that schema evaluation is deterministic
// by evaluating each schema file twice and comparing results.
func (c *Client) checkDeterminism(tables []*ast.TableDef) error {
	// Re-evaluate all schemas
	tables2, err := c.eval.EvalDirStrict(c.config.SchemasDir)
	if err != nil {
		return &SchemaError{
			Message: "determinism check failed: second evaluation produced error",
			Cause:   err,
		}
	}

	// Include auto-generated join tables from many_to_many relationships
	joinTables2 := c.eval.GetJoinTables()
	tables2 = append(tables2, joinTables2...)

	// Compare table counts
	if len(tables) != len(tables2) {
		return &SchemaError{
			Message: fmt.Sprintf("determinism check failed: first pass produced %d tables, second produced %d",
				len(tables), len(tables2)),
		}
	}

	// Tables are sorted by qualified name, so we can compare directly
	for i, t1 := range tables {
		t2 := tables2[i]
		if t1.QualifiedName() != t2.QualifiedName() {
			return &SchemaError{
				Message: fmt.Sprintf("determinism check failed: table order differs at position %d", i),
			}
		}
		if len(t1.Columns) != len(t2.Columns) {
			return &SchemaError{
				Namespace: t1.Namespace,
				Table:     t1.Name,
				Message:   "determinism check failed: column count differs between evaluations",
			}
		}
	}

	return nil
}

// SchemaDump returns the SQL representation of the current schema.
// This is the SQL that would create all tables, indexes, and constraints.
func (c *Client) SchemaDump() (string, error) {
	c.log("Dumping schema from %s", c.config.SchemasDir)

	schema, tables, err := c.getCompiledSchema()
	if err != nil {
		return "", err
	}

	if len(tables) == 0 {
		return "", nil
	}

	// Get tables in dependency order
	orderedTables, err := schema.GetCreationOrder()
	if err != nil {
		return "", &SchemaError{
			Message: "failed to determine creation order",
			Cause:   err,
		}
	}

	var result strings.Builder

	// Generate SQL for each table
	for i, table := range orderedTables {
		if i > 0 {
			result.WriteString("\n\n")
		}

		// Convert table to CreateTable operation with sorted columns
		createOp := &ast.CreateTable{
			TableOp: ast.TableOp{
				Namespace: table.Namespace,
				Name:      table.Name,
			},
			Columns:     c.sortColumnsForDatabase(table.Columns),
			Indexes:     table.Indexes,
			ForeignKeys: table.ForeignKeys,
			IfNotExists: true,
		}

		sql, err := c.dialect.CreateTableSQL(createOp)
		if err != nil {
			return "", &SchemaError{
				Namespace: table.Namespace,
				Table:     table.Name,
				Message:   "failed to generate SQL",
				Cause:     err,
			}
		}

		result.WriteString(sql)
		result.WriteString(";")
	}

	return result.String(), nil
}

// SchemaDiff compares the current schema files against the database state
// and returns the operations needed to bring the database in sync.
//
// Uses dev database normalization for accurate comparison:
// 1. Load schema files (natural form)
// 2. Normalize through dev database (canonical form)
// 3. Introspect production database (already canonical)
// 4. Compare canonical forms (100% accurate)
//
// This is useful for understanding what migrations would be generated.
func (c *Client) SchemaDiff() ([]ast.Operation, error) {
	c.log("Computing schema diff")

	// Step 1: Load the desired schema from files (natural form)
	desiredSchema, err := c.getSchema()
	if err != nil {
		return nil, err
	}

	// Step 2: Normalize desired schema through dev database (Atlas pattern)
	normalizedDesired, err := c.normalizeSchema(desiredSchema)
	if err != nil {
		return nil, &SchemaError{
			Message: "failed to normalize desired schema",
			Cause:   err,
		}
	}

	// Step 3: Get the current database schema via introspection (already canonical)
	ctx := context.Background()
	intro := introspect.New(c.db, c.dialect)
	if intro == nil {
		return nil, &SchemaError{
			Message: "unsupported dialect for introspection",
		}
	}

	actualSchema, err := intro.IntrospectSchema(ctx)
	if err != nil {
		return nil, &SchemaError{
			Message: "failed to introspect database",
			Cause:   err,
		}
	}

	// Step 4: Compute the diff between canonical forms
	ops, err := engine.Diff(actualSchema, normalizedDesired)
	if err != nil {
		return nil, &SchemaError{
			Message: "failed to compute diff",
			Cause:   err,
		}
	}

	c.log("Diff found %d operations", len(ops))
	return ops, nil
}

// SchemaDiffFromMigrations compares the desired schema files against the schema state
// from existing migrations (not the database). This ensures generated migrations are
// incremental and only contain the delta from the last migration.
//
// This is used for migration generation to prevent recreating existing tables.
func (c *Client) SchemaDiffFromMigrations() ([]ast.Operation, error) {
	c.log("Computing schema diff from migrations")

	// Load the desired schema from files
	newSchema, err := c.getSchema()
	if err != nil {
		return nil, err
	}

	// Compute schema state from existing migration files
	oldSchema, err := c.getSchemaFromMigrations()
	if err != nil {
		return nil, err
	}

	// Compute the diff
	ops, err := engine.Diff(oldSchema, newSchema)
	if err != nil {
		return nil, &SchemaError{
			Message: "failed to compute diff",
			Cause:   err,
		}
	}

	c.log("Diff found %d operations", len(ops))
	return ops, nil
}

// getSchemaFromMigrations replays all migration files to compute the current schema state.
// This is used for incremental migration generation.
func (c *Client) getSchemaFromMigrations() (*engine.Schema, error) {
	// Load all migration files
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, err
	}

	// If no migrations exist, start with empty schema
	if len(migrations) == 0 {
		return &engine.Schema{
			Tables: make(map[string]*ast.TableDef),
		}, nil
	}

	// Replay all migrations to build the schema
	schema, err := engine.ReplayMigrations(migrations)
	if err != nil {
		return nil, &SchemaError{
			Message: "failed to replay migrations",
			Cause:   err,
		}
	}

	return schema, nil
}

// normalizeSchema normalizes a schema by applying it to a dev database.
// This follows the Atlas pattern: apply schema to database, let the database
// normalize it to canonical form, then introspect to get the normalized version.
//
// This is critical for accurate drift detection because:
// - Schema files define things in "natural form" (how humans write it)
// - Databases store things in "canonical form" (normalized representation)
// - Direct comparison of natural vs canonical fails (false positives)
// - Normalizing both sides through dev database ensures accurate comparison
func (c *Client) normalizeSchema(schema *engine.Schema) (*engine.Schema, error) {
	// Create ephemeral dev database
	dev, err := devdb.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create dev database: %w", err)
	}
	defer dev.Close()

	// Normalize schema through dev database
	ctx := context.Background()
	normalized, err := dev.NormalizeSchema(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize schema: %w", err)
	}

	return normalized, nil
}

// SchemaExport exports the schema in the specified format.
// Supported formats:
//   - "openapi" - OpenAPI 3.0 specification (JSON)
//   - "typescript" - TypeScript type definitions
//   - "go" - Go struct definitions
//   - "python" - Python dataclass definitions
//   - "rust" - Rust struct definitions with serde
//
// Returns the exported content as bytes.
func (c *Client) SchemaExport(format string, opts ...ExportOption) ([]byte, error) {
	c.log("Exporting schema as %s", format)

	tables, err := c.loadTables()
	if err != nil {
		return nil, err
	}

	cfg := applyExportOptions(opts)

	// Attach metadata for many_to_many and polymorphic relationship info
	cfg.Metadata = c.eval.Metadata()

	// Filter by namespace if specified
	if cfg.Namespace != "" {
		var filtered []*ast.TableDef
		for _, t := range tables {
			if t.Namespace == cfg.Namespace {
				filtered = append(filtered, t)
			}
		}
		tables = filtered
	}

	switch strings.ToLower(format) {
	case "openapi":
		return exportOpenAPI(tables, cfg)
	case "graphql", "gql":
		return exportGraphQL(tables, cfg)
	case "graphql-examples":
		return exportGraphQLExamples(tables, cfg)
	case "typescript", "ts":
		return exportTypeScript(tables, cfg)
	case "go", "golang":
		return exportGo(tables, cfg)
	case "python", "py":
		return exportPython(tables, cfg)
	case "rust", "rs":
		return exportRust(tables, cfg)
	default:
		return nil, fmt.Errorf("%w: %s (valid formats: openapi, graphql, typescript, go, python, rust)", ErrExportFormatUnknown, format)
	}
}

// TableInfo provides information about a single table in the schema.
type TableInfo struct {
	// Namespace is the logical grouping (e.g., "auth").
	Namespace string

	// Name is the table name.
	Name string

	// SQLName is the full SQL table name (namespace_tablename).
	SQLName string

	// ColumnCount is the number of columns.
	ColumnCount int

	// HasPrimaryKey indicates if the table has a primary key.
	HasPrimaryKey bool

	// IndexCount is the number of indexes.
	IndexCount int

	// ForeignKeyCount is the number of foreign key constraints.
	ForeignKeyCount int

	// Docs is the table description (if provided).
	Docs string
}

// SchemaTables returns information about all tables in the schema.
func (c *Client) SchemaTables() ([]TableInfo, error) {
	tables, err := c.loadTables()
	if err != nil {
		return nil, err
	}

	result := make([]TableInfo, len(tables))
	for i, t := range tables {
		result[i] = TableInfo{
			Namespace:       t.Namespace,
			Name:            t.Name,
			SQLName:         t.SQLName(),
			ColumnCount:     len(t.Columns),
			HasPrimaryKey:   t.PrimaryKey() != nil,
			IndexCount:      len(t.Indexes),
			ForeignKeyCount: len(t.ForeignKeys),
			Docs:            t.Docs,
		}
	}

	return result, nil
}

// SchemaNamespaces returns a list of all namespaces in the schema.
func (c *Client) SchemaNamespaces() ([]string, error) {
	schema, err := c.getSchema()
	if err != nil {
		return nil, err
	}
	return schema.Namespaces(), nil
}

// GetNamespaces returns a list of all namespaces in the schema.
// This is an alias for SchemaNamespaces.
func (c *Client) GetNamespaces() ([]string, error) {
	return c.SchemaNamespaces()
}

// SchemaAtRevision returns the schema state at a specific migration revision.
// It loads migrations up to and including the target revision, replays the
// operations, and returns the resulting schema.
//
// This is useful for understanding what the schema looked like at a specific
// point in history.
//
// Example:
//
//	schema, err := client.SchemaAtRevision("003")
//	// Returns the schema state after migrations 001, 002, 003 were applied
func (c *Client) SchemaAtRevision(revision string) (*engine.Schema, error) {
	c.log("Computing schema at revision %s", revision)

	// Load all migrations
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, &MigrationError{
			Revision:  revision,
			Operation: "load migrations",
			Cause:     err,
		}
	}

	if len(migrations) == 0 {
		return nil, &MigrationError{
			Revision:  revision,
			Operation: "schema at revision",
			Cause:     fmt.Errorf("no migrations found"),
		}
	}

	// Find the target revision
	found := false
	for _, m := range migrations {
		if m.Revision == revision {
			found = true
			break
		}
	}

	if !found {
		return nil, &MigrationError{
			Revision:  revision,
			Operation: "schema at revision",
			Cause:     fmt.Errorf("revision not found: %s", revision),
		}
	}

	// Replay migrations up to the target
	schema, err := engine.ReplayMigrationsUpTo(migrations, revision)
	if err != nil {
		return nil, &MigrationError{
			Revision:  revision,
			Operation: "replay migrations",
			Cause:     err,
		}
	}

	c.log("Schema at revision %s: %d tables", revision, schema.Count())
	return schema, nil
}

// ListRevisions returns a list of all available migration revisions.
// This is useful for displaying available revisions in the CLI.
func (c *Client) ListRevisions() ([]string, error) {
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, err
	}

	revisions := make([]string, len(migrations))
	for i, m := range migrations {
		revisions[i] = m.Revision
	}

	return revisions, nil
}

// GetMigrationInfo returns information about a specific migration.
func (c *Client) GetMigrationInfo(revision string) (*engine.Migration, error) {
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, err
	}

	for _, m := range migrations {
		if m.Revision == revision {
			return &m, nil
		}
	}

	return nil, &MigrationError{
		Revision:  revision,
		Operation: "get migration info",
		Cause:     fmt.Errorf("revision not found"),
	}
}

// CacheStats holds statistics about the local cache.
type CacheStats struct {
	Revisions       int    // Number of cached revisions
	SchemaSnapshots int    // Number of cached schema snapshots
	MerkleHashes    int    // Number of cached merkle hashes
	CachePath       string // Path to cache database
}

// RebuildCache clears and rebuilds the local cache from migration files.
// This computes schema snapshots and merkle hashes for all revisions.
//
// The cache is stored in .alab/cache.db and is used to speed up operations like:
//   - alab schema --at <rev> (instant instead of replaying migrations)
//   - alab check --quick (compare merkle root hashes)
//
// The cache is always rebuildable from migration files and is gitignored.
func (c *Client) RebuildCache(progress func(revision string, current, total int)) (*CacheStats, error) {
	return c.rebuildCacheInternal(progress, false)
}

// RebuildCacheIncremental only adds missing revisions to the cache.
// Use this for faster updates when you've only added new migrations.
func (c *Client) RebuildCacheIncremental(progress func(revision string, current, total int)) (*CacheStats, error) {
	return c.rebuildCacheInternal(progress, true)
}

func (c *Client) rebuildCacheInternal(progress func(revision string, current, total int), incremental bool) (*CacheStats, error) {
	// Import cache package
	cacheImpl, err := c.openCache()
	if err != nil {
		return nil, err
	}
	defer cacheImpl.Close()

	// Clear cache if not incremental
	if !incremental {
		if err := cacheImpl.Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear cache: %w", err)
		}
	}

	// List all revisions
	revisions, err := c.ListRevisions()
	if err != nil {
		return nil, fmt.Errorf("failed to list revisions: %w", err)
	}

	total := len(revisions)
	cached := 0

	// Build cache for each revision
	for i, rev := range revisions {
		// Check if already cached (for incremental mode)
		if incremental {
			if _, err := cacheImpl.GetSchemaSnapshot(rev); err == nil {
				// Already cached, skip
				continue
			}
		}

		// Report progress
		if progress != nil {
			progress(rev, i+1, total)
		}

		// Compute schema at this revision
		schema, err := c.SchemaAtRevision(rev)
		if err != nil {
			return nil, fmt.Errorf("failed to compute schema at %s: %w", rev, err)
		}

		// Store schema snapshot
		if err := cacheImpl.SetSchemaSnapshot(rev, schema); err != nil {
			return nil, fmt.Errorf("failed to cache schema at %s: %w", rev, err)
		}

		// Compute and store merkle hash
		hash, err := c.computeMerkleHash(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to compute merkle hash at %s: %w", rev, err)
		}

		if err := cacheImpl.SetMerkleHash(rev, hash); err != nil {
			return nil, fmt.Errorf("failed to cache merkle hash at %s: %w", rev, err)
		}

		cached++
	}

	// Get final stats
	stats, err := cacheImpl.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	return &CacheStats{
		Revisions:       total,
		SchemaSnapshots: stats.SchemaSnapshots,
		MerkleHashes:    stats.MerkleHashes,
		CachePath:       cacheImpl.Path(),
	}, nil
}

// ClearCache removes all cached data.
func (c *Client) ClearCache() error {
	cacheImpl, err := c.openCache()
	if err != nil {
		return err
	}
	defer cacheImpl.Close()

	return cacheImpl.Clear()
}

// GetCacheStats returns statistics about the local cache.
func (c *Client) GetCacheStats() (*CacheStats, error) {
	cacheImpl, err := c.openCache()
	if err != nil {
		return nil, err
	}
	defer cacheImpl.Close()

	stats, err := cacheImpl.GetStats()
	if err != nil {
		return nil, err
	}

	revisions, _ := c.ListRevisions()

	return &CacheStats{
		Revisions:       len(revisions),
		SchemaSnapshots: stats.SchemaSnapshots,
		MerkleHashes:    stats.MerkleHashes,
		CachePath:       cacheImpl.Path(),
	}, nil
}
