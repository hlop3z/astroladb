package builder

import (
	"strings"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
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
		if precision <= 0 || scale < 0 {
			panic(tb.vm.ToValue(ErrMsgTableDecimalRequiresArgs.String()))
		}
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
	})

	// soft_delete() - adds deleted_at
	_ = obj.Set("soft_delete", func() {
		tb.Columns = append(tb.Columns, &ColumnDef{
			Name:     "deleted_at",
			Type:     "datetime",
			Nullable: true,
		})
	})

	// sortable() - adds position
	_ = obj.Set("sortable", func() {
		tb.Columns = append(tb.Columns, &ColumnDef{
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
		// Validate: reference cannot be empty
		if ref == "" {
			panic(tb.vm.ToValue(alerr.NewMissingReferenceError("t.belongs_to").Error()))
		}
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
		tb.Columns = append(tb.Columns, col)

		// Note: Index is NOT created here for migrations (same as belongs_to).
		// See belongs_to comment above for explanation.

		return tb.relationshipBuilder(ref, col, nil)
	})

	// many_to_many(ref, opts)
	_ = obj.Set("many_to_many", func(ref string, opts ...map[string]any) {
		// Validate: reference cannot be empty
		if ref == "" {
			panic(tb.vm.ToValue(alerr.NewMissingReferenceError("many_to_many").Error()))
		}
		// Validate: reference must have namespace
		if !alerr.HasNamespace(ref) {
			panic(tb.vm.ToValue(alerr.NewMissingNamespaceError(ref).Error()))
		}
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
		tb.Columns = append(tb.Columns, &ColumnDef{
			Name:     typeCol,
			Type:     "string",
			TypeArgs: []any{100},
		})

		// Add id column
		tb.Columns = append(tb.Columns, &ColumnDef{
			Name: idCol,
			Type: "uuid",
		})

		// Note: Index is NOT created here for migrations (same as belongs_to).
		// See belongs_to comment above for explanation.

		// Store polymorphic relationship metadata
		tb.Relationships = append(tb.Relationships, &RelationshipDef{
			Type:    "polymorphic",
			Targets: refs,
			As:      as,
		})
	})

	// index(columns...)
	_ = obj.Set("index", func(columns ...string) {
		tb.Indexes = append(tb.Indexes, &IndexDef{
			Columns: columns,
		})
	})

	// unique(columns...) - composite uniqueness constraint
	// Clean API for relationship tables: unique("follower", "follows")
	// Automatically appends _id suffix for relationship aliases
	_ = obj.Set("unique", func(columns ...string) {
		// Build a set of existing column names
		existingCols := make(map[string]bool)
		for _, c := range tb.Columns {
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
		tb.Indexes = append(tb.Indexes, &IndexDef{
			Columns: cols,
			Unique:  true,
		})
	})

	// docs(description)
	_ = obj.Set("docs", func(description string) {
		tb.Docs = description
	})

	// deprecated(reason)
	_ = obj.Set("deprecated", func(reason string) {
		tb.Deprecated = reason
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
// Supports: .as(alias), .optional(), .on_delete(action), .on_update(action)
func (tb *TableBuilder) relationshipBuilder(ref string, col *ColumnDef, idx *IndexDef) *goja.Object {
	obj := tb.vm.NewObject()

	// as(alias) - Set custom column name
	_ = obj.Set("as", func(alias string) *goja.Object {
		newName := alias + "_id"
		col.Name = newName
		// Update index columns if index exists (may be nil in migrations)
		if idx != nil {
			idx.Columns = []string{newName}
		}
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

	// read_only() - marks column as read-only
	_ = obj.Set("read_only", func() *goja.Object {
		col.ReadOnly = true
		return obj
	})

	// write_only() - marks column as write-only
	_ = obj.Set("write_only", func() *goja.Object {
		col.WriteOnly = true
		return obj
	})

	return obj
}

// ToResult converts the builder state to a result object for JavaScript.
// Converts typed structs back to maps for sandbox.go consumption.
func (tb *TableBuilder) ToResult() goja.Value {
	result := tb.vm.NewObject()

	// Convert columns to maps (using standalone functions to avoid duplication)
	columns := make([]map[string]any, 0, len(tb.Columns)+len(tb.Relationships))
	for _, col := range tb.Columns {
		columns = append(columns, columnDefToMap(col))
	}

	// Add relationship markers (many_to_many, polymorphic)
	for _, rel := range tb.Relationships {
		columns = append(columns, relationshipDefToMap(rel))
	}

	// Convert indexes to maps
	indexes := make([]map[string]any, 0, len(tb.Indexes))
	for _, idx := range tb.Indexes {
		indexes = append(indexes, indexDefToMap(idx))
	}

	_ = result.Set("columns", columns)
	_ = result.Set("indexes", indexes)
	if tb.Docs != "" {
		_ = result.Set("docs", tb.Docs)
	}
	if tb.Deprecated != "" {
		_ = result.Set("deprecated", tb.Deprecated)
	}
	return result
}

// extractTableName extracts the table name from a reference.
// With singular table convention: "auth.user" -> "user", "core.order" -> "order"
