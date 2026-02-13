package builder

import (
	"testing"

	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// convertMap Tests
// -----------------------------------------------------------------------------

func TestFnBuilder_convertMap(t *testing.T) {
	t.Run("map_with_fn_and_args", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{
			"fn":   "upper",
			"args": []any{"hello"},
		}

		result := fb.convertMap(m)
		expr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}

		if expr.Fn != "upper" {
			t.Errorf("Fn = %q, want %q", expr.Fn, "upper")
		}
		if len(expr.Args) != 1 {
			t.Fatalf("Args length = %d, want 1", len(expr.Args))
		}
		if expr.Args[0] != "hello" {
			t.Errorf("Args[0] = %v, want %q", expr.Args[0], "hello")
		}
	})

	t.Run("map_with_fn_args_and_col", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{
			"fn":   "concat",
			"args": []any{"prefix_", map[string]any{"col": "name"}},
			"col":  "result_col",
		}

		result := fb.convertMap(m)
		expr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}

		if expr.Fn != "concat" {
			t.Errorf("Fn = %q, want %q", expr.Fn, "concat")
		}
		if expr.Col != "result_col" {
			t.Errorf("Col = %q, want %q", expr.Col, "result_col")
		}
		if len(expr.Args) != 2 {
			t.Fatalf("Args length = %d, want 2", len(expr.Args))
		}
	})

	t.Run("map_with_col_only", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{
			"col": "username",
		}

		result := fb.convertMap(m)
		expr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}

		if expr.Col != "username" {
			t.Errorf("Col = %q, want %q", expr.Col, "username")
		}
		if expr.Fn != "" {
			t.Errorf("Fn should be empty, got %q", expr.Fn)
		}
	})

	t.Run("regular_map_without_fn_or_col", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{
			"key1": "value1",
			"key2": 42,
		}

		result := fb.convertMap(m)
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map[string]any, got %T", result)
		}

		if resultMap["key1"] != "value1" {
			t.Errorf("key1 = %v, want %q", resultMap["key1"], "value1")
		}
		if resultMap["key2"] != 42 {
			t.Errorf("key2 = %v, want 42", resultMap["key2"])
		}
	})

	t.Run("map_with_fn_but_no_args", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{
			"fn": "now",
		}

		result := fb.convertMap(m)
		expr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}

		if expr.Fn != "now" {
			t.Errorf("Fn = %q, want %q", expr.Fn, "now")
		}
		if expr.Args != nil {
			t.Errorf("Args should be nil, got %v", expr.Args)
		}
	})

	t.Run("nested_fn_expr_in_args", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		// When convertMap is called directly with Go maps (not goja.Values),
		// nested maps in args are NOT automatically converted to FnExpr.
		// The convertArgs function only converts maps when they're goja.Values.
		// This test verifies that convertMap properly processes the top-level
		// structure and calls convertArgs on the args slice.
		m := map[string]any{
			"fn": "concat",
			"args": []any{
				map[string]any{
					"fn":   "upper",
					"args": []any{"hello"},
				},
				map[string]any{
					"fn":   "lower",
					"args": []any{"WORLD"},
				},
			},
		}

		result := fb.convertMap(m)
		expr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}

		if expr.Fn != "concat" {
			t.Errorf("Fn = %q, want %q", expr.Fn, "concat")
		}
		if len(expr.Args) != 2 {
			t.Fatalf("Args length = %d, want 2", len(expr.Args))
		}

		// Args will be raw maps because convertArg only converts goja.Values to FnExpr
		// This is expected behavior for direct convertMap calls with Go types
		arg0, ok := expr.Args[0].(map[string]any)
		if !ok {
			t.Fatalf("Args[0] should be map[string]any (not converted), got %T", expr.Args[0])
		}
		if arg0["fn"] != "upper" {
			t.Errorf("Args[0]['fn'] = %v, want 'upper'", arg0["fn"])
		}
	})

	t.Run("empty_map", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{}

		result := fb.convertMap(m)
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map[string]any, got %T", result)
		}

		if len(resultMap) != 0 {
			t.Errorf("Result should be empty map, got %v", resultMap)
		}
	})

	t.Run("map_with_mixed_types", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		m := map[string]any{
			"fn":   "add",
			"args": []any{42, 3.14, "text", true, nil},
		}

		result := fb.convertMap(m)
		expr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}

		if len(expr.Args) != 5 {
			t.Fatalf("Args length = %d, want 5", len(expr.Args))
		}

		// Verify all argument types are preserved
		// Note: Go maps preserve exact types, so int stays as int
		if expr.Args[0] != 42 {
			t.Errorf("Args[0] = %v (%T), want 42", expr.Args[0], expr.Args[0])
		}
		if expr.Args[1] != 3.14 {
			t.Errorf("Args[1] = %v, want 3.14", expr.Args[1])
		}
		if expr.Args[2] != "text" {
			t.Errorf("Args[2] = %v, want 'text'", expr.Args[2])
		}
		if expr.Args[3] != true {
			t.Errorf("Args[3] = %v, want true", expr.Args[3])
		}
		if expr.Args[4] != nil {
			t.Errorf("Args[4] = %v, want nil", expr.Args[4])
		}
	})
}

// -----------------------------------------------------------------------------
// convertArg Tests
// -----------------------------------------------------------------------------

func TestFnBuilder_convertArg(t *testing.T) {
	t.Run("fnexpr_pointer", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		expr := &FnExpr{Fn: "upper", Args: []any{"test"}}
		result := fb.convertArg(expr)

		// Should return the same pointer
		if result != expr {
			t.Error("convertArg should return FnExpr pointer as-is")
		}
	})

	t.Run("goja_value_with_fnexpr", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		// Create a FnExpr and wrap it in a Goja value
		expr := &FnExpr{Fn: "lower", Args: []any{"TEST"}}
		gojaVal := vm.ToValue(expr)

		result := fb.convertArg(gojaVal)

		// Should extract and return the FnExpr
		resultExpr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}
		if resultExpr.Fn != "lower" {
			t.Errorf("Fn = %q, want %q", resultExpr.Fn, "lower")
		}
	})

	t.Run("goja_value_with_map", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		// Create a map that represents a FnExpr
		m := map[string]any{
			"fn":   "concat",
			"args": []any{"hello", " ", "world"},
		}
		gojaVal := vm.ToValue(m)

		result := fb.convertArg(gojaVal)

		// Should convert map to FnExpr via convertMap
		resultExpr, ok := result.(*FnExpr)
		if !ok {
			t.Fatalf("Result should be *FnExpr, got %T", result)
		}
		if resultExpr.Fn != "concat" {
			t.Errorf("Fn = %q, want %q", resultExpr.Fn, "concat")
		}
	})

	t.Run("goja_value_primitive", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		// Test with a primitive value
		gojaVal := vm.ToValue("hello")
		result := fb.convertArg(gojaVal)

		if result != "hello" {
			t.Errorf("Result = %v, want %q", result, "hello")
		}
	})

	t.Run("non_goja_value", func(t *testing.T) {
		vm := goja.New()
		fb := NewFnBuilder(vm)

		// Regular Go value
		result := fb.convertArg(42)
		if result != 42 {
			t.Errorf("Result = %v, want 42", result)
		}

		result = fb.convertArg("test")
		if result != "test" {
			t.Errorf("Result = %v, want %q", result, "test")
		}
	})
}
