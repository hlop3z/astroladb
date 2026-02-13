package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestReplayMigrationsUpTo(t *testing.T) {
	migrations := []Migration{
		{
			Revision: "001",
			Name:     "create_users",
			Operations: []ast.Operation{
				&ast.CreateTable{
					TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
					Columns: []*ast.ColumnDef{
						{Name: "id", Type: "uuid", PrimaryKey: true},
					},
				},
			},
		},
		{
			Revision: "002",
			Name:     "add_email",
			Operations: []ast.Operation{
				&ast.AddColumn{
					TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
					Column:   &ast.ColumnDef{Name: "email", Type: "string", TypeArgs: []any{255}},
				},
			},
		},
		{
			Revision: "003",
			Name:     "add_posts",
			Operations: []ast.Operation{
				&ast.CreateTable{
					TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
					Columns: []*ast.ColumnDef{
						{Name: "id", Type: "uuid", PrimaryKey: true},
						{Name: "title", Type: "string", TypeArgs: []any{200}},
					},
				},
			},
		},
	}

	// Test at revision 001
	schema, err := ReplayMigrationsUpTo(migrations, "001")
	if err != nil {
		t.Fatalf("ReplayMigrationsUpTo failed: %v", err)
	}

	if schema.Count() != 1 {
		t.Errorf("At revision 001: Expected 1 table, got %d", schema.Count())
	}

	table, _ := schema.GetTable("auth.user")
	if len(table.Columns) != 1 {
		t.Errorf("At revision 001: Expected 1 column, got %d", len(table.Columns))
	}

	// Test at revision 002
	schema, err = ReplayMigrationsUpTo(migrations, "002")
	if err != nil {
		t.Fatalf("ReplayMigrationsUpTo failed: %v", err)
	}

	if schema.Count() != 1 {
		t.Errorf("At revision 002: Expected 1 table, got %d", schema.Count())
	}

	table, _ = schema.GetTable("auth.user")
	if len(table.Columns) != 2 {
		t.Errorf("At revision 002: Expected 2 columns, got %d", len(table.Columns))
	}

	// Test at revision 003
	schema, err = ReplayMigrationsUpTo(migrations, "003")
	if err != nil {
		t.Fatalf("ReplayMigrationsUpTo failed: %v", err)
	}

	if schema.Count() != 2 {
		t.Errorf("At revision 003: Expected 2 tables, got %d", schema.Count())
	}
}
