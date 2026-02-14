// Package schema provides schema registry for collecting table definitions.
package schema

import (
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
)

// SchemaRegistry collects table definitions and operations during JS evaluation.
// It provides a centralized place for managing evaluation state and results.
type SchemaRegistry struct {
	tables     []*ast.TableDef
	operations []ast.Operation
	meta       *metadata.Metadata
	parser     *SchemaParser
}

// NewSchemaRegistry creates a new schema registry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		tables:     make([]*ast.TableDef, 0),
		operations: make([]ast.Operation, 0),
		meta:       metadata.New(),
		parser:     NewSchemaParser(),
	}
}

// RegisterTable adds a parsed table definition to the registry.
func (r *SchemaRegistry) RegisterTable(table *ast.TableDef) {
	if table != nil {
		r.tables = append(r.tables, table)
		r.meta.AddTable(table)
	}
}

// RegisterOperation adds an operation to the registry.
func (r *SchemaRegistry) RegisterOperation(op ast.Operation) {
	if op != nil {
		r.operations = append(r.operations, op)
	}
}

// Tables returns all collected table definitions.
func (r *SchemaRegistry) Tables() []*ast.TableDef {
	return r.tables
}

// Operations returns all collected operations.
func (r *SchemaRegistry) Operations() []ast.Operation {
	return r.operations
}

// Metadata returns the schema metadata.
func (r *SchemaRegistry) Metadata() *metadata.Metadata {
	return r.meta
}

// Parser returns the schema parser.
func (r *SchemaRegistry) Parser() *SchemaParser {
	return r.parser
}

// ClearTables resets the collected tables.
func (r *SchemaRegistry) ClearTables() {
	r.tables = make([]*ast.TableDef, 0)
}

// ClearOperations resets the collected operations.
func (r *SchemaRegistry) ClearOperations() {
	r.operations = make([]ast.Operation, 0)
}

// ClearMetadata resets the metadata.
func (r *SchemaRegistry) ClearMetadata() {
	r.meta = metadata.New()
}

// Clear resets all state.
func (r *SchemaRegistry) Clear() {
	r.ClearTables()
	r.ClearOperations()
	r.ClearMetadata()
}

// AddManyToMany registers a many-to-many relationship in metadata.
func (r *SchemaRegistry) AddManyToMany(namespace, tableName, target, sourceFile string) {
	r.meta.AddManyToMany(namespace, tableName, target, sourceFile)
}

// AddPolymorphic registers a polymorphic relationship in metadata.
func (r *SchemaRegistry) AddPolymorphic(namespace, tableName, alias string, targets []string) {
	r.meta.AddPolymorphic(namespace, tableName, alias, targets)
}

// AddExplicitJunction registers an explicit junction table in metadata.
func (r *SchemaRegistry) AddExplicitJunction(sourceRef, targetRef, sourceFK, targetFK string) {
	r.meta.AddExplicitJunction(sourceRef, targetRef, sourceFK, targetFK)
}

// GetJoinTables returns all auto-generated join table definitions.
func (r *SchemaRegistry) GetJoinTables() []*ast.TableDef {
	return r.meta.GetJoinTables()
}

// SaveMetadata saves the metadata to the .alab directory.
func (r *SchemaRegistry) SaveMetadata(projectDir string) error {
	return r.meta.Save(projectDir)
}

// ParseAndRegisterTable parses a table definition and registers it.
func (r *SchemaRegistry) ParseAndRegisterTable(namespace, name string, defObj any) *ast.TableDef {
	def, ok := defObj.(map[string]any)
	if !ok {
		return nil
	}

	tableDef := &ast.TableDef{
		Namespace: namespace,
		Name:      name,
		Columns:   make([]*ast.ColumnDef, 0),
		Indexes:   make([]*ast.IndexDef, 0),
		Checks:    make([]*ast.CheckDef, 0),
	}

	// Parse columns and extract relationships
	if colsRaw, exists := def["columns"]; exists {
		columns := r.parseColumnsWithRelationships(namespace, name, colsRaw)
		tableDef.Columns = columns
	}

	// Parse indexes
	if idxsRaw, exists := def["indexes"]; exists {
		tableDef.Indexes = r.parser.ParseIndexes(idxsRaw)
	}

	// Parse metadata
	if docs, ok := def["docs"].(string); ok {
		tableDef.Docs = docs
	}
	if deprecated, ok := def["deprecated"].(string); ok {
		tableDef.Deprecated = deprecated
	}

	// Parse x-db metadata
	if auditable, ok := def["auditable"].(bool); ok {
		tableDef.Auditable = auditable
	}
	tableDef.SortBy = r.parser.ParseStringSlice(def["sort_by"])
	tableDef.Searchable = r.parser.ParseStringSlice(def["searchable"])
	tableDef.Filterable = r.parser.ParseStringSlice(def["filterable"])

	r.RegisterTable(tableDef)
	return tableDef
}

// parseColumnsWithRelationships parses columns and extracts relationship metadata.
func (r *SchemaRegistry) parseColumnsWithRelationships(namespace, tableName string, colsRaw any) []*ast.ColumnDef {
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
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}

		// Check for many_to_many relationship
		if t, ok := m["_type"].(string); ok && t == "relationship" {
			if rel, ok := m["relationship"].(map[string]any); ok {
				if relType, ok := rel["type"].(string); ok && relType == "many_to_many" {
					if target, ok := rel["target"].(string); ok {
						// Note: legacy registry code doesn't track source files
						r.AddManyToMany(namespace, tableName, target, "")
					}
				}
			}
			continue
		}

		// Check for polymorphic relationship
		if t, ok := m["_type"].(string); ok && t == "polymorphic" {
			if poly, ok := m["polymorphic"].(map[string]any); ok {
				alias, _ := poly["as"].(string)
				targets := r.parser.ParseStringSlice(poly["targets"])
				r.AddPolymorphic(namespace, tableName, alias, targets)
			}
			continue
		}

		// Check for junction marker
		if t, ok := m["_type"].(string); ok && t == "junction" {
			if junc, ok := m["junction"].(map[string]any); ok {
				// Collect FK columns from the table
				var fkCols []*ast.ColumnDef
				for _, col := range cols {
					if colMap, ok := col.(map[string]any); ok {
						if colDef := r.parser.ParseColumnDef(colMap); colDef != nil {
							if colDef.Reference != nil {
								fkCols = append(fkCols, colDef)
							}
						}
					}
				}

				// Check for explicit refs
				sourceRef, hasSource := junc["source"].(string)
				targetRef, hasTarget := junc["target"].(string)

				if hasSource && hasTarget {
					// Explicit refs provided - find matching FK columns
					var sourceFK, targetFK string
					for _, fk := range fkCols {
						if fk.Reference.Table == sourceRef {
							sourceFK = fk.Name
						}
						if fk.Reference.Table == targetRef {
							targetFK = fk.Name
						}
					}
					if sourceFK != "" && targetFK != "" {
						r.AddExplicitJunction(sourceRef, targetRef, sourceFK, targetFK)
					}
				} else {
					// Auto-detect: require exactly 2 FKs
					if len(fkCols) == 2 {
						r.AddExplicitJunction(
							fkCols[0].Reference.Table,
							fkCols[1].Reference.Table,
							fkCols[0].Name,
							fkCols[1].Name,
						)
					}
				}
			}
			continue
		}

		// Parse regular column
		if colDef := r.parser.ParseColumnDef(c); colDef != nil {
			result = append(result, colDef)
		}
	}
	return result
}
