package introspect

import (
	"database/sql"
	"testing"
)

// TestMapPostgresTypeEdgeCases tests edge cases and less common types
func TestMapPostgresTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		pgType       string
		charMaxLen   sql.NullInt64
		numPrecision sql.NullInt64
		numScale     sql.NullInt64
		wantType     string
		wantArgs     []any
	}{
		// Numeric edge cases
		{
			name:         "decimal_with_precision_and_scale",
			pgType:       "NUMERIC",
			charMaxLen:   sql.NullInt64{},
			numPrecision: sql.NullInt64{Valid: true, Int64: 10},
			numScale:     sql.NullInt64{Valid: true, Int64: 2},
			wantType:     "decimal",
			wantArgs:     []any{10, 2},
		},
		{
			name:         "decimal_with_only_precision",
			pgType:       "NUMERIC",
			charMaxLen:   sql.NullInt64{},
			numPrecision: sql.NullInt64{Valid: true, Int64: 10},
			numScale:     sql.NullInt64{},
			wantType:     "decimal",
			wantArgs:     []any{10},
		},
		{
			name:         "decimal_without_precision",
			pgType:       "NUMERIC",
			charMaxLen:   sql.NullInt64{},
			numPrecision: sql.NullInt64{},
			numScale:     sql.NullInt64{},
			wantType:     "decimal",
			wantArgs:     nil,
		},
		{
			name:     "bigint",
			pgType:   "BIGINT",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "smallint",
			pgType:   "SMALLINT",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "real",
			pgType:   "REAL",
			wantType: "float",
			wantArgs: nil,
		},
		{
			name:     "double_precision",
			pgType:   "DOUBLE PRECISION",
			wantType: "float",
			wantArgs: nil,
		},
		// String type edge cases
		{
			name:     "text",
			pgType:   "TEXT",
			wantType: "text",
			wantArgs: nil,
		},
		{
			name:       "char_with_length",
			pgType:     "CHARACTER",
			charMaxLen: sql.NullInt64{Valid: true, Int64: 10},
			wantType:   "string",
			wantArgs:   []any{10},
		},
		{
			name:       "char_without_length",
			pgType:     "CHARACTER",
			charMaxLen: sql.NullInt64{},
			wantType:   "string",
			wantArgs:   []any{1},
		},
		// Boolean
		{
			name:     "boolean",
			pgType:   "BOOLEAN",
			wantType: "boolean",
			wantArgs: nil,
		},
		{
			name:     "bool",
			pgType:   "BOOL",
			wantType: "boolean",
			wantArgs: nil,
		},
		// Date/Time types
		{
			name:     "date",
			pgType:   "DATE",
			wantType: "date",
			wantArgs: nil,
		},
		{
			name:     "time",
			pgType:   "TIME",
			wantType: "time",
			wantArgs: nil,
		},
		{
			name:     "time_without_timezone",
			pgType:   "TIME WITHOUT TIME ZONE",
			wantType: "time",
			wantArgs: nil,
		},
		{
			name:     "timestamp",
			pgType:   "TIMESTAMP",
			wantType: "datetime",
			wantArgs: nil,
		},
		{
			name:     "timestamp_without_timezone",
			pgType:   "TIMESTAMP WITHOUT TIME ZONE",
			wantType: "datetime",
			wantArgs: nil,
		},
		{
			name:     "timestamptz",
			pgType:   "TIMESTAMPTZ",
			wantType: "datetime",
			wantArgs: nil,
		},
		// Binary types
		{
			name:     "bytea",
			pgType:   "BYTEA",
			wantType: "base64",
			wantArgs: nil,
		},
		// JSON types
		{
			name:     "json",
			pgType:   "JSON",
			wantType: "json",
			wantArgs: nil,
		},
		{
			name:     "jsonb",
			pgType:   "JSONB",
			wantType: "json",
			wantArgs: nil,
		},
		// Unknown type
		{
			name:     "unknown_type_fallback",
			pgType:   "UNKNOWN_TYPE",
			wantType: "text",
			wantArgs: nil,
		},
		// Case insensitivity
		{
			name:     "lowercase_uuid",
			pgType:   "uuid",
			wantType: "uuid",
			wantArgs: nil,
		},
		{
			name:       "mixed_case_varchar",
			pgType:     "VarChar",
			charMaxLen: sql.NullInt64{Valid: true, Int64: 200},
			wantType:   "string",
			wantArgs:   []any{200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapPostgresType(tt.pgType, tt.charMaxLen, tt.numPrecision, tt.numScale)

			if result.AlabType != tt.wantType {
				t.Errorf("AlabType = %q, want %q", result.AlabType, tt.wantType)
			}

			if len(result.TypeArgs) != len(tt.wantArgs) {
				t.Errorf("TypeArgs length = %d, want %d", len(result.TypeArgs), len(tt.wantArgs))
			}

			for i, arg := range tt.wantArgs {
				if i >= len(result.TypeArgs) {
					t.Errorf("Missing TypeArgs[%d], want %v", i, arg)
					continue
				}
				if result.TypeArgs[i] != arg {
					t.Errorf("TypeArgs[%d] = %v, want %v", i, result.TypeArgs[i], arg)
				}
			}
		})
	}
}

// TestMapSQLiteTypeEdgeCases tests SQLite type mapping edge cases
func TestMapSQLiteTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		sqlType  string
		wantType string
		wantArgs []any
	}{
		// Integer variations
		{
			name:     "INT",
			sqlType:  "INT",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "INTEGER",
			sqlType:  "INTEGER",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "TINYINT",
			sqlType:  "TINYINT",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "SMALLINT",
			sqlType:  "SMALLINT",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "MEDIUMINT",
			sqlType:  "MEDIUMINT",
			wantType: "integer",
			wantArgs: nil,
		},
		{
			name:     "BIGINT",
			sqlType:  "BIGINT",
			wantType: "integer",
			wantArgs: nil,
		},
		// Real/Float variations
		{
			name:     "REAL",
			sqlType:  "REAL",
			wantType: "float",
			wantArgs: nil,
		},
		{
			name:     "DOUBLE",
			sqlType:  "DOUBLE",
			wantType: "float",
			wantArgs: nil,
		},
		{
			name:     "FLOAT",
			sqlType:  "FLOAT",
			wantType: "float",
			wantArgs: nil,
		},
		// Text variations
		{
			name:     "TEXT",
			sqlType:  "TEXT",
			wantType: "text",
			wantArgs: nil,
		},
		{
			name:     "CLOB",
			sqlType:  "CLOB",
			wantType: "text",
			wantArgs: nil,
		},
		// String with length
		{
			name:     "VARCHAR(100)",
			sqlType:  "VARCHAR(100)",
			wantType: "string",
			wantArgs: []any{100},
		},
		{
			name:     "CHARACTER(50)",
			sqlType:  "CHARACTER(50)",
			wantType: "string",
			wantArgs: []any{50},
		},
		{
			name:     "CHAR(10)",
			sqlType:  "CHAR(10)",
			wantType: "string",
			wantArgs: []any{10},
		},
		// Decimal with precision
		{
			name:     "NUMERIC(10,2)",
			sqlType:  "NUMERIC(10,2)",
			wantType: "decimal",
			wantArgs: []any{10, 2},
		},
		{
			name:     "DECIMAL(8)",
			sqlType:  "DECIMAL(8)",
			wantType: "decimal",
			wantArgs: []any{8},
		},
		// Boolean
		{
			name:     "BOOLEAN",
			sqlType:  "BOOLEAN",
			wantType: "boolean",
			wantArgs: nil,
		},
		// Blob
		{
			name:     "BLOB",
			sqlType:  "BLOB",
			wantType: "base64",
			wantArgs: nil,
		},
		// Date/Time (SQLite stores as TEXT/INTEGER)
		{
			name:     "DATE",
			sqlType:  "DATE",
			wantType: "date",
			wantArgs: nil,
		},
		{
			name:     "DATETIME",
			sqlType:  "DATETIME",
			wantType: "datetime",
			wantArgs: nil,
		},
		{
			name:     "TIMESTAMP",
			sqlType:  "TIMESTAMP",
			wantType: "datetime",
			wantArgs: nil,
		},
		// Unknown type fallback
		{
			name:     "UNKNOWN",
			sqlType:  "UNKNOWN",
			wantType: "text",
			wantArgs: nil,
		},
		// Case insensitivity
		{
			name:     "lowercase_text",
			sqlType:  "text",
			wantType: "text",
			wantArgs: nil,
		},
		{
			name:     "MixedCase",
			sqlType:  "Integer",
			wantType: "integer",
			wantArgs: nil,
		},
		// With extra whitespace
		{
			name:     "VARCHAR with spaces",
			sqlType:  "VARCHAR(  255  )",
			wantType: "string",
			wantArgs: []any{255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapSQLiteType(tt.sqlType)

			if result.AlabType != tt.wantType {
				t.Errorf("AlabType = %q, want %q", result.AlabType, tt.wantType)
			}

			if len(result.TypeArgs) != len(tt.wantArgs) {
				t.Errorf("TypeArgs length = %d, want %d (got %v)", len(result.TypeArgs), len(tt.wantArgs), result.TypeArgs)
			}

			for i, arg := range tt.wantArgs {
				if i >= len(result.TypeArgs) {
					t.Errorf("Missing TypeArgs[%d], want %v", i, arg)
					continue
				}
				if result.TypeArgs[i] != arg {
					t.Errorf("TypeArgs[%d] = %v, want %v", i, result.TypeArgs[i], arg)
				}
			}
		})
	}
}

// TestTypeMapping_ConsistencyBetweenDialects tests that similar types map consistently
func TestTypeMapping_ConsistencyBetweenDialects(t *testing.T) {
	tests := []struct {
		name         string
		postgresType string
		sqliteType   string
		wantAlabType string
	}{
		{
			name:         "integer_types",
			postgresType: "INTEGER",
			sqliteType:   "INTEGER",
			wantAlabType: "integer",
		},
		{
			name:         "float_types",
			postgresType: "DOUBLE PRECISION",
			sqliteType:   "REAL",
			wantAlabType: "float",
		},
		{
			name:         "text_types",
			postgresType: "TEXT",
			sqliteType:   "TEXT",
			wantAlabType: "text",
		},
		{
			name:         "boolean_types",
			postgresType: "BOOLEAN",
			sqliteType:   "BOOLEAN",
			wantAlabType: "boolean",
		},
		{
			name:         "date_types",
			postgresType: "DATE",
			sqliteType:   "DATE",
			wantAlabType: "date",
		},
		{
			name:         "datetime_types",
			postgresType: "TIMESTAMP",
			sqliteType:   "DATETIME",
			wantAlabType: "datetime",
		},
		{
			name:         "binary_types",
			postgresType: "BYTEA",
			sqliteType:   "BLOB",
			wantAlabType: "base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgResult := MapPostgresType(tt.postgresType, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
			sqliteResult := MapSQLiteType(tt.sqliteType)

			if pgResult.AlabType != tt.wantAlabType {
				t.Errorf("Postgres: AlabType = %q, want %q", pgResult.AlabType, tt.wantAlabType)
			}
			if sqliteResult.AlabType != tt.wantAlabType {
				t.Errorf("SQLite: AlabType = %q, want %q", sqliteResult.AlabType, tt.wantAlabType)
			}
			if pgResult.AlabType != sqliteResult.AlabType {
				t.Errorf("Inconsistent mapping: Postgres=%q, SQLite=%q", pgResult.AlabType, sqliteResult.AlabType)
			}
		})
	}
}
