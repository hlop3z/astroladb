package runtime

import (
	"strings"
	"testing"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/runtime/schema"
)

func TestBindSQL(t *testing.T) {
	vm := goja.New()
	BindSQL(vm)

	t.Run("sql function exists", func(t *testing.T) {
		result, err := vm.RunString("typeof sql")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "function" {
			t.Errorf("sql should be a function, got %q", result.String())
		}
	})

	t.Run("sql returns correct structure", func(t *testing.T) {
		result, err := vm.RunString("JSON.stringify(sql('NOW()'))")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"_type":"sql_expr"`) {
			t.Errorf("sql() should return _type: 'sql_expr', got %s", resultStr)
		}
		if !strings.Contains(resultStr, `"expr":"NOW()"`) {
			t.Errorf("sql() should return expr: 'NOW()', got %s", resultStr)
		}
	})

	t.Run("sql with complex expression", func(t *testing.T) {
		result, err := vm.RunString("sql('COALESCE(a, b, c)').expr")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "COALESCE(a, b, c)" {
			t.Errorf("sql().expr = %q, want %q", result.String(), "COALESCE(a, b, c)")
		}
	})
}

func TestObjectAPI_CoreTypes(t *testing.T) {
	sb := NewSandbox(nil)

	tests := []struct {
		name     string
		code     string
		wantType string
	}{
		{"string", `table({ name: col.string(100) })`, "string"},
		{"text", `table({ content: col.text() })`, "text"},
		{"integer", `table({ count: col.integer() })`, "integer"},
		{"float", `table({ rating: col.float() })`, "float"},
		{"decimal", `table({ price: col.decimal(10, 2) })`, "decimal"},
		{"boolean", `table({ active: col.boolean() })`, "boolean"},
		{"date", `table({ birth_date: col.date() })`, "date"},
		{"time", `table({ start_time: col.time() })`, "time"},
		{"datetime", `table({ created: col.datetime() })`, "datetime"},
		{"uuid", `table({ token: col.uuid() })`, "uuid"},
		{"json", `table({ data: col.json() })`, "json"},
		{"base64", `table({ binary: col.base64() })`, "base64"},
		{"enum", `table({ status: col.enum(["a", "b"]) })`, "enum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableDef, err := sb.EvalSchema(tt.code, "test", "entity")
			if err != nil {
				t.Fatalf("Error: %v", err)
			}
			// Find the column (skip auto-added id column if present)
			found := false
			for _, col := range tableDef.Columns {
				if col.Name == "id" {
					continue
				}
				if col.Type != tt.wantType {
					t.Errorf("column type = %q, want %q", col.Type, tt.wantType)
				}
				found = true
				break
			}
			if !found {
				t.Error("expected at least one non-id column")
			}
		})
	}
}

func TestObjectAPI_ColumnModifiers(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("optional", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ name: col.string(100).optional() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "name")
		if !col.Nullable {
			t.Error("optional() should set Nullable to true")
		}
	})

	t.Run("unique", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ email: col.string(255).unique() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "email")
		if !col.Unique {
			t.Error("unique() should set Unique to true")
		}
	})

	t.Run("default value", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ status: col.string(50).default("active") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "status")
		if col.Default != "active" {
			t.Errorf("Default = %v, want %q", col.Default, "active")
		}
	})

	t.Run("backfill value", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ category: col.string(50).backfill("unknown") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "category")
		if col.Backfill != "unknown" {
			t.Errorf("Backfill = %v, want %q", col.Backfill, "unknown")
		}
	})

	t.Run("min constraint", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ age: col.integer().min(0) })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "age")
		if col.Min == nil || *col.Min != 0 {
			t.Errorf("Min = %v, want 0", col.Min)
		}
	})

	t.Run("max constraint", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ age: col.integer().max(150) })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "age")
		if col.Max == nil || *col.Max != 150 {
			t.Errorf("Max = %v, want 150", col.Max)
		}
	})

	t.Run("pattern", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ code: col.string(10).pattern("^[A-Z]{2}-\\d{4}$") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "code")
		expected := `^[A-Z]{2}-\d{4}$`
		if col.Pattern != expected {
			t.Errorf("Pattern = %q, want %q", col.Pattern, expected)
		}
	})

	t.Run("docs", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ email: col.string(255).docs("User email address") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "email")
		if col.Docs != "User email address" {
			t.Errorf("Docs = %q, want %q", col.Docs, "User email address")
		}
	})

	t.Run("deprecated", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ old_email: col.string(255).deprecated("Use email instead") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "old_email")
		if col.Deprecated != "Use email instead" {
			t.Errorf("Deprecated = %q, want %q", col.Deprecated, "Use email instead")
		}
	})
}

func TestObjectAPI_SemanticTypes(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("email", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ contact: col.email() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "contact")
		if col.Type != "string" {
			t.Errorf("Type = %q, want %q", col.Type, "string")
		}
		if col.Format != "email" {
			t.Errorf("Format = %q, want %q", col.Format, "email")
		}
		if col.Pattern == "" {
			t.Error("should have email pattern")
		}
	})

	t.Run("username", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ handle: col.username() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "handle")
		if col.Min == nil || *col.Min != 3 {
			t.Errorf("Min = %v, want 3", col.Min)
		}
		if col.Pattern == "" {
			t.Error("should have username pattern")
		}
	})

	t.Run("password_hash", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ pw: col.password_hash() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "pw")
		if col.Type != "string" {
			t.Errorf("Type = %q, want %q", col.Type, "string")
		}
	})

	t.Run("phone", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ phone_number: col.phone() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "phone_number")
		if col.Type != "string" {
			t.Errorf("Type = %q, want %q", col.Type, "string")
		}
	})

	t.Run("name", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ full_name: col.name() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "full_name")
		if col.Type != "string" {
			t.Errorf("Type = %q, want %q", col.Type, "string")
		}
	})

	t.Run("title", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ heading: col.title() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "heading")
		if col.Type != "string" {
			t.Errorf("Type = %q, want %q", col.Type, "string")
		}
	})

	t.Run("slug", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ url_slug: col.slug() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "url_slug")
		if !col.Unique {
			t.Error("slug should be unique")
		}
		if col.Pattern == "" {
			t.Error("should have slug pattern")
		}
	})

	t.Run("body", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ content: col.body() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "content")
		if col.Type != "text" {
			t.Errorf("Type = %q, want %q", col.Type, "text")
		}
	})

	t.Run("money", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ price: col.money() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "price")
		if col.Type != "decimal" {
			t.Errorf("Type = %q, want %q", col.Type, "decimal")
		}
		if col.Min == nil || *col.Min != 0 {
			t.Errorf("Min = %v, want 0", col.Min)
		}
	})

	t.Run("flag with default true", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ is_enabled: col.flag(true) })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "is_enabled")
		if col.Type != "boolean" {
			t.Errorf("Type = %q, want %q", col.Type, "boolean")
		}
		if col.Default != true {
			t.Errorf("Default = %v, want true", col.Default)
		}
	})

	t.Run("flag with default false", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ is_active: col.flag() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "is_active")
		if col.Type != "boolean" {
			t.Errorf("Type = %q, want %q", col.Type, "boolean")
		}
		if col.Default != false {
			t.Errorf("Default = %v, want false", col.Default)
		}
	})
}

func TestObjectAPI_Relationships(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("basic belongs_to", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ user: col.belongs_to("auth.user") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "user_id")
		if col == nil {
			t.Fatal("expected user_id column")
		}
		if col.Reference == nil {
			t.Fatal("expected Reference to be set")
		}
		if col.Reference.Table != "auth.user" {
			t.Errorf("Reference.Table = %q, want %q", col.Reference.Table, "auth.user")
		}
	})

	t.Run("belongs_to with optional", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ user: col.belongs_to("auth.user").optional() })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "user_id")
		if !col.Nullable {
			t.Error("belongs_to with .optional() should be nullable")
		}
	})

	t.Run("belongs_to with on_delete cascade", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ user: col.belongs_to("auth.user").on_delete("cascade") })`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		col := findCol(t, tableDef, "user_id")
		if col.Reference == nil {
			t.Fatal("expected Reference to be set")
		}
		if col.Reference.OnDelete != "cascade" {
			t.Errorf("Reference.OnDelete = %q, want %q", col.Reference.OnDelete, "cascade")
		}
	})
}

func TestObjectAPI_TableChain(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("timestamps", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ name: col.string(100) }).timestamps()`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if findCol(t, tableDef, "created_at") == nil {
			t.Error("timestamps() should add created_at")
		}
		if findCol(t, tableDef, "updated_at") == nil {
			t.Error("timestamps() should add updated_at")
		}
	})

	t.Run("soft_delete", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ name: col.string(100) }).soft_delete()`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if findCol(t, tableDef, "deleted_at") == nil {
			t.Error("soft_delete() should add deleted_at")
		}
	})

	t.Run("sortable", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ name: col.string(100) }).sortable()`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if findCol(t, tableDef, "position") == nil {
			t.Error("sortable() should add position")
		}
	})

	t.Run("index", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ email: col.string(255) }).index("email")`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if len(tableDef.Indexes) == 0 {
			t.Fatal("expected at least one index")
		}
		found := false
		for _, idx := range tableDef.Indexes {
			for _, c := range idx.Columns {
				if c == "email" {
					found = true
					if idx.Unique {
						t.Error("index() should not be unique")
					}
				}
			}
		}
		if !found {
			t.Error("index should contain 'email' column")
		}
	})

	t.Run("unique constraint", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ email: col.string(255) }).unique("email")`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		found := false
		for _, idx := range tableDef.Indexes {
			for _, c := range idx.Columns {
				if c == "email" && idx.Unique {
					found = true
				}
			}
		}
		if !found {
			t.Error("unique() should create a unique index on email")
		}
	})

	t.Run("docs", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ name: col.string(100) }).docs("Users table")`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if tableDef.Docs != "Users table" {
			t.Errorf("Docs = %q, want %q", tableDef.Docs, "Users table")
		}
	})

	t.Run("deprecated", func(t *testing.T) {
		tableDef, err := sb.EvalSchema(`table({ name: col.string(100) }).deprecated("Use new_users instead")`, "test", "entity")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if tableDef.Deprecated != "Use new_users instead" {
			t.Errorf("Deprecated = %q, want %q", tableDef.Deprecated, "Use new_users instead")
		}
	})
}

func TestExtractTableName(t *testing.T) {
	// Note: Convention is singular table names (user, not users)
	// ExtractTableName just extracts the table part, no singularization needed
	tests := []struct {
		ref  string
		want string
	}{
		{"auth.user", "user"},
		{".role", "role"},
		{"post", "post"},
		{"auth.user_profile", "user_profile"},
		{"item", "item"},
		{"data", "data"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := schema.ExtractTableName(tt.ref)
			if got != tt.want {
				t.Errorf("schema.ExtractTableName(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}

func TestNewBindingsContext(t *testing.T) {
	vm := goja.New()
	ctx := NewBindingsContext(vm, "auth")

	if ctx == nil {
		t.Fatal("NewBindingsContext() returned nil")
	}
	if ctx.vm != vm {
		t.Error("vm should be set")
	}
	if ctx.namespace != "auth" {
		t.Errorf("namespace = %q, want %q", ctx.namespace, "auth")
	}
	if ctx.tables == nil {
		t.Error("tables should be initialized")
	}
}

func TestParseRef(t *testing.T) {
	// Using singular table names per project convention
	tests := []struct {
		ref       string
		wantNS    string
		wantTable string
	}{
		{"auth.user", "auth", "user"},
		{"blog.post", "blog", "post"},
		{"user", "", "user"},
		{".role", "", "role"},
		{"app.auth.user", "app.auth", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			ns, table := parseRef(tt.ref)
			if ns != tt.wantNS {
				t.Errorf("namespace = %q, want %q", ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("table = %q, want %q", table, tt.wantTable)
			}
		})
	}
}

func TestSandbox_BindMigration(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	t.Run("migration function exists", func(t *testing.T) {
		result, err := sb.RunWithResult("typeof migration")
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "function" {
			t.Errorf("migration should be a function, got %q", result.String())
		}
	})

	t.Run("drop_table operation", func(t *testing.T) {
		code := `
			migration({
				up: function(m) {
					m.drop_table('auth.old_users');
				}
			});
		`
		err := sb.Run(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
	})
}

// TestMigration_NoSemanticTypes verifies that semantic types are NOT available
// in migration create_table. Migrations should only use low-level types.
func TestMigration_NoSemanticTypes(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindMigration()

	// List of semantic types that should NOT be available in migrations
	semanticTypes := []string{
		"email", "username", "password_hash", "phone", "name", "title",
		"slug", "body", "summary", "url", "ip", "ipv4", "ipv6", "user_agent",
		"money", "percentage", "counter", "quantity", "rating", "duration",
		"token", "code", "country", "currency", "locale", "timezone", "color",
		"markdown", "html", "flag",
	}

	for _, typeName := range semanticTypes {
		t.Run(typeName+"_not_available", func(t *testing.T) {
			code := `
				var result = "not_run";
				migration({
					up: function(m) {
						m.create_table("test.table", function(t) {
							result = typeof t.` + typeName + `;
						});
					}
				});
				result;
			`
			result, err := sb.RunWithResult(code)
			if err != nil {
				t.Fatalf("Error running code: %v", err)
			}
			if result.String() != "undefined" {
				t.Errorf("t.%s should be undefined in migrations, got %q", typeName, result.String())
			}
		})
	}

	// Verify low-level types ARE available
	lowLevelTypes := []string{
		"id", "string", "text", "integer", "float", "decimal",
		"boolean", "date", "time", "datetime", "uuid", "json", "base64", "enum",
	}

	for _, typeName := range lowLevelTypes {
		t.Run(typeName+"_is_available", func(t *testing.T) {
			code := `
				var result = "not_run";
				migration({
					up: function(m) {
						m.create_table("test.table", function(t) {
							result = typeof t.` + typeName + `;
						});
					}
				});
				result;
			`
			result, err := sb.RunWithResult(code)
			if err != nil {
				t.Fatalf("Error running code: %v", err)
			}
			if result.String() != "function" {
				t.Errorf("t.%s should be a function in migrations, got %q", typeName, result.String())
			}
		})
	}
}

// findCol is a test helper that finds a column by name in a TableDef.
// Returns nil if not found (does not fail the test).
func findCol(t *testing.T, tableDef *ast.TableDef, name string) *ast.ColumnDef {
	t.Helper()
	for _, c := range tableDef.Columns {
		if c.Name == name {
			return c
		}
	}
	return nil
}
