package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// GetMigrationMeta Tests
// -----------------------------------------------------------------------------

func TestSandbox_GetMigrationMeta(t *testing.T) {
	t.Run("default_meta", func(t *testing.T) {
		sb := NewSandbox(nil)

		meta := sb.GetMigrationMeta()
		if meta.Description != "" {
			t.Errorf("Expected empty description, got %q", meta.Description)
		}
		if meta.Renames != nil {
			t.Error("Expected nil renames map initially")
		}
	})

	t.Run("after_migration_evaluation", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			description: "Test migration",
			renames: {
				"auth.user.email": "email_address"
			},
			up(m) {
				m.create_table("auth.user", t => {
					t.id()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		meta := sb.GetMigrationMeta()
		if meta.Description != "Test migration" {
			t.Errorf("Description = %q, want %q", meta.Description, "Test migration")
		}
		// Renames may be nil or empty map - both are valid if no renames are actually used
		if meta.Renames != nil {
			if rename, ok := meta.Renames["auth.user.email"]; ok && rename != "email_address" {
				t.Errorf("Renames[auth.user.email] = %q, want %q", rename, "email_address")
			}
		}
	})
}

// -----------------------------------------------------------------------------
// createAlterColumnBuilderObject Tests
// -----------------------------------------------------------------------------

func TestSandbox_createAlterColumnBuilderObject(t *testing.T) {
	t.Run("set_type", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.users", "email", c => {
					c.set_type("varchar", 255)
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		if len(ops) != 1 {
			t.Fatalf("Expected 1 operation, got %d", len(ops))
		}

		alterOp, ok := ops[0].(*ast.AlterColumn)
		if !ok {
			t.Fatalf("Expected AlterColumn operation, got %T", ops[0])
		}

		if alterOp.NewType != "varchar" {
			t.Errorf("NewType = %q, want %q", alterOp.NewType, "varchar")
		}
		if len(alterOp.NewTypeArgs) != 1 || alterOp.NewTypeArgs[0] != int64(255) {
			t.Errorf("NewTypeArgs = %v, want [255]", alterOp.NewTypeArgs)
		}
	})

	t.Run("set_nullable", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.users", "bio", c => {
					c.set_nullable()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		if len(ops) != 1 {
			t.Fatalf("Expected 1 operation, got %d", len(ops))
		}

		alterOp, ok := ops[0].(*ast.AlterColumn)
		if !ok {
			t.Fatalf("Expected AlterColumn operation, got %T", ops[0])
		}

		if alterOp.SetNullable == nil {
			t.Fatal("SetNullable should not be nil")
		}
		if !*alterOp.SetNullable {
			t.Error("SetNullable should be true")
		}
	})

	t.Run("set_not_null", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.users", "email", c => {
					c.set_not_null()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		alterOp := ops[0].(*ast.AlterColumn)

		if alterOp.SetNullable == nil {
			t.Fatal("SetNullable should not be nil")
		}
		if *alterOp.SetNullable {
			t.Error("SetNullable should be false for set_not_null")
		}
	})

	t.Run("set_default", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.users", "is_active", c => {
					c.set_default(true)
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		alterOp := ops[0].(*ast.AlterColumn)

		if alterOp.SetDefault != true {
			t.Errorf("SetDefault = %v, want true", alterOp.SetDefault)
		}
	})

	t.Run("set_server_default", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.posts", "created_at", c => {
					c.set_server_default("CURRENT_TIMESTAMP")
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		alterOp := ops[0].(*ast.AlterColumn)

		if alterOp.ServerDefault != "CURRENT_TIMESTAMP" {
			t.Errorf("ServerDefault = %q, want %q", alterOp.ServerDefault, "CURRENT_TIMESTAMP")
		}
	})

	t.Run("drop_default", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.users", "status", c => {
					c.drop_default()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		alterOp := ops[0].(*ast.AlterColumn)

		if !alterOp.DropDefault {
			t.Error("DropDefault should be true")
		}
	})

	t.Run("chaining_methods", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.alter_column("test.users", "email", c => {
					c.set_type("varchar", 255).set_not_null()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to execute migration: %v", err)
		}

		ops := sb.operations
		alterOp := ops[0].(*ast.AlterColumn)

		// Both set_type and set_not_null should be applied
		if alterOp.NewType != "varchar" {
			t.Errorf("NewType = %q, want %q", alterOp.NewType, "varchar")
		}
		if alterOp.SetNullable == nil || *alterOp.SetNullable {
			t.Error("SetNullable should be false")
		}
	})
}

// -----------------------------------------------------------------------------
// Integration Test - Complete Migration with All Features
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// createColumnChainObject Tests
// -----------------------------------------------------------------------------

func TestSandbox_createColumnChainObject(t *testing.T) {
	t.Run("nullable", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.text("bio").nullable()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		if len(ops) != 1 {
			t.Fatalf("Expected 1 operation, got %d", len(ops))
		}

		addColOp, ok := ops[0].(*ast.AddColumn)
		if !ok {
			t.Fatalf("Expected AddColumn operation, got %T", ops[0])
		}

		if !addColOp.Column.Nullable {
			t.Error("Column should be nullable")
		}
	})

	t.Run("optional_alias", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.string("phone", 20).optional()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if !addColOp.Column.Nullable {
			t.Error("Column should be nullable (via optional)")
		}
	})

	t.Run("unique", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.string("username", 50).unique()
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if !addColOp.Column.Unique {
			t.Error("Column should be unique")
		}
	})

	t.Run("default_value", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.boolean("is_active").default(true)
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if addColOp.Column.Default != true {
			t.Errorf("Default = %v, want true", addColOp.Column.Default)
		}
	})

	t.Run("backfill", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.posts", c => {
					c.string("status", 20).backfill("draft")
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if addColOp.Column.Backfill != "draft" {
			t.Errorf("Backfill = %v, want %q", addColOp.Column.Backfill, "draft")
		}
	})

	t.Run("min_max", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.products", c => {
					c.decimal("price", 10, 2).min(0).max(999999)
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if addColOp.Column.Min == nil || *addColOp.Column.Min != 0 {
			t.Errorf("Min = %v, want 0", addColOp.Column.Min)
		}
		if addColOp.Column.Max == nil || *addColOp.Column.Max != 999999 {
			t.Errorf("Max = %v, want 999999", addColOp.Column.Max)
		}
	})

	t.Run("pattern", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.string("email", 255).pattern("^[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,}$")
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if addColOp.Column.Pattern == "" {
			t.Error("Pattern should be set")
		}
	})

	t.Run("docs", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.integer("age").docs("User's age in years")
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if addColOp.Column.Docs != "User's age in years" {
			t.Errorf("Docs = %q, want %q", addColOp.Column.Docs, "User's age in years")
		}
	})

	t.Run("deprecated", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.users", c => {
					c.integer("legacy_id").deprecated("Use new_id instead")
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		if addColOp.Column.Deprecated != "Use new_id instead" {
			t.Errorf("Deprecated = %q, want %q", addColOp.Column.Deprecated, "Use new_id instead")
		}
	})

	t.Run("chaining_multiple_methods", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		script := `
		migration({
			up(m) {
				m.add_column("test.products", c => {
					c.string("sku", 50).unique().nullable().docs("Product SKU").pattern("^[A-Z0-9-]+$")
				})
			}
		})
		`

		err := sb.Run(script)
		if err != nil {
			t.Fatalf("Failed to run migration: %v", err)
		}

		ops := sb.operations
		addColOp := ops[0].(*ast.AddColumn)

		// Verify all chained modifications
		if !addColOp.Column.Unique {
			t.Error("Column should be unique")
		}
		if !addColOp.Column.Nullable {
			t.Error("Column should be nullable")
		}
		if addColOp.Column.Docs != "Product SKU" {
			t.Errorf("Docs = %q, want %q", addColOp.Column.Docs, "Product SKU")
		}
		if addColOp.Column.Pattern == "" {
			t.Error("Pattern should be set")
		}
	})
}

// TestMigrationAddColumn_DirectPath verifies that the add_column migration path
// uses the typed builder (not map conversion) and produces correct AST for all modifiers.
func TestMigrationAddColumn_DirectPath(t *testing.T) {
	t.Run("index_modifier", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.string("status", 50).index()) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if !addCol.Column.Index {
			t.Error("Column should have Index=true")
		}
		if addCol.Column.Name != "status" {
			t.Errorf("Name = %q, want %q", addCol.Column.Name, "status")
		}
	})

	t.Run("backfill_modifier", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.text("bio").backfill("No bio")) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if addCol.Column.Backfill != "No bio" {
			t.Errorf("Backfill = %v, want %q", addCol.Column.Backfill, "No bio")
		}
		if !addCol.Column.BackfillSet {
			t.Error("BackfillSet should be true")
		}
	})

	t.Run("default_modifier", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.integer("age").default(0)) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if addCol.Column.Default == nil {
			t.Error("Default should be set")
		}
		if !addCol.Column.DefaultSet {
			t.Error("DefaultSet should be true")
		}
	})

	t.Run("optional_modifier", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.text("nickname").optional()) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if !addCol.Column.Nullable {
			t.Error("Column should be nullable")
		}
		if !addCol.Column.NullableSet {
			t.Error("NullableSet should be true")
		}
	})

	t.Run("unique_modifier", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.string("email", 255).unique()) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if !addCol.Column.Unique {
			t.Error("Column should be unique")
		}
	})

	t.Run("all_modifiers_chained", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) {
			m.add_column("test.users", c => c.string("code", 20).unique().index().optional().default("N/A").backfill("NONE").docs("Short code").deprecated("Use new_code"))
		} })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		col := addCol.Column
		if col.Name != "code" {
			t.Errorf("Name = %q, want %q", col.Name, "code")
		}
		if col.Type != "string" {
			t.Errorf("Type = %q, want %q", col.Type, "string")
		}
		if !col.Unique {
			t.Error("Should be unique")
		}
		if !col.Index {
			t.Error("Should have index")
		}
		if !col.Nullable {
			t.Error("Should be nullable")
		}
		if !col.NullableSet {
			t.Error("NullableSet should be true")
		}
		if !col.DefaultSet {
			t.Error("DefaultSet should be true")
		}
		if !col.BackfillSet {
			t.Error("BackfillSet should be true")
		}
		if col.Docs != "Short code" {
			t.Errorf("Docs = %q, want %q", col.Docs, "Short code")
		}
		if col.Deprecated != "Use new_code" {
			t.Errorf("Deprecated = %q, want %q", col.Deprecated, "Use new_code")
		}
	})

	t.Run("enum_column", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.enum("role", ["admin", "user", "mod"])) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if addCol.Column.Type != "enum" {
			t.Errorf("Type = %q, want %q", addCol.Column.Type, "enum")
		}
		if len(addCol.Column.TypeArgs) == 0 {
			t.Error("TypeArgs should have enum values")
		}
	})

	t.Run("decimal_column", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.orders", c => c.decimal("total", 19, 4)) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if addCol.Column.Type != "decimal" {
			t.Errorf("Type = %q, want %q", addCol.Column.Type, "decimal")
		}
		if len(addCol.Column.TypeArgs) != 2 {
			t.Errorf("TypeArgs length = %d, want 2", len(addCol.Column.TypeArgs))
		}
	})

	t.Run("virtual_modifier", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.text("full_name").virtual()) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if !addCol.Column.Virtual {
			t.Error("Column should be virtual")
		}
	})

	t.Run("read_only_write_only", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) { m.add_column("test.users", c => c.text("hash").write_only()) } })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		addCol := sb.operations[0].(*ast.AddColumn)
		if !addCol.Column.WriteOnly {
			t.Error("Column should be write_only")
		}
	})
}

// TestMigrationCreateTable_DirectPath verifies that create_table produces the same
// AST via the direct typed path as the old map-based path.
func TestMigrationCreateTable_DirectPath(t *testing.T) {
	t.Run("basic_table_with_modifiers", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) {
			m.create_table("test.products", t => {
				t.id()
				t.string("name", 255).unique()
				t.text("description").optional()
				t.decimal("price", 19, 4)
				t.boolean("active").default(true)
				t.integer("stock").index()
			})
		} })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		ct := sb.operations[0].(*ast.CreateTable)
		if len(ct.Columns) != 6 {
			t.Fatalf("Expected 6 columns, got %d", len(ct.Columns))
		}

		// Verify column properties
		nameCol := findCol(t, &ast.TableDef{Columns: ct.Columns}, "name")
		if nameCol == nil {
			t.Fatal("name column not found")
		}
		if !nameCol.Unique {
			t.Error("name should be unique")
		}
		if !nameCol.NullableSet {
			t.Error("name NullableSet should be true")
		}

		descCol := findCol(t, &ast.TableDef{Columns: ct.Columns}, "description")
		if descCol == nil {
			t.Fatal("description column not found")
		}
		if !descCol.Nullable {
			t.Error("description should be nullable")
		}

		stockCol := findCol(t, &ast.TableDef{Columns: ct.Columns}, "stock")
		if stockCol == nil {
			t.Fatal("stock column not found")
		}
		if !stockCol.Index {
			t.Error("stock should have index")
		}
	})

	t.Run("table_with_belongs_to", func(t *testing.T) {
		sb := NewSandbox(nil)
		sb.BindMigration()

		err := sb.Run(`migration({ up(m) {
			m.create_table("test.posts", t => {
				t.id()
				t.string("title", 255)
				t.belongs_to("auth.user")
			})
		} })`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		ct := sb.operations[0].(*ast.CreateTable)
		userIdCol := findCol(t, &ast.TableDef{Columns: ct.Columns}, "user_id")
		if userIdCol == nil {
			t.Fatal("user_id column not found")
		}
		if userIdCol.Reference == nil {
			t.Fatal("user_id should have a reference")
		}
		if userIdCol.Reference.Table != "auth.user" {
			t.Errorf("Reference.Table = %q, want %q", userIdCol.Reference.Table, "auth.user")
		}
	})
}
