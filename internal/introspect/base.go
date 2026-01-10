// Package introspect provides database schema introspection.
// This file contains shared helper functions used by all introspector implementations.
package introspect

import (
	"context"
	"database/sql"

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

// introspectTableCommon is the shared implementation of IntrospectTable.
func introspectTableCommon(
	ctx context.Context,
	tableName string,
	colIntrospector ColumnIntrospector,
	idxIntrospector IndexIntrospector,
	fkIntrospector ForeignKeyIntrospector,
) (*ast.TableDef, error) {
	namespace, name := parseTableName(tableName)

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

	return &ast.TableDef{
		Namespace:   namespace,
		Name:        name,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
	}, nil
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
