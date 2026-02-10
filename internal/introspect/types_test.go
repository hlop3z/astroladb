package introspect

import (
	"database/sql"
	"testing"
)

// -----------------------------------------------------------------------------
// MapPostgresType Tests
// -----------------------------------------------------------------------------

func TestMapPostgresType(t *testing.T) {
	t.Run("uuid", func(t *testing.T) {
		result := MapPostgresType("UUID", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "uuid" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "uuid")
		}
		if len(result.TypeArgs) != 0 {
			t.Errorf("TypeArgs = %v, want empty", result.TypeArgs)
		}
	})

	t.Run("varchar_with_length", func(t *testing.T) {
		result := MapPostgresType("VARCHAR", sql.NullInt64{Valid: true, Int64: 100}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "string" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "string")
		}
		if len(result.TypeArgs) != 1 || result.TypeArgs[0] != 100 {
			t.Errorf("TypeArgs = %v, want [100]", result.TypeArgs)
		}
	})

	t.Run("varchar_without_length", func(t *testing.T) {
		result := MapPostgresType("VARCHAR", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "string" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "string")
		}
		if len(result.TypeArgs) != 1 || result.TypeArgs[0] != 255 {
			t.Errorf("TypeArgs = %v, want [255]", result.TypeArgs)
		}
	})

	t.Run("character_varying", func(t *testing.T) {
		result := MapPostgresType("CHARACTER VARYING", sql.NullInt64{Valid: true, Int64: 50}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "string" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "string")
		}
		if len(result.TypeArgs) != 1 || result.TypeArgs[0] != 50 {
			t.Errorf("TypeArgs = %v, want [50]", result.TypeArgs)
		}
	})

	t.Run("text", func(t *testing.T) {
		result := MapPostgresType("TEXT", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "text" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "text")
		}
	})

	t.Run("integer_types", func(t *testing.T) {
		tests := []string{"INTEGER", "INT", "INT4"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "integer" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "integer")
				}
			})
		}
	})

	t.Run("smallint_types", func(t *testing.T) {
		tests := []string{"SMALLINT", "INT2"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "integer" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "integer")
				}
			})
		}
	})

	t.Run("bigint_types", func(t *testing.T) {
		tests := []string{"BIGINT", "INT8"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "integer" {
					t.Errorf("AlabType = %q, want %q (bigint mapped to integer)", result.AlabType, "integer")
				}
			})
		}
	})

	t.Run("float_types", func(t *testing.T) {
		tests := []string{"REAL", "FLOAT4"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "float" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "float")
				}
			})
		}
	})

	t.Run("double_precision", func(t *testing.T) {
		tests := []string{"DOUBLE PRECISION", "FLOAT8"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "float" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "float")
				}
			})
		}
	})

	t.Run("numeric_with_precision", func(t *testing.T) {
		result := MapPostgresType("NUMERIC", sql.NullInt64{}, sql.NullInt64{Valid: true, Int64: 15}, sql.NullInt64{Valid: true, Int64: 4})
		if result.AlabType != "decimal" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "decimal")
		}
		if len(result.TypeArgs) != 2 || result.TypeArgs[0] != 15 || result.TypeArgs[1] != 4 {
			t.Errorf("TypeArgs = %v, want [15, 4]", result.TypeArgs)
		}
	})

	t.Run("decimal_without_precision", func(t *testing.T) {
		result := MapPostgresType("DECIMAL", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "decimal" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "decimal")
		}
		// When precision/scale are not provided, TypeArgs should be empty
		if len(result.TypeArgs) != 0 {
			t.Errorf("TypeArgs = %v, want [] (no precision metadata)", result.TypeArgs)
		}
	})

	t.Run("boolean", func(t *testing.T) {
		tests := []string{"BOOLEAN", "BOOL"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "boolean" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "boolean")
				}
			})
		}
	})

	t.Run("date", func(t *testing.T) {
		result := MapPostgresType("DATE", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "date" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "date")
		}
	})

	t.Run("time", func(t *testing.T) {
		tests := []string{"TIME", "TIME WITHOUT TIME ZONE"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "time" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "time")
				}
			})
		}
	})

	t.Run("timestamp_with_tz", func(t *testing.T) {
		tests := []string{"TIMESTAMP WITH TIME ZONE", "TIMESTAMPTZ"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "datetime" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "datetime")
				}
			})
		}
	})

	t.Run("timestamp_without_tz", func(t *testing.T) {
		tests := []string{"TIMESTAMP", "TIMESTAMP WITHOUT TIME ZONE"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "datetime" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "datetime")
				}
			})
		}
	})

	t.Run("json", func(t *testing.T) {
		tests := []string{"JSON", "JSONB"}
		for _, sqlType := range tests {
			t.Run(sqlType, func(t *testing.T) {
				result := MapPostgresType(sqlType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
				if result.AlabType != "json" {
					t.Errorf("AlabType = %q, want %q", result.AlabType, "json")
				}
			})
		}
	})

	t.Run("bytea", func(t *testing.T) {
		result := MapPostgresType("BYTEA", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "base64" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "base64")
		}
	})

	t.Run("unknown_type", func(t *testing.T) {
		result := MapPostgresType("CUSTOMTYPE", sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "text" {
			t.Errorf("AlabType = %q, want %q (unknown types default to text)", result.AlabType, "text")
		}
	})

	t.Run("lowercase_handling", func(t *testing.T) {
		result := MapPostgresType("varchar", sql.NullInt64{Valid: true, Int64: 50}, sql.NullInt64{}, sql.NullInt64{})
		if result.AlabType != "string" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "string")
		}
	})
}

// -----------------------------------------------------------------------------
// MapSQLiteType Tests
// -----------------------------------------------------------------------------

func TestMapSQLiteType(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		result := MapSQLiteType("TEXT")
		if result.AlabType != "text" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "text")
		}
	})

	t.Run("integer", func(t *testing.T) {
		result := MapSQLiteType("INTEGER")
		if result.AlabType != "integer" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "integer")
		}
	})

	t.Run("real", func(t *testing.T) {
		result := MapSQLiteType("REAL")
		if result.AlabType != "float" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "float")
		}
	})

	t.Run("blob", func(t *testing.T) {
		result := MapSQLiteType("BLOB")
		if result.AlabType != "base64" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "base64")
		}
	})

	t.Run("numeric", func(t *testing.T) {
		result := MapSQLiteType("NUMERIC")
		if result.AlabType != "decimal" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "decimal")
		}
		if len(result.TypeArgs) != 0 {
			t.Errorf("TypeArgs = %v, want empty (SQLite doesn't preserve precision)", result.TypeArgs)
		}
	})

	t.Run("decimal", func(t *testing.T) {
		result := MapSQLiteType("DECIMAL")
		if result.AlabType != "decimal" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "decimal")
		}
	})

	t.Run("datetime", func(t *testing.T) {
		result := MapSQLiteType("DATETIME")
		if result.AlabType != "datetime" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "datetime")
		}
	})

	t.Run("timestamp", func(t *testing.T) {
		result := MapSQLiteType("TIMESTAMP")
		if result.AlabType != "datetime" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "datetime")
		}
	})

	t.Run("date", func(t *testing.T) {
		result := MapSQLiteType("DATE")
		if result.AlabType != "date" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "date")
		}
	})

	t.Run("time", func(t *testing.T) {
		result := MapSQLiteType("TIME")
		if result.AlabType != "time" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "time")
		}
	})

	t.Run("varchar_with_length", func(t *testing.T) {
		result := MapSQLiteType("VARCHAR(100)")
		if result.AlabType != "string" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "string")
		}
		if len(result.TypeArgs) != 1 || result.TypeArgs[0] != 100 {
			t.Errorf("TypeArgs = %v, want [100]", result.TypeArgs)
		}
	})

	t.Run("unknown_custom_type", func(t *testing.T) {
		result := MapSQLiteType("CUSTOMTYPE")
		if result.AlabType != "text" {
			t.Errorf("AlabType = %q, want %q (unknown types default to text)", result.AlabType, "text")
		}
	})

	t.Run("lowercase_handling", func(t *testing.T) {
		result := MapSQLiteType("integer")
		if result.AlabType != "integer" {
			t.Errorf("AlabType = %q, want %q", result.AlabType, "integer")
		}
	})

	t.Run("empty_string", func(t *testing.T) {
		result := MapSQLiteType("")
		if result.AlabType != "text" {
			t.Errorf("AlabType = %q, want %q (empty defaults to text)", result.AlabType, "text")
		}
	})
}

// -----------------------------------------------------------------------------
// TypeMapping Struct Tests
// -----------------------------------------------------------------------------

func TestTypeMapping(t *testing.T) {
	t.Run("with_type_args", func(t *testing.T) {
		tm := TypeMapping{
			AlabType: "string",
			TypeArgs: []any{100},
		}
		if tm.AlabType != "string" {
			t.Errorf("AlabType = %q, want %q", tm.AlabType, "string")
		}
		if len(tm.TypeArgs) != 1 || tm.TypeArgs[0] != 100 {
			t.Errorf("TypeArgs = %v, want [100]", tm.TypeArgs)
		}
	})

	t.Run("without_type_args", func(t *testing.T) {
		tm := TypeMapping{
			AlabType: "text",
		}
		if tm.AlabType != "text" {
			t.Errorf("AlabType = %q, want %q", tm.AlabType, "text")
		}
		if tm.TypeArgs != nil && len(tm.TypeArgs) != 0 {
			t.Errorf("TypeArgs = %v, want nil or empty", tm.TypeArgs)
		}
	})
}
