// Package runtime provides a secure, deterministic JavaScript execution environment
// for evaluating schema and migration files using the Goja JS engine.
package runtime

import (
	"io"
	"log/slog"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
	"github.com/hlop3z/astroladb/internal/registry"
	"github.com/hlop3z/astroladb/internal/runtime/builder"
	"github.com/hlop3z/astroladb/internal/runtime/schema"
)

// FixedTime is the deterministic time used for all schema evaluations.
// Using a fixed time ensures reproducible output across environments.
var FixedTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// FixedSeed is the deterministic seed for random number generation.
const FixedSeed = 12345

// maxCallStackSize limits the JavaScript VM call stack depth to prevent stack overflow attacks.
const maxCallStackSize = 500

// defaultTimeout is the maximum execution time for a single schema/migration evaluation.
const defaultTimeout = 5 * time.Second

// exportRe matches ES6 "export " at the start of a line (statement boundary).
var exportRe = regexp.MustCompile(`(?m)^export `)

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

	// tainted is set to true after any interrupt/timeout, indicating the VM
	// may have leftover state and should be reset before reuse.
	tainted bool

	// restricted is set to true after restrictGlobals() runs, indicating the VM
	// has non-DSL globals removed (Date, Math, JSON, etc.). The VM must be Reset()
	// before it can be used for generators which need full JS access.
	restricted bool

	// migrationMeta stores metadata parsed from migration DSL.
	migrationMeta MigrationMeta
}

// NewSandbox creates a new hardened JavaScript sandbox.
// The registry is used for reference resolution during schema evaluation.
func NewSandbox(reg *registry.ModelRegistry) *Sandbox {
	if reg == nil {
		reg = registry.NewModelRegistry()
	}

	vm := goja.New()

	// 1. Resource limits - prevent stack overflow attacks
	vm.SetMaxCallStackSize(maxCallStackSize)

	// 2. Deterministic execution - fixed random source and time
	seedRand := rand.New(rand.NewSource(FixedSeed))
	vm.SetRandSource(func() float64 { return seedRand.Float64() })

	// 3. Disable dangerous globals
	disableDangerousGlobals(vm)

	s := &Sandbox{
		vm:       vm,
		registry: reg,
		timeout:  defaultTimeout,
		tables:   make([]*ast.TableDef, 0),
		meta:     metadata.New(),
	}

	// 4. Bind DSL functions
	s.bindDSL()

	return s
}

// Reset recreates the VM and re-initializes all bindings.
// This is called automatically when the sandbox is tainted (e.g., after a timeout interrupt).
func (s *Sandbox) Reset() {
	vm := goja.New()
	vm.SetMaxCallStackSize(maxCallStackSize)

	seedRand := rand.New(rand.NewSource(FixedSeed))
	vm.SetRandSource(func() float64 { return seedRand.Float64() })

	disableDangerousGlobals(vm)

	s.vm = vm
	s.tainted = false
	s.restricted = false
	s.tables = make([]*ast.TableDef, 0)
	s.operations = nil
	s.migrationMeta = MigrationMeta{}
	s.meta = metadata.New()
	s.currentFile = ""
	s.currentCode = ""
	s.lineOffset = 0

	s.bindDSL()
}

// disableDangerousGlobals removes or disables JS features that could
// cause security issues or non-deterministic behavior.
func disableDangerousGlobals(vm *goja.Runtime) {
	// Disable eval and Function constructor
	vm.Set("eval", goja.Undefined())

	// Block Function constructor (equivalent to eval)
	_, _ = vm.RunString(`(function() {
		try {
			var F = Function;
			Object.defineProperty(Function.prototype, 'constructor', {
				value: function() { throw new TypeError('Function constructor is disabled'); },
				writable: false,
				configurable: false
			});
		} catch(e) {}
	})();`)

	// Freeze prototypes to prevent pollution attacks
	// This is done safely - errors are ignored if freeze fails
	freezeCode := mustReadJSFile("js/freeze.js")
	_, _ = vm.RunString(freezeCode)
}

// restrictGlobals removes non-declarative JS globals (Date, Math, JSON, etc.)
// from the VM. This makes schemas and migrations behave like typed, executable JSON
// where only the DSL functions are available. After calling this, the sandbox is
// marked as restricted and must be Reset() before use with generators.
func (s *Sandbox) restrictGlobals() {
	restrictCode := mustReadJSFile("js/restrict.js")
	_, _ = s.vm.RunString(restrictCode)
	s.restricted = true
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

	// postgres() helper - marks a string as PostgreSQL-specific SQL expression
	s.vm.Set("postgres", func(expr string) map[string]any {
		return map[string]any{
			"_type":   "sql_expr",
			"expr":    expr,
			"dialect": "postgres",
		}
	})

	// sqlite() helper - marks a string as SQLite-specific SQL expression
	s.vm.Set("sqlite", func(expr string) map[string]any {
		return map[string]any{
			"_type":   "sql_expr",
			"expr":    expr,
			"dialect": "sqlite",
		}
	})

	// col global - column factory for object-based table definitions
	colBuilder := builder.NewColBuilder(s.vm)
	s.vm.Set("col", colBuilder.ToObject())

	// fn global - expression builder for computed columns
	fnBuilder := builder.NewFnBuilder(s.vm)
	s.vm.Set("fn", fnBuilder.ToObject())

	// table() function - defines a table using object API: table({ id: col.id() })
	s.vm.Set("table", s.tableFunc())

	// Export default handler - stores the exported table
	s.vm.Set("__registerTable", s.registerTableFunc())

	// migration() function - defines migration operations
	s.BindMigration()
}

// tableFunc returns the table() DSL function.
// Uses object-based API: table({ id: col.id() }).timestamps()
func (s *Sandbox) tableFunc() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(s.vm.ToValue("table() requires a column definitions object"))
		}

		arg := call.Arguments[0]

		// Object-based API: table({ id: col.id() })
		columnsObj, ok := arg.(*goja.Object)
		if !ok {
			panic(s.vm.ToValue("table() argument must be an object of column definitions"))
		}

		// Extract column definitions from the object
		columns := make([]*builder.ColumnDef, 0)
		indexes := make([]*builder.IndexDef, 0)

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
			colDef, ok := colDefVal.Export().(*builder.ColDef)
			if !ok {
				continue
			}

			// Convert ColDef to ColumnDef with the name from the object key
			columnDef := &builder.ColumnDef{
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
				ReadOnly:   colDef.ReadOnly,
				WriteOnly:  colDef.WriteOnly,
				Hidden:     colDef.Hidden,
				XRef:       colDef.XRef,
				Computed:   colDef.Computed,
				Virtual:    colDef.Virtual,
			}

			// Handle polymorphic relationships specially - they create two columns
			if colDef.Type == "polymorphic" {
				// Create {key}_type column
				typeCol := &builder.ColumnDef{
					Name:     key + "_type",
					Type:     "string",
					TypeArgs: []any{100},
					Nullable: colDef.Nullable,
				}
				columns = append(columns, typeCol)

				// Create {key}_id column
				idCol := &builder.ColumnDef{
					Name:     key + "_id",
					Type:     "uuid",
					Nullable: colDef.Nullable,
				}
				columns = append(columns, idCol)

				// Create composite index on type + id
				indexes = append(indexes, &builder.IndexDef{
					Columns: []string{key + "_type", key + "_id"},
					Unique:  false,
				})
				continue
			}

			// Set the column name based on relationship or regular column
			if colDef.IsRelationship {
				// For relationships, key becomes the alias: author -> author_id
				// But if key already ends with _id, use it as-is
				if strings.HasSuffix(key, "_id") {
					columnDef.Name = key
				} else {
					columnDef.Name = key + "_id"
				}
				columnDef.Reference = colDef.Reference

				// Auto-create index on foreign key
				indexes = append(indexes, &builder.IndexDef{
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
		tc := builder.NewTableChain(s.vm, columns, indexes)
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
			// Set source file for error reporting
			tableDef.SourceFile = s.currentFile
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
							s.meta.AddManyToMany(namespace, name, target, s.currentFile)
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

			// Check for junction marker
			if t, ok := m["_type"].(string); ok && t == "junction" {
				if junc, ok := m["junction"].(map[string]any); ok {
					// Collect FK columns from already-parsed columns
					var fkCols []*ast.ColumnDef
					for _, col := range tableDef.Columns {
						if col.Reference != nil {
							fkCols = append(fkCols, col)
						}
					}

					// Check for explicit refs
					sourceRef, hasSource := junc["source"].(string)
					targetRef, hasTarget := junc["target"].(string)

					if hasSource && hasTarget {
						// Explicit refs provided - find matching FK columns
						var sourceFK, targetFK string
						for _, fk := range fkCols {
							if fk.Reference.Table == sourceRef {
								sourceFK = fk.Name
							}
							if fk.Reference.Table == targetRef {
								targetFK = fk.Name
							}
						}
						if sourceFK != "" && targetFK != "" {
							s.meta.AddExplicitJunction(sourceRef, targetRef, sourceFK, targetFK)
						}
					} else {
						// Auto-detect: require exactly 2 FKs
						if len(fkCols) == 2 {
							s.meta.AddExplicitJunction(
								fkCols[0].Reference.Table,
								fkCols[1].Reference.Table,
								fkCols[0].Name,
								fkCols[1].Name,
							)
						}
					}
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
			if idxDef := schema.ParseIndexDefMap(i); idxDef != nil {
				tableDef.Indexes = append(tableDef.Indexes, idxDef)
			}
		}
	} else if idxsMap, ok := def["indexes"].([]map[string]any); ok {
		// Handle direct []map[string]any from Go slices passed through Goja
		for _, i := range idxsMap {
			if idxDef := schema.ParseIndexDefMap(i); idxDef != nil {
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
	tableDef.SortBy = toStringSlice(def["sort_by"])
	tableDef.Searchable = toStringSlice(def["searchable"])
	tableDef.Filterable = toStringSlice(def["filterable"])

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
		slog.Debug("parseColumnDef: found default in map",
			"column", name,
			"default", def)
		col.Default = s.convertValue(def)
		col.DefaultSet = true
	} else {
		slog.Debug("parseColumnDef: NO default in map",
			"column", name)
	}
	if backfill, exists := m["backfill"]; exists {
		col.Backfill = s.convertValue(backfill)
		col.BackfillSet = true
	}

	// Parse reference
	if ref, ok := m["reference"].(map[string]any); ok {
		col.Reference = schema.ParseReferenceMap(ref)
	}

	// Parse validation
	if min, ok := m["min"].(float64); ok {
		col.Min = &min
	}
	if max, ok := m["max"].(float64); ok {
		col.Max = &max
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

	// Parse access control
	if readOnly, ok := m["read_only"].(bool); ok {
		col.ReadOnly = readOnly
	}
	if writeOnly, ok := m["write_only"].(bool); ok {
		col.WriteOnly = writeOnly
	}

	// Parse computed expression
	if computed, exists := m["computed"]; exists {
		col.Computed = computed
	}
	if v, ok := m["virtual"].(bool); ok {
		col.Virtual = v
	}

	return col
}

// toStringSlice converts a Goja value to []string.
// Handles both []string and []any (where each element is asserted to string).
func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	if ss, ok := v.([]string); ok {
		return ss
	}
	if aa, ok := v.([]any); ok {
		var result []string
		for _, item := range aa {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// convertValue converts JS values (like sql("...") results) to Go types.
func (s *Sandbox) convertValue(v any) any {
	return ast.ConvertSQLExprValue(v)
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
	// This converts from wrapped code line numbers back to original source line numbers.
	adjustedLine := jsErr.Line
	if s.lineOffset > 0 && adjustedLine > s.lineOffset {
		adjustedLine = adjustedLine - s.lineOffset
	}

	// BUGFIX: Empirical testing shows we need +1 to get correct source line from file.
	// This is due to how Goja indexes lines in wrapped vs unwrapped code.
	sourceFileLine := adjustedLine + 1

	// Add file location if available (use adjusted line for display)
	if s.currentFile != "" {
		alErr.WithLocation(s.currentFile, sourceFileLine, jsErr.Column)
	} else if sourceFileLine > 0 {
		alErr.With("line", sourceFileLine)
		if jsErr.Column > 0 {
			alErr.With("column", jsErr.Column)
		}
	}

	// Try to get source line for context
	if sourceFileLine > 0 {
		var sourceLine string
		if s.currentFile != "" {
			sourceLine = GetSourceLineFromFile(s.currentFile, sourceFileLine)
		} else if s.currentCode != "" {
			// For in-memory code, use the original adjusted line (no +1 needed)
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
		if _, ok := err.(*goja.InterruptedError); ok {
			s.tainted = true
		}
		return s.wrapJSError(err, alerr.ErrJSExecution, alerr.MsgJSExecutionFailed)
	}

	// Clear any pending interrupt to prevent race condition
	// where the timer fires between RunString returning and timer.Stop()
	s.vm.ClearInterrupt()
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
		if _, ok := err.(*goja.InterruptedError); ok {
			s.tainted = true
		}
		return nil, s.wrapJSError(err, alerr.ErrJSExecution, "alerr.MsgJSExecutionFailed")
	}

	// Clear any pending interrupt to prevent race condition
	s.vm.ClearInterrupt()
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
// The code should define a table using `table({ id: col.id() }).timestamps()` syntax.
// ES6 export statements are stripped since Goja only supports ES5.1.
func (s *Sandbox) EvalSchema(code string, namespace string, tableName string) (*ast.TableDef, error) {
	// Reset VM if tainted by a previous interrupt/timeout
	if s.tainted {
		s.Reset()
	}

	// Restrict non-DSL globals (Date, Math, JSON, etc.)
	// Schemas are declarative — only the DSL should be available.
	if !s.restricted {
		s.restrictGlobals()
	}

	// Store original code for error context (if not already set by EvalSchemaFile)
	if s.currentCode == "" {
		s.currentCode = code
	}

	// Reset tables for this evaluation
	s.tables = make([]*ast.TableDef, 0)

	// Strip ES6 export statements (Goja only supports ES5.1)
	// Only match "export" at statement boundaries to avoid corrupting string literals
	originalCode := code
	code = strings.Replace(code, "export default ", "", 1)
	// Use regex to only strip "export " at line starts (statement boundary)
	code = exportRe.ReplaceAllString(code, "")

	// Wrap the schema code to capture the result
	// Handle both old API (returns result directly) and new API (returns chainable object)
	wrapperTmpl := mustReadJSFile("js/wrapper.js")
	wrappedCode := strings.ReplaceAll(wrapperTmpl, "{{CODE}}", code)
	wrappedCode = strings.ReplaceAll(wrappedCode, "{{NAMESPACE}}", namespace)
	wrappedCode = strings.ReplaceAll(wrappedCode, "{{TABLE_NAME}}", tableName)

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
// Deprecated: Direct VM access bypasses sandbox protections (timeout, Function
// constructor block, etc.). Only use in tests. Prefer adding scoped methods to
// Sandbox instead of exposing the raw runtime.
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
			s.tainted = true
			timeoutErr := alerr.New(alerr.ErrJSTimeout, "script execution timed out").
				With("timeout", timeout.String()).
				With("interrupt", interruptErr.String())
			if s.currentFile != "" {
				timeoutErr.WithFile(s.currentFile, 0)
			}
			return timeoutErr
		}

		return s.wrapJSError(err, alerr.ErrJSExecution, "alerr.MsgJSExecutionFailed")
	}

	// Clear any pending interrupt
	s.vm.ClearInterrupt()

	return nil
}

// RunFile reads and executes a JavaScript file (migration file).
// It strips ES6 export statements since Goja only supports ES5.1.
func (s *Sandbox) RunFile(path string) ([]ast.Operation, error) {
	// Reset VM if tainted by a previous interrupt/timeout
	if s.tainted {
		s.Reset()
	}

	// Restrict non-DSL globals (Date, Math, JSON, etc.)
	// Migrations are declarative — only the DSL should be available.
	if !s.restricted {
		s.restrictGlobals()
	}

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
	code = exportRe.ReplaceAllString(code, "")

	// Store original code for error context (before ES6 stripping)
	s.currentCode = originalCode
	// No wrapper used, line numbers match source directly
	s.lineOffset = 0

	// Reset results before evaluation
	s.tables = make([]*ast.TableDef, 0)
	s.operations = make([]ast.Operation, 0)
	s.migrationMeta = MigrationMeta{}

	if err := s.Run(code); err != nil {
		return nil, err
	}

	// The migration() wrapper function handles calling up() internally
	// Operations are collected in s.operations by the migration builder

	// Migration files must use: export default migration({ up, down })
	if len(s.operations) == 0 {
		return nil, alerr.New(alerr.ErrJSExecution, "migration file must use migration({ up, down }) wrapper").
			WithFile(path, 0)
	}

	return s.operations, nil
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

// SaveMetadataToFile saves the metadata to a custom file path.
func (s *Sandbox) SaveMetadataToFile(filePath string) error {
	return s.meta.SaveToFile(filePath)
}

// ClearMetadata resets the metadata for a fresh evaluation.
func (s *Sandbox) ClearMetadata() {
	s.meta = metadata.New()
}
