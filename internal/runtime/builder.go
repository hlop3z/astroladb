package runtime

import (
	"log/slog"
	"strings"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// Type-safe column and index definitions
// -----------------------------------------------------------------------------

// ColumnDef represents a type-safe column definition during building.
// This provides compile-time safety and IDE support when creating columns.
type ColumnDef struct {
	Name           string
	Type           string
	TypeArgs       []any
	Nullable       bool
	Unique         bool
	PrimaryKey     bool
	Default        any
	Backfill       any
	Format         string
	Pattern        string
	Min            *float64
	Max            *float64
	Docs           string
	Deprecated     string
	Reference      *RefDef
	Hidden         bool   // x_hidden for OpenAPI
	XRef           string // Original reference (e.g., "auth.user")
	Computed       any    // Computed expression (FnExpr or map)
	Virtual        bool   // VIRTUAL instead of STORED, or app-only if no Computed
	IsRelationship bool   // True if this is a relationship column
}

// RefDef represents a foreign key reference.
type RefDef struct {
	Table    string
	Column   string
	OnDelete string
	OnUpdate string
}

// IndexDef represents a type-safe index definition.
type IndexDef struct {
	Name    string
	Columns []string
	Unique  bool
	Where   string
}

// RelationshipDef represents a many-to-many or polymorphic relationship.
type RelationshipDef struct {
	Type    string   // "many_to_many" or "polymorphic"
	Target  string   // Target table for many_to_many
	Targets []string // Target tables for polymorphic
	As      string   // Alias
}

// -----------------------------------------------------------------------------
// TableBuilder
// -----------------------------------------------------------------------------

// TableBuilder provides the JavaScript interface for building table definitions.
// It accumulates column definitions, indexes, and other table properties.
type TableBuilder struct {
	vm *goja.Runtime

	columns       []*ColumnDef
	indexes       []*IndexDef
	relationships []*RelationshipDef
	docs          string
	deprecated    string
}

// NewTableBuilder creates a new table builder for JavaScript.
func NewTableBuilder(vm *goja.Runtime) *TableBuilder {
	return &TableBuilder{
		vm:            vm,
		columns:       make([]*ColumnDef, 0),
		indexes:       make([]*IndexDef, 0),
		relationships: make([]*RelationshipDef, 0),
	}
}

// -----------------------------------------------------------------------------
// Functional options for column configuration
// -----------------------------------------------------------------------------

// ColOpt is a functional option for configuring column properties.
type ColOpt func(*ColumnDef)

// optsFromSemantic converts a SemanticType to a slice of ColOpts.
// This enables DRY registration of semantic types from the centralized map.
func optsFromSemantic(st SemanticType) []ColOpt {
	var opts []ColOpt

	// Handle type-specific arguments
	switch st.BaseType {
	case "string":
		if st.Length > 0 {
			opts = append(opts, withLength(st.Length))
		}
	case "decimal":
		if st.Precision > 0 {
			opts = append(opts, withArgs(st.Precision, st.Scale))
		}
	}

	// Common options
	if st.Format != "" {
		opts = append(opts, withFormat(st.Format))
	}
	if st.Pattern != "" {
		opts = append(opts, withPattern(st.Pattern))
	}
	if st.Unique {
		opts = append(opts, withUnique())
	}
	if st.Default != nil {
		opts = append(opts, withDefault(st.Default))
	}
	if st.Hidden {
		opts = append(opts, withHidden())
	}
	// Min/Max - copy pointers to avoid closure issues
	if st.Min != nil {
		m := st.Min
		opts = append(opts, func(c *ColumnDef) { c.Min = m })
	}
	if st.Max != nil {
		m := st.Max
		opts = append(opts, func(c *ColumnDef) { c.Max = m })
	}

	return opts
}

// Column option helpers for DRY column definitions.
func withLength(n int) ColOpt     { return func(c *ColumnDef) { c.TypeArgs = []any{n} } }
func withArgs(args ...any) ColOpt { return func(c *ColumnDef) { c.TypeArgs = args } }
func withFormat(f string) ColOpt  { return func(c *ColumnDef) { c.Format = f } }
func withPattern(p string) ColOpt { return func(c *ColumnDef) { c.Pattern = p } }
func withUnique() ColOpt          { return func(c *ColumnDef) { c.Unique = true } }
func withDefault(v any) ColOpt    { return func(c *ColumnDef) { c.Default = v } }
func withHidden() ColOpt          { return func(c *ColumnDef) { c.Hidden = true } }
func withPrimaryKey() ColOpt      { return func(c *ColumnDef) { c.PrimaryKey = true } }
func withNullable() ColOpt        { return func(c *ColumnDef) { c.Nullable = true } }

func withMin(n float64) ColOpt { return func(c *ColumnDef) { c.Min = &n } }
func withMax(n float64) ColOpt { return func(c *ColumnDef) { c.Max = &n } }

// addColumn is a DRY helper that creates a column with the given options.
func (tb *TableBuilder) addColumn(name, colType string, opts ...ColOpt) *goja.Object {
	col := &ColumnDef{
		Name:     name,
		Type:     colType,
		Nullable: false,
	}
	for _, opt := range opts {
		opt(col)
	}
	tb.columns = append(tb.columns, col)
	return tb.columnBuilder(col)
}

// ToMigrationObject converts the builder to a JavaScript object with LOW-LEVEL types only.
// Used by migrations where only explicit types are allowed (no semantic types).
// This ensures migrations are explicit and stable even if semantic type definitions change.
func (tb *TableBuilder) ToMigrationObject() *goja.Object {
	return tb.toBaseObject()
}

// toBaseObject creates the base JavaScript object with low-level types, relationships, and helpers.
// Used by ToMigrationObject() for migration table builders.
func (tb *TableBuilder) toBaseObject() *goja.Object {
	obj := tb.vm.NewObject()

	// id() - UUID primary key
	_ = obj.Set("id", func() *goja.Object {
		return tb.addColumn("id", "uuid", withPrimaryKey())
	})

	// string(name, length) - length is required
	_ = obj.Set("string", func(name string, length int) *goja.Object {
		if length <= 0 {
			panic(tb.vm.ToValue(alerr.NewMissingLengthError(name).Error()))
		}
		return tb.addColumn(name, "string", withLength(length))
	})

	// text(name)
	_ = obj.Set("text", func(name string) *goja.Object { return tb.addColumn(name, "text") })

	// integer(name)
	_ = obj.Set("integer", func(name string) *goja.Object { return tb.addColumn(name, "integer") })

	// float(name)
	_ = obj.Set("float", func(name string) *goja.Object { return tb.addColumn(name, "float") })

	// decimal(name, precision, scale)
	_ = obj.Set("decimal", func(name string, precision, scale int) *goja.Object {
		return tb.addColumn(name, "decimal", withArgs(precision, scale))
	})

	// boolean(name)
	_ = obj.Set("boolean", func(name string) *goja.Object { return tb.addColumn(name, "boolean") })

	// date(name)
	_ = obj.Set("date", func(name string) *goja.Object { return tb.addColumn(name, "date") })

	// time(name)
	_ = obj.Set("time", func(name string) *goja.Object { return tb.addColumn(name, "time") })

	// datetime(name) - timestamp column
	_ = obj.Set("datetime", func(name string) *goja.Object { return tb.addColumn(name, "datetime") })

	// uuid(name)
	_ = obj.Set("uuid", func(name string) *goja.Object { return tb.addColumn(name, "uuid") })

	// json(name)
	_ = obj.Set("json", func(name string) *goja.Object { return tb.addColumn(name, "json") })

	// base64(name)
	_ = obj.Set("base64", func(name string) *goja.Object { return tb.addColumn(name, "base64") })

	// enum(name, values)
	_ = obj.Set("enum", func(name string, values []string) *goja.Object {
		return tb.addColumn(name, "enum", withArgs(values))
	})

	// timestamps() - adds created_at and updated_at
	_ = obj.Set("timestamps", func() {
		tb.columns = append(tb.columns, &ColumnDef{
			Name:    "created_at",
			Type:    "datetime",
			Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
		})
		tb.columns = append(tb.columns, &ColumnDef{
			Name:    "updated_at",
			Type:    "datetime",
			Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
		})
	})

	// soft_delete() - adds deleted_at
	_ = obj.Set("soft_delete", func() {
		tb.columns = append(tb.columns, &ColumnDef{
			Name:     "deleted_at",
			Type:     "datetime",
			Nullable: true,
		})
	})

	// sortable() - adds position
	_ = obj.Set("sortable", func() {
		tb.columns = append(tb.columns, &ColumnDef{
			Name:    "position",
			Type:    "integer",
			Default: 0,
		})
	})

	// belongs_to(ref) - Foreign key relationship with chaining API
	// Examples:
	//   belongs_to("auth.user")                                    -> user_id
	//   belongs_to("auth.user").as("author")                       -> author_id
	//   belongs_to("auth.user").optional()                         -> user_id, nullable
	//   belongs_to("auth.user").as("author").optional().on_delete("cascade")
	_ = obj.Set("belongs_to", func(ref string) *goja.Object {
		// Validate: reference must have namespace
		if !alerr.HasNamespace(ref) {
			panic(tb.vm.ToValue(alerr.NewMissingNamespaceError(ref).Error()))
		}

		// Default column name from table reference
		colName := extractTableName(ref) + "_id"

		col := &ColumnDef{
			Name: colName,
			Type: "uuid",
			Reference: &RefDef{
				Table:  ref,
				Column: "id",
			},
			XRef: ref,
		}
		tb.columns = append(tb.columns, col)

		// Auto-create index on foreign key
		idx := &IndexDef{
			Columns: []string{colName},
			Unique:  false,
		}
		tb.indexes = append(tb.indexes, idx)

		return tb.relationshipBuilder(ref, col, idx)
	})

	// one_to_one(ref) - Unique foreign key relationship with chaining API
	// Same as belongs_to but creates unique constraint
	_ = obj.Set("one_to_one", func(ref string) *goja.Object {
		// Validate: reference must have namespace
		if !alerr.HasNamespace(ref) {
			panic(tb.vm.ToValue(alerr.NewMissingNamespaceError(ref).Error()))
		}

		colName := extractTableName(ref) + "_id"

		col := &ColumnDef{
			Name:   colName,
			Type:   "uuid",
			Unique: true,
			Reference: &RefDef{
				Table:  ref,
				Column: "id",
			},
			XRef: ref,
		}
		tb.columns = append(tb.columns, col)

		// Auto-create unique index on foreign key
		idx := &IndexDef{
			Columns: []string{colName},
			Unique:  true,
		}
		tb.indexes = append(tb.indexes, idx)

		return tb.relationshipBuilder(ref, col, idx)
	})

	// many_to_many(ref, opts)
	_ = obj.Set("many_to_many", func(ref string, opts ...map[string]any) {
		// many_to_many generates a join table automatically
		// Store as relationship metadata for later processing
		rel := &RelationshipDef{
			Type:   "many_to_many",
			Target: ref,
		}
		if len(opts) > 0 {
			if v, ok := opts[0]["as"].(string); ok {
				rel.As = v
			}
		}
		tb.relationships = append(tb.relationships, rel)
	})

	// belongs_to_any(refs, opts)
	_ = obj.Set("belongs_to_any", func(refs []string, opts ...map[string]any) {
		// Polymorphic relationship: adds type and id columns
		var as string
		if len(opts) > 0 {
			if v, ok := opts[0]["as"].(string); ok {
				as = v
			}
		}
		if as == "" {
			as = "polymorphic"
		}

		typeCol := as + "_type"
		idCol := as + "_id"

		// Add type column
		tb.columns = append(tb.columns, &ColumnDef{
			Name:     typeCol,
			Type:     "string",
			TypeArgs: []any{100},
		})

		// Add id column
		tb.columns = append(tb.columns, &ColumnDef{
			Name: idCol,
			Type: "uuid",
		})

		// Auto-create composite index on (type, id)
		tb.indexes = append(tb.indexes, &IndexDef{
			Columns: []string{typeCol, idCol},
		})

		// Store polymorphic relationship metadata
		tb.relationships = append(tb.relationships, &RelationshipDef{
			Type:    "polymorphic",
			Targets: refs,
			As:      as,
		})
	})

	// index(columns...)
	_ = obj.Set("index", func(columns ...string) {
		tb.indexes = append(tb.indexes, &IndexDef{
			Columns: columns,
		})
	})

	// unique(columns...) - composite uniqueness constraint
	// Clean API for relationship tables: unique("follower", "follows")
	// Automatically appends _id suffix for relationship aliases
	_ = obj.Set("unique", func(columns ...string) {
		// Build a set of existing column names
		existingCols := make(map[string]bool)
		for _, c := range tb.columns {
			existingCols[c.Name] = true
		}

		// Convert field names to column names
		cols := make([]string, len(columns))
		for i, col := range columns {
			// If column already ends with _id or _type, use as-is
			if strings.HasSuffix(col, "_id") || strings.HasSuffix(col, "_type") {
				cols[i] = col
			} else if existingCols[col+"_id"] {
				// If {col}_id exists as a column, use it (relationship alias)
				cols[i] = col + "_id"
			} else {
				// Otherwise use the column name as-is (regular field like "type")
				cols[i] = col
			}
		}
		tb.indexes = append(tb.indexes, &IndexDef{
			Columns: cols,
			Unique:  true,
		})
	})

	// docs(description)
	_ = obj.Set("docs", func(description string) {
		tb.docs = description
	})

	// deprecated(reason)
	_ = obj.Set("deprecated", func(reason string) {
		tb.deprecated = reason
	})

	return obj
}

// columnBuilder creates a fluent column builder object for method chaining.
func (tb *TableBuilder) columnBuilder(col *ColumnDef) *goja.Object {
	return createChainBuilder(tb.vm, col, false)
}

// createChainBuilder creates a shared fluent builder for column definitions.
// This consolidates the common methods between columnBuilder, colDefBuilder, and relDefBuilder.
// includeRelOpts adds on_delete/on_update methods for relationship columns.
func createChainBuilder(vm *goja.Runtime, col *ColumnDef, includeRelOpts bool, convertExpr ...func(any) any) *goja.Object {
	obj := vm.NewObject()

	// optional() - marks column as nullable (allows NULL values)
	_ = obj.Set("optional", func() *goja.Object {
		col.Nullable = true
		return obj
	})

	// unique()
	_ = obj.Set("unique", func() *goja.Object {
		col.Unique = true
		return obj
	})

	// default(value)
	_ = obj.Set("default", func(value any) *goja.Object {
		col.Default = value
		return obj
	})

	// backfill(value)
	_ = obj.Set("backfill", func(value any) *goja.Object {
		col.Backfill = value
		return obj
	})

	// min(n)
	_ = obj.Set("min", func(n int) *goja.Object {
		minVal := float64(n)
		col.Min = &minVal
		return obj
	})

	// max(n)
	_ = obj.Set("max", func(n int) *goja.Object {
		maxVal := float64(n)
		col.Max = &maxVal
		return obj
	})

	// pattern(regex)
	_ = obj.Set("pattern", func(regex string) *goja.Object {
		col.Pattern = regex
		return obj
	})

	// format(fmt)
	_ = obj.Set("format", func(f goja.Value) *goja.Object {
		if f == nil || goja.IsUndefined(f) || goja.IsNull(f) {
			return obj
		}
		// Handle format object from fmt.* constants
		if fObj, ok := f.(*goja.Object); ok {
			if format := fObj.Get("format"); format != nil && !goja.IsUndefined(format) {
				col.Format = format.String()
			}
			// Auto-apply pattern if not already set
			if col.Pattern == "" {
				if pattern := fObj.Get("pattern"); pattern != nil && !goja.IsUndefined(pattern) {
					patternStr := pattern.String()
					if patternStr != "" {
						col.Pattern = patternStr
					}
				}
			}
		} else if s, ok := f.Export().(string); ok {
			col.Format = s
		}
		return obj
	})

	// docs(description)
	_ = obj.Set("docs", func(description string) *goja.Object {
		col.Docs = description
		return obj
	})

	// deprecated(reason)
	_ = obj.Set("deprecated", func(reason string) *goja.Object {
		col.Deprecated = reason
		return obj
	})

	// computed(expr) - marks column as a computed/virtual column
	if len(convertExpr) > 0 && convertExpr[0] != nil {
		converter := convertExpr[0]
		_ = obj.Set("computed", func(expr any) *goja.Object {
			col.Computed = converter(expr)
			return obj
		})
	}

	// virtual() - marks column as VIRTUAL (not STORED) or app-only (no DB column)
	_ = obj.Set("virtual", func() *goja.Object {
		col.Virtual = true
		return obj
	})

	// Relationship-specific options
	if includeRelOpts {
		// on_delete(action)
		_ = obj.Set("on_delete", func(action string) *goja.Object {
			if col.Reference != nil {
				col.Reference.OnDelete = action
			}
			return obj
		})

		// on_update(action)
		_ = obj.Set("on_update", func(action string) *goja.Object {
			if col.Reference != nil {
				col.Reference.OnUpdate = action
			}
			return obj
		})
	}

	return obj
}

// relationshipBuilder creates a fluent builder for FK relationships with chaining.
// Supports: .as(alias), .optional(), .on_delete(action), .on_update(action)
func (tb *TableBuilder) relationshipBuilder(ref string, col *ColumnDef, idx *IndexDef) *goja.Object {
	obj := tb.vm.NewObject()

	// as(alias) - Set custom column name
	_ = obj.Set("as", func(alias string) *goja.Object {
		newName := alias + "_id"
		col.Name = newName
		idx.Columns = []string{newName}
		return obj
	})

	// optional() - marks foreign key as nullable
	_ = obj.Set("optional", func() *goja.Object {
		col.Nullable = true
		return obj
	})

	// on_delete(action) - Set ON DELETE action
	_ = obj.Set("on_delete", func(action string) *goja.Object {
		if col.Reference != nil {
			col.Reference.OnDelete = action
		}
		return obj
	})

	// on_update(action) - Set ON UPDATE action
	_ = obj.Set("on_update", func(action string) *goja.Object {
		if col.Reference != nil {
			col.Reference.OnUpdate = action
		}
		return obj
	})

	// Also include column modifiers for convenience
	// docs(description)
	_ = obj.Set("docs", func(description string) *goja.Object {
		col.Docs = description
		return obj
	})

	// deprecated(reason)
	_ = obj.Set("deprecated", func(reason string) *goja.Object {
		col.Deprecated = reason
		return obj
	})

	return obj
}

// ToResult converts the builder state to a result object for JavaScript.
// Converts typed structs back to maps for sandbox.go consumption.
func (tb *TableBuilder) ToResult() goja.Value {
	result := tb.vm.NewObject()

	// Convert columns to maps (using standalone functions to avoid duplication)
	columns := make([]map[string]any, 0, len(tb.columns)+len(tb.relationships))
	for _, col := range tb.columns {
		columns = append(columns, columnDefToMap(col))
	}

	// Add relationship markers (many_to_many, polymorphic)
	for _, rel := range tb.relationships {
		columns = append(columns, relationshipDefToMap(rel))
	}

	// Convert indexes to maps
	indexes := make([]map[string]any, 0, len(tb.indexes))
	for _, idx := range tb.indexes {
		indexes = append(indexes, indexDefToMap(idx))
	}

	_ = result.Set("columns", columns)
	_ = result.Set("indexes", indexes)
	if tb.docs != "" {
		_ = result.Set("docs", tb.docs)
	}
	if tb.deprecated != "" {
		_ = result.Set("deprecated", tb.deprecated)
	}
	return result
}

// extractTableName extracts the table name from a reference.
// With singular table convention: "auth.user" -> "user", "core.order" -> "order"
func extractTableName(ref string) string {
	// Find the table part after the last dot
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			return ref[i+1:]
		}
	}
	return ref
}

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
	colWithFormat     = withFormat
	colWithPattern    = withPattern
	colWithUnique     = withUnique
	colWithDefault    = withDefault
	colWithHidden     = withHidden
	colWithPrimaryKey = withPrimaryKey
	colWithMin        = withMin
	colWithMax        = withMax
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
			panic(cb.vm.ToValue("string() requires a positive length"))
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

	// enum(values)
	_ = obj.Set("enum", func(values []string) *goja.Object {
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

// -----------------------------------------------------------------------------
// TableChain - Chainable table operations for object-based API
// -----------------------------------------------------------------------------

// TableChain provides chainable table-level operations after column definitions.
type TableChain struct {
	vm            *goja.Runtime
	columns       []*ColumnDef
	indexes       []*IndexDef
	relationships []*RelationshipDef
	docs          string
	deprecated    string

	// Table-level metadata for x-db extensions
	auditable  bool     // Add created_by, updated_by columns
	sortBy     []string // Default ordering (e.g., ["-created_at", "name"])
	searchable []string // Columns for fulltext search
	filterable []string // Columns allowed in WHERE clauses
}

// NewTableChain creates a TableChain from collected column definitions.
func NewTableChain(vm *goja.Runtime, columns []*ColumnDef, indexes []*IndexDef) *TableChain {
	return &TableChain{
		vm:            vm,
		columns:       columns,
		indexes:       indexes,
		relationships: make([]*RelationshipDef, 0),
	}
}

// ToChainableObject returns a JS object with chainable table methods.
func (tc *TableChain) ToChainableObject() *goja.Object {
	obj := tc.vm.NewObject()

	// timestamps() - adds created_at and updated_at
	_ = obj.Set("timestamps", func() *goja.Object {
		tc.columns = append(tc.columns, &ColumnDef{
			Name:    "created_at",
			Type:    "datetime",
			Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
		})
		tc.columns = append(tc.columns, &ColumnDef{
			Name:    "updated_at",
			Type:    "datetime",
			Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
		})
		return obj
	})

	// soft_delete() - adds deleted_at
	_ = obj.Set("soft_delete", func() *goja.Object {
		tc.columns = append(tc.columns, &ColumnDef{
			Name:     "deleted_at",
			Type:     "datetime",
			Nullable: true,
		})
		return obj
	})

	// sortable() - adds position
	_ = obj.Set("sortable", func() *goja.Object {
		tc.columns = append(tc.columns, &ColumnDef{
			Name:    "position",
			Type:    "integer",
			Default: 0,
		})
		return obj
	})

	// index(...columns) - composite index
	_ = obj.Set("index", func(columns ...string) *goja.Object {
		tc.indexes = append(tc.indexes, &IndexDef{
			Columns: columns,
		})
		return obj
	})

	// unique(...columns) - composite uniqueness
	_ = obj.Set("unique", func(columns ...string) *goja.Object {
		existingCols := make(map[string]bool)
		for _, c := range tc.columns {
			existingCols[c.Name] = true
		}

		cols := make([]string, len(columns))
		for i, col := range columns {
			if strings.HasSuffix(col, "_id") || strings.HasSuffix(col, "_type") {
				cols[i] = col
			} else if existingCols[col+"_id"] {
				cols[i] = col + "_id"
			} else {
				cols[i] = col
			}
		}
		tc.indexes = append(tc.indexes, &IndexDef{
			Columns: cols,
			Unique:  true,
		})
		return obj
	})

	// many_to_many(ref)
	_ = obj.Set("many_to_many", func(ref string) *goja.Object {
		tc.relationships = append(tc.relationships, &RelationshipDef{
			Type:   "many_to_many",
			Target: ref,
		})
		return obj
	})

	// belongs_to_any(refs, opts)
	_ = obj.Set("belongs_to_any", func(refs []string, opts ...map[string]any) *goja.Object {
		var as string
		if len(opts) > 0 {
			if v, ok := opts[0]["as"].(string); ok {
				as = v
			}
		}
		if as == "" {
			as = "polymorphic"
		}

		typeCol := as + "_type"
		idCol := as + "_id"

		tc.columns = append(tc.columns, &ColumnDef{
			Name:     typeCol,
			Type:     "string",
			TypeArgs: []any{100},
		})
		tc.columns = append(tc.columns, &ColumnDef{
			Name: idCol,
			Type: "uuid",
		})
		tc.indexes = append(tc.indexes, &IndexDef{
			Columns: []string{typeCol, idCol},
		})
		tc.relationships = append(tc.relationships, &RelationshipDef{
			Type:    "polymorphic",
			Targets: refs,
			As:      as,
		})
		return obj
	})

	// docs(description) - table documentation
	_ = obj.Set("docs", func(description string) *goja.Object {
		tc.docs = description
		return obj
	})

	// deprecated(reason) - mark table as deprecated
	_ = obj.Set("deprecated", func(reason string) *goja.Object {
		tc.deprecated = reason
		return obj
	})

	// auditable() - adds created_by and updated_by columns
	_ = obj.Set("auditable", func() *goja.Object {
		tc.auditable = true
		tc.columns = append(tc.columns, &ColumnDef{
			Name:     "created_by",
			Type:     "uuid",
			Nullable: true,
		})
		tc.columns = append(tc.columns, &ColumnDef{
			Name:     "updated_by",
			Type:     "uuid",
			Nullable: true,
		})
		return obj
	})

	// sort_by(...columns) - default ordering (e.g., ["-created_at", "name"])
	_ = obj.Set("sort_by", func(columns ...string) *goja.Object {
		tc.sortBy = columns
		return obj
	})

	// searchable(...columns) - columns for fulltext search
	_ = obj.Set("searchable", func(columns ...string) *goja.Object {
		tc.searchable = columns
		return obj
	})

	// filterable(...columns) - columns allowed in WHERE clauses
	_ = obj.Set("filterable", func(columns ...string) *goja.Object {
		tc.filterable = columns
		return obj
	})

	// Store result data for extraction
	_ = obj.Set("_getResult", func() goja.Value {
		return tc.ToResult()
	})

	return obj
}

// ToResult converts the TableChain to a result object for JavaScript.
func (tc *TableChain) ToResult() goja.Value {
	result := tc.vm.NewObject()

	// Convert columns to maps
	columns := make([]map[string]any, 0, len(tc.columns)+len(tc.relationships))
	for _, col := range tc.columns {
		columns = append(columns, columnDefToMap(col))
	}

	// Add relationship markers
	for _, rel := range tc.relationships {
		columns = append(columns, relationshipDefToMap(rel))
	}

	// Convert indexes to maps
	indexes := make([]map[string]any, 0, len(tc.indexes))
	for _, idx := range tc.indexes {
		indexes = append(indexes, indexDefToMap(idx))
	}

	_ = result.Set("columns", columns)
	_ = result.Set("indexes", indexes)
	if tc.docs != "" {
		_ = result.Set("docs", tc.docs)
	}
	if tc.deprecated != "" {
		_ = result.Set("deprecated", tc.deprecated)
	}

	// Table-level metadata for x-db extensions
	if tc.auditable {
		_ = result.Set("auditable", true)
	}
	if len(tc.sortBy) > 0 {
		_ = result.Set("sort_by", tc.sortBy)
	}
	if len(tc.searchable) > 0 {
		_ = result.Set("searchable", tc.searchable)
	}
	if len(tc.filterable) > 0 {
		_ = result.Set("filterable", tc.filterable)
	}

	return result
}

// Helper functions for converting defs to maps (used by both TableBuilder and TableChain)

func columnDefToMap(col *ColumnDef) map[string]any {
	slog.Debug("columnDefToMap: converting column",
		"name", col.Name,
		"type", col.Type,
		"default", col.Default)

	m := map[string]any{
		"name":     col.Name,
		"type":     col.Type,
		"nullable": col.Nullable,
	}

	if len(col.TypeArgs) > 0 {
		m["type_args"] = col.TypeArgs
	}
	if col.Unique {
		m["unique"] = true
	}
	if col.PrimaryKey {
		m["primary_key"] = true
	}
	if col.Default != nil {
		slog.Debug("columnDefToMap: adding default to map",
			"column", col.Name,
			"default", col.Default)
		m["default"] = col.Default
	} else {
		slog.Debug("columnDefToMap: NO default for column",
			"column", col.Name)
	}
	if col.Backfill != nil {
		m["backfill"] = col.Backfill
	}
	if col.Format != "" {
		m["format"] = col.Format
	}
	if col.Pattern != "" {
		m["pattern"] = col.Pattern
	}
	if col.Min != nil {
		m["min"] = *col.Min
	}
	if col.Max != nil {
		m["max"] = *col.Max
	}
	if col.Docs != "" {
		m["docs"] = col.Docs
	}
	if col.Deprecated != "" {
		m["deprecated"] = col.Deprecated
	}
	if col.Hidden {
		m["x_hidden"] = true
	}
	if col.XRef != "" {
		m["x_ref"] = col.XRef
	}

	if col.Reference != nil {
		m["reference"] = map[string]any{
			"table":     col.Reference.Table,
			"column":    col.Reference.Column,
			"on_delete": col.Reference.OnDelete,
			"on_update": col.Reference.OnUpdate,
		}
	}

	if col.Computed != nil {
		m["computed"] = col.Computed
	}

	return m
}

func indexDefToMap(idx *IndexDef) map[string]any {
	cols := make([]any, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = c
	}

	m := map[string]any{
		"columns": cols,
		"unique":  idx.Unique,
	}
	if idx.Name != "" {
		m["name"] = idx.Name
	}
	if idx.Where != "" {
		m["where"] = idx.Where
	}
	return m
}

func relationshipDefToMap(rel *RelationshipDef) map[string]any {
	switch rel.Type {
	case "many_to_many":
		relMap := map[string]any{
			"type":   "many_to_many",
			"target": rel.Target,
		}
		if rel.As != "" {
			relMap["as"] = rel.As
		}
		return map[string]any{
			"_type":        "relationship",
			"relationship": relMap,
		}
	case "polymorphic":
		return map[string]any{
			"_type": "polymorphic",
			"polymorphic": map[string]any{
				"targets": rel.Targets,
				"as":      rel.As,
			},
		}
	default:
		return map[string]any{}
	}
}
