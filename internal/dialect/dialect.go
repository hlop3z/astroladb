// Package dialect provides database-specific SQL generation.
// Each dialect implements type mappings from the JS DSL to SQL,
// identifier quoting, and DDL statement generation.
package dialect

import (
	"strings"

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

// OperationToSQL dispatches an AST operation to the appropriate DDLGenerator method.
// AlterColumn may produce multiple statements (separated by ";\n"), which are split
// into individual entries. Returns nil, nil for unrecognized operation types.
func OperationToSQL(d Dialect, op ast.Operation) ([]string, error) {
	switch o := op.(type) {
	case *ast.CreateTable:
		s, err := d.CreateTableSQL(o)
		return []string{s}, err
	case *ast.DropTable:
		s, err := d.DropTableSQL(o)
		return []string{s}, err
	case *ast.RenameTable:
		s, err := d.RenameTableSQL(o)
		return []string{s}, err
	case *ast.AddColumn:
		s, err := d.AddColumnSQL(o)
		return []string{s}, err
	case *ast.DropColumn:
		s, err := d.DropColumnSQL(o)
		return []string{s}, err
	case *ast.RenameColumn:
		s, err := d.RenameColumnSQL(o)
		return []string{s}, err
	case *ast.AlterColumn:
		s, err := d.AlterColumnSQL(o)
		if err != nil {
			return nil, err
		}
		return SplitStatements(s), nil
	case *ast.CreateIndex:
		s, err := d.CreateIndexSQL(o)
		return []string{s}, err
	case *ast.DropIndex:
		s, err := d.DropIndexSQL(o)
		return []string{s}, err
	case *ast.AddForeignKey:
		s, err := d.AddForeignKeySQL(o)
		return []string{s}, err
	case *ast.DropForeignKey:
		s, err := d.DropForeignKeySQL(o)
		return []string{s}, err
	case *ast.AddCheck:
		s, err := d.AddCheckSQL(o)
		return []string{s}, err
	case *ast.DropCheck:
		s, err := d.DropCheckSQL(o)
		return []string{s}, err
	case *ast.RawSQL:
		s, err := d.RawSQLFor(o)
		return []string{s}, err
	default:
		return nil, nil
	}
}

// SplitStatements splits multi-statement SQL (separated by ";\n") into individual statements.
func SplitStatements(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";\n") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
