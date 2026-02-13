// Package registry provides the ModelRegistry for managing table definitions
// organized by namespace. It implements Django-style app.model reference resolution
// with flat SQL naming (auth.users -> auth_users).
package registry

import (
	"slices"
	"strings"
	"sync"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// ModelRegistry stores and manages table definitions by namespace.
// It provides thread-safe access to registered tables and supports
// reference resolution with various syntaxes.
//
// Key: "namespace.table" (e.g., "auth.users")
type ModelRegistry struct {
	tables map[string]*ast.TableDef // key: "namespace.table"
	mu     sync.RWMutex
}

// NewModelRegistry creates a new empty ModelRegistry.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		tables: make(map[string]*ast.TableDef),
	}
}

// Register adds a table definition to the registry.
// Returns an error if a table with the same namespace.table already exists.
func (r *ModelRegistry) Register(ns, table string, def *ast.TableDef) error {
	if ns == "" {
		return alerr.New(alerr.ErrInvalidIdentifier, "namespace cannot be empty").
			With("table", table)
	}
	if table == "" {
		return alerr.New(alerr.ErrInvalidIdentifier, "table name cannot be empty").
			With("namespace", ns)
	}
	if def == nil {
		return alerr.New(alerr.ErrSchemaInvalid, "table definition cannot be nil").
			WithTable(ns, table)
	}

	key := ns + "." + table

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tables[key]; exists {
		return alerr.New(alerr.ErrSchemaDuplicate, "table already registered").
			WithTable(ns, table)
	}

	// Ensure the definition has the correct namespace and name
	def.Namespace = ns
	def.Name = table

	r.tables[key] = def
	return nil
}

// Get retrieves a table definition by namespace and table name.
// Returns the definition and true if found, nil and false otherwise.
func (r *ModelRegistry) Get(ns, table string) (*ast.TableDef, bool) {
	key := ns + "." + table

	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.tables[key]
	return def, ok
}

// GetByRef retrieves a table definition by its qualified reference.
// The ref must be in "namespace.table" format.
// Returns an error if the reference format is invalid or the table is not found.
func (r *ModelRegistry) GetByRef(ref string) (*ast.TableDef, error) {
	ns, table, isRelative := ParseReference(ref)

	if isRelative {
		return nil, alerr.New(alerr.ErrInvalidReference, "GetByRef requires fully qualified reference (namespace.table)").
			With("ref", ref).
			With("hint", "use Resolve() for relative references")
	}

	if ns == "" {
		return nil, alerr.New(alerr.ErrInvalidReference, "reference must include namespace").
			With("ref", ref)
	}

	def, ok := r.Get(ns, table)
	if !ok {
		return nil, alerr.New(alerr.ErrSchemaNotFound, "table not found").
			With("ref", ref)
	}

	return def, nil
}

// All returns a copy of all registered tables.
// The returned map is safe to iterate over without holding locks.
func (r *ModelRegistry) All() map[string]*ast.TableDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ast.TableDef, len(r.tables))
	for k, v := range r.tables {
		result[k] = v
	}
	return result
}

// AllInNamespace returns all tables in the given namespace.
// Returns an empty slice if the namespace has no tables.
// Tables are returned in alphabetical order by name.
func (r *ModelRegistry) AllInNamespace(ns string) []*ast.TableDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ast.TableDef
	for _, def := range r.tables {
		if def.Namespace == ns {
			result = append(result, def)
		}
	}

	// Sort by table name for deterministic output
	slices.SortFunc(result, func(a, b *ast.TableDef) int {
		return strings.Compare(a.Name, b.Name)
	})

	return result
}

// Namespaces returns a sorted list of all registered namespaces.
func (r *ModelRegistry) Namespaces() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nsSet := make(map[string]struct{})
	for _, def := range r.tables {
		nsSet[def.Namespace] = struct{}{}
	}

	result := make([]string, 0, len(nsSet))
	for ns := range nsSet {
		result = append(result, ns)
	}
	slices.Sort(result)

	return result
}

// Clear removes all tables from the registry.
// This is primarily useful for testing.
func (r *ModelRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tables = make(map[string]*ast.TableDef)
}

// Count returns the total number of registered tables.
func (r *ModelRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tables)
}

// CountInNamespace returns the number of tables in the given namespace.
func (r *ModelRegistry) CountInNamespace(ns string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, def := range r.tables {
		if def.Namespace == ns {
			count++
		}
	}
	return count
}
