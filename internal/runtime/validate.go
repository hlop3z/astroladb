package runtime

import (
	"reflect"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/validate"
)

// ValidateDeterminism validates that a schema or migration is deterministic
// by evaluating it twice and comparing the results.
// Returns an error if the two evaluations produce different results.
func (s *Sandbox) ValidateDeterminism(code string) error {
	// Create two fresh sandboxes with the same configuration
	sandbox1 := NewSandbox(s.registry)
	sandbox2 := NewSandbox(s.registry)

	// Evaluate the code twice
	err1 := sandbox1.Run(code)
	if err1 != nil {
		return alerr.Wrap(alerr.ErrJSExecution, err1, "first evaluation failed")
	}

	err2 := sandbox2.Run(code)
	if err2 != nil {
		return alerr.Wrap(alerr.ErrJSExecution, err2, "second evaluation failed")
	}

	// Compare the results
	tables1 := sandbox1.GetTables()
	tables2 := sandbox2.GetTables()

	if !tablesEqual(tables1, tables2) {
		return alerr.New(alerr.ErrJSNonDeterministic, "schema is non-deterministic: evaluations produced different results").
			With("first_eval_tables", len(tables1)).
			With("second_eval_tables", len(tables2))
	}

	return nil
}

// ValidateDeterminismStrict performs a stricter determinism check by
// evaluating the code multiple times and comparing all results.
func (s *Sandbox) ValidateDeterminismStrict(code string, iterations int) error {
	if iterations < 2 {
		iterations = 2
	}

	var firstTables []*ast.TableDef

	for i := 0; i < iterations; i++ {
		sandbox := NewSandbox(s.registry)
		if err := sandbox.Run(code); err != nil {
			return alerr.Wrap(alerr.ErrJSExecution, err, "evaluation failed").
				With("iteration", i+1)
		}

		tables := sandbox.GetTables()

		if i == 0 {
			firstTables = tables
		} else {
			if !tablesEqual(firstTables, tables) {
				return alerr.New(alerr.ErrJSNonDeterministic, "schema is non-deterministic").
					With("iteration", i+1).
					With("first_tables", len(firstTables)).
					With("current_tables", len(tables))
			}
		}
	}

	return nil
}

// tablesEqual compares two slices of table definitions for equality.
func tablesEqual(t1, t2 []*ast.TableDef) bool {
	if len(t1) != len(t2) {
		return false
	}

	for i := range t1 {
		if !tableDefEqual(t1[i], t2[i]) {
			return false
		}
	}

	return true
}

// tableDefEqual compares two table definitions for equality.
func tableDefEqual(t1, t2 *ast.TableDef) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil || t2 == nil {
		return false
	}

	if t1.Namespace != t2.Namespace || t1.Name != t2.Name {
		return false
	}

	if len(t1.Columns) != len(t2.Columns) {
		return false
	}

	for i := range t1.Columns {
		if !columnDefEqual(t1.Columns[i], t2.Columns[i]) {
			return false
		}
	}

	if len(t1.Indexes) != len(t2.Indexes) {
		return false
	}

	for i := range t1.Indexes {
		if !reflect.DeepEqual(t1.Indexes[i], t2.Indexes[i]) {
			return false
		}
	}

	return true
}

// columnDefEqual compares two column definitions for equality.
func columnDefEqual(c1, c2 *ast.ColumnDef) bool {
	if c1 == nil && c2 == nil {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}

	return c1.Name == c2.Name &&
		c1.Type == c2.Type &&
		c1.Nullable == c2.Nullable &&
		c1.Unique == c2.Unique &&
		c1.PrimaryKey == c2.PrimaryKey &&
		reflect.DeepEqual(c1.TypeArgs, c2.TypeArgs) &&
		reflect.DeepEqual(c1.Default, c2.Default) &&
		reflect.DeepEqual(c1.Reference, c2.Reference)
}

// ValidateIdentifiers validates that all identifiers in operations are snake_case.
// Returns an error describing all invalid identifiers found.
func ValidateIdentifiers(ops []ast.Operation) error {
	var errs validate.ValidationErrors

	for _, op := range ops {
		switch o := op.(type) {
		case *ast.CreateTable:
			validateTableIdentifiers(o, &errs)
		case *ast.AddColumn:
			validateColumnName(o.Column, o.Namespace, o.Table_, &errs)
		case *ast.RenameColumn:
			if err := validate.ColumnName(o.NewName); err != nil {
				errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "new column name must be snake_case").
					WithTable(o.Namespace, o.Table_).
					WithColumn(o.NewName))
			}
		case *ast.RenameTable:
			if err := validate.TableName(o.NewName); err != nil {
				errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "new table name must be snake_case").
					WithTable(o.Namespace, o.NewName))
			}
		case *ast.CreateIndex:
			for _, col := range o.Columns {
				if err := validate.ColumnName(col); err != nil {
					errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "index column name must be snake_case").
						WithTable(o.Namespace, o.Table_).
						With("column", col))
				}
			}
		}
	}

	return errs.ToError()
}

// validateTableIdentifiers validates all identifiers in a CreateTable operation.
func validateTableIdentifiers(op *ast.CreateTable, errs *validate.ValidationErrors) {
	// Validate namespace
	if op.Namespace != "" {
		if err := validate.Namespace(op.Namespace); err != nil {
			errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "namespace must be snake_case").
				With("namespace", op.Namespace))
		}
	}

	// Validate table name
	if err := validate.TableName(op.Name); err != nil {
		errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "table name must be snake_case").
			WithTable(op.Namespace, op.Name))
	}

	// Validate column names
	for _, col := range op.Columns {
		validateColumnName(col, op.Namespace, op.Name, errs)
	}

	// Validate index column names
	for _, idx := range op.Indexes {
		for _, col := range idx.Columns {
			if err := validate.ColumnName(col); err != nil {
				errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "index column must be snake_case").
					WithTable(op.Namespace, op.Name).
					With("column", col))
			}
		}
	}
}

// validateColumnName validates a column name.
func validateColumnName(col *ast.ColumnDef, ns, table string, errs *validate.ValidationErrors) {
	if col == nil {
		return
	}

	if err := validate.ColumnName(col.Name); err != nil {
		errs.Add(alerr.New(alerr.ErrInvalidSnakeCase, "column name must be snake_case").
			WithTable(ns, table).
			WithColumn(col.Name))
	}
}

// ValidateOperations validates all operations for correctness.
// This includes identifier validation and structural validation.
func ValidateOperations(ops []ast.Operation) error {
	var errs validate.ValidationErrors

	// First, validate identifiers
	if err := ValidateIdentifiers(ops); err != nil {
		if ve, ok := err.(validate.ValidationErrors); ok {
			errs.Merge(ve)
		} else {
			errs.Add(err)
		}
	}

	// Then, validate each operation's structure
	for _, op := range ops {
		if err := op.Validate(); err != nil {
			errs.Add(err)
		}
	}

	return errs.ToError()
}

// ValidateSchema validates a schema code string without executing it for operations.
// This performs:
// 1. Determinism check
// 2. Identifier validation
// 3. Reference validation (if registry is provided)
func (s *Sandbox) ValidateSchema(code string) error {
	// Check determinism first
	if err := s.ValidateDeterminism(code); err != nil {
		return err
	}

	// Execute to get operations
	sandbox := NewSandbox(s.registry)
	if err := sandbox.Run(code); err != nil {
		return err
	}

	// Validate tables
	tables := sandbox.GetTables()
	for _, table := range tables {
		// Convert table to CreateTable operation for validation
		op := &ast.CreateTable{
			TableOp: ast.TableOp{Namespace: table.Namespace, Name: table.Name},
			Columns: table.Columns,
			Indexes: table.Indexes,
		}
		if err := op.Validate(); err != nil {
			return err
		}
	}

	// Validate identifiers
	ops := make([]ast.Operation, 0, len(tables))
	for _, table := range tables {
		ops = append(ops, &ast.CreateTable{
			TableOp: ast.TableOp{Namespace: table.Namespace, Name: table.Name},
			Columns: table.Columns,
			Indexes: table.Indexes,
		})
	}

	if err := ValidateIdentifiers(ops); err != nil {
		return err
	}

	return nil
}

// DetectForbiddenTypes checks for JavaScript-unsafe types in operations.
// Returns an error if any forbidden types are found.
func DetectForbiddenTypes(ops []ast.Operation) error {
	var errs validate.ValidationErrors

	for _, op := range ops {
		switch o := op.(type) {
		case *ast.CreateTable:
			for _, col := range o.Columns {
				if err := checkForbiddenType(col.Type); err != nil {
					errs.Add(alerr.New(alerr.ErrInvalidType, "forbidden type detected").
						WithTable(o.Namespace, o.Name).
						WithColumn(col.Name).
						With("type", col.Type).
						With("reason", err.Error()))
				}
			}
		case *ast.AddColumn:
			if o.Column != nil {
				if err := checkForbiddenType(o.Column.Type); err != nil {
					errs.Add(alerr.New(alerr.ErrInvalidType, "forbidden type detected").
						WithTable(o.Namespace, o.Table_).
						WithColumn(o.Column.Name).
						With("type", o.Column.Type).
						With("reason", err.Error()))
				}
			}
		case *ast.AlterColumn:
			if o.NewType != "" {
				if err := checkForbiddenType(o.NewType); err != nil {
					errs.Add(alerr.New(alerr.ErrInvalidType, "forbidden type detected").
						WithTable(o.Namespace, o.Table_).
						WithColumn(o.Name).
						With("type", o.NewType).
						With("reason", err.Error()))
				}
			}
		}
	}

	return errs.ToError()
}

// checkForbiddenType returns an error if the type is forbidden.
func checkForbiddenType(typeName string) error {
	switch typeName {
	case "int64", "bigint":
		return alerr.New(alerr.ErrInvalidType, "int64/bigint loses precision in JavaScript (> 2^53)")
	case "float64", "double":
		return alerr.New(alerr.ErrInvalidType, "float64/double has precision issues in JavaScript")
	case "auto_increment", "serial", "bigserial":
		return alerr.New(alerr.ErrInvalidType, "auto-increment IDs are forbidden; use UUID instead")
	}
	return nil
}
