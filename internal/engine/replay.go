// Package engine provides schema replay functionality for computing schema state at a revision.
package engine

import (
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// ReplayOperations applies a sequence of operations to build the resulting schema state.
// This is used for computing what the schema looks like at a specific migration revision.
func ReplayOperations(ops []ast.Operation) (*Schema, error) {
	schema := NewSchema()

	for _, op := range ops {
		if err := applyOperation(schema, op); err != nil {
			return nil, err
		}
	}

	return schema, nil
}

// applyOperation applies a single operation to the schema state.
func applyOperation(schema *Schema, op ast.Operation) error {
	switch o := op.(type) {
	case *ast.CreateTable:
		return applyCreateTable(schema, o)
	case *ast.DropTable:
		return applyDropTable(schema, o)
	case *ast.RenameTable:
		return applyRenameTable(schema, o)
	case *ast.AddColumn:
		return applyAddColumn(schema, o)
	case *ast.DropColumn:
		return applyDropColumn(schema, o)
	case *ast.RenameColumn:
		return applyRenameColumn(schema, o)
	case *ast.AlterColumn:
		return applyAlterColumn(schema, o)
	case *ast.CreateIndex:
		return applyCreateIndex(schema, o)
	case *ast.DropIndex:
		return applyDropIndex(schema, o)
	case *ast.AddForeignKey:
		return applyAddForeignKey(schema, o)
	case *ast.DropForeignKey:
		return applyDropForeignKey(schema, o)
	case *ast.RawSQL:
		// Raw SQL operations can't be replayed for schema state
		// Just ignore them
		return nil
	default:
		return alerr.New(alerr.ErrSchemaInvalid, "unknown operation type").
			With("type", op.Type().String())
	}
}

// applyCreateTable adds a new table to the schema.
func applyCreateTable(schema *Schema, op *ast.CreateTable) error {
	qualifiedName := op.Namespace + "." + op.Name
	if op.Namespace == "" {
		qualifiedName = op.Name
	}

	if _, exists := schema.Tables[qualifiedName]; exists {
		return alerr.New(alerr.ErrSchemaDuplicate, "table already exists").
			WithTable(op.Namespace, op.Name)
	}

	table := &ast.TableDef{
		Namespace:   op.Namespace,
		Name:        op.Name,
		Columns:     make([]*ast.ColumnDef, len(op.Columns)),
		Indexes:     make([]*ast.IndexDef, 0),
		ForeignKeys: make([]*ast.ForeignKeyDef, 0),
		Checks:      make([]*ast.CheckDef, 0),
	}

	// Copy columns
	for i, col := range op.Columns {
		colCopy := *col
		table.Columns[i] = &colCopy
	}

	// Copy indexes
	if op.Indexes != nil {
		table.Indexes = make([]*ast.IndexDef, len(op.Indexes))
		for i, idx := range op.Indexes {
			idxCopy := *idx
			table.Indexes[i] = &idxCopy
		}
	}

	// Copy foreign keys
	if op.ForeignKeys != nil {
		table.ForeignKeys = make([]*ast.ForeignKeyDef, len(op.ForeignKeys))
		for i, fk := range op.ForeignKeys {
			fkCopy := *fk
			table.ForeignKeys[i] = &fkCopy
		}
	}

	schema.Tables[qualifiedName] = table
	return nil
}

// applyDropTable removes a table from the schema.
func applyDropTable(schema *Schema, op *ast.DropTable) error {
	qualifiedName := op.Namespace + "." + op.Name
	if op.Namespace == "" {
		qualifiedName = op.Name
	}

	if _, exists := schema.Tables[qualifiedName]; !exists && !op.IfExists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Name)
	}

	delete(schema.Tables, qualifiedName)
	return nil
}

// applyRenameTable renames a table in the schema.
func applyRenameTable(schema *Schema, op *ast.RenameTable) error {
	oldQualified := op.Namespace + "." + op.OldName
	newQualified := op.Namespace + "." + op.NewName
	if op.Namespace == "" {
		oldQualified = op.OldName
		newQualified = op.NewName
	}

	table, exists := schema.Tables[oldQualified]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.OldName)
	}

	if _, exists := schema.Tables[newQualified]; exists {
		return alerr.New(alerr.ErrSchemaDuplicate, "target table name already exists").
			WithTable(op.Namespace, op.NewName)
	}

	// Update the table's name and move it
	table.Name = op.NewName
	delete(schema.Tables, oldQualified)
	schema.Tables[newQualified] = table

	return nil
}

// applyAddColumn adds a column to an existing table.
func applyAddColumn(schema *Schema, op *ast.AddColumn) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Check if column already exists
	for _, col := range table.Columns {
		if col.Name == op.Column.Name {
			return alerr.New(alerr.ErrSchemaDuplicate, "column already exists").
				WithTable(op.Namespace, op.Table_).
				WithColumn(op.Column.Name)
		}
	}

	// Add the column
	colCopy := *op.Column
	table.Columns = append(table.Columns, &colCopy)

	return nil
}

// applyDropColumn removes a column from an existing table.
func applyDropColumn(schema *Schema, op *ast.DropColumn) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Find and remove the column
	found := false
	newColumns := make([]*ast.ColumnDef, 0, len(table.Columns)-1)
	for _, col := range table.Columns {
		if col.Name == op.Name {
			found = true
			continue
		}
		newColumns = append(newColumns, col)
	}

	if !found {
		return alerr.New(alerr.ErrSchemaNotFound, "column does not exist").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.Name)
	}

	table.Columns = newColumns
	return nil
}

// applyRenameColumn renames a column in an existing table.
func applyRenameColumn(schema *Schema, op *ast.RenameColumn) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Find the column and rename it
	found := false
	for _, col := range table.Columns {
		if col.Name == op.OldName {
			col.Name = op.NewName
			found = true
			break
		}
	}

	if !found {
		return alerr.New(alerr.ErrSchemaNotFound, "column does not exist").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.OldName)
	}

	// Also update any indexes that reference this column
	for _, idx := range table.Indexes {
		for i, colName := range idx.Columns {
			if colName == op.OldName {
				idx.Columns[i] = op.NewName
			}
		}
	}

	return nil
}

// applyAlterColumn modifies a column's properties.
func applyAlterColumn(schema *Schema, op *ast.AlterColumn) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Find the column
	var col *ast.ColumnDef
	for _, c := range table.Columns {
		if c.Name == op.Name {
			col = c
			break
		}
	}

	if col == nil {
		return alerr.New(alerr.ErrSchemaNotFound, "column does not exist").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.Name)
	}

	// Apply changes
	if op.NewType != "" {
		col.Type = op.NewType
		col.TypeArgs = op.NewTypeArgs
	}
	if op.SetNullable != nil {
		col.Nullable = *op.SetNullable
		col.NullableSet = true
	}
	if op.DropDefault {
		col.Default = nil
		col.DefaultSet = false
	} else if op.SetDefault != nil {
		col.Default = op.SetDefault
		col.DefaultSet = true
	}

	return nil
}

// applyCreateIndex adds an index to an existing table.
func applyCreateIndex(schema *Schema, op *ast.CreateIndex) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Check if index already exists (by name)
	if op.Name != "" && !op.IfNotExists {
		for _, idx := range table.Indexes {
			if idx.Name == op.Name {
				return alerr.New(alerr.ErrSchemaDuplicate, "index already exists").
					WithTable(op.Namespace, op.Table_).
					With("index", op.Name)
			}
		}
	}

	// Add the index
	idx := &ast.IndexDef{
		Name:    op.Name,
		Columns: make([]string, len(op.Columns)),
		Unique:  op.Unique,
	}
	copy(idx.Columns, op.Columns)

	// Auto-generate name if not provided
	if idx.Name == "" {
		idx.Name = "idx_" + table.SQLName() + "_" + strings.Join(op.Columns, "_")
	}

	table.Indexes = append(table.Indexes, idx)
	return nil
}

// applyDropIndex removes an index from an existing table.
func applyDropIndex(schema *Schema, op *ast.DropIndex) error {
	// Find table containing this index
	var table *ast.TableDef
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	if op.Table_ != "" {
		var exists bool
		table, exists = schema.Tables[qualifiedName]
		if !exists && !op.IfExists {
			return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
				WithTable(op.Namespace, op.Table_)
		}
	} else {
		// Search all tables for the index
		for _, t := range schema.Tables {
			for _, idx := range t.Indexes {
				if idx.Name == op.Name {
					table = t
					break
				}
			}
			if table != nil {
				break
			}
		}
	}

	if table == nil {
		if op.IfExists {
			return nil
		}
		return alerr.New(alerr.ErrSchemaNotFound, "index does not exist").
			With("index", op.Name)
	}

	// Remove the index
	found := false
	newIndexes := make([]*ast.IndexDef, 0, len(table.Indexes)-1)
	for _, idx := range table.Indexes {
		if idx.Name == op.Name {
			found = true
			continue
		}
		newIndexes = append(newIndexes, idx)
	}

	if !found && !op.IfExists {
		return alerr.New(alerr.ErrSchemaNotFound, "index does not exist").
			With("index", op.Name)
	}

	table.Indexes = newIndexes
	return nil
}

// applyAddForeignKey adds a foreign key constraint to an existing table.
func applyAddForeignKey(schema *Schema, op *ast.AddForeignKey) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Add the foreign key
	fk := &ast.ForeignKeyDef{
		Name:       op.Name,
		Columns:    make([]string, len(op.Columns)),
		RefTable:   op.RefTable,
		RefColumns: make([]string, len(op.RefColumns)),
		OnDelete:   op.OnDelete,
		OnUpdate:   op.OnUpdate,
	}
	copy(fk.Columns, op.Columns)
	copy(fk.RefColumns, op.RefColumns)

	table.ForeignKeys = append(table.ForeignKeys, fk)
	return nil
}

// applyDropForeignKey removes a foreign key constraint from an existing table.
func applyDropForeignKey(schema *Schema, op *ast.DropForeignKey) error {
	qualifiedName := op.Namespace + "." + op.Table_
	if op.Namespace == "" {
		qualifiedName = op.Table_
	}

	table, exists := schema.Tables[qualifiedName]
	if !exists {
		return alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
			WithTable(op.Namespace, op.Table_)
	}

	// Remove the foreign key
	found := false
	newFKs := make([]*ast.ForeignKeyDef, 0, len(table.ForeignKeys)-1)
	for _, fk := range table.ForeignKeys {
		if fk.Name == op.Name {
			found = true
			continue
		}
		newFKs = append(newFKs, fk)
	}

	if !found {
		// FK might not exist, which is okay for idempotent operations
		return nil
	}

	table.ForeignKeys = newFKs
	return nil
}

// ReplayMigrations replays a list of migrations and returns the resulting schema.
func ReplayMigrations(migrations []Migration) (*Schema, error) {
	var allOps []ast.Operation
	for _, m := range migrations {
		allOps = append(allOps, m.Operations...)
	}
	return ReplayOperations(allOps)
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
	return ReplayOperations(allOps)
}
