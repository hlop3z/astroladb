package introspect

import (
	"database/sql"
	"regexp"
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
		return TypeMapping{AlabType: "date_time"}

	case upper == "TIMESTAMP", upper == "TIMESTAMP WITHOUT TIME ZONE":
		return TypeMapping{AlabType: "date_time"}

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
func MapSQLiteType(sqlType string) TypeMapping {
	upper := strings.ToUpper(sqlType)

	switch {
	case upper == "TEXT":
		// Could be many types - need context from column name
		return TypeMapping{AlabType: "text"}

	case upper == "INTEGER":
		return TypeMapping{AlabType: "integer"}

	case upper == "REAL":
		return TypeMapping{AlabType: "float"}

	case upper == "BLOB":
		return TypeMapping{AlabType: "base64"}

	case upper == "NUMERIC":
		return TypeMapping{AlabType: "decimal", TypeArgs: []any{10, 2}}

	default:
		// SQLite is lenient with types, default to text
		return TypeMapping{AlabType: "text"}
	}
}

// uuidPattern matches UUID generation functions in defaults
var uuidPattern = regexp.MustCompile(`(?i)gen_random_uuid|uuid_generate|uuid\(\)`)

// InferAlabTypeFromContext uses column name and default to refine type inference.
// This helps with SQLite where types are more ambiguous.
func InferAlabTypeFromContext(col RawColumn, baseType TypeMapping) TypeMapping {
	name := strings.ToLower(col.Name)

	// Primary key "id" column with TEXT type -> id type
	if name == "id" && col.IsPrimaryKey && baseType.AlabType == "text" {
		return TypeMapping{AlabType: "id"}
	}

	// *_id columns with TEXT type -> uuid (foreign keys)
	if strings.HasSuffix(name, "_id") && baseType.AlabType == "text" {
		return TypeMapping{AlabType: "uuid"}
	}

	// Check default for UUID generation
	if col.Default.Valid {
		def := col.Default.String
		if uuidPattern.MatchString(def) {
			return TypeMapping{AlabType: "id"}
		}
	}

	// created_at / updated_at with TEXT -> date_time
	if (name == "created_at" || name == "updated_at") && baseType.AlabType == "text" {
		return TypeMapping{AlabType: "date_time"}
	}

	// *_date columns with TEXT -> date
	if strings.HasSuffix(name, "_date") && baseType.AlabType == "text" {
		return TypeMapping{AlabType: "date"}
	}

	// *_time columns with TEXT -> time
	if strings.HasSuffix(name, "_time") && baseType.AlabType == "text" {
		return TypeMapping{AlabType: "time"}
	}

	// *_day / *_at columns with TEXT -> date_time
	if (strings.HasSuffix(name, "_day") || strings.HasSuffix(name, "_at")) && baseType.AlabType == "text" {
		return TypeMapping{AlabType: "date_time"}
	}

	// Boolean-like column names with INTEGER -> boolean (SQLite stores booleans as 0/1)
	if baseType.AlabType == "integer" {
		if isBooleanColumnName(name) {
			return TypeMapping{AlabType: "boolean"}
		}
	}

	return baseType
}

// isBooleanColumnName checks if a column name suggests it's a boolean.
func isBooleanColumnName(name string) bool {
	// Common boolean prefixes
	boolPrefixes := []string{"is_", "has_", "can_", "should_", "was_", "will_", "did_"}
	for _, prefix := range boolPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	// Common boolean suffixes/names
	boolNames := []string{"featured", "active", "enabled", "disabled", "verified", "published", "archived", "deleted", "hidden", "visible", "public", "private"}
	for _, bn := range boolNames {
		if name == bn {
			return true
		}
	}

	return false
}
