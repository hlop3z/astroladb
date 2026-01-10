package ast

import (
	"github.com/hlop3z/astroladb/internal/alerr"
)

// Operation represents a single atomic change to the database schema.
// All schema changes (from both schema files and migrations) are converted
// to Operations before being rendered to SQL.
type Operation interface {
	// Type returns the operation type (OpCreateTable, OpAddColumn, etc.)
	Type() OpType

	// Table returns the full table name (namespace_tablename format).
	// For operations that don't target a specific table (e.g., RawSQL),
	// this returns an empty string.
	Table() string

	// Validate checks that the operation is well-formed.
	// Returns an error if the operation has invalid or missing fields.
	Validate() error
}

// -----------------------------------------------------------------------------
// Embedded types for DRY operation definitions
// -----------------------------------------------------------------------------

// TableOp provides common Namespace+Name fields for table-level operations.
// Operations that create, drop, or modify tables embed this type.
type TableOp struct {
	Namespace string
	Name      string
}

// Table returns the full table name in namespace_tablename format.
func (t TableOp) Table() string {
	if t.Namespace == "" {
		return t.Name
	}
	return t.Namespace + "_" + t.Name
}

// TableRef provides common Namespace+Table_ fields for column/index operations.
// Operations that target columns or indexes within a table embed this type.
type TableRef struct {
	Namespace string
	Table_    string
}

// Table returns the full table name in namespace_tablename format.
func (t TableRef) Table() string {
	if t.Namespace == "" {
		return t.Table_
	}
	return t.Namespace + "_" + t.Table_
}

// -----------------------------------------------------------------------------
// CreateTable - creates a new table
// -----------------------------------------------------------------------------

// CreateTable represents creating a new table with columns, indexes, and constraints.
type CreateTable struct {
	TableOp
	Columns     []*ColumnDef
	Indexes     []*IndexDef
	ForeignKeys []*ForeignKeyDef
	IfNotExists bool // If true, generates CREATE TABLE IF NOT EXISTS
}

func (op *CreateTable) Type() OpType { return OpCreateTable }

func (op *CreateTable) Validate() error {
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required")
	}
	if len(op.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, "table must have at least one column").
			WithTable(op.Namespace, op.Name)
	}
	// Validate each column
	for _, col := range op.Columns {
		if err := col.Validate(); err != nil {
			return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid column").
				WithTable(op.Namespace, op.Name).
				WithColumn(col.Name)
		}
	}
	// Validate each index
	for _, idx := range op.Indexes {
		if err := idx.Validate(); err != nil {
			return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid index").
				WithTable(op.Namespace, op.Name)
		}
	}
	// Validate each foreign key
	for _, fk := range op.ForeignKeys {
		if err := fk.Validate(); err != nil {
			return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid foreign key").
				WithTable(op.Namespace, op.Name)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// DropTable - removes an existing table
// -----------------------------------------------------------------------------

// DropTable represents dropping an existing table.
type DropTable struct {
	TableOp
	IfExists bool
}

func (op *DropTable) Type() OpType { return OpDropTable }

func (op *DropTable) Validate() error {
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for drop")
	}
	return nil
}

// -----------------------------------------------------------------------------
// RenameTable - renames an existing table
// -----------------------------------------------------------------------------

// RenameTable represents renaming an existing table.
type RenameTable struct {
	Namespace string
	OldName   string
	NewName   string
}

func (op *RenameTable) Type() OpType { return OpRenameTable }

func (op *RenameTable) Table() string {
	if op.Namespace != "" {
		return op.Namespace + "_" + op.OldName
	}
	return op.OldName
}

func (op *RenameTable) Validate() error {
	if op.OldName == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "old table name is required for rename")
	}
	if op.NewName == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "new table name is required for rename")
	}
	if op.OldName == op.NewName {
		return alerr.New(alerr.ErrSchemaInvalid, "old and new table names must be different").
			WithTable(op.Namespace, op.OldName)
	}
	return nil
}

// -----------------------------------------------------------------------------
// AddColumn - adds a column to an existing table
// -----------------------------------------------------------------------------

// AddColumn represents adding a new column to an existing table.
type AddColumn struct {
	TableRef
	Column *ColumnDef
}

func (op *AddColumn) Type() OpType { return OpAddColumn }

func (op *AddColumn) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for add column")
	}
	if op.Column == nil {
		return alerr.New(alerr.ErrSchemaInvalid, "column definition is required").
			WithTable(op.Namespace, op.Table_)
	}
	if err := op.Column.Validate(); err != nil {
		return alerr.Wrap(alerr.ErrSchemaInvalid, err, "invalid column").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.Column.Name)
	}
	return nil
}

// -----------------------------------------------------------------------------
// DropColumn - removes a column from an existing table
// -----------------------------------------------------------------------------

// DropColumn represents removing a column from an existing table.
type DropColumn struct {
	TableRef
	Name string
}

func (op *DropColumn) Type() OpType { return OpDropColumn }

func (op *DropColumn) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for drop column")
	}
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "column name is required for drop column").
			WithTable(op.Namespace, op.Table_)
	}
	return nil
}

// -----------------------------------------------------------------------------
// RenameColumn - renames a column
// -----------------------------------------------------------------------------

// RenameColumn represents renaming a column in an existing table.
type RenameColumn struct {
	TableRef
	OldName string
	NewName string
}

func (op *RenameColumn) Type() OpType { return OpRenameColumn }

func (op *RenameColumn) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for rename column")
	}
	if op.OldName == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "old column name is required for rename").
			WithTable(op.Namespace, op.Table_)
	}
	if op.NewName == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "new column name is required for rename").
			WithTable(op.Namespace, op.Table_)
	}
	if op.OldName == op.NewName {
		return alerr.New(alerr.ErrSchemaInvalid, "old and new column names must be different").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.OldName)
	}
	return nil
}

// -----------------------------------------------------------------------------
// AlterColumn - modifies a column's type, nullability, or default
// -----------------------------------------------------------------------------

// AlterColumn represents modifying a column's properties.
// Only set the fields that should change; nil/zero values are ignored.
type AlterColumn struct {
	TableRef
	Name string

	// Changes (only set what needs to change)
	NewType       string // New type (empty = no change)
	NewTypeArgs   []any  // Type arguments (e.g., length, precision)
	SetNullable   *bool  // nil = no change, true/false = set nullable/not null
	SetDefault    any    // nil = no change, use DropDefault to remove
	DropDefault   bool   // true = remove default value
	ServerDefault string // Raw SQL expression for default
}

func (op *AlterColumn) Type() OpType { return OpAlterColumn }

func (op *AlterColumn) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for alter column")
	}
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "column name is required for alter column").
			WithTable(op.Namespace, op.Table_)
	}
	// Must have at least one change
	hasChange := op.NewType != "" ||
		op.SetNullable != nil ||
		op.SetDefault != nil ||
		op.DropDefault ||
		op.ServerDefault != ""
	if !hasChange {
		return alerr.New(alerr.ErrSchemaInvalid, "alter column must specify at least one change").
			WithTable(op.Namespace, op.Table_).
			WithColumn(op.Name)
	}
	return nil
}

// -----------------------------------------------------------------------------
// CreateIndex - creates a new index
// -----------------------------------------------------------------------------

// CreateIndex represents creating a new index on one or more columns.
type CreateIndex struct {
	TableRef
	Name        string   // Index name (auto-generated if empty)
	Columns     []string // Columns to index
	Unique      bool     // UNIQUE index
	IfNotExists bool
}

func (op *CreateIndex) Type() OpType { return OpCreateIndex }

func (op *CreateIndex) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for create index")
	}
	if len(op.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, "index must have at least one column").
			WithTable(op.Namespace, op.Table_)
	}
	return nil
}

// -----------------------------------------------------------------------------
// DropIndex - removes an existing index
// -----------------------------------------------------------------------------

// DropIndex represents removing an existing index.
type DropIndex struct {
	TableRef
	Name     string
	IfExists bool
}

func (op *DropIndex) Type() OpType { return OpDropIndex }

func (op *DropIndex) Validate() error {
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "index name is required for drop index")
	}
	return nil
}

// -----------------------------------------------------------------------------
// AddForeignKey - adds a foreign key constraint
// -----------------------------------------------------------------------------

// AddForeignKey represents adding a foreign key constraint.
type AddForeignKey struct {
	TableRef
	Name       string   // Constraint name (auto-generated if empty)
	Columns    []string // Local columns
	RefTable   string   // Referenced table (namespace_tablename format)
	RefColumns []string // Referenced columns
	OnDelete   string   // CASCADE, SET NULL, RESTRICT, etc.
	OnUpdate   string   // CASCADE, SET NULL, RESTRICT, etc.
}

func (op *AddForeignKey) Type() OpType { return OpAddForeignKey }

func (op *AddForeignKey) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for add foreign key")
	}
	if len(op.Columns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, "foreign key must have at least one column").
			WithTable(op.Namespace, op.Table_)
	}
	if op.RefTable == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "foreign key must reference a table").
			WithTable(op.Namespace, op.Table_)
	}
	if len(op.RefColumns) == 0 {
		return alerr.New(alerr.ErrSchemaInvalid, "foreign key must reference at least one column").
			WithTable(op.Namespace, op.Table_)
	}
	if len(op.Columns) != len(op.RefColumns) {
		return alerr.New(alerr.ErrSchemaInvalid, "foreign key column count must match referenced column count").
			WithTable(op.Namespace, op.Table_).
			With("columns", len(op.Columns)).
			With("ref_columns", len(op.RefColumns))
	}
	return nil
}

// -----------------------------------------------------------------------------
// DropForeignKey - removes a foreign key constraint
// -----------------------------------------------------------------------------

// DropForeignKey represents removing a foreign key constraint.
type DropForeignKey struct {
	TableRef
	Name string // Constraint name
}

func (op *DropForeignKey) Type() OpType { return OpDropForeignKey }

func (op *DropForeignKey) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for drop foreign key")
	}
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "constraint name is required for drop foreign key").
			WithTable(op.Namespace, op.Table_)
	}
	return nil
}

// -----------------------------------------------------------------------------
// AddCheck - adds a CHECK constraint
// -----------------------------------------------------------------------------

// AddCheck represents adding a CHECK constraint to an existing table.
type AddCheck struct {
	TableRef
	Name       string // Constraint name
	Expression string // SQL expression for the check (e.g., "age >= 0")
}

func (op *AddCheck) Type() OpType { return OpAddCheck }

func (op *AddCheck) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for add check")
	}
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "constraint name is required for add check").
			WithTable(op.Namespace, op.Table_)
	}
	if op.Expression == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "check expression is required").
			WithTable(op.Namespace, op.Table_)
	}
	return nil
}

// -----------------------------------------------------------------------------
// DropCheck - removes a CHECK constraint
// -----------------------------------------------------------------------------

// DropCheck represents removing a CHECK constraint from an existing table.
type DropCheck struct {
	TableRef
	Name string // Constraint name
}

func (op *DropCheck) Type() OpType { return OpDropCheck }

func (op *DropCheck) Validate() error {
	if op.Table_ == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "table name is required for drop check")
	}
	if op.Name == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "constraint name is required for drop check").
			WithTable(op.Namespace, op.Table_)
	}
	return nil
}

// -----------------------------------------------------------------------------
// RawSQL - executes raw SQL
// -----------------------------------------------------------------------------

// RawSQL represents a raw SQL statement (escape hatch for unsupported operations).
// Use sparingly; prefer structured operations for better dialect support.
type RawSQL struct {
	SQL string
	// Per-dialect overrides (optional)
	Postgres string
	SQLite   string
}

func (op *RawSQL) Type() OpType { return OpRawSQL }

func (op *RawSQL) Table() string {
	return "" // Raw SQL doesn't target a specific table
}

func (op *RawSQL) Validate() error {
	if op.SQL == "" && op.Postgres == "" && op.SQLite == "" {
		return alerr.New(alerr.ErrSchemaInvalid, "raw SQL statement is required")
	}
	return nil
}
