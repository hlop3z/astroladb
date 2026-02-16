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
		Name:       col.Name,
		Type:       col.Type,
		TypeArgs:   col.TypeArgs,
		Nullable:   col.Nullable,
		Unique:     col.Unique,
		Index:      col.Index,
		PrimaryKey: col.PrimaryKey,
		Format:     col.Format,
		Pattern:    col.Pattern,
		Docs:       col.Docs,
		Deprecated: col.Deprecated,
		ReadOnly:   col.ReadOnly,
		WriteOnly:  col.WriteOnly,
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

	// Convert reference
	if col.Reference != nil {
		astCol.Reference = c.RefToAST(col.Reference)
	}

	// Convert computed expression
	if col.Computed != nil {
		astCol.Computed = col.Computed
	}

	return astCol
}

// RefToAST converts a runtime RefDef to an ast.Reference.
func (c *ColumnConverter) RefToAST(ref *builder.RefDef) *ast.Reference {
	if ref == nil {
		return nil
	}

	column := ref.Column
	if column == "" {
		column = "id" // Default to id
	}

	return &ast.Reference{
		Table:    ref.Table,
		Column:   column,
		OnDelete: ref.OnDelete,
		OnUpdate: ref.OnUpdate,
	}
}

// IndexToAST converts a runtime IndexDef to an ast.IndexDef.
func (c *ColumnConverter) IndexToAST(idx *builder.IndexDef) *ast.IndexDef {
	if idx == nil {
		return nil
	}

	return &ast.IndexDef{
		Name:    idx.Name,
		Columns: idx.Columns,
		Unique:  idx.Unique,
		Where:   idx.Where,
	}
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

// IndexesToAST converts a slice of runtime IndexDefs to ast.IndexDefs.
func (c *ColumnConverter) IndexesToAST(idxs []*builder.IndexDef) []*ast.IndexDef {
	result := make([]*ast.IndexDef, 0, len(idxs))
	for _, idx := range idxs {
		if astIdx := c.IndexToAST(idx); astIdx != nil {
			result = append(result, astIdx)
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
func (c *ColumnConverter) TableBuilderToAST(tb *builder.TableBuilder, namespace, name string) *ast.TableDef {
	return &ast.TableDef{
		Namespace:  namespace,
		Name:       name,
		Columns:    c.ColumnsToAST(tb.Columns),
		Indexes:    c.IndexesToAST(tb.Indexes),
		Checks:     make([]*ast.CheckDef, 0),
		Docs:       tb.Docs,
		Deprecated: tb.Deprecated,
	}
}

// TableChainToAST converts a TableChain directly to an ast.TableDef.
func (c *ColumnConverter) TableChainToAST(tc *builder.TableChain, namespace, name string) *ast.TableDef {
	return &ast.TableDef{
		Namespace:  namespace,
		Name:       name,
		Columns:    c.ColumnsToAST(tc.Columns),
		Indexes:    c.IndexesToAST(tc.Indexes),
		Checks:     make([]*ast.CheckDef, 0),
		Docs:       tc.Docs,
		Deprecated: tc.Deprecated,
		Auditable:  tc.Auditable,
		SortBy:     tc.SortBy,
		Searchable: tc.Searchable,
		Filterable: tc.Filterable,
	}
}
