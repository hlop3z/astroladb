// Package dialect provides database-specific SQL generation.
// Each dialect implements type mappings from the JS DSL to SQL,
// identifier quoting, and DDL statement generation.
package dialect

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// Dialect defines the interface for database-specific SQL generation.
// Implementations exist for PostgreSQL and SQLite.
type Dialect interface {
	// Name returns the dialect name (postgres, sqlite).
	Name() string

	// -------------------------------------------------------------------------
	// Type mappings (JS DSL -> SQL)
	// -------------------------------------------------------------------------

	// IDType returns the UUID primary key type with auto-generation.
	// PostgreSQL: UUID DEFAULT gen_random_uuid()
	// SQLite: TEXT
	IDType() string

	// StringType returns a bounded string type.
	// PostgreSQL: VARCHAR(length)
	// SQLite: TEXT
	StringType(length int) string

	// TextType returns an unbounded text type.
	// All dialects: TEXT
	TextType() string

	// IntegerType returns a 32-bit integer type (JS-safe).
	// PostgreSQL: INTEGER
	// SQLite: INTEGER
	IntegerType() string

	// FloatType returns a 32-bit floating-point type.
	// PostgreSQL: REAL
	// SQLite: REAL
	FloatType() string

	// DecimalType returns an arbitrary-precision decimal type for money.
	// PostgreSQL: DECIMAL(precision, scale)
	// SQLite: TEXT (stored as string for precision)
	DecimalType(precision, scale int) string

	// BooleanType returns a boolean type.
	// PostgreSQL: BOOLEAN
	// SQLite: INTEGER
	BooleanType() string

	// DateType returns a date-only type.
	// PostgreSQL: DATE
	// SQLite: TEXT (ISO 8601 format)
	DateType() string

	// TimeType returns a time-only type.
	// PostgreSQL: TIME
	// SQLite: TEXT (RFC 3339 format)
	TimeType() string

	// DateTimeType returns a timestamp with timezone type.
	// PostgreSQL: TIMESTAMPTZ
	// SQLite: TEXT (RFC 3339 format)
	DateTimeType() string

	// UUIDType returns a UUID storage type.
	// PostgreSQL: UUID
	// SQLite: TEXT
	UUIDType() string

	// JSONType returns a JSON storage type.
	// PostgreSQL: JSONB
	// SQLite: TEXT
	JSONType() string

	// Base64Type returns a binary data type for base64 content.
	// PostgreSQL: BYTEA
	// SQLite: BLOB
	Base64Type() string

	// EnumType returns an enum type definition.
	// PostgreSQL: Creates a named type (handled separately)
	// SQLite: TEXT with CHECK constraint
	EnumType(name string, values []string) string

	// -------------------------------------------------------------------------
	// Identifiers
	// -------------------------------------------------------------------------

	// QuoteIdent quotes an identifier (table/column name) for the dialect.
	// PostgreSQL/SQLite: "name"
	QuoteIdent(name string) string

	// Placeholder returns a parameter placeholder for the given index (1-based).
	// PostgreSQL: $1, $2, $3, ...
	// SQLite: ?, ?, ?, ...
	Placeholder(index int) string

	// -------------------------------------------------------------------------
	// Feature support
	// -------------------------------------------------------------------------

	// SupportsTransactionalDDL returns true if DDL can be wrapped in transactions.
	// PostgreSQL: true
	// SQLite: true
	SupportsTransactionalDDL() bool

	// SupportsIfExists returns true if the dialect supports IF EXISTS clauses.
	// All supported dialects: true
	SupportsIfExists() bool

	// -------------------------------------------------------------------------
	// SQL generation for operations
	// -------------------------------------------------------------------------

	// CreateTableSQL generates CREATE TABLE statement.
	CreateTableSQL(op *ast.CreateTable) (string, error)

	// DropTableSQL generates DROP TABLE statement.
	DropTableSQL(op *ast.DropTable) (string, error)

	// AddColumnSQL generates ALTER TABLE ADD COLUMN statement.
	AddColumnSQL(op *ast.AddColumn) (string, error)

	// DropColumnSQL generates ALTER TABLE DROP COLUMN statement.
	DropColumnSQL(op *ast.DropColumn) (string, error)

	// RenameColumnSQL generates column rename statement.
	// PostgreSQL/SQLite: ALTER TABLE t RENAME COLUMN old TO new
	RenameColumnSQL(op *ast.RenameColumn) (string, error)

	// AlterColumnSQL generates column modification statement.
	AlterColumnSQL(op *ast.AlterColumn) (string, error)

	// CreateIndexSQL generates CREATE INDEX statement.
	CreateIndexSQL(op *ast.CreateIndex) (string, error)

	// DropIndexSQL generates DROP INDEX statement.
	DropIndexSQL(op *ast.DropIndex) (string, error)

	// RenameTableSQL generates ALTER TABLE RENAME statement.
	RenameTableSQL(op *ast.RenameTable) (string, error)

	// AddForeignKeySQL generates ALTER TABLE ADD CONSTRAINT FOREIGN KEY statement.
	AddForeignKeySQL(op *ast.AddForeignKey) (string, error)

	// DropForeignKeySQL generates ALTER TABLE DROP CONSTRAINT/FOREIGN KEY statement.
	DropForeignKeySQL(op *ast.DropForeignKey) (string, error)

	// AddCheckSQL generates ALTER TABLE ADD CONSTRAINT CHECK statement.
	AddCheckSQL(op *ast.AddCheck) (string, error)

	// DropCheckSQL generates ALTER TABLE DROP CONSTRAINT statement for CHECK constraints.
	DropCheckSQL(op *ast.DropCheck) (string, error)

	// RawSQLFor returns the SQL for a RawSQL operation, using dialect-specific override if available.
	RawSQLFor(op *ast.RawSQL) (string, error)
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
