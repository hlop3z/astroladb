package builder

import (
	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// FnBuilder - Expression builder for computed columns
// -----------------------------------------------------------------------------

// FnExpr represents a computed expression that can be serialized to JSON.
// Adapters translate these to SQL for their specific database dialect.
type FnExpr struct {
	Fn   string   `json:"fn"`             // Function name (e.g., "concat", "upper")
	Args []any    `json:"args,omitempty"` // Function arguments (can be FnExpr, column refs, or literals)
	Col  string   `json:"col,omitempty"`  // Column reference (for simple column access)
	SQL  SQLExprs `json:"sql,omitempty"`  // Raw SQL escape hatch (per-dialect)
}

// SQLExprs holds raw SQL expressions for different database dialects.
type SQLExprs struct {
	Postgres string `json:"postgres,omitempty"`
	SQLite   string `json:"sqlite,omitempty"`
}

// FnBuilder provides the fn.* global for creating computed expressions.
type FnBuilder struct {
	vm *goja.Runtime
}

// NewFnBuilder creates a new expression builder factory.
func NewFnBuilder(vm *goja.Runtime) *FnBuilder {
	return &FnBuilder{vm: vm}
}

// ToObject returns the fn.* global object with all expression functions.
func (fb *FnBuilder) ToObject() *goja.Object {
	obj := fb.vm.NewObject()

	// ===========================================
	// Column Reference
	// ===========================================

	// col(name) - Reference a column by name
	_ = obj.Set("col", func(name string) *FnExpr {
		return &FnExpr{Col: name}
	})

	// ===========================================
	// String Functions
	// ===========================================

	// concat(...args) - Concatenate strings
	_ = obj.Set("concat", func(args ...any) *FnExpr {
		return &FnExpr{Fn: "concat", Args: fb.convertArgs(args)}
	})

	// upper(arg) - Convert to uppercase
	_ = obj.Set("upper", func(arg any) *FnExpr {
		return &FnExpr{Fn: "upper", Args: fb.convertArgs([]any{arg})}
	})

	// lower(arg) - Convert to lowercase
	_ = obj.Set("lower", func(arg any) *FnExpr {
		return &FnExpr{Fn: "lower", Args: fb.convertArgs([]any{arg})}
	})

	// trim(arg) - Remove leading/trailing whitespace
	_ = obj.Set("trim", func(arg any) *FnExpr {
		return &FnExpr{Fn: "trim", Args: fb.convertArgs([]any{arg})}
	})

	// length(arg) - Get string length
	_ = obj.Set("length", func(arg any) *FnExpr {
		return &FnExpr{Fn: "length", Args: fb.convertArgs([]any{arg})}
	})

	// substring(str, start, length) - Extract substring
	_ = obj.Set("substring", func(str any, start, length int) *FnExpr {
		return &FnExpr{Fn: "substring", Args: fb.convertArgs([]any{str, start, length})}
	})

	// ===========================================
	// Math Functions
	// ===========================================

	// add(a, b) - Addition
	_ = obj.Set("add", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "add", Args: fb.convertArgs([]any{a, b})}
	})

	// sub(a, b) - Subtraction
	_ = obj.Set("sub", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "sub", Args: fb.convertArgs([]any{a, b})}
	})

	// mul(a, b) - Multiplication
	_ = obj.Set("mul", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "mul", Args: fb.convertArgs([]any{a, b})}
	})

	// div(a, b) - Division
	_ = obj.Set("div", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "div", Args: fb.convertArgs([]any{a, b})}
	})

	// abs(arg) - Absolute value
	_ = obj.Set("abs", func(arg any) *FnExpr {
		return &FnExpr{Fn: "abs", Args: fb.convertArgs([]any{arg})}
	})

	// round(arg) - Round to nearest integer
	_ = obj.Set("round", func(arg any) *FnExpr {
		return &FnExpr{Fn: "round", Args: fb.convertArgs([]any{arg})}
	})

	// floor(arg) - Round down
	_ = obj.Set("floor", func(arg any) *FnExpr {
		return &FnExpr{Fn: "floor", Args: fb.convertArgs([]any{arg})}
	})

	// ceil(arg) - Round up
	_ = obj.Set("ceil", func(arg any) *FnExpr {
		return &FnExpr{Fn: "ceil", Args: fb.convertArgs([]any{arg})}
	})

	// ===========================================
	// Date/Time Functions
	// ===========================================

	// year(arg) - Extract year from date/datetime
	_ = obj.Set("year", func(arg any) *FnExpr {
		return &FnExpr{Fn: "year", Args: fb.convertArgs([]any{arg})}
	})

	// month(arg) - Extract month from date/datetime
	_ = obj.Set("month", func(arg any) *FnExpr {
		return &FnExpr{Fn: "month", Args: fb.convertArgs([]any{arg})}
	})

	// day(arg) - Extract day from date/datetime
	_ = obj.Set("day", func(arg any) *FnExpr {
		return &FnExpr{Fn: "day", Args: fb.convertArgs([]any{arg})}
	})

	// now() - Current timestamp
	_ = obj.Set("now", func() *FnExpr {
		return &FnExpr{Fn: "now"}
	})

	// years_since(date) - Years elapsed since date
	_ = obj.Set("years_since", func(arg any) *FnExpr {
		return &FnExpr{Fn: "years_since", Args: fb.convertArgs([]any{arg})}
	})

	// days_since(date) - Days elapsed since date
	_ = obj.Set("days_since", func(arg any) *FnExpr {
		return &FnExpr{Fn: "days_since", Args: fb.convertArgs([]any{arg})}
	})

	// ===========================================
	// Null Handling Functions
	// ===========================================

	// coalesce(...args) - Return first non-null value
	_ = obj.Set("coalesce", func(args ...any) *FnExpr {
		return &FnExpr{Fn: "coalesce", Args: fb.convertArgs(args)}
	})

	// nullif(a, b) - Return NULL if a equals b
	_ = obj.Set("nullif", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "nullif", Args: fb.convertArgs([]any{a, b})}
	})

	// if_null(arg, default) - Return default if arg is NULL
	_ = obj.Set("if_null", func(arg, defaultVal any) *FnExpr {
		return &FnExpr{Fn: "if_null", Args: fb.convertArgs([]any{arg, defaultVal})}
	})

	// ===========================================
	// Conditional Functions
	// ===========================================

	// if_then(condition, thenVal, elseVal) - CASE WHEN condition THEN thenVal ELSE elseVal END
	_ = obj.Set("if_then", func(condition, thenVal, elseVal any) *FnExpr {
		return &FnExpr{Fn: "if_then", Args: fb.convertArgs([]any{condition, thenVal, elseVal})}
	})

	// ===========================================
	// Comparison Functions (for conditions)
	// ===========================================

	// gt(a, b) - a > b
	_ = obj.Set("gt", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "gt", Args: fb.convertArgs([]any{a, b})}
	})

	// gte(a, b) - a >= b
	_ = obj.Set("gte", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "gte", Args: fb.convertArgs([]any{a, b})}
	})

	// lt(a, b) - a < b
	_ = obj.Set("lt", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "lt", Args: fb.convertArgs([]any{a, b})}
	})

	// lte(a, b) - a <= b
	_ = obj.Set("lte", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "lte", Args: fb.convertArgs([]any{a, b})}
	})

	// eq(a, b) - a = b
	_ = obj.Set("eq", func(a, b any) *FnExpr {
		return &FnExpr{Fn: "eq", Args: fb.convertArgs([]any{a, b})}
	})

	// ===========================================
	// Raw SQL Escape Hatch
	// ===========================================

	// sql({ postgres: "...", sqlite: "..." }) - Raw SQL per dialect
	_ = obj.Set("sql", func(exprs map[string]any) *FnExpr {
		sqlExprs := SQLExprs{}
		if v, ok := exprs["postgres"].(string); ok {
			sqlExprs.Postgres = v
		}
		if v, ok := exprs["sqlite"].(string); ok {
			sqlExprs.SQLite = v
		}
		return &FnExpr{Fn: "raw", SQL: sqlExprs}
	})

	return obj
}

// convertArgs converts JS values to Go values, preserving FnExpr pointers.
func (fb *FnBuilder) convertArgs(args []any) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		result[i] = fb.convertArg(arg)
	}
	return result
}

// convertArg converts a single argument, handling FnExpr and Goja values.
func (fb *FnBuilder) convertArg(arg any) any {
	// Already a FnExpr pointer - return as-is
	if expr, ok := arg.(*FnExpr); ok {
		return expr
	}

	// Handle Goja values
	if gojaVal, ok := arg.(goja.Value); ok {
		exported := gojaVal.Export()
		// Check if the exported value is a FnExpr
		if expr, ok := exported.(*FnExpr); ok {
			return expr
		}
		// Check if it's a map (raw SQL object or nested expression)
		if m, ok := exported.(map[string]any); ok {
			return fb.convertMap(m)
		}
		return exported
	}

	return arg
}

// convertMap converts a map that might be a FnExpr or raw SQL.
func (fb *FnBuilder) convertMap(m map[string]any) any {
	// Check if this is a serialized FnExpr
	if fn, ok := m["fn"].(string); ok {
		expr := &FnExpr{Fn: fn}
		if args, ok := m["args"].([]any); ok {
			expr.Args = fb.convertArgs(args)
		}
		if col, ok := m["col"].(string); ok {
			expr.Col = col
		}
		return expr
	}

	// Check if this is a column reference
	if col, ok := m["col"].(string); ok {
		return &FnExpr{Col: col}
	}

	// Return as-is for other maps
	return m
}

// ExprToMap converts a FnExpr to a map for JSON serialization.
func ExprToMap(expr *FnExpr) map[string]any {
	if expr == nil {
		return nil
	}

	m := make(map[string]any)

	if expr.Fn != "" {
		m["fn"] = expr.Fn
	}
	if expr.Col != "" {
		m["col"] = expr.Col
	}
	if len(expr.Args) > 0 {
		args := make([]any, len(expr.Args))
		for i, arg := range expr.Args {
			if subExpr, ok := arg.(*FnExpr); ok {
				args[i] = ExprToMap(subExpr)
			} else {
				args[i] = arg
			}
		}
		m["args"] = args
	}
	if expr.SQL.Postgres != "" || expr.SQL.SQLite != "" {
		sql := make(map[string]any)
		if expr.SQL.Postgres != "" {
			sql["postgres"] = expr.SQL.Postgres
		}
		if expr.SQL.SQLite != "" {
			sql["sqlite"] = expr.SQL.SQLite
		}
		m["sql"] = sql
	}

	return m
}
