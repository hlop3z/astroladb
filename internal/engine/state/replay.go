package state

import (
	"slices"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// tableNotFoundErr creates a standard "table does not exist" error with fuzzy suggestions.
func tableNotFoundErr(schema *Schema, ns, name string) *alerr.Error {
	err := alerr.New(alerr.ErrSchemaNotFound, "table does not exist").
		WithTable(ns, name)
	names := make([]string, 0, len(schema.Tables))
	for k := range schema.Tables {
		names = append(names, k)
	}
	if suggestion := alerr.SuggestSimilar(qualifyName(ns, name), names); suggestion != "" {
		err.WithHelp(suggestion)
	}
	return err
}

// columnNotFoundErr creates a standard "column does not exist" error with fuzzy suggestions.
func columnNotFoundErr(table *ast.TableDef, ns, tableName, colName string) *alerr.Error {
	err := alerr.New(alerr.ErrSchemaNotFound, "column does not exist").
		WithTable(ns, tableName).
		WithColumn(colName)
	names := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		names = append(names, col.Name)
	}
	if suggestion := alerr.SuggestSimilar(colName, names); suggestion != "" {
		err.WithHelp(suggestion)
	}
	return err
}

// qualifyName returns the dot-separated qualified name (ns.name or name).
func qualifyName(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "." + name
}

// lookupTable finds a table by namespace and name, returning a not-found error if missing.
func lookupTable(schema *Schema, ns, name string) (*ast.TableDef, string, error) {
	qn := qualifyName(ns, name)
	table, exists := schema.Tables[qn]
	if !exists {
		return nil, qn, tableNotFoundErr(schema, ns, name)
	}
	return table, qn, nil
}

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
	qualifiedName := op.QualifiedName()

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

	// Copy indexes (auto-generate names if empty)
	if op.Indexes != nil {
		table.Indexes = make([]*ast.IndexDef, len(op.Indexes))
		for i, idx := range op.Indexes {
			idxCopy := *idx
			// Auto-generate name if not provided
			if idxCopy.Name == "" {
				idxCopy.Name = "idx_" + table.FullName() + "_" + strings.Join(idxCopy.Columns, "_")
			}
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
	qualifiedName := op.QualifiedName()

	if _, exists := schema.Tables[qualifiedName]; !exists && !op.IfExists {
		return tableNotFoundErr(schema, op.Namespace, op.Name)
	}

	delete(schema.Tables, qualifiedName)
	return nil
}

// applyRenameTable renames a table in the schema.
func applyRenameTable(schema *Schema, op *ast.RenameTable) error {
	oldQualified := qualifyName(op.Namespace, op.OldName)
	newQualified := qualifyName(op.Namespace, op.NewName)

	table, exists := schema.Tables[oldQualified]
	if !exists {
		return tableNotFoundErr(schema, op.Namespace, op.OldName)
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
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
	}

	// Check if column already exists
	if table.HasColumn(op.Column.Name) {
		return alerr.New(alerr.ErrSchemaDuplicate, "column already exists").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.Column.Name)
	}

	// Add the column
	colCopy := *op.Column
	table.Columns = append(table.Columns, &colCopy)

	return nil
}

// applyDropColumn removes a column from an existing table.
func applyDropColumn(schema *Schema, op *ast.DropColumn) error {
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
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
		return columnNotFoundErr(table, op.Namespace, op.Table_, op.Name)
	}

	table.Columns = newColumns
	return nil
}

// applyRenameColumn renames a column in an existing table.
func applyRenameColumn(schema *Schema, op *ast.RenameColumn) error {
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
	}

	// Find the column and rename it
	col := table.GetColumn(op.OldName)
	if col == nil {
		return columnNotFoundErr(table, op.Namespace, op.Table_, op.OldName)
	}
	col.Name = op.NewName

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
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
	}

	// Find the column
	col := table.GetColumn(op.Name)
	if col == nil {
		return columnNotFoundErr(table, op.Namespace, op.Table_, op.Name)
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
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
	}

	// Auto-generate name if not provided (do this first to enable duplicate detection)
	indexName := op.Name
	if indexName == "" {
		indexName = "idx_" + table.FullName() + "_" + strings.Join(op.Columns, "_")
	}

	// Check if index already exists (by name or by columns)
	for _, idx := range table.Indexes {
		// Check by explicit name - if user explicitly named both indexes the same, that's an error
		// (unless IfNotExists is set)
		if op.Name != "" && idx.Name == op.Name {
			if op.IfNotExists {
				return nil // Silently skip
			}
			return alerr.New(alerr.ErrSchemaDuplicate, "index already exists").
				WithTable(op.Namespace, op.Table_).
				With("index", op.Name)
		}
		// Check by columns - silently skip duplicates to handle cases where:
		// 1. belongs_to creates an index inside CreateTable (with auto-generated name)
		// 2. A separate CreateIndex operation tries to create the same index
		// This prevents duplicate indexes on the same columns
		if slices.Equal(idx.Columns, op.Columns) {
			return nil
		}
	}

	// Add the index
	idx := &ast.IndexDef{
		Name:    indexName,
		Columns: make([]string, len(op.Columns)),
		Unique:  op.Unique,
	}
	copy(idx.Columns, op.Columns)

	table.Indexes = append(table.Indexes, idx)
	return nil
}

// applyDropIndex removes an index from an existing table.
func applyDropIndex(schema *Schema, op *ast.DropIndex) error {
	// Find table containing this index
	var table *ast.TableDef

	if op.Table_ != "" {
		t, _, err := lookupTable(schema, op.Namespace, op.Table_)
		if err != nil && !op.IfExists {
			return err
		}
		table = t
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
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
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
	table, _, err := lookupTable(schema, op.Namespace, op.Table_)
	if err != nil {
		return err
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
