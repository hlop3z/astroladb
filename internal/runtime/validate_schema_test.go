package runtime

import (
	"testing"
)

// -----------------------------------------------------------------------------
// ValidateSchema Tests
// -----------------------------------------------------------------------------

func TestSandbox_ValidateSchemaAdditional(t *testing.T) {
	t.Run("invalid_identifier", func(t *testing.T) {
		sb := NewSandbox(nil)

		// Table name with invalid characters
		code := `
		table("test.Invalid-Name", {
			id: { type: "uuid", primary_key: true }
		})
		`

		err := sb.ValidateSchema(code)
		if err == nil {
			t.Error("Expected validation error for invalid identifier")
		}
	})

	t.Run("javascript_syntax_error", func(t *testing.T) {
		sb := NewSandbox(nil)

		// Syntax error in JavaScript
		code := `
		table("test.users", {
			id: { type: "uuid"
		})
		`

		err := sb.ValidateSchema(code)
		if err == nil {
			t.Error("Expected error for JavaScript syntax error")
		}
	})

	t.Run("undefined_variable", func(t *testing.T) {
		sb := NewSandbox(nil)

		// Reference to undefined variable
		code := `
		table("test.users", {
			id: { type: undefinedVariable }
		})
		`

		err := sb.ValidateSchema(code)
		if err == nil {
			t.Error("Expected error for undefined variable")
		}
	})

	t.Run("empty_code", func(t *testing.T) {
		sb := NewSandbox(nil)

		err := sb.ValidateSchema("")
		// Empty code should pass (no operations to validate)
		if err != nil {
			t.Errorf("ValidateSchema failed for empty code: %v", err)
		}
	})

	t.Run("invalid_column_type", func(t *testing.T) {
		sb := NewSandbox(nil)

		code := `
		table("test.users", {
			id: { type: "invalid_type_that_does_not_exist" }
		})
		`

		err := sb.ValidateSchema(code)
		if err == nil {
			t.Error("Expected validation error for invalid column type")
		}
	})

	t.Run("string_without_length", func(t *testing.T) {
		sb := NewSandbox(nil)

		code := `
		table("test.users", {
			id: { type: "uuid", primary_key: true },
			name: { type: "string" }
		})
		`

		err := sb.ValidateSchema(code)
		if err == nil {
			t.Error("Expected validation error for string without length")
		}
	})
}
