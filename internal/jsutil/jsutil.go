// Package jsutil provides safe, consistent JS<->Go value conversion for Goja runtime.
package jsutil

import (
	"strconv"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// GetString safely retrieves a string property from a Goja object.
// Returns the value and true if the key exists and is a string, otherwise returns "" and false.
func GetString(obj *goja.Object, key string) (string, bool) {
	if obj == nil {
		return "", false
	}
	v := obj.Get(key)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "", false
	}
	s, ok := v.Export().(string)
	return s, ok
}

// GetInt safely retrieves an integer property from a Goja object.
// Returns the value and true if the key exists and is a number, otherwise returns 0 and false.
func GetInt(obj *goja.Object, key string) (int, bool) {
	if obj == nil {
		return 0, false
	}
	v := obj.Get(key)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return 0, false
	}
	return toInt(v.Export())
}

// GetBool safely retrieves a boolean property from a Goja object.
// Returns the value and true if the key exists and is a boolean, otherwise returns false and false.
func GetBool(obj *goja.Object, key string) (bool, bool) {
	if obj == nil {
		return false, false
	}
	v := obj.Get(key)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return false, false
	}
	b, ok := v.Export().(bool)
	return b, ok
}

// GetObject safely retrieves an object property from a Goja object.
// Returns the object and true if the key exists and is an object, otherwise returns nil and false.
func GetObject(obj *goja.Object, key string) (*goja.Object, bool) {
	if obj == nil {
		return nil, false
	}
	v := obj.Get(key)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, false
	}
	o, ok := v.(*goja.Object)
	return o, ok
}

// GetArray safely retrieves an array property from a Goja object.
// Returns the array values and true if the key exists and is an array, otherwise returns nil and false.
func GetArray(obj *goja.Object, key string) ([]goja.Value, bool) {
	if obj == nil {
		return nil, false
	}
	v := obj.Get(key)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, false
	}
	o, ok := v.(*goja.Object)
	if !ok {
		return nil, false
	}
	// Check if it's an array by looking for a length property
	lengthVal := o.Get("length")
	if lengthVal == nil || goja.IsUndefined(lengthVal) {
		return nil, false
	}
	length, ok := toInt(lengthVal.Export())
	if !ok || length < 0 {
		return nil, false
	}
	result := make([]goja.Value, 0, length)
	for i := 0; i < length; i++ {
		result = append(result, o.Get(strconv.Itoa(i)))
	}
	return result, true
}

// GetStringArray safely retrieves a string array property from a Goja object.
// Returns the string slice and true if successful, otherwise returns nil and false.
func GetStringArray(obj *goja.Object, key string) ([]string, bool) {
	arr, ok := GetArray(obj, key)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			continue
		}
		if s, ok := v.Export().(string); ok {
			result = append(result, s)
		} else {
			return nil, false
		}
	}
	return result, true
}

// Call safely calls a Goja function with error handling.
// Returns the result value and any error that occurred.
func Call(fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
	if fn == nil {
		return nil, alerr.New(alerr.ErrJSExecution, "cannot call nil function")
	}
	result, err := fn(this, args...)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrJSExecution, err, "function call failed")
	}
	return result, nil
}

// MustCall calls a Goja function and panics on error.
// Use with caution - only when you're certain the call will succeed.
func MustCall(fn goja.Callable, this goja.Value, args ...goja.Value) goja.Value {
	result, err := Call(fn, this, args...)
	if err != nil {
		panic(err)
	}
	return result
}

// ToGoValue converts a Goja value to a Go value.
// Returns nil for undefined/null values.
func ToGoValue(v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	return v.Export()
}

// ToGoString converts a Goja value to a Go string.
// Returns an empty string for undefined/null values or non-string types.
func ToGoString(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	if s, ok := v.Export().(string); ok {
		return s
	}
	// Fallback: convert to string representation
	return v.String()
}

// ToGoInt converts a Goja value to a Go int.
// Returns 0 for undefined/null values or non-numeric types.
func ToGoInt(v goja.Value) int {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return 0
	}
	n, _ := toInt(v.Export())
	return n
}

// ToGoBool converts a Goja value to a Go bool.
// Returns false for undefined/null values or non-boolean types.
func ToGoBool(v goja.Value) bool {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return false
	}
	if b, ok := v.Export().(bool); ok {
		return b
	}
	return false
}

// ToGoStringSlice converts a Goja array value to a Go string slice.
// Returns nil for undefined/null values or non-array types.
func ToGoStringSlice(v goja.Value) []string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	o, ok := v.(*goja.Object)
	if !ok {
		return nil
	}
	// Check for array-like object with length property
	lengthVal := o.Get("length")
	if lengthVal == nil || goja.IsUndefined(lengthVal) {
		return nil
	}
	length, ok := toInt(lengthVal.Export())
	if !ok || length < 0 {
		return nil
	}
	result := make([]string, 0, length)
	for i := 0; i < length; i++ {
		elem := o.Get(strconv.Itoa(i))
		if elem == nil || goja.IsUndefined(elem) || goja.IsNull(elem) {
			result = append(result, "")
		} else if s, ok := elem.Export().(string); ok {
			result = append(result, s)
		} else {
			result = append(result, elem.String())
		}
	}
	return result
}

// ToGoMap converts a Goja object value to a Go map[string]any.
// Returns nil for undefined/null values or non-object types.
func ToGoMap(v goja.Value) map[string]any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	o, ok := v.(*goja.Object)
	if !ok {
		return nil
	}
	exported := o.Export()
	if m, ok := exported.(map[string]any); ok {
		return m
	}
	// Manual conversion for nested objects
	result := make(map[string]any)
	for _, key := range o.Keys() {
		val := o.Get(key)
		result[key] = ToGoValue(val)
	}
	return result
}

// WrapJSError wraps a JavaScript error with the specified error code.
// Returns nil if the input error is nil.
func WrapJSError(err error, code alerr.Code) *alerr.Error {
	if err == nil {
		return nil
	}
	// Check if it's a Goja exception for better error messages
	if exception, ok := err.(*goja.Exception); ok {
		return alerr.Wrap(code, err, exception.String())
	}
	return alerr.Wrap(code, err, err.Error())
}

// toInt converts various numeric types to int.
// This helper handles the fact that Goja may return different numeric types.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float32:
		return int(n), true
	case float64:
		// JS numbers are float64 internally, but we only accept safe integer range
		if n >= -2147483648 && n <= 2147483647 && n == float64(int(n)) {
			return int(n), true
		}
		return 0, false
	default:
		return 0, false
	}
}
