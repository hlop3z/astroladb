package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// tablesEqual Tests
// -----------------------------------------------------------------------------

func TestTablesEqual(t *testing.T) {
	t.Run("both_nil", func(t *testing.T) {
		if !tablesEqual(nil, nil) {
			t.Error("tablesEqual(nil, nil) should be true")
		}
	})

	t.Run("both_empty", func(t *testing.T) {
		if !tablesEqual([]*ast.TableDef{}, []*ast.TableDef{}) {
			t.Error("tablesEqual with empty slices should be true")
		}
	})

	t.Run("different_lengths", func(t *testing.T) {
		t1 := []*ast.TableDef{{Name: "a"}}
		t2 := []*ast.TableDef{{Name: "a"}, {Name: "b"}}
		if tablesEqual(t1, t2) {
			t.Error("tablesEqual with different lengths should be false")
		}
	})

	t.Run("same_tables", func(t *testing.T) {
		t1 := []*ast.TableDef{
			{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}},
		}
		t2 := []*ast.TableDef{
			{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}},
		}
		if !tablesEqual(t1, t2) {
			t.Error("tablesEqual with identical tables should be true")
		}
	})

	t.Run("different_tables", func(t *testing.T) {
		t1 := []*ast.TableDef{{Namespace: "auth", Name: "users"}}
		t2 := []*ast.TableDef{{Namespace: "auth", Name: "roles"}}
		if tablesEqual(t1, t2) {
			t.Error("tablesEqual with different tables should be false")
		}
	})
}

// -----------------------------------------------------------------------------
// tableDefEqual Tests
// -----------------------------------------------------------------------------

func TestTableDefEqual(t *testing.T) {
	t.Run("both_nil", func(t *testing.T) {
		if !tableDefEqual(nil, nil) {
			t.Error("tableDefEqual(nil, nil) should be true")
		}
	})

	t.Run("one_nil", func(t *testing.T) {
		table := &ast.TableDef{Name: "users"}
		if tableDefEqual(nil, table) {
			t.Error("tableDefEqual(nil, table) should be false")
		}
		if tableDefEqual(table, nil) {
			t.Error("tableDefEqual(table, nil) should be false")
		}
	})

	t.Run("different_namespace", func(t *testing.T) {
		t1 := &ast.TableDef{Namespace: "auth", Name: "users"}
		t2 := &ast.TableDef{Namespace: "blog", Name: "users"}
		if tableDefEqual(t1, t2) {
			t.Error("tables with different namespaces should not be equal")
		}
	})

	t.Run("different_name", func(t *testing.T) {
		t1 := &ast.TableDef{Namespace: "auth", Name: "users"}
		t2 := &ast.TableDef{Namespace: "auth", Name: "roles"}
		if tableDefEqual(t1, t2) {
			t.Error("tables with different names should not be equal")
		}
	})

	t.Run("different_column_count", func(t *testing.T) {
		t1 := &ast.TableDef{
			Name:    "users",
			Columns: []*ast.ColumnDef{{Name: "id"}},
		}
		t2 := &ast.TableDef{
			Name:    "users",
			Columns: []*ast.ColumnDef{{Name: "id"}, {Name: "email"}},
		}
		if tableDefEqual(t1, t2) {
			t.Error("tables with different column counts should not be equal")
		}
	})

	t.Run("different_index_count", func(t *testing.T) {
		t1 := &ast.TableDef{
			Name:    "users",
			Indexes: []*ast.IndexDef{{Columns: []string{"id"}}},
		}
		t2 := &ast.TableDef{
			Name:    "users",
			Indexes: []*ast.IndexDef{},
		}
		if tableDefEqual(t1, t2) {
			t.Error("tables with different index counts should not be equal")
		}
	})

	t.Run("equal_tables", func(t *testing.T) {
		col := &ast.ColumnDef{Name: "id", Type: "uuid", Nullable: false}
		idx := &ast.IndexDef{Columns: []string{"id"}, Unique: true}
		t1 := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{col},
			Indexes:   []*ast.IndexDef{idx},
		}
		t2 := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", Nullable: false}},
			Indexes:   []*ast.IndexDef{{Columns: []string{"id"}, Unique: true}},
		}
		if !tableDefEqual(t1, t2) {
			t.Error("identical tables should be equal")
		}
	})
}

// -----------------------------------------------------------------------------
// columnDefEqual Tests
// -----------------------------------------------------------------------------

func TestColumnDefEqual(t *testing.T) {
	t.Run("both_nil", func(t *testing.T) {
		if !columnDefEqual(nil, nil) {
			t.Error("columnDefEqual(nil, nil) should be true")
		}
	})

	t.Run("one_nil", func(t *testing.T) {
		col := &ast.ColumnDef{Name: "id"}
		if columnDefEqual(nil, col) {
			t.Error("columnDefEqual(nil, col) should be false")
		}
		if columnDefEqual(col, nil) {
			t.Error("columnDefEqual(col, nil) should be false")
		}
	})

	t.Run("different_name", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "id", Type: "uuid"}
		c2 := &ast.ColumnDef{Name: "user_id", Type: "uuid"}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different names should not be equal")
		}
	})

	t.Run("different_type", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "id", Type: "uuid"}
		c2 := &ast.ColumnDef{Name: "id", Type: "integer"}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different types should not be equal")
		}
	})

	t.Run("different_nullable", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "email", Type: "string", Nullable: true}
		c2 := &ast.ColumnDef{Name: "email", Type: "string", Nullable: false}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different nullable should not be equal")
		}
	})

	t.Run("different_unique", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "email", Type: "string", Unique: true}
		c2 := &ast.ColumnDef{Name: "email", Type: "string", Unique: false}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different unique should not be equal")
		}
	})

	t.Run("different_primary_key", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "id", Type: "uuid", PrimaryKey: true}
		c2 := &ast.ColumnDef{Name: "id", Type: "uuid", PrimaryKey: false}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different primary_key should not be equal")
		}
	})

	t.Run("different_type_args", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "name", Type: "string", TypeArgs: []any{100}}
		c2 := &ast.ColumnDef{Name: "name", Type: "string", TypeArgs: []any{255}}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different type_args should not be equal")
		}
	})

	t.Run("different_default", func(t *testing.T) {
		c1 := &ast.ColumnDef{Name: "status", Type: "string", Default: "active"}
		c2 := &ast.ColumnDef{Name: "status", Type: "string", Default: "inactive"}
		if columnDefEqual(c1, c2) {
			t.Error("columns with different default should not be equal")
		}
	})

	t.Run("equal_columns", func(t *testing.T) {
		c1 := &ast.ColumnDef{
			Name:       "email",
			Type:       "string",
			TypeArgs:   []any{255},
			Nullable:   false,
			Unique:     true,
			PrimaryKey: false,
			Default:    nil,
		}
		c2 := &ast.ColumnDef{
			Name:       "email",
			Type:       "string",
			TypeArgs:   []any{255},
			Nullable:   false,
			Unique:     true,
			PrimaryKey: false,
			Default:    nil,
		}
		if !columnDefEqual(c1, c2) {
			t.Error("identical columns should be equal")
		}
	})
}

// -----------------------------------------------------------------------------
// ValidateDeterminism Tests
// -----------------------------------------------------------------------------

func TestSandbox_ValidateDeterminism(t *testing.T) {
	t.Run("deterministic_code", func(t *testing.T) {
		sb := NewSandbox(nil)
		// Simple deterministic code without bindings
		err := sb.ValidateDeterminism(`
			var x = 1 + 1;
			var y = [1, 2, 3];
		`)
		if err != nil {
			t.Errorf("deterministic code should pass: %v", err)
		}
	})

	t.Run("syntax_error_fails", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.ValidateDeterminism("invalid javascript {{{{")
		if err == nil {
			t.Error("invalid code should fail")
		}
	})
}

// -----------------------------------------------------------------------------
// ValidateDeterminismStrict Tests
// -----------------------------------------------------------------------------

func TestSandbox_ValidateDeterminismStrict(t *testing.T) {
	t.Run("iterations_minimum", func(t *testing.T) {
		sb := NewSandbox(nil)
		// With iterations < 2, it should default to 2
		err := sb.ValidateDeterminismStrict(`
			var x = 1 + 1;
		`, 1)
		if err != nil {
			t.Errorf("deterministic code should pass: %v", err)
		}
	})

	t.Run("multiple_iterations", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.ValidateDeterminismStrict(`
			var items = [1, 2, 3];
			var sum = items.reduce(function(a, b) { return a + b; }, 0);
		`, 5)
		if err != nil {
			t.Errorf("deterministic code with 5 iterations should pass: %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// ValidateIdentifiers Tests
// -----------------------------------------------------------------------------

func TestValidateIdentifiers(t *testing.T) {
	t.Run("empty_operations", func(t *testing.T) {
		err := ValidateIdentifiers(nil)
		if err != nil {
			t.Errorf("empty operations should pass: %v", err)
		}
	})

	t.Run("valid_create_table", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid"},
					{Name: "user_name", Type: "string"},
				},
			},
		}
		err := ValidateIdentifiers(ops)
		if err != nil {
			t.Errorf("valid identifiers should pass: %v", err)
		}
	})

	t.Run("invalid_table_name", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "InvalidTable"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid table name should fail")
		}
		// The error is wrapped in ValidationErrors, check that it contains the code
		errStr := err.Error()
		if errStr == "" {
			t.Error("error string should not be empty")
		}
	})

	t.Run("invalid_column_name", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "userName", Type: "string"}},
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid column name should fail")
		}
	})

	t.Run("invalid_namespace", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "AuthModule", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid namespace should fail")
		}
	})

	t.Run("add_column_validation", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Column:   &ast.ColumnDef{Name: "InvalidCol", Type: "string"},
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid add_column name should fail")
		}
	})

	t.Run("rename_column_validation", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.RenameColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				OldName:  "old_name",
				NewName:  "NewName",
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid rename_column new_name should fail")
		}
	})

	t.Run("rename_table_validation", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.RenameTable{
				Namespace: "auth",
				OldName:   "users",
				NewName:   "NewUsers",
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid rename_table new_name should fail")
		}
	})

	t.Run("create_index_column_validation", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Columns:  []string{"validColumn", "InvalidCol"},
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid index column should fail")
		}
	})

	t.Run("index_columns_in_create_table", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
				Indexes: []*ast.IndexDef{{Columns: []string{"InvalidColumn"}}},
			},
		}
		err := ValidateIdentifiers(ops)
		if err == nil {
			t.Error("invalid index column in create_table should fail")
		}
	})
}

// -----------------------------------------------------------------------------
// ValidateOperations Tests
// -----------------------------------------------------------------------------

func TestValidateOperations(t *testing.T) {
	t.Run("valid_operations", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string"},
				},
			},
		}
		err := ValidateOperations(ops)
		if err != nil {
			t.Errorf("valid operations should pass: %v", err)
		}
	})

	t.Run("empty_operations", func(t *testing.T) {
		err := ValidateOperations(nil)
		if err != nil {
			t.Errorf("empty operations should pass: %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// ValidateSchema Tests
// -----------------------------------------------------------------------------

func TestSandbox_ValidateSchema(t *testing.T) {
	t.Run("invalid_syntax", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.ValidateSchema("invalid {{{{ javascript")
		if err == nil {
			t.Error("invalid syntax should fail")
		}
	})

	t.Run("runtime_error", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.ValidateSchema("throw new Error('test');")
		if err == nil {
			t.Error("runtime error should fail")
		}
	})
}

// -----------------------------------------------------------------------------
// DetectForbiddenTypes Tests
// -----------------------------------------------------------------------------

func TestDetectForbiddenTypes(t *testing.T) {
	t.Run("no_forbidden_types", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid"},
					{Name: "name", Type: "string"},
					{Name: "age", Type: "integer"},
				},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err != nil {
			t.Errorf("valid types should pass: %v", err)
		}
	})

	t.Run("int64_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "big_id", Type: "int64"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("int64 should be forbidden")
		}
	})

	t.Run("bigint_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "big_id", Type: "bigint"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("bigint should be forbidden")
		}
	})

	t.Run("float64_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "rate", Type: "float64"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("float64 should be forbidden")
		}
	})

	t.Run("double_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "rate", Type: "double"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("double should be forbidden")
		}
	})

	t.Run("auto_increment_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "auto_increment"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("auto_increment should be forbidden")
		}
	})

	t.Run("serial_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "serial"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("serial should be forbidden")
		}
	})

	t.Run("bigserial_forbidden", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "bigserial"}},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("bigserial should be forbidden")
		}
	})

	t.Run("add_column_forbidden_type", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Column:   &ast.ColumnDef{Name: "big_count", Type: "int64"},
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("add_column with int64 should be forbidden")
		}
	})

	t.Run("alter_column_forbidden_type", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.AlterColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:     "count",
				NewType:  "bigint",
			},
		}
		err := DetectForbiddenTypes(ops)
		if err == nil {
			t.Error("alter_column to bigint should be forbidden")
		}
	})

	t.Run("add_column_nil_column", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Column:   nil,
			},
		}
		err := DetectForbiddenTypes(ops)
		if err != nil {
			t.Errorf("nil column should not cause error: %v", err)
		}
	})

	t.Run("alter_column_empty_new_type", func(t *testing.T) {
		ops := []ast.Operation{
			&ast.AlterColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:     "count",
				NewType:  "",
			},
		}
		err := DetectForbiddenTypes(ops)
		if err != nil {
			t.Errorf("empty new_type should not cause error: %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// checkForbiddenType Tests
// -----------------------------------------------------------------------------

func TestCheckForbiddenType(t *testing.T) {
	tests := []struct {
		typeName  string
		wantError bool
	}{
		{"int64", true},
		{"bigint", true},
		{"float64", true},
		{"double", true},
		{"auto_increment", true},
		{"serial", true},
		{"bigserial", true},
		{"uuid", false},
		{"string", false},
		{"integer", false},
		{"float", false},
		{"decimal", false},
		{"boolean", false},
		{"text", false},
		{"json", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			err := checkForbiddenType(tt.typeName)
			if tt.wantError && err == nil {
				t.Errorf("checkForbiddenType(%q) should return error", tt.typeName)
			}
			if !tt.wantError && err != nil {
				t.Errorf("checkForbiddenType(%q) should not return error: %v", tt.typeName, err)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// validateColumnName Tests (via ValidateIdentifiers)
// -----------------------------------------------------------------------------

func TestValidateColumnName_NilColumn(t *testing.T) {
	// Testing nil column via AddColumn operation
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
			Column:   nil,
		},
	}
	err := ValidateIdentifiers(ops)
	// nil column should not cause errors (it's just skipped)
	if err != nil {
		t.Errorf("nil column should not cause error: %v", err)
	}
}
