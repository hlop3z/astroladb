// Package schema provides schema parsing from JavaScript objects to AST.
package schema

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// SchemaParser converts JavaScript objects to AST definitions.
// It handles the conversion of columns, indexes, and references
// from JS map representations to typed Go structs.
type SchemaParser struct{}

// NewSchemaParser creates a new schema parser.
func NewSchemaParser() *SchemaParser {
	return &SchemaParser{}
}

// ParseColumnDef converts a JS object to an ast.ColumnDef.
func (p *SchemaParser) ParseColumnDef(obj any) *ast.ColumnDef {
	m, ok := obj.(map[string]any)
	if !ok {
		return nil
	}

	// Skip metadata-only entries (relationships, polymorphic markers)
	if _, isMetadata := m["_type"]; isMetadata {
		return nil
	}

	// Skip entries without a name (invalid column definitions)
	name, hasName := m["name"].(string)
	if !hasName || name == "" {
		return nil
	}

	col := &ast.ColumnDef{
		Name: name,
	}
	if typ, ok := m["type"].(string); ok {
		col.Type = typ
	}
	if args, ok := m["type_args"].([]any); ok {
		col.TypeArgs = args
	}
	if nullable, ok := m["nullable"].(bool); ok {
		col.Nullable = nullable
		col.NullableSet = true
	}
	if unique, ok := m["unique"].(bool); ok {
		col.Unique = unique
	}
	if pk, ok := m["primary_key"].(bool); ok {
		col.PrimaryKey = pk
	}
	if def, exists := m["default"]; exists {
		col.Default = p.convertValue(def)
		col.DefaultSet = true
	}
	if backfill, exists := m["backfill"]; exists {
		col.Backfill = p.convertValue(backfill)
		col.BackfillSet = true
	}

	// Parse reference
	if ref, ok := m["reference"].(map[string]any); ok {
		col.Reference = p.ParseReference(ref)
	}

	// Parse validation (preserve float64, no truncation)
	if min, ok := m["min"].(float64); ok {
		col.Min = &min
	}
	if max, ok := m["max"].(float64); ok {
		col.Max = &max
	}
	if pattern, ok := m["pattern"].(string); ok {
		col.Pattern = pattern
	}
	if format, ok := m["format"].(string); ok {
		col.Format = format
	}

	// Parse documentation
	if docs, ok := m["docs"].(string); ok {
		col.Docs = docs
	}
	if deprecated, ok := m["deprecated"].(string); ok {
		col.Deprecated = deprecated
	}

	// Parse computed expression
	if computed, exists := m["computed"]; exists {
		col.Computed = computed
	}

	return col
}

// ParseReference converts a JS object to an ast.Reference.
func (p *SchemaParser) ParseReference(m map[string]any) *ast.Reference {
	return ParseReferenceMap(m)
}

// ParseIndexDef converts a JS object to an ast.IndexDef.
func (p *SchemaParser) ParseIndexDef(obj any) *ast.IndexDef {
	return ParseIndexDefMap(obj)
}

// ParseReferenceMap converts a JS object to an ast.Reference (standalone function).
func ParseReferenceMap(m map[string]any) *ast.Reference {
	ref := &ast.Reference{}

	if table, ok := m["table"].(string); ok {
		// Keep the original reference format (e.g., "auth.users")
		// The SQL dialect will convert to flat table name when generating SQL
		ref.Table = table
	}
	if column, ok := m["column"].(string); ok {
		ref.Column = column
	} else {
		ref.Column = "id" // Default to id
	}
	if onDelete, ok := m["on_delete"].(string); ok {
		ref.OnDelete = onDelete
	}
	if onUpdate, ok := m["on_update"].(string); ok {
		ref.OnUpdate = onUpdate
	}

	return ref
}

// ParseIndexDefMap converts a JS object to an ast.IndexDef (standalone function).
func ParseIndexDefMap(obj any) *ast.IndexDef {
	m, ok := obj.(map[string]any)
	if !ok {
		return nil
	}

	idx := &ast.IndexDef{}

	if name, ok := m["name"].(string); ok {
		idx.Name = name
	}
	if cols, ok := m["columns"].([]any); ok {
		for _, c := range cols {
			if s, ok := c.(string); ok {
				idx.Columns = append(idx.Columns, s)
			}
		}
	}
	if unique, ok := m["unique"].(bool); ok {
		idx.Unique = unique
	}
	if where, ok := m["where"].(string); ok {
		idx.Where = where
	}

	return idx
}

// convertValue converts JS values (like sql("...") results) to Go types.
func (p *SchemaParser) convertValue(v any) any {
	return ast.ConvertSQLExprValue(v)
}

// ParseColumns parses a list of column objects into ast.ColumnDef slice.
func (p *SchemaParser) ParseColumns(colsRaw any) []*ast.ColumnDef {
	var cols []any
	switch c := colsRaw.(type) {
	case []any:
		cols = c
	case []map[string]any:
		for _, m := range c {
			cols = append(cols, m)
		}
	default:
		return nil
	}

	result := make([]*ast.ColumnDef, 0, len(cols))
	for _, c := range cols {
		if colDef := p.ParseColumnDef(c); colDef != nil {
			result = append(result, colDef)
		}
	}
	return result
}

// ParseIndexes parses a list of index objects into ast.IndexDef slice.
func (p *SchemaParser) ParseIndexes(idxsRaw any) []*ast.IndexDef {
	result := make([]*ast.IndexDef, 0)

	switch idxs := idxsRaw.(type) {
	case []any:
		for _, i := range idxs {
			if idxDef := p.ParseIndexDef(i); idxDef != nil {
				result = append(result, idxDef)
			}
		}
	case []map[string]any:
		for _, i := range idxs {
			if idxDef := p.ParseIndexDef(i); idxDef != nil {
				result = append(result, idxDef)
			}
		}
	}

	return result
}

// ParseStringSlice extracts a string slice from various formats.
func (p *SchemaParser) ParseStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
