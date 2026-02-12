package runtime

import (
	"strings"

	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// TableChain - Chainable table operations for object-based API
// -----------------------------------------------------------------------------

// TableChain provides chainable table-level operations after column definitions.
type TableChain struct {
	vm            *goja.Runtime
	columns       []*ColumnDef
	indexes       []*IndexDef
	relationships []*RelationshipDef
	docs          string
	deprecated    string

	// Table-level metadata for x-db extensions
	auditable  bool     // Add created_by, updated_by columns
	sortBy     []string // Default ordering (e.g., ["-created_at", "name"])
	searchable []string // Columns for fulltext search
	filterable []string // Columns allowed in WHERE clauses
}

// NewTableChain creates a TableChain from collected column definitions.
func NewTableChain(vm *goja.Runtime, columns []*ColumnDef, indexes []*IndexDef) *TableChain {
	return &TableChain{
		vm:            vm,
		columns:       columns,
		indexes:       indexes,
		relationships: make([]*RelationshipDef, 0),
	}
}

// ToChainableObject returns a JS object with chainable table methods.
func (tc *TableChain) ToChainableObject() *goja.Object {
	obj := tc.vm.NewObject()

	// timestamps() - adds created_at and updated_at
	_ = obj.Set("timestamps", func() *goja.Object {
		tc.columns = append(tc.columns, &ColumnDef{
			Name:    "created_at",
			Type:    "datetime",
			Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
		})
		tc.columns = append(tc.columns, &ColumnDef{
			Name:    "updated_at",
			Type:    "datetime",
			Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
		})
		return obj
	})

	// soft_delete() - adds deleted_at
	_ = obj.Set("soft_delete", func() *goja.Object {
		tc.columns = append(tc.columns, &ColumnDef{
			Name:     "deleted_at",
			Type:     "datetime",
			Nullable: true,
		})
		return obj
	})

	// sortable() - adds position
	_ = obj.Set("sortable", func() *goja.Object {
		tc.columns = append(tc.columns, &ColumnDef{
			Name:    "position",
			Type:    "integer",
			Default: 0,
		})
		return obj
	})

	// index(...columns) - composite index
	_ = obj.Set("index", func(columns ...string) *goja.Object {
		tc.indexes = append(tc.indexes, &IndexDef{
			Columns: columns,
		})
		return obj
	})

	// unique(...columns) - composite uniqueness
	_ = obj.Set("unique", func(columns ...string) *goja.Object {
		existingCols := make(map[string]bool)
		for _, c := range tc.columns {
			existingCols[c.Name] = true
		}

		cols := make([]string, len(columns))
		for i, col := range columns {
			if strings.HasSuffix(col, "_id") || strings.HasSuffix(col, "_type") {
				cols[i] = col
			} else if existingCols[col+"_id"] {
				cols[i] = col + "_id"
			} else {
				cols[i] = col
			}
		}
		tc.indexes = append(tc.indexes, &IndexDef{
			Columns: cols,
			Unique:  true,
		})
		return obj
	})

	// many_to_many(ref)
	_ = obj.Set("many_to_many", func(ref string) *goja.Object {
		tc.relationships = append(tc.relationships, &RelationshipDef{
			Type:   "many_to_many",
			Target: ref,
		})
		return obj
	})

	// junction(refs...) - marks table as M2M junction
	// Accepts 0 or 2 parameters:
	//   junction() - auto-detect from FKs (2 FKs only)
	//   junction("blog.post", "blog.tag") - explicit refs
	_ = obj.Set("junction", func(refs ...string) *goja.Object {
		if len(refs) != 0 && len(refs) != 2 {
			panic(tc.vm.ToValue("junction() requires 0 or 2 parameters, got " + string(rune(len(refs)))))
		}

		rel := &RelationshipDef{
			Type: "junction",
		}
		if len(refs) == 2 {
			rel.JunctionSource = refs[0]
			rel.JunctionTarget = refs[1]
		}
		tc.relationships = append(tc.relationships, rel)
		return obj
	})

	// belongs_to_any(refs, opts)
	_ = obj.Set("belongs_to_any", func(refs []string, opts ...map[string]any) *goja.Object {
		var as string
		if len(opts) > 0 {
			if v, ok := opts[0]["as"].(string); ok {
				as = v
			}
		}
		if as == "" {
			as = "polymorphic"
		}

		typeCol := as + "_type"
		idCol := as + "_id"

		tc.columns = append(tc.columns, &ColumnDef{
			Name:     typeCol,
			Type:     "string",
			TypeArgs: []any{100},
		})
		tc.columns = append(tc.columns, &ColumnDef{
			Name: idCol,
			Type: "uuid",
		})
		tc.indexes = append(tc.indexes, &IndexDef{
			Columns: []string{typeCol, idCol},
		})
		tc.relationships = append(tc.relationships, &RelationshipDef{
			Type:    "polymorphic",
			Targets: refs,
			As:      as,
		})
		return obj
	})

	// docs(description) - table documentation
	_ = obj.Set("docs", func(description string) *goja.Object {
		tc.docs = description
		return obj
	})

	// deprecated(reason) - mark table as deprecated
	_ = obj.Set("deprecated", func(reason string) *goja.Object {
		tc.deprecated = reason
		return obj
	})

	// auditable() - adds created_by and updated_by columns
	_ = obj.Set("auditable", func() *goja.Object {
		tc.auditable = true
		tc.columns = append(tc.columns, &ColumnDef{
			Name:     "created_by",
			Type:     "uuid",
			Nullable: true,
		})
		tc.columns = append(tc.columns, &ColumnDef{
			Name:     "updated_by",
			Type:     "uuid",
			Nullable: true,
		})
		return obj
	})

	// sort_by(...columns) - default ordering (e.g., ["-created_at", "name"])
	_ = obj.Set("sort_by", func(columns ...string) *goja.Object {
		tc.sortBy = columns
		return obj
	})

	// searchable(...columns) - columns for fulltext search
	_ = obj.Set("searchable", func(columns ...string) *goja.Object {
		tc.searchable = columns
		return obj
	})

	// filterable(...columns) - columns allowed in WHERE clauses
	_ = obj.Set("filterable", func(columns ...string) *goja.Object {
		tc.filterable = columns
		return obj
	})

	// Store result data for extraction
	_ = obj.Set("_getResult", func() goja.Value {
		return tc.ToResult()
	})

	return obj
}

// ToResult converts the TableChain to a result object for JavaScript.
func (tc *TableChain) ToResult() goja.Value {
	result := tc.vm.NewObject()

	// Convert columns to maps
	columns := make([]map[string]any, 0, len(tc.columns)+len(tc.relationships))
	for _, col := range tc.columns {
		columns = append(columns, columnDefToMap(col))
	}

	// Add relationship markers
	for _, rel := range tc.relationships {
		columns = append(columns, relationshipDefToMap(rel))
	}

	// Convert indexes to maps
	indexes := make([]map[string]any, 0, len(tc.indexes))
	for _, idx := range tc.indexes {
		indexes = append(indexes, indexDefToMap(idx))
	}

	_ = result.Set("columns", columns)
	_ = result.Set("indexes", indexes)
	if tc.docs != "" {
		_ = result.Set("docs", tc.docs)
	}
	if tc.deprecated != "" {
		_ = result.Set("deprecated", tc.deprecated)
	}

	// Table-level metadata for x-db extensions
	if tc.auditable {
		_ = result.Set("auditable", true)
	}
	if len(tc.sortBy) > 0 {
		_ = result.Set("sort_by", tc.sortBy)
	}
	if len(tc.searchable) > 0 {
		_ = result.Set("searchable", tc.searchable)
	}
	if len(tc.filterable) > 0 {
		_ = result.Set("filterable", tc.filterable)
	}

	return result
}

// Helper functions for converting defs to maps (used by both TableBuilder and TableChain)
