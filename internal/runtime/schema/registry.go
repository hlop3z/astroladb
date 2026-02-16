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
}

// NewSchemaRegistry creates a new schema registry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		tables:     make([]*ast.TableDef, 0),
		operations: make([]ast.Operation, 0),
		meta:       metadata.New(),
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
