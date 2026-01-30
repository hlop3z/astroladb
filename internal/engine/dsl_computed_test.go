//go:build integration

package engine_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/dsl"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// addTextPK adds a simple TEXT primary key (no UUID, works on all dialects).
func addTextPK(tb *dsl.TableBuilder) {
	col := dsl.NewColumnBuilder("id", "text")
	col.PrimaryKey()
	tb.AddColumn(col.Build())
}

// ---------------------------------------------------------------------------
// Computed Columns — integration tests against real databases
// ---------------------------------------------------------------------------

func TestDSL_ComputedColumn_StringConcat(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_concat", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					tb.String("first_name", 50)
					tb.String("last_name", 50)
					cb := dsl.NewColumnBuilder("full_name", "text")
					cb.Computed(map[string]any{
						"fn": "concat",
						"args": []any{
							map[string]any{"col": "first_name"},
							" ",
							map[string]any{"col": "last_name"},
						},
					})
					tb.AddColumn(cb.Build())
				})
			})
			execOps(t, d.db, d.dialect, ops)
			testutil.AssertTableExists(t, d.db, "test_comp_concat")
			testutil.AssertColumnExists(t, d.db, "test_comp_concat", "full_name")

			_, err := d.db.Exec(`INSERT INTO "test_comp_concat" ("id", "first_name", "last_name") VALUES ('1', 'John', 'Doe')`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			var fullName string
			err = d.db.QueryRow(`SELECT "full_name" FROM "test_comp_concat" WHERE "id" = '1'`).Scan(&fullName)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if fullName != "John Doe" {
				t.Errorf("full_name = %q, want %q", fullName, "John Doe")
			}
		})
	}
}

func TestDSL_ComputedColumn_MathAdd(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_math", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					tb.Integer("a")
					tb.Integer("b")
					cb := dsl.NewColumnBuilder("total", "integer")
					cb.Computed(map[string]any{
						"fn": "add",
						"args": []any{
							map[string]any{"col": "a"},
							map[string]any{"col": "b"},
						},
					})
					tb.AddColumn(cb.Build())
				})
			})
			execOps(t, d.db, d.dialect, ops)
			testutil.AssertColumnExists(t, d.db, "test_comp_math", "total")

			_, err := d.db.Exec(`INSERT INTO "test_comp_math" ("id", "a", "b") VALUES ('1', 10, 20)`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			var total int
			err = d.db.QueryRow(`SELECT "total" FROM "test_comp_math" WHERE "id" = '1'`).Scan(&total)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if total != 30 {
				t.Errorf("total = %d, want 30", total)
			}
		})
	}
}

func TestDSL_ComputedColumn_Upper(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_upper", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					tb.String("name", 100)
					cb := dsl.NewColumnBuilder("name_upper", "text")
					cb.Computed(map[string]any{
						"fn":   "upper",
						"args": []any{map[string]any{"col": "name"}},
					})
					tb.AddColumn(cb.Build())
				})
			})
			execOps(t, d.db, d.dialect, ops)

			_, err := d.db.Exec(`INSERT INTO "test_comp_upper" ("id", "name") VALUES ('1', 'hello')`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			var val string
			err = d.db.QueryRow(`SELECT "name_upper" FROM "test_comp_upper" WHERE "id" = '1'`).Scan(&val)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if val != "HELLO" {
				t.Errorf("name_upper = %q, want %q", val, "HELLO")
			}
		})
	}
}

func TestDSL_ComputedColumn_BooleanComparison(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_bool", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					tb.Integer("age")
					cb := dsl.NewColumnBuilder("is_adult", "boolean")
					cb.Computed(map[string]any{
						"fn": "gte",
						"args": []any{
							map[string]any{"col": "age"},
							float64(18),
						},
					})
					tb.AddColumn(cb.Build())
				})
			})
			execOps(t, d.db, d.dialect, ops)

			_, err := d.db.Exec(`INSERT INTO "test_comp_bool" ("id", "age") VALUES ('1', 25)`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			_, err = d.db.Exec(`INSERT INTO "test_comp_bool" ("id", "age") VALUES ('2', 10)`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}

			var adult, minor bool
			err = d.db.QueryRow(`SELECT "is_adult" FROM "test_comp_bool" WHERE "id" = '1'`).Scan(&adult)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if !adult {
				t.Error("is_adult should be true for age=25")
			}

			err = d.db.QueryRow(`SELECT "is_adult" FROM "test_comp_bool" WHERE "id" = '2'`).Scan(&minor)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if minor {
				t.Error("is_adult should be false for age=10")
			}
		})
	}
}

func TestDSL_ComputedColumn_Coalesce(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_coal", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					cb1 := dsl.NewColumnBuilder("nickname", "text")
					cb1.Nullable()
					tb.AddColumn(cb1.Build())
					tb.String("username", 100)
					cb := dsl.NewColumnBuilder("display_name", "text")
					cb.Computed(map[string]any{
						"fn": "coalesce",
						"args": []any{
							map[string]any{"col": "nickname"},
							map[string]any{"col": "username"},
						},
					})
					tb.AddColumn(cb.Build())
				})
			})
			execOps(t, d.db, d.dialect, ops)

			_, err := d.db.Exec(`INSERT INTO "test_comp_coal" ("id", "nickname", "username") VALUES ('1', NULL, 'jdoe')`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			var val string
			err = d.db.QueryRow(`SELECT "display_name" FROM "test_comp_coal" WHERE "id" = '1'`).Scan(&val)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if val != "jdoe" {
				t.Errorf("display_name = %q, want %q", val, "jdoe")
			}

			_, err = d.db.Exec(`INSERT INTO "test_comp_coal" ("id", "nickname", "username") VALUES ('2', 'Johnny', 'jdoe')`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			err = d.db.QueryRow(`SELECT "display_name" FROM "test_comp_coal" WHERE "id" = '2'`).Scan(&val)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if val != "Johnny" {
				t.Errorf("display_name = %q, want %q", val, "Johnny")
			}
		})
	}
}

func TestDSL_ComputedColumn_Virtual(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_virt", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					tb.String("name", 100)
					cb := dsl.NewColumnBuilder("name_upper", "text")
					cb.Computed(map[string]any{
						"fn":   "upper",
						"args": []any{map[string]any{"col": "name"}},
					})
					cb.Virtual()
					tb.AddColumn(cb.Build())
				})
			})

			// SQLite and CockroachDB support VIRTUAL; PostgreSQL < 18 does not.
			// We attempt execution and skip on dialect-specific errors.
			err := tryExecOps(t, d.db, d.dialect, ops)
			if err != nil {
				t.Skipf("skipping VIRTUAL test on %s: %v", d.name, err)
			}

			testutil.AssertTableExists(t, d.db, "test_comp_virt")

			_, err = d.db.Exec(`INSERT INTO "test_comp_virt" ("id", "name") VALUES ('1', 'hello')`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			var val string
			err = d.db.QueryRow(`SELECT "name_upper" FROM "test_comp_virt" WHERE "id" = '1'`).Scan(&val)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if val != "HELLO" {
				t.Errorf("name_upper = %q, want %q", val, "HELLO")
			}
		})
	}
}

func TestDSL_ComputedColumn_NestedMath(t *testing.T) {
	for _, d := range setupDialects(t) {
		t.Run(d.name, func(t *testing.T) {
			ops := buildOps(func(m *dsl.MigrationBuilder) {
				m.CreateTable("test.comp_nested", func(tb *dsl.TableBuilder) {
					addTextPK(tb)
					tb.Integer("qty")
					tb.Decimal("unit_price", 10, 2)
					tb.Decimal("discount", 10, 2)
					cb := dsl.NewColumnBuilder("total", "decimal", 10, 2)
					cb.Computed(map[string]any{
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
					})
					tb.AddColumn(cb.Build())
				})
			})
			execOps(t, d.db, d.dialect, ops)

			_, err := d.db.Exec(`INSERT INTO "test_comp_nested" ("id", "qty", "unit_price", "discount") VALUES ('1', 3, 10.50, 0.50)`)
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			var total float64
			err = d.db.QueryRow(`SELECT "total" FROM "test_comp_nested" WHERE "id" = '1'`).Scan(&total)
			if err != nil {
				t.Fatalf("select failed: %v", err)
			}
			if total != 30 {
				t.Errorf("total = %v, want 30", total)
			}
		})
	}
}
