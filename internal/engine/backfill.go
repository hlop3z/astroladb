// Package engine provides migration planning and execution.
package engine

import (
	"github.com/hlop3z/astroladb/internal/ast"
)

// BackfillCandidate represents a NOT NULL column that needs a backfill value.
type BackfillCandidate struct {
	Namespace string // Table namespace
	Table     string // Table name
	Column    string // Column name
	ColType   string // Column type (for suggesting defaults)
}

// DetectMissingBackfills finds AddColumn operations that add NOT NULL columns
// without a default or backfill value. These will fail on tables with existing data.
func DetectMissingBackfills(ops []ast.Operation) []BackfillCandidate {
	var candidates []BackfillCandidate

	for _, op := range ops {
		addCol, ok := op.(*ast.AddColumn)
		if !ok {
			continue
		}

		col := addCol.Column
		if col == nil {
			continue
		}

		// Skip if column is nullable - no backfill needed
		if col.Nullable {
			continue
		}

		// Skip if column has a default value
		if col.DefaultSet || col.ServerDefault != "" {
			continue
		}

		// Skip if column already has a backfill
		if col.BackfillSet || col.Backfill != nil {
			continue
		}

		// Skip primary keys (usually auto-generated like UUID)
		if col.PrimaryKey {
			continue
		}

		// This column needs a backfill value
		candidates = append(candidates, BackfillCandidate{
			Namespace: addCol.Namespace,
			Table:     addCol.Table_,
			Column:    col.Name,
			ColType:   col.Type,
		})
	}

	return candidates
}

// SuggestDefault returns a suggested default value based on column type.
func SuggestDefault(colType string) string {
	switch colType {
	case "string", "text":
		return `""`
	case "integer", "float", "decimal":
		return "0"
	case "boolean":
		return "false"
	case "date":
		return `"1970-01-01"`
	case "time":
		return `"00:00:00"`
	case "date_time":
		return `"1970-01-01T00:00:00Z"`
	case "uuid":
		return `sql("gen_random_uuid()")` // PostgreSQL
	case "json":
		return `{}`
	default:
		return `""`
	}
}

// ApplyBackfills adds backfill values to AddColumn operations.
// The backfills map is keyed by "namespace.table.column" -> backfill value.
func ApplyBackfills(ops []ast.Operation, backfills map[string]string) []ast.Operation {
	if len(backfills) == 0 {
		return ops
	}

	result := make([]ast.Operation, len(ops))
	for i, op := range ops {
		addCol, ok := op.(*ast.AddColumn)
		if !ok {
			result[i] = op
			continue
		}

		key := addCol.Namespace + "." + addCol.Table_ + "." + addCol.Column.Name
		backfill, hasBackfill := backfills[key]
		if !hasBackfill {
			result[i] = op
			continue
		}

		// Clone the operation with backfill applied
		newCol := *addCol.Column
		newCol.Backfill = backfill
		newCol.BackfillSet = true

		result[i] = &ast.AddColumn{
			TableRef: ast.TableRef{Namespace: addCol.Namespace, Table_: addCol.Table_},
			Column:   &newCol,
		}
	}

	return result
}
