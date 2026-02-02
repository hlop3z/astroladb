package engine

import (
	"sort"

	"github.com/hlop3z/astroladb/internal/ast"
)

// GenerateBaseline converts a set of table definitions into create operations
// suitable for a baseline migration. Operations are ordered by FK dependency
// so that referenced tables are created before referencing tables.
func GenerateBaseline(tables []*ast.TableDef) []ast.Operation {
	if len(tables) == 0 {
		return nil
	}

	sorted := sortTablesByDependency(tables)

	var ops []ast.Operation
	for _, table := range sorted {
		ct := &ast.CreateTable{
			TableOp: ast.TableOp{
				Namespace: table.Namespace,
				Name:      table.Name,
			},
			Columns:     table.Columns,
			ForeignKeys: table.ForeignKeys,
		}
		ops = append(ops, ct)

		for _, idx := range table.Indexes {
			ci := &ast.CreateIndex{
				TableRef: ast.TableRef{
					Namespace: table.Namespace,
					Table_:    table.Name,
				},
				Columns: idx.Columns,
				Unique:  idx.Unique,
				Name:    idx.Name,
			}
			ops = append(ops, ci)
		}
	}

	return ops
}

// sortTablesByDependency sorts tables so referenced tables come before
// referencing tables using topological sort (Kahn's algorithm).
func sortTablesByDependency(tables []*ast.TableDef) []*ast.TableDef {
	if len(tables) <= 1 {
		return tables
	}

	type node struct {
		table *ast.TableDef
		deps  map[string]bool // full names this table depends on
	}

	nodes := make(map[string]*node)
	for _, t := range tables {
		fn := t.Namespace + "." + t.Name
		n := &node{table: t, deps: make(map[string]bool)}

		for _, fk := range t.ForeignKeys {
			if fk.RefTable != "" && fk.RefTable != fn {
				n.deps[fk.RefTable] = true
			}
		}
		for _, col := range t.Columns {
			if col.Reference != nil && col.Reference.Table != "" && col.Reference.Table != fn {
				n.deps[col.Reference.Table] = true
			}
		}

		nodes[fn] = n
	}

	// Compute in-degree (count of unsatisfied dependencies within our set)
	inDeg := make(map[string]int)
	for fn, n := range nodes {
		count := 0
		for dep := range n.deps {
			if _, ok := nodes[dep]; ok {
				count++
			}
		}
		inDeg[fn] = count
	}

	// Seed queue with zero-dependency tables
	var queue []string
	for fn, deg := range inDeg {
		if deg == 0 {
			queue = append(queue, fn)
		}
	}
	sort.Strings(queue)

	var sorted []*ast.TableDef
	for len(queue) > 0 {
		fn := queue[0]
		queue = queue[1:]
		sorted = append(sorted, nodes[fn].table)

		// Reduce in-degree for tables that depend on this one
		for otherFn, otherNode := range nodes {
			if otherNode.deps[fn] {
				inDeg[otherFn]--
				if inDeg[otherFn] == 0 {
					queue = append(queue, otherFn)
					sort.Strings(queue)
				}
			}
		}
	}

	// Append any remaining (circular deps) in alphabetical order
	if len(sorted) < len(tables) {
		seen := make(map[string]bool)
		for _, t := range sorted {
			seen[t.Namespace+"."+t.Name] = true
		}
		var remaining []string
		for fn := range nodes {
			if !seen[fn] {
				remaining = append(remaining, fn)
			}
		}
		sort.Strings(remaining)
		for _, fn := range remaining {
			sorted = append(sorted, nodes[fn].table)
		}
	}

	return sorted
}
