package dsl

import (
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// TableBuilder provides a fluent API for building table definitions.
// It includes convenience methods for common patterns and relationships.
type TableBuilder struct {
	def       *ast.TableDef
	namespace string
}

// NewTableBuilder creates a new TableBuilder for the given namespace and table name.
func NewTableBuilder(namespace, name string) *TableBuilder {
	return &TableBuilder{
		namespace: namespace,
		def: &ast.TableDef{
			Namespace: namespace,
			Name:      name,
			Columns:   make([]*ast.ColumnDef, 0),
			Indexes:   make([]*ast.IndexDef, 0),
			Checks:    make([]*ast.CheckDef, 0),
		},
	}
}

// Build returns the final table definition.
func (t *TableBuilder) Build() *ast.TableDef {
	return t.def
}

// AddColumn adds a column definition to the table.
func (t *TableBuilder) AddColumn(col *ast.ColumnDef) {
	t.def.Columns = append(t.def.Columns, col)
}

// AddIndex adds an index definition to the table.
func (t *TableBuilder) AddIndex(idx *ast.IndexDef) {
	t.def.Indexes = append(t.def.Indexes, idx)
}

// -----------------------------------------------------------------------------
// Primary Key
// -----------------------------------------------------------------------------

// ID adds a UUID primary key column named "id".
// This is the only way to create a primary key (opinionated).
func (t *TableBuilder) ID() *ColumnBuilder {
	col := NewColumnBuilder("id", "uuid")
	col.PrimaryKey()
	col.Default(&ast.SQLExpr{Expr: "gen_random_uuid()"})
	t.AddColumn(col.Build())
	return col
}

// -----------------------------------------------------------------------------
// Core Type Methods
// -----------------------------------------------------------------------------

// String adds a VARCHAR column with a required length.
func (t *TableBuilder) String(name string, length int) *ColumnBuilder {
	col := NewColumnBuilder(name, "string", length)
	t.AddColumn(col.Build())
	return col
}

// Text adds a TEXT column (unlimited length).
func (t *TableBuilder) Text(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "text")
	t.AddColumn(col.Build())
	return col
}

// Integer adds a 32-bit signed integer column (JS-safe).
func (t *TableBuilder) Integer(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "integer")
	t.AddColumn(col.Build())
	return col
}

// Float adds a 32-bit floating-point column.
func (t *TableBuilder) Float(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "float")
	t.AddColumn(col.Build())
	return col
}

// Decimal adds an arbitrary-precision decimal column.
// Use for money and financial data.
func (t *TableBuilder) Decimal(name string, precision, scale int) *ColumnBuilder {
	col := NewColumnBuilder(name, "decimal", precision, scale)
	t.AddColumn(col.Build())
	return col
}

// Boolean adds a boolean column.
func (t *TableBuilder) Boolean(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "boolean")
	t.AddColumn(col.Build())
	return col
}

// Date adds a date-only column (YYYY-MM-DD).
func (t *TableBuilder) Date(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "date")
	t.AddColumn(col.Build())
	return col
}

// Time adds a time-only column (HH:MM:SS).
func (t *TableBuilder) Time(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "time")
	t.AddColumn(col.Build())
	return col
}

// DateTime adds a timestamp with timezone column.
func (t *TableBuilder) DateTime(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "date_time")
	t.AddColumn(col.Build())
	return col
}

// UUID adds a UUID column.
func (t *TableBuilder) UUID(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "uuid")
	t.AddColumn(col.Build())
	return col
}

// JSON adds a JSON column.
func (t *TableBuilder) JSON(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "json")
	t.AddColumn(col.Build())
	return col
}

// Base64 adds a binary/base64 column.
func (t *TableBuilder) Base64(name string) *ColumnBuilder {
	col := NewColumnBuilder(name, "base64")
	t.AddColumn(col.Build())
	return col
}

// Enum adds an enumerated type column.
func (t *TableBuilder) Enum(name string, values []string) *ColumnBuilder {
	col := NewColumnBuilder(name, "enum", values)
	t.AddColumn(col.Build())
	return col
}

// -----------------------------------------------------------------------------
// Convenience Patterns
// -----------------------------------------------------------------------------

// Timestamps adds created_at and updated_at columns.
func (t *TableBuilder) Timestamps() {
	createdAt := NewColumnBuilder("created_at", "date_time")
	createdAt.Default(&ast.SQLExpr{Expr: "NOW()"})
	t.AddColumn(createdAt.Build())

	updatedAt := NewColumnBuilder("updated_at", "date_time")
	updatedAt.Default(&ast.SQLExpr{Expr: "NOW()"})
	t.AddColumn(updatedAt.Build())
}

// SoftDelete adds a deleted_at column for soft deletion.
func (t *TableBuilder) SoftDelete() {
	deletedAt := NewColumnBuilder("deleted_at", "date_time")
	deletedAt.Nullable()
	t.AddColumn(deletedAt.Build())
}

// Sortable adds a position column for manual ordering.
func (t *TableBuilder) Sortable() {
	position := NewColumnBuilder("position", "integer")
	position.Default(0)
	t.AddColumn(position.Build())
}

// Slugged adds a unique slug column based on a source column.
func (t *TableBuilder) Slugged(sourceColumn string) {
	// Get max length from source column if it exists
	maxLen := 200 // default
	for _, col := range t.def.Columns {
		if col.Name == sourceColumn && len(col.TypeArgs) > 0 {
			if l, ok := col.TypeArgs[0].(int); ok {
				maxLen = l
			}
		}
	}

	slug := NewColumnBuilder("slug", "string", maxLen)
	slug.Unique()
	t.AddColumn(slug.Build())
}

// -----------------------------------------------------------------------------
// Relationships
// -----------------------------------------------------------------------------

// BelongsTo adds a foreign key column to another table.
// Creates a NOT NULL column by default (use opts to make nullable).
func (t *TableBuilder) BelongsTo(ref string, opts ...BelongsToOption) *ColumnBuilder {
	cfg := &belongsToConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Determine column name
	colName := cfg.as
	if colName == "" {
		// Extract table name from reference and create FK column name
		_, table, _ := parseRefParts(ref)
		colName = strutil.FKColumn(table)
	}

	col := NewColumnBuilder(colName, "uuid")
	col.References(ref)

	// Apply cascade options
	if cfg.onDelete != "" {
		col.def.Reference.OnDelete = cfg.onDelete
	}
	if cfg.onUpdate != "" {
		col.def.Reference.OnUpdate = cfg.onUpdate
	}

	// Nullable only if explicitly requested
	if cfg.nullable {
		col.Nullable()
	}

	t.AddColumn(col.Build())

	// Auto-create index on foreign key
	if !cfg.noIndex {
		t.Index(colName)
	}

	return col
}

// belongsToConfig holds configuration for BelongsTo relationships.
type belongsToConfig struct {
	as       string // custom column name
	nullable bool   // allow null
	onDelete string // ON DELETE action
	onUpdate string // ON UPDATE action
	noIndex  bool   // skip auto-index
}

// BelongsToOption is a functional option for BelongsTo.
type BelongsToOption func(*belongsToConfig)

// As sets a custom column name for the foreign key.
func As(name string) BelongsToOption {
	return func(c *belongsToConfig) {
		c.as = name + "_id"
	}
}

// BelongsToNullable makes the foreign key nullable.
func BelongsToNullable() BelongsToOption {
	return func(c *belongsToConfig) {
		c.nullable = true
	}
}

// BelongsToOnDelete sets the ON DELETE action.
func BelongsToOnDelete(action string) BelongsToOption {
	return func(c *belongsToConfig) {
		c.onDelete = action
	}
}

// BelongsToOnUpdate sets the ON UPDATE action.
func BelongsToOnUpdate(action string) BelongsToOption {
	return func(c *belongsToConfig) {
		c.onUpdate = action
	}
}

// BelongsToNoIndex disables auto-indexing of the foreign key.
func BelongsToNoIndex() BelongsToOption {
	return func(c *belongsToConfig) {
		c.noIndex = true
	}
}

// OneToOne adds a unique foreign key column (one-to-one relationship).
func (t *TableBuilder) OneToOne(ref string, opts ...BelongsToOption) *ColumnBuilder {
	col := t.BelongsTo(ref, opts...)
	col.Unique()
	return col
}

// ManyToMany declares a many-to-many relationship.
// This generates a join table automatically.
// Note: The actual join table is created by the schema processor, not here.
type ManyToManyDef struct {
	OtherTable string
	Through    string // Custom join table name
}

// ManyToMany adds a many-to-many relationship.
// Returns a ManyToManyDef that can be used by the schema processor.
func (t *TableBuilder) ManyToMany(ref string, opts ...M2MOption) *ManyToManyDef {
	cfg := &ManyToManyDef{
		OtherTable: ref,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	// Note: The join table is created by the schema processor
	return cfg
}

// M2MOption is a functional option for ManyToMany.
type M2MOption func(*ManyToManyDef)

// Through sets a custom join table name.
func Through(tableName string) M2MOption {
	return func(m *ManyToManyDef) {
		m.Through = tableName
	}
}

// BelongsToAny adds a polymorphic reference (type + id columns).
func (t *TableBuilder) BelongsToAny(refs []string, opts ...BelongsToAnyOption) {
	cfg := &belongsToAnyConfig{
		name: "parent", // default
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Add type column (e.g., parent_type)
	typeCol := NewColumnBuilder(cfg.name+"_type", "string", 100)
	t.AddColumn(typeCol.Build())

	// Add id column (e.g., parent_id)
	idCol := NewColumnBuilder(cfg.name+"_id", "uuid")
	t.AddColumn(idCol.Build())

	// Add composite index
	t.Index(cfg.name+"_type", cfg.name+"_id")
}

// belongsToAnyConfig holds configuration for polymorphic relationships.
type belongsToAnyConfig struct {
	name string // base name for type/id columns
}

// BelongsToAnyOption is a functional option for BelongsToAny.
type BelongsToAnyOption func(*belongsToAnyConfig)

// PolymorphicAs sets the base name for polymorphic columns.
func PolymorphicAs(name string) BelongsToAnyOption {
	return func(c *belongsToAnyConfig) {
		c.name = name
	}
}

// -----------------------------------------------------------------------------
// Indexes
// -----------------------------------------------------------------------------

// Index adds a non-unique index on the given columns.
func (t *TableBuilder) Index(columns ...string) {
	idx := &ast.IndexDef{
		Columns: columns,
		Unique:  false,
	}
	t.AddIndex(idx)
}

// Unique adds a unique constraint/index on the given columns.
// This is the preferred way to add uniqueness constraints.
func (t *TableBuilder) Unique(columns ...string) {
	idx := &ast.IndexDef{
		Columns: columns,
		Unique:  true,
	}
	t.AddIndex(idx)
}

// -----------------------------------------------------------------------------
// Documentation
// -----------------------------------------------------------------------------

// Docs sets the documentation for the table.
func (t *TableBuilder) Docs(description string) {
	t.def.Docs = description
}

// Deprecated marks the table as deprecated.
func (t *TableBuilder) Deprecated(reason string) {
	t.def.Deprecated = reason
}

// -----------------------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------------------

// parseRefParts extracts namespace and table from a reference.
// Simple implementation that mirrors registry.ParseReference.
func parseRefParts(ref string) (ns, table string, isRelative bool) {
	if ref == "" {
		return "", "", false
	}
	if len(ref) > 0 && ref[0] == '.' {
		return "", ref[1:], true
	}
	for i, c := range ref {
		if c == '.' {
			return ref[:i], ref[i+1:], false
		}
	}
	return "", ref, false
}
