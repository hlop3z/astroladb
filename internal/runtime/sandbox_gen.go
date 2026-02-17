package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// errGen is a shorthand for generator error codes.
var errGen = alerr.ErrJSExecution

// GeneratorResult holds the output of a generator execution.
type GeneratorResult struct {
	Files map[string]string // path → content
}

// RunGenerator evaluates a generator JS file and returns the render output.
// The schema parameter is a map that will be frozen and passed to the generator callback.
func (s *Sandbox) RunGenerator(code string, schema map[string]any) (*GeneratorResult, error) {
	// Reset VM if tainted or restricted (generators need full JS access)
	if s.tainted || s.restricted {
		s.Reset()
	}

	// Bind generator functions
	var result *GeneratorResult
	s.bindGeneratorDSL(&result, schema)

	// Strip ES6 export
	code = strings.Replace(code, "export default ", "", 1)
	code = exportRe.ReplaceAllString(code, "")

	// Run
	if err := s.Run(code); err != nil {
		return nil, err
	}

	if result == nil {
		return nil, alerr.New(alerr.ErrJSExecution, "generator must call render() inside gen()").
			WithHelp("gen(function(schema) { render({ 'path/file.py': content }) })")
	}

	return result, nil
}

// bindGeneratorDSL binds gen(), render(), and helper functions.
func (s *Sandbox) bindGeneratorDSL(result **GeneratorResult, schema map[string]any) {
	vm := s.vm

	// Convert schema to a frozen JavaScript object
	// We use JSON round-trip to create a pure JS object that can be frozen
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		panicStructured(vm, errGen, fmt.Sprintf("failed to marshal schema: %v", err), "check that the schema contains only JSON-serializable values")
	}

	// Set the JSON string as a global variable
	vm.Set("__schemaJSON", string(schemaJSON))

	parseAndFreeze := `(function() {
		function deepFreeze(o) {
			Object.freeze(o);
			if (typeof o !== 'object' || o === null) {
				return o;
			}
			Object.getOwnPropertyNames(o).forEach(function(prop) {
				if (o[prop] !== null && typeof o[prop] === 'object' && !Object.isFrozen(o[prop])) {
					deepFreeze(o[prop]);
				}
			});
			return o;
		}
		var obj = JSON.parse(__schemaJSON);
		return deepFreeze(obj);
	})()`

	frozenSchema, err := vm.RunString(parseAndFreeze)
	if err != nil {
		panicStructured(vm, errGen, fmt.Sprintf("failed to freeze schema: %v", err), "this is an internal error, please report it")
	}

	// Clean up the temporary variable
	vm.Set("__schemaJSON", goja.Undefined())

	// render() — captures output
	vm.Set("render", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panicStructured(vm, errGen, "render() requires an object argument", "render({ 'path/file.py': content })")
		}

		arg := call.Arguments[0]
		if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
			panicStructured(vm, errGen, "render() argument must be a plain object", "render({ 'path/file.py': content })")
		}

		obj, ok := arg.(*goja.Object)
		if !ok {
			panicStructured(vm, errGen, "render() argument must be a plain object", "render({ 'path/file.py': content })")
		}

		// Check it's a plain object (not array)
		if obj.ClassName() == "Array" {
			panicStructured(vm, errGen, "render() argument must be a plain object, not an array", "render({ 'path/file.py': content })")
		}

		files := make(map[string]string)
		for _, key := range obj.Keys() {
			val := obj.Get(key)
			exported := val.Export()
			str, ok := exported.(string)
			if !ok {
				panicStructured(vm, errGen, fmt.Sprintf("render() value for key %q must be a string, got %T", key, exported), "all render() values must be strings")
			}
			files[key] = str
		}

		*result = &GeneratorResult{Files: files}
		return vm.ToValue(files)
	})

	// gen() — receives callback, passes schema
	vm.Set("gen", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panicStructured(vm, errGen, "gen() requires a callback function", "gen(function(schema) { render({ ... }) })")
		}

		fn, ok := goja.AssertFunction(call.Arguments[0])
		if !ok {
			panicStructured(vm, errGen, "gen() argument must be a function", "gen(function(schema) { render({ ... }) })")
		}

		ret, err := fn(goja.Undefined(), frozenSchema)
		if err != nil {
			panicPassthrough(vm, err)
		}

		return ret
	})

	// Helper: perTable(schema, fn) — maps each table to { path: content }, merges
	vm.Set("perTable", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panicStructured(vm, errGen, "perTable() requires (schema, fn)", "perTable(schema, function(table) { return { ... } })")
		}
		schemaArg := call.Arguments[0]
		fnArg := call.Arguments[1]

		fn, ok := goja.AssertFunction(fnArg)
		if !ok {
			panicStructured(vm, errGen, "perTable() second argument must be a function", "perTable(schema, function(table) { return { ... } })")
		}

		// Convert to object if needed
		var schemaObj *goja.Object
		if obj, ok := schemaArg.(*goja.Object); ok {
			schemaObj = obj
		} else {
			panicStructured(vm, errGen, "perTable() first argument must be an object", "perTable(schema, function(table) { return { ... } })")
		}

		tablesVal := schemaObj.Get("tables")
		if tablesVal == nil || goja.IsUndefined(tablesVal) {
			return vm.ToValue(map[string]string{})
		}

		merged := vm.NewObject()
		tablesObj, ok := tablesVal.(*goja.Object)
		if !ok {
			return vm.ToValue(map[string]string{})
		}

		// Iterate array
		lengthVal := tablesObj.Get("length")
		if lengthVal == nil {
			return merged
		}
		length := int(lengthVal.ToInteger())

		for i := 0; i < length; i++ {
			tableVal := tablesObj.Get(fmt.Sprintf("%d", i))
			ret, err := fn(goja.Undefined(), tableVal)
			if err != nil {
				panicPassthrough(vm, err)
			}
			if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
				continue
			}
			retObj, ok := ret.(*goja.Object)
			if !ok {
				continue
			}
			for _, k := range retObj.Keys() {
				_ = merged.Set(k, retObj.Get(k))
			}
		}

		return merged
	})

	// Helper: perNamespace(schema, fn)
	vm.Set("perNamespace", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panicStructured(vm, errGen, "perNamespace() requires (schema, fn)", "perNamespace(schema, function(ns, tables) { return { ... } })")
		}
		schemaArg := call.Arguments[0]
		fnArg := call.Arguments[1]

		fn, ok := goja.AssertFunction(fnArg)
		if !ok {
			panicStructured(vm, errGen, "perNamespace() second argument must be a function", "perNamespace(schema, function(ns, tables) { return { ... } })")
		}

		// Convert to object if needed
		var schemaObj *goja.Object
		if obj, ok := schemaArg.(*goja.Object); ok {
			schemaObj = obj
		} else {
			panicStructured(vm, errGen, "perNamespace() first argument must be an object", "perNamespace(schema, function(ns, tables) { return { ... } })")
		}

		nsVal := schemaObj.Get("models")
		if nsVal == nil || goja.IsUndefined(nsVal) {
			return vm.ToValue(map[string]string{})
		}

		nsObj, ok := nsVal.(*goja.Object)
		if !ok {
			return vm.ToValue(map[string]string{})
		}

		merged := vm.NewObject()
		for _, nsKey := range nsObj.Keys() {
			nsTablesVal := nsObj.Get(nsKey)
			ret, err := fn(goja.Undefined(), vm.ToValue(nsKey), nsTablesVal)
			if err != nil {
				panicPassthrough(vm, err)
			}
			if ret == nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
				continue
			}
			retObj, ok := ret.(*goja.Object)
			if !ok {
				continue
			}
			for _, k := range retObj.Keys() {
				_ = merged.Set(k, retObj.Get(k))
			}
		}

		return merged
	})

	// Helper: json(value, indent?)
	vm.Set("json", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panicStructured(vm, errGen, "json() requires a value", "json(value) or json(value, '  ')")
		}
		value := call.Arguments[0].Export()
		indent := ""
		if len(call.Arguments) > 1 {
			indent = call.Arguments[1].String()
		}

		var data []byte
		var err error
		if indent != "" {
			data, err = json.MarshalIndent(value, "", indent)
		} else {
			data, err = json.Marshal(value)
		}
		if err != nil {
			panicStructured(vm, errGen, fmt.Sprintf("json() serialization failed: %v", err), "ensure the value is JSON-serializable")
		}
		return vm.ToValue(string(data))
	})

	// Helper: dedent(str) — removes common leading whitespace
	vm.Set("dedent", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue("")
		}
		str := call.Arguments[0].String()
		lines := strings.Split(str, "\n")

		// Find minimum indentation (ignoring empty lines)
		minIndent := -1
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if minIndent < 0 || indent < minIndent {
				minIndent = indent
			}
		}

		if minIndent <= 0 {
			return vm.ToValue(str)
		}

		// Remove common indentation
		result := make([]string, len(lines))
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				result[i] = ""
			} else if len(line) >= minIndent {
				result[i] = line[minIndent:]
			} else {
				result[i] = line
			}
		}
		return vm.ToValue(strings.Join(result, "\n"))
	})
}
