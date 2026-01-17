package introspect

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
)

// -----------------------------------------------------------------------------
// FKAccumulator Tests
// -----------------------------------------------------------------------------

func TestNewFKAccumulator(t *testing.T) {
	acc := NewFKAccumulator()
	if acc == nil {
		t.Fatal("NewFKAccumulator() returned nil")
	}
	if acc.fks == nil {
		t.Error("fks map should be initialized")
	}
	if acc.order == nil {
		t.Error("order slice should be initialized")
	}
	if len(acc.Values()) != 0 {
		t.Error("new accumulator should have no values")
	}
}

func TestFKAccumulator_Add(t *testing.T) {
	t.Run("single_column_fk", func(t *testing.T) {
		acc := NewFKAccumulator()
		acc.Add("fk_posts_user", "user_id", "auth_users", "id", "CASCADE", "NO ACTION")

		values := acc.Values()
		if len(values) != 1 {
			t.Fatalf("Values() = %d entries, want 1", len(values))
		}

		fk := values[0]
		if fk.Name != "fk_posts_user" {
			t.Errorf("Name = %q, want %q", fk.Name, "fk_posts_user")
		}
		if len(fk.Columns) != 1 || fk.Columns[0] != "user_id" {
			t.Errorf("Columns = %v, want [user_id]", fk.Columns)
		}
		if fk.RefTable != "auth_users" {
			t.Errorf("RefTable = %q, want %q", fk.RefTable, "auth_users")
		}
		if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
			t.Errorf("RefColumns = %v, want [id]", fk.RefColumns)
		}
		if fk.OnDelete != "CASCADE" {
			t.Errorf("OnDelete = %q, want %q", fk.OnDelete, "CASCADE")
		}
		if fk.OnUpdate != "" {
			t.Errorf("OnUpdate = %q, want empty (NO ACTION)", fk.OnUpdate)
		}
	})

	t.Run("composite_fk", func(t *testing.T) {
		acc := NewFKAccumulator()
		// Add first column
		acc.Add("fk_composite", "col1", "ref_table", "ref_col1", "SET NULL", "CASCADE")
		// Add second column (same FK name)
		acc.Add("fk_composite", "col2", "ref_table", "ref_col2", "SET NULL", "CASCADE")

		values := acc.Values()
		if len(values) != 1 {
			t.Fatalf("Values() = %d entries, want 1 (composite FK)", len(values))
		}

		fk := values[0]
		if len(fk.Columns) != 2 {
			t.Errorf("Columns = %d, want 2", len(fk.Columns))
		}
		if fk.Columns[0] != "col1" || fk.Columns[1] != "col2" {
			t.Errorf("Columns = %v, want [col1, col2]", fk.Columns)
		}
		if len(fk.RefColumns) != 2 {
			t.Errorf("RefColumns = %d, want 2", len(fk.RefColumns))
		}
		if fk.RefColumns[0] != "ref_col1" || fk.RefColumns[1] != "ref_col2" {
			t.Errorf("RefColumns = %v, want [ref_col1, ref_col2]", fk.RefColumns)
		}
	})

	t.Run("multiple_fks", func(t *testing.T) {
		acc := NewFKAccumulator()
		acc.Add("fk_first", "col_a", "table_a", "id", "CASCADE", "")
		acc.Add("fk_second", "col_b", "table_b", "id", "RESTRICT", "")

		values := acc.Values()
		if len(values) != 2 {
			t.Fatalf("Values() = %d entries, want 2", len(values))
		}

		if values[0].Name != "fk_first" {
			t.Errorf("first FK name = %q, want %q", values[0].Name, "fk_first")
		}
		if values[1].Name != "fk_second" {
			t.Errorf("second FK name = %q, want %q", values[1].Name, "fk_second")
		}
	})
}

func TestFKAccumulator_SetActions(t *testing.T) {
	t.Run("existing_fk", func(t *testing.T) {
		acc := NewFKAccumulator()
		acc.Add("fk_test", "col", "ref_table", "ref_col", "", "")

		acc.SetActions("fk_test", "CASCADE", "SET NULL")

		values := acc.Values()
		if len(values) != 1 {
			t.Fatal("expected 1 FK")
		}
		if values[0].OnDelete != "CASCADE" {
			t.Errorf("OnDelete = %q, want %q", values[0].OnDelete, "CASCADE")
		}
		if values[0].OnUpdate != "SET NULL" {
			t.Errorf("OnUpdate = %q, want %q", values[0].OnUpdate, "SET NULL")
		}
	})

	t.Run("non_existent_fk", func(t *testing.T) {
		acc := NewFKAccumulator()
		// Should not panic when FK doesn't exist
		acc.SetActions("nonexistent", "CASCADE", "CASCADE")

		if len(acc.Values()) != 0 {
			t.Error("should have no values after setting actions on nonexistent FK")
		}
	})
}

func TestFKAccumulator_Values(t *testing.T) {
	t.Run("preserves_order", func(t *testing.T) {
		acc := NewFKAccumulator()
		acc.Add("fk_c", "c", "t", "id", "", "")
		acc.Add("fk_a", "a", "t", "id", "", "")
		acc.Add("fk_b", "b", "t", "id", "", "")

		values := acc.Values()
		if len(values) != 3 {
			t.Fatalf("Values() = %d entries, want 3", len(values))
		}

		// Should preserve insertion order, not alphabetical
		if values[0].Name != "fk_c" {
			t.Errorf("values[0].Name = %q, want %q", values[0].Name, "fk_c")
		}
		if values[1].Name != "fk_a" {
			t.Errorf("values[1].Name = %q, want %q", values[1].Name, "fk_a")
		}
		if values[2].Name != "fk_b" {
			t.Errorf("values[2].Name = %q, want %q", values[2].Name, "fk_b")
		}
	})

	t.Run("empty_accumulator", func(t *testing.T) {
		acc := NewFKAccumulator()
		values := acc.Values()
		if values == nil {
			t.Error("Values() should return empty slice, not nil")
		}
		if len(values) != 0 {
			t.Errorf("Values() = %d entries, want 0", len(values))
		}
	})
}

func TestFKAccumulator_Names(t *testing.T) {
	t.Run("returns_names_in_order", func(t *testing.T) {
		acc := NewFKAccumulator()
		acc.Add("fk_first", "a", "t", "id", "", "")
		acc.Add("fk_second", "b", "t", "id", "", "")
		acc.Add("fk_third", "c", "t", "id", "", "")

		names := acc.Names()
		if len(names) != 3 {
			t.Fatalf("Names() = %d entries, want 3", len(names))
		}
		if names[0] != "fk_first" || names[1] != "fk_second" || names[2] != "fk_third" {
			t.Errorf("Names() = %v, want [fk_first, fk_second, fk_third]", names)
		}
	})

	t.Run("empty_accumulator", func(t *testing.T) {
		acc := NewFKAccumulator()
		names := acc.Names()
		if len(names) != 0 {
			t.Errorf("Names() = %d entries, want 0", len(names))
		}
	})
}

// -----------------------------------------------------------------------------
// markUniqueColumnsAndFilterAutoIndexes Tests
// -----------------------------------------------------------------------------

func TestMarkUniqueColumnsAndFilterAutoIndexes(t *testing.T) {
	t.Run("single_column_unique_index", func(t *testing.T) {
		columns := []*ast.ColumnDef{
			{Name: "id", Type: "integer"},
			{Name: "email", Type: "string", Unique: false},
			{Name: "name", Type: "string"},
		}
		indexes := []*ast.IndexDef{
			{Name: "idx_email", Columns: []string{"email"}, Unique: true},
		}

		result := markUniqueColumnsAndFilterAutoIndexes(columns, indexes)

		// email column should now be marked as Unique
		if !columns[1].Unique {
			t.Error("email column should be marked as Unique")
		}

		// Index should still be returned (not auto-generated)
		if len(result) != 1 {
			t.Errorf("result = %d indexes, want 1", len(result))
		}
	})

	t.Run("multi_column_unique_index", func(t *testing.T) {
		columns := []*ast.ColumnDef{
			{Name: "first_name", Type: "string"},
			{Name: "last_name", Type: "string"},
		}
		indexes := []*ast.IndexDef{
			{Name: "idx_name", Columns: []string{"first_name", "last_name"}, Unique: true},
		}

		result := markUniqueColumnsAndFilterAutoIndexes(columns, indexes)

		// Columns should NOT be marked as Unique (multi-column index)
		if columns[0].Unique {
			t.Error("first_name should not be marked Unique (part of multi-column index)")
		}
		if columns[1].Unique {
			t.Error("last_name should not be marked Unique (part of multi-column index)")
		}

		// Index should still be returned
		if len(result) != 1 {
			t.Errorf("result = %d indexes, want 1", len(result))
		}
	})

	t.Run("filter_sqlite_autoindex", func(t *testing.T) {
		columns := []*ast.ColumnDef{
			{Name: "email", Type: "string"},
		}
		indexes := []*ast.IndexDef{
			{Name: "sqlite_autoindex_users_1", Columns: []string{"email"}, Unique: true},
		}

		result := markUniqueColumnsAndFilterAutoIndexes(columns, indexes)

		// email should be marked Unique
		if !columns[0].Unique {
			t.Error("email column should be marked as Unique")
		}

		// Auto-generated index should be filtered out
		if len(result) != 0 {
			t.Errorf("result = %d indexes, want 0 (autoindex filtered)", len(result))
		}
	})

	t.Run("non_unique_index_not_modified", func(t *testing.T) {
		columns := []*ast.ColumnDef{
			{Name: "status", Type: "string", Unique: false},
		}
		indexes := []*ast.IndexDef{
			{Name: "idx_status", Columns: []string{"status"}, Unique: false},
		}

		result := markUniqueColumnsAndFilterAutoIndexes(columns, indexes)

		// Column should NOT be marked Unique
		if columns[0].Unique {
			t.Error("status should not be marked Unique (non-unique index)")
		}

		// Index should be returned
		if len(result) != 1 {
			t.Errorf("result = %d indexes, want 1", len(result))
		}
	})

	t.Run("mixed_indexes", func(t *testing.T) {
		columns := []*ast.ColumnDef{
			{Name: "id", Type: "integer"},
			{Name: "email", Type: "string"},
			{Name: "slug", Type: "string"},
			{Name: "created_at", Type: "datetime"},
		}
		indexes := []*ast.IndexDef{
			{Name: "sqlite_autoindex_posts_1", Columns: []string{"email"}, Unique: true},
			{Name: "idx_slug", Columns: []string{"slug"}, Unique: true},
			{Name: "idx_created", Columns: []string{"created_at"}, Unique: false},
		}

		result := markUniqueColumnsAndFilterAutoIndexes(columns, indexes)

		// email should be Unique (autoindex)
		if !columns[1].Unique {
			t.Error("email should be marked as Unique")
		}
		// slug should be Unique (explicit index)
		if !columns[2].Unique {
			t.Error("slug should be marked as Unique")
		}
		// created_at should NOT be Unique
		if columns[3].Unique {
			t.Error("created_at should not be marked as Unique")
		}

		// Only idx_slug and idx_created should be returned
		if len(result) != 2 {
			t.Errorf("result = %d indexes, want 2", len(result))
		}
	})

	t.Run("empty_inputs", func(t *testing.T) {
		result := markUniqueColumnsAndFilterAutoIndexes(nil, nil)
		if result == nil {
			t.Error("result should be empty slice, not nil")
		}
		if len(result) != 0 {
			t.Errorf("result = %d indexes, want 0", len(result))
		}
	})

	t.Run("column_not_found", func(t *testing.T) {
		columns := []*ast.ColumnDef{
			{Name: "id", Type: "integer"},
		}
		indexes := []*ast.IndexDef{
			// Index references column that doesn't exist
			{Name: "idx_nonexistent", Columns: []string{"nonexistent"}, Unique: true},
		}

		// Should not panic
		result := markUniqueColumnsAndFilterAutoIndexes(columns, indexes)
		if len(result) != 1 {
			t.Errorf("result = %d indexes, want 1", len(result))
		}
	})
}

// -----------------------------------------------------------------------------
// New Factory Tests
// -----------------------------------------------------------------------------

func TestNew(t *testing.T) {
	t.Run("postgres_dialect", func(t *testing.T) {
		d := dialect.Get("postgres")
		if d == nil {
			t.Skip("postgres dialect not available")
		}
		result := New(nil, d)
		if result == nil {
			t.Error("New() should return introspector for postgres")
		}
	})

	t.Run("sqlite_dialect", func(t *testing.T) {
		d := dialect.Get("sqlite")
		if d == nil {
			t.Skip("sqlite dialect not available")
		}
		result := New(nil, d)
		if result == nil {
			t.Error("New() should return introspector for sqlite")
		}
	})

	t.Run("unsupported_dialect", func(t *testing.T) {
		// Using a dialect we don't support (only postgres/sqlite supported)
		// The dialect.Get returns nil for unknown names, so we can't test this
		// directly without mocking. Instead, we just verify that postgres and sqlite work.
	})
}
