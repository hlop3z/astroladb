package introspect

import (
	"database/sql"
	"regexp"
	"strconv"
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

	case upper == "CHARACTER", upper == "CHAR", strings.HasPrefix(upper, "BPCHAR"):
		// CHARACTER/CHAR without length defaults to 1
		if maxLen.Valid && maxLen.Int64 > 0 {
			return TypeMapping{AlabType: "string", TypeArgs: []any{int(maxLen.Int64)}}
		}
		return TypeMapping{AlabType: "string", TypeArgs: []any{1}}

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
		// Only include TypeArgs if precision/scale are actually provided
		if precision.Valid && scale.Valid {
			return TypeMapping{AlabType: "decimal", TypeArgs: []any{int(precision.Int64), int(scale.Int64)}}
		} else if precision.Valid {
			return TypeMapping{AlabType: "decimal", TypeArgs: []any{int(precision.Int64)}}
		}
		return TypeMapping{AlabType: "decimal"}

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
// Parses type parameters from strings like VARCHAR(100) or NUMERIC(10,2).
func MapSQLiteType(sqlType string) TypeMapping {
	// Parse base type and arguments from strings like "VARCHAR(100)" or "NUMERIC(10,2)"
	baseType, args := parseSQLiteType(sqlType)
	upper := strings.ToUpper(baseType)

	switch {
	// Text types
	case upper == "TEXT", upper == "CLOB":
		return TypeMapping{AlabType: "text"}

	case strings.Contains(upper, "CHAR"), strings.Contains(upper, "VARCHAR"):
		// VARCHAR(n), CHARACTER(n), CHAR(n)
		if len(args) > 0 {
			return TypeMapping{AlabType: "string", TypeArgs: []any{args[0]}}
		}
		// CHAR without length defaults to length 1
		if upper == "CHARACTER" || upper == "CHAR" {
			return TypeMapping{AlabType: "string", TypeArgs: []any{1}}
		}
		return TypeMapping{AlabType: "string", TypeArgs: []any{255}}

	// Integer types
	case upper == "INTEGER", upper == "INT", upper == "TINYINT",
		upper == "SMALLINT", upper == "MEDIUMINT", upper == "BIGINT",
		upper == "INT2", upper == "INT8":
		return TypeMapping{AlabType: "integer"}

	// Float types
	case upper == "REAL", upper == "DOUBLE", upper == "FLOAT":
		return TypeMapping{AlabType: "float"}

	// Decimal/Numeric types
	case upper == "NUMERIC", upper == "DECIMAL":
		// Parse precision and scale from args
		if len(args) >= 2 {
			return TypeMapping{AlabType: "decimal", TypeArgs: []any{args[0], args[1]}}
		} else if len(args) == 1 {
			return TypeMapping{AlabType: "decimal", TypeArgs: []any{args[0]}}
		}
		return TypeMapping{AlabType: "decimal"}

	// Boolean
	case upper == "BOOLEAN", upper == "BOOL":
		return TypeMapping{AlabType: "boolean"}

	// Binary
	case upper == "BLOB":
		return TypeMapping{AlabType: "base64"}

	// Date/Time types
	case upper == "DATE":
		return TypeMapping{AlabType: "date"}

	case upper == "TIME":
		return TypeMapping{AlabType: "time"}

	case upper == "DATETIME", upper == "TIMESTAMP":
		return TypeMapping{AlabType: "datetime"}

	default:
		// SQLite is lenient with types, default to text
		return TypeMapping{AlabType: "text"}
	}
}

// parseSQLiteType extracts base type and arguments from SQLite type string.
// Examples:
//   - "VARCHAR(100)" -> ("VARCHAR", [100])
//   - "NUMERIC(10,2)" -> ("NUMERIC", [10, 2])
//   - "INTEGER" -> ("INTEGER", [])
func parseSQLiteType(sqlType string) (string, []int) {
	sqlType = strings.TrimSpace(sqlType)

	// Match pattern: TYPE(args) or TYPE
	re := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_\s]*)\s*(?:\(([^)]+)\))?`)
	matches := re.FindStringSubmatch(sqlType)

	if len(matches) < 2 {
		return sqlType, nil
	}

	baseType := strings.TrimSpace(matches[1])

	// No arguments
	if len(matches) < 3 || matches[2] == "" {
		return baseType, nil
	}

	// Parse arguments (comma-separated integers)
	argsStr := matches[2]
	parts := strings.Split(argsStr, ",")
	var args []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if num, err := strconv.Atoi(part); err == nil {
			args = append(args, num)
		}
	}

	return baseType, args
}

// Removed heuristic type inference functions.
// Introspection now uses only database-provided metadata for deterministic behavior.
