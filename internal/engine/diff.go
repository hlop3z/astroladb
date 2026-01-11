package engine

import (
	"reflect"
	"sort"

	"github.com/hlop3z/astroladb/internal/ast"
)

// Diff compares two schemas and returns the operations needed to transform
// the old schema into the new schema.
//
// Algorithm:
//  1. Tables to create (in new, not in old)
//  2. Tables to drop (in old, not in new)
//  3. Tables to alter (in both) - diff columns, indexes, FKs
//  4. Sort by dependency (FKs must be created after referenced tables)
func Diff(old, new *Schema) ([]ast.Operation, error) {
	var ops []ast.Operation

	// Handle nil schemas
	if old == nil {
		old = NewSchema()
	}
	if new == nil {
		new = NewSchema()
	}

	// 1. Tables to create (in new, not in old)
	createOps := diffCreateTables(old, new)
	ops = append(ops, createOps...)

	// 2. Tables to drop (in old, not in new)
	dropOps := diffDropTables(old, new)
	ops = append(ops, dropOps...)

	// 3. Tables to alter (in both)
	alterOps := diffAlterTables(old, new)
	ops = append(ops, alterOps...)

	// 4. Sort by dependency
	ops = sortByDependency(ops, new)

	return ops, nil
}

// diffCreateTables returns CreateTable operations for tables in new but not in old.
func diffCreateTables(old, new *Schema) []ast.Operation {
	var ops []ast.Operation

	for name, newTable := range new.Tables {
		if _, exists := old.Tables[name]; !exists {
			ops = append(ops, &ast.CreateTable{
				TableOp: ast.TableOp{
					Namespace: newTable.Namespace,
					Name:      newTable.Name,
				},
				Columns:     newTable.Columns,
				Indexes:     tableIndexesToIndexDefs(newTable),
				ForeignKeys: tableColumnsToFKDefs(newTable),
			})
		}
	}

	// Sort for deterministic output
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Table() < ops[j].Table()
	})

	return ops
}

// diffDropTables returns DropTable operations for tables in old but not in new.
func diffDropTables(old, new *Schema) []ast.Operation {
	var ops []ast.Operation

	for name, oldTable := range old.Tables {
		if _, exists := new.Tables[name]; !exists {
			ops = append(ops, &ast.DropTable{
				TableOp: ast.TableOp{
					Namespace: oldTable.Namespace,
					Name:      oldTable.Name,
				},
			})
		}
	}

	// Sort for deterministic output (reversed for drop order)
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Table() > ops[j].Table()
	})

	return ops
}

// diffAlterTables returns operations to transform old tables into new tables.
func diffAlterTables(old, new *Schema) []ast.Operation {
	var ops []ast.Operation

	for name, newTable := range new.Tables {
		oldTable, exists := old.Tables[name]
		if !exists {
			continue // Will be handled by create
		}

		// Diff columns
		columnOps := diffColumns(oldTable, newTable)
		ops = append(ops, columnOps...)

		// Diff indexes
		indexOps := diffIndexes(oldTable, newTable)
		ops = append(ops, indexOps...)

		// Diff foreign keys
		fkOps := diffForeignKeys(oldTable, newTable)
		ops = append(ops, fkOps...)

		// Diff CHECK constraints
		checkOps := diffCheckConstraints(oldTable, newTable)
		ops = append(ops, checkOps...)
	}

	return ops
}

// diffColumns compares columns between old and new tables.
func diffColumns(oldTable, newTable *ast.TableDef) []ast.Operation {
	var ops []ast.Operation

	oldCols := ToMap(oldTable.Columns, func(c *ast.ColumnDef) string { return c.Name })
	newCols := ToMap(newTable.Columns, func(c *ast.ColumnDef) string { return c.Name })

	// Columns to add (in new, not in old)
	for name, newCol := range newCols {
		if _, exists := oldCols[name]; !exists {
			ops = append(ops, &ast.AddColumn{
				TableRef: ast.TableRef{
					Namespace: newTable.Namespace,
					Table_:    newTable.Name,
				},
				Column: newCol,
			})
		}
	}

	// Columns to drop (in old, not in new)
	for name, oldCol := range oldCols {
		if _, exists := newCols[name]; !exists {
			ops = append(ops, &ast.DropColumn{
				TableRef: ast.TableRef{
					Namespace: oldTable.Namespace,
					Table_:    oldTable.Name,
				},
				Name: oldCol.Name,
			})
		}
	}

	// Columns to alter (in both, but changed)
	for name, newCol := range newCols {
		oldCol, exists := oldCols[name]
		if !exists {
			continue // Will be handled by add
		}

		alterOps := diffColumnDef(oldTable, oldCol, newCol)
		ops = append(ops, alterOps...)
	}

	// Sort for deterministic output
	sort.SliceStable(ops, func(i, j int) bool {
		// AddColumn before DropColumn before AlterColumn
		ti, tj := ops[i].Type(), ops[j].Type()
		if ti != tj {
			return ti < tj
		}
		// Then by table name
		return ops[i].Table() < ops[j].Table()
	})

	return ops
}

// diffColumnDef compares two column definitions and returns alter operations if needed.
func diffColumnDef(table *ast.TableDef, oldCol, newCol *ast.ColumnDef) []ast.Operation {
	var ops []ast.Operation

	// Check if column needs modification
	needsAlter := false
	alterOp := &ast.AlterColumn{
		TableRef: ast.TableRef{
			Namespace: table.Namespace,
			Table_:    table.Name,
		},
		Name: newCol.Name,
	}

	// Type changed
	if oldCol.Type != newCol.Type || !reflect.DeepEqual(oldCol.TypeArgs, newCol.TypeArgs) {
		needsAlter = true
		alterOp.NewType = newCol.Type
		alterOp.NewTypeArgs = newCol.TypeArgs
	}

	// Nullable changed
	if oldCol.Nullable != newCol.Nullable {
		needsAlter = true
		nullable := newCol.Nullable
		alterOp.SetNullable = &nullable
	}

	// Default changed
	if !defaultsEqual(oldCol.Default, newCol.Default) {
		needsAlter = true
		if newCol.Default == nil && newCol.DefaultSet == false {
			alterOp.DropDefault = true
		} else {
			alterOp.SetDefault = newCol.Default
		}
	}

	if needsAlter {
		ops = append(ops, alterOp)
	}

	return ops
}

// defaultsEqual compares two default values for equality.
func defaultsEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

// diffIndexes compares indexes between old and new tables.
func diffIndexes(oldTable, newTable *ast.TableDef) []ast.Operation {
	var ops []ast.Operation

	oldIdxs := ToMap(oldTable.Indexes, func(idx *ast.IndexDef) string { return indexKey(oldTable, idx) })
	newIdxs := ToMap(newTable.Indexes, func(idx *ast.IndexDef) string { return indexKey(newTable, idx) })

	// Indexes to create (in new, not in old)
	for key, newIdx := range newIdxs {
		if _, exists := oldIdxs[key]; !exists {
			ops = append(ops, &ast.CreateIndex{
				TableRef: ast.TableRef{
					Namespace: newTable.Namespace,
					Table_:    newTable.Name,
				},
				Name:    newIdx.Name,
				Columns: newIdx.Columns,
				Unique:  newIdx.Unique,
			})
		}
	}

	// Indexes to drop (in old, not in new)
	for key, oldIdx := range oldIdxs {
		if _, exists := newIdxs[key]; !exists {
			ops = append(ops, &ast.DropIndex{
				TableRef: ast.TableRef{
					Namespace: oldTable.Namespace,
					Table_:    oldTable.Name,
				},
				Name: oldIdx.Name,
			})
		}
	}

	return ops
}

// indexKey generates a unique key for an index based on its columns and uniqueness.
func indexKey(table *ast.TableDef, idx *ast.IndexDef) string {
	// Use column names as key since indexes on same columns should match
	key := table.QualifiedName() + ":"
	if idx.Unique {
		key += "unique:"
	}
	for i, col := range idx.Columns {
		if i > 0 {
			key += ","
		}
		key += col
	}
	return key
}

// diffForeignKeys compares foreign keys between old and new tables.
func diffForeignKeys(oldTable, newTable *ast.TableDef) []ast.Operation {
	var ops []ast.Operation

	// Extract FKs from columns
	oldFKs := extractFKsFromColumns(oldTable)
	newFKs := extractFKsFromColumns(newTable)

	// FKs to add (in new, not in old)
	for key, fk := range newFKs {
		if _, exists := oldFKs[key]; !exists {
			ops = append(ops, &ast.AddForeignKey{
				TableRef: ast.TableRef{
					Namespace: newTable.Namespace,
					Table_:    newTable.Name,
				},
				Name:       fk.Name,
				Columns:    fk.Columns,
				RefTable:   fk.RefTable,
				RefColumns: fk.RefColumns,
				OnDelete:   fk.OnDelete,
				OnUpdate:   fk.OnUpdate,
			})
		}
	}

	// FKs to drop (in old, not in new)
	for key, fk := range oldFKs {
		if _, exists := newFKs[key]; !exists {
			ops = append(ops, &ast.DropForeignKey{
				TableRef: ast.TableRef{
					Namespace: oldTable.Namespace,
					Table_:    oldTable.Name,
				},
				Name: fk.Name,
			})
		}
	}

	return ops
}

// extractFKsFromColumns extracts foreign key definitions from column references.
func extractFKsFromColumns(table *ast.TableDef) map[string]*fkInfo {
	fks := make(map[string]*fkInfo)

	for _, col := range table.Columns {
		if col.Reference == nil {
			continue
		}

		// Generate FK name
		fkName := "fk_" + table.FullName() + "_" + col.Name

		// Create key for deduplication
		key := col.Name + "->" + col.Reference.Table

		fks[key] = &fkInfo{
			Name:       fkName,
			Columns:    []string{col.Name},
			RefTable:   col.Reference.Table,
			RefColumns: []string{col.Reference.TargetColumn()},
			OnDelete:   col.Reference.OnDelete,
			OnUpdate:   col.Reference.OnUpdate,
		}
	}

	return fks
}

// fkInfo holds foreign key information for diffing.
type fkInfo struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
}

// diffCheckConstraints compares CHECK constraints between old and new tables.
func diffCheckConstraints(oldTable, newTable *ast.TableDef) []ast.Operation {
	var ops []ast.Operation

	oldChecks := ToMap(oldTable.Checks, func(c *ast.CheckDef) string { return checkKey(c) })
	newChecks := ToMap(newTable.Checks, func(c *ast.CheckDef) string { return checkKey(c) })

	// CHECKs to add (in new, not in old)
	for key, newCheck := range newChecks {
		if _, exists := oldChecks[key]; !exists {
			// Generate name if not provided
			name := newCheck.Name
			if name == "" {
				name = "chk_" + newTable.FullName() + "_" + sanitizeCheckName(newCheck.Expression)
			}
			ops = append(ops, &ast.AddCheck{
				TableRef: ast.TableRef{
					Namespace: newTable.Namespace,
					Table_:    newTable.Name,
				},
				Name:       name,
				Expression: newCheck.Expression,
			})
		}
	}

	// CHECKs to drop (in old, not in new)
	for key, oldCheck := range oldChecks {
		if _, exists := newChecks[key]; !exists {
			// Use existing name or generate one
			name := oldCheck.Name
			if name == "" {
				name = "chk_" + oldTable.FullName() + "_" + sanitizeCheckName(oldCheck.Expression)
			}
			ops = append(ops, &ast.DropCheck{
				TableRef: ast.TableRef{
					Namespace: oldTable.Namespace,
					Table_:    oldTable.Name,
				},
				Name: name,
			})
		}
	}

	return ops
}

// checkKey generates a unique key for a CHECK constraint based on its expression.
// We use the expression as the key since that defines the constraint's behavior.
func checkKey(c *ast.CheckDef) string {
	return c.Expression
}

// sanitizeCheckName creates a safe constraint name suffix from an expression.
func sanitizeCheckName(expr string) string {
	// Simple sanitization - take first 20 chars, replace non-alphanumeric with underscore
	result := make([]byte, 0, 20)
	for i := 0; i < len(expr) && len(result) < 20; i++ {
		c := expr[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else if len(result) > 0 && result[len(result)-1] != '_' {
			result = append(result, '_')
		}
	}
	// Trim trailing underscore
	if len(result) > 0 && result[len(result)-1] == '_' {
		result = result[:len(result)-1]
	}
	if len(result) == 0 {
		return "constraint"
	}
	return string(result)
}

// sortByDependency sorts operations so that dependencies are satisfied.
// Foreign keys must be created after their referenced tables exist.
func sortByDependency(ops []ast.Operation, schema *Schema) []ast.Operation {
	if len(ops) == 0 {
		return ops
	}

	// Group operations by type
	var createTables []*ast.CreateTable
	var dropTables []*ast.DropTable
	var addColumns []*ast.AddColumn
	var dropColumns []*ast.DropColumn
	var alterColumns []*ast.AlterColumn
	var createIndexes []*ast.CreateIndex
	var dropIndexes []*ast.DropIndex
	var addFKs []*ast.AddForeignKey
	var dropFKs []*ast.DropForeignKey
	var addChecks []*ast.AddCheck
	var dropChecks []*ast.DropCheck
	var rawSQL []*ast.RawSQL

	for _, op := range ops {
		switch o := op.(type) {
		case *ast.CreateTable:
			createTables = append(createTables, o)
		case *ast.DropTable:
			dropTables = append(dropTables, o)
		case *ast.AddColumn:
			addColumns = append(addColumns, o)
		case *ast.DropColumn:
			dropColumns = append(dropColumns, o)
		case *ast.AlterColumn:
			alterColumns = append(alterColumns, o)
		case *ast.CreateIndex:
			createIndexes = append(createIndexes, o)
		case *ast.DropIndex:
			dropIndexes = append(dropIndexes, o)
		case *ast.AddForeignKey:
			addFKs = append(addFKs, o)
		case *ast.DropForeignKey:
			dropFKs = append(dropFKs, o)
		case *ast.AddCheck:
			addChecks = append(addChecks, o)
		case *ast.DropCheck:
			dropChecks = append(dropChecks, o)
		case *ast.RawSQL:
			rawSQL = append(rawSQL, o)
		}
	}

	// Sort create tables by dependency
	createTables = sortCreateTablesByDependency(createTables, schema)

	// Sort drop tables in reverse dependency order
	dropTables = sortDropTablesByDependency(dropTables)

	// Build final ordered list
	var sorted []ast.Operation

	// 1. Drop foreign keys first (to allow dropping referenced columns/tables)
	for _, op := range dropFKs {
		sorted = append(sorted, op)
	}

	// 2. Drop CHECK constraints (before dropping columns they reference)
	for _, op := range dropChecks {
		sorted = append(sorted, op)
	}

	// 3. Drop indexes
	for _, op := range dropIndexes {
		sorted = append(sorted, op)
	}

	// 4. Drop columns
	for _, op := range dropColumns {
		sorted = append(sorted, op)
	}

	// 5. Drop tables (dependents first)
	for _, op := range dropTables {
		sorted = append(sorted, op)
	}

	// 6. Create tables (dependencies first)
	for _, op := range createTables {
		sorted = append(sorted, op)
	}

	// 7. Add columns
	for _, op := range addColumns {
		sorted = append(sorted, op)
	}

	// 8. Alter columns
	for _, op := range alterColumns {
		sorted = append(sorted, op)
	}

	// 9. Create indexes
	for _, op := range createIndexes {
		sorted = append(sorted, op)
	}

	// 10. Add CHECK constraints (after columns exist)
	for _, op := range addChecks {
		sorted = append(sorted, op)
	}

	// 11. Add foreign keys last (referenced tables must exist)
	for _, op := range addFKs {
		sorted = append(sorted, op)
	}

	// 12. Raw SQL at the end
	for _, op := range rawSQL {
		sorted = append(sorted, op)
	}

	return sorted
}

// sortCreateTablesByDependency sorts CreateTable operations so dependencies come first.
func sortCreateTablesByDependency(ops []*ast.CreateTable, schema *Schema) []*ast.CreateTable {
	if len(ops) <= 1 {
		return ops
	}

	// Build nodes for topological sort
	nodes := make([]*createTableSortNode, len(ops))
	for i, op := range ops {
		nodes[i] = &createTableSortNode{
			table: op.Table(),
			deps:  getCreateTableDependencies(op, schema),
			op:    op,
		}
	}

	// Perform topological sort
	sorted, err := TopoSort(nodes)
	if err != nil {
		// Circular dependency - return original order (shouldn't happen with validated schema)
		return ops
	}

	// Extract sorted operations
	result := make([]*ast.CreateTable, len(sorted))
	for i, node := range sorted {
		result[i] = node.op
	}
	return result
}

// createTableSortNode wraps a CreateTable operation for topological sorting.
type createTableSortNode struct {
	table string
	deps  []string
	op    *ast.CreateTable
}

func (n *createTableSortNode) ID() string             { return n.table }
func (n *createTableSortNode) Dependencies() []string { return n.deps }

// getCreateTableDependencies returns tables that a CreateTable operation depends on.
func getCreateTableDependencies(op *ast.CreateTable, schema *Schema) []string {
	var deps []string
	seen := make(map[string]bool)

	for _, col := range op.Columns {
		if col.Reference == nil {
			continue
		}

		refTable := col.Reference.Table
		// Try to resolve the reference
		ns, table, _ := parseSimpleRef(refTable)
		if ns == "" && op.Namespace != "" {
			ns = op.Namespace
		}

		qualified := ns + "_" + table // SQL name format
		if !seen[qualified] && qualified != op.Table() {
			seen[qualified] = true
			deps = append(deps, qualified)
		}
	}

	sort.Strings(deps)
	return deps
}

// sortDropTablesByDependency sorts DropTable operations so dependents come first.
func sortDropTablesByDependency(ops []*ast.DropTable) []*ast.DropTable {
	// For now, just sort alphabetically in reverse
	// A proper implementation would analyze the old schema's dependencies
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Table() > ops[j].Table()
	})
	return ops
}

// parseSimpleRef parses a simple "ns.table" or "table" reference.
func parseSimpleRef(ref string) (ns, table string, isRelative bool) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			if i == 0 {
				// ".table" - relative
				return "", ref[1:], true
			}
			return ref[:i], ref[i+1:], false
		}
	}
	// No dot - just table name
	return "", ref, false
}

// tableIndexesToIndexDefs converts a table's indexes to IndexDef slice.
func tableIndexesToIndexDefs(table *ast.TableDef) []*ast.IndexDef {
	if table.Indexes == nil {
		return nil
	}
	return table.Indexes
}

// tableColumnsToFKDefs extracts foreign key definitions from columns.
func tableColumnsToFKDefs(table *ast.TableDef) []*ast.ForeignKeyDef {
	var fks []*ast.ForeignKeyDef

	for _, col := range table.Columns {
		if col.Reference == nil {
			continue
		}

		fks = append(fks, &ast.ForeignKeyDef{
			Name:       "fk_" + table.FullName() + "_" + col.Name,
			Columns:    []string{col.Name},
			RefTable:   col.Reference.Table,
			RefColumns: []string{col.Reference.TargetColumn()},
			OnDelete:   col.Reference.OnDelete,
			OnUpdate:   col.Reference.OnUpdate,
		})
	}

	return fks
}

// DiffSummary provides a human-readable summary of the diff.
type DiffSummary struct {
	TablesToCreate int
	TablesToDrop   int
	ColumnsToAdd   int
	ColumnsToDrop  int
	ColumnsToAlter int
	IndexesToAdd   int
	IndexesToDrop  int
	FKsToAdd       int
	FKsToDrop      int
	ChecksToAdd    int
	ChecksToDrop   int
	TotalOps       int
}

// Summarize returns a summary of the operations in a diff.
func Summarize(ops []ast.Operation) DiffSummary {
	s := DiffSummary{TotalOps: len(ops)}

	for _, op := range ops {
		switch op.Type() {
		case ast.OpCreateTable:
			s.TablesToCreate++
		case ast.OpDropTable:
			s.TablesToDrop++
		case ast.OpAddColumn:
			s.ColumnsToAdd++
		case ast.OpDropColumn:
			s.ColumnsToDrop++
		case ast.OpAlterColumn:
			s.ColumnsToAlter++
		case ast.OpCreateIndex:
			s.IndexesToAdd++
		case ast.OpDropIndex:
			s.IndexesToDrop++
		case ast.OpAddForeignKey:
			s.FKsToAdd++
		case ast.OpDropForeignKey:
			s.FKsToDrop++
		case ast.OpAddCheck:
			s.ChecksToAdd++
		case ast.OpDropCheck:
			s.ChecksToDrop++
		}
	}

	return s
}

// HasChanges returns true if there are any operations in the diff.
func HasChanges(ops []ast.Operation) bool {
	return len(ops) > 0
}
