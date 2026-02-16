package engine

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// DestructiveOperation represents an operation that could cause data loss.
type DestructiveOperation struct {
	Op          ast.Operation
	Type        string // "drop_table", "drop_column", "drop_index"
	Target      string // Table or column name
	HasData     bool   // True if table/column contains data
	RowCount    int64  // Number of rows in table (0 if unknown or not applicable)
	Description string // Human-readable description of the risk
}

// IsDestructive returns true if the operation could cause data loss.
func (d *DestructiveOperation) IsDestructive() bool {
	return d.HasData && d.RowCount > 0
}

// DetectDestructiveOperations analyzes operations and identifies those that could cause data loss.
// Returns a list of potentially destructive operations with metadata about the risk.
func DetectDestructiveOperations(ctx context.Context, db *sql.DB, ops []ast.Operation, dialect string) ([]DestructiveOperation, error) {
	var destructive []DestructiveOperation

	for _, op := range ops {
		switch v := op.(type) {
		case *ast.DropTable:
			d, err := checkDropTable(ctx, db, v, dialect)
			if err != nil {
				return nil, err
			}
			if d != nil {
				destructive = append(destructive, *d)
			}

		case *ast.DropColumn:
			d, err := checkDropColumn(ctx, db, v, dialect)
			if err != nil {
				return nil, err
			}
			if d != nil {
				destructive = append(destructive, *d)
			}

		case *ast.DropIndex:
			// Indexes don't contain data - safe to drop
			// But we still track it as a destructive operation for audit purposes
			destructive = append(destructive, DestructiveOperation{
				Op:          op,
				Type:        "drop_index",
				Target:      v.Name,
				HasData:     false,
				RowCount:    0,
				Description: fmt.Sprintf("Dropping index %q", v.Name),
			})
		}
	}

	return destructive, nil
}

// checkDropTable checks if a DROP TABLE operation would cause data loss.
func checkDropTable(ctx context.Context, db *sql.DB, op *ast.DropTable, dialect string) (*DestructiveOperation, error) {
	tableName := op.Table()

	// Check if table exists
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		// Table doesn't exist - not destructive
		return nil, nil
	}

	// Count rows in table
	rowCount, err := countRows(ctx, db, tableName, dialect)
	if err != nil {
		// If we can't count rows, assume it might have data
		return &DestructiveOperation{
			Op:          op,
			Type:        "drop_table",
			Target:      tableName,
			HasData:     true,
			RowCount:    -1, // Unknown
			Description: fmt.Sprintf("Dropping table %q (row count unknown)", tableName),
		}, nil
	}

	return &DestructiveOperation{
		Op:          op,
		Type:        "drop_table",
		Target:      tableName,
		HasData:     rowCount > 0,
		RowCount:    rowCount,
		Description: fmt.Sprintf("Dropping table %q with %d rows", tableName, rowCount),
	}, nil
}

// checkDropColumn checks if a DROP COLUMN operation would cause data loss.
func checkDropColumn(ctx context.Context, db *sql.DB, op *ast.DropColumn, dialect string) (*DestructiveOperation, error) {
	tableName := op.Table()
	columnName := op.Name

	// Check if table exists
	exists, err := tableExists(ctx, db, tableName, dialect)
	if err != nil {
		return nil, err
	}
	if !exists {
		// Table doesn't exist - not destructive
		return nil, nil
	}

	// Check if column has any non-NULL values
	hasData, err := columnHasData(ctx, db, tableName, columnName, dialect)
	if err != nil {
		// If we can't check, assume it might have data
		return &DestructiveOperation{
			Op:          op,
			Type:        "drop_column",
			Target:      fmt.Sprintf("%s.%s", tableName, columnName),
			HasData:     true,
			RowCount:    -1, // Unknown
			Description: fmt.Sprintf("Dropping column %q from table %q (data check failed)", columnName, tableName),
		}, nil
	}

	// Get row count for context
	rowCount, _ := countRows(ctx, db, tableName, dialect)

	return &DestructiveOperation{
		Op:          op,
		Type:        "drop_column",
		Target:      fmt.Sprintf("%s.%s", tableName, columnName),
		HasData:     hasData,
		RowCount:    rowCount,
		Description: fmt.Sprintf("Dropping column %q from table %q with %d rows", columnName, tableName, rowCount),
	}, nil
}

// tableExists checks if a table exists in the database.
func tableExists(ctx context.Context, db *sql.DB, tableName string, dialect string) (bool, error) {
	var query string
	var args []any

	switch dialect {
	case "sqlite":
		query = "SELECT 1 FROM sqlite_master WHERE type='table' AND name=? LIMIT 1"
		args = []any{tableName}
	case "postgres", "postgresql":
		query = "SELECT 1 FROM information_schema.tables WHERE table_name=$1 LIMIT 1"
		args = []any{tableName}
	default:
		return false, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	var exists int
	err := db.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// countRows counts the number of rows in a table.
func countRows(ctx context.Context, db *sql.DB, tableName string, dialect string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", strutil.QuoteSQL(tableName))

	var count int64
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// columnHasData checks if a column has any non-NULL values.
func columnHasData(ctx context.Context, db *sql.DB, tableName, columnName string, dialect string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT 1 FROM %s WHERE %s IS NOT NULL LIMIT 1",
		strutil.QuoteSQL(tableName),
		strutil.QuoteSQL(columnName),
	)

	var hasData int
	err := db.QueryRowContext(ctx, query).Scan(&hasData)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
