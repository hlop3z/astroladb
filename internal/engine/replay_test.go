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

// -----------------------------------------------------------------------------
// DropIndex Tests
// -----------------------------------------------------------------------------

func TestReplayOperations_DropIndex(t *testing.T) {
	t.Run("drop_index_with_table", func(t *testing.T) {
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
			&ast.DropIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_user_email",
			},
		}

		schema, err := ReplayOperations(ops)
		if err != nil {
			t.Fatalf("ReplayOperations failed: %v", err)
		}

		table, _ := schema.GetTable("auth.user")
		if len(table.Indexes) != 0 {
			t.Errorf("Expected 0 indexes after drop, got %d", len(table.Indexes))
		}
	})

	t.Run("drop_index_if_exists_nonexistent", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string"},
				},
			},
			// Create an index first so the table has indexes slice
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_existing",
				Columns:  []string{"email"},
			},
			// Then drop a nonexistent index with IfExists
			&ast.DropIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "nonexistent_idx",
				IfExists: true,
			},
		}

		schema, err := ReplayOperations(ops)
		if err != nil {
			t.Fatalf("ReplayOperations with IfExists should not fail: %v", err)
		}

		if schema.Count() != 1 {
			t.Errorf("Expected 1 table, got %d", schema.Count())
		}

		// The existing index should still be there
		table, _ := schema.GetTable("auth.user")
		if len(table.Indexes) != 1 {
			t.Errorf("Expected 1 index, got %d", len(table.Indexes))
		}
	})

	t.Run("drop_index_search_all_tables", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string"},
				},
			},
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_email",
				Columns:  []string{"email"},
			},
			// Drop without specifying table - should search all tables
			&ast.DropIndex{
				Name: "idx_email",
			},
		}

		schema, err := ReplayOperations(ops)
		if err != nil {
			t.Fatalf("ReplayOperations failed: %v", err)
		}

		table, _ := schema.GetTable("auth.user")
		if len(table.Indexes) != 0 {
			t.Errorf("Expected 0 indexes after drop, got %d", len(table.Indexes))
		}
	})

	t.Run("drop_index_nonexistent_error", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string"},
				},
			},
			// Create an index first so the table has indexes slice
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_existing",
				Columns:  []string{"email"},
			},
			&ast.DropIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "nonexistent_idx",
				IfExists: false, // Should fail
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for nonexistent index")
		}
	})
}

// -----------------------------------------------------------------------------
// AddForeignKey Tests
// -----------------------------------------------------------------------------

func TestReplayOperations_AddForeignKey(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "author_id", Type: "uuid"},
			},
		},
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Namespace: "blog", Table_: "post"},
			Name:       "fk_post_author",
			Columns:    []string{"author_id"},
			RefTable:   "auth.user",
			RefColumns: []string{"id"},
			OnDelete:   "CASCADE",
			OnUpdate:   "NO ACTION",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("blog.post")
	if len(table.ForeignKeys) != 1 {
		t.Fatalf("Expected 1 foreign key, got %d", len(table.ForeignKeys))
	}

	fk := table.ForeignKeys[0]
	if fk.Name != "fk_post_author" {
		t.Errorf("FK name = %q, want %q", fk.Name, "fk_post_author")
	}
	if fk.RefTable != "auth.user" {
		t.Errorf("FK RefTable = %q, want %q", fk.RefTable, "auth.user")
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("FK OnDelete = %q, want %q", fk.OnDelete, "CASCADE")
	}
}

func TestReplayOperations_AddForeignKey_TableNotFound(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Namespace: "blog", Table_: "nonexistent"},
			Name:       "fk_test",
			Columns:    []string{"col"},
			RefTable:   "other",
			RefColumns: []string{"id"},
		},
	}

	_, err := ReplayOperations(ops)
	if err == nil {
		t.Error("ReplayOperations should fail when table does not exist")
	}
}

// -----------------------------------------------------------------------------
// DropForeignKey Tests
// -----------------------------------------------------------------------------

func TestReplayOperations_DropForeignKey(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "author_id", Type: "uuid"},
			},
		},
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Namespace: "blog", Table_: "post"},
			Name:       "fk_post_author",
			Columns:    []string{"author_id"},
			RefTable:   "auth.user",
			RefColumns: []string{"id"},
		},
		&ast.DropForeignKey{
			TableRef: ast.TableRef{Namespace: "blog", Table_: "post"},
			Name:     "fk_post_author",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("blog.post")
	if len(table.ForeignKeys) != 0 {
		t.Errorf("Expected 0 foreign keys after drop, got %d", len(table.ForeignKeys))
	}
}

func TestReplayOperations_DropForeignKey_Idempotent(t *testing.T) {
	// Dropping a non-existent FK should be idempotent
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "author_id", Type: "uuid"},
			},
		},
		// Add a FK first so the table has ForeignKeys slice
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Namespace: "blog", Table_: "post"},
			Name:       "fk_existing",
			Columns:    []string{"author_id"},
			RefTable:   "auth.user",
			RefColumns: []string{"id"},
		},
		// Then try to drop a nonexistent FK - should be idempotent
		&ast.DropForeignKey{
			TableRef: ast.TableRef{Namespace: "blog", Table_: "post"},
			Name:     "nonexistent_fk",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations should be idempotent for missing FK: %v", err)
	}

	if schema.Count() != 2 {
		t.Errorf("Expected 2 tables, got %d", schema.Count())
	}

	// The existing FK should still be there
	table, _ := schema.GetTable("blog.post")
	if len(table.ForeignKeys) != 1 {
		t.Errorf("Expected 1 FK, got %d", len(table.ForeignKeys))
	}
}

func TestReplayOperations_DropForeignKey_TableNotFound(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropForeignKey{
			TableRef: ast.TableRef{Namespace: "blog", Table_: "nonexistent"},
			Name:     "fk_test",
		},
	}

	_, err := ReplayOperations(ops)
	if err == nil {
		t.Error("ReplayOperations should fail when table does not exist")
	}
}

// -----------------------------------------------------------------------------
// Error Cases
// -----------------------------------------------------------------------------

func TestReplayOperations_Errors(t *testing.T) {
	t.Run("create_duplicate_table", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"}, // Duplicate
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for duplicate table")
		}
	})

	t.Run("add_duplicate_column", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Column:   &ast.ColumnDef{Name: "id", Type: "string"}, // Duplicate column
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for duplicate column")
		}
	})

	t.Run("drop_nonexistent_column", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.DropColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "nonexistent",
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for nonexistent column")
		}
	})

	t.Run("rename_nonexistent_column", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.RenameColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				OldName:  "nonexistent",
				NewName:  "new_name",
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for nonexistent column rename")
		}
	})

	t.Run("alter_nonexistent_column", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.AlterColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "nonexistent",
				NewType:  "text",
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for nonexistent column alter")
		}
	})

	t.Run("drop_nonexistent_table", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.DropTable{
				TableOp:  ast.TableOp{Namespace: "auth", Name: "nonexistent"},
				IfExists: false,
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for nonexistent table drop")
		}
	})

	t.Run("drop_table_if_exists", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.DropTable{
				TableOp:  ast.TableOp{Namespace: "auth", Name: "nonexistent"},
				IfExists: true,
			},
		}

		_, err := ReplayOperations(ops)
		if err != nil {
			t.Errorf("DropTable with IfExists should not fail: %v", err)
		}
	})

	t.Run("rename_to_existing_table", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "admin"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
			},
			&ast.RenameTable{
				Namespace: "auth",
				OldName:   "user",
				NewName:   "admin", // Already exists
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail when renaming to existing table")
		}
	})

	t.Run("create_index_table_not_found", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "nonexistent"},
				Name:     "idx_test",
				Columns:  []string{"col"},
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail when table does not exist")
		}
	})

	t.Run("create_duplicate_index", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string"},
				},
			},
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_email",
				Columns:  []string{"email"},
			},
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_email", // Duplicate
				Columns:  []string{"email"},
			},
		}

		_, err := ReplayOperations(ops)
		if err == nil {
			t.Error("ReplayOperations should fail for duplicate index")
		}
	})
}

// -----------------------------------------------------------------------------
// RawSQL and edge cases
// -----------------------------------------------------------------------------

func TestReplayOperations_RawSQL(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
		},
		&ast.RawSQL{
			SQL: "INSERT INTO auth_user (id) VALUES ('test')",
		},
	}

	// RawSQL should be ignored by replay
	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	if schema.Count() != 1 {
		t.Errorf("Expected 1 table, got %d", schema.Count())
	}
}

func TestReplayOperations_NoNamespace(t *testing.T) {
	// Test tables without namespace (for databases that don't support schemas)
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "", Name: "users"},
			Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "", Table_: "users"},
			Column:   &ast.ColumnDef{Name: "name", Type: "string"},
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	// Table should be stored with just the name
	table, exists := schema.GetTable("users")
	if !exists {
		t.Error("Table 'users' should exist")
	}
	if len(table.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(table.Columns))
	}
}

func TestReplayOperations_AlterColumnMultipleChanges(t *testing.T) {
	setNullable := true
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "bio", Type: "string", TypeArgs: []any{100}, Nullable: false, Default: "N/A"},
			},
		},
		&ast.AlterColumn{
			TableRef:    ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:        "bio",
			NewType:     "text",
			SetNullable: &setNullable,
			DropDefault: true,
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	col := table.GetColumn("bio")

	if col.Type != "text" {
		t.Errorf("Column type = %q, want %q", col.Type, "text")
	}
	if !col.Nullable {
		t.Error("Column should be nullable after alter")
	}
	if col.Default != nil {
		t.Errorf("Column default should be nil after DropDefault, got %v", col.Default)
	}
}

func TestReplayOperations_RenameColumnUpdatesIndexes(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "user_email", Type: "string"},
			},
		},
		&ast.CreateIndex{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "idx_email",
			Columns:  []string{"user_email"},
		},
		&ast.RenameColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			OldName:  "user_email",
			NewName:  "email",
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	if len(table.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(table.Indexes))
	}

	// Index should reference the new column name
	if table.Indexes[0].Columns[0] != "email" {
		t.Errorf("Index column = %q, want %q (should be updated after rename)", table.Indexes[0].Columns[0], "email")
	}
}

func TestReplayOperations_CreateIndexAutoName(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string"},
			},
		},
		&ast.CreateIndex{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "", // No name - should be auto-generated
			Columns:  []string{"email"},
		},
	}

	schema, err := ReplayOperations(ops)
	if err != nil {
		t.Fatalf("ReplayOperations failed: %v", err)
	}

	table, _ := schema.GetTable("auth.user")
	if len(table.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(table.Indexes))
	}

	// Name should be auto-generated
	if table.Indexes[0].Name == "" {
		t.Error("Index name should be auto-generated when empty")
	}
}
