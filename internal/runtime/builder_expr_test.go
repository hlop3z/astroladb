package runtime

import (
	"testing"

	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// convertExpr Tests (ColBuilder)
// -----------------------------------------------------------------------------

func TestColBuilder_convertExpr(t *testing.T) {
	t.Run("fnexpr_pointer", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		expr := &FnExpr{Fn: "upper", Args: []any{"test"}}
		result := cb.convertExpr(expr)

		// Should convert to map
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map[string]any, got %T", result)
		}

		if resultMap["fn"] != "upper" {
			t.Errorf("fn = %v, want %q", resultMap["fn"], "upper")
		}
	})

	t.Run("goja_value_with_fnexpr", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		// Create a FnExpr and wrap it in a Goja value
		expr := &FnExpr{Fn: "lower", Args: []any{"TEST"}, Col: "name"}
		gojaVal := vm.ToValue(expr)

		result := cb.convertExpr(gojaVal)

		// Should convert to map via ExprToMap
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map[string]any, got %T", result)
		}

		if resultMap["fn"] != "lower" {
			t.Errorf("fn = %v, want %q", resultMap["fn"], "lower")
		}
		if resultMap["col"] != "name" {
			t.Errorf("col = %v, want %q", resultMap["col"], "name")
		}
	})

	t.Run("goja_value_primitive", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		// Test with a primitive value
		gojaVal := vm.ToValue("hello")
		result := cb.convertExpr(gojaVal)

		if result != "hello" {
			t.Errorf("Result = %v, want %q", result, "hello")
		}
	})

	t.Run("goja_value_number", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		gojaVal := vm.ToValue(42)
		result := cb.convertExpr(gojaVal)

		// Goja exports numbers as int64
		if result != int64(42) {
			t.Errorf("Result = %v (%T), want int64(42)", result, result)
		}
	})

	t.Run("non_goja_non_fnexpr", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		// Regular Go value (not Goja, not FnExpr)
		result := cb.convertExpr("plain string")
		if result != "plain string" {
			t.Errorf("Result = %v, want %q", result, "plain string")
		}

		result = cb.convertExpr(123)
		if result != 123 {
			t.Errorf("Result = %v, want 123", result)
		}
	})

	t.Run("fnexpr_with_args", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		expr := &FnExpr{
			Fn:   "concat",
			Args: []any{"Hello", " ", "World"},
		}

		result := cb.convertExpr(expr)

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map[string]any, got %T", result)
		}

		if resultMap["fn"] != "concat" {
			t.Errorf("fn = %v, want %q", resultMap["fn"], "concat")
		}

		args, ok := resultMap["args"].([]any)
		if !ok {
			t.Fatalf("args should be []any, got %T", resultMap["args"])
		}

		if len(args) != 3 {
			t.Errorf("args length = %d, want 3", len(args))
		}
	})

	t.Run("fnexpr_with_col_reference", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		expr := &FnExpr{
			Col: "username",
		}

		result := cb.convertExpr(expr)

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map[string]any, got %T", result)
		}

		if resultMap["col"] != "username" {
			t.Errorf("col = %v, want %q", resultMap["col"], "username")
		}
	})
}
