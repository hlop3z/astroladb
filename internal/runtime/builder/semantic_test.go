package builder

import (
	"sort"
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

	// Should be sorted
	if !sort.StringsAreSorted(names) {
		t.Error("SemanticTypeNames() should return sorted names")
	}
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
