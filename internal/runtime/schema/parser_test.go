package schema

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestNewSchemaParser tests the parser constructor
func TestNewSchemaParser(t *testing.T) {
	parser := NewSchemaParser()
	if parser == nil {
		t.Fatal("NewSchemaParser should return a non-nil parser")
	}
}

// TestParseColumnDef_BasicColumn tests parsing a basic column definition
func TestParseColumnDef_BasicColumn(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name": "email",
		"type": "string",
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Name != "email" {
		t.Errorf("Expected name 'email', got '%s'", result.Name)
	}
	if result.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", result.Type)
	}
}

// TestParseColumnDef_WithTypeArgs tests parsing column with type arguments
func TestParseColumnDef_WithTypeArgs(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":      "username",
		"type":      "varchar",
		"type_args": []any{255},
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Name != "username" {
		t.Errorf("Expected name 'username', got '%s'", result.Name)
	}
	if len(result.TypeArgs) != 1 || result.TypeArgs[0] != 255 {
		t.Errorf("Expected type_args [255], got %v", result.TypeArgs)
	}
}

// TestParseColumnDef_WithNullable tests parsing column with nullable flag
func TestParseColumnDef_WithNullable(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name     string
		nullable bool
	}{
		{"nullable true", true},
		{"nullable false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]any{
				"name":     "field",
				"type":     "string",
				"nullable": tt.nullable,
			}

			result := parser.ParseColumnDef(input)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Nullable != tt.nullable {
				t.Errorf("Expected nullable %v, got %v", tt.nullable, result.Nullable)
			}
			if !result.NullableSet {
				t.Error("NullableSet should be true when nullable is provided")
			}
		})
	}
}

// TestParseColumnDef_WithConstraints tests parsing column with constraints
func TestParseColumnDef_WithConstraints(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":        "email",
		"type":        "string",
		"unique":      true,
		"primary_key": false,
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !result.Unique {
		t.Error("Expected unique to be true")
	}
	if result.PrimaryKey {
		t.Error("Expected primary_key to be false")
	}
}

// TestParseColumnDef_WithDefault tests parsing column with default value
func TestParseColumnDef_WithDefault(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name         string
		defaultValue any
	}{
		{"string default", "active"},
		{"number default", float64(42)},
		{"boolean default", true},
		{"nil default", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]any{
				"name":    "field",
				"type":    "string",
				"default": tt.defaultValue,
			}

			result := parser.ParseColumnDef(input)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if !result.DefaultSet {
				t.Error("DefaultSet should be true when default is provided")
			}
		})
	}
}

// TestParseColumnDef_WithSQLDefault tests parsing column with SQL expression default
func TestParseColumnDef_WithSQLDefault(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name": "created_at",
		"type": "timestamp",
		"default": map[string]any{
			"_type": "sql_expr",
			"expr":  "NOW()",
		},
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !result.DefaultSet {
		t.Error("DefaultSet should be true")
	}

	sqlExpr, ok := result.Default.(*ast.SQLExpr)
	if !ok {
		t.Fatal("Default should be *ast.SQLExpr")
	}
	if sqlExpr.Expr != "NOW()" {
		t.Errorf("Expected expr 'NOW()', got '%s'", sqlExpr.Expr)
	}
}

// TestParseColumnDef_WithBackfill tests parsing column with backfill value
func TestParseColumnDef_WithBackfill(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":     "status",
		"type":     "string",
		"backfill": "pending",
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Backfill != "pending" {
		t.Errorf("Expected backfill 'pending', got '%v'", result.Backfill)
	}
	if !result.BackfillSet {
		t.Error("BackfillSet should be true when backfill is provided")
	}
}

// TestParseColumnDef_WithReference tests parsing column with reference
func TestParseColumnDef_WithReference(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name": "user_id",
		"type": "uuid",
		"reference": map[string]any{
			"table":     "auth.users",
			"column":    "id",
			"on_delete": "CASCADE",
		},
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Reference == nil {
		t.Fatal("Expected reference to be set")
	}
	if result.Reference.Table != "auth.users" {
		t.Errorf("Expected table 'auth.users', got '%s'", result.Reference.Table)
	}
	if result.Reference.Column != "id" {
		t.Errorf("Expected column 'id', got '%s'", result.Reference.Column)
	}
	if result.Reference.OnDelete != "CASCADE" {
		t.Errorf("Expected on_delete 'CASCADE', got '%s'", result.Reference.OnDelete)
	}
}

// TestParseColumnDef_WithValidation tests parsing column with validation constraints
func TestParseColumnDef_WithValidation(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":    "age",
		"type":    "integer",
		"min":     float64(0),
		"max":     float64(120),
		"pattern": "^[0-9]+$",
		"format":  "email",
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Min == nil || *result.Min != 0 {
		t.Errorf("Expected min 0, got %v", result.Min)
	}
	if result.Max == nil || *result.Max != 120 {
		t.Errorf("Expected max 120, got %v", result.Max)
	}
	if result.Pattern != "^[0-9]+$" {
		t.Errorf("Expected pattern '^[0-9]+$', got '%s'", result.Pattern)
	}
	if result.Format != "email" {
		t.Errorf("Expected format 'email', got '%s'", result.Format)
	}
}

// TestParseColumnDef_WithDocumentation tests parsing column with docs
func TestParseColumnDef_WithDocumentation(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":       "legacy_field",
		"type":       "string",
		"docs":       "User's email address",
		"deprecated": "Use new_field instead",
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Docs != "User's email address" {
		t.Errorf("Expected docs 'User's email address', got '%s'", result.Docs)
	}
	if result.Deprecated != "Use new_field instead" {
		t.Errorf("Expected deprecated 'Use new_field instead', got '%s'", result.Deprecated)
	}
}

// TestParseColumnDef_WithComputed tests parsing computed column
func TestParseColumnDef_WithComputed(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":     "full_name",
		"type":     "string",
		"computed": "first_name || ' ' || last_name",
	}

	result := parser.ParseColumnDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Computed != "first_name || ' ' || last_name" {
		t.Errorf("Expected computed expression, got '%v'", result.Computed)
	}
}

// TestParseColumnDef_InvalidInputs tests error handling for invalid inputs
func TestParseColumnDef_InvalidInputs(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"nil input", nil},
		{"string input", "not a map"},
		{"number input", 42},
		{"empty map", map[string]any{}},
		{"map without name", map[string]any{"type": "string"}},
		{"map with empty name", map[string]any{"name": "", "type": "string"}},
		{"metadata entry", map[string]any{"_type": "relationship", "name": "ref"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseColumnDef(tt.input)
			if result != nil {
				t.Error("Should return nil for invalid input")
			}
		})
	}
}

// TestParseReference_Complete tests parsing a complete reference
func TestParseReference_Complete(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"table":     "auth.users",
		"column":    "id",
		"on_delete": "CASCADE",
		"on_update": "RESTRICT",
	}

	result := parser.ParseReference(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Table != "auth.users" {
		t.Errorf("Expected table 'auth.users', got '%s'", result.Table)
	}
	if result.Column != "id" {
		t.Errorf("Expected column 'id', got '%s'", result.Column)
	}
	if result.OnDelete != "CASCADE" {
		t.Errorf("Expected on_delete 'CASCADE', got '%s'", result.OnDelete)
	}
	if result.OnUpdate != "RESTRICT" {
		t.Errorf("Expected on_update 'RESTRICT', got '%s'", result.OnUpdate)
	}
}

// TestParseReference_Minimal tests parsing reference with minimal fields
func TestParseReference_Minimal(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"table": "users",
	}

	result := parser.ParseReference(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Table != "users" {
		t.Errorf("Expected table 'users', got '%s'", result.Table)
	}
	if result.Column != "id" {
		t.Errorf("Expected default column 'id', got '%s'", result.Column)
	}
}

// TestParseIndexDef_Complete tests parsing a complete index definition
func TestParseIndexDef_Complete(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"name":    "idx_user_email",
		"columns": []any{"user_id", "email"},
		"unique":  true,
		"where":   "deleted_at IS NULL",
	}

	result := parser.ParseIndexDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Name != "idx_user_email" {
		t.Errorf("Expected name 'idx_user_email', got '%s'", result.Name)
	}
	if len(result.Columns) != 2 || result.Columns[0] != "user_id" || result.Columns[1] != "email" {
		t.Errorf("Expected columns [user_id, email], got %v", result.Columns)
	}
	if !result.Unique {
		t.Error("Expected unique to be true")
	}
	if result.Where != "deleted_at IS NULL" {
		t.Errorf("Expected where 'deleted_at IS NULL', got '%s'", result.Where)
	}
}

// TestParseIndexDef_Minimal tests parsing index with minimal fields
func TestParseIndexDef_Minimal(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"columns": []any{"email"},
	}

	result := parser.ParseIndexDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Columns) != 1 || result.Columns[0] != "email" {
		t.Errorf("Expected columns [email], got %v", result.Columns)
	}
	if result.Unique {
		t.Error("Expected unique to be false")
	}
}

// TestParseIndexDef_InvalidInputs tests error handling for invalid index inputs
func TestParseIndexDef_InvalidInputs(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"nil input", nil},
		{"string input", "not a map"},
		{"number input", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseIndexDef(tt.input)
			if result != nil {
				t.Error("Should return nil for invalid input")
			}
		})
	}
}

// TestParseIndexDef_NonStringColumns tests handling non-string columns
func TestParseIndexDef_NonStringColumns(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"columns": []any{"valid", 42, "another_valid", nil, true},
	}

	result := parser.ParseIndexDef(input)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	// Should only include string values
	if len(result.Columns) != 2 || result.Columns[0] != "valid" || result.Columns[1] != "another_valid" {
		t.Errorf("Expected columns [valid, another_valid], got %v", result.Columns)
	}
}

// TestConvertValue_SQLExpression tests converting SQL expression markers
func TestConvertValue_SQLExpression(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"_type": "sql_expr",
		"expr":  "CURRENT_TIMESTAMP",
	}

	result := parser.convertValue(input)
	sqlExpr, ok := result.(*ast.SQLExpr)
	if !ok {
		t.Fatal("Should convert to *ast.SQLExpr")
	}
	if sqlExpr.Expr != "CURRENT_TIMESTAMP" {
		t.Errorf("Expected expr 'CURRENT_TIMESTAMP', got '%s'", sqlExpr.Expr)
	}
}

// TestConvertValue_RegularMap tests that regular maps are preserved
func TestConvertValue_RegularMap(t *testing.T) {
	parser := NewSchemaParser()

	input := map[string]any{
		"key": "value",
		"num": 42,
	}

	result := parser.convertValue(input)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Should preserve regular map")
	}
	if m["key"] != "value" {
		t.Errorf("Expected key='value', got '%v'", m["key"])
	}
}

// TestConvertValue_PrimitiveTypes tests that primitive types are preserved
func TestConvertValue_PrimitiveTypes(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"string", "hello"},
		{"number", float64(42)},
		{"boolean", true},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.convertValue(tt.input)
			if result != tt.input {
				t.Errorf("Expected %v, got %v", tt.input, result)
			}
		})
	}
}

// TestParseColumns_ValidArray tests parsing valid column array
func TestParseColumns_ValidArray(t *testing.T) {
	parser := NewSchemaParser()

	input := []any{
		map[string]any{"name": "id", "type": "uuid"},
		map[string]any{"name": "email", "type": "string"},
		map[string]any{"name": "age", "type": "integer"},
	}

	result := parser.ParseColumns(input)
	if len(result) != 3 {
		t.Fatalf("Expected 3 columns, got %d", len(result))
	}
	if result[0].Name != "id" {
		t.Errorf("Expected first column 'id', got '%s'", result[0].Name)
	}
	if result[1].Name != "email" {
		t.Errorf("Expected second column 'email', got '%s'", result[1].Name)
	}
	if result[2].Name != "age" {
		t.Errorf("Expected third column 'age', got '%s'", result[2].Name)
	}
}

// TestParseColumns_TypedSlice tests parsing []map[string]any
func TestParseColumns_TypedSlice(t *testing.T) {
	parser := NewSchemaParser()

	input := []map[string]any{
		{"name": "id", "type": "uuid"},
		{"name": "email", "type": "string"},
	}

	result := parser.ParseColumns(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(result))
	}
	if result[0].Name != "id" {
		t.Errorf("Expected first column 'id', got '%s'", result[0].Name)
	}
}

// TestParseColumns_WithInvalidEntries tests filtering invalid entries
func TestParseColumns_WithInvalidEntries(t *testing.T) {
	parser := NewSchemaParser()

	input := []any{
		map[string]any{"name": "valid", "type": "string"},
		map[string]any{"type": "string"},        // Missing name
		map[string]any{"_type": "relationship"}, // Metadata entry
		"invalid",                               // Not a map
		42,                                      // Not a map
		map[string]any{"name": "another_valid", "type": "integer"},
	}

	result := parser.ParseColumns(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 valid columns, got %d", len(result))
	}
	if result[0].Name != "valid" {
		t.Errorf("Expected first column 'valid', got '%s'", result[0].Name)
	}
	if result[1].Name != "another_valid" {
		t.Errorf("Expected second column 'another_valid', got '%s'", result[1].Name)
	}
}

// TestParseColumns_InvalidInput tests handling invalid input types
func TestParseColumns_InvalidInput(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"string", "not an array"},
		{"number", 42},
		{"map", map[string]any{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseColumns(tt.input)
			if result != nil {
				t.Error("Should return nil for invalid input")
			}
		})
	}
}

// TestParseColumns_EmptyArray tests parsing empty array
func TestParseColumns_EmptyArray(t *testing.T) {
	parser := NewSchemaParser()

	result := parser.ParseColumns([]any{})
	if result == nil {
		t.Fatal("Should return non-nil slice for empty input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(result))
	}
}

// TestParseIndexes_ValidArray tests parsing valid index array
func TestParseIndexes_ValidArray(t *testing.T) {
	parser := NewSchemaParser()

	input := []any{
		map[string]any{"name": "idx_email", "columns": []any{"email"}},
		map[string]any{"name": "idx_user", "columns": []any{"user_id", "org_id"}},
	}

	result := parser.ParseIndexes(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 indexes, got %d", len(result))
	}
	if result[0].Name != "idx_email" {
		t.Errorf("Expected first index 'idx_email', got '%s'", result[0].Name)
	}
	if result[1].Name != "idx_user" {
		t.Errorf("Expected second index 'idx_user', got '%s'", result[1].Name)
	}
}

// TestParseIndexes_TypedSlice tests parsing []map[string]any
func TestParseIndexes_TypedSlice(t *testing.T) {
	parser := NewSchemaParser()

	input := []map[string]any{
		{"name": "idx_email", "columns": []any{"email"}},
		{"name": "idx_name", "columns": []any{"name"}},
	}

	result := parser.ParseIndexes(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 indexes, got %d", len(result))
	}
}

// TestParseIndexes_WithInvalidEntries tests filtering invalid entries
func TestParseIndexes_WithInvalidEntries(t *testing.T) {
	parser := NewSchemaParser()

	input := []any{
		map[string]any{"name": "valid_idx", "columns": []any{"col1"}},
		"invalid", // Not a map
		42,        // Not a map
		map[string]any{"name": "another_valid", "columns": []any{"col2"}},
	}

	result := parser.ParseIndexes(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 valid indexes, got %d", len(result))
	}
	if result[0].Name != "valid_idx" {
		t.Errorf("Expected first index 'valid_idx', got '%s'", result[0].Name)
	}
}

// TestParseIndexes_InvalidInput tests handling invalid input types
func TestParseIndexes_InvalidInput(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"string", "not an array"},
		{"number", 42},
		{"map", map[string]any{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseIndexes(tt.input)
			if result == nil {
				t.Fatal("Should return non-nil slice for invalid input")
			}
			if len(result) != 0 {
				t.Errorf("Expected empty slice, got length %d", len(result))
			}
		})
	}
}

// TestParseIndexes_EmptyArray tests parsing empty array
func TestParseIndexes_EmptyArray(t *testing.T) {
	parser := NewSchemaParser()

	result := parser.ParseIndexes([]any{})
	if result == nil {
		t.Fatal("Should return non-nil slice for empty input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(result))
	}
}

// TestParseStringSlice_StringSlice tests parsing []string
func TestParseStringSlice_StringSlice(t *testing.T) {
	parser := NewSchemaParser()

	input := []string{"first", "second", "third"}

	result := parser.ParseStringSlice(input)
	if len(result) != 3 {
		t.Fatalf("Expected 3 strings, got %d", len(result))
	}
	if result[0] != "first" || result[1] != "second" || result[2] != "third" {
		t.Errorf("Expected [first, second, third], got %v", result)
	}
}

// TestParseStringSlice_AnySlice tests parsing []any with strings
func TestParseStringSlice_AnySlice(t *testing.T) {
	parser := NewSchemaParser()

	input := []any{"alpha", "beta", "gamma"}

	result := parser.ParseStringSlice(input)
	if len(result) != 3 {
		t.Fatalf("Expected 3 strings, got %d", len(result))
	}
	if result[0] != "alpha" || result[1] != "beta" || result[2] != "gamma" {
		t.Errorf("Expected [alpha, beta, gamma], got %v", result)
	}
}

// TestParseStringSlice_MixedTypes tests filtering non-string values
func TestParseStringSlice_MixedTypes(t *testing.T) {
	parser := NewSchemaParser()

	input := []any{"valid", 42, "another", true, nil, "third"}

	result := parser.ParseStringSlice(input)
	if len(result) != 3 {
		t.Fatalf("Expected 3 strings, got %d", len(result))
	}
	if result[0] != "valid" || result[1] != "another" || result[2] != "third" {
		t.Errorf("Expected [valid, another, third], got %v", result)
	}
}

// TestParseStringSlice_EmptySlice tests parsing empty slice
func TestParseStringSlice_EmptySlice(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"empty []string", []string{}},
		{"empty []any", []any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseStringSlice(tt.input)
			if result == nil {
				t.Fatal("Should return non-nil slice")
			}
			if len(result) != 0 {
				t.Errorf("Expected empty slice, got length %d", len(result))
			}
		})
	}
}

// TestParseStringSlice_InvalidInput tests handling invalid input types
func TestParseStringSlice_InvalidInput(t *testing.T) {
	parser := NewSchemaParser()

	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"string", "not a slice"},
		{"number", 42},
		{"map", map[string]any{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseStringSlice(tt.input)
			if result != nil {
				t.Error("Should return nil for invalid input")
			}
		})
	}
}
