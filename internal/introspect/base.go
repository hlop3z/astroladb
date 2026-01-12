// Package introspect provides database schema introspection.
// This file contains shared helper functions used by all introspector implementations.
package introspect

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// TableLister provides the ability to list tables in a database.
type TableLister interface {
	listTables(ctx context.Context) ([]string, error)
}

// TableIntrospector provides the ability to introspect a single table.
type TableIntrospector interface {
	IntrospectTable(ctx context.Context, tableName string) (*ast.TableDef, error)
}

// TableIntrospectorWithMapping extends TableIntrospector with mapping support.
type TableIntrospectorWithMapping interface {
	TableIntrospector
	introspectTableWithMapping(ctx context.Context, tableName string, mapping TableNameMapping) (*ast.TableDef, error)
}

// ColumnIntrospector provides the ability to introspect columns of a table.
type ColumnIntrospector interface {
	introspectColumns(ctx context.Context, tableName string) ([]*ast.ColumnDef, error)
}

// IndexIntrospector provides the ability to introspect indexes of a table.
type IndexIntrospector interface {
	introspectIndexes(ctx context.Context, tableName string) ([]*ast.IndexDef, error)
}

// ForeignKeyIntrospector provides the ability to introspect foreign keys of a table.
type ForeignKeyIntrospector interface {
	introspectForeignKeys(ctx context.Context, tableName string) ([]*ast.ForeignKeyDef, error)
}

// FullIntrospector combines all introspection capabilities.
type FullIntrospector interface {
	TableLister
	TableIntrospector
	ColumnIntrospector
	IndexIntrospector
	ForeignKeyIntrospector
}

// introspectSchemaCommon is the shared implementation of IntrospectSchema.
// It uses the provided lister and introspector to gather schema information.
func introspectSchemaCommon(ctx context.Context, lister TableLister, introspector TableIntrospector) (*engine.Schema, error) {
	schema := engine.NewSchema()

	tables, err := lister.listTables(ctx)
	if err != nil {
		return nil, err
	}

	for _, tableName := range tables {
		if isInternalTable(tableName) {
			continue
		}

		tableDef, err := introspector.IntrospectTable(ctx, tableName)
		if err != nil {
			return nil, err
		}
		if tableDef != nil {
			schema.Tables[tableDef.QualifiedName()] = tableDef
		}
	}

	return schema, nil
}

// introspectSchemaCommonWithMapping is the shared implementation of IntrospectSchemaWithMapping.
// It uses the provided lister and introspector to gather schema information, using the mapping
// to resolve namespace/table name ambiguity.
func introspectSchemaCommonWithMapping(ctx context.Context, lister TableLister, introspector TableIntrospectorWithMapping, mapping TableNameMapping) (*engine.Schema, error) {
	schema := engine.NewSchema()

	tables, err := lister.listTables(ctx)
	if err != nil {
		return nil, err
	}

	for _, tableName := range tables {
		if isInternalTable(tableName) {
			continue
		}

		tableDef, err := introspector.introspectTableWithMapping(ctx, tableName, mapping)
		if err != nil {
			return nil, err
		}
		if tableDef != nil {
			schema.Tables[tableDef.QualifiedName()] = tableDef
		}
	}

	return schema, nil
}

// introspectTableCommon is the shared implementation of IntrospectTable.
func introspectTableCommon(
	ctx context.Context,
	tableName string,
	colIntrospector ColumnIntrospector,
	idxIntrospector IndexIntrospector,
	fkIntrospector ForeignKeyIntrospector,
) (*ast.TableDef, error) {
	return introspectTableCommonWithMapping(ctx, tableName, colIntrospector, idxIntrospector, fkIntrospector, nil)
}

// introspectTableCommonWithMapping is the shared implementation with mapping support.
func introspectTableCommonWithMapping(
	ctx context.Context,
	tableName string,
	colIntrospector ColumnIntrospector,
	idxIntrospector IndexIntrospector,
	fkIntrospector ForeignKeyIntrospector,
	mapping TableNameMapping,
) (*ast.TableDef, error) {
	namespace, name := parseTableName(tableName, mapping)

	columns, err := colIntrospector.introspectColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}

	if len(columns) == 0 {
		return nil, nil // Table doesn't exist
	}

	indexes, err := idxIntrospector.introspectIndexes(ctx, tableName)
	if err != nil {
		return nil, err
	}

	foreignKeys, err := fkIntrospector.introspectForeignKeys(ctx, tableName)
	if err != nil {
		return nil, err
	}

	// Post-process: Mark columns as Unique if they have single-column unique indexes
	// This handles the SQLite pattern where column-level UNIQUE constraints
	// are stored as indexes, not as column properties
	// Also filters out auto-generated indexes that represent column constraints
	indexes = markUniqueColumnsAndFilterAutoIndexes(columns, indexes)

	return &ast.TableDef{
		Namespace:   namespace,
		Name:        name,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
	}, nil
}

// markUniqueColumnsAndFilterAutoIndexes marks columns as Unique if they have single-column
// unique indexes, and filters out auto-generated indexes that represent column constraints.
// This handles databases like SQLite where column-level UNIQUE constraints are stored
// as indexes rather than column properties.
func markUniqueColumnsAndFilterAutoIndexes(columns []*ast.ColumnDef, indexes []*ast.IndexDef) []*ast.IndexDef {
	// Build map of column name -> column def for quick lookup
	colMap := make(map[string]*ast.ColumnDef)
	for _, col := range columns {
		colMap[col.Name] = col
	}

	// Track which indexes are auto-generated constraint indexes
	isConstraintIndex := make(map[string]bool)

	// Check each unique index
	for _, idx := range indexes {
		if idx.Unique && len(idx.Columns) == 1 {
			// Single-column unique index - mark the column as Unique
			colName := idx.Columns[0]
			if col, exists := colMap[colName]; exists {
				col.Unique = true
				// Mark this index as a constraint index to be filtered out
				// These are auto-generated by SQLite for PRIMARY KEY and UNIQUE constraints
				if strings.HasPrefix(idx.Name, "sqlite_autoindex_") {
					isConstraintIndex[idx.Name] = true
				}
			}
		}
	}

	// Filter out constraint indexes
	filteredIndexes := make([]*ast.IndexDef, 0, len(indexes))
	for _, idx := range indexes {
		if !isConstraintIndex[idx.Name] {
			filteredIndexes = append(filteredIndexes, idx)
		}
	}

	return filteredIndexes
}

// tableExistsCommon is the shared implementation of TableExists.
func tableExistsCommon(ctx context.Context, db *sql.DB, query string, tableName string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// FKAccumulator merges composite FK columns into single FKDef.
// It handles the common pattern where foreign key information is returned
// row-by-row from database catalogs and needs to be accumulated.
type FKAccumulator struct {
	fks   map[string]*ast.ForeignKeyDef
	order []string // preserve insertion order
}

// NewFKAccumulator creates a new FKAccumulator.
func NewFKAccumulator() *FKAccumulator {
	return &FKAccumulator{
		fks:   make(map[string]*ast.ForeignKeyDef),
		order: make([]string, 0),
	}
}

// Add adds or updates a foreign key entry.
// If a FK with the same name exists, it appends the column to the existing FK.
// Otherwise, it creates a new FK entry.
func (a *FKAccumulator) Add(name, column, refTable, refColumn, onDelete, onUpdate string) {
	if fk, exists := a.fks[name]; exists {
		fk.Columns = append(fk.Columns, column)
		fk.RefColumns = append(fk.RefColumns, refColumn)
	} else {
		a.fks[name] = &ast.ForeignKeyDef{
			Name:       name,
			Columns:    []string{column},
			RefTable:   refTable,
			RefColumns: []string{refColumn},
			OnDelete:   normalizeAction(onDelete),
			OnUpdate:   normalizeAction(onUpdate),
		}
		a.order = append(a.order, name)
	}
}

// SetActions sets the ON DELETE and ON UPDATE actions for a named FK.
// This is useful when action rules are fetched separately from the main FK query.
func (a *FKAccumulator) SetActions(name, onDelete, onUpdate string) {
	if fk, exists := a.fks[name]; exists {
		fk.OnDelete = normalizeAction(onDelete)
		fk.OnUpdate = normalizeAction(onUpdate)
	}
}

// Values returns all accumulated foreign keys in insertion order.
func (a *FKAccumulator) Values() []*ast.ForeignKeyDef {
	result := make([]*ast.ForeignKeyDef, 0, len(a.fks))
	for _, name := range a.order {
		result = append(result, a.fks[name])
	}
	return result
}

// Names returns the names of all accumulated foreign keys in insertion order.
func (a *FKAccumulator) Names() []string {
	return a.order
}
