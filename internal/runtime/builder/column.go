package builder

import (
	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// ColBuilder - Standalone column factory for object-based table definitions
// -----------------------------------------------------------------------------

// ColDef is an alias for ColumnDef used by the col.* global.
// The Name field is ignored when used with ColBuilder (name comes from object key).
type ColDef = ColumnDef

// ColBuilder provides the col.* global for creating column definitions.
type ColBuilder struct {
	vm *goja.Runtime
}

// NewColBuilder creates a new column builder factory.
func NewColBuilder(vm *goja.Runtime) *ColBuilder {
	return &ColBuilder{vm: vm}
}

// colOpt is an alias for ColOpt since ColDef = ColumnDef.
type colOpt = ColOpt

// Internal convenience aliases that reference the shared with* functions.
var (
	colWithLength     = withLength
	colWithArgs       = withArgs
	colWithDefault    = withDefault
	colWithPrimaryKey = withPrimaryKey
)

// colOptsFromSemantic is an alias for optsFromSemantic since ColDef = ColumnDef.
var colOptsFromSemantic = optsFromSemantic

// createColDef creates a ColDef and returns a chainable JS object.
func (cb *ColBuilder) createColDef(colType string, opts ...colOpt) *goja.Object {
	col := &ColDef{
		Type:     colType,
		Nullable: false,
	}
	for _, opt := range opts {
		opt(col)
	}
	return cb.colDefBuilder(col)
}

// colDefBuilder creates a fluent builder object for column definition chaining.
func (cb *ColBuilder) colDefBuilder(col *ColDef) *goja.Object {
	obj := createChainBuilder(cb.vm, col, false, cb.convertExpr)
	// Store the column definition for extraction (required for col.* API)
	_ = obj.Set("_colDef", col)
	return obj
}

// relDefBuilder creates a fluent builder for relationship definitions.
func (cb *ColBuilder) relDefBuilder(col *ColDef) *goja.Object {
	obj := createChainBuilder(cb.vm, col, true, cb.convertExpr)
	// Store the column definition for extraction (required for col.* API)
	_ = obj.Set("_colDef", col)
	return obj
}

// ToObject converts the ColBuilder to a JavaScript object with all column factory methods.
func (cb *ColBuilder) ToObject() *goja.Object {
	obj := cb.vm.NewObject()

	// id() - UUID primary key
	_ = obj.Set("id", func() *goja.Object {
		return cb.createColDef("uuid", colWithPrimaryKey())
	})

	// string(length) - length is required
	_ = obj.Set("string", func(length int) *goja.Object {
		if length <= 0 {
			panic(cb.vm.ToValue(ErrMsgStringRequiresLength.String()))
		}
		return cb.createColDef("string", colWithLength(length))
	})

	// text()
	_ = obj.Set("text", func() *goja.Object { return cb.createColDef("text") })

	// integer()
	_ = obj.Set("integer", func() *goja.Object { return cb.createColDef("integer") })

	// float()
	_ = obj.Set("float", func() *goja.Object { return cb.createColDef("float") })

	// decimal(precision, scale)
	_ = obj.Set("decimal", func(precision, scale int) *goja.Object {
		if precision <= 0 || scale < 0 {
			panic(cb.vm.ToValue(ErrMsgDecimalRequiresArgs.String()))
		}
		return cb.createColDef("decimal", colWithArgs(precision, scale))
	})

	// boolean()
	_ = obj.Set("boolean", func() *goja.Object { return cb.createColDef("boolean") })

	// date()
	_ = obj.Set("date", func() *goja.Object { return cb.createColDef("date") })

	// time()
	_ = obj.Set("time", func() *goja.Object { return cb.createColDef("time") })

	// datetime()
	_ = obj.Set("datetime", func() *goja.Object { return cb.createColDef("datetime") })

	// uuid()
	_ = obj.Set("uuid", func() *goja.Object { return cb.createColDef("uuid") })

	// json()
	_ = obj.Set("json", func() *goja.Object { return cb.createColDef("json") })

	// base64()
	_ = obj.Set("base64", func() *goja.Object { return cb.createColDef("base64") })

	// enum(values) - values is required and must not be empty
	_ = obj.Set("enum", func(values []string) *goja.Object {
		if len(values) == 0 {
			panic(cb.vm.ToValue(ErrMsgEnumRequiresValues.String()))
		}
		return cb.createColDef("enum", colWithArgs(values))
	})

	// ===========================================
	// SEMANTIC TYPES - Registered from centralized SemanticTypes map
	// ===========================================
	for typeName, st := range SemanticTypes {
		// Capture loop variables for closure
		tn, semantic := typeName, st

		// flag is special - takes optional default value argument
		if tn == "flag" {
			_ = obj.Set("flag", func(call goja.FunctionCall) goja.Value {
				defaultVal := false
				if len(call.Arguments) > 0 {
					if v, ok := call.Arguments[0].Export().(bool); ok {
						defaultVal = v
					}
				}
				return cb.createColDef("boolean", colWithDefault(defaultVal))
			})
			continue
		}

		// All other semantic types: typeName() - no column name argument
		_ = obj.Set(tn, func() *goja.Object {
			return cb.createColDef(semantic.BaseType, colOptsFromSemantic(semantic)...)
		})
	}

	// ===========================================
	// RELATIONSHIPS
	// ===========================================

	// belongs_to(ref) - FK relationship (key becomes alias_id)
	_ = obj.Set("belongs_to", func(ref string) *goja.Object {
		// Validate: reference cannot be empty
		if ref == "" {
			panic(cb.vm.ToValue(ErrMsgBelongsToRequiresRef.String()))
		}
		// Validate: reference must have namespace
		if !alerr.HasNamespace(ref) {
			panic(cb.vm.ToValue(alerr.NewMissingNamespaceError(ref).Error()))
		}

		col := &ColDef{
			Type:           "uuid",
			IsRelationship: true,
			Reference: &RefDef{
				Table:  ref,
				Column: "id",
			},
			XRef: ref,
		}
		return cb.relDefBuilder(col)
	})

	// one_to_one(ref) - Unique FK relationship
	_ = obj.Set("one_to_one", func(ref string) *goja.Object {
		if !alerr.HasNamespace(ref) {
			panic(cb.vm.ToValue(alerr.NewMissingNamespaceError(ref).Error()))
		}

		col := &ColDef{
			Type:           "uuid",
			Unique:         true,
			IsRelationship: true,
			Reference: &RefDef{
				Table:  ref,
				Column: "id",
			},
			XRef: ref,
		}
		return cb.relDefBuilder(col)
	})

	// belongs_to_any(refs) - Polymorphic relationship (creates _type and _id columns)
	// The key name in the object becomes the polymorphic alias
	_ = obj.Set("belongs_to_any", func(refs []string) *goja.Object {
		// Create a special ColDef that marks this as polymorphic
		// The sandbox will expand this into two columns: {alias}_type and {alias}_id
		col := &ColDef{
			Type:           "polymorphic",
			IsRelationship: true,
			TypeArgs:       make([]any, len(refs)),
		}
		// Store refs in TypeArgs for later extraction
		for i, ref := range refs {
			col.TypeArgs[i] = ref
		}
		return cb.polyDefBuilder(col)
	})

	return obj
}

// convertExpr converts a JS expression value to a Go value.
func (cb *ColBuilder) convertExpr(expr any) any {
	// Handle Goja values
	if gojaVal, ok := expr.(goja.Value); ok {
		exported := gojaVal.Export()
		// Check if it's a FnExpr pointer
		if fnExpr, ok := exported.(*FnExpr); ok {
			return ExprToMap(fnExpr)
		}
		return exported
	}
	// Check if it's already a FnExpr pointer
	if fnExpr, ok := expr.(*FnExpr); ok {
		return ExprToMap(fnExpr)
	}
	return expr
}

// polyDefBuilder creates a builder for polymorphic relationship definitions.
func (cb *ColBuilder) polyDefBuilder(col *ColDef) *goja.Object {
	obj := cb.vm.NewObject()

	// Store the column definition for extraction
	_ = obj.Set("_colDef", col)

	// optional() - marks polymorphic FK as nullable
	_ = obj.Set("optional", func() *goja.Object {
		col.Nullable = true
		return obj
	})

	return obj
}
