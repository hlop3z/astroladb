package builder

import (
	"github.com/dop251/goja"
)

// -----------------------------------------------------------------------------
// ColBuilder - Standalone column factory for object-based table definitions
// -----------------------------------------------------------------------------

// ColBuilder provides the col.* global for creating column definitions.
type ColBuilder struct {
	vm *goja.Runtime
}

// NewColBuilder creates a new column builder factory.
func NewColBuilder(vm *goja.Runtime) *ColBuilder {
	return &ColBuilder{vm: vm}
}

// createColumnDef creates a ColumnDef and returns a chainable JS object.
func (cb *ColBuilder) createColumnDef(colType string, opts ...ColOpt) *goja.Object {
	col := &ColumnDef{
		Type:     colType,
		Nullable: false,
	}
	for _, opt := range opts {
		opt(col)
	}
	return cb.colDefBuilder(col)
}

// colDefBuilder creates a fluent builder object for column definition chaining.
func (cb *ColBuilder) colDefBuilder(col *ColumnDef) *goja.Object {
	obj := createChainBuilder(cb.vm, col, false, cb.convertExpr)
	// Store the column definition for extraction (required for col.* API)
	_ = obj.Set("_colDef", col)
	return obj
}

// relDefBuilder creates a fluent builder for relationship definitions.
func (cb *ColBuilder) relDefBuilder(col *ColumnDef) *goja.Object {
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
		return cb.createColumnDef("uuid", withPrimaryKey())
	})

	// string(length) - length is required
	_ = obj.Set("string", func(length int) *goja.Object {
		validateStringLength(cb.vm, length, "")
		return cb.createColumnDef("string", withLength(length))
	})

	// Simple types (no extra args)
	for _, t := range simpleTypes {
		typeName := t
		_ = obj.Set(typeName, func() *goja.Object { return cb.createColumnDef(typeName) })
	}

	// decimal(precision, scale)
	_ = obj.Set("decimal", func(precision, scale int) *goja.Object {
		validateDecimalArgs(cb.vm, precision, scale, false)
		return cb.createColumnDef("decimal", withArgs(precision, scale))
	})

	// enum(values) - values is required and must not be empty
	_ = obj.Set("enum", func(values []string) *goja.Object {
		validateEnumValues(cb.vm, values)
		return cb.createColumnDef("enum", withArgs(values))
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
				return cb.createColumnDef("boolean", withDefault(defaultVal))
			})
			continue
		}

		// All other semantic types: typeName() - no column name argument
		_ = obj.Set(tn, func() *goja.Object {
			return cb.createColumnDef(semantic.BaseType, optsFromSemantic(semantic)...)
		})
	}

	// ===========================================
	// RELATIONSHIPS
	// ===========================================

	// belongs_to(ref) - FK relationship (key becomes alias_id)
	_ = obj.Set("belongs_to", func(ref string) *goja.Object {
		validateRef(cb.vm, ref, "col.belongs_to")

		col := &ColumnDef{
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
		validateRef(cb.vm, ref, "col.one_to_one")

		col := &ColumnDef{
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
		// Create a special ColumnDef that marks this as polymorphic
		// The sandbox will expand this into two columns: {alias}_type and {alias}_id
		col := &ColumnDef{
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
func (cb *ColBuilder) polyDefBuilder(col *ColumnDef) *goja.Object {
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
