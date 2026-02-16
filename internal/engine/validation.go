package engine

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

// ConstraintViolation represents a constraint that would fail if applied to existing data.
type ConstraintViolation struct {
	Op             ast.Operation
	Type           string   // "not_null", "unique", "foreign_key"
	Target         string   // Column or constraint name
	ViolationCount int64    // Number of rows that violate the constraint
	Description    string   // Human-readable description
	SampleValues   []string // Sample violating values (for debugging)
}

// ValidateConstraints checks if constraint operations would fail on existing data.
// Returns violations that would prevent the migration from succeeding.
func ValidateConstraints(ctx context.Context, db *sql.DB, ops []ast.Operation, dialect string) ([]ConstraintViolation, error) {
	var violations []ConstraintViolation

	for _, op := range ops {
		switch v := op.(type) {
		case *ast.AddColumn:
			// Check NOT NULL constraint on new columns
			if v.Column != nil && !v.Column.Nullable && !v.Column.HasDefault() && !v.Column.HasBackfill() {
				violation, err := validateNotNullColumn(ctx, db, v, dialect)
				if err != nil {
					return nil, err
				}
				if violation != nil {
					violations = append(violations, *violation)
				}
			}

			// Check UNIQUE constraint on new columns
			if v.Column != nil && v.Column.Unique {
				violation, err := validateUniqueColumn(ctx, db, v, dialect)
				if err != nil {
					return nil, err
				}
				if violation != nil {
					violations = append(violations, *violation)
				}
			}

		case *ast.AlterColumn:
			// Check NOT NULL constraint when making column non-nullable
			if v.SetNullable != nil && !*v.SetNullable {
				violation, err := validateAlterNotNull(ctx, db, v, dialect)
				if err != nil {
					return nil, err
				}
				if violation != nil {
					violations = append(violations, *violation)
				}
			}

		case *ast.CreateIndex:
			// Check UNIQUE index constraints
			if v.Unique {
				violation, err := validateUniqueIndex(ctx, db, v, dialect)
				if err != nil {
					return nil, err
				}
				if violation != nil {
					violations = append(violations, *violation)
				}
			}

		case *ast.AddForeignKey:
			// Check foreign key references exist
			violation, err := validateForeignKey(ctx, db, v, dialect)
			if err != nil {
				return nil, err
			}
			if violation != nil {
				violations = append(violations, *violation)
			}
		}
	}

	return violations, nil
}

// validateNotNullColumn checks if adding a NOT NULL column would fail.
func validateNotNullColumn(ctx context.Context, db *sql.DB, op *ast.AddColumn, dialect string) (*ConstraintViolation, error) {
	tableName := op.Table()

	// Check if table exists and has rows
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil // Table doesn't exist yet - constraint is OK
	}

	rowCount, err := countRows(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if rowCount == 0 {
		return nil, nil // No rows - constraint is OK
	}

	// Table has rows - NOT NULL column without default/backfill will fail
	return &ConstraintViolation{
		Op:             op,
		Type:           "not_null",
		Target:         fmt.Sprintf("%s.%s", tableName, op.Column.Name),
		ViolationCount: rowCount,
		Description: fmt.Sprintf(
			"Cannot add NOT NULL column %q to table %q with %d existing rows (no default or backfill provided)",
			op.Column.Name, tableName, rowCount,
		),
	}, nil
}

// validateUniqueColumn checks if a new UNIQUE column would have duplicates.
func validateUniqueColumn(ctx context.Context, db *sql.DB, op *ast.AddColumn, dialect string) (*ConstraintViolation, error) {
	tableName := op.Table()

	// Check if table exists
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	rowCount, err := countRows(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if rowCount == 0 {
		return nil, nil
	}

	// If column has a default value, check if that default would create duplicates
	if op.Column.HasDefault() {
		// All rows would get the same default value - this violates UNIQUE
		return &ConstraintViolation{
			Op:             op,
			Type:           "unique",
			Target:         fmt.Sprintf("%s.%s", tableName, op.Column.Name),
			ViolationCount: rowCount,
			Description: fmt.Sprintf(
				"Cannot add UNIQUE column %q with default value to table %q with %d rows (all rows would have same value)",
				op.Column.Name, tableName, rowCount,
			),
		}, nil
	}

	// If no default, NULL values are allowed (UNIQUE allows multiple NULLs in most databases)
	return nil, nil
}

// validateAlterNotNull checks if making a column NOT NULL would fail.
func validateAlterNotNull(ctx context.Context, db *sql.DB, op *ast.AlterColumn, dialect string) (*ConstraintViolation, error) {
	tableName := op.Table()
	columnName := op.Name

	// Check if table exists
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	// Count NULL values in column
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE %s IS NULL",
		quoteName(tableName, dialect),
		quoteName(columnName, dialect),
	)

	var nullCount int64
	err = db.QueryRowContext(ctx, query).Scan(&nullCount)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrSQLExecution, err, "failed to count NULL values")
	}

	if nullCount == 0 {
		return nil, nil // No NULLs - constraint is OK
	}

	// Get sample NULL row IDs for debugging
	sampleQuery := fmt.Sprintf(
		"SELECT id FROM %s WHERE %s IS NULL LIMIT 3",
		quoteName(tableName, dialect),
		quoteName(columnName, dialect),
	)

	rows, err := db.QueryContext(ctx, sampleQuery)
	if err == nil {
		defer rows.Close()
		var samples []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				samples = append(samples, id)
			}
		}

		return &ConstraintViolation{
			Op:             op,
			Type:           "not_null",
			Target:         fmt.Sprintf("%s.%s", tableName, columnName),
			ViolationCount: nullCount,
			Description: fmt.Sprintf(
				"Cannot make column %q NOT NULL: %d rows have NULL values",
				columnName, nullCount,
			),
			SampleValues: samples,
		}, nil
	}

	return &ConstraintViolation{
		Op:             op,
		Type:           "not_null",
		Target:         fmt.Sprintf("%s.%s", tableName, columnName),
		ViolationCount: nullCount,
		Description: fmt.Sprintf(
			"Cannot make column %q NOT NULL: %d rows have NULL values",
			columnName, nullCount,
		),
	}, nil
}

// validateUniqueIndex checks if creating a UNIQUE index would fail due to duplicates.
func validateUniqueIndex(ctx context.Context, db *sql.DB, op *ast.CreateIndex, dialect string) (*ConstraintViolation, error) {
	tableName := op.Table()

	// Check if table exists
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	// Build column list for GROUP BY
	columnList := ""
	for i, col := range op.Columns {
		if i > 0 {
			columnList += ", "
		}
		columnList += quoteName(col, dialect)
	}

	// Find duplicate values
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM (SELECT %s, COUNT(*) as cnt FROM %s GROUP BY %s HAVING COUNT(*) > 1) as duplicates",
		columnList,
		quoteName(tableName, dialect),
		columnList,
	)

	var duplicateCount int64
	err = db.QueryRowContext(ctx, query).Scan(&duplicateCount)
	if err != nil {
		// Query might fail if table is empty or columns don't exist yet
		return nil, nil
	}

	if duplicateCount == 0 {
		return nil, nil // No duplicates - constraint is OK
	}

	return &ConstraintViolation{
		Op:             op,
		Type:           "unique",
		Target:         fmt.Sprintf("%s(%s)", tableName, columnList),
		ViolationCount: duplicateCount,
		Description: fmt.Sprintf(
			"Cannot create UNIQUE index on %q: %d duplicate value groups found",
			columnList, duplicateCount,
		),
	}, nil
}

// validateForeignKey checks if foreign key references exist.
func validateForeignKey(ctx context.Context, db *sql.DB, op *ast.AddForeignKey, dialect string) (*ConstraintViolation, error) {
	tableName := op.Table()
	refTable := op.RefTable

	// Check if source table exists
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil // Source table doesn't exist - will be created in same migration
	}

	// Check if referenced table exists
	refExists, err := tableExists(ctx, db, refTable, dialect)
	if err != nil {
		return nil, err
	}
	if !refExists {
		return &ConstraintViolation{
			Op:             op,
			Type:           "foreign_key",
			Target:         fmt.Sprintf("%s -> %s", tableName, refTable),
			ViolationCount: 0,
			Description: fmt.Sprintf(
				"Cannot add foreign key: referenced table %q does not exist",
				refTable,
			),
		}, nil
	}

	// Build column lists
	sourceColumns := ""
	refColumns := ""
	for i, col := range op.Columns {
		if i > 0 {
			sourceColumns += ", "
			refColumns += ", "
		}
		sourceColumns += quoteName(col, dialect)
		refColumns += quoteName(op.RefColumns[i], dialect)
	}

	// Check if source table has rows
	rowCount, err := countRows(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if rowCount == 0 {
		return nil, nil // No rows to validate
	}

	// Check for orphaned rows (values in source that don't exist in reference)
	query := fmt.Sprintf(
		`SELECT COUNT(*) FROM %s src
		 WHERE NOT EXISTS (
		   SELECT 1 FROM %s ref
		   WHERE src.%s = ref.%s
		 )`,
		quoteName(tableName, dialect),
		quoteName(refTable, dialect),
		op.Columns[0], // Simplified for single-column FK
		op.RefColumns[0],
	)

	var orphanCount int64
	err = db.QueryRowContext(ctx, query).Scan(&orphanCount)
	if err != nil {
		// Query might fail - skip validation
		return nil, nil
	}

	if orphanCount == 0 {
		return nil, nil // All references exist
	}

	return &ConstraintViolation{
		Op:             op,
		Type:           "foreign_key",
		Target:         fmt.Sprintf("%s.%s -> %s.%s", tableName, op.Columns[0], refTable, op.RefColumns[0]),
		ViolationCount: orphanCount,
		Description: fmt.Sprintf(
			"Cannot add foreign key: %d rows have values that don't exist in referenced table %q",
			orphanCount, refTable,
		),
	}, nil
}
