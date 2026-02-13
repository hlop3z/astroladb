// Package dialect provides database-specific SQL generation.
// Each dialect implements type mappings from the JS DSL to SQL,
// identifier quoting, and DDL statement generation.
package dialect

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// DDLGenerator generates SQL DDL statements from AST operations.
// Used by the engine to produce migration SQL.
type DDLGenerator interface {
	CreateTableSQL(op *ast.CreateTable) (string, error)
	DropTableSQL(op *ast.DropTable) (string, error)
	AddColumnSQL(op *ast.AddColumn) (string, error)
	DropColumnSQL(op *ast.DropColumn) (string, error)
	RenameColumnSQL(op *ast.RenameColumn) (string, error)
	AlterColumnSQL(op *ast.AlterColumn) (string, error)
	CreateIndexSQL(op *ast.CreateIndex) (string, error)
	DropIndexSQL(op *ast.DropIndex) (string, error)
	RenameTableSQL(op *ast.RenameTable) (string, error)
	AddForeignKeySQL(op *ast.AddForeignKey) (string, error)
	DropForeignKeySQL(op *ast.DropForeignKey) (string, error)
	AddCheckSQL(op *ast.AddCheck) (string, error)
	DropCheckSQL(op *ast.DropCheck) (string, error)
	RawSQLFor(op *ast.RawSQL) (string, error)
}

// SQLFormatter handles SQL identifier quoting and parameter placeholders.
type SQLFormatter interface {
	QuoteIdent(name string) string
	Placeholder(index int) string
	BooleanLiteral(b bool) string
	CurrentTimestamp() string
}

// FeatureDetector reports database-specific capabilities.
type FeatureDetector interface {
	SupportsTransactionalDDL() bool
	SupportsIfExists() bool
}

// Dialect defines the complete interface for database-specific SQL generation.
// It embeds all sub-interfaces plus a Name() identity method.
// Implementations exist for PostgreSQL and SQLite.
//
// Sub-interfaces allow consumers to depend only on what they need:
//   - TypeMapper (defined in base.go): DSL type → SQL type conversion
//   - DDLGenerator: SQL DDL statement generation from AST operations
//   - SQLFormatter: identifier quoting and parameter placeholders
//   - FeatureDetector: database capability detection
type Dialect interface {
	Name() string
	TypeMapper
	DDLGenerator
	SQLFormatter
	FeatureDetector
}

// Get returns the dialect implementation for the given name.
// Valid names: "postgres", "postgresql", "sqlite", "sqlite3".
// Returns nil if the dialect is not supported.
func Get(name string) Dialect {
	switch name {
	case "postgres", "postgresql":
		return Postgres()
	case "sqlite", "sqlite3":
		return SQLite()
	default:
		return nil
	}
}

// Names returns the list of supported dialect names.
func Names() []string {
	return []string{"postgres", "sqlite"}
}
