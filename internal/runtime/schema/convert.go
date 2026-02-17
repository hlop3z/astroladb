// Package schema provides direct conversion between runtime types and AST.
package schema

import (
	"log/slog"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/runtime/builder"
)

// ColumnConverter converts runtime ColumnDef directly to ast.ColumnDef.
// This eliminates the intermediate map representation and provides
// a direct path from builder types to AST types.
type ColumnConverter struct{}

// NewColumnConverter creates a new column converter.
func NewColumnConverter() *ColumnConverter {
	return &ColumnConverter{}
}

// ToAST converts a runtime ColumnDef to an ast.ColumnDef.
// This bypasses the map intermediate step for direct, efficient conversion.
func (c *ColumnConverter) ToAST(col *builder.ColumnDef) *ast.ColumnDef {
	if col == nil {
		return nil
	}

	astCol := &ast.ColumnDef{
		Name:     col.Name,
		Type:     col.Type,
		TypeArgs: col.TypeArgs,
		Nullable: col.Nullable,
		// NullableSet is always true for builder-created columns because the DSL
		// requires explicit .optional() calls. This lets the diff engine distinguish
		// "not specified" (NullableSet=false) from "explicitly NOT NULL" (NullableSet=true).
		NullableSet: true,
		Unique:      col.Unique,
		Index:       col.Index,
		PrimaryKey:  col.PrimaryKey,
		Format:      col.Format,
		Pattern:     col.Pattern,
		Docs:        col.Docs,
		Deprecated:  col.Deprecated,
		ReadOnly:    col.ReadOnly,
		WriteOnly:   col.WriteOnly,
		Virtual:     col.Virtual,
	}

	// Set default if present
	if col.Default != nil {
		slog.Debug("convert: column has default",
			"column", col.Name,
			"default", col.Default)
		astCol.Default = c.convertValue(col.Default)
		astCol.DefaultSet = true
	} else {
		slog.Debug("convert: column has NO default",
			"column", col.Name)
	}

	// Set backfill if present
	if col.Backfill != nil {
		astCol.Backfill = c.convertValue(col.Backfill)
		astCol.BackfillSet = true
	}

	// Preserve min/max as float64 (no truncation)
	if col.Min != nil {
		astCol.Min = col.Min
	}
	if col.Max != nil {
		astCol.Max = col.Max
	}

	// Pass through reference directly (builder already uses ast.Reference)
	if col.Reference != nil {
		astCol.Reference = col.Reference
	}

	// Convert computed expression
	if col.Computed != nil {
		astCol.Computed = col.Computed
	}

	return astCol
}

// ColumnsToAST converts a slice of runtime ColumnDefs to ast.ColumnDefs.
func (c *ColumnConverter) ColumnsToAST(cols []*builder.ColumnDef) []*ast.ColumnDef {
	result := make([]*ast.ColumnDef, 0, len(cols))
	for _, col := range cols {
		if astCol := c.ToAST(col); astCol != nil {
			result = append(result, astCol)
		}
	}
	return result
}

// convertValue converts JS values (like sql("...") results) to Go types.
func (c *ColumnConverter) convertValue(v any) any {
	return ast.ConvertSQLExprValue(v)
}

// TableBuilderToAST converts a TableBuilder directly to an ast.TableDef.
// This provides a complete conversion path without map intermediaries.
// Used by both the schema path and migration path.
func (c *ColumnConverter) TableBuilderToAST(tb *builder.TableBuilder, namespace, name string) *ast.TableDef {
	return &ast.TableDef{
		Namespace:  namespace,
		Name:       name,
		Columns:    c.ColumnsToAST(tb.Columns),
		Indexes:    tb.Indexes,
		Checks:     make([]*ast.CheckDef, 0),
		Docs:       tb.Docs,
		Deprecated: tb.Deprecated,
		Auditable:  tb.Auditable,
		SortBy:     tb.SortBy,
		Searchable: tb.Searchable,
		Filterable: tb.Filterable,
		Meta:       tb.Meta,
		Lifecycle:  tb.Lifecycle,
		Policy:     tb.Policy,
		Events:     tb.Events,
	}
}
