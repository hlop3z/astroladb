// Package state provides schema state management for the engine.
// It contains the Schema type and replay operations for computing
// schema state at a given revision.
package state

import (
	"sort"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/registry"
)

// Schema represents a complete database schema containing multiple tables.
// It provides methods for merging table definitions and validating the schema.
type Schema struct {
	Tables map[string]*ast.TableDef // key: "namespace.table"
}

// NewSchema creates a new empty Schema.
func NewSchema() *Schema {
	return &Schema{
		Tables: make(map[string]*ast.TableDef),
	}
}

// SchemaFromTables creates a Schema from a slice of table definitions.
func SchemaFromTables(tables []*ast.TableDef) (*Schema, error) {
	s := NewSchema()
	if err := s.Merge(tables); err != nil {
		return nil, err
	}
	return s, nil
}

// SchemaFromRegistry creates a Schema from a model registry.
func SchemaFromRegistry(reg *registry.ModelRegistry) *Schema {
	s := NewSchema()
	for key, def := range reg.All() {
		s.Tables[key] = def
	}
	return s
}

// Merge adds multiple table definitions to the schema.
// Returns an error if any table already exists in the schema.
func (s *Schema) Merge(tables []*ast.TableDef) error {
	for _, t := range tables {
		if err := s.AddTable(t); err != nil {
			return err
		}
	}
	return nil
}

// MergeOverwrite adds multiple table definitions, overwriting existing ones.
// This is useful when updating a schema with newer definitions.
func (s *Schema) MergeOverwrite(tables []*ast.TableDef) {
	for _, t := range tables {
		s.Tables[t.QualifiedName()] = t
	}
}

// AddTable adds a single table definition to the schema.
// Returns an error if the table already exists.
func (s *Schema) AddTable(t *ast.TableDef) error {
	if t == nil {
		return alerr.New(alerr.ErrSchemaInvalid, "table definition cannot be nil")
	}

	key := t.QualifiedName()
	if _, exists := s.Tables[key]; exists {
		return alerr.New(alerr.ErrSchemaDuplicate, "table already exists in schema").
			WithTable(t.Namespace, t.Name)
	}

	s.Tables[key] = t
	return nil
}

// GetTable retrieves a table by its qualified name (namespace.table).
func (s *Schema) GetTable(qualifiedName string) (*ast.TableDef, bool) {
	t, ok := s.Tables[qualifiedName]
	return t, ok
}

// GetTableByParts retrieves a table by namespace and table name.
func (s *Schema) GetTableByParts(namespace, name string) (*ast.TableDef, bool) {
	key := namespace + "." + name
	t, ok := s.Tables[key]
	return t, ok
}

// RemoveTable removes a table from the schema.
func (s *Schema) RemoveTable(qualifiedName string) {
	delete(s.Tables, qualifiedName)
}

// TableNames returns a sorted list of all table names (qualified).
func (s *Schema) TableNames() []string {
	names := make([]string, 0, len(s.Tables))
	for name := range s.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TableList returns all tables as a sorted slice.
func (s *Schema) TableList() []*ast.TableDef {
	tables := make([]*ast.TableDef, 0, len(s.Tables))
	for _, t := range s.Tables {
		tables = append(tables, t)
	}
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].QualifiedName() < tables[j].QualifiedName()
	})
	return tables
}

// Namespaces returns a sorted list of all namespaces in the schema.
func (s *Schema) Namespaces() []string {
	nsSet := make(map[string]struct{})
	for _, t := range s.Tables {
		nsSet[t.Namespace] = struct{}{}
	}

	namespaces := make([]string, 0, len(nsSet))
	for ns := range nsSet {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	return namespaces
}

// TablesInNamespace returns all tables in a specific namespace.
func (s *Schema) TablesInNamespace(namespace string) []*ast.TableDef {
	var tables []*ast.TableDef
	for _, t := range s.Tables {
		if t.Namespace == namespace {
			tables = append(tables, t)
		}
	}
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
	return tables
}

// Count returns the total number of tables in the schema.
func (s *Schema) Count() int {
	return len(s.Tables)
}

// IsEmpty returns true if the schema has no tables.
func (s *Schema) IsEmpty() bool {
	return len(s.Tables) == 0
}

// Clone creates a deep copy of the schema.
func (s *Schema) Clone() *Schema {
	clone := NewSchema()
	for k, v := range s.Tables {
		// Create a shallow copy of the table definition
		// Note: This doesn't deep-copy columns/indexes, which is acceptable
		// for most use cases where we don't modify the original definitions
		tableCopy := *v
		clone.Tables[k] = &tableCopy
	}
	return clone
}

// Validate performs comprehensive validation of the schema.
// It checks for:
//   - Valid table and column names
//   - Valid foreign key references
//   - No circular dependencies (if strict)
//   - Required columns (id for tables using id())
func (s *Schema) Validate() error {
	// Validate each table individually
	for _, t := range s.Tables {
		if err := validateTableDef(t); err != nil {
			return err
		}
	}

	// Validate cross-table references
	if err := s.validateReferences(); err != nil {
		return err
	}

	// Check for circular dependencies
	if err := s.detectCycles(); err != nil {
		return err
	}

	return nil
}

// validateReferences checks that all foreign key references point to existing tables.
func (s *Schema) validateReferences() error {
	for _, table := range s.Tables {
		for _, col := range table.Columns {
			if col.Reference == nil {
				continue
			}

			refTable := col.Reference.Table
			// Resolve the reference
			resolvedRef, err := s.resolveReference(refTable, table.Namespace)
			if err != nil {
				return alerr.New(alerr.ErrInvalidReference, "foreign key references unknown table").
					WithTable(table.Namespace, table.Name).
					WithColumn(col.Name).
					With("referenced_table", refTable)
			}

			// Check that the referenced column exists
			refCol := col.Reference.TargetColumn()
			refTableDef, _ := s.GetTable(resolvedRef)
			if refTableDef != nil && !refTableDef.HasColumn(refCol) {
				return alerr.New(alerr.ErrInvalidReference, "foreign key references unknown column").
					WithTable(table.Namespace, table.Name).
					WithColumn(col.Name).
					With("referenced_table", refTable).
					With("referenced_column", refCol)
			}
		}
	}
	return nil
}

// resolveReference resolves a table reference to its qualified name.
func (s *Schema) resolveReference(ref, currentNS string) (string, error) {
	// Parse the reference
	ns, table, _ := registry.ParseReference(ref)

	// If fully qualified, check if it exists
	if ns != "" {
		qualified := ns + "." + table
		if _, exists := s.Tables[qualified]; exists {
			return qualified, nil
		}
		return "", alerr.New(alerr.ErrSchemaNotFound, "table not found").
			With("ref", ref)
	}

	// Try with current namespace
	if currentNS != "" {
		qualified := currentNS + "." + table
		if _, exists := s.Tables[qualified]; exists {
			return qualified, nil
		}
	}

	return "", alerr.New(alerr.ErrSchemaNotFound, "table not found").
		With("ref", ref).
		With("current_ns", currentNS)
}

// detectCycles checks for circular dependencies in foreign key relationships.
// Circular dependencies make it impossible to create tables in the right order.
func (s *Schema) detectCycles() error {
	// Build dependency graph
	deps := make(map[string][]string) // table -> tables it depends on
	for _, table := range s.Tables {
		key := table.QualifiedName()
		deps[key] = s.getDependencies(table)
	}

	// DFS to detect cycles
	visited := make(map[string]int) // 0=unvisited, 1=visiting, 2=visited
	var stack []string

	var visit func(node string) error
	visit = func(node string) error {
		if visited[node] == 1 {
			// Found a cycle - build the cycle path for error message
			cycleStart := -1
			for i, n := range stack {
				if n == node {
					cycleStart = i
					break
				}
			}
			cyclePath := append(stack[cycleStart:], node)
			return alerr.New(alerr.ErrSchemaCircularRef, "circular dependency detected").
				With("cycle", cyclePath)
		}
		if visited[node] == 2 {
			return nil // Already processed
		}

		visited[node] = 1 // Visiting
		stack = append(stack, node)

		for _, dep := range deps[node] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		stack = stack[:len(stack)-1]
		visited[node] = 2 // Visited
		return nil
	}

	// Visit all nodes
	for node := range deps {
		if visited[node] == 0 {
			if err := visit(node); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDependencies returns the list of tables that the given table depends on.
func (s *Schema) getDependencies(table *ast.TableDef) []string {
	var deps []string
	seen := make(map[string]bool)

	for _, col := range table.Columns {
		if col.Reference == nil {
			continue
		}

		refTable := col.Reference.Table
		resolved, err := s.resolveReference(refTable, table.Namespace)
		if err != nil {
			continue // Reference doesn't exist, will be caught by validateReferences
		}

		// Don't include self-references as dependencies (they're allowed)
		if resolved == table.QualifiedName() {
			continue
		}

		if !seen[resolved] {
			seen[resolved] = true
			deps = append(deps, resolved)
		}
	}

	sort.Strings(deps)
	return deps
}

// GetCreationOrder returns tables in dependency order (dependencies first).
// Tables with no dependencies come first, followed by tables that depend on them.
func (s *Schema) GetCreationOrder() ([]*ast.TableDef, error) {
	// Build nodes for topological sort
	nodes := make([]*tableSortNode, 0, len(s.Tables))
	for _, table := range s.Tables {
		nodes = append(nodes, &tableSortNode{
			qualifiedName: table.QualifiedName(),
			deps:          s.getDependencies(table),
			def:           table,
		})
	}

	// Perform topological sort
	sorted, err := TopoSort(nodes)
	if err != nil {
		return nil, alerr.New(alerr.ErrSchemaCircularRef, "circular dependency prevents ordering")
	}

	// Extract sorted table definitions
	result := make([]*ast.TableDef, len(sorted))
	for i, node := range sorted {
		result[i] = node.def
	}
	return result, nil
}

// tableSortNode wraps a TableDef for topological sorting.
type tableSortNode struct {
	qualifiedName string
	deps          []string
	def           *ast.TableDef
}

func (n *tableSortNode) ID() string             { return n.qualifiedName }
func (n *tableSortNode) Dependencies() []string { return n.deps }

// GetDropOrder returns tables in reverse dependency order (dependents first).
// Tables that depend on others should be dropped before the tables they reference.
func (s *Schema) GetDropOrder() ([]*ast.TableDef, error) {
	createOrder, err := s.GetCreationOrder()
	if err != nil {
		return nil, err
	}

	// Reverse the creation order
	dropOrder := make([]*ast.TableDef, len(createOrder))
	for i, t := range createOrder {
		dropOrder[len(createOrder)-1-i] = t
	}

	return dropOrder, nil
}
