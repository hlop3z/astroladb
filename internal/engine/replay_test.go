package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestReplayOperations_CreateTable(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true},
				{Name: "name", Type: "string", TypeArgs: []any{100}},
			},
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	if schema.Count() != 1 {
		t.Errorf("Expected 1 table, got %d", schema.Count())
	}

	table, exists := schema.GetTable("auth.user")
	if !exists {
		t.Fatal("Expected table 'auth.user' not found")
	}

	if len(table.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(table.Columns))
	}
}

func TestReplayOperations_AddColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column:   &ast.ColumnDef{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	if len(table.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(table.Columns))
	}

	if !table.HasColumn("email") {
		t.Error("Expected column 'email' not found")
	}
}

func TestReplayOperations_DropColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", TypeArgs: []any{255}},
			},
		},
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "email",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	if len(table.Columns) != 1 {
		t.Errorf("Expected 1 column, got %d", len(table.Columns))
	}

	if table.HasColumn("email") {
		t.Error("Column 'email' should have been dropped")
	}
}

func TestReplayOperations_DropTable(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "session"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.DropTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "session"},
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	if schema.Count() != 1 {
		t.Errorf("Expected 1 table, got %d", schema.Count())
	}

	if _, exists := schema.GetTable("auth.session"); exists {
		t.Error("Table 'auth.session' should have been dropped")
	}
}

func TestReplayOperations_RenameColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "user_name", Type: "string", TypeArgs: []any{100}},
			},
		},
		&ast.RenameColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			OldName:  "user_name",
			NewName:  "username",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")

	if table.HasColumn("user_name") {
		t.Error("Column 'user_name' should have been renamed")
	}
	if !table.HasColumn("username") {
		t.Error("Column 'username' should exist after rename")
	}
}

func TestReplayOperations_RenameTable(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.RenameTable{
			Namespace: "auth",
			OldName:   "users",
			NewName:   "user",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	if _, exists := schema.GetTable("auth.users"); exists {
		t.Error("Table 'auth.users' should have been renamed")
	}
	if _, exists := schema.GetTable("auth.user"); !exists {
		t.Error("Table 'auth.user' should exist after rename")
	}
}

func TestReplayOperations_CreateIndex(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", TypeArgs: []any{255}},
			},
		},
		&ast.CreateIndex{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "idx_user_email",
			Columns:  []string{"email"},
			Unique:   true,
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	if len(table.Indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(table.Indexes))
	}

	if table.Indexes[0].Name != "idx_user_email" {
		t.Errorf("Expected index name 'idx_user_email', got '%s'", table.Indexes[0].Name)
	}
}

func TestReplayOperations_AlterColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "bio", Type: "string", TypeArgs: []any{100}},
			},
		},
		&ast.AlterColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "bio",
			NewType:  "text",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	col := table.GetColumn("bio")
	if col.Type != "text" {
		t.Errorf("Expected column type 'text', got '%s'", col.Type)
	}
}

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
