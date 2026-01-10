// Package runtime provides direct conversion between runtime types and AST.
package runtime

import (
	"github.com/hlop3z/astroladb/internal/ast"
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
func (c *ColumnConverter) ToAST(col *ColumnDef) *ast.ColumnDef {
	if col == nil {
		return nil
	}

	astCol := &ast.ColumnDef{
		Name:       col.Name,
		Type:       col.Type,
		TypeArgs:   col.TypeArgs,
		Nullable:   col.Nullable,
		Unique:     col.Unique,
		PrimaryKey: col.PrimaryKey,
		Format:     col.Format,
		Pattern:    col.Pattern,
		Docs:       col.Docs,
		Deprecated: col.Deprecated,
	}

	// Set default if present
	if col.Default != nil {
		astCol.Default = c.convertValue(col.Default)
		astCol.DefaultSet = true
	}

	// Set backfill if present
	if col.Backfill != nil {
		astCol.Backfill = c.convertValue(col.Backfill)
		astCol.BackfillSet = true
	}

	// Convert min/max (float64* to int*)
	if col.Min != nil {
		minInt := int(*col.Min)
		astCol.Min = &minInt
	}
	if col.Max != nil {
		maxInt := int(*col.Max)
		astCol.Max = &maxInt
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
func (c *ColumnConverter) RefToAST(ref *RefDef) *ast.Reference {
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
func (c *ColumnConverter) IndexToAST(idx *IndexDef) *ast.IndexDef {
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
func (c *ColumnConverter) ColumnsToAST(cols []*ColumnDef) []*ast.ColumnDef {
	result := make([]*ast.ColumnDef, 0, len(cols))
	for _, col := range cols {
		if astCol := c.ToAST(col); astCol != nil {
			result = append(result, astCol)
		}
	}
	return result
}

// IndexesToAST converts a slice of runtime IndexDefs to ast.IndexDefs.
func (c *ColumnConverter) IndexesToAST(idxs []*IndexDef) []*ast.IndexDef {
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
	switch val := v.(type) {
	case map[string]any:
		// Check if this is a SQL expression marker from sql("...")
		if typ, ok := val["_type"].(string); ok && typ == "sql_expr" {
			if expr, ok := val["expr"].(string); ok {
				return &ast.SQLExpr{Expr: expr}
			}
		}
		return val
	default:
		return v
	}
}

// TableBuilderToAST converts a TableBuilder directly to an ast.TableDef.
// This provides a complete conversion path without map intermediaries.
func (c *ColumnConverter) TableBuilderToAST(tb *TableBuilder, namespace, name string) *ast.TableDef {
	return &ast.TableDef{
		Namespace:  namespace,
		Name:       name,
		Columns:    c.ColumnsToAST(tb.columns),
		Indexes:    c.IndexesToAST(tb.indexes),
		Checks:     make([]*ast.CheckDef, 0),
		Docs:       tb.docs,
		Deprecated: tb.deprecated,
	}
}

// TableChainToAST converts a TableChain directly to an ast.TableDef.
func (c *ColumnConverter) TableChainToAST(tc *TableChain, namespace, name string) *ast.TableDef {
	return &ast.TableDef{
		Namespace:  namespace,
		Name:       name,
		Columns:    c.ColumnsToAST(tc.columns),
		Indexes:    c.IndexesToAST(tc.indexes),
		Checks:     make([]*ast.CheckDef, 0),
		Docs:       tc.docs,
		Deprecated: tc.deprecated,
		Auditable:  tc.auditable,
		SortBy:     tc.sortBy,
		Searchable: tc.searchable,
		Filterable: tc.filterable,
	}
}
