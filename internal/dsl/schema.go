package dsl

import (
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/registry"
)

// SchemaBuilder provides a declarative API for defining schemas.
// Used in schema files: schema("namespace", s => { ... })
type SchemaBuilder struct {
	namespace string
	registry  *registry.ModelRegistry
	tables    []*ast.TableDef
}

// NewSchemaBuilder creates a new SchemaBuilder for the given namespace.
func NewSchemaBuilder(namespace string, reg *registry.ModelRegistry) *SchemaBuilder {
	return &SchemaBuilder{
		namespace: namespace,
		registry:  reg,
		tables:    make([]*ast.TableDef, 0),
	}
}

// Namespace returns the current namespace.
func (s *SchemaBuilder) Namespace() string {
	return s.namespace
}

// Table creates a new table definition in this namespace.
// The callback receives a TableBuilder for defining columns and relationships.
func (s *SchemaBuilder) Table(name string, fn func(*TableBuilder)) *ast.TableDef {
	tb := NewTableBuilder(s.namespace, name)
	fn(tb)
	def := tb.Build()
	s.tables = append(s.tables, def)

	// Register with the registry if available
	if s.registry != nil {
		_ = s.registry.Register(s.namespace, name, def)
	}

	return def
}

// Tables returns all defined tables.
func (s *SchemaBuilder) Tables() []*ast.TableDef {
	return s.tables
}

// SingleTableBuilder provides a standalone table definition API.
// Used in single-file table definitions: table(t => { ... })
type SingleTableBuilder struct {
	def      *ast.TableDef
	registry *registry.ModelRegistry
}

// NewSingleTableBuilder creates a builder for a standalone table definition.
// The namespace and name should be derived from the file path.
func NewSingleTableBuilder(namespace, name string, reg *registry.ModelRegistry) *SingleTableBuilder {
	return &SingleTableBuilder{
		def: &ast.TableDef{
			Namespace: namespace,
			Name:      name,
			Columns:   make([]*ast.ColumnDef, 0),
			Indexes:   make([]*ast.IndexDef, 0),
			Checks:    make([]*ast.CheckDef, 0),
		},
		registry: reg,
	}
}

// Build finalizes and returns the table definition.
func (s *SingleTableBuilder) Build() *ast.TableDef {
	// Register with the registry if available
	if s.registry != nil && s.def.Namespace != "" {
		_ = s.registry.Register(s.def.Namespace, s.def.Name, s.def)
	}
	return s.def
}

// TableBuilder returns a TableBuilder for this definition.
func (s *SingleTableBuilder) TableBuilder() *TableBuilder {
	return &TableBuilder{
		namespace: s.def.Namespace,
		def:       s.def,
	}
}
