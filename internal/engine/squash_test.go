package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestGenerateBaseline_Empty(t *testing.T) {
	ops := GenerateBaseline(nil)
	if ops != nil {
		t.Errorf("expected nil for empty tables, got %d ops", len(ops))
	}
}

func TestGenerateBaseline_SingleTable(t *testing.T) {
	tables := []*ast.TableDef{
		{
			Namespace: "public",
			Name:      "users",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "name", Type: "text"},
			},
		},
	}

	ops := GenerateBaseline(tables)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	ct, ok := ops[0].(*ast.CreateTable)
	if !ok {
		t.Fatalf("expected CreateTable, got %T", ops[0])
	}
	if ct.Name != "users" {
		t.Errorf("expected table name 'users', got %q", ct.Name)
	}
	if len(ct.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(ct.Columns))
	}
}

func TestGenerateBaseline_WithIndexes(t *testing.T) {
	tables := []*ast.TableDef{
		{
			Namespace: "public",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "integer"}},
			Indexes: []*ast.IndexDef{
				{Name: "idx_users_id", Columns: []string{"id"}, Unique: true},
			},
		},
	}

	ops := GenerateBaseline(tables)
	if len(ops) != 2 {
		t.Fatalf("expected 2 ops (create_table + create_index), got %d", len(ops))
	}
	if _, ok := ops[0].(*ast.CreateTable); !ok {
		t.Errorf("expected CreateTable first, got %T", ops[0])
	}
	ci, ok := ops[1].(*ast.CreateIndex)
	if !ok {
		t.Fatalf("expected CreateIndex second, got %T", ops[1])
	}
	if ci.Name != "idx_users_id" {
		t.Errorf("expected index name 'idx_users_id', got %q", ci.Name)
	}
	if !ci.Unique {
		t.Error("expected unique index")
	}
}

func TestGenerateBaseline_FKDependencyOrder(t *testing.T) {
	tables := []*ast.TableDef{
		{
			Namespace: "public",
			Name:      "posts",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "integer"}, {Name: "user_id", Type: "integer"}},
			ForeignKeys: []*ast.ForeignKeyDef{
				{RefTable: "public.users"},
			},
		},
		{
			Namespace: "public",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "integer"}},
		},
	}

	ops := GenerateBaseline(tables)
	// users should come before posts due to FK dependency
	var tableOrder []string
	for _, op := range ops {
		if ct, ok := op.(*ast.CreateTable); ok {
			tableOrder = append(tableOrder, ct.Name)
		}
	}
	if len(tableOrder) < 2 {
		t.Fatalf("expected at least 2 create_table ops, got %d", len(tableOrder))
	}
	if tableOrder[0] != "users" || tableOrder[1] != "posts" {
		t.Errorf("expected [users, posts] order, got %v", tableOrder)
	}
}

func TestSortTablesByDependency_Empty(t *testing.T) {
	result := sortTablesByDependency(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSortTablesByDependency_Single(t *testing.T) {
	tables := []*ast.TableDef{{Namespace: "ns", Name: "a"}}
	result := sortTablesByDependency(tables)
	if len(result) != 1 || result[0].Name != "a" {
		t.Errorf("expected single table 'a'")
	}
}

func TestSortTablesByDependency_CircularDeps(t *testing.T) {
	// Circular: a -> b -> a
	tables := []*ast.TableDef{
		{
			Namespace:   "ns",
			Name:        "a",
			ForeignKeys: []*ast.ForeignKeyDef{{RefTable: "ns.b"}},
		},
		{
			Namespace:   "ns",
			Name:        "b",
			ForeignKeys: []*ast.ForeignKeyDef{{RefTable: "ns.a"}},
		},
	}
	result := sortTablesByDependency(tables)
	// Should still return both tables (circular deps appended alphabetically)
	if len(result) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(result))
	}
}
