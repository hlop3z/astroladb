package runtime

import (
	"testing"
)

func TestObjectBasedTableAPI(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  author: col.belongs_to("auth.user"),
  is_active: col.flag(true),
}).timestamps()
`

	tableDef, err := sb.EvalSchema(code, "auth", "profile")
	if err != nil {
		t.Fatalf("EvalSchema failed: %v", err)
	}

	if tableDef == nil {
		t.Fatal("tableDef is nil")
	}

	t.Logf("Table: %s.%s", tableDef.Namespace, tableDef.Name)
	t.Logf("Columns: %d", len(tableDef.Columns))

	for _, col := range tableDef.Columns {
		t.Logf("  - %s: %s (nullable=%v, unique=%v, pk=%v)",
			col.Name, col.Type, col.Nullable, col.Unique, col.PrimaryKey)
		if col.Reference != nil {
			t.Logf("      -> FK to %s.%s", col.Reference.Table, col.Reference.Column)
		}
	}

	t.Logf("Indexes: %d", len(tableDef.Indexes))
	for _, idx := range tableDef.Indexes {
		t.Logf("  - %v (unique=%v)", idx.Columns, idx.Unique)
	}

	// Verify expected columns exist
	expectedCols := map[string]bool{
		"id": false, "email": false, "username": false, "password": false,
		"author_id": false, "is_active": false, "created_at": false, "updated_at": false,
	}

	for _, col := range tableDef.Columns {
		if _, ok := expectedCols[col.Name]; ok {
			expectedCols[col.Name] = true
		}
	}

	for name, found := range expectedCols {
		if !found {
			t.Errorf("Expected column %q not found", name)
		}
	}
}

func TestObjectBasedTableAPI_WithChainedMethods(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
table({
  id: col.id(),
  title: col.title(),
  slug: col.slug(),
  author: col.belongs_to("auth.user"),
  category: col.belongs_to("blog.category").optional(),
}).timestamps().soft_delete().index("title").unique("author", "slug")
`

	tableDef, err := sb.EvalSchema(code, "blog", "post")
	if err != nil {
		t.Fatalf("EvalSchema failed: %v", err)
	}

	// Check soft_delete added deleted_at
	hasDeletedAt := false
	for _, col := range tableDef.Columns {
		if col.Name == "deleted_at" {
			hasDeletedAt = true
			if !col.Nullable {
				t.Error("deleted_at should be nullable")
			}
		}
	}
	if !hasDeletedAt {
		t.Error("soft_delete() should add deleted_at column")
	}

	// Check indexes
	t.Logf("Indexes: %d", len(tableDef.Indexes))
	for _, idx := range tableDef.Indexes {
		t.Logf("  - %v (unique=%v)", idx.Columns, idx.Unique)
	}

	// Should have: index on title, unique on (author_id, slug), FK indexes
	hasUniqueAuthorSlug := false
	hasTitleIndex := false
	for _, idx := range tableDef.Indexes {
		if len(idx.Columns) == 2 && idx.Unique {
			if (idx.Columns[0] == "author_id" && idx.Columns[1] == "slug") ||
				(idx.Columns[0] == "slug" && idx.Columns[1] == "author_id") {
				hasUniqueAuthorSlug = true
			}
		}
		if len(idx.Columns) == 1 && idx.Columns[0] == "title" && !idx.Unique {
			hasTitleIndex = true
		}
	}

	if !hasTitleIndex {
		t.Error("Expected index on title")
	}
	if !hasUniqueAuthorSlug {
		t.Error("Expected unique constraint on (author_id, slug)")
	}
}
