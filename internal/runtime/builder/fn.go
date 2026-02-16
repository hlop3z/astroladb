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

	// col(name) - Reference a column by name
	_ = obj.Set("col", func(name string) *FnExpr {
		return &FnExpr{Col: name}
	})

	// Single-arg functions: fn(arg)
	singleArgFns := []string{
		"upper", "lower", "trim", "length",
		"abs", "round", "floor", "ceil",
		"year", "month", "day",
		"years_since", "days_since",
	}
	for _, name := range singleArgFns {
		fnName := name
		_ = obj.Set(fnName, func(arg any) *FnExpr {
			return &FnExpr{Fn: fnName, Args: fb.convertArgs([]any{arg})}
		})
	}

	// Two-arg functions: fn(a, b)
	twoArgFns := []string{
		"add", "sub", "mul", "div",
		"nullif", "if_null",
		"gt", "gte", "lt", "lte", "eq",
	}
	for _, name := range twoArgFns {
		fnName := name
		_ = obj.Set(fnName, func(a, b any) *FnExpr {
			return &FnExpr{Fn: fnName, Args: fb.convertArgs([]any{a, b})}
		})
	}

	// Special cases: variadic, 3-arg, 0-arg, or unique signatures

	// concat(...args)
	_ = obj.Set("concat", func(args ...any) *FnExpr {
		return &FnExpr{Fn: "concat", Args: fb.convertArgs(args)}
	})

	// substring(str, start, length)
	_ = obj.Set("substring", func(str any, start, length int) *FnExpr {
		return &FnExpr{Fn: "substring", Args: fb.convertArgs([]any{str, start, length})}
	})

	// now()
	_ = obj.Set("now", func() *FnExpr {
		return &FnExpr{Fn: "now"}
	})

	// coalesce(...args)
	_ = obj.Set("coalesce", func(args ...any) *FnExpr {
		return &FnExpr{Fn: "coalesce", Args: fb.convertArgs(args)}
	})

	// if_then(condition, thenVal, elseVal)
	_ = obj.Set("if_then", func(condition, thenVal, elseVal any) *FnExpr {
		return &FnExpr{Fn: "if_then", Args: fb.convertArgs([]any{condition, thenVal, elseVal})}
	})

	// sql({ postgres: "...", sqlite: "..." })
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
