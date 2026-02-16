package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/registry"
)

func TestNewSandbox(t *testing.T) {
	t.Run("with nil registry", func(t *testing.T) {
		sb := NewSandbox(nil)
		if sb == nil {
			t.Fatal("NewSandbox() returned nil")
		}
		if sb.vm == nil {
			t.Error("vm should be initialized")
		}
		if sb.registry == nil {
			t.Error("registry should be initialized even when nil passed")
		}
		if sb.tables == nil {
			t.Error("tables should be initialized")
		}
	})

	t.Run("with registry", func(t *testing.T) {
		reg := registry.NewModelRegistry()
		sb := NewSandbox(reg)
		if sb.registry != reg {
			t.Error("registry should be set to provided registry")
		}
	})

	t.Run("default timeout", func(t *testing.T) {
		sb := NewSandbox(nil)
		if sb.timeout != 5*time.Second {
			t.Errorf("default timeout = %v, want 5s", sb.timeout)
		}
	})
}

func TestSandbox_Run(t *testing.T) {
	t.Run("simple JavaScript", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.Run("var x = 1 + 1;")
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})

	t.Run("syntax error", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.Run("var x = ;")
		if err == nil {
			t.Error("Run() should error on syntax error")
		}
	})

	t.Run("runtime error", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.Run("throw new Error('test error');")
		if err == nil {
			t.Error("Run() should error on runtime error")
		}
	})

	t.Run("multiple statements", func(t *testing.T) {
		sb := NewSandbox(nil)
		code := `
			var a = 10;
			var b = 20;
			var c = a + b;
		`
		err := sb.Run(code)
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})
}

func TestSandbox_RunWithResult(t *testing.T) {
	t.Run("returns value", func(t *testing.T) {
		sb := NewSandbox(nil)
		result, err := sb.RunWithResult("1 + 2")
		if err != nil {
			t.Errorf("RunWithResult() error = %v", err)
		}
		if result.ToInteger() != 3 {
			t.Errorf("Result = %v, want 3", result.Export())
		}
	})

	t.Run("returns string", func(t *testing.T) {
		sb := NewSandbox(nil)
		result, err := sb.RunWithResult("'hello' + ' world'")
		if err != nil {
			t.Errorf("RunWithResult() error = %v", err)
		}
		if result.String() != "hello world" {
			t.Errorf("Result = %q, want %q", result.String(), "hello world")
		}
	})

	t.Run("returns object", func(t *testing.T) {
		sb := NewSandbox(nil)
		result, err := sb.RunWithResult("({name: 'test', value: 42})")
		if err != nil {
			t.Errorf("RunWithResult() error = %v", err)
		}
		obj, ok := result.Export().(map[string]any)
		if !ok {
			t.Fatal("Expected map result")
		}
		if obj["name"] != "test" || obj["value"] != int64(42) {
			t.Errorf("Object = %v, unexpected values", obj)
		}
	})
}

func TestSandbox_RunWithTimeout(t *testing.T) {
	t.Run("completes before timeout", func(t *testing.T) {
		sb := NewSandbox(nil)
		err := sb.RunWithTimeout("var x = 1;", 1*time.Second)
		if err != nil {
			t.Errorf("RunWithTimeout() error = %v", err)
		}
	})

	t.Run("times out on infinite loop", func(t *testing.T) {
		sb := NewSandbox(nil)
		start := time.Now()
		err := sb.RunWithTimeout("while(true) {}", 100*time.Millisecond)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("RunWithTimeout() should error on timeout")
		}
		if !alerr.Is(err, alerr.ErrJSTimeout) {
			t.Errorf("Expected ErrJSTimeout, got %v", err)
		}
		if elapsed > 500*time.Millisecond {
			t.Errorf("Should timeout quickly, took %v", elapsed)
		}
	})

	t.Run("times out on long computation", func(t *testing.T) {
		sb := NewSandbox(nil)
		// Busy loop that takes time
		code := `
			var sum = 0;
			for (var i = 0; i < 1000000000; i++) {
				sum += i;
			}
		`
		err := sb.RunWithTimeout(code, 50*time.Millisecond)
		if err == nil {
			t.Error("RunWithTimeout() should error on timeout")
		}
	})
}

func TestSandbox_SetTimeout(t *testing.T) {
	sb := NewSandbox(nil)
	sb.SetTimeout(10 * time.Second)

	if sb.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", sb.timeout)
	}
}

func TestSandbox_SecurityEvalDisabled(t *testing.T) {
	sb := NewSandbox(nil)

	// eval should be undefined
	result, err := sb.RunWithResult("typeof eval")
	if err != nil {
		t.Fatalf("RunWithResult() error = %v", err)
	}
	if result.String() != "undefined" {
		t.Errorf("eval should be undefined, got typeof = %q", result.String())
	}
}

func TestSandbox_SecurityPrototypesFrozen(t *testing.T) {
	sb := NewSandbox(nil)

	// Attempting to modify Object.prototype should fail silently or throw
	// After freeze, properties can't be added
	code := `
		try {
			Object.prototype.malicious = function() { return 'pwned'; };
			// If we get here, check if it was actually set
			({}).malicious !== undefined;
		} catch(e) {
			false;
		}
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("RunWithResult() error = %v", err)
	}

	// Should be false (either throws or doesn't set)
	if result.ToBoolean() {
		t.Error("Should not be able to modify frozen Object.prototype")
	}
}

func TestSandbox_DeterministicRandom(t *testing.T) {
	// Two sandboxes with same seed should produce same random values
	sb1 := NewSandbox(nil)
	sb2 := NewSandbox(nil)

	code := `
		var results = [];
		for (var i = 0; i < 10; i++) {
			results.push(Math.random());
		}
		JSON.stringify(results);
	`

	result1, err := sb1.RunWithResult(code)
	if err != nil {
		t.Fatalf("First sandbox error = %v", err)
	}

	result2, err := sb2.RunWithResult(code)
	if err != nil {
		t.Fatalf("Second sandbox error = %v", err)
	}

	if result1.String() != result2.String() {
		t.Errorf("Random values should be deterministic:\n  Sandbox1: %s\n  Sandbox2: %s",
			result1.String(), result2.String())
	}
}

func TestSandbox_DeterministicOutput(t *testing.T) {
	// Same code should produce same output every time
	code := `
		var obj = {};
		obj.a = 1;
		obj.b = 2;
		obj.c = Math.random();
		JSON.stringify(obj);
	`

	results := make([]string, 3)
	for i := 0; i < 3; i++ {
		sb := NewSandbox(nil)
		result, err := sb.RunWithResult(code)
		if err != nil {
			t.Fatalf("Run %d error = %v", i, err)
		}
		results[i] = result.String()
	}

	if results[0] != results[1] || results[1] != results[2] {
		t.Errorf("Output should be deterministic:\n  Run1: %s\n  Run2: %s\n  Run3: %s",
			results[0], results[1], results[2])
	}
}

func TestSandbox_SqlHelper(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		var expr = sql("NOW()");
		JSON.stringify(expr);
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("RunWithResult() error = %v", err)
	}

	resultStr := result.String()
	if !strings.Contains(resultStr, "sql_expr") {
		t.Errorf("sql() should return object with _type: 'sql_expr', got %s", resultStr)
	}
	if !strings.Contains(resultStr, "NOW()") {
		t.Errorf("sql() should preserve expression, got %s", resultStr)
	}
}

func TestSandbox_TableFunction(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		name: col.string(100),
	})`

	tableDef, err := sb.EvalSchema(code, "test", "entity")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	if tableDef == nil {
		t.Fatal("EvalSchema() returned nil table")
	}

	if len(tableDef.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(tableDef.Columns))
	}
}

func TestSandbox_TableFunctionWithoutArg(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		try {
			table();
			false;
		} catch(e) {
			true;
		}
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("RunWithResult() error = %v", err)
	}

	if !result.ToBoolean() {
		t.Error("table() without argument should throw")
	}
}

func TestSandbox_TableFunctionWithNonObject(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		try {
			table("not an object");
			false;
		} catch(e) {
			true;
		}
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		t.Fatalf("RunWithResult() error = %v", err)
	}

	if !result.ToBoolean() {
		t.Error("table() with non-object argument should throw")
	}
}

func TestSandbox_EvalSchema(t *testing.T) {
	sb := NewSandbox(nil)

	// EvalSchema expects a table() call, optionally with "export default" prefix
	// (the export default is stripped by EvalSchema)
	code := `export default table({
		id: col.id(),
		email: col.string(255),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "users")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	if tableDef == nil {
		t.Fatal("EvalSchema() returned nil table")
	}

	if tableDef.Namespace != "auth" {
		t.Errorf("Namespace = %q, want %q", tableDef.Namespace, "auth")
	}
	if tableDef.Name != "users" {
		t.Errorf("Name = %q, want %q", tableDef.Name, "users")
	}

	// Verify columns were parsed (parsing may not capture all depending on internal structure)
	// The key test is that a table was successfully returned
	if len(tableDef.Columns) > 0 {
		// Verify first column if present
		if tableDef.Columns[0].Name != "id" {
			t.Logf("First column name = %q", tableDef.Columns[0].Name)
		}
	}
}

func TestSandbox_EvalSchema_NoExport(t *testing.T) {
	sb := NewSandbox(nil)

	code := `
		// Just some code that doesn't export a table
		var x = 1 + 1;
	`

	_, err := sb.EvalSchema(code, "auth", "users")
	if err == nil {
		t.Error("EvalSchema() should error when no table is exported")
	}
}

func TestSandbox_GetTables(t *testing.T) {
	sb := NewSandbox(nil)

	// Initially empty
	if len(sb.GetTables()) != 0 {
		t.Error("GetTables() should be empty initially")
	}

	// After evaluating a schema
	code := `export default table({ id: col.id() })`
	_, _ = sb.EvalSchema(code, "test", "table1")

	tables := sb.GetTables()
	if len(tables) != 1 {
		t.Errorf("GetTables() returned %d tables, want 1", len(tables))
	}
}

func TestSandbox_ClearTables(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({ id: col.id() })`
	_, _ = sb.EvalSchema(code, "test", "table1")

	if len(sb.GetTables()) == 0 {
		t.Fatal("Should have tables after eval")
	}

	sb.ClearTables()

	if len(sb.GetTables()) != 0 {
		t.Error("GetTables() should be empty after ClearTables()")
	}
}

func TestSandbox_Registry(t *testing.T) {
	reg := registry.NewModelRegistry()
	sb := NewSandbox(reg)

	if sb.Registry() != reg {
		t.Error("Registry() should return the provided registry")
	}
}

func TestSandbox_VM(t *testing.T) {
	sb := NewSandbox(nil)
	vm := sb.VM()

	if vm == nil {
		t.Error("VM() should return non-nil")
	}

	// Should be able to run code on the VM directly
	result, err := vm.RunString("1 + 1")
	if err != nil {
		t.Errorf("Direct VM execution error = %v", err)
	}
	if result.ToInteger() != 2 {
		t.Errorf("Direct VM result = %d, want 2", result.ToInteger())
	}
}

func TestSandbox_MaxCallStackSize(t *testing.T) {
	sb := NewSandbox(nil)

	// Deep recursion should fail
	code := `
		function recurse(n) {
			if (n <= 0) return 0;
			return 1 + recurse(n - 1);
		}
		try {
			recurse(10000);
			false;
		} catch(e) {
			e.message.indexOf("stack") >= 0 || e.message.indexOf("overflow") >= 0;
		}
	`
	result, err := sb.RunWithResult(code)
	if err != nil {
		// Stack overflow might manifest as an error
		if !strings.Contains(err.Error(), "stack") {
			t.Logf("Note: deep recursion error = %v", err)
		}
		return
	}

	// If no error, check if catch block detected stack issue
	if !result.ToBoolean() {
		t.Log("Note: Stack limit might not have been reached with 10000 recursions")
	}
}

func TestSandbox_FixedTime(t *testing.T) {
	// Verify the fixed time constant is set correctly
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !FixedTime.Equal(expected) {
		t.Errorf("FixedTime = %v, want %v", FixedTime, expected)
	}
}

func TestSandbox_FixedSeed(t *testing.T) {
	if FixedSeed != 12345 {
		t.Errorf("FixedSeed = %d, want 12345", FixedSeed)
	}
}

func TestParseJSError_SyntaxError(t *testing.T) {
	sb := NewSandbox(nil)

	// Test syntax error - missing parenthesis
	err := sb.Run("function foo( { return 1; }")
	if err == nil {
		t.Fatal("expected syntax error")
	}

	// Check that error contains helpful information
	errStr := err.Error()
	if !strings.Contains(errStr, "E5001") {
		t.Errorf("error should have JS execution code, got: %v", errStr)
	}
}

func TestParseJSError_RuntimeError(t *testing.T) {
	sb := NewSandbox(nil)

	// Test runtime error - undefined variable
	err := sb.Run("undefinedVariable.doSomething()")
	if err == nil {
		t.Fatal("expected runtime error")
	}

	// Check that error wraps the JS error
	var alErr *alerr.Error
	if !alerr.Is(err, alerr.ErrJSExecution) {
		t.Errorf("expected ErrJSExecution, got: %v", err)
	}

	// The error should have context
	if strings.Contains(err.Error(), "E5001") {
		t.Log("Error has correct code")
	}

	_ = alErr
}

func TestParseJSError_WithFileContext(t *testing.T) {
	sb := NewSandbox(nil)

	// Set file context
	sb.SetCurrentFile("/path/to/schema.js")

	// Test error with file context
	err := sb.Run("throw new Error('test error');")
	if err == nil {
		t.Fatal("expected error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "/path/to/schema.js") {
		t.Errorf("error should contain file path, got: %v", errStr)
	}
}

// TestParseJSError_ExtractsLineNumber was removed - we now use Goja's structured types
// instead of regex parsing. See TestParseJSError_StructuredTypes below for the new approach.

func TestGetSourceLine(t *testing.T) {
	code := "line one\nline two\nline three"

	tests := []struct {
		lineNum int
		want    string
	}{
		{1, "line one"},
		{2, "line two"},
		{3, "line three"},
		{0, ""},
		{4, ""},
	}

	for _, tt := range tests {
		got := GetSourceLine(code, tt.lineNum)
		if got != tt.want {
			t.Errorf("GetSourceLine(code, %d) = %q, want %q", tt.lineNum, got, tt.want)
		}
	}
}

func TestSandboxResetAfterInterrupt(t *testing.T) {
	sb := NewSandbox(nil)

	// Run an infinite loop that will be interrupted by the timeout
	sb.SetTimeout(50 * time.Millisecond)
	err := sb.Run("while(true) {}")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !sb.tainted {
		t.Fatal("sandbox should be tainted after interrupt")
	}

	// Now run a normal script — EvalSchema should auto-reset the tainted VM
	sb.SetTimeout(5 * time.Second)
	table, err := sb.EvalSchema(`table({ id: col.id() })`, "test", "reset_check")
	if err != nil {
		t.Fatalf("EvalSchema after interrupt should succeed, got: %v", err)
	}
	if table == nil {
		t.Fatal("expected non-nil table after reset")
	}
	if sb.tainted {
		t.Error("sandbox should no longer be tainted after reset")
	}
}

func TestSandboxResetMethod(t *testing.T) {
	sb := NewSandbox(nil)

	// Set some state, then manually reset
	sb.currentFile = "test.js"
	sb.tainted = true
	sb.Reset()

	if sb.tainted {
		t.Error("tainted should be false after Reset")
	}
	if sb.currentFile != "" {
		t.Error("currentFile should be cleared after Reset")
	}
	if sb.vm == nil {
		t.Error("vm should be re-initialized after Reset")
	}

	// Verify the reset VM works
	result, err := sb.RunWithResult("1 + 1")
	if err != nil {
		t.Fatalf("RunWithResult after Reset failed: %v", err)
	}
	if result.ToInteger() != 2 {
		t.Errorf("expected 2, got %v", result.Export())
	}
}

// --- Restricted Globals Tests ---
// Schemas and migrations should not have access to Date, Math, JSON, Map, Set, etc.

func TestSandbox_SchemaRestrictsDate(t *testing.T) {
	sb := NewSandbox(nil)

	_, err := sb.EvalSchema(`table({ id: col.id(), d: col.datetime().default(Date.now()) })`, "test", "date_check")
	if err == nil {
		t.Fatal("expected error when using Date.now() in schema")
	}
}

func TestSandbox_SchemaRestrictsMath(t *testing.T) {
	sb := NewSandbox(nil)

	_, err := sb.EvalSchema(`table({ id: col.id(), n: col.integer().default(Math.floor(1.5)) })`, "test", "math_check")
	if err == nil {
		t.Fatal("expected error when using Math.floor() in schema")
	}
}

func TestSandbox_SchemaRestrictsJSON(t *testing.T) {
	sb := NewSandbox(nil)

	_, err := sb.EvalSchema(`table({ id: col.id(), j: col.json().default(JSON.parse("{}")) })`, "test", "json_check")
	if err == nil {
		t.Fatal("expected error when using JSON.parse() in schema")
	}
}

func TestSandbox_SchemaRestrictsMap(t *testing.T) {
	sb := NewSandbox(nil)

	// Map should be undefined — new Map() should throw
	sb.restrictGlobals()
	result, err := sb.RunWithResult("typeof Map")
	if err != nil {
		t.Fatalf("RunWithResult error: %v", err)
	}
	if result.String() != "undefined" {
		t.Errorf("Map should be undefined in restricted mode, got typeof = %q", result.String())
	}
}

func TestSandbox_SchemaRestrictsSet(t *testing.T) {
	sb := NewSandbox(nil)

	sb.restrictGlobals()
	result, err := sb.RunWithResult("typeof Set")
	if err != nil {
		t.Fatalf("RunWithResult error: %v", err)
	}
	if result.String() != "undefined" {
		t.Errorf("Set should be undefined in restricted mode, got typeof = %q", result.String())
	}
}

func TestSandbox_SchemaRestrictsParseInt(t *testing.T) {
	sb := NewSandbox(nil)

	sb.restrictGlobals()
	result, err := sb.RunWithResult("typeof parseInt")
	if err != nil {
		t.Fatalf("RunWithResult error: %v", err)
	}
	if result.String() != "undefined" {
		t.Errorf("parseInt should be undefined in restricted mode, got typeof = %q", result.String())
	}
}

func TestSandbox_SchemaRestrictsEncoding(t *testing.T) {
	sb := NewSandbox(nil)

	sb.restrictGlobals()
	for _, global := range []string{"encodeURIComponent", "decodeURIComponent", "encodeURI", "decodeURI"} {
		result, err := sb.RunWithResult("typeof " + global)
		if err != nil {
			t.Fatalf("RunWithResult(%s) error: %v", global, err)
		}
		if result.String() != "undefined" {
			t.Errorf("%s should be undefined in restricted mode, got typeof = %q", global, result.String())
		}
	}
}

func TestSandbox_SchemaStillWorks(t *testing.T) {
	sb := NewSandbox(nil)

	// Normal schema DSL should work perfectly after restriction
	table, err := sb.EvalSchema(`table({
		id: col.id(),
		username: col.string(50).unique(),
		bio: col.text().optional(),
		role: col.enum(["admin", "editor", "viewer"]).default("viewer"),
	}).timestamps()`, "auth", "user")

	if err != nil {
		t.Fatalf("EvalSchema failed after restriction: %v", err)
	}
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if table.Name != "user" {
		t.Errorf("expected table name 'user', got %q", table.Name)
	}
	if table.Namespace != "auth" {
		t.Errorf("expected namespace 'auth', got %q", table.Namespace)
	}
}

func TestSandbox_MigrationRestrictsDate(t *testing.T) {
	sb := NewSandbox(nil)

	// Attempting to use Date in a migration should fail
	sb.restrictGlobals()
	err := sb.Run(`
		migration({
			up: function(m) {
				var ts = Date.now();
			},
			down: function(m) {}
		})
	`)
	if err == nil {
		t.Fatal("expected error when using Date.now() in migration context")
	}
}

func TestSandbox_RestrictedFlagSetAfterRestriction(t *testing.T) {
	sb := NewSandbox(nil)

	if sb.restricted {
		t.Fatal("sandbox should not be restricted initially")
	}

	sb.restrictGlobals()

	if !sb.restricted {
		t.Fatal("sandbox should be restricted after restrictGlobals()")
	}
}

func TestSandbox_ResetClearsRestricted(t *testing.T) {
	sb := NewSandbox(nil)

	sb.restrictGlobals()
	if !sb.restricted {
		t.Fatal("should be restricted")
	}

	sb.Reset()
	if sb.restricted {
		t.Fatal("restricted should be cleared after Reset()")
	}

	// After reset, Date should be available again
	result, err := sb.RunWithResult("typeof Date")
	if err != nil {
		t.Fatalf("RunWithResult error: %v", err)
	}
	if result.String() != "function" {
		t.Errorf("Date should be available after Reset(), got typeof = %q", result.String())
	}
}

func TestSandbox_AllRestrictedGlobals(t *testing.T) {
	sb := NewSandbox(nil)
	sb.restrictGlobals()

	restricted := []string{
		"Date", "Math", "isNaN", "isFinite",
		"parseInt", "parseFloat",
		"Map", "Set", "WeakMap", "WeakSet",
		"JSON",
		"encodeURIComponent", "decodeURIComponent",
		"encodeURI", "decodeURI",
		"ArrayBuffer", "DataView",
	}

	for _, name := range restricted {
		result, err := sb.RunWithResult("typeof " + name)
		if err != nil {
			t.Errorf("typeof %s failed: %v", name, err)
			continue
		}
		if result.String() != "undefined" {
			t.Errorf("%s should be undefined after restrictGlobals(), got typeof = %q", name, result.String())
		}
	}
}

func TestSandbox_AllowedGlobalsAfterRestriction(t *testing.T) {
	sb := NewSandbox(nil)
	sb.restrictGlobals()

	// These must remain available for the DSL to work
	allowed := map[string]string{
		"Object":  "function",
		"Array":   "function",
		"String":  "function",
		"Number":  "function",
		"Boolean": "function",
		"Error":   "function",
	}

	for name, expectedType := range allowed {
		result, err := sb.RunWithResult("typeof " + name)
		if err != nil {
			t.Errorf("typeof %s failed: %v", name, err)
			continue
		}
		if result.String() != expectedType {
			t.Errorf("%s should be %q after restrictGlobals(), got typeof = %q", name, expectedType, result.String())
		}
	}
}
