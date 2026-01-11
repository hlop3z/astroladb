package introspect

import (
	"database/sql"
	"strings"
)

// TypeMapping holds the result of parsing a SQL type back to Alab type.
type TypeMapping struct {
	AlabType string
	TypeArgs []any
}

// MapPostgresType converts PostgreSQL type to Alab type.
func MapPostgresType(sqlType string, maxLen, precision, scale sql.NullInt64) TypeMapping {
	upper := strings.ToUpper(sqlType)

	switch {
	case upper == "UUID":
		return TypeMapping{AlabType: "uuid"}

	case strings.HasPrefix(upper, "CHARACTER VARYING"), strings.HasPrefix(upper, "VARCHAR"):
		if maxLen.Valid && maxLen.Int64 > 0 {
			return TypeMapping{AlabType: "string", TypeArgs: []any{int(maxLen.Int64)}}
		}
		return TypeMapping{AlabType: "string", TypeArgs: []any{255}}

	case upper == "TEXT":
		return TypeMapping{AlabType: "text"}

	case upper == "INTEGER", upper == "INT", upper == "INT4":
		return TypeMapping{AlabType: "integer"}

	case upper == "SMALLINT", upper == "INT2":
		return TypeMapping{AlabType: "integer"}

	case upper == "BIGINT", upper == "INT8":
		// Note: int64 is forbidden in Alab, but we still need to introspect it
		return TypeMapping{AlabType: "integer"}

	case upper == "REAL", upper == "FLOAT4":
		return TypeMapping{AlabType: "float"}

	case upper == "DOUBLE PRECISION", upper == "FLOAT8":
		return TypeMapping{AlabType: "float"}

	case strings.HasPrefix(upper, "NUMERIC"), strings.HasPrefix(upper, "DECIMAL"):
		p, s := 10, 2
		if precision.Valid {
			p = int(precision.Int64)
		}
		if scale.Valid {
			s = int(scale.Int64)
		}
		return TypeMapping{AlabType: "decimal", TypeArgs: []any{p, s}}

	case upper == "BOOLEAN", upper == "BOOL":
		return TypeMapping{AlabType: "boolean"}

	case upper == "DATE":
		return TypeMapping{AlabType: "date"}

	case upper == "TIME", upper == "TIME WITHOUT TIME ZONE":
		return TypeMapping{AlabType: "time"}

	case upper == "TIMESTAMP WITH TIME ZONE", upper == "TIMESTAMPTZ":
		return TypeMapping{AlabType: "datetime"}

	case upper == "TIMESTAMP", upper == "TIMESTAMP WITHOUT TIME ZONE":
		return TypeMapping{AlabType: "datetime"}

	case upper == "JSONB", upper == "JSON":
		return TypeMapping{AlabType: "json"}

	case upper == "BYTEA":
		return TypeMapping{AlabType: "base64"}

	default:
		// Check for user-defined enum types - they appear as the type name
		// For now, treat unknown types as text
		return TypeMapping{AlabType: "text"}
	}
}

// MapSQLiteType converts SQLite type to Alab type.
// SQLite has dynamic typing with type affinity, so we map based on declared type.
// Note: SQLite does not preserve type precision/scale, so TypeArgs may be empty.
func MapSQLiteType(sqlType string) TypeMapping {
	upper := strings.ToUpper(sqlType)

	switch {
	case upper == "TEXT":
		return TypeMapping{AlabType: "text"}

	case upper == "INTEGER":
		return TypeMapping{AlabType: "integer"}

	case upper == "REAL":
		return TypeMapping{AlabType: "float"}

	case upper == "BLOB":
		return TypeMapping{AlabType: "base64"}

	case upper == "NUMERIC", upper == "DECIMAL":
		// SQLite stores NUMERIC/DECIMAL as TEXT without precision metadata
		// Return decimal type with empty args to indicate unknown precision
		return TypeMapping{AlabType: "decimal", TypeArgs: []any{}}

	case upper == "DATETIME", upper == "TIMESTAMP":
		return TypeMapping{AlabType: "datetime"}

	case upper == "DATE":
		return TypeMapping{AlabType: "date"}

	case upper == "TIME":
		return TypeMapping{AlabType: "time"}

	default:
		// SQLite is lenient with types, default to text
		return TypeMapping{AlabType: "text"}
	}
}

// Removed heuristic type inference functions.
// Introspection now uses only database-provided metadata for deterministic behavior.
