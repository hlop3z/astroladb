package types

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// Type Registry Tests
// -----------------------------------------------------------------------------

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantNil  bool
	}{
		// Built-in types should exist
		{"id", "id", false},
		{"string", "string", false},
		{"text", "text", false},
		{"integer", "integer", false},
		{"float", "float", false},
		{"decimal", "decimal", false},
		{"boolean", "boolean", false},
		{"date", "date", false},
		{"time", "time", false},
		{"date_time", "date_time", false},
		{"uuid", "uuid", false},
		{"json", "json", false},
		{"base64", "base64", false},
		{"enum", "enum", false},

		// Non-existent types
		{"unknown", "unknown_type", true},
		{"bigint", "bigint", true}, // Forbidden, not registered
		{"serial", "serial", true}, // Forbidden, not registered
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get(tt.typeName)
			if (got == nil) != tt.wantNil {
				if tt.wantNil {
					t.Errorf("Get(%q) should return nil", tt.typeName)
				} else {
					t.Errorf("Get(%q) should not return nil", tt.typeName)
				}
			}
		})
	}
}

func TestExists(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		// Built-in types
		{"id", true},
		{"string", true},
		{"text", true},
		{"integer", true},
		{"boolean", true},
		{"json", true},

		// Non-existent
		{"unknown", false},
		{"bigint", false},
		{"int64", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := Exists(tt.typeName)
			if got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestAll(t *testing.T) {
	types := All()

	if len(types) == 0 {
		t.Fatal("All() should return registered types")
	}

	// Check that we have the expected minimum number of types
	expectedTypes := []string{
		"id", "string", "text", "integer", "float", "decimal",
		"boolean", "date", "time", "date_time", "uuid", "json", "base64", "enum",
	}

	typeMap := make(map[string]bool)
	for _, td := range types {
		typeMap[td.Name] = true
	}

	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("All() missing expected type: %s", expected)
		}
	}
}

// -----------------------------------------------------------------------------
// Built-in Type Definition Tests
// -----------------------------------------------------------------------------

func TestBuiltInTypeDefinitions(t *testing.T) {
	tests := []struct {
		name        string
		wantJSName  string
		wantGoType  string
		wantHasArgs bool
		wantMinArgs int
		wantMaxArgs int
	}{
		// UUID primary key
		{
			name:        "id",
			wantJSName:  "id",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// Variable-length string
		{
			name:        "string",
			wantJSName:  "string",
			wantGoType:  "string",
			wantHasArgs: true,
			wantMinArgs: 1,
			wantMaxArgs: 1,
		},
		// Unlimited text
		{
			name:        "text",
			wantJSName:  "text",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// 32-bit integer (JS-safe)
		{
			name:        "integer",
			wantJSName:  "integer",
			wantGoType:  "int32",
			wantHasArgs: false,
		},
		// 32-bit float (JS-safe)
		{
			name:        "float",
			wantJSName:  "float",
			wantGoType:  "float32",
			wantHasArgs: false,
		},
		// Decimal (string-serialized)
		{
			name:        "decimal",
			wantJSName:  "decimal",
			wantGoType:  "string",
			wantHasArgs: true,
			wantMinArgs: 2,
			wantMaxArgs: 2,
		},
		// Boolean
		{
			name:        "boolean",
			wantJSName:  "boolean",
			wantGoType:  "bool",
			wantHasArgs: false,
		},
		// Date
		{
			name:        "date",
			wantJSName:  "date",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// Time
		{
			name:        "time",
			wantJSName:  "time",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// DateTime
		{
			name:        "date_time",
			wantJSName:  "date_time",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// UUID
		{
			name:        "uuid",
			wantJSName:  "uuid",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// JSON
		{
			name:        "json",
			wantJSName:  "json",
			wantGoType:  "any",
			wantHasArgs: false,
		},
		// Base64
		{
			name:        "base64",
			wantJSName:  "base64",
			wantGoType:  "string",
			wantHasArgs: false,
		},
		// Enum
		{
			name:        "enum",
			wantJSName:  "enum",
			wantGoType:  "string",
			wantHasArgs: true,
			wantMinArgs: 1,
			wantMaxArgs: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := Get(tt.name)
			if td == nil {
				t.Fatalf("type %q not found", tt.name)
			}

			if td.JSName != tt.wantJSName {
				t.Errorf("JSName = %q, want %q", td.JSName, tt.wantJSName)
			}
			if td.GoType != tt.wantGoType {
				t.Errorf("GoType = %q, want %q", td.GoType, tt.wantGoType)
			}
			if td.HasArgs != tt.wantHasArgs {
				t.Errorf("HasArgs = %v, want %v", td.HasArgs, tt.wantHasArgs)
			}
			if tt.wantHasArgs {
				if td.MinArgs != tt.wantMinArgs {
					t.Errorf("MinArgs = %d, want %d", td.MinArgs, tt.wantMinArgs)
				}
				if td.MaxArgs != tt.wantMaxArgs {
					t.Errorf("MaxArgs = %d, want %d", td.MaxArgs, tt.wantMaxArgs)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Forbidden Types Tests
// -----------------------------------------------------------------------------

func TestIsForbidden(t *testing.T) {
	tests := []struct {
		typeName   string
		wantForbid bool
		wantReason string
	}{
		// Forbidden types (JS-unsafe)
		{"bigint", true, "JavaScript loses precision"},
		{"double", true, "JavaScript has precision issues"},
		{"serial", true, "Auto-increment IDs are not allowed"},
		{"bigserial", true, "Auto-increment IDs are not allowed"},

		// Allowed types
		{"id", false, ""},
		{"string", false, ""},
		{"integer", false, ""},
		{"float", false, ""},
		{"decimal", false, ""},
		{"uuid", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			forbidden, reason := IsForbidden(tt.typeName)
			if forbidden != tt.wantForbid {
				t.Errorf("IsForbidden(%q) = %v, want %v", tt.typeName, forbidden, tt.wantForbid)
			}
			if tt.wantForbid && tt.wantReason != "" {
				if reason == "" {
					t.Errorf("expected reason for forbidden type %q", tt.typeName)
				}
				// Check that reason contains expected substring
				if tt.wantReason != "" && len(reason) > 0 {
					// Partial match is fine
					found := false
					for _, part := range []string{tt.wantReason} {
						if contains(reason, part) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("reason = %q, should contain %q", reason, tt.wantReason)
					}
				}
			}
		})
	}
}

// Helper to check substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// Validate Tests
// -----------------------------------------------------------------------------

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		args     []any
		wantErr  bool
		errCode  alerr.Code
	}{
		// Valid types without args
		{"id_no_args", "id", nil, false, ""},
		{"text_no_args", "text", nil, false, ""},
		{"integer_no_args", "integer", nil, false, ""},
		{"boolean_no_args", "boolean", nil, false, ""},
		{"date_no_args", "date", nil, false, ""},
		{"uuid_no_args", "uuid", nil, false, ""},
		{"json_no_args", "json", nil, false, ""},

		// Valid types with required args
		{"string_with_length", "string", []any{255}, false, ""},
		{"decimal_with_precision", "decimal", []any{10, 2}, false, ""},
		{"enum_with_values", "enum", []any{"active", "inactive"}, false, ""},

		// Invalid: forbidden types
		{"bigint_forbidden", "bigint", nil, true, alerr.ErrInvalidType},
		{"double_forbidden", "double", nil, true, alerr.ErrInvalidType},
		{"serial_forbidden", "serial", nil, true, alerr.ErrInvalidType},
		{"bigserial_forbidden", "bigserial", nil, true, alerr.ErrInvalidType},

		// Invalid: unknown type
		{"unknown_type", "unknown_type", nil, true, alerr.ErrInvalidType},

		// Invalid: type doesn't accept args but got some
		{"id_with_args", "id", []any{36}, true, alerr.ErrInvalidType},
		{"integer_with_args", "integer", []any{32}, true, alerr.ErrInvalidType},

		// Invalid: type requires args but got none
		{"string_no_args", "string", nil, true, alerr.ErrInvalidType},
		{"string_no_args_empty", "string", []any{}, true, alerr.ErrInvalidType},
		{"decimal_no_args", "decimal", nil, true, alerr.ErrInvalidType},

		// Invalid: too few args
		{"decimal_one_arg", "decimal", []any{10}, true, alerr.ErrInvalidType},

		// Invalid: too many args
		{"string_too_many", "string", []any{255, 100}, true, alerr.ErrInvalidType},
		{"decimal_too_many", "decimal", []any{10, 2, 1}, true, alerr.ErrInvalidType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.typeName, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q, %v) error = %v, wantErr %v", tt.typeName, tt.args, err, tt.wantErr)
				return
			}
			if err != nil && tt.errCode != "" {
				code := alerr.GetErrorCode(err)
				if code != tt.errCode {
					t.Errorf("error code = %v, want %v", code, tt.errCode)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// OpenAPI Type Mapping Tests
// -----------------------------------------------------------------------------

func TestOpenAPITypeMappings(t *testing.T) {
	tests := []struct {
		typeName   string
		wantType   string
		wantFormat string
	}{
		// String types
		{"id", "string", "uuid"},
		{"string", "string", ""},
		{"text", "string", ""},
		{"uuid", "string", "uuid"},
		{"base64", "string", "byte"},

		// Number types
		{"integer", "integer", "int32"},
		{"float", "number", "float"},
		{"decimal", "string", "decimal"}, // String to preserve precision

		// Boolean
		{"boolean", "boolean", ""},

		// Date/Time types (all serialized as strings)
		{"date", "string", "date"},
		{"time", "string", "time"},
		{"date_time", "string", "date-time"},

		// JSON
		{"json", "object", ""},

		// Enum
		{"enum", "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			td := Get(tt.typeName)
			if td == nil {
				t.Fatalf("type %q not found", tt.typeName)
			}

			if td.OpenAPI.Type != tt.wantType {
				t.Errorf("OpenAPI.Type = %q, want %q", td.OpenAPI.Type, tt.wantType)
			}
			if td.OpenAPI.Format != tt.wantFormat {
				t.Errorf("OpenAPI.Format = %q, want %q", td.OpenAPI.Format, tt.wantFormat)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Standard Formats Tests
// -----------------------------------------------------------------------------

func TestGetFormat(t *testing.T) {
	tests := []struct {
		name       string
		wantName   string
		wantRFC    string
		hasPattern bool
	}{
		{"email", "email", "RFC 5322", true},
		{"uri", "uri", "RFC 3986", true},
		{"uuid", "uuid", "RFC 4122", true},
		{"date", "date", "ISO 8601", true},
		{"time", "time", "RFC 3339", true},
		{"date_time", "date-time", "RFC 3339", true},
		{"hostname", "hostname", "RFC 1123", true},
		{"ipv4", "ipv4", "RFC 791", true},
		{"ipv6", "ipv6", "RFC 4291", true},
		{"password", "password", "", false}, // No pattern for password
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := GetFormat(tt.name)
			if f == nil {
				t.Fatalf("GetFormat(%q) returned nil", tt.name)
			}

			if f.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", f.Name, tt.wantName)
			}
			if f.RFC != tt.wantRFC {
				t.Errorf("RFC = %q, want %q", f.RFC, tt.wantRFC)
			}
			if tt.hasPattern && f.Pattern == "" {
				t.Errorf("expected pattern for format %q", tt.name)
			}
			if !tt.hasPattern && f.Pattern != "" {
				t.Errorf("unexpected pattern for format %q: %s", tt.name, f.Pattern)
			}
		})
	}
}

func TestGetFormatUnknown(t *testing.T) {
	f := GetFormat("unknown_format")
	if f != nil {
		t.Errorf("GetFormat(unknown) should return nil, got %v", f)
	}
}

func TestStandardFormats(t *testing.T) {
	expectedFormats := []string{
		"email", "uri", "uuid", "date", "time", "date_time",
		"hostname", "ipv4", "ipv6", "password",
	}

	for _, name := range expectedFormats {
		t.Run(name, func(t *testing.T) {
			if _, ok := StandardFormats[name]; !ok {
				t.Errorf("StandardFormats missing %q", name)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// JS-Safe Type Guarantees Tests
// -----------------------------------------------------------------------------

func TestJSSafeTypes(t *testing.T) {
	// All registered types should be JS-safe
	for _, td := range All() {
		t.Run(td.Name, func(t *testing.T) {
			// Should not be int64 or float64
			if td.GoType == "int64" || td.GoType == "float64" {
				t.Errorf("type %q uses JS-unsafe Go type: %s", td.Name, td.GoType)
			}
		})
	}

	// Forbidden types should not be in registry
	forbiddenNames := []string{"bigint", "double", "serial", "bigserial", "int64", "float64"}
	for _, name := range forbiddenNames {
		t.Run("forbidden_"+name, func(t *testing.T) {
			if Exists(name) {
				t.Errorf("forbidden type %q should not be in registry", name)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Type Argument Validation Tests
// -----------------------------------------------------------------------------

func TestTypeArgumentValidation(t *testing.T) {
	// Types that don't accept arguments
	noArgsTypes := []string{"id", "text", "integer", "float", "boolean", "date", "time", "date_time", "uuid", "json", "base64"}
	for _, typeName := range noArgsTypes {
		t.Run(typeName+"_no_args", func(t *testing.T) {
			td := Get(typeName)
			if td == nil {
				t.Fatalf("type %q not found", typeName)
			}
			if td.HasArgs {
				t.Errorf("type %q should not accept arguments", typeName)
			}
		})
	}

	// Types that require arguments
	argsTypes := map[string]struct {
		min int
		max int
	}{
		"string":  {1, 1},
		"decimal": {2, 2},
		"enum":    {1, 100},
	}

	for typeName, expected := range argsTypes {
		t.Run(typeName+"_requires_args", func(t *testing.T) {
			td := Get(typeName)
			if td == nil {
				t.Fatalf("type %q not found", typeName)
			}
			if !td.HasArgs {
				t.Errorf("type %q should accept arguments", typeName)
			}
			if td.MinArgs != expected.min {
				t.Errorf("type %q MinArgs = %d, want %d", typeName, td.MinArgs, expected.min)
			}
			if td.MaxArgs != expected.max {
				t.Errorf("type %q MaxArgs = %d, want %d", typeName, td.MaxArgs, expected.max)
			}
		})
	}
}
