package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestOpName tests the opName helper function.
func TestOpName(t *testing.T) {
	tests := []struct {
		name string
		op   ast.Operation
		want string
	}{
		{
			name: "CreateTable operation",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{
					Namespace: "public",
					Name:      "users",
				},
			},
			want: "users",
		},
		{
			name: "AddColumn operation - no Name field",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "products",
				},
				Column: &ast.ColumnDef{Name: "price"},
			},
			want: "", // AddColumn doesn't have a Name field, only a Column field
		},
		{
			name: "CreateIndex operation",
			op: &ast.CreateIndex{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "users",
				},
				Name: "idx_email",
			},
			want: "idx_email",
		},
		{
			name: "DropTable operation",
			op: &ast.DropTable{
				TableOp: ast.TableOp{
					Namespace: "public",
					Name:      "old_table",
				},
			},
			want: "old_table",
		},
		{
			name: "RawSQL operation without Name field",
			op: &ast.RawSQL{
				SQL: "SELECT 1",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := opName(tt.op)
			if got != tt.want {
				t.Errorf("opName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSortOpsByTypeAndName tests the sortOpsByTypeAndName helper.
func TestSortOpsByTypeAndName(t *testing.T) {
	tests := []struct {
		name     string
		input    []ast.Operation
		wantType []ast.OpType // Expected order by operation type
	}{
		{
			name: "mixed operations",
			input: []ast.Operation{
				&ast.DropTable{
					TableOp: ast.TableOp{Name: "z_table"},
				},
				&ast.CreateTable{
					TableOp: ast.TableOp{Name: "b_table"},
				},
				&ast.CreateTable{
					TableOp: ast.TableOp{Name: "a_table"},
				},
				&ast.AddColumn{
					TableRef: ast.TableRef{Table_: "products"},
					Column:   &ast.ColumnDef{Name: "price"},
				},
			},
			wantType: []ast.OpType{
				ast.OpCreateTable, // OpCreateTable=0, sorted by name (a_table first)
				ast.OpCreateTable, // OpCreateTable=0, sorted by name (b_table second)
				ast.OpDropTable,   // OpDropTable=1
				ast.OpAddColumn,   // OpAddColumn=3
			},
		},
		{
			name: "same type different names",
			input: []ast.Operation{
				&ast.CreateIndex{
					TableRef: ast.TableRef{Table_: "users"},
					Name:     "idx_z",
				},
				&ast.CreateIndex{
					TableRef: ast.TableRef{Table_: "users"},
					Name:     "idx_a",
				},
				&ast.CreateIndex{
					TableRef: ast.TableRef{Table_: "users"},
					Name:     "idx_m",
				},
			},
			wantType: []ast.OpType{
				ast.OpCreateIndex,
				ast.OpCreateIndex,
				ast.OpCreateIndex,
			},
		},
		{
			name:     "empty operations",
			input:    []ast.Operation{},
			wantType: []ast.OpType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test input
			ops := make([]ast.Operation, len(tt.input))
			copy(ops, tt.input)

			sortOpsByTypeAndName(ops)

			if len(ops) != len(tt.wantType) {
				t.Fatalf("length mismatch: got %d, want %d", len(ops), len(tt.wantType))
			}

			for i, op := range ops {
				gotType := op.Type()
				if gotType != tt.wantType[i] {
					t.Errorf("ops[%d].Type() = %v, want %v", i, gotType, tt.wantType[i])
				}
			}

			// Additional check: verify same-type operations are sorted by name
			if tt.name == "same type different names" {
				names := []string{opName(ops[0]), opName(ops[1]), opName(ops[2])}
				if !(names[0] <= names[1] && names[1] <= names[2]) {
					t.Errorf("operations not sorted by name: %v", names)
				}
			}
		})
	}
}

// TestToFloat64 tests the toFloat64 helper function.
func TestToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  float64
		ok    bool
	}{
		{
			name:  "int",
			value: int(42),
			want:  42.0,
			ok:    true,
		},
		{
			name:  "int32",
			value: int32(100),
			want:  100.0,
			ok:    true,
		},
		{
			name:  "int64",
			value: int64(9999),
			want:  9999.0,
			ok:    true,
		},
		{
			name:  "float32",
			value: float32(3.14),
			want:  float64(float32(3.14)),
			ok:    true,
		},
		{
			name:  "float64",
			value: float64(2.718),
			want:  2.718,
			ok:    true,
		},
		{
			name:  "string not convertible",
			value: "not a number",
			want:  0,
			ok:    false,
		},
		{
			name:  "nil not convertible",
			value: nil,
			want:  0,
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.value)
			if ok != tt.ok {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if tt.ok && got != tt.want {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestNumericEqual tests the numericEqual helper function.
func TestNumericEqual(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{
			name: "int and float64 with same value",
			a:    int(42),
			b:    float64(42.0),
			want: true,
		},
		{
			name: "int32 and int64 with same value",
			a:    int32(100),
			b:    int64(100),
			want: true,
		},
		{
			name: "float32 and float64 with same value",
			a:    float32(3.14),
			b:    float64(float32(3.14)),
			want: true,
		},
		{
			name: "different numeric values",
			a:    int(42),
			b:    float64(43.0),
			want: false,
		},
		{
			name: "non-numeric values equal",
			a:    "hello",
			b:    "hello",
			want: true,
		},
		{
			name: "non-numeric values not equal",
			a:    "hello",
			b:    "world",
			want: false,
		},
		{
			name: "numeric and non-numeric",
			a:    int(42),
			b:    "42",
			want: false,
		},
		{
			name: "nil values",
			a:    nil,
			b:    nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := numericEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("numericEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestTypeArgsEqual tests the typeArgsEqual helper function.
func TestTypeArgsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []any
		b    []any
		want bool
	}{
		{
			name: "both empty",
			a:    []any{},
			b:    []any{},
			want: true,
		},
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "different lengths",
			a:    []any{1, 2},
			b:    []any{1},
			want: false,
		},
		{
			name: "int and float64 with same numeric value",
			a:    []any{int(255)},
			b:    []any{float64(255)},
			want: true,
		},
		{
			name: "multiple args with numeric equality",
			a:    []any{int(10), int(20)},
			b:    []any{float64(10), float64(20)},
			want: true,
		},
		{
			name: "string args equal",
			a:    []any{"varchar", int(255)},
			b:    []any{"varchar", float64(255)},
			want: true,
		},
		{
			name: "string args not equal",
			a:    []any{"varchar", int(255)},
			b:    []any{"text", int(255)},
			want: false,
		},
		{
			name: "mixed types",
			a:    []any{int(100), "test", true},
			b:    []any{float64(100), "test", true},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := typeArgsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("typeArgsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
