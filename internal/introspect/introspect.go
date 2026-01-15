// Package introspect provides database schema introspection for all supported dialects.
// It queries database system catalogs to discover existing tables, columns, indexes,
// and foreign keys, then converts them to AST structures for schema comparison.
package introspect

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// Introspector queries database catalogs to discover schema information.
type Introspector interface {
	// IntrospectSchema returns the complete database schema as engine.Schema.
	// It skips internal tables like alab_migrations.
	IntrospectSchema(ctx context.Context) (*engine.Schema, error)

	// IntrospectSchemaWithMapping returns the complete database schema using a
	// table name mapping to resolve namespace/table name ambiguity.
	IntrospectSchemaWithMapping(ctx context.Context, mapping TableNameMapping) (*engine.Schema, error)

	// IntrospectTable returns a single table definition, or nil if not found.
	IntrospectTable(ctx context.Context, tableName string) (*ast.TableDef, error)

	// TableExists checks if a table exists in the database.
	TableExists(ctx context.Context, tableName string) (bool, error)
}

// BuildTableNameMapping creates a mapping from SQL table names to their
// namespace and table name components from an engine.Schema.
// This is used to resolve parsing ambiguity when introspecting databases.
func BuildTableNameMapping(schema *engine.Schema) TableNameMapping {
	if schema == nil {
		return nil
	}

	mapping := make(TableNameMapping)
	for _, table := range schema.Tables {
		// Use the same SQLName function that's used during schema evaluation
		sqlName := strutil.SQLName(table.Namespace, table.Name)

		mapping[sqlName] = struct {
			Namespace string
			Name      string
		}{
			Namespace: table.Namespace,
			Name:      table.Name,
		}
	}
	return mapping
}

// New creates an Introspector for the given dialect.
// Returns nil if the dialect is not supported.
func New(db *sql.DB, d dialect.Dialect) Introspector {
	switch d.Name() {
	case "postgres":
		return &postgresIntrospector{db: db, dialect: d}
	case "sqlite":
		return &sqliteIntrospector{db: db, dialect: d}
	default:
		return nil
	}
}

// RawColumn represents column metadata from database catalog.
type RawColumn struct {
	Name         string
	DataType     string // Raw SQL type (VARCHAR, INTEGER, etc.)
	IsNullable   bool
	Default      sql.NullString // Raw default expression
	IsPrimaryKey bool
	IsUnique     bool
	MaxLength    sql.NullInt64 // For VARCHAR(n)
	Precision    sql.NullInt64 // For DECIMAL(p,s)
	Scale        sql.NullInt64
}

// RawIndex represents index metadata from database catalog.
type RawIndex struct {
	Name    string
	Columns []string
	Unique  bool
}

// RawForeignKey represents FK metadata from database catalog.
type RawForeignKey struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
}

// TableNameMapping holds SQL table names to their namespace/table components.
// This is used to resolve the ambiguity when parsing SQL table names that may
// contain underscores in either the namespace or table name.
type TableNameMapping map[string]struct {
	Namespace string
	Name      string
}

// parseTableName extracts namespace and table name from SQL table name.
// If a mapping is provided, it will use that first (context-aware parsing).
// Otherwise, it falls back to naive first-underscore splitting.
//
// Example with mapping:
//
//	mapping["my_app_users"] = {Namespace: "my_app", Name: "users"}
//	parseTableName("my_app_users", mapping) -> ("my_app", "users")  ✓ Correct!
//
// Example without mapping (fallback):
//
//	parseTableName("my_app_users", nil) -> ("my", "app_users")  ✗ Wrong, but unavoidable
func parseTableName(sqlName string, mapping TableNameMapping) (namespace, name string) {
	// First, try to find it in the mapping (from schema files)
	if mapping != nil {
		if entry, found := mapping[sqlName]; found {
			return entry.Namespace, entry.Name
		}
	}

	// Fallback to naive parsing (for tables not in schema files)
	idx := strings.Index(sqlName, "_")
	if idx == -1 {
		return "", sqlName
	}
	return sqlName[:idx], sqlName[idx+1:]
}

// normalizeAction converts SQL action to Alab format.
func normalizeAction(action string) string {
	switch strings.ToUpper(action) {
	case "CASCADE":
		return "CASCADE"
	case "SET NULL":
		return "SET NULL"
	case "RESTRICT":
		return "RESTRICT"
	case "NO ACTION", "":
		return "" // Default, don't include
	default:
		return ""
	}
}

// internalTables lists tables that should be skipped during introspection.
var internalTables = map[string]bool{
	"alab_migrations": true,
}

// isInternalTable checks if a table should be skipped.
func isInternalTable(name string) bool {
	return internalTables[name]
}
