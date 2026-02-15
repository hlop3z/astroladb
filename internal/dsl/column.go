// Package dsl provides the builder API for defining database schemas and migrations.
// These builders are used by both schema files (declarative) and migration files (imperative).
package dsl

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// ColumnBuilder provides a fluent API for building column definitions.
// All methods return the builder for method chaining.
type ColumnBuilder struct {
	def *ast.ColumnDef
}

// NewColumnBuilder creates a new ColumnBuilder with the given name and type.
func NewColumnBuilder(name, typeName string, args ...any) *ColumnBuilder {
	return &ColumnBuilder{
		def: &ast.ColumnDef{
			Name:     name,
			Type:     typeName,
			TypeArgs: args,
			// NOT NULL by default (opinionated)
			Nullable:    false,
			NullableSet: false,
		},
	}
}

// Build returns the final column definition.
func (c *ColumnBuilder) Build() *ast.ColumnDef {
	return c.def
}

// Nullable marks the column as allowing NULL values.
// This opts-out of the default NOT NULL behavior.
func (c *ColumnBuilder) Nullable() *ColumnBuilder {
	c.def.Nullable = true
	c.def.NullableSet = true
	return c
}

// NotNull marks the column as NOT NULL (default behavior).
// This is the default and is provided for explicit clarity.
func (c *ColumnBuilder) NotNull() *ColumnBuilder {
	c.def.Nullable = false
	c.def.NullableSet = true
	return c
}

// Unique adds a unique constraint to the column.
// Unique columns are automatically indexed.
func (c *ColumnBuilder) Unique() *ColumnBuilder {
	c.def.Unique = true
	return c
}

// PrimaryKey marks the column as the primary key.
func (c *ColumnBuilder) PrimaryKey() *ColumnBuilder {
	c.def.PrimaryKey = true
	return c
}

// Default sets the default value for new inserts.
// The value is permanent and applies to all new rows.
func (c *ColumnBuilder) Default(value any) *ColumnBuilder {
	c.def.Default = value
	c.def.DefaultSet = true
	return c
}

// Backfill sets a one-time value for existing rows during migrations.
// Use this when adding a NOT NULL column to a table with existing data.
// For SQL expressions, use sql("expression") to wrap the value.
func (c *ColumnBuilder) Backfill(value any) *ColumnBuilder {
	c.def.Backfill = value
	c.def.BackfillSet = true
	return c
}

// References sets a foreign key reference to another table.
// The ref can be in any supported format: "ns.table", ".table", or "table".
func (c *ColumnBuilder) References(ref string, opts ...RefOption) *ColumnBuilder {
	c.def.Reference = &ast.Reference{
		Table:  ref,
		Column: "id", // Default to id column
	}
	for _, opt := range opts {
		opt(c.def.Reference)
	}
	return c
}

// RefOption is a functional option for configuring a reference.
type RefOption func(*ast.Reference)

// RefColumn sets the referenced column (default is "id").
func RefColumn(col string) RefOption {
	return func(r *ast.Reference) {
		r.Column = col
	}
}

// OnDelete sets the ON DELETE action.
func OnDelete(action string) RefOption {
	return func(r *ast.Reference) {
		r.OnDelete = action
	}
}

// OnUpdate sets the ON UPDATE action.
func OnUpdate(action string) RefOption {
	return func(r *ast.Reference) {
		r.OnUpdate = action
	}
}

// Min sets the minimum constraint.
// For strings: minimum length. For numbers: minimum value.
func (c *ColumnBuilder) Min(n int) *ColumnBuilder {
	f := float64(n)
	c.def.Min = &f
	return c
}

// Max sets the maximum constraint.
// For strings: maximum length. For numbers: maximum value.
func (c *ColumnBuilder) Max(n int) *ColumnBuilder {
	f := float64(n)
	c.def.Max = &f
	return c
}

// Pattern sets a regex pattern for validation.
// Used for CHECK constraints and OpenAPI pattern.
func (c *ColumnBuilder) Pattern(pattern string) *ColumnBuilder {
	c.def.Pattern = pattern
	return c
}

// Format sets the format for validation and OpenAPI.
// If the format has an associated pattern, it is applied automatically.
func (c *ColumnBuilder) Format(format string) *ColumnBuilder {
	c.def.Format = format
	return c
}

// FormatDef sets format from a FormatDef object (from fmt.X).
// This applies both format and pattern in one call.
func (c *ColumnBuilder) FormatDef(fd FormatDef) *ColumnBuilder {
	c.def.Format = fd.Format
	// Only auto-apply pattern if not already set
	if c.def.Pattern == "" && fd.Pattern != "" {
		c.def.Pattern = fd.Pattern
	}
	return c
}

// Docs sets the documentation/description for the column.
// This becomes a SQL COMMENT and OpenAPI description.
func (c *ColumnBuilder) Docs(description string) *ColumnBuilder {
	c.def.Docs = description
	return c
}

// Deprecated marks the column as deprecated with a reason.
func (c *ColumnBuilder) Deprecated(reason string) *ColumnBuilder {
	c.def.Deprecated = reason
	return c
}

// Computed sets the computed expression for the column.
// The column becomes GENERATED ALWAYS AS (expr) STORED (or VIRTUAL if .Virtual() is also called).
func (c *ColumnBuilder) Computed(expr any) *ColumnBuilder {
	c.def.Computed = expr
	return c
}

// Virtual marks the column as virtual.
// With .Computed(): generates VIRTUAL instead of STORED.
// Without .Computed(): app-only field (no DB column, OpenAPI/schema only).
func (c *ColumnBuilder) Virtual() *ColumnBuilder {
	c.def.Virtual = true
	return c
}

// Common ON DELETE/UPDATE action constants.
const (
	Cascade    = "CASCADE"
	SetNull    = "SET NULL"
	Restrict   = "RESTRICT"
	NoAction   = "NO ACTION"
	SetDefault = "SET DEFAULT"
)

// IsStringType returns true if the column type is a string type.
func (c *ColumnBuilder) IsStringType() bool {
	return c.def.Type == "string" || c.def.Type == "text"
}

// IsNumericType returns true if the column type is a numeric type.
func (c *ColumnBuilder) IsNumericType() bool {
	switch c.def.Type {
	case "integer", "float", "decimal":
		return true
	default:
		return false
	}
}

// Def returns the underlying ColumnDef for inspection.
func (c *ColumnBuilder) Def() *ast.ColumnDef {
	return c.def
}
