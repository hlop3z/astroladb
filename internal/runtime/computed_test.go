package runtime

import (
	"testing"
)

// =============================================================================
// Computed Columns - Chainable Modifier (.computed)
// =============================================================================

func TestComputed_BasicChaining(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("string column with fn.concat", func(t *testing.T) {
		code := `
table({
  first_name: col.string(50),
  last_name: col.string(50),
  full_name: col.text().computed(fn.concat(fn.col("first_name"), " ", fn.col("last_name"))),
})`
		tableDef, err := sb.EvalSchema(code, "auth", "user")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		var found bool
		for _, c := range tableDef.Columns {
			if c.Name == "full_name" {
				found = true
				if c.Type != "text" {
					t.Errorf("full_name type = %q, want %q", c.Type, "text")
				}
				if c.Computed == nil {
					t.Error("full_name.Computed should not be nil")
				}
				// Verify the expression structure
				m, ok := c.Computed.(map[string]any)
				if !ok {
					t.Fatalf("Computed should be a map, got %T", c.Computed)
				}
				if m["fn"] != "concat" {
					t.Errorf("Computed fn = %v, want %q", m["fn"], "concat")
				}
			}
		}
		if !found {
			t.Error("full_name column not found")
		}
	})

	t.Run("integer column with fn.years_since", func(t *testing.T) {
		code := `
table({
  birthdate: col.date(),
  age: col.integer().computed(fn.years_since(fn.col("birthdate"))),
})`
		tableDef, err := sb.EvalSchema(code, "auth", "person")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "age" {
				if c.Type != "integer" {
					t.Errorf("age type = %q, want %q", c.Type, "integer")
				}
				if c.Computed == nil {
					t.Fatal("age.Computed should not be nil")
				}
				m := c.Computed.(map[string]any)
				if m["fn"] != "years_since" {
					t.Errorf("fn = %v, want %q", m["fn"], "years_since")
				}
				return
			}
		}
		t.Error("age column not found")
	})

	t.Run("decimal column with fn.mul", func(t *testing.T) {
		code := `
table({
  price: col.decimal(10, 2),
  tax_rate: col.decimal(5, 4),
  total: col.decimal(10, 2).computed(fn.mul(fn.col("price"), fn.add(1, fn.col("tax_rate")))),
})`
		tableDef, err := sb.EvalSchema(code, "shop", "product")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "total" {
				if c.Type != "decimal" {
					t.Errorf("total type = %q, want %q", c.Type, "decimal")
				}
				if c.Computed == nil {
					t.Fatal("total.Computed should not be nil")
				}
				m := c.Computed.(map[string]any)
				if m["fn"] != "mul" {
					t.Errorf("fn = %v, want %q", m["fn"], "mul")
				}
				return
			}
		}
		t.Error("total column not found")
	})

	t.Run("boolean column with fn.gt", func(t *testing.T) {
		code := `
table({
  score: col.float(),
  passing: col.boolean().computed(fn.gt(fn.col("score"), 50)),
})`
		tableDef, err := sb.EvalSchema(code, "exam", "result")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "passing" {
				if c.Type != "boolean" {
					t.Errorf("passing type = %q, want %q", c.Type, "boolean")
				}
				if c.Computed == nil {
					t.Fatal("passing.Computed should not be nil")
				}
				return
			}
		}
		t.Error("passing column not found")
	})
}

func TestComputed_ChainingWithOtherModifiers(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("computed + optional", func(t *testing.T) {
		code := `
table({
  first_name: col.string(50),
  display: col.text().optional().computed(fn.upper(fn.col("first_name"))),
})`
		tableDef, err := sb.EvalSchema(code, "auth", "user")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "display" {
				if !c.Nullable {
					t.Error("display should be nullable")
				}
				if c.Computed == nil {
					t.Error("display.Computed should not be nil")
				}
				return
			}
		}
		t.Error("display column not found")
	})

	t.Run("computed + docs", func(t *testing.T) {
		code := `
table({
  first_name: col.string(50),
  upper_name: col.text().computed(fn.upper(fn.col("first_name"))).docs("Uppercased first name"),
})`
		tableDef, err := sb.EvalSchema(code, "auth", "user")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "upper_name" {
				if c.Computed == nil {
					t.Error("upper_name.Computed should not be nil")
				}
				if c.Docs != "Uppercased first name" {
					t.Errorf("docs = %q, want %q", c.Docs, "Uppercased first name")
				}
				return
			}
		}
		t.Error("upper_name column not found")
	})

	t.Run("optional + computed + deprecated", func(t *testing.T) {
		code := `
table({
  first_name: col.string(50),
  legacy_name: col.text().optional().computed(fn.lower(fn.col("first_name"))).deprecated("Use display_name instead"),
})`
		tableDef, err := sb.EvalSchema(code, "auth", "user")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "legacy_name" {
				if !c.Nullable {
					t.Error("should be nullable")
				}
				if c.Computed == nil {
					t.Error("Computed should not be nil")
				}
				if c.Deprecated != "Use display_name instead" {
					t.Errorf("deprecated = %q, want %q", c.Deprecated, "Use display_name instead")
				}
				return
			}
		}
		t.Error("legacy_name column not found")
	})
}

// =============================================================================
// Expression Builder (fn.*) - All Functions
// =============================================================================

func TestComputed_FnStringFunctions(t *testing.T) {
	sb := NewSandbox(nil)

	tests := []struct {
		name   string
		expr   string
		wantFn string
	}{
		{"concat", `fn.concat(fn.col("a"), " ", fn.col("b"))`, "concat"},
		{"upper", `fn.upper(fn.col("name"))`, "upper"},
		{"lower", `fn.lower(fn.col("name"))`, "lower"},
		{"trim", `fn.trim(fn.col("name"))`, "trim"},
		{"length", `fn.length(fn.col("name"))`, "length"},
		{"substring", `fn.substring(fn.col("name"), 0, 5)`, "substring"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := `
table({
  name: col.string(100),
  result: col.text().computed(` + tt.expr + `),
})`
			tableDef, err := sb.EvalSchema(code, "test", tt.name)
			if err != nil {
				t.Fatalf("EvalSchema failed: %v", err)
			}

			for _, c := range tableDef.Columns {
				if c.Name == "result" {
					if c.Computed == nil {
						t.Fatal("Computed should not be nil")
					}
					m := c.Computed.(map[string]any)
					if m["fn"] != tt.wantFn {
						t.Errorf("fn = %v, want %q", m["fn"], tt.wantFn)
					}
					return
				}
			}
			t.Error("result column not found")
		})
	}
}

func TestComputed_FnMathFunctions(t *testing.T) {
	sb := NewSandbox(nil)

	tests := []struct {
		name   string
		expr   string
		wantFn string
	}{
		{"add", `fn.add(fn.col("a"), fn.col("b"))`, "add"},
		{"sub", `fn.sub(fn.col("a"), fn.col("b"))`, "sub"},
		{"mul", `fn.mul(fn.col("a"), fn.col("b"))`, "mul"},
		{"div", `fn.div(fn.col("a"), fn.col("b"))`, "div"},
		{"abs", `fn.abs(fn.col("a"))`, "abs"},
		{"round", `fn.round(fn.col("a"))`, "round"},
		{"floor", `fn.floor(fn.col("a"))`, "floor"},
		{"ceil", `fn.ceil(fn.col("a"))`, "ceil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := `
table({
  a: col.float(),
  b: col.float(),
  result: col.float().computed(` + tt.expr + `),
})`
			tableDef, err := sb.EvalSchema(code, "test", "math_"+tt.name)
			if err != nil {
				t.Fatalf("EvalSchema failed: %v", err)
			}

			for _, c := range tableDef.Columns {
				if c.Name == "result" {
					if c.Computed == nil {
						t.Fatal("Computed should not be nil")
					}
					m := c.Computed.(map[string]any)
					if m["fn"] != tt.wantFn {
						t.Errorf("fn = %v, want %q", m["fn"], tt.wantFn)
					}
					return
				}
			}
			t.Error("result column not found")
		})
	}
}

func TestComputed_FnDateFunctions(t *testing.T) {
	sb := NewSandbox(nil)

	tests := []struct {
		name   string
		expr   string
		wantFn string
	}{
		{"year", `fn.year(fn.col("d"))`, "year"},
		{"month", `fn.month(fn.col("d"))`, "month"},
		{"day", `fn.day(fn.col("d"))`, "day"},
		{"now", `fn.now()`, "now"},
		{"years_since", `fn.years_since(fn.col("d"))`, "years_since"},
		{"days_since", `fn.days_since(fn.col("d"))`, "days_since"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := `
table({
  d: col.date(),
  result: col.integer().computed(` + tt.expr + `),
})`
			tableDef, err := sb.EvalSchema(code, "test", "date_"+tt.name)
			if err != nil {
				t.Fatalf("EvalSchema failed: %v", err)
			}

			for _, c := range tableDef.Columns {
				if c.Name == "result" {
					if c.Computed == nil {
						t.Fatal("Computed should not be nil")
					}
					m := c.Computed.(map[string]any)
					if m["fn"] != tt.wantFn {
						t.Errorf("fn = %v, want %q", m["fn"], tt.wantFn)
					}
					return
				}
			}
			t.Error("result column not found")
		})
	}
}

func TestComputed_FnConditionalFunctions(t *testing.T) {
	sb := NewSandbox(nil)

	tests := []struct {
		name   string
		expr   string
		wantFn string
	}{
		{"coalesce", `fn.coalesce(fn.col("a"), fn.col("b"), "default")`, "coalesce"},
		{"nullif", `fn.nullif(fn.col("a"), 0)`, "nullif"},
		{"if_null", `fn.if_null(fn.col("a"), "fallback")`, "if_null"},
		{"if_then", `fn.if_then(fn.col("active"), "Yes", "No")`, "if_then"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := `
table({
  a: col.string(50).optional(),
  b: col.string(50).optional(),
  active: col.boolean().default(true),
  result: col.text().computed(` + tt.expr + `),
})`
			tableDef, err := sb.EvalSchema(code, "test", "cond_"+tt.name)
			if err != nil {
				t.Fatalf("EvalSchema failed: %v", err)
			}

			for _, c := range tableDef.Columns {
				if c.Name == "result" {
					if c.Computed == nil {
						t.Fatal("Computed should not be nil")
					}
					m := c.Computed.(map[string]any)
					if m["fn"] != tt.wantFn {
						t.Errorf("fn = %v, want %q", m["fn"], tt.wantFn)
					}
					return
				}
			}
			t.Error("result column not found")
		})
	}
}

func TestComputed_FnComparisonFunctions(t *testing.T) {
	sb := NewSandbox(nil)

	tests := []struct {
		name   string
		expr   string
		wantFn string
	}{
		{"gt", `fn.gt(fn.col("score"), 50)`, "gt"},
		{"gte", `fn.gte(fn.col("score"), 50)`, "gte"},
		{"lt", `fn.lt(fn.col("score"), 50)`, "lt"},
		{"lte", `fn.lte(fn.col("score"), 50)`, "lte"},
		{"eq", `fn.eq(fn.col("score"), 100)`, "eq"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := `
table({
  score: col.float(),
  result: col.boolean().computed(` + tt.expr + `),
})`
			tableDef, err := sb.EvalSchema(code, "test", "cmp_"+tt.name)
			if err != nil {
				t.Fatalf("EvalSchema failed: %v", err)
			}

			for _, c := range tableDef.Columns {
				if c.Name == "result" {
					if c.Computed == nil {
						t.Fatal("Computed should not be nil")
					}
					m := c.Computed.(map[string]any)
					if m["fn"] != tt.wantFn {
						t.Errorf("fn = %v, want %q", m["fn"], tt.wantFn)
					}
					return
				}
			}
			t.Error("result column not found")
		})
	}
}

func TestComputed_FnSQL(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("raw SQL escape hatch", func(t *testing.T) {
		code := `
table({
  price: col.decimal(10, 2),
  tax_rate: col.decimal(5, 4),
  total: col.decimal(10, 2).computed(fn.sql({
    postgres: "price * (1 + tax_rate)",
    sqlite: "price * (1 + tax_rate)",
  })),
})`
		tableDef, err := sb.EvalSchema(code, "shop", "item")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "total" {
				if c.Type != "decimal" {
					t.Errorf("type = %q, want %q", c.Type, "decimal")
				}
				if c.Computed == nil {
					t.Fatal("Computed should not be nil")
				}
				m := c.Computed.(map[string]any)
				if m["fn"] != "raw" {
					t.Errorf("fn = %v, want %q", m["fn"], "raw")
				}
				// Verify SQL expressions are present
				sql, ok := m["sql"].(map[string]any)
				if !ok {
					t.Fatalf("sql should be a map, got %T", m["sql"])
				}
				if sql["postgres"] != "price * (1 + tax_rate)" {
					t.Errorf("postgres = %v", sql["postgres"])
				}
				if sql["sqlite"] != "price * (1 + tax_rate)" {
					t.Errorf("sqlite = %v", sql["sqlite"])
				}
				return
			}
		}
		t.Error("total column not found")
	})
}

// =============================================================================
// Nested Expressions
// =============================================================================

func TestComputed_NestedExpressions(t *testing.T) {
	sb := NewSandbox(nil)

	t.Run("deeply nested", func(t *testing.T) {
		code := `
table({
  first_name: col.string(50),
  last_name: col.string(50),
  display: col.text().computed(
    fn.upper(fn.concat(fn.trim(fn.col("first_name")), " ", fn.trim(fn.col("last_name"))))
  ),
})`
		tableDef, err := sb.EvalSchema(code, "auth", "nested")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "display" {
				if c.Computed == nil {
					t.Fatal("Computed should not be nil")
				}
				m := c.Computed.(map[string]any)
				if m["fn"] != "upper" {
					t.Errorf("outer fn = %v, want %q", m["fn"], "upper")
				}
				// The args should contain the nested concat
				args, ok := m["args"].([]any)
				if !ok {
					t.Fatalf("args should be []any, got %T", m["args"])
				}
				if len(args) == 0 {
					t.Fatal("args should not be empty")
				}
				inner, ok := args[0].(map[string]any)
				if !ok {
					t.Fatalf("inner arg should be map, got %T", args[0])
				}
				if inner["fn"] != "concat" {
					t.Errorf("inner fn = %v, want %q", inner["fn"], "concat")
				}
				return
			}
		}
		t.Error("display column not found")
	})

	t.Run("conditional with comparison", func(t *testing.T) {
		code := `
table({
  score: col.float(),
  grade: col.text().computed(
    fn.if_then(fn.gte(fn.col("score"), 90), "A",
      fn.if_then(fn.gte(fn.col("score"), 80), "B", "C"))
  ),
})`
		tableDef, err := sb.EvalSchema(code, "exam", "grade")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "grade" {
				if c.Computed == nil {
					t.Fatal("Computed should not be nil")
				}
				m := c.Computed.(map[string]any)
				if m["fn"] != "if_then" {
					t.Errorf("fn = %v, want %q", m["fn"], "if_then")
				}
				return
			}
		}
		t.Error("grade column not found")
	})

	t.Run("math chain", func(t *testing.T) {
		code := `
table({
  qty: col.integer(),
  unit_price: col.decimal(10, 2),
  discount: col.decimal(5, 2),
  final_price: col.decimal(10, 2).computed(
    fn.round(fn.mul(fn.col("qty"), fn.sub(fn.col("unit_price"), fn.col("discount"))))
  ),
})`
		tableDef, err := sb.EvalSchema(code, "order", "line")
		if err != nil {
			t.Fatalf("EvalSchema failed: %v", err)
		}

		for _, c := range tableDef.Columns {
			if c.Name == "final_price" {
				if c.Computed == nil {
					t.Fatal("Computed should not be nil")
				}
				m := c.Computed.(map[string]any)
				if m["fn"] != "round" {
					t.Errorf("fn = %v, want %q", m["fn"], "round")
				}
				return
			}
		}
		t.Error("final_price column not found")
	})
}

// =============================================================================
// All Column Types with .computed()
// =============================================================================

func TestComputed_AllColumnTypes(t *testing.T) {
	sb := NewSandbox(nil)

	types := []struct {
		name     string
		colType  string
		wantType string
	}{
		{"string", `col.string(100).computed(fn.col("x"))`, "string"},
		{"text", `col.text().computed(fn.col("x"))`, "text"},
		{"integer", `col.integer().computed(fn.col("x"))`, "integer"},
		{"float", `col.float().computed(fn.col("x"))`, "float"},
		{"decimal", `col.decimal(10, 2).computed(fn.col("x"))`, "decimal"},
		{"boolean", `col.boolean().computed(fn.col("x"))`, "boolean"},
		{"date", `col.date().computed(fn.col("x"))`, "date"},
		{"time", `col.time().computed(fn.col("x"))`, "time"},
		{"datetime", `col.datetime().computed(fn.col("x"))`, "datetime"},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			code := `
table({
  x: col.string(50),
  result: ` + tt.colType + `,
})`
			tableDef, err := sb.EvalSchema(code, "test", "type_"+tt.name)
			if err != nil {
				t.Fatalf("EvalSchema failed: %v", err)
			}

			for _, c := range tableDef.Columns {
				if c.Name == "result" {
					if c.Type != tt.wantType {
						t.Errorf("type = %q, want %q", c.Type, tt.wantType)
					}
					if c.Computed == nil {
						t.Error("Computed should not be nil")
					}
					return
				}
			}
			t.Error("result column not found")
		})
	}
}

// =============================================================================
// Non-computed columns should NOT have Computed set
// =============================================================================

func TestComputed_RegularColumnsUnaffected(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
table({
  id: col.id(),
  name: col.string(100),
  bio: col.text().optional(),
  score: col.float().default(0),
  active: col.boolean().default(true),
})`
	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema failed: %v", err)
	}

	for _, c := range tableDef.Columns {
		if c.Computed != nil {
			t.Errorf("column %q should NOT have Computed set, but has %v", c.Name, c.Computed)
		}
	}
}

// =============================================================================
// Mixed computed and non-computed columns
// =============================================================================

func TestComputed_MixedColumns(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
table({
  id: col.id(),
  first_name: col.string(50),
  last_name: col.string(50),
  full_name: col.text().computed(fn.concat(fn.col("first_name"), " ", fn.col("last_name"))),
  email: col.string(255).unique(),
  age: col.integer().optional(),
  is_adult: col.boolean().computed(fn.gte(fn.col("age"), 18)),
}).timestamps()`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema failed: %v", err)
	}

	computedCols := map[string]bool{}
	regularCols := map[string]bool{}

	for _, c := range tableDef.Columns {
		if c.Computed != nil {
			computedCols[c.Name] = true
		} else {
			regularCols[c.Name] = true
		}
	}

	// Verify computed
	for _, name := range []string{"full_name", "is_adult"} {
		if !computedCols[name] {
			t.Errorf("%q should be computed", name)
		}
	}

	// Verify NOT computed
	for _, name := range []string{"id", "first_name", "last_name", "email", "age", "created_at", "updated_at"} {
		if !regularCols[name] {
			t.Errorf("%q should NOT be computed", name)
		}
	}
}

// =============================================================================
// Table-level methods still work with computed columns
// =============================================================================

func TestComputed_WithTableMethods(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
table({
  id: col.id(),
  name: col.string(100),
  upper_name: col.text().computed(fn.upper(fn.col("name"))),
}).timestamps().soft_delete().index("name")`

	tableDef, err := sb.EvalSchema(code, "app", "entity")
	if err != nil {
		t.Fatalf("EvalSchema failed: %v", err)
	}

	// Verify timestamps exist
	hasCreatedAt := false
	hasUpdatedAt := false
	hasDeletedAt := false
	hasUpperName := false

	for _, c := range tableDef.Columns {
		switch c.Name {
		case "created_at":
			hasCreatedAt = true
		case "updated_at":
			hasUpdatedAt = true
		case "deleted_at":
			hasDeletedAt = true
		case "upper_name":
			hasUpperName = true
			if c.Computed == nil {
				t.Error("upper_name.Computed should not be nil")
			}
		}
	}

	if !hasCreatedAt {
		t.Error("missing created_at")
	}
	if !hasUpdatedAt {
		t.Error("missing updated_at")
	}
	if !hasDeletedAt {
		t.Error("missing deleted_at")
	}
	if !hasUpperName {
		t.Error("missing upper_name")
	}

	// Verify index exists
	if len(tableDef.Indexes) == 0 {
		t.Error("expected at least one index")
	}
}
