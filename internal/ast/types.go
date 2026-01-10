// Package ast defines the abstract syntax tree for database schema operations.
// Operations represent atomic changes to the database schema that can be
// serialized to SQL for any supported dialect.
package ast

// OpType represents the type of a schema operation.
type OpType int

const (
	// OpCreateTable creates a new table with columns, indexes, and constraints.
	OpCreateTable OpType = iota

	// OpDropTable removes an existing table.
	OpDropTable

	// OpRenameTable changes a table's name.
	OpRenameTable

	// OpAddColumn adds a new column to an existing table.
	OpAddColumn

	// OpDropColumn removes a column from an existing table.
	OpDropColumn

	// OpRenameColumn changes a column's name.
	OpRenameColumn

	// OpAlterColumn modifies a column's type, nullability, or default.
	OpAlterColumn

	// OpCreateIndex creates a new index on one or more columns.
	OpCreateIndex

	// OpDropIndex removes an existing index.
	OpDropIndex

	// OpAddForeignKey adds a foreign key constraint.
	OpAddForeignKey

	// OpDropForeignKey removes a foreign key constraint.
	OpDropForeignKey

	// OpAddCheck adds a CHECK constraint.
	OpAddCheck

	// OpDropCheck removes a CHECK constraint.
	OpDropCheck

	// OpRawSQL executes raw SQL (escape hatch for unsupported operations).
	OpRawSQL
)

// String returns the string representation of an OpType.
func (o OpType) String() string {
	switch o {
	case OpCreateTable:
		return "CreateTable"
	case OpDropTable:
		return "DropTable"
	case OpRenameTable:
		return "RenameTable"
	case OpAddColumn:
		return "AddColumn"
	case OpDropColumn:
		return "DropColumn"
	case OpRenameColumn:
		return "RenameColumn"
	case OpAlterColumn:
		return "AlterColumn"
	case OpCreateIndex:
		return "CreateIndex"
	case OpDropIndex:
		return "DropIndex"
	case OpAddForeignKey:
		return "AddForeignKey"
	case OpDropForeignKey:
		return "DropForeignKey"
	case OpAddCheck:
		return "AddCheck"
	case OpDropCheck:
		return "DropCheck"
	case OpRawSQL:
		return "RawSQL"
	default:
		return "Unknown"
	}
}
