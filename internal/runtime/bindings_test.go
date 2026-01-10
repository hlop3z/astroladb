package runtime

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
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

func TestTableBuilder_TypeMethods(t *testing.T) {
	sb := NewSandbox(nil)

	typeMethods := []struct {
		name     string
		call     string
		wantType string
	}{
		{"id", "t.id()", "uuid"},
		{"string", "t.string('name', 100)", "string"},
		{"text", "t.text('content')", "text"},
		{"integer", "t.integer('count')", "integer"},
		{"float", "t.float('rating')", "float"},
		{"decimal", "t.decimal('price', 10, 2)", "decimal"},
		{"boolean", "t.boolean('active')", "boolean"},
		{"date", "t.date('birth_date')", "date"},
		{"time", "t.time('start_time')", "time"},
		{"datetime", "t.datetime('created')", "date_time"},
		{"uuid", "t.uuid('token')", "uuid"},
		{"json", "t.json('data')", "json"},
		{"base64", "t.base64('binary')", "base64"},
		{"enum", "t.enum('status', ['a', 'b'])", "enum"},
	}

	for _, tt := range typeMethods {
		t.Run(tt.name, func(t *testing.T) {
			code := `
				var result = table(function(t) {
					` + tt.call + `;
				});
				result.columns[result.columns.length - 1].type;
			`
			result, err := sb.RunWithResult(code)
			if err != nil {
				t.Fatalf("Error: %v", err)
			}
			if result.String() != tt.wantType {
				t.Errorf("type = %q, want %q", result.String(), tt.wantType)
			}
		})
	}
}

func TestTableBuilder_ColumnModifiers(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("optional", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('name', 100).optional();
			});
			result.columns[0].nullable;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if !result.ToBoolean() {
			t.Error("optional() should set nullable to true")
		}
	})

	t.Run("unique", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('email', 255).unique();
			});
			result.columns[0].unique;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if !result.ToBoolean() {
			t.Error("unique() should set unique to true")
		}
	})

	t.Run("default value", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('status', 50).default('active');
			});
			result.columns[0].default;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "active" {
			t.Errorf("default = %q, want %q", result.String(), "active")
		}
	})

	t.Run("backfill value", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('category', 50).backfill('unknown');
			});
			result.columns[0].backfill;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "unknown" {
			t.Errorf("backfill = %q, want %q", result.String(), "unknown")
		}
	})

	t.Run("min constraint", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.integer('age').min(0);
			});
			result.columns[0].min;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.ToFloat() != 0 {
			t.Errorf("min = %v, want 0", result.Export())
		}
	})

	t.Run("max constraint", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.integer('age').max(150);
			});
			result.columns[0].max;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.ToFloat() != 150 {
			t.Errorf("max = %v, want 150", result.Export())
		}
	})

	t.Run("pattern", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('code', 10).pattern('^[A-Z]{2}-\\d{4}$');
			});
			result.columns[0].pattern;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		expected := `^[A-Z]{2}-\d{4}$`
		if result.String() != expected {
			t.Errorf("pattern = %q, want %q", result.String(), expected)
		}
	})

	t.Run("format with string value", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('data', 255).format('custom-format');
			});
			result.columns[0].format;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "custom-format" {
			t.Errorf("format = %q, want %q", result.String(), "custom-format")
		}
	})

	t.Run("docs", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('email', 255).docs('User email address');
			});
			result.columns[0].docs;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "User email address" {
			t.Errorf("docs = %q, want %q", result.String(), "User email address")
		}
	})

	t.Run("deprecated", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('old_email', 255).deprecated('Use email instead');
			});
			result.columns[0].deprecated;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "Use email instead" {
			t.Errorf("deprecated = %q, want %q", result.String(), "Use email instead")
		}
	})
}

func TestTableBuilder_ChainedModifiers(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.string('code', 50)
				.unique()
				.min(5)
				.max(50)
				.pattern('^[A-Z]+$')
				.docs('Product code');
		});
		JSON.stringify({
			unique: result.columns[0].unique,
			pattern: result.columns[0].pattern,
			min: result.columns[0].min,
			max: result.columns[0].max,
			docs: result.columns[0].docs
		});
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	resultStr := result.String()
	if !strings.Contains(resultStr, `"unique":true`) {
		t.Error("unique should be true")
	}
	if !strings.Contains(resultStr, `"pattern":"^[A-Z]+$"`) {
		t.Error("pattern should be '^[A-Z]+$'")
	}
	if !strings.Contains(resultStr, `"min":5`) {
		t.Error("min should be 5")
	}
	if !strings.Contains(resultStr, `"max":50`) {
		t.Error("max should be 50")
	}
	if !strings.Contains(resultStr, `"docs":"Product code"`) {
		t.Error("docs should be set")
	}
}

func TestTableBuilder_SemanticTypes(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("email type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.email('contact');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"name":"contact"`) {
			t.Error("name should be 'contact'")
		}
		if !strings.Contains(resultStr, `"type":"string"`) {
			t.Error("type should be 'string'")
		}
		if !strings.Contains(resultStr, `"format":"email"`) {
			t.Error("format should be 'email'")
		}
		if !strings.Contains(resultStr, `"pattern"`) {
			t.Error("should have email pattern")
		}
	})

	t.Run("username type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.username('handle');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"name":"handle"`) {
			t.Error("name should be 'handle'")
		}
		if !strings.Contains(resultStr, `"min":3`) {
			t.Error("min should be 3")
		}
		if !strings.Contains(resultStr, `"pattern"`) {
			t.Error("should have username pattern")
		}
	})

	t.Run("money type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.money('price');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"type":"decimal"`) {
			t.Error("type should be 'decimal'")
		}
		if !strings.Contains(resultStr, `"min":0`) {
			t.Error("min should be 0")
		}
	})

	t.Run("counter type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.counter('view_count');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"type":"integer"`) {
			t.Error("type should be 'integer'")
		}
		if !strings.Contains(resultStr, `"default":0`) {
			t.Error("default should be 0")
		}
	})

	t.Run("flag type with default false", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.flag('is_active');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"type":"boolean"`) {
			t.Error("type should be 'boolean'")
		}
		if !strings.Contains(resultStr, `"default":false`) {
			t.Error("default should be false")
		}
	})

	t.Run("flag type with default true", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.flag('is_enabled', true);
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"default":true`) {
			t.Error("default should be true")
		}
	})

	t.Run("slug type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.slug('url_slug');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"unique":true`) {
			t.Error("slug should be unique")
		}
		if !strings.Contains(resultStr, `"pattern"`) {
			t.Error("should have slug pattern")
		}
	})

	t.Run("url type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.url('website');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"format":"uri"`) {
			t.Error("format should be 'uri'")
		}
	})

	t.Run("percentage type", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.percentage('rate');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"min":0`) {
			t.Error("min should be 0")
		}
		if !strings.Contains(resultStr, `"max":100`) {
			t.Error("max should be 100")
		}
	})

	t.Run("datetime alias", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.datetime('created_at');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"type":"date_time"`) {
			t.Error("type should be 'date_time'")
		}
	})

	t.Run("optional alias", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.datetime('deleted_at').optional();
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"nullable":true`) {
			t.Error("optional() should set nullable to true")
		}
	})

	t.Run("semantic type with override", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.username('handle').min(1).max(100);
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		// min should be overridden to 1
		if !strings.Contains(resultStr, `"min":1`) {
			t.Error("min should be overridden to 1")
		}
		if !strings.Contains(resultStr, `"max":100`) {
			t.Error("max should be 100")
		}
	})
}

func TestTableBuilder_Timestamps(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.timestamps();
		});
		JSON.stringify(result.columns.map(function(c) { return c.name; }));
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	resultStr := result.String()
	if !strings.Contains(resultStr, "created_at") {
		t.Error("timestamps() should add created_at")
	}
	if !strings.Contains(resultStr, "updated_at") {
		t.Error("timestamps() should add updated_at")
	}
}

func TestTableBuilder_SoftDelete(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.soft_delete();
		});
		result.columns[0].name;
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.String() != "deleted_at" {
		t.Errorf("soft_delete() should add deleted_at, got %q", result.String())
	}
}

func TestTableBuilder_Sortable(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.sortable();
		});
		result.columns[0].name;
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.String() != "position" {
		t.Errorf("sortable() should add position, got %q", result.String())
	}
}

func TestTableBuilder_BelongsTo(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("basic belongs_to", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user');
			});
			result.columns[0].name;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "user_id" {
			t.Errorf("belongs_to column name = %q, want %q", result.String(), "user_id")
		}
	})

	t.Run("belongs_to with .as() chaining", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user').as('author');
			});
			result.columns[0].name;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "author_id" {
			t.Errorf("belongs_to column name = %q, want %q", result.String(), "author_id")
		}
	})

	t.Run("belongs_to with .optional() chaining", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user').optional();
			});
			result.columns[0].nullable;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if !result.ToBoolean() {
			t.Error("belongs_to with .optional() should be nullable")
		}
	})

	t.Run("belongs_to has reference", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user');
			});
			result.columns[0].reference.table;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "auth.user" {
			t.Errorf("reference.table = %q, want %q", result.String(), "auth.user")
		}
	})

	t.Run("belongs_to with .on_delete() chaining", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user').on_delete('cascade');
			});
			result.columns[0].reference.on_delete;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "cascade" {
			t.Errorf("on_delete = %q, want %q", result.String(), "cascade")
		}
	})

	t.Run("belongs_to with full chaining", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user').as('referred_by').optional().on_delete('set null');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"name":"referred_by_id"`) {
			t.Error("column name should be 'referred_by_id'")
		}
		if !strings.Contains(resultStr, `"nullable":true`) {
			t.Error("should be nullable")
		}
		if !strings.Contains(resultStr, `"on_delete":"set null"`) {
			t.Error("on_delete should be 'set null'")
		}
	})

	t.Run("belongs_to creates index", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user').as('author');
			});
			JSON.stringify(result.indexes[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"author_id"`) {
			t.Error("index should be on author_id")
		}
		if !strings.Contains(resultStr, `"unique":false`) {
			t.Error("belongs_to index should not be unique")
		}
	})
}

func TestTableBuilder_OneToOne(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("basic one_to_one is unique", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.one_to_one('auth.user');
			});
			result.columns[0].unique;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if !result.ToBoolean() {
			t.Error("one_to_one() should set unique to true")
		}
	})

	t.Run("one_to_one with .as() chaining", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.one_to_one('auth.profile').as('profile');
			});
			result.columns[0].name;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.String() != "profile_id" {
			t.Errorf("one_to_one column name = %q, want %q", result.String(), "profile_id")
		}
	})

	t.Run("one_to_one creates unique index", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.one_to_one('auth.profile').as('profile');
			});
			JSON.stringify(result.indexes[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"profile_id"`) {
			t.Error("index should be on profile_id")
		}
		if !strings.Contains(resultStr, `"unique":true`) {
			t.Error("one_to_one index should be unique")
		}
	})

	t.Run("one_to_one with full chaining", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.one_to_one('auth.profile').as('profile').optional().on_delete('cascade');
			});
			JSON.stringify(result.columns[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"name":"profile_id"`) {
			t.Error("column name should be 'profile_id'")
		}
		if !strings.Contains(resultStr, `"unique":true`) {
			t.Error("should be unique")
		}
		if !strings.Contains(resultStr, `"nullable":true`) {
			t.Error("should be nullable")
		}
		if !strings.Contains(resultStr, `"on_delete":"cascade"`) {
			t.Error("on_delete should be 'cascade'")
		}
	})
}

func TestTableBuilder_ValidationErrors(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("belongs_to without namespace errors", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('user');
			});
		`
		_, err := sb.RunWithResult(code)
		if err == nil {
			t.Fatal("Expected error for belongs_to without namespace")
		}
		if !strings.Contains(err.Error(), "missing namespace") {
			t.Errorf("Error should mention missing namespace, got: %v", err)
		}
	})

	t.Run("one_to_one without namespace errors", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.one_to_one('profile');
			});
		`
		_, err := sb.RunWithResult(code)
		if err == nil {
			t.Fatal("Expected error for one_to_one without namespace")
		}
		if !strings.Contains(err.Error(), "missing namespace") {
			t.Errorf("Error should mention missing namespace, got: %v", err)
		}
	})

	t.Run("string without length errors", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('email');
			});
		`
		_, err := sb.RunWithResult(code)
		if err == nil {
			t.Fatal("Expected error for string without length")
		}
		if !strings.Contains(err.Error(), "length required") {
			t.Errorf("Error should mention length required, got: %v", err)
		}
	})

	t.Run("string with zero length errors", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('email', 0);
			});
		`
		_, err := sb.RunWithResult(code)
		if err == nil {
			t.Fatal("Expected error for string with zero length")
		}
		if !strings.Contains(err.Error(), "length required") {
			t.Errorf("Error should mention length required, got: %v", err)
		}
	})

	t.Run("string error suggests semantic type for email", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('email');
			});
		`
		_, err := sb.RunWithResult(code)
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "t.email") {
			t.Errorf("Error should suggest t.email(), got: %v", err)
		}
	})

	t.Run("belongs_to with namespace succeeds", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('auth.user');
			});
			result.columns[0].name;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.String() != "user_id" {
			t.Errorf("Expected user_id, got %s", result.String())
		}
	})

	t.Run("belongs_to with dot prefix succeeds", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.belongs_to('.user');
			});
			result.columns[0].name;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.String() != "user_id" {
			t.Errorf("Expected user_id, got %s", result.String())
		}
	})
}

func TestTableBuilder_Index(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("single column index", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('email', 255);
				t.index('email');
			});
			JSON.stringify(result.indexes[0]);
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		resultStr := result.String()
		if !strings.Contains(resultStr, `"email"`) {
			t.Error("index should contain 'email' column")
		}
		if !strings.Contains(resultStr, `"unique":false`) {
			t.Error("index should not be unique")
		}
	})

	t.Run("multi column index", func(t *testing.T) {
		code := `
			var result = table(function(t) {
				t.string('first_name', 100);
				t.string('last_name', 100);
				t.index('first_name', 'last_name');
			});
			result.indexes[0].columns.length;
		`
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result.ToInteger() != 2 {
			t.Errorf("index columns count = %d, want 2", result.ToInteger())
		}
	})
}

func TestTableBuilder_Unique(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.string('email', 255);
			t.unique('email');
		});
		result.indexes[0].unique;
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !result.ToBoolean() {
		t.Error("unique() should set unique to true")
	}
}

func TestTableBuilder_Docs(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.id();
			t.docs('Users table');
		});
		result.docs;
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.String() != "Users table" {
		t.Errorf("table docs = %q, want %q", result.String(), "Users table")
	}
}

func TestTableBuilder_Deprecated(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.id();
			t.deprecated('Use new_users instead');
		});
		result.deprecated;
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if result.String() != "Use new_users instead" {
		t.Errorf("table deprecated = %q, want %q", result.String(), "Use new_users instead")
	}
}

func TestTableBuilder_DefaultValueWithSQL(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var result = table(function(t) {
			t.datetime('created_at').default(sql('NOW()'));
		});
		JSON.stringify(result.columns[0].default);
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	resultStr := result.String()
	if !strings.Contains(resultStr, "sql_expr") {
		t.Error("default with sql() should have sql_expr type")
	}
	if !strings.Contains(resultStr, "NOW()") {
		t.Error("default with sql() should contain the expression")
	}
}

func TestExtractTableName(t *testing.T) {
	// Note: Convention is singular table names (user, not users)
	// extractTableName just extracts the table part, no singularization needed
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
			got := extractTableName(tt.ref)
			if got != tt.want {
				t.Errorf("extractTableName(%q) = %q, want %q", tt.ref, got, tt.want)
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

func TestSandbox_BindSchema(t *testing.T) {
	sb := NewSandbox(nil)
	sb.BindSchema()

	code := `
		schema('auth', function(s) {
			s.table('users', function(t) {
				t.id();
				t.string('email', 255);
			});
		});
	`
	err := sb.Run(code)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	tables := sb.GetTables()
	if len(tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(tables))
	}

	if len(tables) > 0 {
		if tables[0].Namespace != "auth" {
			t.Errorf("Namespace = %q, want %q", tables[0].Namespace, "auth")
		}
		if tables[0].Name != "users" {
			t.Errorf("Name = %q, want %q", tables[0].Name, "users")
		}
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
			migration(function(m) {
				m.drop_table('auth.old_users');
			});
		`
		err := sb.Run(code)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
	})
}
