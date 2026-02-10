package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/registry"
)

// TestColumnModifiers tests the column modifier functions that had 0% coverage
func TestColumnModifiers(t *testing.T) {
	tests := []struct {
		name     string
		modifier func() ColOpt
		verify   func(t *testing.T, col *ColumnDef)
	}{
		{
			name:     "withNullable",
			modifier: withNullable,
			verify: func(t *testing.T, col *ColumnDef) {
				if !col.Nullable {
					t.Error("Expected Nullable to be true")
				}
			},
		},
		{
			name: "withMin",
			modifier: func() ColOpt {
				return withMin(5.0)
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if col.Min == nil {
					t.Error("Expected Min to be set")
					return
				}
				if *col.Min != 5.0 {
					t.Errorf("Expected Min to be 5.0, got %v", *col.Min)
				}
			},
		},
		{
			name: "withMax",
			modifier: func() ColOpt {
				return withMax(100.0)
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if col.Max == nil {
					t.Error("Expected Max to be set")
					return
				}
				if *col.Max != 100.0 {
					t.Errorf("Expected Max to be 100.0, got %v", *col.Max)
				}
			},
		},
		{
			name: "withLength",
			modifier: func() ColOpt {
				return withLength(255)
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if len(col.TypeArgs) != 1 {
					t.Errorf("Expected 1 TypeArg, got %d", len(col.TypeArgs))
					return
				}
				if col.TypeArgs[0] != 255 {
					t.Errorf("Expected TypeArgs[0] to be 255, got %v", col.TypeArgs[0])
				}
			},
		},
		{
			name: "withArgs",
			modifier: func() ColOpt {
				return withArgs(10, 2)
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if len(col.TypeArgs) != 2 {
					t.Errorf("Expected 2 TypeArgs, got %d", len(col.TypeArgs))
					return
				}
				if col.TypeArgs[0] != 10 || col.TypeArgs[1] != 2 {
					t.Errorf("Expected TypeArgs [10, 2], got %v", col.TypeArgs)
				}
			},
		},
		{
			name: "withFormat",
			modifier: func() ColOpt {
				return withFormat("email")
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if col.Format != "email" {
					t.Errorf("Expected Format to be 'email', got %q", col.Format)
				}
			},
		},
		{
			name: "withPattern",
			modifier: func() ColOpt {
				return withPattern("^[a-z]+$")
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if col.Pattern != "^[a-z]+$" {
					t.Errorf("Expected Pattern to be '^[a-z]+$', got %q", col.Pattern)
				}
			},
		},
		{
			name:     "withUnique",
			modifier: withUnique,
			verify: func(t *testing.T, col *ColumnDef) {
				if !col.Unique {
					t.Error("Expected Unique to be true")
				}
			},
		},
		{
			name: "withDefault",
			modifier: func() ColOpt {
				return withDefault("default_value")
			},
			verify: func(t *testing.T, col *ColumnDef) {
				if col.Default != "default_value" {
					t.Errorf("Expected Default to be 'default_value', got %v", col.Default)
				}
			},
		},
		{
			name:     "withHidden",
			modifier: withHidden,
			verify: func(t *testing.T, col *ColumnDef) {
				if !col.Hidden {
					t.Error("Expected Hidden to be true")
				}
			},
		},
		{
			name:     "withPrimaryKey",
			modifier: withPrimaryKey,
			verify: func(t *testing.T, col *ColumnDef) {
				if !col.PrimaryKey {
					t.Error("Expected PrimaryKey to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &ColumnDef{
				Name:     "test_column",
				Type:     "string",
				Nullable: false,
			}

			modifier := tt.modifier()
			modifier(col)

			tt.verify(t, col)
		})
	}
}

// TestColumnModifiersCombined tests combining multiple modifiers
func TestColumnModifiersCombined(t *testing.T) {
	col := &ColumnDef{
		Name:     "email",
		Type:     "string",
		Nullable: false,
	}

	// Apply multiple modifiers
	modifiers := []ColOpt{
		withLength(255),
		withUnique(),
		withFormat("email"),
		withPattern("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"),
	}

	for _, mod := range modifiers {
		mod(col)
	}

	// Verify all modifiers were applied
	if len(col.TypeArgs) != 1 || col.TypeArgs[0] != 255 {
		t.Errorf("Expected length 255, got %v", col.TypeArgs)
	}
	if !col.Unique {
		t.Error("Expected Unique to be true")
	}
	if col.Format != "email" {
		t.Errorf("Expected format 'email', got %q", col.Format)
	}
	if col.Pattern == "" {
		t.Error("Expected Pattern to be set")
	}
}

// TestTableBuilderAddColumn tests the addColumn method
func TestTableBuilderAddColumn(t *testing.T) {
	sb := NewSandbox(registry.NewModelRegistry())

	tb := NewTableBuilder(sb.vm)

	// Add a column using addColumn with modifiers
	obj := tb.addColumn("email", "string",
		withLength(255),
		withUnique(),
		withNullable(),
	)

	if obj == nil {
		t.Fatal("Expected addColumn to return non-nil object")
	}

	// Verify column was added
	if len(tb.columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(tb.columns))
	}

	col := tb.columns[0]
	if col.Name != "email" {
		t.Errorf("Expected name 'email', got %q", col.Name)
	}
	if col.Type != "string" {
		t.Errorf("Expected type 'string', got %q", col.Type)
	}
	if !col.Unique {
		t.Error("Expected Unique to be true")
	}
	if !col.Nullable {
		t.Error("Expected Nullable to be true")
	}
	if len(col.TypeArgs) != 1 || col.TypeArgs[0] != 255 {
		t.Errorf("Expected TypeArgs [255], got %v", col.TypeArgs)
	}
}

// TestNumericModifiers tests min/max modifiers specifically
func TestNumericModifiers(t *testing.T) {
	tests := []struct {
		name string
		min  *float64
		max  *float64
	}{
		{
			name: "both min and max",
			min:  floatPtr(0),
			max:  floatPtr(100),
		},
		{
			name: "only min",
			min:  floatPtr(18),
			max:  nil,
		},
		{
			name: "only max",
			min:  nil,
			max:  floatPtr(65),
		},
		{
			name: "negative min",
			min:  floatPtr(-100),
			max:  floatPtr(100),
		},
		{
			name: "decimal values",
			min:  floatPtr(0.01),
			max:  floatPtr(99.99),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &ColumnDef{
				Name: "value",
				Type: "integer",
			}

			if tt.min != nil {
				withMin(*tt.min)(col)
			}
			if tt.max != nil {
				withMax(*tt.max)(col)
			}

			if tt.min != nil {
				if col.Min == nil {
					t.Error("Expected Min to be set")
				} else if *col.Min != *tt.min {
					t.Errorf("Expected Min=%v, got %v", *tt.min, *col.Min)
				}
			}

			if tt.max != nil {
				if col.Max == nil {
					t.Error("Expected Max to be set")
				} else if *col.Max != *tt.max {
					t.Errorf("Expected Max=%v, got %v", *tt.max, *col.Max)
				}
			}
		})
	}
}

// TestDefaultValueTypes tests different default value types
func TestDefaultValueTypes(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue any
	}{
		{"string", "default_value"},
		{"int", 42},
		{"float", 3.14},
		{"bool true", true},
		{"bool false", false},
		{"nil", nil},
		{"empty string", ""},
		{"zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &ColumnDef{
				Name: "test",
				Type: "string",
			}

			withDefault(tt.defaultValue)(col)

			if col.Default != tt.defaultValue {
				t.Errorf("Expected Default=%v, got %v", tt.defaultValue, col.Default)
			}
		})
	}
}

// Helper function to create float pointers
func floatPtr(f float64) *float64 {
	return &f
}
