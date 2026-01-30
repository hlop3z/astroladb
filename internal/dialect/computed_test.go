package dialect

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// =============================================================================
// Computed Column SQL Generation - PostgreSQL
// =============================================================================

func TestPostgresComputedColumn_StringConcat(t *testing.T) {
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
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	contains := []string{
		`"full_name" TEXT NOT NULL GENERATED ALWAYS AS (("first_name" || ' ' || "last_name")) STORED`,
	}

	for _, want := range contains {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
		}
	}
}

func TestPostgresComputedColumn_NumericMath(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		col      *ast.ColumnDef
		contains string
	}{
		{
			name: "integer_add",
			col: &ast.ColumnDef{Name: "total", Type: "integer", Computed: map[string]any{
				"fn": "add",
				"args": []any{
					map[string]any{"col": "qty"},
					map[string]any{"col": "bonus"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("qty" + "bonus")) STORED`,
		},
		{
			name: "float_mul",
			col: &ast.ColumnDef{Name: "area", Type: "float", Computed: map[string]any{
				"fn": "mul",
				"args": []any{
					map[string]any{"col": "width"},
					map[string]any{"col": "height"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("width" * "height")) STORED`,
		},
		{
			name: "decimal_div",
			col: &ast.ColumnDef{Name: "avg_price", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "div",
				"args": []any{
					map[string]any{"col": "total_price"},
					map[string]any{"col": "qty"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("total_price" / "qty")) STORED`,
		},
		{
			name: "sub",
			col: &ast.ColumnDef{Name: "profit", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "sub",
				"args": []any{
					map[string]any{"col": "revenue"},
					map[string]any{"col": "cost"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("revenue" - "cost")) STORED`,
		},
		{
			name: "abs",
			col: &ast.ColumnDef{Name: "magnitude", Type: "float", Computed: map[string]any{
				"fn":   "abs",
				"args": []any{map[string]any{"col": "value"}},
			}},
			contains: `GENERATED ALWAYS AS (ABS("value")) STORED`,
		},
		{
			name: "round",
			col: &ast.ColumnDef{Name: "rounded", Type: "integer", Computed: map[string]any{
				"fn":   "round",
				"args": []any{map[string]any{"col": "score"}},
			}},
			contains: `GENERATED ALWAYS AS (ROUND("score")) STORED`,
		},
		{
			name: "floor",
			col: &ast.ColumnDef{Name: "floored", Type: "integer", Computed: map[string]any{
				"fn":   "floor",
				"args": []any{map[string]any{"col": "price"}},
			}},
			contains: `GENERATED ALWAYS AS (FLOOR("price")) STORED`,
		},
		{
			name: "ceil",
			col: &ast.ColumnDef{Name: "ceiling", Type: "integer", Computed: map[string]any{
				"fn":   "ceil",
				"args": []any{map[string]any{"col": "price"}},
			}},
			contains: `GENERATED ALWAYS AS (CEIL("price")) STORED`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "shop", Name: tt.name},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					tt.col,
				},
			}
			sql, err := d.CreateTableSQL(op)
			if err != nil {
				t.Fatalf("CreateTableSQL() error = %v", err)
			}
			if !strings.Contains(sql, tt.contains) {
				t.Errorf("SQL missing %q\nGot:\n%s", tt.contains, sql)
			}
		})
	}
}

func TestPostgresComputedColumn_NestedMath(t *testing.T) {
	d := Postgres()

	// total = round(qty * (unit_price - discount))
	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "order", Name: "line"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "qty", Type: "integer"},
			{Name: "unit_price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "discount", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "total", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "round",
				"args": []any{
					map[string]any{
						"fn": "mul",
						"args": []any{
							map[string]any{"col": "qty"},
							map[string]any{
								"fn": "sub",
								"args": []any{
									map[string]any{"col": "unit_price"},
									map[string]any{"col": "discount"},
								},
							},
						},
					},
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (ROUND(("qty" * ("unit_price" - "discount")))) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestPostgresComputedColumn_BooleanComparison(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "exam", Name: "result"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "score", Type: "float"},
			{Name: "passing", Type: "boolean", Computed: map[string]any{
				"fn": "gte",
				"args": []any{
					map[string]any{"col": "score"},
					float64(50),
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (("score" >= 50)) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestPostgresComputedColumn_Conditional(t *testing.T) {
	d := Postgres()

	// grade = CASE WHEN score >= 90 THEN 'A' ELSE 'B' END
	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "exam", Name: "grades"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "score", Type: "float"},
			{Name: "grade", Type: "text", Computed: map[string]any{
				"fn": "if_then",
				"args": []any{
					map[string]any{
						"fn": "gte",
						"args": []any{
							map[string]any{"col": "score"},
							float64(90),
						},
					},
					"A",
					"B",
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (CASE WHEN ("score" >= 90) THEN 'A' ELSE 'B' END) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestPostgresComputedColumn_DateFunctions(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "hr", Name: "employees"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "birthdate", Type: "date"},
			{Name: "age", Type: "integer", Computed: map[string]any{
				"fn":   "years_since",
				"args": []any{map[string]any{"col": "birthdate"}},
			}},
			{Name: "birth_year", Type: "integer", Computed: map[string]any{
				"fn":   "year",
				"args": []any{map[string]any{"col": "birthdate"}},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	contains := []string{
		`GENERATED ALWAYS AS (EXTRACT(YEAR FROM AGE(NOW(), "birthdate"))) STORED`,
		`GENERATED ALWAYS AS (EXTRACT(YEAR FROM "birthdate")) STORED`,
	}

	for _, want := range contains {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
		}
	}
}

func TestPostgresComputedColumn_RawSQL(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "shop", Name: "products"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "tax_rate", Type: "decimal", TypeArgs: []any{5, 4}},
			{Name: "total", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "raw",
				"sql": map[string]any{
					"postgres": "price * (1 + tax_rate)",
					"sqlite":   "price * (1 + tax_rate)",
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (price * (1 + tax_rate)) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestPostgresComputedColumn_Coalesce(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "nickname", Type: "string", TypeArgs: []any{50}, Nullable: true, NullableSet: true},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
			{Name: "display_name", Type: "text", Computed: map[string]any{
				"fn": "coalesce",
				"args": []any{
					map[string]any{"col": "nickname"},
					map[string]any{"col": "username"},
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (COALESCE("nickname", "username")) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

// =============================================================================
// Computed Column SQL Generation - SQLite
// =============================================================================

func TestSQLiteComputedColumn_StringConcat(t *testing.T) {
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
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	// SQLite uses || for concatenation
	want := `GENERATED ALWAYS AS (("first_name" || ' ' || "last_name")) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestSQLiteComputedColumn_NumericMath(t *testing.T) {
	d := SQLite()

	tests := []struct {
		name     string
		col      *ast.ColumnDef
		contains string
	}{
		{
			name: "integer_add",
			col: &ast.ColumnDef{Name: "total", Type: "integer", Computed: map[string]any{
				"fn": "add",
				"args": []any{
					map[string]any{"col": "qty"},
					map[string]any{"col": "bonus"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("qty" + "bonus")) STORED`,
		},
		{
			name: "float_mul",
			col: &ast.ColumnDef{Name: "area", Type: "float", Computed: map[string]any{
				"fn": "mul",
				"args": []any{
					map[string]any{"col": "width"},
					map[string]any{"col": "height"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("width" * "height")) STORED`,
		},
		{
			name: "decimal_div",
			col: &ast.ColumnDef{Name: "avg_price", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "div",
				"args": []any{
					map[string]any{"col": "total_price"},
					map[string]any{"col": "qty"},
				},
			}},
			contains: `GENERATED ALWAYS AS (("total_price" / "qty")) STORED`,
		},
		{
			name: "round",
			col: &ast.ColumnDef{Name: "rounded", Type: "integer", Computed: map[string]any{
				"fn":   "round",
				"args": []any{map[string]any{"col": "score"}},
			}},
			contains: `GENERATED ALWAYS AS (ROUND("score")) STORED`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "shop", Name: tt.name},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					tt.col,
				},
			}
			sql, err := d.CreateTableSQL(op)
			if err != nil {
				t.Fatalf("CreateTableSQL() error = %v", err)
			}
			if !strings.Contains(sql, tt.contains) {
				t.Errorf("SQL missing %q\nGot:\n%s", tt.contains, sql)
			}
		})
	}
}

func TestSQLiteComputedColumn_DateFunctions(t *testing.T) {
	d := SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "hr", Name: "employees"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "birthdate", Type: "date"},
			{Name: "age", Type: "integer", Computed: map[string]any{
				"fn":   "years_since",
				"args": []any{map[string]any{"col": "birthdate"}},
			}},
			{Name: "birth_year", Type: "integer", Computed: map[string]any{
				"fn":   "year",
				"args": []any{map[string]any{"col": "birthdate"}},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	contains := []string{
		// SQLite uses JULIANDAY for years_since
		`GENERATED ALWAYS AS (CAST((JULIANDAY('now') - JULIANDAY("birthdate")) / 365.25 AS INTEGER)) STORED`,
		// SQLite uses STRFTIME for year extraction
		`GENERATED ALWAYS AS (CAST(STRFTIME('%Y', "birthdate") AS INTEGER)) STORED`,
	}

	for _, want := range contains {
		if !strings.Contains(sql, want) {
			t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
		}
	}
}

func TestSQLiteComputedColumn_BooleanComparison(t *testing.T) {
	d := SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "exam", Name: "result"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "score", Type: "float"},
			{Name: "passing", Type: "boolean", Computed: map[string]any{
				"fn": "gte",
				"args": []any{
					map[string]any{"col": "score"},
					float64(50),
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (("score" >= 50)) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestSQLiteComputedColumn_RawSQL(t *testing.T) {
	d := SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "shop", Name: "products"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "tax_rate", Type: "decimal", TypeArgs: []any{5, 4}},
			{Name: "total", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "raw",
				"sql": map[string]any{
					"postgres": "price * (1 + tax_rate)",
					"sqlite":   "price * (1.0 + tax_rate)",
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (price * (1.0 + tax_rate)) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

// =============================================================================
// CockroachDB uses PostgreSQL dialect — verify it produces the same SQL
// =============================================================================

func TestCockroachDBComputedColumn_UsesPostgresDialect(t *testing.T) {
	// CockroachDB uses the Postgres dialect
	d := Get("postgres")
	if d == nil {
		t.Fatal("postgres dialect not found")
	}

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "shop", Name: "orders"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "qty", Type: "integer"},
			{Name: "unit_price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "total", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "mul",
				"args": []any{
					map[string]any{"col": "qty"},
					map[string]any{"col": "unit_price"},
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	// CockroachDB supports the same GENERATED ALWAYS AS syntax as PostgreSQL
	want := `GENERATED ALWAYS AS (("qty" * "unit_price")) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestCockroachDBComputedColumn_NumericWithLiterals(t *testing.T) {
	d := Get("postgres") // CockroachDB uses postgres dialect

	// tax_total = price * 1.08 (8% tax)
	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "shop", Name: "items"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "tax_total", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
				"fn": "mul",
				"args": []any{
					map[string]any{"col": "price"},
					1.08,
				},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (("price" * 1.08)) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

// =============================================================================
// Dialect Differences - Same Expression, Different SQL
// =============================================================================

func TestComputedColumn_DialectDifferences(t *testing.T) {
	pg := Postgres()
	sl := SQLite()

	// Same computed column expression
	computedConcat := map[string]any{
		"fn": "concat",
		"args": []any{
			map[string]any{"col": "first_name"},
			" ",
			map[string]any{"col": "last_name"},
		},
	}

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "people"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "first_name", Type: "string", TypeArgs: []any{50}},
			{Name: "last_name", Type: "string", TypeArgs: []any{50}},
			{Name: "full_name", Type: "text", Computed: computedConcat},
		},
	}

	pgSQL, err := pg.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("Postgres CreateTableSQL error = %v", err)
	}

	slSQL, err := sl.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("SQLite CreateTableSQL error = %v", err)
	}

	// Both dialects use || operator for concat (CONCAT is not immutable in PostgreSQL)
	if !strings.Contains(pgSQL, " || ") {
		t.Errorf("PostgreSQL should use || operator\nGot:\n%s", pgSQL)
	}
	if !strings.Contains(slSQL, " || ") {
		t.Errorf("SQLite should use || operator\nGot:\n%s", slSQL)
	}

	// Both should have GENERATED ALWAYS AS ... STORED
	for _, sql := range []string{pgSQL, slSQL} {
		if !strings.Contains(sql, "GENERATED ALWAYS AS") {
			t.Errorf("Missing GENERATED ALWAYS AS in:\n%s", sql)
		}
		if !strings.Contains(sql, "STORED") {
			t.Errorf("Missing STORED in:\n%s", sql)
		}
	}
}

func TestComputedColumn_DateDialectDifferences(t *testing.T) {
	pg := Postgres()
	sl := SQLite()

	computedYear := map[string]any{
		"fn":   "year",
		"args": []any{map[string]any{"col": "created_at"}},
	}

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "log", Name: "events"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "created_at", Type: "datetime"},
			{Name: "event_year", Type: "integer", Computed: computedYear},
		},
	}

	pgSQL, err := pg.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("Postgres error = %v", err)
	}

	slSQL, err := sl.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("SQLite error = %v", err)
	}

	// PostgreSQL uses EXTRACT
	if !strings.Contains(pgSQL, "EXTRACT(YEAR FROM") {
		t.Errorf("PostgreSQL should use EXTRACT\nGot:\n%s", pgSQL)
	}

	// SQLite uses STRFTIME
	if !strings.Contains(slSQL, "STRFTIME('%Y'") {
		t.Errorf("SQLite should use STRFTIME\nGot:\n%s", slSQL)
	}
}

// =============================================================================
// AddColumn with Computed
// =============================================================================

func TestPostgresAddColumn_Computed(t *testing.T) {
	d := Postgres()

	op := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "shop", Table_: "products"},
		Column: &ast.ColumnDef{Name: "margin", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
			"fn": "sub",
			"args": []any{
				map[string]any{"col": "price"},
				map[string]any{"col": "cost"},
			},
		}},
	}

	sql, err := d.AddColumnSQL(op)
	if err != nil {
		t.Fatalf("AddColumnSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (("price" - "cost")) STORED`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot: %s", want, sql)
	}
}

func TestSQLiteAddColumn_Computed_ReturnsError(t *testing.T) {
	d := SQLite()

	op := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "shop", Table_: "products"},
		Column: &ast.ColumnDef{Name: "margin", Type: "decimal", TypeArgs: []any{10, 2}, Computed: map[string]any{
			"fn": "sub",
			"args": []any{
				map[string]any{"col": "price"},
				map[string]any{"col": "cost"},
			},
		}},
	}

	// SQLite cannot ALTER TABLE ADD COLUMN with STORED generated columns.
	_, err := d.AddColumnSQL(op)
	if err == nil {
		t.Fatal("expected error for SQLite ADD COLUMN with STORED computed column")
	}
	if !strings.Contains(err.Error(), "STORED generated column") {
		t.Errorf("error message should mention STORED generated column, got: %v", err)
	}
}

// =============================================================================
// All Comparison Operators with Numbers
// =============================================================================

func TestPostgresComputedColumn_AllComparisonOps(t *testing.T) {
	d := Postgres()

	tests := []struct {
		fn string
		op string
	}{
		{"gt", ">"},
		{"gte", ">="},
		{"lt", "<"},
		{"lte", "<="},
		{"eq", "="},
	}

	for _, tt := range tests {
		t.Run(tt.fn, func(t *testing.T) {
			col := &ast.ColumnDef{Name: "result", Type: "boolean", Computed: map[string]any{
				"fn": tt.fn,
				"args": []any{
					map[string]any{"col": "score"},
					float64(100),
				},
			}}

			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "test", Name: "cmp_" + tt.fn},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "score", Type: "float"},
					col,
				},
			}

			sql, err := d.CreateTableSQL(op)
			if err != nil {
				t.Fatalf("error = %v", err)
			}

			want := `("score" ` + tt.op + ` 100)`
			if !strings.Contains(sql, want) {
				t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
			}
		})
	}
}

// =============================================================================
// String functions across dialects
// =============================================================================

func TestPostgresComputedColumn_StringFunctions(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		computed map[string]any
		contains string
	}{
		{
			name: "upper",
			computed: map[string]any{
				"fn":   "upper",
				"args": []any{map[string]any{"col": "name"}},
			},
			contains: `UPPER("name")`,
		},
		{
			name: "lower",
			computed: map[string]any{
				"fn":   "lower",
				"args": []any{map[string]any{"col": "name"}},
			},
			contains: `LOWER("name")`,
		},
		{
			name: "trim",
			computed: map[string]any{
				"fn":   "trim",
				"args": []any{map[string]any{"col": "name"}},
			},
			contains: `TRIM("name")`,
		},
		{
			name: "length",
			computed: map[string]any{
				"fn":   "length",
				"args": []any{map[string]any{"col": "name"}},
			},
			contains: `LENGTH("name")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "test", Name: "str_" + tt.name},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "name", Type: "string", TypeArgs: []any{100}},
					{Name: "result", Type: "text", Computed: tt.computed},
				},
			}
			sql, err := d.CreateTableSQL(op)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !strings.Contains(sql, tt.contains) {
				t.Errorf("SQL missing %q\nGot:\n%s", tt.contains, sql)
			}
		})
	}
}

// =============================================================================
// Non-computed columns should NOT have GENERATED ALWAYS AS
// =============================================================================

// =============================================================================
// VIRTUAL keyword (instead of STORED)
// =============================================================================

func TestPostgresComputedColumn_Virtual(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "shop", Name: "items"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "qty", Type: "integer"},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "total", Type: "decimal", TypeArgs: []any{10, 2}, Virtual: true, Computed: map[string]any{
				"fn":   "mul",
				"args": []any{map[string]any{"col": "qty"}, map[string]any{"col": "price"}},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (("qty" * "price")) VIRTUAL`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
	if strings.Contains(sql, "STORED") {
		t.Errorf("VIRTUAL column should NOT contain STORED\nGot:\n%s", sql)
	}
}

func TestSQLiteComputedColumn_Virtual(t *testing.T) {
	d := SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "shop", Name: "items"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "name_upper", Type: "text", Virtual: true, Computed: map[string]any{
				"fn":   "upper",
				"args": []any{map[string]any{"col": "name"}},
			}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	want := `GENERATED ALWAYS AS (UPPER("name")) VIRTUAL`
	if !strings.Contains(sql, want) {
		t.Errorf("SQL missing %q\nGot:\n%s", want, sql)
	}
}

func TestSQLiteAddColumn_VirtualComputed_Allowed(t *testing.T) {
	d := SQLite()

	op := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "shop", Table_: "products"},
		Column: &ast.ColumnDef{Name: "name_upper", Type: "text", Virtual: true, Computed: map[string]any{
			"fn":   "upper",
			"args": []any{map[string]any{"col": "name"}},
		}},
	}

	// VIRTUAL computed columns CAN be added via ALTER TABLE in SQLite
	_, err := d.AddColumnSQL(op)
	if err != nil {
		t.Fatalf("expected no error for SQLite ADD COLUMN with VIRTUAL computed, got: %v", err)
	}
}

// =============================================================================
// App-only virtual columns (no DB column)
// =============================================================================

func TestPostgresAppOnlyVirtual_SkippedInSQL(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "display_label", Type: "text", Virtual: true}, // app-only
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	if strings.Contains(sql, "display_label") {
		t.Errorf("App-only virtual column should NOT appear in SQL\nGot:\n%s", sql)
	}
}

func TestSQLiteAppOnlyVirtual_SkippedInSQL(t *testing.T) {
	d := SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "display_label", Type: "text", Virtual: true}, // app-only
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	if strings.Contains(sql, "display_label") {
		t.Errorf("App-only virtual column should NOT appear in SQL\nGot:\n%s", sql)
	}
}

// =============================================================================
// Non-computed columns should NOT have GENERATED ALWAYS AS
// =============================================================================

func TestPostgresNonComputedColumn_NoGeneratedClause(t *testing.T) {
	d := Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "score", Type: "float"},
			{Name: "balance", Type: "decimal", TypeArgs: []any{10, 2}, Default: 0, DefaultSet: true},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	if strings.Contains(sql, "GENERATED ALWAYS AS") {
		t.Errorf("Non-computed table should NOT contain GENERATED ALWAYS AS\nGot:\n%s", sql)
	}
}
