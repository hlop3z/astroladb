package engine

import (
	"testing"
)

// -----------------------------------------------------------------------------
// ToMap Tests
// -----------------------------------------------------------------------------

func TestToMap(t *testing.T) {
	t.Run("string_slice", func(t *testing.T) {
		items := []string{"apple", "banana", "cherry"}
		result := ToMap(items, func(s string) string { return s[:1] })

		if len(result) != 3 {
			t.Errorf("ToMap() = %d items, want 3", len(result))
		}
		if result["a"] != "apple" {
			t.Errorf("result['a'] = %q, want 'apple'", result["a"])
		}
		if result["b"] != "banana" {
			t.Errorf("result['b'] = %q, want 'banana'", result["b"])
		}
		if result["c"] != "cherry" {
			t.Errorf("result['c'] = %q, want 'cherry'", result["c"])
		}
	})

	t.Run("struct_slice", func(t *testing.T) {
		type User struct {
			ID   int
			Name string
		}
		users := []User{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		}
		result := ToMap(users, func(u User) int { return u.ID })

		if len(result) != 2 {
			t.Errorf("ToMap() = %d items, want 2", len(result))
		}
		if result[1].Name != "Alice" {
			t.Errorf("result[1].Name = %q, want 'Alice'", result[1].Name)
		}
		if result[2].Name != "Bob" {
			t.Errorf("result[2].Name = %q, want 'Bob'", result[2].Name)
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		var items []string
		result := ToMap(items, func(s string) string { return s })

		if len(result) != 0 {
			t.Errorf("ToMap() = %d items, want 0", len(result))
		}
	})

	t.Run("duplicate_keys", func(t *testing.T) {
		// Last item wins when there are duplicate keys
		items := []string{"apple", "apricot", "avocado"}
		result := ToMap(items, func(s string) string { return s[:1] })

		// All start with 'a', so only last one survives
		if result["a"] != "avocado" {
			t.Errorf("result['a'] = %q, want 'avocado' (last one)", result["a"])
		}
	})
}

// -----------------------------------------------------------------------------
// ToSet Tests
// -----------------------------------------------------------------------------

func TestToSet(t *testing.T) {
	t.Run("string_slice", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		result := ToSet(items)

		if len(result) != 3 {
			t.Errorf("ToSet() = %d items, want 3", len(result))
		}
		if _, exists := result["a"]; !exists {
			t.Error("result should contain 'a'")
		}
		if _, exists := result["b"]; !exists {
			t.Error("result should contain 'b'")
		}
		if _, exists := result["c"]; !exists {
			t.Error("result should contain 'c'")
		}
	})

	t.Run("int_slice", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		result := ToSet(items)

		if len(result) != 5 {
			t.Errorf("ToSet() = %d items, want 5", len(result))
		}
		for _, item := range items {
			if _, exists := result[item]; !exists {
				t.Errorf("result should contain %d", item)
			}
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		var items []string
		result := ToSet(items)

		if len(result) != 0 {
			t.Errorf("ToSet() = %d items, want 0", len(result))
		}
	})

	t.Run("duplicates_deduplicated", func(t *testing.T) {
		items := []string{"a", "b", "a", "c", "b", "a"}
		result := ToSet(items)

		// Should only have unique items
		if len(result) != 3 {
			t.Errorf("ToSet() = %d items, want 3 (deduplicated)", len(result))
		}
	})

	t.Run("non_existent_item", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		result := ToSet(items)

		if _, exists := result["z"]; exists {
			t.Error("result should not contain 'z'")
		}
	})
}
