package introspect

import (
	"reflect"
	"testing"
)

// -----------------------------------------------------------------------------
// parseEnumValues Tests
// -----------------------------------------------------------------------------

func TestParseEnumValues(t *testing.T) {
	t.Run("simple_enum", func(t *testing.T) {
		checkSQL := "CHECK(status IN ('draft', 'published', 'archived'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"draft", "published", "archived"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("lowercase_in", func(t *testing.T) {
		checkSQL := "CHECK(status in ('active', 'inactive'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"active", "inactive"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("no_space_before_parenthesis", func(t *testing.T) {
		checkSQL := "CHECK(status IN('a', 'b'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"a", "b"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("double_quotes", func(t *testing.T) {
		checkSQL := `CHECK(type IN ("option1", "option2"))`
		values := parseEnumValues(checkSQL)

		expected := []string{"option1", "option2"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("mixed_spacing", func(t *testing.T) {
		checkSQL := "CHECK(status IN ('one' , 'two' ,  'three'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"one", "two", "three"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("no_in_clause", func(t *testing.T) {
		checkSQL := "CHECK(age > 0)"
		values := parseEnumValues(checkSQL)

		if values != nil {
			t.Errorf("parseEnumValues() = %v, want nil", values)
		}
	})

	t.Run("empty_string", func(t *testing.T) {
		values := parseEnumValues("")
		if values != nil {
			t.Errorf("parseEnumValues() = %v, want nil", values)
		}
	})

	t.Run("malformed_no_closing_paren", func(t *testing.T) {
		checkSQL := "CHECK(status IN ('a', 'b'"
		values := parseEnumValues(checkSQL)

		if values != nil {
			t.Errorf("parseEnumValues() = %v, want nil", values)
		}
	})

	t.Run("malformed_no_opening_paren", func(t *testing.T) {
		checkSQL := "CHECK status IN 'a', 'b')"
		values := parseEnumValues(checkSQL)

		if values != nil {
			t.Errorf("parseEnumValues() = %v, want nil", values)
		}
	})

	t.Run("single_value", func(t *testing.T) {
		checkSQL := "CHECK(status IN ('only_one'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"only_one"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("values_with_underscores", func(t *testing.T) {
		checkSQL := "CHECK(type IN ('user_admin', 'user_regular', 'user_guest'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"user_admin", "user_regular", "user_guest"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("numeric_values", func(t *testing.T) {
		// Even though numeric, they should be extracted as strings
		checkSQL := "CHECK(priority IN ('1', '2', '3'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"1", "2", "3"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("mixed_case_in_keyword", func(t *testing.T) {
		checkSQL := "CHECK(status In ('A', 'B'))"
		values := parseEnumValues(checkSQL)

		expected := []string{"A", "B"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})

	t.Run("empty_values_filtered", func(t *testing.T) {
		checkSQL := "CHECK(status IN ('', 'valid'))"
		values := parseEnumValues(checkSQL)

		// Empty strings are filtered out
		expected := []string{"valid"}
		if !reflect.DeepEqual(values, expected) {
			t.Errorf("parseEnumValues() = %v, want %v", values, expected)
		}
	})
}

// -----------------------------------------------------------------------------
// sqliteIntrospector struct test (basic functionality)
// -----------------------------------------------------------------------------

func TestSQLiteIntrospectorStruct(t *testing.T) {
	// Just verify the struct can be created
	// Actual introspection requires a real database
	introspector := &sqliteIntrospector{
		db:      nil,
		dialect: nil,
	}

	if introspector == nil {
		t.Error("sqliteIntrospector should be creatable")
	}
}
