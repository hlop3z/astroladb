package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// createMigrationObject Tests - Additional Migration Operations
// -----------------------------------------------------------------------------

func TestMigrationObject_DropColumn(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	script := `
	migration({
		description: "Drop column test",
		up(m) {
			m.drop_column("test.users", "old_field")
		}
	})
	`

	err := sb.Run(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Check operations
	if len(sb.operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
	}

	op, ok := sb.operations[0].(*ast.DropColumn)
	if !ok {
		t.Fatalf("Expected *ast.DropColumn, got %T", sb.operations[0])
	}

	if op.Namespace != "test" || op.Table_ != "users" || op.Name != "old_field" {
		t.Errorf("DropColumn = {ns: %s, table: %s, name: %s}, want {ns: 'test', table: 'users', name: 'old_field'}",
			op.Namespace, op.Table_, op.Name)
	}
}

func TestMigrationObject_RenameColumn(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	script := `
	migration({
		description: "Rename column test",
		up(m) {
			m.rename_column("test.users", "old_name", "new_name")
		}
	})
	`

	err := sb.Run(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Check operations
	if len(sb.operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
	}

	op, ok := sb.operations[0].(*ast.RenameColumn)
	if !ok {
		t.Fatalf("Expected *ast.RenameColumn, got %T", sb.operations[0])
	}

	if op.Namespace != "test" || op.Table_ != "users" {
		t.Errorf("RenameColumn namespace/table = {ns: %s, table: %s}, want {ns: 'test', table: 'users'}",
			op.Namespace, op.Table_)
	}
	if op.OldName != "old_name" || op.NewName != "new_name" {
		t.Errorf("RenameColumn names = {old: %s, new: %s}, want {old: 'old_name', new: 'new_name'}",
			op.OldName, op.NewName)
	}
}

func TestMigrationObject_DropTable(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	script := `
	migration({
		description: "Drop table test",
		up(m) {
			m.drop_table("test.old_table")
		}
	})
	`

	err := sb.Run(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Check operations
	if len(sb.operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
	}

	op, ok := sb.operations[0].(*ast.DropTable)
	if !ok {
		t.Fatalf("Expected *ast.DropTable, got %T", sb.operations[0])
	}

	if op.Namespace != "test" || op.Name != "old_table" {
		t.Errorf("DropTable = {ns: %s, name: %s}, want {ns: 'test', name: 'old_table'}",
			op.Namespace, op.Name)
	}
}

func TestMigrationObject_RenameTable(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	script := `
	migration({
		description: "Rename table test",
		up(m) {
			m.rename_table("test.old_name", "new_name")
		}
	})
	`

	err := sb.Run(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Check operations
	if len(sb.operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
	}

	op, ok := sb.operations[0].(*ast.RenameTable)
	if !ok {
		t.Fatalf("Expected *ast.RenameTable, got %T", sb.operations[0])
	}

	if op.Namespace != "test" {
		t.Errorf("RenameTable namespace = %s, want 'test'", op.Namespace)
	}
	if op.OldName != "old_name" || op.NewName != "new_name" {
		t.Errorf("RenameTable names = {old: %s, new: %s}, want {old: 'old_name', new: 'new_name'}",
			op.OldName, op.NewName)
	}
}

func TestMigrationObject_CreateIndex(t *testing.T) {
	t.Run("basic_index", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			description: "Create index test",
			up(m) {
				m.create_index("test.users", ["email"])
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		// Check operations
		if len(sb.operations) != 1 {
			t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
		}

		op, ok := sb.operations[0].(*ast.CreateIndex)
		if !ok {
			t.Fatalf("Expected *ast.CreateIndex, got %T", sb.operations[0])
		}

		if op.Namespace != "test" || op.Table_ != "users" {
			t.Errorf("CreateIndex ref = {ns: %s, table: %s}, want {ns: 'test', table: 'users'}",
				op.Namespace, op.Table_)
		}
		if len(op.Columns) != 1 || op.Columns[0] != "email" {
			t.Errorf("CreateIndex columns = %v, want ['email']", op.Columns)
		}
		// Auto-generated name should exist
		if op.Name == "" {
			t.Error("CreateIndex should auto-generate name")
		}
	})

	t.Run("unique_index_with_name", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			description: "Create unique index with name",
			up(m) {
				m.create_index("test.users", ["username"], {
					unique: true,
					name: "idx_users_username_unique"
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		op, ok := sb.operations[0].(*ast.CreateIndex)
		if !ok {
			t.Fatalf("Expected *ast.CreateIndex, got %T", sb.operations[0])
		}

		if !op.Unique {
			t.Error("Index should be unique")
		}
		if op.Name != "idx_users_username_unique" {
			t.Errorf("Index name = %s, want 'idx_users_username_unique'", op.Name)
		}
	})

	t.Run("composite_index", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			description: "Create composite index",
			up(m) {
				m.create_index("test.posts", ["user_id", "created_at"])
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		op, ok := sb.operations[0].(*ast.CreateIndex)
		if !ok {
			t.Fatalf("Expected *ast.CreateIndex, got %T", sb.operations[0])
		}

		if len(op.Columns) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(op.Columns))
		}
		if op.Columns[0] != "user_id" || op.Columns[1] != "created_at" {
			t.Errorf("Columns = %v, want ['user_id', 'created_at']", op.Columns)
		}
	})
}

func TestMigrationObject_DropIndex(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	script := `
	migration({
		description: "Drop index test",
		up(m) {
			m.drop_index("idx_users_email")
		}
	})
	`

	err := sb.Run(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Check operations
	if len(sb.operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
	}

	op, ok := sb.operations[0].(*ast.DropIndex)
	if !ok {
		t.Fatalf("Expected *ast.DropIndex, got %T", sb.operations[0])
	}

	if op.Name != "idx_users_email" {
		t.Errorf("DropIndex name = %s, want 'idx_users_email'", op.Name)
	}
}

func TestMigrationObject_SQL(t *testing.T) {
	t.Run("sql_without_down", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			description: "Raw SQL test",
			up(m) {
				m.sql("CREATE EXTENSION IF NOT EXISTS pgcrypto")
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		// Check operations
		if len(sb.operations) != 1 {
			t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
		}

		op, ok := sb.operations[0].(*ast.RawSQL)
		if !ok {
			t.Fatalf("Expected *ast.RawSQL, got %T", sb.operations[0])
		}

		if op.SQL != "CREATE EXTENSION IF NOT EXISTS pgcrypto" {
			t.Errorf("RawSQL = %s, want 'CREATE EXTENSION IF NOT EXISTS pgcrypto'", op.SQL)
		}
	})

	t.Run("sql_with_down", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			description: "Raw SQL with down",
			up(m) {
				m.sql(
					"CREATE EXTENSION IF NOT EXISTS pgcrypto",
					"DROP EXTENSION IF EXISTS pgcrypto"
				)
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		// Check operations
		if len(sb.operations) != 1 {
			t.Fatalf("Expected 1 operation, got %d", len(sb.operations))
		}

		op, ok := sb.operations[0].(*ast.RawSQL)
		if !ok {
			t.Fatalf("Expected *ast.RawSQL, got %T", sb.operations[0])
		}

		if op.SQL != "CREATE EXTENSION IF NOT EXISTS pgcrypto" {
			t.Errorf("RawSQL = %s", op.SQL)
		}
		// Note: Down SQL is stored in metadata, not in the RawSQL operation itself
	})
}

func TestMigrationObject_MultipleOperations(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	script := `
	migration({
		description: "Multiple operations test",
		up(m) {
			m.create_table("test.products", t => {
				t.id()
				t.string("name", 255)
			})
			m.create_index("test.products", ["name"])
			m.sql("ALTER TABLE test.products ENABLE ROW LEVEL SECURITY")
		}
	})
	`

	err := sb.Run(script)
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Should have 3 operations: CreateTable, CreateIndex, RawSQL
	if len(sb.operations) != 3 {
		t.Fatalf("Expected 3 operations, got %d", len(sb.operations))
	}

	// Check operation types
	if _, ok := sb.operations[0].(*ast.CreateTable); !ok {
		t.Errorf("Operation 0 should be CreateTable, got %T", sb.operations[0])
	}
	if _, ok := sb.operations[1].(*ast.CreateIndex); !ok {
		t.Errorf("Operation 1 should be CreateIndex, got %T", sb.operations[1])
	}
	if _, ok := sb.operations[2].(*ast.RawSQL); !ok {
		t.Errorf("Operation 2 should be RawSQL, got %T", sb.operations[2])
	}
}
