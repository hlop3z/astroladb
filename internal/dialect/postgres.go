package dialect

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// postgres implements the Dialect interface for PostgreSQL.
type postgres struct{}

// Postgres returns the PostgreSQL dialect implementation.
func Postgres() Dialect {
	return &postgres{}
}

func (d *postgres) Name() string {
	return "postgres"
}

// -----------------------------------------------------------------------------
// Type mappings
// -----------------------------------------------------------------------------

func (d *postgres) IDType() string {
	return "UUID DEFAULT gen_random_uuid()"
}

func (d *postgres) StringType(length int) string {
	return fmt.Sprintf("VARCHAR(%d)", length)
}

func (d *postgres) TextType() string {
	return "TEXT"
}

func (d *postgres) IntegerType() string {
	return "INTEGER"
}

func (d *postgres) FloatType() string {
	return "REAL"
}

func (d *postgres) DecimalType(precision, scale int) string {
	return fmt.Sprintf("DECIMAL(%d, %d)", precision, scale)
}

func (d *postgres) BooleanType() string {
	return "BOOLEAN"
}

func (d *postgres) DateType() string {
	return "DATE"
}

func (d *postgres) TimeType() string {
	return "TIME"
}

func (d *postgres) DateTimeType() string {
	return "TIMESTAMPTZ"
}

func (d *postgres) UUIDType() string {
	return "UUID"
}

func (d *postgres) JSONType() string {
	return "JSONB"
}

func (d *postgres) Base64Type() string {
	return "BYTEA"
}

func (d *postgres) EnumType(name string, values []string) string {
	// Use VARCHAR(50) with CHECK constraint (same approach as SQLite and export.go).
	// This avoids needing separate CREATE TYPE DDL for enums.
	return "VARCHAR(50)"
}

// -----------------------------------------------------------------------------
// Identifiers
// -----------------------------------------------------------------------------

func (d *postgres) QuoteIdent(name string) string {
	return quoteIdentDoubleQuote(name)
}

func (d *postgres) Placeholder(index int) string {
	return "$" + strconv.Itoa(index)
}

func (d *postgres) BooleanLiteral(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

func (d *postgres) CurrentTimestamp() string {
	return "CURRENT_TIMESTAMP"
}

// -----------------------------------------------------------------------------
// Feature support
// -----------------------------------------------------------------------------

func (d *postgres) SupportsTransactionalDDL() bool {
	return true
}

func (d *postgres) SupportsIfExists() bool {
	return true
}

// -----------------------------------------------------------------------------
// SQL generation
// -----------------------------------------------------------------------------

func (d *postgres) CreateTableSQL(op *ast.CreateTable) (string, error) {
	return buildCreateTableSQL(op, d.QuoteIdent, d.columnDefSQL, d.foreignKeyConstraintSQL)
}

func (d *postgres) DropTableSQL(op *ast.DropTable) (string, error) {
	return buildDropTableSQL(op, d.QuoteIdent)
}

func (d *postgres) AddColumnSQL(op *ast.AddColumn) (string, error) {
	return buildAddColumnSQL(op, d.QuoteIdent, d.columnDefSQL)
}

func (d *postgres) DropColumnSQL(op *ast.DropColumn) (string, error) {
	return buildDropColumnSQL(op, d.QuoteIdent)
}

func (d *postgres) RenameColumnSQL(op *ast.RenameColumn) (string, error) {
	return buildRenameColumnSQL(op, d.QuoteIdent)
}

func (d *postgres) AlterColumnSQL(op *ast.AlterColumn) (string, error) {
	var statements []string
	tableName := d.QuoteIdent(op.Table())
	colName := d.QuoteIdent(op.Name)

	// Type change
	if op.NewType != "" {
		sqlType := d.columnTypeSQL(op.NewType, op.NewTypeArgs)
		stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", tableName, colName, sqlType)
		statements = append(statements, stmt)
	}

	// Nullability change
	if op.SetNullable != nil {
		if *op.SetNullable {
			stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", tableName, colName)
			statements = append(statements, stmt)
		} else {
			stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", tableName, colName)
			statements = append(statements, stmt)
		}
	}

	// Default change
	if op.DropDefault {
		stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", tableName, colName)
		statements = append(statements, stmt)
	} else if op.ServerDefault != "" {
		if err := ast.ValidateSQLExpression(op.ServerDefault); err != nil {
			return "", fmt.Errorf("unsafe ServerDefault expression: %w", err)
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", tableName, colName, op.ServerDefault)
		statements = append(statements, stmt)
	} else if op.SetDefault != nil {
		defaultVal := d.defaultValueSQL(op.SetDefault)
		stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", tableName, colName, defaultVal)
		statements = append(statements, stmt)
	}

	return strings.Join(statements, ";\n"), nil
}

func (d *postgres) CreateIndexSQL(op *ast.CreateIndex) (string, error) {
	return buildCreateIndexSQL(op, d.QuoteIdent, IndexSQLOpts{
		SupportsIfNotExists: true,
		IndexNameFunc:       indexNameFunc,
	})
}

func (d *postgres) DropIndexSQL(op *ast.DropIndex) (string, error) {
	return buildDropIndexSQL(op, d.QuoteIdent)
}

func (d *postgres) RenameTableSQL(op *ast.RenameTable) (string, error) {
	return buildRenameTableSQL(op, d.QuoteIdent, strutil.QualifyTable)
}

func (d *postgres) AddForeignKeySQL(op *ast.AddForeignKey) (string, error) {
	return buildAddForeignKeySQL(op, d.QuoteIdent)
}

func (d *postgres) DropForeignKeySQL(op *ast.DropForeignKey) (string, error) {
	return buildDropConstraintSQL(op.Table(), op.Name, d.QuoteIdent), nil
}

func (d *postgres) AddCheckSQL(op *ast.AddCheck) (string, error) {
	return buildAddCheckSQL(op, d.QuoteIdent)
}

func (d *postgres) DropCheckSQL(op *ast.DropCheck) (string, error) {
	return buildDropConstraintSQL(op.Table(), op.Name, d.QuoteIdent), nil
}

func (d *postgres) RawSQLFor(op *ast.RawSQL) (string, error) {
	if op.Postgres != "" {
		return op.Postgres, nil
	}
	return op.SQL, nil
}

// -----------------------------------------------------------------------------
// Helper methods
// -----------------------------------------------------------------------------

// columnDefSQL generates the SQL for a column definition.
func (d *postgres) columnDefSQL(col *ast.ColumnDef, tableName string) string {
	return buildColumnDefSQL(col, ColumnDefConfig{
		QuoteIdent:  d.QuoteIdent,
		TypeSQL:     d.columnTypeSQL,
		DefaultSQL:  d.defaultValueSQL,
		Order:       PostgresColumnOrder,
		TableName:   tableName,
		EnumCheck:   d.enumCheckSQL,
		DialectName: "postgres",
	})
}

// enumCheckSQL generates the CHECK constraint for enum columns.
func (d *postgres) enumCheckSQL(col *ast.ColumnDef, tableName string) string {
	return buildEnumCheckSQL(col, tableName, d.QuoteIdent)
}

// columnTypeSQL returns the SQL type for a column.
func (d *postgres) columnTypeSQL(typeName string, typeArgs []any) string {
	// Handle enum — use VARCHAR(50) with CHECK (same approach as SQLite)
	if typeName == "enum" {
		return "VARCHAR(50)"
	}
	// Use shared implementation for all other types
	return buildColumnTypeSQL(typeName, typeArgs, d)
}

// defaultValueSQL returns the SQL representation of a default value.
func (d *postgres) defaultValueSQL(value any) string {
	return buildDefaultValueSQL(value, "postgres", PostgresBooleans)
}

// foreignKeyConstraintSQL generates a foreign key constraint.
func (d *postgres) foreignKeyConstraintSQL(fk *ast.ForeignKeyDef) string {
	return buildForeignKeyConstraintSQL(fk, d.QuoteIdent)
}
