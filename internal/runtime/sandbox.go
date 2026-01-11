// Package runtime provides a secure, deterministic JavaScript execution environment
// for evaluating schema and migration files using the Goja JS engine.
package runtime

import (
	"io"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
	"github.com/hlop3z/astroladb/internal/registry"
)

// FixedTime is the deterministic time used for all schema evaluations.
// Using a fixed time ensures reproducible output across environments.
var FixedTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// FixedSeed is the deterministic seed for random number generation.
const FixedSeed = 12345

// Sandbox provides a secure JavaScript execution environment for evaluating
// schema and migration files. It enforces deterministic execution and
// resource limits to prevent runaway scripts.
type Sandbox struct {
	vm       *goja.Runtime
	registry *registry.ModelRegistry

	// Execution timeout for JS code
	timeout time.Duration

	// Results collected during evaluation
	tables     []*ast.TableDef
	operations []ast.Operation    // Migration operations
	meta       *metadata.Metadata // Schema metadata (many_to_many, polymorphic)

	// Current file context for rich error messages
	currentFile string
	currentCode string

	// Line offset for wrapped code (to convert error line numbers back to source)
	lineOffset int
}

// NewSandbox creates a new hardened JavaScript sandbox.
// The registry is used for reference resolution during schema evaluation.
func NewSandbox(reg *registry.ModelRegistry) *Sandbox {
	if reg == nil {
		reg = registry.NewModelRegistry()
	}

	vm := goja.New()

	// 1. Resource limits - prevent stack overflow attacks
	vm.SetMaxCallStackSize(500)

	// 2. Deterministic execution - fixed random source and time
	seedRand := rand.New(rand.NewSource(FixedSeed))
	vm.SetRandSource(func() float64 { return seedRand.Float64() })

	// 3. Disable dangerous globals
	disableDangerousGlobals(vm)

	s := &Sandbox{
		vm:       vm,
		registry: reg,
		timeout:  5 * time.Second,
		tables:   make([]*ast.TableDef, 0),
		meta:     metadata.New(),
	}

	// 4. Bind DSL functions
	s.bindDSL()

	return s
}

// disableDangerousGlobals removes or disables JS features that could
// cause security issues or non-deterministic behavior.
func disableDangerousGlobals(vm *goja.Runtime) {
	// Disable eval and Function constructor
	vm.Set("eval", goja.Undefined())

	// Freeze prototypes to prevent pollution attacks
	// This is done safely - errors are ignored if freeze fails
	_, _ = vm.RunString(`
		(function() {
			try {
				Object.freeze(Object.prototype);
				Object.freeze(Array.prototype);
				Object.freeze(String.prototype);
				Object.freeze(Number.prototype);
				Object.freeze(Boolean.prototype);
			} catch(e) {}
		})();
	`)
}

// bindDSL binds the schema DSL functions to the JS runtime.
func (s *Sandbox) bindDSL() {
	// sql() helper - marks a string as raw SQL expression
	s.vm.Set("sql", func(expr string) map[string]any {
		return map[string]any{
			"_type": "sql_expr",
			"expr":  expr,
		}
	})

	// col global - column factory for object-based table definitions
	colBuilder := NewColBuilder(s.vm)
	s.vm.Set("col", colBuilder.ToObject())

	// fn global - expression builder for computed columns
	fnBuilder := NewFnBuilder(s.vm)
	s.vm.Set("fn", fnBuilder.ToObject())

	// table() function - defines a table (supports both callback and object API)
	s.vm.Set("table", s.tableFunc())

	// Export default handler - stores the exported table
	s.vm.Set("__registerTable", s.registerTableFunc())

	// migration() function - defines migration operations
	s.BindMigration()
}

// tableFunc returns the table() DSL function.
// Supports both callback-based API: table(t => { t.id() })
// and object-based API: table({ id: col.id() }).timestamps()
func (s *Sandbox) tableFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(s.vm.ToValue("table() requires a builder function or column object"))
		}

		arg := call.Arguments[0]

		// Check if it's a function (callback-based API)
		if builderFn, ok := goja.AssertFunction(arg); ok {
			// Old callback-based API: table(t => { t.id() })
			tb := NewTableBuilder(s.vm)
			tbObj := tb.ToObject()
			_, err := builderFn(goja.Undefined(), tbObj)
			if err != nil {
				panic(s.vm.ToValue("error in table builder: " + err.Error()))
			}
			return tb.ToResult()
		}

		// Object-based API: table({ id: col.id() })
		columnsObj, ok := arg.(*goja.Object)
		if !ok {
			panic(s.vm.ToValue("table() argument must be a function or object"))
		}

		// Extract column definitions from the object
		columns := make([]*ColumnDef, 0)
		indexes := make([]*IndexDef, 0)

		// Get keys and sort them for deterministic column order
		keys := columnsObj.Keys()
		sort.Strings(keys)

		for _, key := range keys {
			colVal := columnsObj.Get(key)
			if colVal == nil || goja.IsUndefined(colVal) || goja.IsNull(colVal) {
				continue
			}

			colObj, ok := colVal.(*goja.Object)
			if !ok {
				continue
			}

			// Get the _colDef from the column builder object
			colDefVal := colObj.Get("_colDef")
			if colDefVal == nil || goja.IsUndefined(colDefVal) {
				continue
			}

			// Extract the ColDef pointer
			colDef, ok := colDefVal.Export().(*ColDef)
			if !ok {
				continue
			}

			// Convert ColDef to ColumnDef with the name from the object key
			columnDef := &ColumnDef{
				Type:       colDef.Type,
				TypeArgs:   colDef.TypeArgs,
				Nullable:   colDef.Nullable,
				Unique:     colDef.Unique,
				PrimaryKey: colDef.PrimaryKey,
				Default:    colDef.Default,
				Backfill:   colDef.Backfill,
				Format:     colDef.Format,
				Pattern:    colDef.Pattern,
				Min:        colDef.Min,
				Max:        colDef.Max,
				Docs:       colDef.Docs,
				Deprecated: colDef.Deprecated,
				Hidden:     colDef.Hidden,
				XRef:       colDef.XRef,
				Computed:   colDef.Computed,
			}

			// Handle polymorphic relationships specially - they create two columns
			if colDef.Type == "polymorphic" {
				// Create {key}_type column
				typeCol := &ColumnDef{
					Name:     key + "_type",
					Type:     "string",
					TypeArgs: []any{100},
					Nullable: colDef.Nullable,
				}
				columns = append(columns, typeCol)

				// Create {key}_id column
				idCol := &ColumnDef{
					Name:     key + "_id",
					Type:     "uuid",
					Nullable: colDef.Nullable,
				}
				columns = append(columns, idCol)

				// Create composite index on type + id
				indexes = append(indexes, &IndexDef{
					Columns: []string{key + "_type", key + "_id"},
					Unique:  false,
				})
				continue
			}

			// Set the column name based on relationship or regular column
			if colDef.IsRelationship {
				// For relationships, key becomes the alias: author -> author_id
				columnDef.Name = key + "_id"
				columnDef.Reference = colDef.Reference

				// Auto-create index on foreign key
				indexes = append(indexes, &IndexDef{
					Columns: []string{columnDef.Name},
					Unique:  colDef.Unique,
				})
			} else {
				// Regular column: key is the column name
				columnDef.Name = key
			}

			columns = append(columns, columnDef)
		}

		// Return a chainable TableChain object
		tc := NewTableChain(s.vm, columns, indexes)
		return tc.ToChainableObject()
	}
}

// registerTableFunc returns the function that registers a table definition.
func (s *Sandbox) registerTableFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(s.vm.ToValue("__registerTable requires namespace, name, and definition"))
		}

		namespace := call.Arguments[0].String()
		name := call.Arguments[1].String()
		defObj := call.Arguments[2].Export()

		tableDef := s.parseTableDef(namespace, name, defObj)
		if tableDef != nil {
			s.tables = append(s.tables, tableDef)
		}

		return goja.Undefined()
	}
}

// parseTableDef converts a JS object to an ast.TableDef.
func (s *Sandbox) parseTableDef(namespace, name string, defObj any) *ast.TableDef {
	def, ok := defObj.(map[string]any)
	if !ok {
		return nil
	}

	tableDef := &ast.TableDef{
		Namespace: namespace,
		Name:      name,
		Columns:   make([]*ast.ColumnDef, 0),
		Indexes:   make([]*ast.IndexDef, 0),
		Checks:    make([]*ast.CheckDef, 0),
	}

	// Parse columns and extract relationships
	if colsRaw, exists := def["columns"]; exists {
		var cols []any
		switch c := colsRaw.(type) {
		case []any:
			cols = c
		case []map[string]any:
			for _, m := range c {
				cols = append(cols, m)
			}
		}
		for _, c := range cols {
			m, ok := c.(map[string]any)
			if !ok {
				continue
			}

			// Check for many_to_many relationship
			if t, ok := m["_type"].(string); ok && t == "relationship" {
				if rel, ok := m["relationship"].(map[string]any); ok {
					if relType, ok := rel["type"].(string); ok && relType == "many_to_many" {
						if target, ok := rel["target"].(string); ok {
							// Register many_to_many relationship in metadata
							s.meta.AddManyToMany(namespace, name, target)
						}
					}
				}
				continue
			}

			// Check for polymorphic relationship
			if t, ok := m["_type"].(string); ok && t == "polymorphic" {
				if poly, ok := m["polymorphic"].(map[string]any); ok {
					alias, _ := poly["as"].(string)
					var targets []string
					if targetsRaw, ok := poly["targets"].([]any); ok {
						for _, t := range targetsRaw {
							if ts, ok := t.(string); ok {
								targets = append(targets, ts)
							}
						}
					} else if targetsRaw, ok := poly["targets"].([]string); ok {
						targets = targetsRaw
					}
					s.meta.AddPolymorphic(namespace, name, alias, targets)
				}
				continue
			}

			// Parse regular column
			if colDef := s.parseColumnDef(c); colDef != nil {
				tableDef.Columns = append(tableDef.Columns, colDef)
			}
		}
	}

	// Parse indexes (handle both []any and []map[string]any from Goja)
	if idxsAny, ok := def["indexes"].([]any); ok {
		for _, i := range idxsAny {
			if idxDef := s.parseIndexDef(i); idxDef != nil {
				tableDef.Indexes = append(tableDef.Indexes, idxDef)
			}
		}
	} else if idxsMap, ok := def["indexes"].([]map[string]any); ok {
		// Handle direct []map[string]any from Go slices passed through Goja
		for _, i := range idxsMap {
			if idxDef := s.parseIndexDef(i); idxDef != nil {
				tableDef.Indexes = append(tableDef.Indexes, idxDef)
			}
		}
	}

	// Parse metadata
	if docs, ok := def["docs"].(string); ok {
		tableDef.Docs = docs
	}
	if deprecated, ok := def["deprecated"].(string); ok {
		tableDef.Deprecated = deprecated
	}

	// Parse x-db metadata
	if auditable, ok := def["auditable"].(bool); ok {
		tableDef.Auditable = auditable
	}
	if sortBy, ok := def["sort_by"].([]string); ok {
		tableDef.SortBy = sortBy
	} else if sortByAny, ok := def["sort_by"].([]any); ok {
		// Handle []any from Goja
		for _, v := range sortByAny {
			if s, ok := v.(string); ok {
				tableDef.SortBy = append(tableDef.SortBy, s)
			}
		}
	}
	if searchable, ok := def["searchable"].([]string); ok {
		tableDef.Searchable = searchable
	} else if searchableAny, ok := def["searchable"].([]any); ok {
		for _, v := range searchableAny {
			if s, ok := v.(string); ok {
				tableDef.Searchable = append(tableDef.Searchable, s)
			}
		}
	}
	if filterable, ok := def["filterable"].([]string); ok {
		tableDef.Filterable = filterable
	} else if filterableAny, ok := def["filterable"].([]any); ok {
		for _, v := range filterableAny {
			if s, ok := v.(string); ok {
				tableDef.Filterable = append(tableDef.Filterable, s)
			}
		}
	}

	// Add table to metadata
	s.meta.AddTable(tableDef)

	return tableDef
}

// parseColumnDef converts a JS object to an ast.ColumnDef.
func (s *Sandbox) parseColumnDef(obj any) *ast.ColumnDef {
	m, ok := obj.(map[string]any)
	if !ok {
		return nil
	}

	// Skip metadata-only entries (relationships, polymorphic markers)
	if _, isMetadata := m["_type"]; isMetadata {
		return nil
	}

	// Skip entries without a name (invalid column definitions)
	name, hasName := m["name"].(string)
	if !hasName || name == "" {
		return nil
	}

	col := &ast.ColumnDef{
		Name: name,
	}
	if typ, ok := m["type"].(string); ok {
		col.Type = typ
	}
	if args, ok := m["type_args"].([]any); ok {
		col.TypeArgs = args
	}
	if nullable, ok := m["nullable"].(bool); ok {
		col.Nullable = nullable
		col.NullableSet = true
	}
	if unique, ok := m["unique"].(bool); ok {
		col.Unique = unique
	}
	if pk, ok := m["primary_key"].(bool); ok {
		col.PrimaryKey = pk
	}
	if def, exists := m["default"]; exists {
		col.Default = s.convertValue(def)
		col.DefaultSet = true
	}
	if backfill, exists := m["backfill"]; exists {
		col.Backfill = s.convertValue(backfill)
		col.BackfillSet = true
	}

	// Parse reference
	if ref, ok := m["reference"].(map[string]any); ok {
		col.Reference = s.parseReference(ref)
	}

	// Parse validation
	if min, ok := m["min"].(float64); ok {
		minInt := int(min)
		col.Min = &minInt
	}
	if max, ok := m["max"].(float64); ok {
		maxInt := int(max)
		col.Max = &maxInt
	}
	if pattern, ok := m["pattern"].(string); ok {
		col.Pattern = pattern
	}
	if format, ok := m["format"].(string); ok {
		col.Format = format
	}

	// Parse documentation
	if docs, ok := m["docs"].(string); ok {
		col.Docs = docs
	}
	if deprecated, ok := m["deprecated"].(string); ok {
		col.Deprecated = deprecated
	}

	// Parse computed expression
	if computed, exists := m["computed"]; exists {
		col.Computed = computed
	}

	return col
}

// parseReference converts a JS object to an ast.Reference.
func (s *Sandbox) parseReference(m map[string]any) *ast.Reference {
	ref := &ast.Reference{}

	if table, ok := m["table"].(string); ok {
		// Keep the original reference format (e.g., "auth.users")
		// The SQL dialect will convert to flat table name when generating SQL
		ref.Table = table
	}
	if column, ok := m["column"].(string); ok {
		ref.Column = column
	} else {
		ref.Column = "id" // Default to id
	}
	if onDelete, ok := m["on_delete"].(string); ok {
		ref.OnDelete = onDelete
	}
	if onUpdate, ok := m["on_update"].(string); ok {
		ref.OnUpdate = onUpdate
	}

	return ref
}

// convertValue converts JS values (like sql("...") results) to Go types.
func (s *Sandbox) convertValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		// Check if this is a SQL expression marker from sql("...")
		if typ, ok := val["_type"].(string); ok && typ == "sql_expr" {
			if expr, ok := val["expr"].(string); ok {
				return &ast.SQLExpr{Expr: expr}
			}
		}
		return val
	default:
		return v
	}
}

// parseIndexDef converts a JS object to an ast.IndexDef.
func (s *Sandbox) parseIndexDef(obj any) *ast.IndexDef {
	m, ok := obj.(map[string]any)
	if !ok {
		return nil
	}

	idx := &ast.IndexDef{}

	if name, ok := m["name"].(string); ok {
		idx.Name = name
	}
	if cols, ok := m["columns"].([]any); ok {
		for _, c := range cols {
			if s, ok := c.(string); ok {
				idx.Columns = append(idx.Columns, s)
			}
		}
	}
	if unique, ok := m["unique"].(bool); ok {
		idx.Unique = unique
	}
	if where, ok := m["where"].(string); ok {
		idx.Where = where
	}

	return idx
}

// SetTimeout sets the execution timeout for JS code.
func (s *Sandbox) SetTimeout(d time.Duration) {
	s.timeout = d
}

// SetCurrentFile sets the current file being evaluated for error context.
func (s *Sandbox) SetCurrentFile(path string) {
	s.currentFile = path
}

// wrapJSError creates a rich error with source location and context.
func (s *Sandbox) wrapJSError(err error, code alerr.Code, message string) *alerr.Error {
	jsErr := ParseJSError(err)
	if jsErr == nil {
		return alerr.Wrap(code, err, message)
	}

	// Create base error
	alErr := alerr.Wrap(code, err, message)

	// Adjust line number for wrapper offset (the IIFE wrapper adds 2 lines before user code)
	// This converts from wrapped code line numbers back to original source line numbers
	adjustedLine := jsErr.Line
	if s.lineOffset > 0 && adjustedLine > s.lineOffset {
		adjustedLine = adjustedLine - s.lineOffset
	}

	// Add file location if available
	if s.currentFile != "" {
		alErr.WithLocation(s.currentFile, adjustedLine, jsErr.Column)
	} else if adjustedLine > 0 {
		alErr.With("line", adjustedLine)
		if jsErr.Column > 0 {
			alErr.With("column", jsErr.Column)
		}
	}

	// Try to get source line for context
	if adjustedLine > 0 {
		var sourceLine string
		if s.currentFile != "" {
			sourceLine = GetSourceLineFromFile(s.currentFile, adjustedLine)
		} else if s.currentCode != "" {
			sourceLine = GetSourceLine(s.currentCode, adjustedLine)
		}

		if sourceLine != "" {
			alErr.WithSource(sourceLine)
			// Add span for the error position if we have a column
			if jsErr.Column > 0 {
				// Highlight from column to end of first token or reasonable length
				end := jsErr.Column + 10
				if end > len(sourceLine) {
					end = len(sourceLine)
				}
				alErr.WithSpan(jsErr.Column, end)
			}
		}
	}

	// Add helpful context based on the error message
	addJSErrorHelp(alErr, jsErr.Message)

	return alErr
}

// Run executes JavaScript code and returns any error.
func (s *Sandbox) Run(code string) error {
	// Store code for error context
	s.currentCode = code

	// Set up timeout
	timer := time.AfterFunc(s.timeout, func() {
		s.vm.Interrupt("execution timeout")
	})
	defer timer.Stop()

	_, err := s.vm.RunString(code)
	if err != nil {
		return s.wrapJSError(err, alerr.ErrJSExecution, "JavaScript execution failed")
	}
	return nil
}

// RunWithResult executes JavaScript code and returns the result.
func (s *Sandbox) RunWithResult(code string) (goja.Value, error) {
	// Store code for error context
	s.currentCode = code

	// Set up timeout
	timer := time.AfterFunc(s.timeout, func() {
		s.vm.Interrupt("execution timeout")
	})
	defer timer.Stop()

	result, err := s.vm.RunString(code)
	if err != nil {
		return nil, s.wrapJSError(err, alerr.ErrJSExecution, "JavaScript execution failed")
	}
	return result, nil
}

// EvalSchemaFile evaluates a schema file from disk and returns the table definition.
// This is the preferred method as it provides rich error context with file paths.
func (s *Sandbox) EvalSchemaFile(path string, namespace string, tableName string) (*ast.TableDef, error) {
	// Set current file for error context
	s.currentFile = path

	codeBytes, err := readFile(path)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrJSExecution, err, "failed to read schema file").
			WithFile(path, 0)
	}

	// Store original code for error context
	originalCode := string(codeBytes)
	s.currentCode = originalCode

	return s.EvalSchema(originalCode, namespace, tableName)
}

// EvalSchema evaluates a schema file and returns the table definition.
// The code should define a table using `table(t => {...})` syntax or
// the new object-based `table({ id: col.id() }).timestamps()` syntax.
// ES6 export statements are stripped since Goja only supports ES5.1.
func (s *Sandbox) EvalSchema(code string, namespace string, tableName string) (*ast.TableDef, error) {
	// Store original code for error context (if not already set by EvalSchemaFile)
	if s.currentCode == "" {
		s.currentCode = code
	}

	// Reset tables for this evaluation
	s.tables = make([]*ast.TableDef, 0)

	// Strip ES6 export statements (Goja only supports ES5.1)
	// Replace "export default " with empty string
	originalCode := code
	code = strings.Replace(code, "export default ", "", 1)
	// Also handle "export " for named exports
	code = strings.ReplaceAll(code, "export ", "")

	// Wrap the schema code to capture the result
	// Handle both old API (returns result directly) and new API (returns chainable object)
	wrappedCode := `
(function() {
	var __result = ` + code + `;

	// Check if result is a chainable object (new API) with _getResult method
	if (__result && typeof __result._getResult === 'function') {
		__result = __result._getResult();
	}

	// Register the table if it has columns (table builder result)
	if (__result && __result.columns) {
		__registerTable("` + namespace + `", "` + tableName + `", __result);
	}
})();
`

	// Temporarily store original code (unwrapped) for better error messages
	s.currentCode = originalCode
	// The wrapper adds 2 lines before user code (empty line + "(function() {" + "var __result = ")
	// Line 3 of wrapped code = line 1 of user code
	s.lineOffset = 2

	if err := s.Run(wrappedCode); err != nil {
		return nil, err
	}

	// Return the first table found
	if len(s.tables) > 0 {
		return s.tables[0], nil
	}

	return nil, alerr.New(alerr.ErrSchemaInvalid, "schema file did not export a table definition").
		WithTable(namespace, tableName).
		WithHelp("ensure your schema exports a table definition using 'export default table({...})'")
}

// GetTables returns all tables collected during evaluation.
func (s *Sandbox) GetTables() []*ast.TableDef {
	return s.tables
}

// ClearTables clears the collected tables.
func (s *Sandbox) ClearTables() {
	s.tables = make([]*ast.TableDef, 0)
}

// Registry returns the model registry.
func (s *Sandbox) Registry() *registry.ModelRegistry {
	return s.registry
}

// VM returns the underlying Goja runtime.
// Use with caution - direct VM access bypasses sandbox protections.
func (s *Sandbox) VM() *goja.Runtime {
	return s.vm
}

// RunWithTimeout executes JavaScript code with a specific timeout.
// Returns any error that occurred during execution.
func (s *Sandbox) RunWithTimeout(code string, timeout time.Duration) error {
	// Store code for error context
	s.currentCode = code
	// No wrapper used, line numbers match source directly
	s.lineOffset = 0

	// Set up timeout
	timer := time.AfterFunc(timeout, func() {
		s.vm.Interrupt("execution timeout")
	})
	defer timer.Stop()

	_, err := s.vm.RunString(code)
	if err != nil {
		// Check if it was a timeout
		if interruptErr, ok := err.(*goja.InterruptedError); ok {
			timeoutErr := alerr.New(alerr.ErrJSTimeout, "script execution timed out").
				With("timeout", timeout.String()).
				With("interrupt", interruptErr.String())
			if s.currentFile != "" {
				timeoutErr.WithFile(s.currentFile, 0)
			}
			return timeoutErr
		}

		return s.wrapJSError(err, alerr.ErrJSExecution, "JavaScript execution failed")
	}

	// Clear any pending interrupt
	s.vm.ClearInterrupt()

	return nil
}

// RunFile reads and executes a JavaScript file (migration file).
// It strips ES6 export statements since Goja only supports ES5.1.
func (s *Sandbox) RunFile(path string) ([]ast.Operation, error) {
	// Set current file for error context
	s.currentFile = path

	codeBytes, err := readFile(path)
	if err != nil {
		return nil, alerr.Wrap(alerr.ErrJSExecution, err, "failed to read file").
			WithFile(path, 0)
	}

	// Strip ES6 export statements (Goja only supports ES5.1)
	code := string(codeBytes)
	originalCode := code // Keep original for error context
	code = strings.Replace(code, "export default ", "", 1)
	code = strings.ReplaceAll(code, "export ", "")

	// Store original code for error context (before ES6 stripping)
	s.currentCode = originalCode
	// No wrapper used, line numbers match source directly
	s.lineOffset = 0

	// Reset results before evaluation
	s.tables = make([]*ast.TableDef, 0)
	s.operations = make([]ast.Operation, 0)

	if err := s.Run(code); err != nil {
		return nil, err
	}

	// Check for export function up(m) { ... } format
	// If an 'up' function exists, call it with a migration builder
	if upFn := s.vm.Get("up"); upFn != nil && upFn != goja.Undefined() {
		fn, ok := goja.AssertFunction(upFn)
		if ok {
			migrationObj := s.createMigrationObject()
			_, err := fn(goja.Undefined(), migrationObj)
			if err != nil {
				return nil, alerr.Wrap(alerr.ErrJSExecution, err, "error calling up() function").
					WithFile(path, 0)
			}
		}
	}

	// Return operations collected from migration() DSL or up() function
	// If no operations were collected, fall back to table definitions
	if len(s.operations) > 0 {
		return s.operations, nil
	}

	// Convert table definitions to operations (for schema files used as migrations)
	ops := make([]ast.Operation, 0, len(s.tables))
	for _, table := range s.tables {
		op := &ast.CreateTable{
			TableOp:     ast.TableOp{Namespace: table.Namespace, Name: table.Name},
			Columns:     table.Columns,
			Indexes:     table.Indexes,
			ForeignKeys: make([]*ast.ForeignKeyDef, 0),
		}
		ops = append(ops, op)
	}

	return ops, nil
}

// readFile reads a file's contents.
func readFile(path string) ([]byte, error) {
	file, err := openFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readAll(file)
}

// openFile opens a file for reading.
var openFile = os.Open

// readAll reads all data from a reader.
var readAll = io.ReadAll

// Metadata returns the schema metadata collected during evaluation.
func (s *Sandbox) Metadata() *metadata.Metadata {
	return s.meta
}

// GetJoinTables returns all auto-generated join table definitions.
// These are created from many_to_many relationships.
func (s *Sandbox) GetJoinTables() []*ast.TableDef {
	return s.meta.GetJoinTables()
}

// SaveMetadata saves the metadata to the .alab directory.
func (s *Sandbox) SaveMetadata(projectDir string) error {
	return s.meta.Save(projectDir)
}

// ClearMetadata resets the metadata for a fresh evaluation.
func (s *Sandbox) ClearMetadata() {
	s.meta = metadata.New()
}
