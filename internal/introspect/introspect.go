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
)

// Introspector queries database catalogs to discover schema information.
type Introspector interface {
	// IntrospectSchema returns the complete database schema as engine.Schema.
	// It skips internal tables like alab_migrations.
	IntrospectSchema(ctx context.Context) (*engine.Schema, error)

	// IntrospectTable returns a single table definition, or nil if not found.
	IntrospectTable(ctx context.Context, tableName string) (*ast.TableDef, error)

	// TableExists checks if a table exists in the database.
	TableExists(ctx context.Context, tableName string) (bool, error)
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

// parseTableName extracts namespace and table name from SQL table name.
// Example: "auth_user" -> ("auth", "user")
// Example: "user" -> ("", "user")
func parseTableName(sqlName string) (namespace, name string) {
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
