// Package engine provides migration planning and execution.
package engine

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// RenameCandidate represents a potential rename operation detected from diff.
type RenameCandidate struct {
	Type      string // "column" or "table"
	Table     string // Table name (for column renames)
	Namespace string // Namespace (for both)
	OldName   string // Old name
	NewName   string // New name
	ColType   string // Column type (for column renames, used for matching)
}

// DetectRenames analyzes diff operations to find potential renames.
// It looks for patterns like:
//   - Drop column + Add column (same table, same type) -> possible column rename
//   - Drop table + Create table (similar structure) -> possible table rename
//
// Returns candidates that the user should confirm.
func DetectRenames(ops []ast.Operation) []RenameCandidate {
	var candidates []RenameCandidate

	// Collect drops and adds by table
	type colInfo struct {
		op        ast.Operation
		namespace string
		table     string
		name      string
		colType   string
		index     int
	}

	var dropColumns []colInfo
	var addColumns []colInfo
	var dropTables []colInfo
	var createTables []colInfo

	for i, op := range ops {
		switch o := op.(type) {
		case *ast.DropColumn:
			dropColumns = append(dropColumns, colInfo{
				op:        op,
				namespace: o.Namespace,
				table:     o.Table_,
				name:      o.Name,
				index:     i,
			})
		case *ast.AddColumn:
			addColumns = append(addColumns, colInfo{
				op:        op,
				namespace: o.Namespace,
				table:     o.Table_,
				name:      o.Column.Name,
				colType:   o.Column.Type,
				index:     i,
			})
		case *ast.DropTable:
			dropTables = append(dropTables, colInfo{
				op:        op,
				namespace: o.Namespace,
				name:      o.Name,
				index:     i,
			})
		case *ast.CreateTable:
			createTables = append(createTables, colInfo{
				op:        op,
				namespace: o.Namespace,
				name:      o.Name,
				index:     i,
			})
		}
	}

	// Match drop columns with add columns (same table, need type from introspection)
	// For now, match by same table - we'll need to get type info from somewhere
	usedDrops := make(map[int]bool)
	usedAdds := make(map[int]bool)

	for _, drop := range dropColumns {
		for _, add := range addColumns {
			if usedDrops[drop.index] || usedAdds[add.index] {
				continue
			}
			// Same table and namespace
			if drop.namespace == add.namespace && drop.table == add.table {
				// Different names (otherwise it's not a rename)
				if drop.name != add.name {
					candidates = append(candidates, RenameCandidate{
						Type:      "column",
						Table:     drop.table,
						Namespace: drop.namespace,
						OldName:   drop.name,
						NewName:   add.name,
						ColType:   add.colType,
					})
					usedDrops[drop.index] = true
					usedAdds[add.index] = true
					break // One drop matches one add
				}
			}
		}
	}

	// Match drop tables with create tables (same namespace)
	usedDropTables := make(map[int]bool)
	usedCreateTables := make(map[int]bool)

	for _, drop := range dropTables {
		for _, create := range createTables {
			if usedDropTables[drop.index] || usedCreateTables[create.index] {
				continue
			}
			// Same namespace, different names
			if drop.namespace == create.namespace && drop.name != create.name {
				candidates = append(candidates, RenameCandidate{
					Type:      "table",
					Namespace: drop.namespace,
					OldName:   drop.name,
					NewName:   create.name,
				})
				usedDropTables[drop.index] = true
				usedCreateTables[create.index] = true
				break
			}
		}
	}

	return candidates
}

// ApplyRenames transforms operations by replacing drop+add pairs with rename operations.
// It takes the original operations and confirmed renames, returns modified operations.
func ApplyRenames(ops []ast.Operation, confirmed []RenameCandidate) []ast.Operation {
	if len(confirmed) == 0 {
		return ops
	}

	// Build lookup maps for quick matching
	columnRenames := make(map[string]RenameCandidate) // "ns.table.oldcol" -> rename
	tableRenames := make(map[string]RenameCandidate)  // "ns.oldtable" -> rename

	for _, r := range confirmed {
		if r.Type == "column" {
			key := r.Namespace + "." + r.Table + "." + r.OldName
			columnRenames[key] = r
		} else if r.Type == "table" {
			key := r.Namespace + "." + r.OldName
			tableRenames[key] = r
		}
	}

	var result []ast.Operation
	skipIndices := make(map[int]bool)

	// First pass: identify operations to skip (they'll be replaced by renames)
	for i, op := range ops {
		switch o := op.(type) {
		case *ast.DropColumn:
			key := o.Namespace + "." + o.Table_ + "." + o.Name
			if _, ok := columnRenames[key]; ok {
				skipIndices[i] = true
			}
		case *ast.AddColumn:
			// Find if this add corresponds to a rename
			for _, r := range confirmed {
				if r.Type == "column" &&
					r.Namespace == o.Namespace &&
					r.Table == o.Table_ &&
					r.NewName == o.Column.Name {
					skipIndices[i] = true
					break
				}
			}
		case *ast.DropTable:
			key := o.Namespace + "." + o.Name
			if _, ok := tableRenames[key]; ok {
				skipIndices[i] = true
			}
		case *ast.CreateTable:
			// Find if this create corresponds to a rename
			for _, r := range confirmed {
				if r.Type == "table" &&
					r.Namespace == o.Namespace &&
					r.NewName == o.Name {
					skipIndices[i] = true
					break
				}
			}
		}
	}

	// Second pass: build result with renames injected
	addedRenames := make(map[string]bool)

	for i, op := range ops {
		if skipIndices[i] {
			// Check if we should inject a rename operation here
			switch o := op.(type) {
			case *ast.DropColumn:
				key := o.Namespace + "." + o.Table_ + "." + o.Name
				if r, ok := columnRenames[key]; ok && !addedRenames[key] {
					result = append(result, &ast.RenameColumn{
						TableRef: ast.TableRef{
							Namespace: r.Namespace,
							Table_:    r.Table,
						},
						OldName: r.OldName,
						NewName: r.NewName,
					})
					addedRenames[key] = true
				}
			case *ast.DropTable:
				key := o.Namespace + "." + o.Name
				if r, ok := tableRenames[key]; ok && !addedRenames[key] {
					result = append(result, &ast.RenameTable{
						Namespace: r.Namespace,
						OldName:   r.OldName,
						NewName:   r.NewName,
					})
					addedRenames[key] = true
				}
			}
			continue
		}
		result = append(result, op)
	}

	return result
}
