// Package dialect provides database-specific SQL generation.
// This file contains shared helper functions used by all dialect implementations.
package dialect

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
)

// QuoteIdentFunc is a function that quotes an identifier.
type QuoteIdentFunc func(name string) string

// writeQuotedList writes comma-separated quoted identifiers to the builder.
// This is a DRY helper used by FK constraints, indexes, and other column lists.
func writeQuotedList(b *strings.Builder, items []string, quote QuoteIdentFunc) {
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quote(item))
	}
}

// TypeMapper provides type-specific SQL generation.
// Each dialect implements these methods.
type TypeMapper interface {
	IDType() string
	StringType(length int) string
	TextType() string
	IntegerType() string
	FloatType() string
	DecimalType(precision, scale int) string
	BooleanType() string
	DateType() string
	TimeType() string
	DateTimeType() string
	UUIDType() string
	JSONType() string
	Base64Type() string
	EnumType(name string, values []string) string
}

// buildColumnTypeSQL generates the SQL type for a column using the type mapper.
// This is shared logic - only enum handling differs between dialects.
func buildColumnTypeSQL(typeName string, typeArgs []any, mapper TypeMapper) string {
	switch typeName {
	case "id":
		return mapper.IDType()
	case "string":
		length := 255
		if len(typeArgs) > 0 {
			if l, ok := typeArgs[0].(int); ok {
				length = l
			} else if l, ok := typeArgs[0].(float64); ok {
				length = int(l)
			}
		}
		return mapper.StringType(length)
	case "text":
		return mapper.TextType()
	case "integer":
		return mapper.IntegerType()
	case "float":
		return mapper.FloatType()
	case "decimal":
		precision, scale := 10, 2
		if len(typeArgs) > 0 {
			if p, ok := typeArgs[0].(int); ok {
				precision = p
			} else if p, ok := typeArgs[0].(float64); ok {
				precision = int(p)
			}
		}
		if len(typeArgs) > 1 {
			if s, ok := typeArgs[1].(int); ok {
				scale = s
			} else if s, ok := typeArgs[1].(float64); ok {
				scale = int(s)
			}
		}
		return mapper.DecimalType(precision, scale)
	case "boolean":
		return mapper.BooleanType()
	case "date":
		return mapper.DateType()
	case "time":
		return mapper.TimeType()
	case "datetime":
		return mapper.DateTimeType()
	case "uuid":
		return mapper.UUIDType()
	case "json":
		return mapper.JSONType()
	case "base64":
		return mapper.Base64Type()
	default:
		// Fallback for custom types
		return strings.ToUpper(typeName)
	}
}

// BooleanLiterals holds the true/false literals for a dialect.
type BooleanLiterals struct {
	True  string
	False string
}

// PostgresBooleans uses TRUE/FALSE.
var PostgresBooleans = BooleanLiterals{True: "TRUE", False: "FALSE"}

// SQLiteBooleans uses 1/0.
var SQLiteBooleans = BooleanLiterals{True: "1", False: "0"}

// buildDefaultValueSQL generates the SQL representation of a default value.
// This is shared logic - only boolean handling differs between dialects.
func buildDefaultValueSQL(value any, bools BooleanLiterals) string {
	switch v := value.(type) {
	case *ast.SQLExpr:
		return v.Expr
	case string:
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case bool:
		if v {
			return bools.True
		}
		return bools.False
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// buildForeignKeyConstraintSQL generates a foreign key constraint clause.
// This is shared logic used by all dialects.
func buildForeignKeyConstraintSQL(fk *ast.ForeignKeyDef, quoteIdent QuoteIdentFunc) string {
	var b strings.Builder

	if fk.Name != "" {
		b.WriteString("CONSTRAINT ")
		b.WriteString(quoteIdent(fk.Name))
		b.WriteString(" ")
	}

	b.WriteString("FOREIGN KEY (")
	writeQuotedList(&b, fk.Columns, quoteIdent)
	b.WriteString(") REFERENCES ")
	b.WriteString(quoteIdent(fk.RefTable))
	b.WriteString(" (")
	writeQuotedList(&b, fk.RefColumns, quoteIdent)
	b.WriteString(")")

	if fk.OnDelete != "" {
		b.WriteString(" ON DELETE ")
		b.WriteString(fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		b.WriteString(" ON UPDATE ")
		b.WriteString(fk.OnUpdate)
	}

	return b.String()
}

// ColumnDefFunc generates SQL for a column definition.
// The tableName parameter is the SQL table name (namespace_table format) for constraint naming.
type ColumnDefFunc func(col *ast.ColumnDef, tableName string) string

// ForeignKeyFunc generates SQL for a foreign key constraint.
type ForeignKeyFunc func(*ast.ForeignKeyDef) string

// buildCreateTableSQL generates CREATE TABLE SQL using provided helper functions.
func buildCreateTableSQL(op *ast.CreateTable, quoteIdent QuoteIdentFunc, columnDef ColumnDefFunc, fkConstraint ForeignKeyFunc) (string, error) {
	var b strings.Builder
	tableName := op.Table()

	b.WriteString("CREATE TABLE ")
	if op.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}
	b.WriteString(quoteIdent(tableName))
	b.WriteString(" (\n")

	first := true
	for _, col := range op.Columns {
		// Skip app-only virtual columns (no DB column)
		if col.Virtual && col.Computed == nil {
			continue
		}
		if !first {
			b.WriteString(",\n")
		}
		first = false
		b.WriteString("  ")
		b.WriteString(columnDef(col, tableName))
	}

	for _, fk := range op.ForeignKeys {
		b.WriteString(",\n  ")
		b.WriteString(fkConstraint(fk))
	}

	b.WriteString("\n)")
	return b.String(), nil
}

// buildDropTableSQL generates DROP TABLE SQL.
func buildDropTableSQL(op *ast.DropTable, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder
	b.WriteString("DROP TABLE ")
	if op.IfExists {
		b.WriteString("IF EXISTS ")
	}
	b.WriteString(quoteIdent(op.Table()))
	return b.String(), nil
}

// buildAddColumnSQL generates ALTER TABLE ADD COLUMN SQL.
func buildAddColumnSQL(op *ast.AddColumn, quoteIdent QuoteIdentFunc, columnDef ColumnDefFunc) (string, error) {
	var b strings.Builder
	tableName := op.Table()
	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(tableName))
	b.WriteString(" ADD COLUMN ")
	b.WriteString(columnDef(op.Column, tableName))
	return b.String(), nil
}

// buildDropColumnSQL generates ALTER TABLE DROP COLUMN SQL.
func buildDropColumnSQL(op *ast.DropColumn, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(op.Table()))
	b.WriteString(" DROP COLUMN ")
	b.WriteString(quoteIdent(op.Name))
	return b.String(), nil
}

// buildAddForeignKeySQL generates ALTER TABLE ADD FOREIGN KEY SQL.
func buildAddForeignKeySQL(op *ast.AddForeignKey, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder

	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(op.Table()))
	b.WriteString(" ADD ")

	if op.Name != "" {
		b.WriteString("CONSTRAINT ")
		b.WriteString(quoteIdent(op.Name))
		b.WriteString(" ")
	}

	b.WriteString("FOREIGN KEY (")
	writeQuotedList(&b, op.Columns, quoteIdent)
	b.WriteString(") REFERENCES ")
	b.WriteString(quoteIdent(op.RefTable))
	b.WriteString(" (")
	writeQuotedList(&b, op.RefColumns, quoteIdent)
	b.WriteString(")")

	if op.OnDelete != "" {
		b.WriteString(" ON DELETE ")
		b.WriteString(op.OnDelete)
	}
	if op.OnUpdate != "" {
		b.WriteString(" ON UPDATE ")
		b.WriteString(op.OnUpdate)
	}

	return b.String(), nil
}

// ColumnDefOrder configures the clause ordering for column definitions.
type ColumnDefOrder struct {
	// PrimaryKeyBeforeDefault: PostgreSQL/SQLite place PK before DEFAULT
	PrimaryKeyBeforeDefault bool
}

// PostgresColumnOrder is the standard PostgreSQL/SQLite ordering.
var PostgresColumnOrder = ColumnDefOrder{PrimaryKeyBeforeDefault: true}

// ColumnDefConfig holds all callbacks and config for buildColumnDefSQL.
type ColumnDefConfig struct {
	QuoteIdent QuoteIdentFunc
	TypeSQL    func(typeName string, typeArgs []any) string
	DefaultSQL func(value any) string
	Order      ColumnDefOrder
	// TableName is the SQL table name (namespace_table format) for constraint naming.
	TableName string
	// EnumCheck is called for SQLite to add CHECK constraint for enums.
	// Returns empty string if not needed.
	EnumCheck func(col *ast.ColumnDef, tableName string) string
	// DialectName is the dialect name ("postgres" or "sqlite") for computed column rendering.
	DialectName string
}

// buildColumnDefSQL generates the SQL for a column definition with configurable ordering.
// This consolidates the nearly identical implementations across dialects.
func buildColumnDefSQL(col *ast.ColumnDef, cfg ColumnDefConfig) string {
	var b strings.Builder

	// Column name and type
	b.WriteString(cfg.QuoteIdent(col.Name))
	b.WriteString(" ")
	b.WriteString(cfg.TypeSQL(col.Type, col.TypeArgs))

	// PostgreSQL/SQLite order: PK, NULL, UNIQUE, DEFAULT, REFERENCES
	if col.PrimaryKey {
		b.WriteString(" PRIMARY KEY")
	}
	writeNullability(&b, col)
	if col.Unique && !col.PrimaryKey {
		// Generate named UNIQUE constraint: CONSTRAINT uniq_table_column UNIQUE
		constraintName := uniqueConstraintName(cfg.TableName, col.Name)
		b.WriteString(" CONSTRAINT ")
		b.WriteString(cfg.QuoteIdent(constraintName))
		b.WriteString(" UNIQUE")
	}
	writeDefault(&b, col, cfg.DefaultSQL)

	// Enum CHECK constraint (SQLite only)
	if cfg.EnumCheck != nil {
		if check := cfg.EnumCheck(col, cfg.TableName); check != "" {
			b.WriteString(check)
		}
	}

	// App-only virtual columns produce no SQL
	if col.Virtual && col.Computed == nil {
		return ""
	}

	// Computed column (GENERATED ALWAYS AS)
	if col.Computed != nil {
		if expr := computedExprSQL(col.Computed, cfg.DialectName, cfg.QuoteIdent); expr != "" {
			b.WriteString(" GENERATED ALWAYS AS (")
			b.WriteString(expr)
			if col.Virtual {
				b.WriteString(") VIRTUAL")
			} else {
				b.WriteString(") STORED")
			}
		}
	}

	// References (inline FK) - same for all dialects
	if col.Reference != nil {
		b.WriteString(" REFERENCES ")
		refTable := strings.ReplaceAll(col.Reference.Table, ".", "_")
		b.WriteString(cfg.QuoteIdent(refTable))
		b.WriteString("(")
		refCol := col.Reference.Column
		if refCol == "" {
			refCol = "id"
		}
		b.WriteString(cfg.QuoteIdent(refCol))
		b.WriteString(")")

		if col.Reference.OnDelete != "" {
			b.WriteString(" ON DELETE ")
			b.WriteString(col.Reference.OnDelete)
		}
		if col.Reference.OnUpdate != "" {
			b.WriteString(" ON UPDATE ")
			b.WriteString(col.Reference.OnUpdate)
		}
	}

	return b.String()
}

// writeNullability writes the NULL/NOT NULL clause.
func writeNullability(b *strings.Builder, col *ast.ColumnDef) {
	if !col.Nullable && !col.PrimaryKey {
		b.WriteString(" NOT NULL")
	} else if col.Nullable && col.NullableSet {
		b.WriteString(" NULL")
	}
}

// writeDefault writes the DEFAULT clause if set.
func writeDefault(b *strings.Builder, col *ast.ColumnDef, defaultSQL func(any) string) {
	// Handle Default field (from migrations/schema files)
	if col.DefaultSet && col.Default != nil {
		b.WriteString(" DEFAULT ")
		b.WriteString(defaultSQL(col.Default))
		return
	}

	// Handle ServerDefault field (from introspection/normalized schemas)
	if col.ServerDefault != "" {
		b.WriteString(" DEFAULT ")
		b.WriteString(col.ServerDefault)
	}
}

// buildRenameColumnSQL generates ALTER TABLE RENAME COLUMN SQL.
// This is identical across PostgreSQL and SQLite 3.25.0+.
func buildRenameColumnSQL(op *ast.RenameColumn, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder

	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(op.Table()))
	b.WriteString(" RENAME COLUMN ")
	b.WriteString(quoteIdent(op.OldName))
	b.WriteString(" TO ")
	b.WriteString(quoteIdent(op.NewName))

	return b.String(), nil
}

// buildRenameTableSQL generates ALTER TABLE RENAME TO SQL for PostgreSQL and SQLite.
func buildRenameTableSQL(op *ast.RenameTable, quoteIdent QuoteIdentFunc, qualifyTable func(ns, name string) string) (string, error) {
	var b strings.Builder

	oldTable := qualifyTable(op.Namespace, op.OldName)
	newTable := qualifyTable(op.Namespace, op.NewName)

	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(oldTable))
	b.WriteString(" RENAME TO ")
	b.WriteString(quoteIdent(newTable))

	return b.String(), nil
}

// buildDropIndexSQL generates DROP INDEX SQL for PostgreSQL and SQLite.
func buildDropIndexSQL(op *ast.DropIndex, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder

	b.WriteString("DROP INDEX ")
	if op.IfExists {
		b.WriteString("IF EXISTS ")
	}
	b.WriteString(quoteIdent(op.Name))

	return b.String(), nil
}

// IndexSQLOpts configures dialect-specific index SQL generation.
type IndexSQLOpts struct {
	// SupportsIfNotExists is true for PostgreSQL and SQLite.
	SupportsIfNotExists bool
	// IndexNameFunc generates the index name. If nil, uses default naming.
	IndexNameFunc func(table string, unique bool, cols ...string) string
}

// indexNameFunc is the default index name generator using strutil conventions.
// This is shared across all dialects for consistent naming.
func indexNameFunc(table string, unique bool, cols ...string) string {
	if unique {
		return "uniq_" + table + "_" + joinCols(cols)
	}
	return "idx_" + table + "_" + joinCols(cols)
}

// joinCols joins column names with underscores.
func joinCols(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	result := cols[0]
	for i := 1; i < len(cols); i++ {
		result += "_" + cols[i]
	}
	return result
}

// uniqueConstraintName generates a unique constraint name: uniq_table_col1_col2...
func uniqueConstraintName(table string, cols ...string) string {
	result := "uniq_" + table
	for _, col := range cols {
		result += "_" + col
	}
	return result
}

// checkConstraintName generates a check constraint name: chk_table_name
func checkConstraintName(table string, name string) string {
	return "chk_" + table + "_" + name
}

// buildAddCheckSQL generates ALTER TABLE ADD CONSTRAINT CHECK SQL.
func buildAddCheckSQL(op *ast.AddCheck, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder

	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(op.Table()))
	b.WriteString(" ADD CONSTRAINT ")
	b.WriteString(quoteIdent(op.Name))
	b.WriteString(" CHECK (")
	b.WriteString(op.Expression)
	b.WriteString(")")

	return b.String(), nil
}

// buildDropCheckSQL generates ALTER TABLE DROP CONSTRAINT SQL for CHECK constraints.
func buildDropCheckSQL(op *ast.DropCheck, quoteIdent QuoteIdentFunc) (string, error) {
	var b strings.Builder

	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdent(op.Table()))
	b.WriteString(" DROP CONSTRAINT ")
	b.WriteString(quoteIdent(op.Name))

	return b.String(), nil
}

// computedExprSQL renders a computed column expression (map[string]any) to SQL.
// The expression is the serialized form of FnExpr from the runtime.
func computedExprSQL(computed any, dialectName string, quoteIdent QuoteIdentFunc) string {
	m, ok := computed.(map[string]any)
	if !ok {
		return ""
	}

	// Column reference: {"col": "name"}
	if col, ok := m["col"].(string); ok && col != "" {
		return quoteIdent(col)
	}

	fn, _ := m["fn"].(string)
	if fn == "" {
		return ""
	}

	// Raw SQL escape hatch: {"fn": "raw", "sql": {"postgres": "...", "sqlite": "..."}}
	if fn == "raw" {
		if sqlMap, ok := m["sql"].(map[string]any); ok {
			if s, ok := sqlMap[dialectName].(string); ok {
				return s
			}
		}
		return ""
	}

	args := exprArgs(m, dialectName, quoteIdent)

	switch fn {
	// String functions
	case "concat":
		// Use || operator for all dialects — CONCAT() is not immutable in PostgreSQL
		// and cannot be used in GENERATED ALWAYS AS expressions.
		return joinArgs(args, " || ")
	case "upper":
		return "UPPER(" + firstArg(args) + ")"
	case "lower":
		return "LOWER(" + firstArg(args) + ")"
	case "trim":
		return "TRIM(" + firstArg(args) + ")"
	case "length":
		return "LENGTH(" + firstArg(args) + ")"
	case "substring":
		if len(args) >= 3 {
			return "SUBSTRING(" + args[0] + " FROM " + args[1] + " FOR " + args[2] + ")"
		}
		return "SUBSTRING(" + strings.Join(args, ", ") + ")"

	// Math functions (binary operators)
	case "add":
		return binaryOp(args, "+")
	case "sub":
		return binaryOp(args, "-")
	case "mul":
		return binaryOp(args, "*")
	case "div":
		return binaryOp(args, "/")
	case "abs":
		return "ABS(" + firstArg(args) + ")"
	case "round":
		return "ROUND(" + firstArg(args) + ")"
	case "floor":
		return "FLOOR(" + firstArg(args) + ")"
	case "ceil":
		return "CEIL(" + firstArg(args) + ")"

	// Date functions
	case "year":
		if dialectName == "sqlite" {
			return "CAST(STRFTIME('%Y', " + firstArg(args) + ") AS INTEGER)"
		}
		return "EXTRACT(YEAR FROM " + firstArg(args) + ")"
	case "month":
		if dialectName == "sqlite" {
			return "CAST(STRFTIME('%m', " + firstArg(args) + ") AS INTEGER)"
		}
		return "EXTRACT(MONTH FROM " + firstArg(args) + ")"
	case "day":
		if dialectName == "sqlite" {
			return "CAST(STRFTIME('%d', " + firstArg(args) + ") AS INTEGER)"
		}
		return "EXTRACT(DAY FROM " + firstArg(args) + ")"
	case "now":
		if dialectName == "sqlite" {
			return "CURRENT_TIMESTAMP"
		}
		return "NOW()"
	case "years_since":
		if dialectName == "sqlite" {
			return "CAST((JULIANDAY('now') - JULIANDAY(" + firstArg(args) + ")) / 365.25 AS INTEGER)"
		}
		return "EXTRACT(YEAR FROM AGE(NOW(), " + firstArg(args) + "))"
	case "days_since":
		if dialectName == "sqlite" {
			return "CAST(JULIANDAY('now') - JULIANDAY(" + firstArg(args) + ") AS INTEGER)"
		}
		return "EXTRACT(DAY FROM (NOW() - " + firstArg(args) + "))"

	// Conditional functions
	case "coalesce":
		return "COALESCE(" + strings.Join(args, ", ") + ")"
	case "nullif":
		return "NULLIF(" + strings.Join(args, ", ") + ")"
	case "if_null":
		if dialectName == "sqlite" {
			return "IFNULL(" + strings.Join(args, ", ") + ")"
		}
		return "COALESCE(" + strings.Join(args, ", ") + ")"
	case "if_then":
		if len(args) >= 3 {
			return "CASE WHEN " + args[0] + " THEN " + args[1] + " ELSE " + args[2] + " END"
		}
		return ""

	// Comparison functions
	case "gt":
		return binaryOp(args, ">")
	case "gte":
		return binaryOp(args, ">=")
	case "lt":
		return binaryOp(args, "<")
	case "lte":
		return binaryOp(args, "<=")
	case "eq":
		return binaryOp(args, "=")

	default:
		// Unknown function — render as FUNCTION(args)
		return strings.ToUpper(fn) + "(" + strings.Join(args, ", ") + ")"
	}
}

// exprArgs extracts and renders the "args" array from a computed expression map.
func exprArgs(m map[string]any, dialectName string, quoteIdent QuoteIdentFunc) []string {
	rawArgs, ok := m["args"].([]any)
	if !ok {
		return nil
	}
	result := make([]string, len(rawArgs))
	for i, arg := range rawArgs {
		result[i] = renderArg(arg, dialectName, quoteIdent)
	}
	return result
}

// renderArg renders a single argument to SQL.
func renderArg(arg any, dialectName string, quoteIdent QuoteIdentFunc) string {
	switch v := arg.(type) {
	case map[string]any:
		// Nested expression
		return computedExprSQL(v, dialectName, quoteIdent)
	case string:
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// binaryOp renders two arguments with an operator between them.
func binaryOp(args []string, op string) string {
	if len(args) >= 2 {
		return "(" + args[0] + " " + op + " " + args[1] + ")"
	}
	return ""
}

// joinArgs joins arguments with a separator (for SQLite concat via ||).
func joinArgs(args []string, sep string) string {
	if len(args) == 0 {
		return ""
	}
	return "(" + strings.Join(args, sep) + ")"
}

// firstArg returns the first argument or empty string.
func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// buildCreateIndexSQL generates CREATE INDEX SQL with dialect-specific options.
// This consolidates the nearly identical implementations across all dialects.
func buildCreateIndexSQL(op *ast.CreateIndex, quoteIdent QuoteIdentFunc, opts IndexSQLOpts) (string, error) {
	var b strings.Builder

	b.WriteString("CREATE ")
	if op.Unique {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("INDEX ")

	if opts.SupportsIfNotExists && op.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}

	// Generate index name if not provided
	indexName := op.Name
	if indexName == "" {
		if opts.IndexNameFunc != nil {
			indexName = opts.IndexNameFunc(op.Table(), op.Unique, op.Columns...)
		} else {
			// Default: use strutil naming conventions
			if op.Unique {
				indexName = "uniq_" + op.Table()
				for _, col := range op.Columns {
					indexName += "_" + col
				}
			} else {
				indexName = "idx_" + op.Table()
				for _, col := range op.Columns {
					indexName += "_" + col
				}
			}
		}
	}

	b.WriteString(quoteIdent(indexName))
	b.WriteString(" ON ")
	b.WriteString(quoteIdent(op.Table()))
	b.WriteString(" (")
	writeQuotedList(&b, op.Columns, quoteIdent)
	b.WriteString(")")

	return b.String(), nil
}
