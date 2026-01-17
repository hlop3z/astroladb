package jsutil

import (
	"testing"

	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// newVM creates a new Goja runtime for testing.
func newVM() *goja.Runtime {
	return goja.New()
}

// createObject creates a Goja object from a Go map.
func createObject(vm *goja.Runtime, data map[string]interface{}) *goja.Object {
	obj := vm.NewObject()
	for k, v := range data {
		_ = obj.Set(k, v)
	}
	return obj
}

// -----------------------------------------------------------------------------
// GetString Tests
// -----------------------------------------------------------------------------

func TestGetString(t *testing.T) {
	vm := newVM()

	tests := []struct {
		name    string
		obj     *goja.Object
		key     string
		wantVal string
		wantOK  bool
	}{
		{
			name:    "existing_string",
			obj:     createObject(vm, map[string]interface{}{"name": "test"}),
			key:     "name",
			wantVal: "test",
			wantOK:  true,
		},
		{
			name:    "missing_key",
			obj:     createObject(vm, map[string]interface{}{"other": "value"}),
			key:     "name",
			wantVal: "",
			wantOK:  false,
		},
		{
			name:    "nil_object",
			obj:     nil,
			key:     "name",
			wantVal: "",
			wantOK:  false,
		},
		{
			name:    "empty_string",
			obj:     createObject(vm, map[string]interface{}{"name": ""}),
			key:     "name",
			wantVal: "",
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := GetString(tt.obj, tt.key)
			if val != tt.wantVal {
				t.Errorf("GetString() value = %q, want %q", val, tt.wantVal)
			}
			if ok != tt.wantOK {
				t.Errorf("GetString() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// GetInt Tests
// -----------------------------------------------------------------------------

func TestGetInt(t *testing.T) {
	t.Run("nil_object", func(t *testing.T) {
		val, ok := GetInt(nil, "count")
		if ok {
			t.Error("GetInt() ok = true, want false")
		}
		if val != 0 {
			t.Errorf("GetInt() = %d, want 0", val)
		}
	})

	t.Run("missing_key", func(t *testing.T) {
		vm := newVM()
		obj := createObject(vm, map[string]interface{}{"other": 1})
		val, ok := GetInt(obj, "count")
		if ok {
			t.Error("GetInt() ok = true, want false")
		}
		if val != 0 {
			t.Errorf("GetInt() = %d, want 0", val)
		}
	})
}

// -----------------------------------------------------------------------------
// GetBool Tests
// -----------------------------------------------------------------------------

func TestGetBool(t *testing.T) {
	vm := newVM()

	tests := []struct {
		name    string
		obj     *goja.Object
		key     string
		wantVal bool
		wantOK  bool
	}{
		{
			name:    "true_value",
			obj:     createObject(vm, map[string]interface{}{"active": true}),
			key:     "active",
			wantVal: true,
			wantOK:  true,
		},
		{
			name:    "false_value",
			obj:     createObject(vm, map[string]interface{}{"active": false}),
			key:     "active",
			wantVal: false,
			wantOK:  true,
		},
		{
			name:    "missing_key",
			obj:     createObject(vm, map[string]interface{}{"other": true}),
			key:     "active",
			wantVal: false,
			wantOK:  false,
		},
		{
			name:    "nil_object",
			obj:     nil,
			key:     "active",
			wantVal: false,
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := GetBool(tt.obj, tt.key)
			if val != tt.wantVal {
				t.Errorf("GetBool() value = %v, want %v", val, tt.wantVal)
			}
			if ok != tt.wantOK {
				t.Errorf("GetBool() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// GetObject Tests
// -----------------------------------------------------------------------------

func TestGetObject(t *testing.T) {
	vm := newVM()

	t.Run("existing_object", func(t *testing.T) {
		inner := createObject(vm, map[string]interface{}{"key": "value"})
		outer := createObject(vm, map[string]interface{}{"nested": inner})

		obj, ok := GetObject(outer, "nested")
		if !ok {
			t.Error("GetObject() ok = false, want true")
		}
		if obj == nil {
			t.Error("GetObject() returned nil object")
		}
	})

	t.Run("missing_key", func(t *testing.T) {
		obj := createObject(vm, map[string]interface{}{"other": "value"})

		result, ok := GetObject(obj, "nested")
		if ok {
			t.Error("GetObject() ok = true, want false")
		}
		if result != nil {
			t.Error("GetObject() should return nil for missing key")
		}
	})

	t.Run("nil_object", func(t *testing.T) {
		result, ok := GetObject(nil, "key")
		if ok {
			t.Error("GetObject() ok = true, want false for nil input")
		}
		if result != nil {
			t.Error("GetObject() should return nil for nil input")
		}
	})
}

// -----------------------------------------------------------------------------
// GetArray Tests
// -----------------------------------------------------------------------------

func TestGetArray(t *testing.T) {
	t.Run("missing_key", func(t *testing.T) {
		vm := newVM()
		obj := createObject(vm, map[string]interface{}{"other": "value"})

		result, ok := GetArray(obj, "items")
		if ok {
			t.Error("GetArray() ok = true, want false")
		}
		if result != nil {
			t.Error("GetArray() should return nil for missing key")
		}
	})

	t.Run("nil_object", func(t *testing.T) {
		result, ok := GetArray(nil, "items")
		if ok {
			t.Error("GetArray() ok = true, want false")
		}
		if result != nil {
			t.Error("GetArray() should return nil for nil input")
		}
	})
}

// -----------------------------------------------------------------------------
// GetStringArray Tests
// -----------------------------------------------------------------------------

func TestGetStringArray(t *testing.T) {
	t.Run("nil_object", func(t *testing.T) {
		result, ok := GetStringArray(nil, "items")
		if ok {
			t.Error("GetStringArray() ok = true, want false")
		}
		if result != nil {
			t.Error("GetStringArray() should return nil for nil input")
		}
	})
}

// -----------------------------------------------------------------------------
// Call Tests
// -----------------------------------------------------------------------------

func TestCall(t *testing.T) {
	t.Run("nil_function", func(t *testing.T) {
		_, err := Call(nil, goja.Undefined())
		if err == nil {
			t.Error("Call() expected error for nil function")
		}
	})
}

// -----------------------------------------------------------------------------
// MustCall Tests
// -----------------------------------------------------------------------------

func TestMustCall(t *testing.T) {
	t.Run("panics_on_nil_function", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustCall() expected panic for nil function")
			}
		}()

		MustCall(nil, goja.Undefined())
	})
}

// -----------------------------------------------------------------------------
// ToGoValue Tests
// -----------------------------------------------------------------------------

func TestToGoValue(t *testing.T) {
	vm := newVM()

	t.Run("string_value", func(t *testing.T) {
		v := vm.ToValue("test")
		result := ToGoValue(v)
		if result != "test" {
			t.Errorf("ToGoValue() = %v, want 'test'", result)
		}
	})

	t.Run("nil_value", func(t *testing.T) {
		result := ToGoValue(nil)
		if result != nil {
			t.Errorf("ToGoValue() = %v, want nil", result)
		}
	})

	t.Run("undefined_value", func(t *testing.T) {
		result := ToGoValue(goja.Undefined())
		if result != nil {
			t.Errorf("ToGoValue() = %v, want nil for undefined", result)
		}
	})

	t.Run("null_value", func(t *testing.T) {
		result := ToGoValue(goja.Null())
		if result != nil {
			t.Errorf("ToGoValue() = %v, want nil for null", result)
		}
	})
}

// -----------------------------------------------------------------------------
// ToGoString Tests
// -----------------------------------------------------------------------------

func TestToGoString(t *testing.T) {
	vm := newVM()

	tests := []struct {
		name string
		val  goja.Value
		want string
	}{
		{"string_value", vm.ToValue("hello"), "hello"},
		{"nil_value", nil, ""},
		{"undefined", goja.Undefined(), ""},
		{"null", goja.Null(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToGoString(tt.val)
			if result != tt.want {
				t.Errorf("ToGoString() = %q, want %q", result, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ToGoInt Tests
// -----------------------------------------------------------------------------

func TestToGoInt(t *testing.T) {
	tests := []struct {
		name string
		val  goja.Value
		want int
	}{
		{"nil_value", nil, 0},
		{"undefined", goja.Undefined(), 0},
		{"null", goja.Null(), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToGoInt(tt.val)
			if result != tt.want {
				t.Errorf("ToGoInt() = %d, want %d", result, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ToGoBool Tests
// -----------------------------------------------------------------------------

func TestToGoBool(t *testing.T) {
	vm := newVM()

	tests := []struct {
		name string
		val  goja.Value
		want bool
	}{
		{"true_value", vm.ToValue(true), true},
		{"false_value", vm.ToValue(false), false},
		{"nil_value", nil, false},
		{"undefined", goja.Undefined(), false},
		{"null", goja.Null(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToGoBool(tt.val)
			if result != tt.want {
				t.Errorf("ToGoBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ToGoStringSlice Tests
// -----------------------------------------------------------------------------

func TestToGoStringSlice(t *testing.T) {
	t.Run("nil_value", func(t *testing.T) {
		result := ToGoStringSlice(nil)
		if result != nil {
			t.Errorf("ToGoStringSlice() = %v, want nil", result)
		}
	})

	t.Run("undefined", func(t *testing.T) {
		result := ToGoStringSlice(goja.Undefined())
		if result != nil {
			t.Errorf("ToGoStringSlice() = %v, want nil for undefined", result)
		}
	})
}

// -----------------------------------------------------------------------------
// ToGoMap Tests
// -----------------------------------------------------------------------------

func TestToGoMap(t *testing.T) {
	vm := newVM()

	t.Run("object_to_map", func(t *testing.T) {
		obj := createObject(vm, map[string]interface{}{
			"name": "test",
		})

		result := ToGoMap(obj)
		if result == nil {
			t.Fatal("ToGoMap() = nil")
		}
		if result["name"] != "test" {
			t.Errorf("ToGoMap()['name'] = %v, want 'test'", result["name"])
		}
	})

	t.Run("nil_value", func(t *testing.T) {
		result := ToGoMap(nil)
		if result != nil {
			t.Errorf("ToGoMap() = %v, want nil", result)
		}
	})

	t.Run("undefined", func(t *testing.T) {
		result := ToGoMap(goja.Undefined())
		if result != nil {
			t.Errorf("ToGoMap() = %v, want nil for undefined", result)
		}
	})

	t.Run("empty_object", func(t *testing.T) {
		obj := vm.NewObject()
		result := ToGoMap(obj)
		if result == nil {
			t.Error("ToGoMap() = nil, want empty map")
		}
		if len(result) != 0 {
			t.Errorf("ToGoMap() = %d entries, want 0", len(result))
		}
	})
}

// -----------------------------------------------------------------------------
// toInt Tests
// -----------------------------------------------------------------------------

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantVal int
		wantOK  bool
	}{
		{"int", 42, 42, true},
		{"float32", float32(42.0), 42, true},
		{"float64", float64(42.0), 42, true},
		{"float64_with_decimal", float64(42.5), 0, false},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
		{"negative_int", -10, -10, true},
		{"zero", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := toInt(tt.input)
			if val != tt.wantVal {
				t.Errorf("toInt() value = %d, want %d", val, tt.wantVal)
			}
			if ok != tt.wantOK {
				t.Errorf("toInt() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}
