package runtime

import (
	"fmt"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/runtime/builder"
	"github.com/hlop3z/astroladb/internal/runtime/schema"
	"github.com/hlop3z/astroladb/internal/strutil"
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

// MigrationMeta stores metadata parsed from the migration DSL definition.
type MigrationMeta struct {
	Description string
	// Renames maps qualified old names to new names for explicit rename hints.
	// Keys use "namespace.table.column" format. Values are the new column name.
	Renames map[string]string
}

// BindMigration binds the migration() DSL function for migration files.
// Usage: migration({ up(m) { ... }, down(m) { ... }, up_hook(h) { ... } })
func (s *Sandbox) BindMigration() {
	s.vm.Set("migration", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "migration() requires a definition object", "try `migration({ up(m) { ... } })`")
		}

		// Expect an object with up() and down() methods
		defObj, ok := call.Arguments[0].(*goja.Object)
		if !ok {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "migration() argument must be an object with up() and down() methods", "try `migration({ up(m) { ... } })`")
		}

		// Parse optional description
		descVal := defObj.Get("description")
		if descVal != nil && !goja.IsUndefined(descVal) && !goja.IsNull(descVal) {
			s.migrationMeta.Description = descVal.String()
		}

		// Parse optional renames hints
		renamesVal := defObj.Get("renames")
		if renamesVal != nil && !goja.IsUndefined(renamesVal) && !goja.IsNull(renamesVal) {
			if renamesObj, ok := renamesVal.(*goja.Object); ok {
				// Look for columns sub-object
				colsVal := renamesObj.Get("columns")
				if colsVal != nil && !goja.IsUndefined(colsVal) && !goja.IsNull(colsVal) {
					if colsObj, ok := colsVal.(*goja.Object); ok {
						renames := make(map[string]string)
						for _, key := range colsObj.Keys() {
							val := colsObj.Get(key)
							if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
								renames[key] = val.String()
							}
						}
						s.migrationMeta.Renames = renames
					}
				}
			}
		}

		// Get the up() function
		upVal := defObj.Get("up")
		if upVal == nil || goja.IsUndefined(upVal) || goja.IsNull(upVal) {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "migration definition must have an up() method", "try `migration({ up(m) { ... } })`")
		}

		upFn, ok := goja.AssertFunction(upVal)
		if !ok {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "migration up property must be a function", "try `migration({ up(m) { ... } })`")
		}

		// Create migration builder object
		migrationObj := s.createMigrationObject()

		// Call the up() function with the migration builder
		_, err := upFn(goja.Undefined(), migrationObj)
		if err != nil {
			panicPassthrough(s.vm, err)
		}

		return goja.Undefined()
	})
}

// GetMigrationMeta returns the metadata parsed from the last migration evaluation.
func (s *Sandbox) GetMigrationMeta() MigrationMeta {
	return s.migrationMeta
}

// createMigrationObject creates the JavaScript object for the migration builder.
// Operations are stored directly in the sandbox's operations slice.
func (s *Sandbox) createMigrationObject() *goja.Object {
	obj := s.vm.NewObject()

	// create_table(ref, fn)
	// NOTE: Uses ToMigrationObject() which only has low-level types (no semantic types).
	// Migrations should be explicit and stable - use string(255), decimal(19,4), etc.
	_ = obj.Set("create_table", func(ref string, fn goja.Value) {
		builderFn, ok := goja.AssertFunction(fn)
		if !ok {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "create_table() second argument must be a function", "try `create_table('ns.table', function(col) { ... })`")
		}

		ns, table := mustParseRef(ref, s.vm)
		tb := builder.NewTableBuilder(s.vm)
		tbObj := tb.ToMigrationObject()

		_, err := builderFn(goja.Undefined(), tbObj)
		if err != nil {
			panicPassthrough(s.vm, err)
		}

		// Direct path: convert typed builder to AST (bypasses map conversion)
		converter := schema.NewColumnConverter()
		tableDef := converter.TableBuilderToAST(tb, ns, table)

		op := &ast.CreateTable{
			TableOp:     ast.TableOp{Namespace: ns, Name: table},
			Columns:     tableDef.Columns,
			Indexes:     tableDef.Indexes,
			ForeignKeys: make([]*ast.ForeignKeyDef, 0),
		}
		s.operations = append(s.operations, op)
	})

	// drop_table(ref)
	_ = obj.Set("drop_table", func(ref string) {
		ns, table := mustParseRef(ref, s.vm)
		s.operations = append(s.operations, &ast.DropTable{
			TableOp: ast.TableOp{Namespace: ns, Name: table},
		})
	})

	// rename_table(oldRef, newName)
	_ = obj.Set("rename_table", func(oldRef, newName string) {
		ns, oldTable := mustParseRef(oldRef, s.vm)
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
			panicStructured(s.vm, alerr.ErrMigrationFailed, "add_column() second argument must be a function", "try `add_column('ns.table', function(col) { ... })`")
		}

		ns, table := mustParseRef(ref, s.vm)

		// Reuse TableBuilder to get the same typed column builder as create_table.
		// The callback calls one method (e.g. col.string("bio", 500).backfill("..."))
		// which adds exactly one column to tb.Columns.
		tb := builder.NewTableBuilder(s.vm)
		tbObj := tb.ToMigrationObject()

		_, err := builderFn(goja.Undefined(), tbObj)
		if err != nil {
			panicPassthrough(s.vm, err)
		}

		if len(tb.Columns) == 0 {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "add_column() callback must define a column", "try `col.string('name', 255)` inside the callback")
		}
		if len(tb.Columns) > 1 {
			panicStructured(s.vm, alerr.ErrMigrationFailed, "add_column() callback must define exactly one column", "use one `add_column()` call per column")
		}

		converter := schema.NewColumnConverter()
		colDef := converter.ToAST(tb.Columns[0])
		s.operations = append(s.operations, &ast.AddColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Column:   colDef,
		})
	})

	// drop_column(ref, column)
	_ = obj.Set("drop_column", func(ref, column string) {
		ns, table := mustParseRef(ref, s.vm)
		s.operations = append(s.operations, &ast.DropColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     column,
		})
	})

	// rename_column(ref, oldName, newName)
	_ = obj.Set("rename_column", func(ref, oldName, newName string) {
		ns, table := mustParseRef(ref, s.vm)
		s.operations = append(s.operations, &ast.RenameColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			OldName:  oldName,
			NewName:  newName,
		})
	})

	// create_index(ref, columns, opts?)
	_ = obj.Set("create_index", func(ref string, columns []string, opts ...map[string]any) {
		ns, table := mustParseRef(ref, s.vm)
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
			op.Name = strutil.IndexName(strutil.SQLName(ns, table), columns...)
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
			panicStructured(s.vm, alerr.ErrMigrationFailed, "alter_column() third argument must be a function", "try `alter_column('ns.table', 'col', function(c) { ... })`")
		}

		ns, table := mustParseRef(ref, s.vm)

		// Create alter column builder
		alterOp := &ast.AlterColumn{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     column,
		}
		alterBuilder := s.createAlterColumnBuilderObject(alterOp)

		_, err := builderFn(goja.Undefined(), alterBuilder)
		if err != nil {
			panicPassthrough(s.vm, err)
		}

		s.operations = append(s.operations, alterOp)
	})

	// add_foreign_key(ref, columns, refTable, refColumns, opts?)
	_ = obj.Set("add_foreign_key", func(ref string, columns []string, refTable string, refColumns []string, opts ...map[string]any) {
		ns, table := mustParseRef(ref, s.vm)
		refNs, refTbl := mustParseRef(refTable, s.vm)
		sqlRefTable := strutil.SQLName(refNs, refTbl)

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
		ns, table := mustParseRef(ref, s.vm)
		s.operations = append(s.operations, &ast.DropForeignKey{
			TableRef: ast.TableRef{Namespace: ns, Table_: table},
			Name:     constraintName,
		})
	})

	// add_check(ref, name, expression)
	_ = obj.Set("add_check", func(ref, name, expression string) {
		ns, table := mustParseRef(ref, s.vm)
		s.operations = append(s.operations, &ast.AddCheck{
			TableRef:   ast.TableRef{Namespace: ns, Table_: table},
			Name:       name,
			Expression: expression,
		})
	})

	// drop_check(ref, name)
	_ = obj.Set("drop_check", func(ref, name string) {
		ns, table := mustParseRef(ref, s.vm)
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

// mustParseRef parses a table reference and validates the parts.
// Panics with a descriptive message if validation fails (for use inside Goja callbacks).
func mustParseRef(ref string, vm *goja.Runtime) (namespace, table string) {
	ns, tbl := strutil.ParseRef(ref)
	if tbl != "" {
		if err := ast.ValidateIdentifier(tbl); err != nil {
			panicStructured(vm, alerr.ErrInvalidReference, fmt.Sprintf("invalid table name '%s'", tbl), "use lowercase letters, digits, and underscores")
		}
	}
	if ns != "" {
		if err := ast.ValidateIdentifier(ns); err != nil {
			panicStructured(vm, alerr.ErrInvalidReference, fmt.Sprintf("invalid namespace '%s'", ns), "use lowercase letters, digits, and underscores")
		}
	}
	return ns, tbl
}

// BindSQL creates the sql() helper function for per-dialect raw SQL expressions.
// sql({ postgres: "...", sqlite: "..." }) returns {_type: "sql_expr", postgres: "...", sqlite: "..."}
func BindSQL(vm *goja.Runtime) {
	vm.Set("sql", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panicStructured(vm, alerr.ErrSchemaInvalid,
				"sql() requires an object with postgres and sqlite keys",
				"try `sql({ postgres: 'NOW()', sqlite: 'CURRENT_TIMESTAMP' })`")
		}
		obj, ok := call.Arguments[0].(*goja.Object)
		if !ok {
			panicStructured(vm, alerr.ErrSchemaInvalid,
				"sql() argument must be an object",
				"try `sql({ postgres: 'NOW()', sqlite: 'CURRENT_TIMESTAMP' })`")
		}
		pg := obj.Get("postgres")
		sl := obj.Get("sqlite")
		if pg == nil || goja.IsUndefined(pg) || sl == nil || goja.IsUndefined(sl) {
			panicStructured(vm, alerr.ErrSchemaInvalid,
				"sql() requires both `postgres` and `sqlite` keys",
				"try `sql({ postgres: 'NOW()', sqlite: 'CURRENT_TIMESTAMP' })`")
		}
		return vm.ToValue(map[string]any{
			"_type":    "sql_expr",
			"postgres": pg.String(),
			"sqlite":   sl.String(),
		})
	})
}
