package dialect

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// =============================================================================
// This test verifies EVERY SQL output claim in docs/src/content/docs/cols/computed.mdx
// against the actual computedExprSQL function output.
// =============================================================================

func quote(name string) string { return `"` + name + `"` }

// --- String Functions (docs lines 57-64) ---

func TestDocsVerify_StringFunctions(t *testing.T) {
	tests := []struct {
		name   string
		expr   map[string]any
		pgWant string
		slWant string
	}{
		{
			name: "concat",
			expr: map[string]any{"fn": "concat", "args": []any{
				map[string]any{"col": "a"}, map[string]any{"col": "b"}, map[string]any{"col": "c"},
			}},
			pgWant: `("a" || "b" || "c")`,
			slWant: `("a" || "b" || "c")`,
		},
		{
			name:   "upper",
			expr:   map[string]any{"fn": "upper", "args": []any{map[string]any{"col": "col"}}},
			pgWant: `UPPER("col")`,
			slWant: `UPPER("col")`,
		},
		{
			name:   "lower",
			expr:   map[string]any{"fn": "lower", "args": []any{map[string]any{"col": "col"}}},
			pgWant: `LOWER("col")`,
			slWant: `LOWER("col")`,
		},
		{
			name:   "trim",
			expr:   map[string]any{"fn": "trim", "args": []any{map[string]any{"col": "col"}}},
			pgWant: `TRIM("col")`,
			slWant: `TRIM("col")`,
		},
		{
			name:   "length",
			expr:   map[string]any{"fn": "length", "args": []any{map[string]any{"col": "col"}}},
			pgWant: `LENGTH("col")`,
			slWant: `LENGTH("col")`,
		},
		{
			name:   "substring",
			expr:   map[string]any{"fn": "substring", "args": []any{map[string]any{"col": "col"}, 1, 5}},
			pgWant: `SUBSTRING("col" FROM 1 FOR 5)`,
			slWant: `SUBSTRING("col" FROM 1 FOR 5)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/postgres", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "postgres", quote)
			if got != tt.pgWant {
				t.Errorf("postgres:\n  got  = %s\n  want = %s", got, tt.pgWant)
			}
		})
		t.Run(tt.name+"/sqlite", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "sqlite", quote)
			if got != tt.slWant {
				t.Errorf("sqlite:\n  got  = %s\n  want = %s", got, tt.slWant)
			}
		})
	}
}

// --- Math Functions (docs lines 77-86) ---

func TestDocsVerify_MathFunctions(t *testing.T) {
	colA := map[string]any{"col": "a"}
	colB := map[string]any{"col": "b"}

	tests := []struct {
		name string
		expr map[string]any
		want string // same for both dialects
	}{
		{"add", map[string]any{"fn": "add", "args": []any{colA, colB}}, `("a" + "b")`},
		{"sub", map[string]any{"fn": "sub", "args": []any{colA, colB}}, `("a" - "b")`},
		{"mul", map[string]any{"fn": "mul", "args": []any{colA, colB}}, `("a" * "b")`},
		{"div", map[string]any{"fn": "div", "args": []any{colA, colB}}, `("a" / "b")`},
		{"abs", map[string]any{"fn": "abs", "args": []any{colA}}, `ABS("a")`},
		{"round", map[string]any{"fn": "round", "args": []any{colA}}, `ROUND("a")`},
		{"floor", map[string]any{"fn": "floor", "args": []any{colA}}, `FLOOR("a")`},
		{"ceil", map[string]any{"fn": "ceil", "args": []any{colA}}, `CEIL("a")`},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/postgres", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "postgres", quote)
			if got != tt.want {
				t.Errorf("postgres:\n  got  = %s\n  want = %s", got, tt.want)
			}
		})
		t.Run(tt.name+"/sqlite", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "sqlite", quote)
			if got != tt.want {
				t.Errorf("sqlite:\n  got  = %s\n  want = %s", got, tt.want)
			}
		})
	}
}

// --- Date Functions (docs lines 102-109) ---

func TestDocsVerify_DateFunctions(t *testing.T) {
	col := map[string]any{"col": "col"}

	tests := []struct {
		name   string
		expr   map[string]any
		pgWant string
		slWant string
	}{
		{
			name:   "year",
			expr:   map[string]any{"fn": "year", "args": []any{col}},
			pgWant: `EXTRACT(YEAR FROM "col")`,
			slWant: `CAST(STRFTIME('%Y', "col") AS INTEGER)`,
		},
		{
			name:   "month",
			expr:   map[string]any{"fn": "month", "args": []any{col}},
			pgWant: `EXTRACT(MONTH FROM "col")`,
			slWant: `CAST(STRFTIME('%m', "col") AS INTEGER)`,
		},
		{
			name:   "day",
			expr:   map[string]any{"fn": "day", "args": []any{col}},
			pgWant: `EXTRACT(DAY FROM "col")`,
			slWant: `CAST(STRFTIME('%d', "col") AS INTEGER)`,
		},
		{
			name:   "now",
			expr:   map[string]any{"fn": "now"},
			pgWant: `NOW()`,
			slWant: `CURRENT_TIMESTAMP`,
		},
		{
			name:   "years_since",
			expr:   map[string]any{"fn": "years_since", "args": []any{col}},
			pgWant: `EXTRACT(YEAR FROM AGE(NOW(), "col"))`,
			slWant: `CAST((JULIANDAY('now') - JULIANDAY("col")) / 365.25 AS INTEGER)`,
		},
		{
			name:   "days_since",
			expr:   map[string]any{"fn": "days_since", "args": []any{col}},
			pgWant: `EXTRACT(DAY FROM (NOW() - "col"))`,
			slWant: `CAST(JULIANDAY('now') - JULIANDAY("col") AS INTEGER)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/postgres", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "postgres", quote)
			if got != tt.pgWant {
				t.Errorf("postgres:\n  got  = %s\n  want = %s", got, tt.pgWant)
			}
		})
		t.Run(tt.name+"/sqlite", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "sqlite", quote)
			if got != tt.slWant {
				t.Errorf("sqlite:\n  got  = %s\n  want = %s", got, tt.slWant)
			}
		})
	}
}

// --- Conditional Functions (docs lines 123-128) ---

func TestDocsVerify_ConditionalFunctions(t *testing.T) {
	colA := map[string]any{"col": "a"}
	colB := map[string]any{"col": "b"}
	colC := map[string]any{"col": "c"}

	tests := []struct {
		name   string
		expr   map[string]any
		pgWant string
		slWant string
	}{
		{
			name:   "coalesce",
			expr:   map[string]any{"fn": "coalesce", "args": []any{colA, colB, colC}},
			pgWant: `COALESCE("a", "b", "c")`,
			slWant: `COALESCE("a", "b", "c")`,
		},
		{
			name:   "nullif",
			expr:   map[string]any{"fn": "nullif", "args": []any{colA, colB}},
			pgWant: `NULLIF("a", "b")`,
			slWant: `NULLIF("a", "b")`,
		},
		{
			name:   "if_null_postgres",
			expr:   map[string]any{"fn": "if_null", "args": []any{colA, colB}},
			pgWant: `COALESCE("a", "b")`,
			slWant: `IFNULL("a", "b")`,
		},
		{
			name: "if_then",
			expr: map[string]any{"fn": "if_then", "args": []any{
				map[string]any{"col": "cond"}, "yes", "no",
			}},
			pgWant: `CASE WHEN "cond" THEN 'yes' ELSE 'no' END`,
			slWant: `CASE WHEN "cond" THEN 'yes' ELSE 'no' END`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/postgres", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "postgres", quote)
			if got != tt.pgWant {
				t.Errorf("postgres:\n  got  = %s\n  want = %s", got, tt.pgWant)
			}
		})
		t.Run(tt.name+"/sqlite", func(t *testing.T) {
			got := computedExprSQL(tt.expr, "sqlite", quote)
			if got != tt.slWant {
				t.Errorf("sqlite:\n  got  = %s\n  want = %s", got, tt.slWant)
			}
		})
	}
}

// --- Comparison Functions (docs lines 147-153) ---

func TestDocsVerify_ComparisonFunctions(t *testing.T) {
	colA := map[string]any{"col": "a"}
	colB := map[string]any{"col": "b"}

	tests := []struct {
		fn   string
		want string
	}{
		{"gt", `("a" > "b")`},
		{"gte", `("a" >= "b")`},
		{"lt", `("a" < "b")`},
		{"lte", `("a" <= "b")`},
		{"eq", `("a" = "b")`},
	}

	for _, tt := range tests {
		expr := map[string]any{"fn": tt.fn, "args": []any{colA, colB}}
		t.Run(tt.fn+"/postgres", func(t *testing.T) {
			got := computedExprSQL(expr, "postgres", quote)
			if got != tt.want {
				t.Errorf("postgres:\n  got  = %s\n  want = %s", got, tt.want)
			}
		})
		t.Run(tt.fn+"/sqlite", func(t *testing.T) {
			got := computedExprSQL(expr, "sqlite", quote)
			if got != tt.want {
				t.Errorf("sqlite:\n  got  = %s\n  want = %s", got, tt.want)
			}
		})
	}
}

// --- Raw SQL (docs lines 170-181) ---

func TestDocsVerify_RawSQL(t *testing.T) {
	expr := map[string]any{
		"fn": "raw",
		"sql": map[string]any{
			"postgres": "price * (1 + tax_rate)",
			"sqlite":   "price * (1.0 + tax_rate)",
		},
	}

	t.Run("postgres", func(t *testing.T) {
		got := computedExprSQL(expr, "postgres", quote)
		want := "price * (1 + tax_rate)"
		if got != want {
			t.Errorf("postgres:\n  got  = %s\n  want = %s", got, want)
		}
	})
	t.Run("sqlite", func(t *testing.T) {
		got := computedExprSQL(expr, "sqlite", quote)
		want := "price * (1.0 + tax_rate)"
		if got != want {
			t.Errorf("sqlite:\n  got  = %s\n  want = %s", got, want)
		}
	})
}

// --- Generated SQL section (docs lines 208-227) ---

func TestDocsVerify_GeneratedSQL_Postgres(t *testing.T) {
	d := Postgres()
	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "first_name", Type: "string", TypeArgs: []any{50}},
			{Name: "last_name", Type: "string", TypeArgs: []any{50}},
			{Name: "full_name", Type: "text", Computed: map[string]any{
				"fn": "concat",
				"args": []any{
					map[string]any{"col": "first_name"},
					" ",
					map[string]any{"col": "last_name"},
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// Exact match of the SQL in the docs
	want := `"full_name" TEXT NOT NULL GENERATED ALWAYS AS (("first_name" || ' ' || "last_name")) STORED`
	if got := sql; !contains(got, want) {
		t.Errorf("postgres CREATE TABLE missing:\n  want = %s\n  got  =\n%s", want, got)
	}
}

func TestDocsVerify_GeneratedSQL_SQLite(t *testing.T) {
	d := SQLite()
	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "first_name", Type: "string", TypeArgs: []any{50}},
			{Name: "last_name", Type: "string", TypeArgs: []any{50}},
			{Name: "full_name", Type: "text", Computed: map[string]any{
				"fn": "concat",
				"args": []any{
					map[string]any{"col": "first_name"},
					" ",
					map[string]any{"col": "last_name"},
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// SQLite uses || for concat
	want := `"full_name" TEXT NOT NULL GENERATED ALWAYS AS (("first_name" || ' ' || "last_name")) STORED`
	if got := sql; !contains(got, want) {
		t.Errorf("sqlite CREATE TABLE missing:\n  want = %s\n  got  =\n%s", want, got)
	}
}

// --- Dialect comparison table (docs lines 39-43) ---

func TestDocsVerify_DialectComparisonTable(t *testing.T) {
	concatExpr := map[string]any{"fn": "concat", "args": []any{"a", "b"}}
	yearExpr := map[string]any{"fn": "year", "args": []any{map[string]any{"col": "col"}}}

	t.Run("postgres/concat", func(t *testing.T) {
		got := computedExprSQL(concatExpr, "postgres", quote)
		if got != "('a' || 'b')" {
			t.Errorf("got = %s", got)
		}
	})
	t.Run("sqlite/concat", func(t *testing.T) {
		got := computedExprSQL(concatExpr, "sqlite", quote)
		if got != "('a' || 'b')" {
			t.Errorf("got = %s", got)
		}
	})
	t.Run("postgres/year", func(t *testing.T) {
		got := computedExprSQL(yearExpr, "postgres", quote)
		if got != `EXTRACT(YEAR FROM "col")` {
			t.Errorf("got = %s", got)
		}
	})
	t.Run("sqlite/year", func(t *testing.T) {
		got := computedExprSQL(yearExpr, "sqlite", quote)
		if got != `CAST(STRFTIME('%Y', "col") AS INTEGER)` {
			t.Errorf("got = %s", got)
		}
	})
	t.Run("cockroachdb/concat", func(t *testing.T) {
		// CockroachDB uses postgres dialect
		got := computedExprSQL(concatExpr, "postgres", quote)
		if got != "('a' || 'b')" {
			t.Errorf("got = %s", got)
		}
	})
	t.Run("cockroachdb/year", func(t *testing.T) {
		got := computedExprSQL(yearExpr, "postgres", quote)
		if got != `EXTRACT(YEAR FROM "col")` {
			t.Errorf("got = %s", got)
		}
	})
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 ||
		findSubstring(haystack, needle))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
