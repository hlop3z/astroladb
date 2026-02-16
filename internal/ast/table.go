package ast

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// Validation messages shared across TableDef, ColumnDef, IndexDef, ForeignKeyDef,
// and their corresponding Operation types (operation.go).
const (
	msgTableNameRequired  = "table name is required"
	msgColumnNameRequired = "column name is required"
	msgTableNeedsColumn   = "table must have at least one column"
	msgIndexNeedsColumn   = "index must have at least one column"
	msgFKNeedsColumn      = "foreign key must have at least one column"
	msgFKNeedsRefTable    = "foreign key must reference a table"
	msgFKNeedsRefColumn   = "foreign key must reference at least one column"
	msgFKColumnCountMatch = "foreign key column count must match referenced column count"
)

// validIdentifierPattern matches safe SQL identifiers (lowercase snake_case).
var validIdentifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// ValidateIdentifier checks that a name is a safe SQL identifier (lowercase snake_case).
func ValidateIdentifier(name string) error {
	if !validIdentifierPattern.MatchString(name) {
		return alerr.New(alerr.ErrSchemaInvalid,
			fmt.Sprintf("invalid identifier %q; must match [a-z_][a-z0-9_]*", name))
	}
	return nil
}

// ValidateQualifiedName checks a "namespace.table" or "table" reference.
func ValidateQualifiedName(name string) error {
	parts := strings.Split(name, ".")
	switch len(parts) {
	case 1:
		return ValidateIdentifier(parts[0])
	case 2:
		if err := ValidateIdentifier(parts[0]); err != nil {
			return err
		}
		return ValidateIdentifier(parts[1])
	default:
		return alerr.New(alerr.ErrSchemaInvalid,
			fmt.Sprintf("invalid qualified name %q; expected 'table' or 'namespace.table'", name))
	}
}

// ValidFKActions is the set of valid ON DELETE / ON UPDATE actions.
var ValidFKActions = map[string]bool{
	"":            true, // empty = no action specified (valid)
	"CASCADE":     true,
	"SET NULL":    true,
	"SET DEFAULT": true,
	"RESTRICT":    true,
	"NO ACTION":   true,
}

// ValidateFKAction checks if the given action is a valid FK referential action.
func ValidateFKAction(action string) error {
	upper := strings.ToUpper(strings.TrimSpace(action))
	if !ValidFKActions[upper] {
		return alerr.New(alerr.ErrSchemaInvalid,
			fmt.Sprintf("invalid foreign key action %q; must be one of: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION", action))
	}
	return nil
}

// NormalizeFKAction normalizes and validates an FK action string.
// Returns the uppercased action or error if invalid.
func NormalizeFKAction(action string) (string, error) {
	if action == "" {
		return "", nil
	}
	upper := strings.ToUpper(strings.TrimSpace(action))
	if !ValidFKActions[upper] {
		return "", alerr.New(alerr.ErrSchemaInvalid,
			fmt.Sprintf("invalid foreign key action %q; must be one of: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION", action))
	}
	return upper, nil
}

// dangerousSQLPattern matches SQL injection patterns in expressions.
// Blocks: semicolons, double-dashes (comments), DDL keywords at word boundaries.
var dangerousSQLPattern = regexp.MustCompile(
	`(?i)(;\s*|--|\b(DROP|ALTER|CREATE|GRANT|REVOKE|TRUNCATE|INSERT|UPDATE|DELETE|EXEC|EXECUTE|UNION|INTO|COPY|pg_read_file|lo_import|pg_sleep)\b)`,
)

// ValidateSQLExpression checks a raw SQL expression for dangerous patterns.
// Used for CHECK constraints, WHERE clauses, and ServerDefault values.
func ValidateSQLExpression(expr string) error {
	if expr == "" {
		return nil
	}
	if dangerousSQLPattern.MatchString(expr) {
		return alerr.New(alerr.ErrSchemaInvalid,
			"SQL expression contains potentially dangerous pattern").
			With("expression", expr).
			WithHelp("expressions must not contain ';', '--', or DDL/DML keywords (DROP, ALTER, CREATE, INSERT, UPDATE, DELETE, etc.)")
	}
	return nil
}

// validTypeNamePattern matches safe SQL type names.
var validTypeNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_ ()\[\],]*$`)

// ValidateTypeName checks that a custom type name is safe for SQL.
func ValidateTypeName(name string) error {
	if !validTypeNamePattern.MatchString(name) {
		return alerr.New(alerr.ErrSchemaInvalid,
			fmt.Sprintf("invalid type name %q; must match [a-zA-Z_][a-zA-Z0-9_ ()\\[\\],]*", name))
	}
	return nil
}

// -----------------------------------------------------------------------------
// TableDef - complete table definition
// -----------------------------------------------------------------------------

// TableDef represents a complete table definition with columns, indexes, and constraints.
// This is the result of evaluating a schema file or building a table in a migration.
type TableDef struct {
	Namespace   string           // Logical grouping (e.g., "auth", "billing")
	Name        string           // Table name (snake_case)
	Columns     []*ColumnDef     // Column definitions in order
	Indexes     []*IndexDef      // Index definitions
	ForeignKeys []*ForeignKeyDef // Foreign key constraints
	Checks      []*CheckDef      // CHECK constraints

	// Documentation (-> SQL COMMENT + OpenAPI)
	Docs       string // Description for the table
	Deprecated string // Deprecation notice (if table is deprecated)

	// Table-level metadata for x-db extensions
	Auditable  bool     // Has created_by/updated_by columns (adapters manage these)
	SortBy     []string // Default ordering (e.g., ["-created_at", "name"])
	Searchable []string // Columns for fulltext search
	Filterable []string // Columns allowed in WHERE clauses

	// Source location (for error reporting)
	SourceFile string // Path to the schema file that defined this table
}

// FullName returns the full table name in namespace_tablename format.
func (t *TableDef) FullName() string {
	return strutil.SQLName(t.Namespace, t.Name)
}

// QualifiedName returns the fully qualified reference (namespace.table).
func (t *TableDef) QualifiedName() string {
	return strutil.QualifiedName(t.Namespace, t.Name)
}

// MatchesReference returns true if ref matches any of the table's name forms:
// QualifiedName (namespace.name), FullName (namespace_name), or bare Name.
func (t *TableDef) MatchesReference(ref string) bool {
	return ref == t.QualifiedName() ||
		ref == t.FullName() ||
		(t.Namespace != "" && ref == t.Namespace+"_"+t.Name) ||
		ref == t.Name
}

// GetColumn returns the column with the given name, or nil if not found.
func (t *TableDef) GetColumn(name string) *ColumnDef {
	for _, col := range t.Columns {
		if col.Name == name {
			return col
		}
	}
	return nil
}

// HasColumn returns true if the table has a column with the given name.
func (t *TableDef) HasColumn(name string) bool {
	return t.GetColumn(name) != nil
}

// PrimaryKey returns the primary key column, or nil if none.
// Tables created with id() will have a single-column UUID primary key.
func (t *TableDef) PrimaryKey() *ColumnDef {
	for _, col := range t.Columns {
		if col.PrimaryKey {
			return col
		}
	}
	return nil
}

// checkDuplicateColumns returns an error if any column name appears more than once.
func (t *TableDef) checkDuplicateColumns() error {
	seen := make(map[string]bool)
	for _, col := range t.Columns {
		if seen[col.Name] {
			return alerr.New(alerr.ErrSchemaDuplicate, "duplicate column name").
				WithTable(t.Namespace, t.Name).
				WithColumn(col.Name)
		}
		seen[col.Name] = true
	}
	return nil
}

// ValidateBasic checks that the table has a name, at least one column,
// and no duplicate column names. It does NOT validate identifier syntax,
// column types, indexes, or foreign keys — use Validate() for that.
func (t *TableDef) ValidateBasic() error {
	if t.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, msgTableNameRequired)
	}
	if len(t.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, msgTableNeedsColumn).
			WithTable(t.Namespace, t.Name)
	}
	for _, col := range t.Columns {
		if col.Name == "" {
			return alerr.New(alerr.ErrSchemaInvalid, msgColumnNameRequired).
				WithTable(t.Namespace, t.Name)
		}
	}
	return t.checkDuplicateColumns()
}

// Validate checks that the table definition is well-formed.
func (t *TableDef) Validate() error {
	if t.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, msgTableNameRequired)
	}
	if err := ValidateIdentifier(t.Name); err != nil {
		return err
	}
	if t.Namespace != "" {
		if err := ValidateIdentifier(t.Namespace); err != nil {
			return err
		}
	}
	if len(t.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, msgTableNeedsColumn).
			WithTable(t.Namespace, t.Name)
	}
	// Check for duplicate column names
	if err := t.checkDuplicateColumns(); err != nil {
		return err
	}
	for _, col := range t.Columns {
		if err := col.Validate(); err != nil {
			return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid column").
				WithTable(t.Namespace, t.Name).
				WithColumn(col.Name)
		}
	}
	// Validate indexes
	for _, idx := range t.Indexes {
		if err := idx.Validate(); err != nil {
			return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid index").
				WithTable(t.Namespace, t.Name)
		}
	}
	// Validate foreign keys
	for _, fk := range t.ForeignKeys {
		if err := fk.Validate(); err != nil {
			return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid foreign key").
				WithTable(t.Namespace, t.Name)
		}
	}
	return nil
}

// SortColumnsForExport returns a new slice with columns sorted for deterministic
// export output: primary key first, then required fields alphabetically, then
// optional fields (nullable or has default) alphabetically.
func SortColumnsForExport(cols []*ColumnDef) []*ColumnDef {
	sorted := slices.Clone(cols)
	slices.SortStableFunc(sorted, func(a, b *ColumnDef) int {
		if c := cmp.Compare(colSortGroup(a), colSortGroup(b)); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	return sorted
}

// colSortGroup returns the sort group for a column:
// 0 = primary key, 1 = required, 2 = optional.
func colSortGroup(c *ColumnDef) int {
	if c.PrimaryKey {
		return 0
	}
	if c.Nullable || c.HasDefault() {
		return 2
	}
	return 1
}

// -----------------------------------------------------------------------------
// ColumnDef - complete column definition
// -----------------------------------------------------------------------------

// ColumnDef represents a complete column definition with type, constraints, and metadata.
type ColumnDef struct {
	Name     string // Column name (snake_case)
	Type     string // Type name (id, string, integer, etc.)
	TypeArgs []any  // Type arguments (e.g., length for string, precision/scale for decimal)

	// Source location (for error reporting)
	Line   int    // Line number in source file (0 if unknown)
	Column int    // Column number in source file (0 if unknown)
	Source string // The source code line (for error display)

	// Nullability
	Nullable    bool // True if column allows NULL (default is NOT NULL)
	NullableSet bool // True if Nullable was explicitly set (vs inferred default)

	// Constraints
	Unique     bool // UNIQUE constraint
	PrimaryKey bool // PRIMARY KEY constraint
	Index      bool // CREATE INDEX (non-unique) - generates concurrent index in Phase 2

	// Default values
	Default       any    // Default value for new rows (Go value or SQLExpr)
	DefaultSet    bool   // True if Default was explicitly set
	ServerDefault string // Raw SQL expression for server-side default (deprecated: use SQLExpr)

	// Reference (for belongs_to, one_to_one)
	Reference *Reference // Foreign key reference

	// Migration-only: backfill value for existing rows
	Backfill    any  // Value or SQLExpr for existing rows (used when adding NOT NULL column)
	BackfillSet bool // Track if backfill was explicitly set

	// Validation constraints (-> DB CHECK + OpenAPI)
	Min     *float64 // string: minLength, number: minimum
	Max     *float64 // string: maxLength, number: maximum
	Pattern string   // Regex pattern for validation
	Format  string   // OpenAPI format (email, uri, uuid, etc.)

	// Documentation (-> SQL COMMENT + OpenAPI)
	Docs       string // Description for the column
	Deprecated string // Deprecation notice (if column is deprecated)

	// Access control (-> OpenAPI readOnly/writeOnly)
	ReadOnly  bool // Column is read-only (computed, generated, server-set)
	WriteOnly bool // Column is write-only (passwords, secrets, tokens)

	// Computed columns (virtual, database-computed)
	Computed any  // fn.* expression or raw SQL map for computed columns
	Virtual  bool // If true + Computed: VIRTUAL instead of STORED; if true alone: app-only (no DB column)
}

// Validate checks that the column definition is well-formed.
func (c *ColumnDef) Validate() error {
	if c.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, msgColumnNameRequired)
	}
	if err := ValidateIdentifier(c.Name); err != nil {
		return err
	}
	if c.Type == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "column type is required").
			WithColumn(c.Name)
	}
	// Validate min/max constraints make sense
	if c.Min != nil && c.Max != nil && *c.Min > *c.Max {
		return alerr.New(alerr.ErrSchemaInvalid, "min cannot be greater than max").
			WithColumn(c.Name).
			With("min", *c.Min).
			With("max", *c.Max)
	}
	return nil
}

// IsNullable returns whether the column allows NULL values.
// If NullableSet is false, returns the default (false = NOT NULL).
func (c *ColumnDef) IsNullable() bool {
	return c.Nullable
}

// GetDefaultSQL returns the SQL representation of the default value.
// It handles both ServerDefault (from introspection) and Default (from DSL).
// Returns empty string if no default is set.
func (c *ColumnDef) GetDefaultSQL() string {
	// Introspected defaults take precedence (already in SQL form)
	if c.ServerDefault != "" {
		return c.ServerDefault
	}

	// DSL defaults need formatting
	if c.DefaultSet && c.Default != nil {
		return formatDefaultValue(c.Default)
	}

	return ""
}

// HasDefault returns true if any default value is set.
func (c *ColumnDef) HasDefault() bool {
	return c.ServerDefault != "" || (c.DefaultSet && c.Default != nil)
}

// formatDefaultValue converts Default field to SQL string.
func formatDefaultValue(val any) string {
	if sqlExpr, ok := val.(*SQLExpr); ok {
		return sqlExpr.Expr
	}

	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		// Unsupported types fall back to escaped string representation
		return fmt.Sprintf("'%s'", strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''"))
	}
}

// HasBackfill returns whether the column has a backfill value for existing rows.
func (c *ColumnDef) HasBackfill() bool {
	return c.BackfillSet || c.Backfill != nil
}

// EnumValues extracts enum string values from TypeArgs.
// Handles both []string and []any at TypeArgs[0] (primary) and TypeArgs[1] (legacy).
func (c *ColumnDef) EnumValues() []string {
	if len(c.TypeArgs) == 0 {
		return nil
	}
	// TypeArgs[0] is the enum values slice (from col.id() API)
	if values, ok := c.TypeArgs[0].([]string); ok {
		return values
	}
	if values, ok := c.TypeArgs[0].([]any); ok {
		var result []string
		for _, v := range values {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	// Legacy format: TypeArgs[1] has enum values
	if len(c.TypeArgs) > 1 {
		if values, ok := c.TypeArgs[1].([]string); ok {
			return values
		}
		if values, ok := c.TypeArgs[1].([]any); ok {
			var result []string
			for _, v := range values {
				if s, ok := v.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// SQLExpr - marks raw SQL expressions
// -----------------------------------------------------------------------------

// SQLExpr marks a string as a raw SQL expression.
// Used with the sql() helper in JS to indicate a value should be passed through
// to the database without escaping.
//
// Examples:
//   - sql("NOW()") -> current timestamp
//   - sql("gen_random_uuid()") -> generate UUID
//   - sql("CURRENT_USER") -> current database user
type SQLExpr struct {
	Expr string `json:"expr"`
}

// ConvertSQLExprValue converts JS values (like sql("...") results) to Go types.
// If v is a map[string]any with {_type: "sql_expr", expr: "..."}, returns *SQLExpr.
// Otherwise returns v unchanged.
func ConvertSQLExprValue(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	if typ, ok := m["_type"].(string); ok && typ == "sql_expr" {
		if expr, ok := m["expr"].(string); ok {
			return &SQLExpr{Expr: expr}
		}
	}
	return v
}

// -----------------------------------------------------------------------------
// IndexDef - index definition
// -----------------------------------------------------------------------------

// IndexDef represents an index definition.
type IndexDef struct {
	Name    string   // Index name (auto-generated if empty)
	Columns []string // Columns to index (in order)
	Unique  bool     // UNIQUE index
	Where   string   // Partial index condition (if supported by dialect)
}

// Validate checks that the index definition is well-formed.
func (i *IndexDef) Validate() error {
	if len(i.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, msgIndexNeedsColumn)
	}
	for _, col := range i.Columns {
		if err := ValidateIdentifier(col); err != nil {
			return err
		}
	}
	if i.Where != "" {
		if err := ValidateSQLExpression(i.Where); err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// ForeignKeyDef - foreign key constraint definition
// -----------------------------------------------------------------------------

// ForeignKeyDef represents a foreign key constraint.
type ForeignKeyDef struct {
	Name       string   // Constraint name (auto-generated if empty)
	Columns    []string // Local columns
	RefTable   string   // Referenced table (namespace_tablename format)
	RefColumns []string // Referenced columns (usually just "id")
	OnDelete   string   // CASCADE, SET NULL, RESTRICT, NO ACTION (default: NO ACTION)
	OnUpdate   string   // CASCADE, SET NULL, RESTRICT, NO ACTION (default: NO ACTION)
}

// Validate checks that the foreign key definition is well-formed.
func (fk *ForeignKeyDef) Validate() error {
	if len(fk.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, msgFKNeedsColumn)
	}
	for _, col := range fk.Columns {
		if err := ValidateIdentifier(col); err != nil {
			return err
		}
	}
	if fk.RefTable == "" {
		return alerr.New(alerr.ErrSchemaInvalid, msgFKNeedsRefTable)
	}
	if len(fk.RefColumns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, msgFKNeedsRefColumn)
	}
	if err := ValidateIdentifier(fk.RefTable); err != nil {
		return err
	}
	for _, col := range fk.RefColumns {
		if err := ValidateIdentifier(col); err != nil {
			return err
		}
	}
	if len(fk.Columns) != len(fk.RefColumns) {
		return alerr.New(alerr.ErrSchemaInvalid, msgFKColumnCountMatch).
			With("columns", len(fk.Columns)).
			With("ref_columns", len(fk.RefColumns))
	}
	if err := ValidateFKAction(fk.OnDelete); err != nil {
		return err
	}
	if err := ValidateFKAction(fk.OnUpdate); err != nil {
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------
// CheckDef - CHECK constraint definition
// -----------------------------------------------------------------------------

// CheckDef represents a CHECK constraint.
type CheckDef struct {
	Name       string // Constraint name (auto-generated if empty)
	Expression string // SQL expression for the check
}

// Validate checks that the check constraint is well-formed.
func (c *CheckDef) Validate() error {
	if c.Name != "" {
		if err := ValidateIdentifier(c.Name); err != nil {
			return err
		}
	}
	if c.Expression == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "check constraint requires an expression")
	}
	if err := ValidateSQLExpression(c.Expression); err != nil {
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------
// Reference - column reference (for belongs_to, one_to_one)
// -----------------------------------------------------------------------------

// Reference represents a foreign key reference from a column.
// This is the structured representation of belongs_to() or one_to_one().
type Reference struct {
	Table    string // Referenced table (can be "ns.table", ".table", or "table")
	Column   string // Referenced column (default: "id")
	OnDelete string // CASCADE, SET NULL, RESTRICT, NO ACTION
	OnUpdate string // CASCADE, SET NULL, RESTRICT, NO ACTION
}

// TargetColumn returns the referenced column, defaulting to "id".
func (r *Reference) TargetColumn() string {
	if r.Column != "" {
		return r.Column
	}
	return "id"
}

// Validate checks that the reference is well-formed.
func (r *Reference) Validate() error {
	if r.Table == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "reference must specify a table")
	}
	// Reference.Table can be "ns.table", ".table", or "table"
	refTable := strings.TrimPrefix(r.Table, ".")
	if err := ValidateQualifiedName(refTable); err != nil {
		return err
	}
	if err := ValidateFKAction(r.OnDelete); err != nil {
		return err
	}
	if err := ValidateFKAction(r.OnUpdate); err != nil {
		return err
	}
	return nil
}
