package builder

import (
	"strings"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/strutil"
)

// -----------------------------------------------------------------------------
// TableBuilder
// -----------------------------------------------------------------------------

// TableBuilder provides the JavaScript interface for building table definitions.
// It accumulates column definitions, indexes, and other table properties.
type TableBuilder struct {
	vm *goja.Runtime

	Columns       []*ColumnDef
	Indexes       []*IndexDef
	Relationships []*RelationshipDef
	Docs          string
	Deprecated    string

	// Schema-only metadata (not used in migrations)
	Auditable  bool
	SortBy     []string
	Searchable []string
	Filterable []string
}

// NewTableBuilder creates a new table builder for JavaScript.
func NewTableBuilder(vm *goja.Runtime) *TableBuilder {
	return &TableBuilder{
		vm:            vm,
		Columns:       make([]*ColumnDef, 0),
		Indexes:       make([]*IndexDef, 0),
		Relationships: make([]*RelationshipDef, 0),
	}
}

// NewTableBuilderWithColumns creates a TableBuilder with pre-existing columns and indexes.
// Used by the schema path where columns are defined as object keys.
func NewTableBuilderWithColumns(vm *goja.Runtime, columns []*ColumnDef, indexes []*IndexDef) *TableBuilder {
	return &TableBuilder{
		vm:            vm,
		Columns:       columns,
		Indexes:       indexes,
		Relationships: make([]*RelationshipDef, 0),
	}
}

// -----------------------------------------------------------------------------
// Functional options for column configuration
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Shared validation helpers (used by both ColBuilder and TableBuilder)
// -----------------------------------------------------------------------------

// validateRef panics if ref is empty or missing namespace.
func validateRef(vm *goja.Runtime, ref, method string) {
	if ref == "" {
		panic(vm.ToValue(alerr.NewMissingReferenceError(method).Error()))
	}
	if !alerr.HasNamespace(ref) {
		panic(vm.ToValue(alerr.NewMissingNamespaceError(ref).Error()))
	}
}

// validateStringLength panics if length is invalid.
// For table context (name != ""), uses alerr for semantic error with column name.
// For col context (name == ""), uses BuilderError.
func validateStringLength(vm *goja.Runtime, length int, name string) {
	if length <= 0 {
		if name != "" {
			panic(vm.ToValue(alerr.NewMissingLengthError(name).Error()))
		}
		panic(vm.ToValue(ErrMsgStringRequiresLength.String()))
	}
}

// validateDecimalArgs panics if precision/scale are invalid.
func validateDecimalArgs(vm *goja.Runtime, precision, scale int, isTable bool) {
	if precision <= 0 || scale < 0 {
		panic(vm.ToValue(ErrDecimalRequiresArgs(isTable).String()))
	}
}

// validateEnumValues panics if values slice is empty.
func validateEnumValues(vm *goja.Runtime, values []string) {
	if len(values) == 0 {
		panic(vm.ToValue(ErrMsgEnumRequiresValues.String()))
	}
}

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
func withIndex() ColOpt           { return func(c *ColumnDef) { c.Index = true } }
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
	tb.Columns = append(tb.Columns, col)
	return tb.columnBuilder(col)
}

// addTimestamps appends created_at and updated_at datetime columns with NOW() defaults.
func (tb *TableBuilder) addTimestamps() {
	tb.Columns = append(tb.Columns, &ColumnDef{
		Name:    "created_at",
		Type:    "datetime",
		Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
	})
	tb.Columns = append(tb.Columns, &ColumnDef{
		Name:    "updated_at",
		Type:    "datetime",
		Default: map[string]any{"_type": "sql_expr", "expr": "NOW()"},
	})
}

// addSoftDelete appends a nullable deleted_at datetime column.
func (tb *TableBuilder) addSoftDelete() {
	tb.Columns = append(tb.Columns, &ColumnDef{
		Name:     "deleted_at",
		Type:     "datetime",
		Nullable: true,
	})
}

// addSortable appends an integer position column with default 0.
func (tb *TableBuilder) addSortable() {
	tb.Columns = append(tb.Columns, &ColumnDef{
		Name:    "position",
		Type:    "integer",
		Default: 0,
	})
}

// resolveUniqueColumns resolves column names, appending _id suffix for relationship aliases.
func (tb *TableBuilder) resolveUniqueColumns(columns []string) []string {
	existingCols := make(map[string]bool)
	for _, c := range tb.Columns {
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
	return cols
}

// addPolymorphicColumns appends _type (string) and _id (uuid) columns for a polymorphic relationship.
func (tb *TableBuilder) addPolymorphicColumns(as string) {
	tb.Columns = append(tb.Columns, &ColumnDef{
		Name:     as + "_type",
		Type:     "string",
		TypeArgs: []any{100},
	})
	tb.Columns = append(tb.Columns, &ColumnDef{
		Name: as + "_id",
		Type: "uuid",
	})
}

// registerCommonHelpers registers methods shared between toBaseObject and ToChainableObject.
// When chainable is true, methods return obj for chaining; when false, they return void.
func (tb *TableBuilder) registerCommonHelpers(obj *goja.Object, chainable bool) {
	if chainable {
		_ = obj.Set("timestamps", func() *goja.Object { tb.addTimestamps(); return obj })
		_ = obj.Set("soft_delete", func() *goja.Object { tb.addSoftDelete(); return obj })
		_ = obj.Set("sortable", func() *goja.Object { tb.addSortable(); return obj })
		_ = obj.Set("index", func(columns ...string) *goja.Object {
			tb.Indexes = append(tb.Indexes, &IndexDef{Columns: columns})
			return obj
		})
		_ = obj.Set("unique", func(columns ...string) *goja.Object {
			tb.Indexes = append(tb.Indexes, &IndexDef{Columns: tb.resolveUniqueColumns(columns), Unique: true})
			return obj
		})
		_ = obj.Set("docs", func(description string) *goja.Object { tb.Docs = description; return obj })
		_ = obj.Set("deprecated", func(reason string) *goja.Object { tb.Deprecated = reason; return obj })
	} else {
		_ = obj.Set("timestamps", func() { tb.addTimestamps() })
		_ = obj.Set("soft_delete", func() { tb.addSoftDelete() })
		_ = obj.Set("sortable", func() { tb.addSortable() })
		_ = obj.Set("index", func(columns ...string) {
			tb.Indexes = append(tb.Indexes, &IndexDef{Columns: columns})
		})
		_ = obj.Set("unique", func(columns ...string) {
			tb.Indexes = append(tb.Indexes, &IndexDef{Columns: tb.resolveUniqueColumns(columns), Unique: true})
		})
		_ = obj.Set("docs", func(description string) { tb.Docs = description })
		_ = obj.Set("deprecated", func(reason string) { tb.Deprecated = reason })
	}
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
		validateStringLength(tb.vm, length, name)
		return tb.addColumn(name, "string", withLength(length))
	})

	// Simple types (no extra args) - each takes (name string)
	for _, t := range simpleTypes {
		typeName := t
		_ = obj.Set(typeName, func(name string) *goja.Object { return tb.addColumn(name, typeName) })
	}

	// decimal(name, precision, scale)
	_ = obj.Set("decimal", func(name string, precision, scale int) *goja.Object {
		validateDecimalArgs(tb.vm, precision, scale, true)
		return tb.addColumn(name, "decimal", withArgs(precision, scale))
	})

	// enum(name, values) - values is required and must not be empty
	_ = obj.Set("enum", func(name string, values []string) *goja.Object {
		validateEnumValues(tb.vm, values)
		return tb.addColumn(name, "enum", withArgs(values))
	})

	tb.registerCommonHelpers(obj, false)

	// belongs_to(ref) - Foreign key relationship with chaining API
	// Examples:
	//   belongs_to("auth.user")                                    -> user_id
	//   belongs_to("auth.user").as("author")                       -> author_id
	//   belongs_to("auth.user").optional()                         -> user_id, nullable
	//   belongs_to("auth.user").as("author").optional().on_delete("cascade")
	_ = obj.Set("belongs_to", func(ref string) *goja.Object {
		validateRef(tb.vm, ref, "t.belongs_to")

		// Default column name from table reference
		colName := strutil.ExtractTableName(ref) + "_id"

		col := &ColumnDef{
			Name: colName,
			Type: "uuid",
			Reference: &RefDef{
				Table:  ref,
				Column: "id",
			},
			XRef: ref,
		}
		tb.Columns = append(tb.Columns, col)

		// Note: Index is NOT created here for migrations.
		// In migrations, indexes are handled as separate create_index operations.
		// The writeCreateTable function generates these explicitly.
		// Creating the index here would cause duplicates when:
		// 1. belongs_to adds index to CreateTable.Indexes
		// 2. Separate create_index operation is also in the migration
		// 3. Both get replayed, causing "index already exists" errors

		return tb.relationshipBuilder(ref, col, nil)
	})

	// one_to_one(ref) - Unique foreign key relationship with chaining API
	// Same as belongs_to but creates unique constraint
	_ = obj.Set("one_to_one", func(ref string) *goja.Object {
		validateRef(tb.vm, ref, "t.one_to_one")

		colName := strutil.ExtractTableName(ref) + "_id"

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
		tb.Columns = append(tb.Columns, col)

		// Note: Index is NOT created here for migrations (same as belongs_to).
		// See belongs_to comment above for explanation.

		return tb.relationshipBuilder(ref, col, nil)
	})

	// many_to_many(ref, opts)
	_ = obj.Set("many_to_many", func(ref string, opts ...map[string]any) {
		validateRef(tb.vm, ref, "many_to_many")
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
		tb.Relationships = append(tb.Relationships, rel)
	})

	// belongs_to_any(refs, opts)
	_ = obj.Set("belongs_to_any", func(refs []string, opts ...map[string]any) {
		var as string
		if len(opts) > 0 {
			if v, ok := opts[0]["as"].(string); ok {
				as = v
			}
		}
		if as == "" {
			as = "polymorphic"
		}
		tb.addPolymorphicColumns(as)
		// Note: Index is NOT created here for migrations (same as belongs_to).
		// See belongs_to comment above for explanation.
		tb.Relationships = append(tb.Relationships, &RelationshipDef{
			Type:    "polymorphic",
			Targets: refs,
			As:      as,
		})
	})

	return obj
}

// ToChainableObject returns a JS object with chainable table-level methods.
// Used by the schema path after columns are defined as object keys.
func (tb *TableBuilder) ToChainableObject() *goja.Object {
	obj := tb.vm.NewObject()

	tb.registerCommonHelpers(obj, true)

	// many_to_many(ref)
	_ = obj.Set("many_to_many", func(ref string) *goja.Object {
		validateRef(tb.vm, ref, "many_to_many")
		tb.Relationships = append(tb.Relationships, &RelationshipDef{
			Type:   "many_to_many",
			Target: ref,
		})
		return obj
	})

	// junction(refs...) - marks table as M2M junction
	_ = obj.Set("junction", func(refs ...string) *goja.Object {
		if len(refs) != 0 && len(refs) != 2 {
			panic(tb.vm.ToValue("junction() requires 0 or 2 parameters, got " + string(rune(len(refs)))))
		}

		rel := &RelationshipDef{
			Type: "junction",
		}
		if len(refs) == 2 {
			rel.JunctionSource = refs[0]
			rel.JunctionTarget = refs[1]
		}
		tb.Relationships = append(tb.Relationships, rel)
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
		tb.addPolymorphicColumns(as)
		tb.Indexes = append(tb.Indexes, &IndexDef{
			Columns: []string{as + "_type", as + "_id"},
		})
		tb.Relationships = append(tb.Relationships, &RelationshipDef{
			Type:    "polymorphic",
			Targets: refs,
			As:      as,
		})
		return obj
	})

	// auditable() - adds created_by and updated_by columns
	_ = obj.Set("auditable", func() *goja.Object {
		tb.Auditable = true
		tb.Columns = append(tb.Columns, &ColumnDef{
			Name:     "created_by",
			Type:     "uuid",
			Nullable: true,
		})
		tb.Columns = append(tb.Columns, &ColumnDef{
			Name:     "updated_by",
			Type:     "uuid",
			Nullable: true,
		})
		return obj
	})

	// sort_by(...columns) - default ordering
	_ = obj.Set("sort_by", func(columns ...string) *goja.Object {
		tb.SortBy = columns
		return obj
	})

	// searchable(...columns) - columns for fulltext search
	_ = obj.Set("searchable", func(columns ...string) *goja.Object {
		tb.Searchable = columns
		return obj
	})

	// filterable(...columns) - columns allowed in WHERE clauses
	_ = obj.Set("filterable", func(columns ...string) *goja.Object {
		tb.Filterable = columns
		return obj
	})

	// Store result data for extraction
	_ = obj.Set("_getResult", func() goja.Value {
		return tb.ToResult()
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

	// nullable() - alias for optional() (used in migrations)
	_ = obj.Set("nullable", func() *goja.Object {
		col.Nullable = true
		return obj
	})

	// unique()
	_ = obj.Set("unique", func() *goja.Object {
		col.Unique = true
		return obj
	})

	// index() - marks column for non-unique index creation
	_ = obj.Set("index", func() *goja.Object {
		col.Index = true
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

	// read_only() - marks column as read-only (computed, generated, server-set)
	_ = obj.Set("read_only", func() *goja.Object {
		col.ReadOnly = true
		return obj
	})

	// write_only() - marks column as write-only (passwords, secrets, tokens)
	_ = obj.Set("write_only", func() *goja.Object {
		col.WriteOnly = true
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
// Composes createChainBuilder (optional, on_delete, on_update, docs, etc.) with an additional "as" method.
func (tb *TableBuilder) relationshipBuilder(ref string, col *ColumnDef, idx *IndexDef) *goja.Object {
	obj := createChainBuilder(tb.vm, col, true)

	// as(alias) - Set custom column name
	_ = obj.Set("as", func(alias string) *goja.Object {
		newName := alias + "_id"
		col.Name = newName
		if idx != nil {
			idx.Columns = []string{newName}
		}
		return obj
	})

	return obj
}

// ToResult converts the builder state to a result object for JavaScript.
// Stashes a reference to the TableBuilder for direct extraction by registerTableFunc().
func (tb *TableBuilder) ToResult() goja.Value {
	result := tb.vm.NewObject()
	_ = result.Set("_tableBuilder", tb)
	return result
}
