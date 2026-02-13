package dialect

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// sqlite implements the Dialect interface for SQLite.
type sqlite struct{}

// SQLite returns the SQLite dialect implementation.
func SQLite() Dialect {
	return &sqlite{}
}

func (d *sqlite) Name() string {
	return "sqlite"
}

// -----------------------------------------------------------------------------
// Type mappings
// SQLite has dynamic typing with type affinities: TEXT, INTEGER, REAL, BLOB
// Most types map to TEXT for simplicity and compatibility.
// -----------------------------------------------------------------------------

func (d *sqlite) IDType() string {
	// SQLite has no native UUID type; use TEXT.
	// UUID generation must be handled by the application.
	return "TEXT"
}

func (d *sqlite) StringType(length int) string {
	// SQLite ignores length constraints; use TEXT.
	// Length validation should happen at the application level.
	return "TEXT"
}

func (d *sqlite) TextType() string {
	return "TEXT"
}

func (d *sqlite) IntegerType() string {
	return "INTEGER"
}

func (d *sqlite) FloatType() string {
	return "REAL"
}

func (d *sqlite) DecimalType(precision, scale int) string {
	// SQLite has no native DECIMAL; use TEXT for precision preservation.
	// The application must handle string-to-decimal conversion.
	return "TEXT"
}

func (d *sqlite) BooleanType() string {
	// SQLite has no native BOOLEAN; use INTEGER (0 = false, 1 = true).
	return "INTEGER"
}

func (d *sqlite) DateType() string {
	// SQLite stores dates as TEXT, but we use DATE type affinity for clarity
	return "DATE"
}

func (d *sqlite) TimeType() string {
	// SQLite stores time as TEXT, but we use TIME type affinity for clarity
	return "TIME"
}

func (d *sqlite) DateTimeType() string {
	// SQLite stores datetime as TEXT, but we use DATETIME type affinity for clarity
	// This allows round-trip: datetime -> DATETIME -> introspect -> datetime
	return "DATETIME"
}

func (d *sqlite) UUIDType() string {
	return "TEXT"
}

func (d *sqlite) JSONType() string {
	// SQLite has JSON1 extension, but stores as TEXT.
	return "TEXT"
}

func (d *sqlite) Base64Type() string {
	return "BLOB"
}

func (d *sqlite) EnumType(name string, values []string) string {
	// SQLite uses TEXT with CHECK constraint for enums.
	// The CHECK constraint is added at the column level.
	return "TEXT"
}

// -----------------------------------------------------------------------------
// Identifiers
// -----------------------------------------------------------------------------

func (d *sqlite) QuoteIdent(name string) string {
	return quoteIdentDoubleQuote(name)
}

func (d *sqlite) Placeholder(index int) string {
	// SQLite uses ? for all placeholders
	return "?"
}

func (d *sqlite) BooleanLiteral(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (d *sqlite) CurrentTimestamp() string {
	return "datetime('now')"
}

// -----------------------------------------------------------------------------
// Feature support
// -----------------------------------------------------------------------------

func (d *sqlite) SupportsTransactionalDDL() bool {
	return true
}

func (d *sqlite) SupportsIfExists() bool {
	return true
}

// -----------------------------------------------------------------------------
// SQL generation
// -----------------------------------------------------------------------------

func (d *sqlite) CreateTableSQL(op *ast.CreateTable) (string, error) {
	return buildCreateTableSQL(op, d.QuoteIdent, d.columnDefSQL, d.foreignKeyConstraintSQL)
}

func (d *sqlite) DropTableSQL(op *ast.DropTable) (string, error) {
	return buildDropTableSQL(op, d.QuoteIdent)
}

func (d *sqlite) AddColumnSQL(op *ast.AddColumn) (string, error) {
	// SQLite cannot ALTER TABLE ADD COLUMN with STORED generated columns.
	// Only VIRTUAL columns can be added via ALTER TABLE.
	// See: https://sqlite.org/gencol.html
	if op.Column != nil && op.Column.Computed != nil && !op.Column.Virtual {
		return "", fmt.Errorf("sqlite: cannot add a STORED generated column via ALTER TABLE; use CREATE TABLE instead")
	}
	return buildAddColumnSQL(op, d.QuoteIdent, d.columnDefSQL)
}

func (d *sqlite) DropColumnSQL(op *ast.DropColumn) (string, error) {
	// SQLite 3.35.0+ supports DROP COLUMN
	return buildDropColumnSQL(op, d.QuoteIdent)
}

func (d *sqlite) RenameColumnSQL(op *ast.RenameColumn) (string, error) {
	// SQLite 3.25.0+ supports RENAME COLUMN
	return buildRenameColumnSQL(op, d.QuoteIdent)
}

func (d *sqlite) AlterColumnSQL(op *ast.AlterColumn) (string, error) {
	// SQLite has very limited ALTER TABLE support.
	// To modify column type, nullability, or default, you need table recreation:
	// 1. Create new table with desired schema
	// 2. Copy data from old table
	// 3. Drop old table
	// 4. Rename new table to old name
	//
	// This is a complex operation that should be handled at a higher level.
	// For now, return an error suggesting table recreation.
	return "", alerr.New(alerr.ErrSQLExecution, "SQLite does not support ALTER COLUMN; use table recreation pattern").
		WithTable(op.Namespace, op.Table_).
		WithColumn(op.Name)
}

func (d *sqlite) CreateIndexSQL(op *ast.CreateIndex) (string, error) {
	return buildCreateIndexSQL(op, d.QuoteIdent, IndexSQLOpts{
		SupportsIfNotExists: true,
		IndexNameFunc:       indexNameFunc,
	})
}

func (d *sqlite) DropIndexSQL(op *ast.DropIndex) (string, error) {
	return buildDropIndexSQL(op, d.QuoteIdent)
}

func (d *sqlite) RenameTableSQL(op *ast.RenameTable) (string, error) {
	return buildRenameTableSQL(op, d.QuoteIdent, strutil.QualifyTable)
}

// sqliteUnsupported returns a standardized error for unsupported ALTER TABLE operations.
func sqliteUnsupported(msg, ns, table, key, name string) (string, error) {
	return "", alerr.New(alerr.ErrSQLExecution, msg).
		WithTable(ns, table).
		With(key, name)
}

func (d *sqlite) AddForeignKeySQL(op *ast.AddForeignKey) (string, error) {
	return sqliteUnsupported("SQLite does not support ALTER TABLE ADD FOREIGN KEY; use table recreation pattern",
		op.Namespace, op.Table_, "fk_name", op.Name)
}

func (d *sqlite) DropForeignKeySQL(op *ast.DropForeignKey) (string, error) {
	return sqliteUnsupported("SQLite does not support ALTER TABLE DROP FOREIGN KEY; use table recreation pattern",
		op.Namespace, op.Table_, "fk_name", op.Name)
}

func (d *sqlite) AddCheckSQL(op *ast.AddCheck) (string, error) {
	return sqliteUnsupported("SQLite does not support ALTER TABLE ADD CHECK; use table recreation pattern",
		op.Namespace, op.Table_, "check_name", op.Name)
}

func (d *sqlite) DropCheckSQL(op *ast.DropCheck) (string, error) {
	return sqliteUnsupported("SQLite does not support ALTER TABLE DROP CHECK; use table recreation pattern",
		op.Namespace, op.Table_, "check_name", op.Name)
}

func (d *sqlite) RawSQLFor(op *ast.RawSQL) (string, error) {
	if op.SQLite != "" {
		return op.SQLite, nil
	}
	return op.SQL, nil
}

// -----------------------------------------------------------------------------
// Helper methods
// -----------------------------------------------------------------------------

// columnDefSQL generates the SQL for a column definition.
func (d *sqlite) columnDefSQL(col *ast.ColumnDef, tableName string) string {
	return buildColumnDefSQL(col, ColumnDefConfig{
		QuoteIdent:  d.QuoteIdent,
		TypeSQL:     d.columnTypeSQL,
		DefaultSQL:  d.defaultValueSQL,
		Order:       PostgresColumnOrder, // SQLite uses same order as PostgreSQL
		TableName:   tableName,
		EnumCheck:   d.enumCheckSQL,
		DialectName: "sqlite",
	})
}

// enumCheckSQL generates the CHECK constraint for enum columns.
func (d *sqlite) enumCheckSQL(col *ast.ColumnDef, tableName string) string {
	return buildEnumCheckSQL(col, tableName, d.QuoteIdent)
}

// columnTypeSQL returns the SQL type for a column.
func (d *sqlite) columnTypeSQL(typeName string, typeArgs []any) string {
	// SQLite-specific overrides
	switch typeName {
	case "string":
		// SQLite ignores length, always use TEXT
		return d.TextType()
	case "enum":
		// SQLite uses TEXT with CHECK constraint (handled elsewhere)
		return d.EnumType("", nil)
	}
	// Use shared implementation for all other types
	return buildColumnTypeSQL(typeName, typeArgs, d)
}

// defaultValueSQL returns the SQL representation of a default value.
func (d *sqlite) defaultValueSQL(value any) string {
	// Handle SQLExpr specially for SQLite's NOW() -> CURRENT_TIMESTAMP conversion
	if sqlExpr, ok := value.(*ast.SQLExpr); ok {
		expr := sqlExpr.Expr
		expr = strings.ReplaceAll(expr, "NOW()", "CURRENT_TIMESTAMP")
		expr = strings.ReplaceAll(expr, "now()", "CURRENT_TIMESTAMP")
		return expr
	}
	// Use shared implementation for all other value types
	return buildDefaultValueSQL(value, SQLiteBooleans)
}

// foreignKeyConstraintSQL generates a foreign key constraint.
// Note: SQLite doesn't fully support named constraints on foreign keys,
// but the name is included when provided for documentation purposes.
func (d *sqlite) foreignKeyConstraintSQL(fk *ast.ForeignKeyDef) string {
	return buildForeignKeyConstraintSQL(fk, d.QuoteIdent)
}
