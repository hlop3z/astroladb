package dsl

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// MigrationBuilder provides an imperative API for defining migrations.
// Used in migration files: migration(m => { ... })
type MigrationBuilder struct {
	operations []ast.Operation
}

// NewMigrationBuilder creates a new MigrationBuilder.
func NewMigrationBuilder() *MigrationBuilder {
	return &MigrationBuilder{
		operations: make([]ast.Operation, 0),
	}
}

// Operations returns all operations defined in this migration.
func (m *MigrationBuilder) Operations() []ast.Operation {
	return m.operations
}

// CreateTable creates a new table.
func (m *MigrationBuilder) CreateTable(ref string, fn func(*TableBuilder)) {
	ns, table, _ := parseRefParts(ref)
	tb := NewTableBuilder(ns, table)
	fn(tb)
	def := tb.Build()

	op := &ast.CreateTable{
		TableOp:     ast.TableOp{Namespace: ns, Name: table},
		Columns:     def.Columns,
		Indexes:     def.Indexes,
		ForeignKeys: make([]*ast.ForeignKeyDef, 0),
	}

	// Extract foreign keys from columns with references
	for _, col := range def.Columns {
		if col.Reference != nil {
			refNs, refTable, _ := parseRefParts(col.Reference.Table)
			sqlRefTable := refTable
			if refNs != "" {
				sqlRefTable = refNs + "_" + refTable
			}

			fk := &ast.ForeignKeyDef{
				Columns:    []string{col.Name},
				RefTable:   sqlRefTable,
				RefColumns: []string{col.Reference.Column},
				OnDelete:   col.Reference.OnDelete,
				OnUpdate:   col.Reference.OnUpdate,
			}
			op.ForeignKeys = append(op.ForeignKeys, fk)
		}
	}

	m.operations = append(m.operations, op)
}

// DropTable drops an existing table.
func (m *MigrationBuilder) DropTable(ref string) {
	ns, table, _ := parseRefParts(ref)
	m.operations = append(m.operations, &ast.DropTable{
		TableOp: ast.TableOp{Namespace: ns, Name: table},
	})
}

// RenameTable renames a table.
func (m *MigrationBuilder) RenameTable(oldRef, newName string) {
	ns, oldTable, _ := parseRefParts(oldRef)
	m.operations = append(m.operations, &ast.RenameTable{
		Namespace: ns,
		OldName:   oldTable,
		NewName:   newName,
	})
}

// AddColumn adds a column to an existing table.
func (m *MigrationBuilder) AddColumn(ref string, fn func(*ColumnBuilder) *ColumnBuilder) {
	ns, table, _ := parseRefParts(ref)

	// Create a temporary column builder
	cb := NewColumnBuilder("", "")
	cb = fn(cb)
	col := cb.Build()

	m.operations = append(m.operations, &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: ns, Table_: table},
		Column:   col,
	})
}

// DropColumn removes a column from a table.
func (m *MigrationBuilder) DropColumn(ref, column string) {
	ns, table, _ := parseRefParts(ref)
	m.operations = append(m.operations, &ast.DropColumn{
		TableRef: ast.TableRef{Namespace: ns, Table_: table},
		Name:     column,
	})
}

// RenameColumn renames a column.
func (m *MigrationBuilder) RenameColumn(ref, oldName, newName string) {
	ns, table, _ := parseRefParts(ref)
	m.operations = append(m.operations, &ast.RenameColumn{
		TableRef: ast.TableRef{Namespace: ns, Table_: table},
		OldName:  oldName,
		NewName:  newName,
	})
}

// AlterColumn modifies a column's properties.
func (m *MigrationBuilder) AlterColumn(ref, column string, fn func(*AlterColumnBuilder)) {
	ns, table, _ := parseRefParts(ref)
	acb := &AlterColumnBuilder{
		op: &ast.AlterColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     column,
		},
	}
	fn(acb)
	m.operations = append(m.operations, acb.op)
}

// CreateIndex creates an index.
func (m *MigrationBuilder) CreateIndex(ref string, columns []string, opts ...IndexOption) {
	ns, table, _ := parseRefParts(ref)

	op := &ast.CreateIndex{
		TableRef: ast.TableRef{Namespace: ns, Table_: table},
		Columns:  columns,
	}

	for _, opt := range opts {
		opt(op)
	}

	m.operations = append(m.operations, op)
}

// DropIndex drops an index.
func (m *MigrationBuilder) DropIndex(name string) {
	m.operations = append(m.operations, &ast.DropIndex{
		Name: name,
	})
}

// SQL executes raw SQL.
func (m *MigrationBuilder) SQL(sql string) {
	m.operations = append(m.operations, &ast.RawSQL{
		SQL: sql,
	})
}

// SQLDialect executes raw SQL with per-dialect variants.
func (m *MigrationBuilder) SQLDialect(postgres, sqlite string) {
	m.operations = append(m.operations, &ast.RawSQL{
		Postgres: postgres,
		SQLite:   sqlite,
	})
}

// AddForeignKey adds a foreign key constraint to a table.
func (m *MigrationBuilder) AddForeignKey(ref string, columns []string, refTable string, refColumns []string, opts ...FKOption) {
	ns, table, _ := parseRefParts(ref)

	// Convert refTable reference to SQL table name
	refNs, refTbl, _ := parseRefParts(refTable)
	sqlRefTable := refTbl
	if refNs != "" {
		sqlRefTable = refNs + "_" + refTbl
	}

	op := &ast.AddForeignKey{
		TableRef:   ast.TableRef{Namespace: ns, Table_: table},
		Columns:    columns,
		RefTable:   sqlRefTable,
		RefColumns: refColumns,
	}

	for _, opt := range opts {
		opt(op)
	}

	m.operations = append(m.operations, op)
}

// DropForeignKey removes a foreign key constraint.
func (m *MigrationBuilder) DropForeignKey(ref, constraintName string) {
	ns, table, _ := parseRefParts(ref)
	m.operations = append(m.operations, &ast.DropForeignKey{
		TableRef: ast.TableRef{Namespace: ns, Table_: table},
		Name:     constraintName,
	})
}

// FKOption is a functional option for AddForeignKey.
type FKOption func(*ast.AddForeignKey)

// FKName sets a custom constraint name.
func FKName(name string) FKOption {
	return func(op *ast.AddForeignKey) {
		op.Name = name
	}
}

// FKOnDelete sets the ON DELETE action.
func FKOnDelete(action string) FKOption {
	return func(op *ast.AddForeignKey) {
		op.OnDelete = action
	}
}

// FKOnUpdate sets the ON UPDATE action.
func FKOnUpdate(action string) FKOption {
	return func(op *ast.AddForeignKey) {
		op.OnUpdate = action
	}
}

// IndexOption is a functional option for CreateIndex.
type IndexOption func(*ast.CreateIndex)

// IndexUnique makes the index unique.
func IndexUnique() IndexOption {
	return func(op *ast.CreateIndex) {
		op.Unique = true
	}
}

// IndexName sets a custom index name.
func IndexName(name string) IndexOption {
	return func(op *ast.CreateIndex) {
		op.Name = name
	}
}

// AlterColumnBuilder provides options for modifying a column.
type AlterColumnBuilder struct {
	op *ast.AlterColumn
}

// SetType changes the column type.
func (a *AlterColumnBuilder) SetType(typeName string, args ...any) *AlterColumnBuilder {
	a.op.NewType = typeName
	a.op.NewTypeArgs = args
	return a
}

// SetNullable sets the column to nullable.
func (a *AlterColumnBuilder) SetNullable() *AlterColumnBuilder {
	nullable := true
	a.op.SetNullable = &nullable
	return a
}

// SetNotNull sets the column to NOT NULL.
func (a *AlterColumnBuilder) SetNotNull() *AlterColumnBuilder {
	nullable := false
	a.op.SetNullable = &nullable
	return a
}

// SetDefault sets the default value.
func (a *AlterColumnBuilder) SetDefault(value any) *AlterColumnBuilder {
	a.op.SetDefault = value
	return a
}

// DropDefault removes the default value.
func (a *AlterColumnBuilder) DropDefault() *AlterColumnBuilder {
	a.op.DropDefault = true
	return a
}

// SetServerDefault sets a raw SQL default expression.
func (a *AlterColumnBuilder) SetServerDefault(expr string) *AlterColumnBuilder {
	a.op.ServerDefault = expr
	return a
}
