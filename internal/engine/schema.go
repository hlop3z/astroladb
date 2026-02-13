package engine

import (
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine/state"
	"github.com/hlop3z/astroladb/internal/registry"
)

// Schema is a type alias for state.Schema.
// The Schema type and its methods live in engine/state/.
type Schema = state.Schema

// NewSchema creates a new empty Schema.
func NewSchema() *Schema {
	return state.NewSchema()
}

// SchemaFromTables creates a Schema from a slice of table definitions.
func SchemaFromTables(tables []*ast.TableDef) (*Schema, error) {
	return state.SchemaFromTables(tables)
}

// SchemaFromRegistry creates a Schema from a model registry.
func SchemaFromRegistry(reg *registry.ModelRegistry) *Schema {
	return state.SchemaFromRegistry(reg)
}

// ReplayOperations applies a sequence of operations to build the resulting schema state.
func ReplayOperations(ops []ast.Operation) (*Schema, error) {
	return state.ReplayOperations(ops)
}

// ReplayMigrations replays a list of migrations and returns the resulting schema.
func ReplayMigrations(migrations []Migration) (*Schema, error) {
	var allOps []ast.Operation
	for _, m := range migrations {
		allOps = append(allOps, m.Operations...)
	}
	return state.ReplayOperations(allOps)
}

// ReplayMigrationsUpTo replays migrations up to and including the specified revision.
func ReplayMigrationsUpTo(migrations []Migration, targetRevision string) (*Schema, error) {
	var allOps []ast.Operation
	for _, m := range migrations {
		allOps = append(allOps, m.Operations...)
		if m.Revision == targetRevision {
			break
		}
	}
	return state.ReplayOperations(allOps)
}
