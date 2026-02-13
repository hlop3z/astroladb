package builder

import (
	"testing"
)

// -----------------------------------------------------------------------------
// SemanticType Registry Tests
// -----------------------------------------------------------------------------

func TestSemanticTypes_Exists(t *testing.T) {
	// Verify all expected semantic types exist
	expectedTypes := []string{
		"email", "username", "password_hash", "phone",
		"name", "title", "slug", "body", "summary",
		"url", "ip", "user_agent",
		"money", "percentage", "counter", "quantity", "rating", "duration",
		"token", "code", "country", "currency", "locale", "timezone", "color",
		"markdown", "html",
		"flag",
	}

	for _, typeName := range expectedTypes {
		t.Run(typeName, func(t *testing.T) {
			if _, ok := SemanticTypes[typeName]; !ok {
				t.Errorf("SemanticTypes should contain %q", typeName)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// GetSemanticType Tests
// -----------------------------------------------------------------------------

func TestGetSemanticType(t *testing.T) {
	t.Run("existing_type", func(t *testing.T) {
		st := GetSemanticType("email")
		if st == nil {
			t.Fatal("GetSemanticType('email') returned nil")
		}
		if st.BaseType != "string" {
			t.Errorf("email.BaseType = %q, want 'string'", st.BaseType)
		}
		if st.Format != "email" {
			t.Errorf("email.Format = %q, want 'email'", st.Format)
		}
		if st.Length != 255 {
			t.Errorf("email.Length = %d, want 255", st.Length)
		}
	})

	t.Run("non_existent_type", func(t *testing.T) {
		st := GetSemanticType("nonexistent")
		if st != nil {
			t.Error("GetSemanticType('nonexistent') should return nil")
		}
	})

	t.Run("money_type", func(t *testing.T) {
		st := GetSemanticType("money")
		if st == nil {
			t.Fatal("GetSemanticType('money') returned nil")
		}
		if st.BaseType != "decimal" {
			t.Errorf("money.BaseType = %q, want 'decimal'", st.BaseType)
		}
		if st.Precision != 19 {
			t.Errorf("money.Precision = %d, want 19", st.Precision)
		}
		if st.Scale != 4 {
			t.Errorf("money.Scale = %d, want 4", st.Scale)
		}
		if st.Min == nil || *st.Min != 0 {
			t.Error("money.Min should be 0")
		}
	})

	t.Run("flag_type", func(t *testing.T) {
		st := GetSemanticType("flag")
		if st == nil {
			t.Fatal("GetSemanticType('flag') returned nil")
		}
		if st.BaseType != "boolean" {
			t.Errorf("flag.BaseType = %q, want 'boolean'", st.BaseType)
		}
		if st.Default != false {
			t.Errorf("flag.Default = %v, want false", st.Default)
		}
	})

	t.Run("slug_type", func(t *testing.T) {
		st := GetSemanticType("slug")
		if st == nil {
			t.Fatal("GetSemanticType('slug') returned nil")
		}
		if !st.Unique {
			t.Error("slug.Unique should be true")
		}
		if st.Pattern == "" {
			t.Error("slug.Pattern should not be empty")
		}
	})

	t.Run("password_hash_type", func(t *testing.T) {
		st := GetSemanticType("password_hash")
		if st == nil {
			t.Fatal("GetSemanticType('password_hash') returned nil")
		}
		if !st.Hidden {
			t.Error("password_hash.Hidden should be true")
		}
	})

	t.Run("percentage_type", func(t *testing.T) {
		st := GetSemanticType("percentage")
		if st == nil {
			t.Fatal("GetSemanticType('percentage') returned nil")
		}
		if st.Min == nil || *st.Min != 0 {
			t.Error("percentage.Min should be 0")
		}
		if st.Max == nil || *st.Max != 100 {
			t.Error("percentage.Max should be 100")
		}
	})

	t.Run("username_type", func(t *testing.T) {
		st := GetSemanticType("username")
		if st == nil {
			t.Fatal("GetSemanticType('username') returned nil")
		}
		if st.Min == nil || *st.Min != 3 {
			t.Error("username.Min should be 3")
		}
		if st.Pattern == "" {
			t.Error("username.Pattern should not be empty")
		}
	})
}

// -----------------------------------------------------------------------------
// IsSemanticType Tests
// -----------------------------------------------------------------------------

func TestIsSemanticType(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"email", true},
		{"money", true},
		{"flag", true},
		{"slug", true},
		{"nonexistent", false},
		{"", false},
		{"string", false}, // base type, not semantic
		{"integer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSemanticType(tt.name)
			if got != tt.want {
				t.Errorf("IsSemanticType(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// SemanticTypeNames Tests
// -----------------------------------------------------------------------------

func TestSemanticTypeNames(t *testing.T) {
	names := SemanticTypeNames()

	// Should return non-empty list
	if len(names) == 0 {
		t.Error("SemanticTypeNames() should not be empty")
	}

	// Should match the number of types in the registry
	if len(names) != len(SemanticTypes) {
		t.Errorf("SemanticTypeNames() returned %d names, but registry has %d types",
			len(names), len(SemanticTypes))
	}

	// All names should exist in the registry
	for _, name := range names {
		if _, ok := SemanticTypes[name]; !ok {
			t.Errorf("name %q from SemanticTypeNames() not in registry", name)
		}
	}
}

// -----------------------------------------------------------------------------
// SemanticType.ApplyTo Tests
// -----------------------------------------------------------------------------

func TestSemanticType_ApplyTo(t *testing.T) {
	t.Run("string_type_with_length", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Length:   100,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if typeArgs, ok := col["type_args"].([]any); ok {
			if len(typeArgs) != 1 || typeArgs[0] != 100 {
				t.Errorf("type_args = %v, want [100]", typeArgs)
			}
		} else {
			t.Error("type_args should be set for string type")
		}
	})

	t.Run("decimal_type_with_precision", func(t *testing.T) {
		st := SemanticType{
			BaseType:  "decimal",
			Precision: 10,
			Scale:     2,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if typeArgs, ok := col["type_args"].([]any); ok {
			if len(typeArgs) != 2 || typeArgs[0] != 10 || typeArgs[1] != 2 {
				t.Errorf("type_args = %v, want [10, 2]", typeArgs)
			}
		} else {
			t.Error("type_args should be set for decimal type")
		}
	})

	t.Run("with_format", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Format:   "email",
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["format"] != "email" {
			t.Errorf("format = %v, want 'email'", col["format"])
		}
	})

	t.Run("with_pattern", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Pattern:  "^[a-z]+$",
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["pattern"] != "^[a-z]+$" {
			t.Errorf("pattern = %v, want '^[a-z]+$'", col["pattern"])
		}
	})

	t.Run("with_unique", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Unique:   true,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["unique"] != true {
			t.Error("unique should be true")
		}
	})

	t.Run("unique_false_not_set", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Unique:   false,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if _, exists := col["unique"]; exists {
			t.Error("unique should not be set when false")
		}
	})

	t.Run("with_default", func(t *testing.T) {
		st := SemanticType{
			BaseType: "integer",
			Default:  0,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["default"] != 0 {
			t.Errorf("default = %v, want 0", col["default"])
		}
	})

	t.Run("with_hidden", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Hidden:   true,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["hidden"] != true {
			t.Error("hidden should be true")
		}
	})

	t.Run("with_min", func(t *testing.T) {
		minVal := 0.0
		st := SemanticType{
			BaseType: "integer",
			Min:      &minVal,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["min"] != 0.0 {
			t.Errorf("min = %v, want 0.0", col["min"])
		}
	})

	t.Run("with_max", func(t *testing.T) {
		maxVal := 100.0
		st := SemanticType{
			BaseType: "integer",
			Max:      &maxVal,
		}
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["max"] != 100.0 {
			t.Errorf("max = %v, want 100.0", col["max"])
		}
	})

	t.Run("full_email_type", func(t *testing.T) {
		st := GetSemanticType("email")
		col := make(map[string]any)
		st.ApplyTo(col)

		if col["format"] != "email" {
			t.Error("email format not applied")
		}
		if col["pattern"] == nil {
			t.Error("email pattern not applied")
		}
		if typeArgs, ok := col["type_args"].([]any); !ok || len(typeArgs) != 1 {
			t.Error("email type_args not applied")
		}
	})
}

// -----------------------------------------------------------------------------
// ptr Helper Tests
// -----------------------------------------------------------------------------

func TestPtr(t *testing.T) {
	val := 42.5
	p := ptr(val)

	if p == nil {
		t.Fatal("ptr() returned nil")
	}
	if *p != val {
		t.Errorf("*ptr(42.5) = %v, want 42.5", *p)
	}
}

// -----------------------------------------------------------------------------
// optsFromSemanticType Tests
// -----------------------------------------------------------------------------

func TestOptsFromSemanticType(t *testing.T) {
	t.Run("string_type", func(t *testing.T) {
		st := SemanticType{
			BaseType: "string",
			Length:   100,
			Format:   "email",
			Pattern:  "test",
		}
		opts := optsFromSemanticType(st)

		// Should have options for length, format, pattern
		if len(opts) < 3 {
			t.Errorf("expected at least 3 options, got %d", len(opts))
		}
	})

	t.Run("decimal_type", func(t *testing.T) {
		st := SemanticType{
			BaseType:  "decimal",
			Precision: 10,
			Scale:     2,
		}
		opts := optsFromSemanticType(st)

		// Should have option for precision/scale
		if len(opts) < 1 {
			t.Errorf("expected at least 1 option, got %d", len(opts))
		}
	})

	t.Run("with_all_options", func(t *testing.T) {
		minVal := 0.0
		maxVal := 100.0
		st := SemanticType{
			BaseType: "string",
			Length:   50,
			Format:   "custom",
			Pattern:  "^test$",
			Unique:   true,
			Default:  "default",
			Hidden:   true,
			Min:      &minVal,
			Max:      &maxVal,
		}
		opts := optsFromSemanticType(st)

		// Should have many options
		if len(opts) < 7 {
			t.Errorf("expected at least 7 options, got %d", len(opts))
		}
	})

	t.Run("empty_type", func(t *testing.T) {
		st := SemanticType{}
		opts := optsFromSemanticType(st)

		// Should have no options
		if len(opts) != 0 {
			t.Errorf("expected 0 options for empty type, got %d", len(opts))
		}
	})
}

// -----------------------------------------------------------------------------
// Specific Semantic Types Properties Tests
// -----------------------------------------------------------------------------

func TestSemanticTypes_Properties(t *testing.T) {
	t.Run("url_properties", func(t *testing.T) {
		st := SemanticTypes["url"]
		if st.Length != 2048 {
			t.Errorf("url.Length = %d, want 2048", st.Length)
		}
		if st.Format != "uri" {
			t.Errorf("url.Format = %q, want 'uri'", st.Format)
		}
	})

	t.Run("country_properties", func(t *testing.T) {
		st := SemanticTypes["country"]
		if st.Length != 2 {
			t.Errorf("country.Length = %d, want 2", st.Length)
		}
		if st.Pattern != "^[A-Z]{2}$" {
			t.Errorf("country.Pattern = %q, want '^[A-Z]{2}$'", st.Pattern)
		}
	})

	t.Run("currency_properties", func(t *testing.T) {
		st := SemanticTypes["currency"]
		if st.Length != 3 {
			t.Errorf("currency.Length = %d, want 3", st.Length)
		}
		if st.Pattern != "^[A-Z]{3}$" {
			t.Errorf("currency.Pattern = %q, want '^[A-Z]{3}$'", st.Pattern)
		}
	})

	t.Run("color_properties", func(t *testing.T) {
		st := SemanticTypes["color"]
		if st.Length != 7 {
			t.Errorf("color.Length = %d, want 7", st.Length)
		}
		if st.Pattern != "^#[0-9A-Fa-f]{6}$" {
			t.Errorf("color.Pattern = %q, want '^#[0-9A-Fa-f]{6}$'", st.Pattern)
		}
	})

	t.Run("rating_properties", func(t *testing.T) {
		st := SemanticTypes["rating"]
		if st.BaseType != "decimal" {
			t.Errorf("rating.BaseType = %q, want 'decimal'", st.BaseType)
		}
		if st.Min == nil || *st.Min != 0 {
			t.Error("rating.Min should be 0")
		}
		if st.Max == nil || *st.Max != 5 {
			t.Error("rating.Max should be 5")
		}
	})

	t.Run("counter_properties", func(t *testing.T) {
		st := SemanticTypes["counter"]
		if st.BaseType != "integer" {
			t.Errorf("counter.BaseType = %q, want 'integer'", st.BaseType)
		}
		if st.Default != 0 {
			t.Errorf("counter.Default = %v, want 0", st.Default)
		}
	})

	t.Run("body_properties", func(t *testing.T) {
		st := SemanticTypes["body"]
		if st.BaseType != "text" {
			t.Errorf("body.BaseType = %q, want 'text'", st.BaseType)
		}
	})

	t.Run("markdown_properties", func(t *testing.T) {
		st := SemanticTypes["markdown"]
		if st.BaseType != "text" {
			t.Errorf("markdown.BaseType = %q, want 'text'", st.BaseType)
		}
	})
}
