package ast

import (
	"testing"
)

func TestSortColumnsForExport(t *testing.T) {
	cols := []*ColumnDef{
		{Name: "zebra", Type: "string"},
		{Name: "id", Type: "id", PrimaryKey: true},
		{Name: "apple", Type: "string"},
		{Name: "mango", Type: "string", Nullable: true},
		{Name: "banana", Type: "string", DefaultSet: true, Default: "hello"},
		{Name: "cherry", Type: "string"},
		{Name: "avocado", Type: "string", Nullable: true},
	}

	sorted := SortColumnsForExport(cols)

	expected := []string{
		"id",      // primary key
		"apple",   // required, alpha
		"cherry",  // required, alpha
		"zebra",   // required, alpha
		"avocado", // optional (nullable), alpha
		"banana",  // optional (has default), alpha
		"mango",   // optional (nullable), alpha
	}

	if len(sorted) != len(expected) {
		t.Fatalf("got %d columns, want %d", len(sorted), len(expected))
	}

	for i, want := range expected {
		if sorted[i].Name != want {
			t.Errorf("sorted[%d] = %q, want %q", i, sorted[i].Name, want)
		}
	}

	// Verify original slice is not mutated
	if cols[0].Name != "zebra" {
		t.Error("original slice was mutated")
	}
}
