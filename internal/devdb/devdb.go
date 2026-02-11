// Package devdb provides dev database functionality for schema normalization.
// Following the Atlas pattern: apply schemas to a dev database to normalize them
// into canonical form before comparison.
package devdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/introspect"
)

// DevDatabase represents an ephemeral database used for schema normalization.
// This follows the Atlas pattern: schemas written by humans are in "natural form",
// but databases store them in "canonical/normalized form". By applying a schema
// to a dev database and introspecting it, we get the canonical form for accurate
// comparison.
type DevDatabase struct {
	db      *sql.DB
	dialect dialect.Dialect
}

// New creates a new in-memory dev database.
// Uses SQLite in-memory mode for fast, isolated schema normalization.
func New() (*DevDatabase, error) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to create dev database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping dev database: %w", err)
	}

	d := dialect.Get("sqlite")
	if d == nil {
		db.Close()
		return nil, fmt.Errorf("sqlite dialect not found")
	}

	return &DevDatabase{
		db:      db,
		dialect: d,
	}, nil
}

// Close closes the dev database.
func (d *DevDatabase) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// NormalizeSchema takes a schema (in natural form) and returns the normalized
// version by applying it to the dev database and introspecting it.
//
// This is the key to accurate schema comparison:
// 1. Apply schema to dev DB (database normalizes it)
// 2. Introspect dev DB (get canonical form)
// 3. Compare canonical forms (100% accurate)
func (d *DevDatabase) NormalizeSchema(ctx context.Context, schema *engine.Schema) (*engine.Schema, error) {
	// Apply schema to dev database
	if err := d.applySchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to apply schema to dev database: %w", err)
	}

	// Introspect dev database to get normalized form
	// Use the input schema to build a table name mapping for correct namespace parsing
	intro := introspect.New(d.db, d.dialect)
	if intro == nil {
		return nil, fmt.Errorf("failed to create introspector for dev database")
	}

	mapping := introspect.BuildTableNameMapping(schema)
	normalized, err := intro.IntrospectSchemaWithMapping(ctx, mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect dev database: %w", err)
	}

	return normalized, nil
}

// applySchema applies a schema to the dev database by generating and executing DDL.
func (d *DevDatabase) applySchema(ctx context.Context, schema *engine.Schema) error {
	// Generate CREATE TABLE statements for all tables
	for _, table := range schema.Tables {
		ddl, err := d.generateCreateTable(table)
		if err != nil {
			return err
		}

		// Execute DDL
		if _, err := d.db.ExecContext(ctx, ddl); err != nil {
			return d.formatSQLError(err, "table", table.QualifiedName(), ddl)
		}

		// Create indexes
		for _, idx := range table.Indexes {
			idxDDL := d.generateCreateIndex(table, idx)
			if _, err := d.db.ExecContext(ctx, idxDDL); err != nil {
				return d.formatSQLError(err, "index", idx.Name, idxDDL)
			}
		}
	}

	return nil
}

// formatSQLError converts raw SQLite errors into user-friendly messages with actionable advice.
func (d *DevDatabase) formatSQLError(err error, objectType, objectName, ddl string) error {
	errMsg := err.Error()

	// Detect common error patterns and provide friendly messages
	switch {
	case strings.Contains(errMsg, "already exists"):
		return fmt.Errorf(
			"duplicate %s definition detected: '%s'\n\n"+
				"This usually means:\n"+
				"  • The %s is defined multiple times in your schema files\n"+
				"  • There are conflicting index definitions\n\n"+
				"How to fix:\n"+
				"  1. Search your schema files for duplicate definitions of '%s'\n"+
				"  2. Remove any duplicate %s definitions\n"+
				"  3. Run 'alab check' again\n\n"+
				"Technical details: %s",
			objectType, objectName,
			objectType,
			objectName,
			objectType,
			errMsg,
		)

	case strings.Contains(errMsg, "no such table"):
		return fmt.Errorf(
			"missing table reference in %s '%s'\n\n"+
				"The %s references a table that doesn't exist or hasn't been created yet.\n\n"+
				"How to fix:\n"+
				"  1. Check that all referenced tables are defined in your schema\n"+
				"  2. Ensure table names are spelled correctly\n"+
				"  3. Verify that foreign key references point to existing tables\n\n"+
				"Technical details: %s",
			objectType, objectName,
			objectType,
			errMsg,
		)

	case strings.Contains(errMsg, "syntax error"):
		return fmt.Errorf(
			"SQL syntax error in %s '%s'\n\n"+
				"The generated SQL has a syntax error. This is likely a bug in astroladb.\n\n"+
				"Generated SQL:\n%s\n\n"+
				"Technical details: %s",
			objectType, objectName,
			ddl,
			errMsg,
		)

	default:
		// Generic error with DDL for debugging
		return fmt.Errorf(
			"failed to create %s '%s'\n\n"+
				"Generated SQL:\n%s\n\n"+
				"Error: %s",
			objectType, objectName,
			ddl,
			errMsg,
		)
	}
}

// generateCreateTable generates CREATE TABLE DDL for a table definition.
func (d *DevDatabase) generateCreateTable(table *ast.TableDef) (string, error) {
	// CRITICAL: Use the SAME SQL generator that migrations use
	// This ensures dev database normalization produces identical results

	// Generate CREATE TABLE operation
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{
			Namespace: table.Namespace,
			Name:      table.Name,
		},
		Columns:     table.Columns,
		Indexes:     table.Indexes,
		ForeignKeys: make([]*ast.ForeignKeyDef, 0),
		IfNotExists: false, // Don't use IF NOT EXISTS in dev DB for cleaner SQL
	}

	sql, err := d.dialect.CreateTableSQL(createOp)
	if err != nil {
		return "", fmt.Errorf("failed to generate CREATE TABLE SQL for %s: %w", table.QualifiedName(), err)
	}

	return sql, nil
}

// generateBasicCreateTable generates a basic CREATE TABLE statement as fallback.
func (d *DevDatabase) generateBasicCreateTable(table *ast.TableDef) string {
	tableName := table.Name
	if table.Namespace != "" {
		tableName = table.Namespace + "_" + table.Name
	}

	sql := fmt.Sprintf("CREATE TABLE %s (\n", d.dialect.QuoteIdent(tableName))

	// Add columns
	for i, col := range table.Columns {
		if i > 0 {
			sql += ",\n"
		}
		sql += "  " + d.generateColumnDef(col)
	}

	sql += "\n)"
	return sql
}

// generateColumnDef generates a column definition.
func (d *DevDatabase) generateColumnDef(col *ast.ColumnDef) string {
	sql := d.dialect.QuoteIdent(col.Name) + " "

	// Map type to SQL type
	sqlType := d.mapTypeToSQL(col.Type, col.TypeArgs)
	sql += sqlType

	// Primary key
	if col.PrimaryKey {
		sql += " PRIMARY KEY"
		if col.Type == "id" || col.Type == "integer" {
			sql += " AUTOINCREMENT"
		}
	}

	// Nullable
	if !col.Nullable && !col.PrimaryKey {
		sql += " NOT NULL"
	}

	// Default
	if col.Default != nil {
		sql += " DEFAULT " + formatDevDefault(col.Default)
	} else if col.ServerDefault != "" {
		if err := ast.ValidateSQLExpression(col.ServerDefault); err == nil {
			sql += " DEFAULT " + col.ServerDefault
		}
	}

	// Unique
	if col.Unique {
		sql += " UNIQUE"
	}

	return sql
}

// formatDevDefault formats a default value safely for SQL.
func formatDevDefault(v any) string {
	switch val := v.(type) {
	case string:
		escaped := strings.ReplaceAll(val, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case nil:
		return "NULL"
	default:
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", val), "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}

// mapTypeToSQL maps Alab types to SQL types.
func (d *DevDatabase) mapTypeToSQL(alabType string, args []any) string {
	switch alabType {
	case "id":
		return "INTEGER"
	case "integer":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "string":
		if len(args) > 0 {
			return "TEXT"
		}
		return "TEXT"
	case "text":
		return "TEXT"
	case "boolean":
		return "INTEGER" // SQLite uses INTEGER for boolean
	case "datetime":
		return "DATETIME"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "float":
		return "REAL"
	case "decimal":
		return "NUMERIC"
	case "json":
		return "TEXT"
	case "uuid":
		return "TEXT"
	default:
		return "TEXT"
	}
}

// generateCreateIndex generates CREATE INDEX DDL.
func (d *DevDatabase) generateCreateIndex(table *ast.TableDef, idx *ast.IndexDef) string {
	tableName := table.Name
	if table.Namespace != "" {
		tableName = table.Namespace + "_" + table.Name
	}

	indexName := idx.Name
	if indexName == "" {
		// Generate index name
		indexName = fmt.Sprintf("idx_%s_%s", tableName, idx.Columns[0])
	}

	unique := ""
	if idx.Unique {
		unique = "UNIQUE "
	}

	columns := ""
	for i, col := range idx.Columns {
		if i > 0 {
			columns += ", "
		}
		columns += d.dialect.QuoteIdent(col)
	}

	return fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)",
		unique,
		d.dialect.QuoteIdent(indexName),
		d.dialect.QuoteIdent(tableName),
		columns,
	)
}
