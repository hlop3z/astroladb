package runtime

import (
	"reflect"
	"strings"

	"github.com/dop251/goja"
	"github.com/hlop3z/astroladb/internal/ast"
)

// BindingsContext provides context for DSL bindings during evaluation.
type BindingsContext struct {
	vm        *goja.Runtime
	namespace string
	tables    []*ast.TableDef
}

// NewBindingsContext creates a new bindings context.
func NewBindingsContext(vm *goja.Runtime, namespace string) *BindingsContext {
	return &BindingsContext{
		vm:        vm,
		namespace: namespace,
		tables:    make([]*ast.TableDef, 0),
	}
}

// BindSchema binds the schema() DSL function for multi-table schema files.
// Usage: schema("namespace", s => { s.table("name", t => {...}) })
func (s *Sandbox) BindSchema() {
	s.vm.Set("schema", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(s.vm.ToValue("schema() requires namespace and builder function"))
		}

		namespace := call.Arguments[0].String()
		builderFn, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(s.vm.ToValue("schema() second argument must be a function"))
		}

		// Create schema builder object
		schemaObj := s.createSchemaObject(namespace)

		// Call the builder function
		_, err := builderFn(goja.Undefined(), schemaObj)
		if err != nil {
			panic(s.vm.ToValue("error in schema builder: " + err.Error()))
		}

		return goja.Undefined()
	})
}

// createSchemaObject creates the JavaScript object for the schema builder.
func (s *Sandbox) createSchemaObject(namespace string) *goja.Object {
	obj := s.vm.NewObject()

	// table(name, fn) - defines a table in this namespace
	_ = obj.Set("table", func(name string, fn goja.Value) *goja.Object {
		builderFn, ok := goja.AssertFunction(fn)
		if !ok {
			panic(s.vm.ToValue("table() second argument must be a function"))
		}

		// Create table builder
		tb := NewTableBuilder(s.vm)

		// Call the builder function with the table builder
		tbObj := tb.ToObject()
		_, err := builderFn(goja.Undefined(), tbObj)
		if err != nil {
			panic(s.vm.ToValue("error in table builder: " + err.Error()))
		}

		// Convert to table definition
		result := tb.ToResult()
		defObj := result.Export()

		tableDef := s.parseTableDef(namespace, name, defObj)
		if tableDef != nil {
			s.tables = append(s.tables, tableDef)

			// Register with the registry if available
			if s.registry != nil {
				_ = s.registry.Register(namespace, name, tableDef)
			}
		}

		return result.(*goja.Object)
	})

	return obj
}

// BindMigration binds the migration() DSL function for migration files.
// Usage: migration({ up(m) { ... }, down(m) { ... } })
func (s *Sandbox) BindMigration() {
	s.vm.Set("migration", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(s.vm.ToValue("migration() requires a definition object"))
		}

		// Expect an object with up() and down() methods
		defObj, ok := call.Arguments[0].(*goja.Object)
		if !ok {
			panic(s.vm.ToValue("migration() argument must be an object with up() and down() methods"))
		}

		// Get the up() function
		upVal := defObj.Get("up")
		if upVal == nil || goja.IsUndefined(upVal) || goja.IsNull(upVal) {
			panic(s.vm.ToValue("migration definition must have an up() method"))
		}

		upFn, ok := goja.AssertFunction(upVal)
		if !ok {
			panic(s.vm.ToValue("migration up property must be a function"))
		}

		// Create migration builder object
		migrationObj := s.createMigrationObject()

		// Call the up() function with the migration builder
		_, err := upFn(goja.Undefined(), migrationObj)
		if err != nil {
			panic(s.vm.ToValue("error in migration up() function: " + err.Error()))
		}

		// Note: down() is not executed here - it's only called during rollback
		// The down() function is accessed separately when needed

		return goja.Undefined()
	})
}

// createMigrationObject creates the JavaScript object for the migration builder.
// Operations are stored directly in the sandbox's operations slice.
func (s *Sandbox) createMigrationObject() *goja.Object {
	obj := s.vm.NewObject()

	// create_table(ref, fn)
	_ = obj.Set("create_table", func(ref string, fn goja.Value) {
		builderFn, ok := goja.AssertFunction(fn)
		if !ok {
			panic(s.vm.ToValue("create_table() second argument must be a function"))
		}

		ns, table := parseRef(ref)
		tb := NewTableBuilder(s.vm)
		tbObj := tb.ToObject()

		_, err := builderFn(goja.Undefined(), tbObj)
		if err != nil {
			panic(s.vm.ToValue("error in table builder: " + err.Error()))
		}

		result := tb.ToResult().Export()
		tableDef := s.parseTableDef(ns, table, result)

		if tableDef != nil {
			op := &ast.CreateTable{
				TableOp:     ast.TableOp{Namespace: ns, Name: table},
				Columns:     tableDef.Columns,
				Indexes:     tableDef.Indexes,
				ForeignKeys: make([]*ast.ForeignKeyDef, 0),
			}
			s.operations = append(s.operations, op)
		}
	})

	// drop_table(ref)
	_ = obj.Set("drop_table", func(ref string) {
		ns, table := parseRef(ref)
		s.operations = append(s.operations, &ast.DropTable{
			TableOp: ast.TableOp{Namespace: ns, Name: table},
		})
	})

	// rename_table(oldRef, newName)
	_ = obj.Set("rename_table", func(oldRef, newName string) {
		ns, oldTable := parseRef(oldRef)
		s.operations = append(s.operations, &ast.RenameTable{
			Namespace: ns,
			OldName:   oldTable,
			NewName:   newName,
		})
	})

	// add_column(ref, fn)
	_ = obj.Set("add_column", func(ref string, fn goja.Value) {
		builderFn, ok := goja.AssertFunction(fn)
		if !ok {
			panic(s.vm.ToValue("add_column() second argument must be a function"))
		}

		ns, table := parseRef(ref)

		// Create a column builder wrapper
		col := make(map[string]any)
		colBuilder := s.createColumnBuilderObject(col)

		// Call the function which should return a column builder
		result, err := builderFn(goja.Undefined(), colBuilder)
		if err != nil {
			panic(s.vm.ToValue("error in column builder: " + err.Error()))
		}

		// The function might return the column builder or call methods on it
		// Try to extract column data from result if it's an object
		if result != nil && !goja.IsUndefined(result) && !goja.IsNull(result) {
			if resultObj, ok := result.(*goja.Object); ok {
				exportedResult := resultObj.Export()
				if m, ok := exportedResult.(map[string]any); ok {
					// Merge result into col, but skip function values (methods)
					for k, v := range m {
						// Skip functions - they are methods from the chain object
						if reflect.TypeOf(v) != nil && reflect.TypeOf(v).Kind() == reflect.Func {
							continue
						}
						col[k] = v
					}
				}
			}
		}

		colDef := s.parseColumnDef(col)
		if colDef != nil {
			s.operations = append(s.operations, &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: ns, Table_: table},
				Column:   colDef,
			})
		}
	})

	// drop_column(ref, column)
	_ = obj.Set("drop_column", func(ref, column string) {
		ns, table := parseRef(ref)
		s.operations = append(s.operations, &ast.DropColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     column,
		})
	})

	// rename_column(ref, oldName, newName)
	_ = obj.Set("rename_column", func(ref, oldName, newName string) {
		ns, table := parseRef(ref)
		s.operations = append(s.operations, &ast.RenameColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			OldName:  oldName,
			NewName:  newName,
		})
	})

	// create_index(ref, columns, opts?)
	_ = obj.Set("create_index", func(ref string, columns []string, opts ...map[string]any) {
		ns, table := parseRef(ref)
		op := &ast.CreateIndex{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Columns:  columns,
		}
		if len(opts) > 0 {
			if unique, ok := opts[0]["unique"].(bool); ok {
				op.Unique = unique
			}
			if name, ok := opts[0]["name"].(string); ok {
				op.Name = name
			}
		}
		// Auto-generate index name if not provided (required for rollback)
		if op.Name == "" {
			sqlTable := table
			if ns != "" {
				sqlTable = ns + "_" + table
			}
			op.Name = "idx_" + sqlTable + "_" + strings.Join(columns, "_")
		}
		s.operations = append(s.operations, op)
	})

	// drop_index(name)
	_ = obj.Set("drop_index", func(name string) {
		s.operations = append(s.operations, &ast.DropIndex{
			Name: name,
		})
	})

	// sql(sql, downSQL?)
	_ = obj.Set("sql", func(sql string, down ...string) {
		op := &ast.RawSQL{
			SQL: sql,
		}
		// Note: downSQL is captured but stored separately in migration metadata
		// The RawSQL operation itself just stores the up SQL
		s.operations = append(s.operations, op)
	})

	// alter_column(ref, column, fn)
	_ = obj.Set("alter_column", func(ref, column string, fn goja.Value) {
		builderFn, ok := goja.AssertFunction(fn)
		if !ok {
			panic(s.vm.ToValue("alter_column() third argument must be a function"))
		}

		ns, table := parseRef(ref)

		// Create alter column builder
		alterOp := &ast.AlterColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     column,
		}
		alterBuilder := s.createAlterColumnBuilderObject(alterOp)

		_, err := builderFn(goja.Undefined(), alterBuilder)
		if err != nil {
			panic(s.vm.ToValue("error in alter_column builder: " + err.Error()))
		}

		s.operations = append(s.operations, alterOp)
	})

	// add_foreign_key(ref, columns, refTable, refColumns, opts?)
	_ = obj.Set("add_foreign_key", func(ref string, columns []string, refTable string, refColumns []string, opts ...map[string]any) {
		ns, table := parseRef(ref)
		refNs, refTbl := parseRef(refTable)
		sqlRefTable := refTbl
		if refNs != "" {
			sqlRefTable = refNs + "_" + refTbl
		}

		op := &ast.AddForeignKey{
			TableRef:   ast.TableRef{Namespace: ns, Table_: table},
			Columns:    columns,
			RefTable:   sqlRefTable,
			RefColumns: refColumns,
		}

		if len(opts) > 0 {
			if name, ok := opts[0]["name"].(string); ok {
				op.Name = name
			}
			if onDelete, ok := opts[0]["on_delete"].(string); ok {
				op.OnDelete = onDelete
			}
			if onUpdate, ok := opts[0]["on_update"].(string); ok {
				op.OnUpdate = onUpdate
			}
		}

		s.operations = append(s.operations, op)
	})

	// drop_foreign_key(ref, constraintName)
	_ = obj.Set("drop_foreign_key", func(ref, constraintName string) {
		ns, table := parseRef(ref)
		s.operations = append(s.operations, &ast.DropForeignKey{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     constraintName,
		})
	})

	// add_check(ref, name, expression)
	_ = obj.Set("add_check", func(ref, name, expression string) {
		ns, table := parseRef(ref)
		s.operations = append(s.operations, &ast.AddCheck{
			TableRef:   ast.TableRef{Namespace: ns, Table_: table},
			Name:       name,
			Expression: expression,
		})
	})

	// drop_check(ref, name)
	_ = obj.Set("drop_check", func(ref, name string) {
		ns, table := parseRef(ref)
		s.operations = append(s.operations, &ast.DropCheck{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     name,
		})
	})

	return obj
}

// createAlterColumnBuilderObject creates the JS object for alter_column modifications.
func (s *Sandbox) createAlterColumnBuilderObject(op *ast.AlterColumn) *goja.Object {
	obj := s.vm.NewObject()

	// set_type(typeName, ...args)
	_ = obj.Set("set_type", func(typeName string, args ...any) *goja.Object {
		op.NewType = typeName
		op.NewTypeArgs = args
		return obj
	})

	// set_nullable()
	_ = obj.Set("set_nullable", func() *goja.Object {
		nullable := true
		op.SetNullable = &nullable
		return obj
	})

	// set_not_null()
	_ = obj.Set("set_not_null", func() *goja.Object {
		nullable := false
		op.SetNullable = &nullable
		return obj
	})

	// set_default(value)
	_ = obj.Set("set_default", func(value any) *goja.Object {
		op.SetDefault = value
		return obj
	})

	// set_server_default(expr)
	_ = obj.Set("set_server_default", func(expr string) *goja.Object {
		op.ServerDefault = expr
		return obj
	})

	// drop_default()
	_ = obj.Set("drop_default", func() *goja.Object {
		op.DropDefault = true
		return obj
	})

	return obj
}

// createColumnBuilderObject creates a column builder for add_column migrations.
func (s *Sandbox) createColumnBuilderObject(col map[string]any) *goja.Object {
	obj := s.vm.NewObject()

	// Type methods that set name and type
	typeMethod := func(typeName string, hasLength bool) func(args ...goja.Value) *goja.Object {
		return func(args ...goja.Value) *goja.Object {
			if len(args) > 0 {
				col["name"] = args[0].String()
			}
			col["type"] = typeName
			col["nullable"] = false

			if hasLength && len(args) > 1 {
				col["type_args"] = []any{int(args[1].ToInteger())}
			}
			return s.createColumnChainObject(col)
		}
	}

	_ = obj.Set("string", typeMethod("string", true))
	_ = obj.Set("text", typeMethod("text", false))
	_ = obj.Set("integer", typeMethod("integer", false))
	_ = obj.Set("float", typeMethod("float", false))
	_ = obj.Set("boolean", typeMethod("boolean", false))
	_ = obj.Set("date", typeMethod("date", false))
	_ = obj.Set("time", typeMethod("time", false))
	_ = obj.Set("datetime", typeMethod("datetime", false))
	_ = obj.Set("uuid", typeMethod("uuid", false))
	_ = obj.Set("json", typeMethod("json", false))
	_ = obj.Set("base64", typeMethod("base64", false))

	// decimal(name, precision, scale)
	_ = obj.Set("decimal", func(name string, precision, scale int) *goja.Object {
		col["name"] = name
		col["type"] = "decimal"
		col["type_args"] = []any{precision, scale}
		col["nullable"] = false
		return s.createColumnChainObject(col)
	})

	// enum(name, values)
	_ = obj.Set("enum", func(name string, values []string) *goja.Object {
		col["name"] = name
		col["type"] = "enum"
		col["type_args"] = []any{values}
		col["nullable"] = false
		return s.createColumnChainObject(col)
	})

	return obj
}

// createColumnChainObject creates the chainable methods for column definitions.
func (s *Sandbox) createColumnChainObject(col map[string]any) *goja.Object {
	obj := s.vm.NewObject()

	_ = obj.Set("nullable", func() *goja.Object {
		col["nullable"] = true
		return obj
	})

	// optional() is an alias for nullable() - matches DSL convention
	_ = obj.Set("optional", func() *goja.Object {
		col["nullable"] = true
		return obj
	})

	_ = obj.Set("unique", func() *goja.Object {
		col["unique"] = true
		return obj
	})

	_ = obj.Set("default", func(value any) *goja.Object {
		col["default"] = value
		return obj
	})

	_ = obj.Set("backfill", func(value any) *goja.Object {
		col["backfill"] = value
		return obj
	})

	_ = obj.Set("min", func(n int) *goja.Object {
		col["min"] = float64(n)
		return obj
	})

	_ = obj.Set("max", func(n int) *goja.Object {
		col["max"] = float64(n)
		return obj
	})

	_ = obj.Set("pattern", func(regex string) *goja.Object {
		col["pattern"] = regex
		return obj
	})

	_ = obj.Set("format", func(f goja.Value) *goja.Object {
		if f == nil || goja.IsUndefined(f) || goja.IsNull(f) {
			return obj
		}
		if fObj, ok := f.(*goja.Object); ok {
			if format := fObj.Get("format"); format != nil && !goja.IsUndefined(format) {
				col["format"] = format.String()
			}
			if _, hasPattern := col["pattern"]; !hasPattern {
				if pattern := fObj.Get("pattern"); pattern != nil && !goja.IsUndefined(pattern) {
					if patternStr := pattern.String(); patternStr != "" {
						col["pattern"] = patternStr
					}
				}
			}
		} else if str, ok := f.Export().(string); ok {
			col["format"] = str
		}
		return obj
	})

	_ = obj.Set("docs", func(description string) *goja.Object {
		col["docs"] = description
		return obj
	})

	_ = obj.Set("deprecated", func(reason string) *goja.Object {
		col["deprecated"] = reason
		return obj
	})

	// Export the column data when the builder is used as a return value
	_ = obj.Set("_data", col)

	return obj
}

// parseRef parses a table reference into namespace and table name.
func parseRef(ref string) (namespace, table string) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			return ref[:i], ref[i+1:]
		}
	}
	return "", ref
}

// BindSQL creates the sql() helper function.
// sql("expr") returns {_type: "sql_expr", expr: "..."}
func BindSQL(vm *goja.Runtime) {
	vm.Set("sql", func(expr string) map[string]any {
		return map[string]any{
			"_type": "sql_expr",
			"expr":  expr,
		}
	})
}
