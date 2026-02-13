package state

import (
	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// validateTableDef validates a single table definition.
func validateTableDef(t *ast.TableDef) error {
	if t.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required")
	}

	if len(t.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, "table must have at least one column").
			WithTable(t.Namespace, t.Name)
	}

	// Check for duplicate columns
	colSeen := make(map[string]bool)
	for _, col := range t.Columns {
		if col.Name == "" {
			return alerr.New(alerr.ErrSchemaInvalid, "column name is required").
				WithTable(t.Namespace, t.Name)
		}
		if colSeen[col.Name] {
			return alerr.New(alerr.ErrSchemaDuplicate, "duplicate column name").
				WithTable(t.Namespace, t.Name).
				WithColumn(col.Name)
		}
		colSeen[col.Name] = true
	}

	return nil
}
