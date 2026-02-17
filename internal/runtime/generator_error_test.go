package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// ---------------------------------------------------------------------------
// Helper: run a generator through the sandbox
// ---------------------------------------------------------------------------

func runGenerator(t *testing.T, code string) error {
	t.Helper()
	sb := NewSandbox(nil)
	schema := map[string]any{"tables": []any{}}
	_, err := sb.RunGenerator(code, schema)
	return err
}

// ---------------------------------------------------------------------------
// CRITICAL: gen() with no arguments
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_GenNoArgs(t *testing.T) {
	err := runGenerator(t, `gen()`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: gen() with non-function argument
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_GenNotFunction(t *testing.T) {
	err := runGenerator(t, `gen("string")`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: render() with no arguments
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_RenderNoArgs(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { render() })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: render() with non-object argument
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_RenderNotObject(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { render("string") })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: render() with array argument
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_RenderArray(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { render([1, 2]) })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: render() with non-string values
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_RenderNonStringValues(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { render({ "f.txt": 42 }) })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: perTable() with no arguments
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_PerTableNoArgs(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { perTable() })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: perTable() with bad function argument
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_PerTableBadFn(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { perTable(s, "not fn") })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: json() with no arguments
// ---------------------------------------------------------------------------

func TestCRITICAL_GeneratorError_JsonNoArgs(t *testing.T) {
	err := runGenerator(t, `gen(function(s) { json() })`)
	assertStructuredError(t, err, alerr.ErrJSExecution, "error[GEN-001]", "help:")
}
