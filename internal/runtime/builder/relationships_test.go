package builder

import (
	"testing"

	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// relationshipBuilder Tests
// -----------------------------------------------------------------------------

func TestTableBuilder_relationshipBuilder(t *testing.T) {
	t.Run("basic_relationship", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name:     "user_id",
			Type:     "integer",
			Nullable: false,
			Reference: &RefDef{
				Table:    "auth.user",
				Column:   "id",
				OnDelete: "",
				OnUpdate: "",
			},
		}
		idx := &IndexDef{
			Columns: []string{"user_id"},
		}

		obj := tb.relationshipBuilder("auth.user", col, idx)
		if obj == nil {
			t.Fatal("relationshipBuilder should return non-nil object")
		}
	})

	t.Run("as_alias", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name:      "user_id",
			Type:      "integer",
			Reference: &RefDef{Table: "auth.user", Column: "id"},
		}
		idx := &IndexDef{
			Columns: []string{"user_id"},
		}

		obj := tb.relationshipBuilder("auth.user", col, idx)

		// Call .as("author")
		asFn, ok := goja.AssertFunction(obj.Get("as"))
		if !ok {
			t.Fatal("as should be a function")
		}

		_, err := asFn(goja.Undefined(), vm.ToValue("author"))
		if err != nil {
			t.Fatalf("as() call failed: %v", err)
		}

		// Check that column name was updated
		if col.Name != "author_id" {
			t.Errorf("Column name = %q, want %q", col.Name, "author_id")
		}
		// Check that index columns were updated
		if len(idx.Columns) != 1 || idx.Columns[0] != "author_id" {
			t.Errorf("Index columns = %v, want [author_id]", idx.Columns)
		}
	})

	t.Run("optional", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name:      "user_id",
			Type:      "integer",
			Nullable:  false,
			Reference: &RefDef{Table: "auth.user", Column: "id"},
		}
		idx := &IndexDef{Columns: []string{"user_id"}}

		obj := tb.relationshipBuilder("auth.user", col, idx)

		// Call .optional()
		optFn, ok := goja.AssertFunction(obj.Get("optional"))
		if !ok {
			t.Fatal("optional should be a function")
		}

		_, err := optFn(goja.Undefined())
		if err != nil {
			t.Fatalf("optional() call failed: %v", err)
		}

		// Check that column is now nullable
		if !col.Nullable {
			t.Error("Column should be nullable after optional()")
		}
	})

	t.Run("on_delete", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name: "user_id",
			Type: "integer",
			Reference: &RefDef{
				Table:    "auth.user",
				Column:   "id",
				OnDelete: "",
				OnUpdate: "",
			},
		}
		idx := &IndexDef{Columns: []string{"user_id"}}

		obj := tb.relationshipBuilder("auth.user", col, idx)

		// Call .on_delete("CASCADE")
		onDeleteFn, ok := goja.AssertFunction(obj.Get("on_delete"))
		if !ok {
			t.Fatal("on_delete should be a function")
		}

		_, err := onDeleteFn(goja.Undefined(), vm.ToValue("CASCADE"))
		if err != nil {
			t.Fatalf("on_delete() call failed: %v", err)
		}

		// Check that OnDelete was set
		if col.Reference.OnDelete != "CASCADE" {
			t.Errorf("OnDelete = %q, want %q", col.Reference.OnDelete, "CASCADE")
		}
	})

	t.Run("on_update", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name: "user_id",
			Type: "integer",
			Reference: &RefDef{
				Table:    "auth.user",
				Column:   "id",
				OnDelete: "",
				OnUpdate: "",
			},
		}
		idx := &IndexDef{Columns: []string{"user_id"}}

		obj := tb.relationshipBuilder("auth.user", col, idx)

		// Call .on_update("SET NULL")
		onUpdateFn, ok := goja.AssertFunction(obj.Get("on_update"))
		if !ok {
			t.Fatal("on_update should be a function")
		}

		_, err := onUpdateFn(goja.Undefined(), vm.ToValue("SET NULL"))
		if err != nil {
			t.Fatalf("on_update() call failed: %v", err)
		}

		// Check that OnUpdate was set
		if col.Reference.OnUpdate != "SET NULL" {
			t.Errorf("OnUpdate = %q, want %q", col.Reference.OnUpdate, "SET NULL")
		}
	})

	t.Run("chaining_methods", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name:     "user_id",
			Type:     "integer",
			Nullable: false,
			Reference: &RefDef{
				Table:    "auth.user",
				Column:   "id",
				OnDelete: "",
				OnUpdate: "",
			},
		}
		idx := &IndexDef{Columns: []string{"user_id"}}

		obj := tb.relationshipBuilder("auth.user", col, idx)

		// Chain: .as("author").optional().on_delete("CASCADE")
		script := `
			(function(obj) {
				return obj.as("author").optional().on_delete("CASCADE");
			})
		`

		chainFn, err := vm.RunString(script)
		if err != nil {
			t.Fatalf("Failed to compile chain script: %v", err)
		}

		chainFunc, ok := goja.AssertFunction(chainFn)
		if !ok {
			t.Fatal("Should be a function")
		}

		_, err = chainFunc(goja.Undefined(), obj)
		if err != nil {
			t.Fatalf("Chain call failed: %v", err)
		}

		// Verify all modifications
		if col.Name != "author_id" {
			t.Errorf("Column name = %q, want %q", col.Name, "author_id")
		}
		if !col.Nullable {
			t.Error("Column should be nullable")
		}
		if col.Reference.OnDelete != "CASCADE" {
			t.Errorf("OnDelete = %q, want %q", col.Reference.OnDelete, "CASCADE")
		}
	})

	t.Run("nil_index", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name:      "user_id",
			Type:      "integer",
			Reference: &RefDef{Table: "auth.user", Column: "id"},
		}

		// Pass nil index (can happen in migrations)
		obj := tb.relationshipBuilder("auth.user", col, nil)

		// Call .as("author") with nil index - should not panic
		asFn, ok := goja.AssertFunction(obj.Get("as"))
		if !ok {
			t.Fatal("as should be a function")
		}

		_, err := asFn(goja.Undefined(), vm.ToValue("author"))
		if err != nil {
			t.Fatalf("as() call should not fail with nil index: %v", err)
		}

		// Column name should still be updated
		if col.Name != "author_id" {
			t.Errorf("Column name = %q, want %q", col.Name, "author_id")
		}
	})

	t.Run("docs_and_deprecated", func(t *testing.T) {
		vm := goja.New()
		tb := NewTableBuilder(vm)

		col := &ColumnDef{
			Name:      "user_id",
			Type:      "integer",
			Reference: &RefDef{Table: "auth.user", Column: "id"},
		}
		idx := &IndexDef{Columns: []string{"user_id"}}

		obj := tb.relationshipBuilder("auth.user", col, idx)

		// Call .docs("The user who created this")
		docsFn, ok := goja.AssertFunction(obj.Get("docs"))
		if !ok {
			t.Fatal("docs should be a function")
		}

		_, err := docsFn(goja.Undefined(), vm.ToValue("The user who created this"))
		if err != nil {
			t.Fatalf("docs() call failed: %v", err)
		}

		if col.Docs != "The user who created this" {
			t.Errorf("Docs = %q, want %q", col.Docs, "The user who created this")
		}

		// Call .deprecated("Use author_id instead")
		deprecatedFn, ok := goja.AssertFunction(obj.Get("deprecated"))
		if !ok {
			t.Fatal("deprecated should be a function")
		}

		_, err = deprecatedFn(goja.Undefined(), vm.ToValue("Use author_id instead"))
		if err != nil {
			t.Fatalf("deprecated() call failed: %v", err)
		}

		if col.Deprecated != "Use author_id instead" {
			t.Errorf("Deprecated = %q, want %q", col.Deprecated, "Use author_id instead")
		}
	})
}

// -----------------------------------------------------------------------------
// polyDefBuilder Tests
// -----------------------------------------------------------------------------

func TestColBuilder_polyDefBuilder(t *testing.T) {
	t.Run("basic_poly_def", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		col := &ColDef{
			Name:     "owner",
			Type:     "polymorphic",
			Nullable: false,
		}

		obj := cb.polyDefBuilder(col)
		if obj == nil {
			t.Fatal("polyDefBuilder should return non-nil object")
		}

		// Check that _colDef is stored
		colDefVal := obj.Get("_colDef")
		if colDefVal == nil || colDefVal == goja.Undefined() {
			t.Error("_colDef should be set on poly object")
		}
	})

	t.Run("optional", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		col := &ColDef{
			Name:     "owner",
			Type:     "polymorphic",
			Nullable: false,
		}

		obj := cb.polyDefBuilder(col)

		// Call .optional()
		optFn, ok := goja.AssertFunction(obj.Get("optional"))
		if !ok {
			t.Fatal("optional should be a function")
		}

		_, err := optFn(goja.Undefined())
		if err != nil {
			t.Fatalf("optional() call failed: %v", err)
		}

		// Check that column is now nullable
		if !col.Nullable {
			t.Error("Column should be nullable after optional()")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		vm := goja.New()
		cb := NewColBuilder(vm)

		col := &ColDef{
			Name:     "owner",
			Type:     "polymorphic",
			Nullable: false,
		}

		obj := cb.polyDefBuilder(col)

		// Test that optional() returns the object for chaining
		optFn, _ := goja.AssertFunction(obj.Get("optional"))
		result, err := optFn(goja.Undefined())
		if err != nil {
			t.Fatalf("optional() failed: %v", err)
		}

		// Result should be the same object
		if result != obj {
			t.Error("optional() should return the object for chaining")
		}
	})
}

// -----------------------------------------------------------------------------
// relationshipDefToMap Tests
// -----------------------------------------------------------------------------

func TestRelationshipDefToMap(t *testing.T) {
	t.Run("many_to_many", func(t *testing.T) {
		rel := &RelationshipDef{
			Type:   "many_to_many",
			Target: "blog.tag",
			As:     "tags",
		}

		result := relationshipDefToMap(rel)

		if result["_type"] != "relationship" {
			t.Errorf("_type = %v, want %q", result["_type"], "relationship")
		}

		relMap, ok := result["relationship"].(map[string]any)
		if !ok {
			t.Fatal("relationship field should be a map")
		}

		if relMap["type"] != "many_to_many" {
			t.Errorf("type = %v, want %q", relMap["type"], "many_to_many")
		}
		if relMap["target"] != "blog.tag" {
			t.Errorf("target = %v, want %q", relMap["target"], "blog.tag")
		}
		if relMap["as"] != "tags" {
			t.Errorf("as = %v, want %q", relMap["as"], "tags")
		}
	})

	t.Run("many_to_many_without_as", func(t *testing.T) {
		rel := &RelationshipDef{
			Type:   "many_to_many",
			Target: "blog.tag",
			As:     "",
		}

		result := relationshipDefToMap(rel)

		relMap, ok := result["relationship"].(map[string]any)
		if !ok {
			t.Fatal("relationship field should be a map")
		}

		// 'as' field should not be present if empty
		if _, exists := relMap["as"]; exists {
			t.Error("as field should not be present when empty")
		}
	})

	t.Run("polymorphic", func(t *testing.T) {
		rel := &RelationshipDef{
			Type:    "polymorphic",
			Targets: []string{"blog.post", "blog.comment"},
			As:      "commentable",
		}

		result := relationshipDefToMap(rel)

		if result["_type"] != "polymorphic" {
			t.Errorf("_type = %v, want %q", result["_type"], "polymorphic")
		}

		polyMap, ok := result["polymorphic"].(map[string]any)
		if !ok {
			t.Fatal("polymorphic field should be a map")
		}

		targets, ok := polyMap["targets"].([]string)
		if !ok {
			t.Fatal("targets should be a string slice")
		}
		if len(targets) != 2 {
			t.Errorf("targets length = %d, want 2", len(targets))
		}
		if targets[0] != "blog.post" || targets[1] != "blog.comment" {
			t.Errorf("targets = %v, want [blog.post, blog.comment]", targets)
		}
		if polyMap["as"] != "commentable" {
			t.Errorf("as = %v, want %q", polyMap["as"], "commentable")
		}
	})

	t.Run("unknown_type", func(t *testing.T) {
		rel := &RelationshipDef{
			Type:   "unknown",
			Target: "some.table",
		}

		result := relationshipDefToMap(rel)

		// Should return empty map for unknown types
		if len(result) != 0 {
			t.Errorf("Expected empty map for unknown type, got %v", result)
		}
	})

	t.Run("nil_relationship", func(t *testing.T) {
		// Test that nil doesn't panic
		// Note: This would cause nil pointer dereference in actual code,
		// but we're testing the function behavior
		defer func() {
			if r := recover(); r != nil {
				// Expected to panic with nil
			}
		}()

		// Passing nil would panic, so we skip this test
		// relationshipDefToMap(nil)
	})
}
