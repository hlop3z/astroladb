package runtime

import (
	"testing"

	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// toBaseObject Tests (TableBuilder)
// -----------------------------------------------------------------------------

func TestTableBuilder_toBaseObject(t *testing.T) {
	t.Run("has_all_base_type_methods", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		// Check that all expected methods exist
		methods := []string{
			"id", "string", "text", "integer", "float", "decimal",
			"boolean", "date", "time", "datetime", "uuid", "json",
			"base64", "enum", "timestamps", "soft_delete", "sortable",
			"belongs_to",
		}

		for _, method := range methods {
			val := obj.Get(method)
			if val == nil || goja.IsUndefined(val) {
				t.Errorf("Expected method %s to exist", method)
			}
		}
	})

	t.Run("id_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		// Call id() method
		idFn, ok := goja.AssertFunction(obj.Get("id"))
		if !ok {
			t.Fatal("id should be a function")
		}

		result, err := idFn(goja.Undefined())
		if err != nil {
			t.Fatalf("id() call failed: %v", err)
		}

		// Should return a chainable object
		if result == nil || goja.IsUndefined(result) {
			t.Error("id() should return a chainable object")
		}

		// Check that column was added
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column, got %d", len(tb.columns))
		}
		if tb.columns[0].Name != "id" {
			t.Errorf("Column name = %s, want 'id'", tb.columns[0].Name)
		}
		if tb.columns[0].Type != "uuid" {
			t.Errorf("Column type = %s, want 'uuid'", tb.columns[0].Type)
		}
		if !tb.columns[0].PrimaryKey {
			t.Error("id column should be primary key")
		}
	})

	t.Run("string_method_with_valid_length", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		stringFn, ok := goja.AssertFunction(obj.Get("string"))
		if !ok {
			t.Fatal("string should be a function")
		}

		result, err := stringFn(goja.Undefined(), vm.ToValue("email"), vm.ToValue(255))
		if err != nil {
			t.Fatalf("string() call failed: %v", err)
		}

		if result == nil || goja.IsUndefined(result) {
			t.Error("string() should return a chainable object")
		}

		// Check that column was added
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column, got %d", len(tb.columns))
		}
		if tb.columns[0].Name != "email" {
			t.Errorf("Column name = %s, want 'email'", tb.columns[0].Name)
		}
		if tb.columns[0].Type != "string" {
			t.Errorf("Column type = %s, want 'string'", tb.columns[0].Type)
		}
		if tb.columns[0].TypeArgs == nil || len(tb.columns[0].TypeArgs) == 0 {
			t.Error("string column should have length in TypeArgs")
		}
	})

	t.Run("string_method_with_zero_length_panics", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		stringFn, ok := goja.AssertFunction(obj.Get("string"))
		if !ok {
			t.Fatal("string should be a function")
		}

		// Should return error (Goja panic becomes error)
		_, err := stringFn(goja.Undefined(), vm.ToValue("name"), vm.ToValue(0))
		if err == nil {
			t.Error("Expected error when length is 0")
		}
	})

	t.Run("string_method_with_negative_length_panics", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		stringFn, ok := goja.AssertFunction(obj.Get("string"))
		if !ok {
			t.Fatal("string should be a function")
		}

		// Should return error (Goja panic becomes error)
		_, err := stringFn(goja.Undefined(), vm.ToValue("name"), vm.ToValue(-1))
		if err == nil {
			t.Error("Expected error when length is negative")
		}
	})

	t.Run("text_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		textFn, ok := goja.AssertFunction(obj.Get("text"))
		if !ok {
			t.Fatal("text should be a function")
		}

		result, err := textFn(goja.Undefined(), vm.ToValue("content"))
		if err != nil {
			t.Fatalf("text() call failed: %v", err)
		}

		if result == nil || goja.IsUndefined(result) {
			t.Error("text() should return a chainable object")
		}

		// Check column
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column, got %d", len(tb.columns))
		}
		if tb.columns[0].Name != "content" || tb.columns[0].Type != "text" {
			t.Errorf("Column = {name: %s, type: %s}, want {name: 'content', type: 'text'}", tb.columns[0].Name, tb.columns[0].Type)
		}
	})

	t.Run("integer_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		integerFn, ok := goja.AssertFunction(obj.Get("integer"))
		if !ok {
			t.Fatal("integer should be a function")
		}

		result, err := integerFn(goja.Undefined(), vm.ToValue("quantity"))
		if err != nil {
			t.Fatalf("integer() call failed: %v", err)
		}

		if result == nil || goja.IsUndefined(result) {
			t.Error("integer() should return a chainable object")
		}

		if len(tb.columns) != 1 || tb.columns[0].Type != "integer" {
			t.Error("integer() should add an integer column")
		}
	})

	t.Run("decimal_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		decimalFn, ok := goja.AssertFunction(obj.Get("decimal"))
		if !ok {
			t.Fatal("decimal should be a function")
		}

		result, err := decimalFn(goja.Undefined(), vm.ToValue("price"), vm.ToValue(10), vm.ToValue(2))
		if err != nil {
			t.Fatalf("decimal() call failed: %v", err)
		}

		if result == nil || goja.IsUndefined(result) {
			t.Error("decimal() should return a chainable object")
		}

		// Check column
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column, got %d", len(tb.columns))
		}
		if tb.columns[0].Type != "decimal" {
			t.Errorf("Column type = %s, want 'decimal'", tb.columns[0].Type)
		}
		if tb.columns[0].TypeArgs == nil || len(tb.columns[0].TypeArgs) != 2 {
			t.Error("decimal column should have precision and scale in TypeArgs")
		}
	})

	t.Run("boolean_float_date_time_datetime_uuid_json_base64", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		types := []struct {
			method  string
			colType string
		}{
			{"boolean", "boolean"},
			{"float", "float"},
			{"date", "date"},
			{"time", "time"},
			{"datetime", "datetime"},
			{"uuid", "uuid"},
			{"json", "json"},
			{"base64", "base64"},
		}

		for _, tt := range types {
			tb.columns = nil // Reset columns

			fn, ok := goja.AssertFunction(obj.Get(tt.method))
			if !ok {
				t.Errorf("%s should be a function", tt.method)
				continue
			}

			result, err := fn(goja.Undefined(), vm.ToValue("test_col"))
			if err != nil {
				t.Errorf("%s() call failed: %v", tt.method, err)
				continue
			}

			if result == nil || goja.IsUndefined(result) {
				t.Errorf("%s() should return a chainable object", tt.method)
			}

			if len(tb.columns) != 1 || tb.columns[0].Type != tt.colType {
				t.Errorf("%s() should add a %s column, got type: %s", tt.method, tt.colType, tb.columns[0].Type)
			}
		}
	})

	t.Run("enum_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		enumFn, ok := goja.AssertFunction(obj.Get("enum"))
		if !ok {
			t.Fatal("enum should be a function")
		}

		values := []string{"active", "inactive", "suspended"}
		result, err := enumFn(goja.Undefined(), vm.ToValue("status"), vm.ToValue(values))
		if err != nil {
			t.Fatalf("enum() call failed: %v", err)
		}

		if result == nil || goja.IsUndefined(result) {
			t.Error("enum() should return a chainable object")
		}

		// Check column
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column, got %d", len(tb.columns))
		}
		if tb.columns[0].Type != "enum" {
			t.Errorf("Column type = %s, want 'enum'", tb.columns[0].Type)
		}
		if tb.columns[0].TypeArgs == nil || len(tb.columns[0].TypeArgs) == 0 {
			t.Error("enum column should have values in TypeArgs")
		}
	})

	t.Run("timestamps_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		timestampsFn, ok := goja.AssertFunction(obj.Get("timestamps"))
		if !ok {
			t.Fatal("timestamps should be a function")
		}

		_, err := timestampsFn(goja.Undefined())
		if err != nil {
			t.Fatalf("timestamps() call failed: %v", err)
		}

		// Should add created_at and updated_at
		if len(tb.columns) != 2 {
			t.Fatalf("Expected 2 columns (created_at, updated_at), got %d", len(tb.columns))
		}

		if tb.columns[0].Name != "created_at" {
			t.Errorf("First column name = %s, want 'created_at'", tb.columns[0].Name)
		}
		if tb.columns[1].Name != "updated_at" {
			t.Errorf("Second column name = %s, want 'updated_at'", tb.columns[1].Name)
		}

		// Both should be datetime with NOW() default
		for i, col := range tb.columns {
			if col.Type != "datetime" {
				t.Errorf("Column %d type = %s, want 'datetime'", i, col.Type)
			}
			if col.Default == nil {
				t.Errorf("Column %d should have NOW() default", i)
			}
		}
	})

	t.Run("soft_delete_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		softDeleteFn, ok := goja.AssertFunction(obj.Get("soft_delete"))
		if !ok {
			t.Fatal("soft_delete should be a function")
		}

		_, err := softDeleteFn(goja.Undefined())
		if err != nil {
			t.Fatalf("soft_delete() call failed: %v", err)
		}

		// Should add deleted_at column
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column (deleted_at), got %d", len(tb.columns))
		}

		col := tb.columns[0]
		if col.Name != "deleted_at" {
			t.Errorf("Column name = %s, want 'deleted_at'", col.Name)
		}
		if col.Type != "datetime" {
			t.Errorf("Column type = %s, want 'datetime'", col.Type)
		}
		if !col.Nullable {
			t.Error("deleted_at should be nullable")
		}
	})

	t.Run("sortable_method", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		sortableFn, ok := goja.AssertFunction(obj.Get("sortable"))
		if !ok {
			t.Fatal("sortable should be a function")
		}

		_, err := sortableFn(goja.Undefined())
		if err != nil {
			t.Fatalf("sortable() call failed: %v", err)
		}

		// Should add position column
		if len(tb.columns) != 1 {
			t.Fatalf("Expected 1 column (position), got %d", len(tb.columns))
		}

		col := tb.columns[0]
		if col.Name != "position" {
			t.Errorf("Column name = %s, want 'position'", col.Name)
		}
		if col.Type != "integer" {
			t.Errorf("Column type = %s, want 'integer'", col.Type)
		}
		if col.Default != 0 {
			t.Errorf("Column default = %v, want 0", col.Default)
		}
	})

	t.Run("belongs_to_method_valid_ref", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		belongsToFn, ok := goja.AssertFunction(obj.Get("belongs_to"))
		if !ok {
			t.Fatal("belongs_to should be a function")
		}

		result, err := belongsToFn(goja.Undefined(), vm.ToValue("auth.user"))
		if err != nil {
			t.Fatalf("belongs_to() call failed: %v", err)
		}

		// Should return a chainable object
		if result == nil || goja.IsUndefined(result) {
			t.Error("belongs_to() should return a chainable object")
		}
	})

	t.Run("belongs_to_method_missing_namespace_panics", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		belongsToFn, ok := goja.AssertFunction(obj.Get("belongs_to"))
		if !ok {
			t.Fatal("belongs_to should be a function")
		}

		// Should return error when namespace is missing (Goja panic becomes error)
		_, err := belongsToFn(goja.Undefined(), vm.ToValue("user"))
		if err == nil {
			t.Error("Expected error when reference missing namespace")
		}
	})

	t.Run("multiple_methods_called_sequentially", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)
		obj := tb.toBaseObject()

		// Call multiple methods
		idFn, _ := goja.AssertFunction(obj.Get("id"))
		_, _ = idFn(goja.Undefined())

		stringFn, _ := goja.AssertFunction(obj.Get("string"))
		_, _ = stringFn(goja.Undefined(), vm.ToValue("name"), vm.ToValue(100))

		integerFn, _ := goja.AssertFunction(obj.Get("integer"))
		_, _ = integerFn(goja.Undefined(), vm.ToValue("quantity"))

		timestampsFn, _ := goja.AssertFunction(obj.Get("timestamps"))
		_, _ = timestampsFn(goja.Undefined())

		// Should have: id, name, quantity, created_at, updated_at
		if len(tb.columns) != 5 {
			t.Errorf("Expected 5 columns, got %d", len(tb.columns))
		}
	})
}
