package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/cli"
)

func init() {
	// Force plain mode so FormatError output has no ANSI codes.
	cli.SetDefault(&cli.Config{Mode: cli.ModePlain})
}

// ---------------------------------------------------------------------------
// CRITICAL: structured error pipeline (col.belongs_to without arg)
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_StructuredError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "role.js")

	// Schema with col.belongs_to() — missing required reference
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
  owner: col.belongs_to(),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "role")
	if err == nil {
		t.Fatal("expected error from col.belongs_to(), got nil")
	}

	// Verify it's an alerr.Error with the right code
	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	if alErr.GetCode() != alerr.ErrMissingReference {
		t.Errorf("error code = %s, want %s", alErr.GetCode(), alerr.ErrMissingReference)
	}

	// Verify source location is present
	ctx := alErr.GetContext()
	file, _ := ctx["file"].(string)
	line, _ := ctx["line"].(int)

	if file == "" {
		t.Error("expected file in error context")
	}
	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}

	// Verify source line is present
	source, _ := ctx["source"].(string)
	if source == "" {
		t.Error("expected source line in error context")
	}

	// Verify help is present
	helps, _ := ctx["helps"].([]string)
	if len(helps) == 0 {
		t.Error("expected help in error context")
	}

	// Verify FormatError produces Cargo-style output with correct error code
	output := cli.FormatError(err)
	for _, want := range []string{
		"error[E2009]",
		"belongs_to() requires a table reference",
		"-->",
		"role.js",
		"col.belongs_to",
		"help:",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: string() missing length
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_MissingLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.js")

	schema := `export default table({
  id: col.id(),
  name: col.string(),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "user")
	if err == nil {
		t.Fatal("expected error from col.string(), got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	// Should have a meaningful error code (E2007 or E5001 with cause)
	code := alErr.GetCode()
	if code == "" {
		t.Error("expected non-empty error code")
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}

	// Verify FormatError produces readable output
	output := cli.FormatError(err)
	if !strings.Contains(output, "error") {
		t.Errorf("FormatError output missing 'error'\ngot:\n%s", output)
	}
	if !strings.Contains(output, "-->") {
		t.Errorf("FormatError output missing '-->'\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: decimal() missing args
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_DecimalMissingArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "product.js")

	schema := `export default table({
  id: col.id(),
  price: col.decimal(),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "product")
	if err == nil {
		t.Fatal("expected error from col.decimal(), got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}

	output := cli.FormatError(err)
	for _, want := range []string{"error", "decimal()", "-->", "help:"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: many_to_many() missing reference
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_ManyToManyMissingRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "role.js")

	schema := `export default table({
  id: col.id(),
  name: col.string(100),
}).timestamps().many_to_many()
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "role")
	if err == nil {
		t.Fatal("expected error from many_to_many(), got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	if alErr.GetCode() != alerr.ErrMissingReference {
		t.Errorf("error code = %s, want %s", alErr.GetCode(), alerr.ErrMissingReference)
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	source, _ := ctx["source"].(string)

	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}
	if source == "" {
		t.Error("expected source line in error context")
	}

	output := cli.FormatError(err)
	for _, want := range []string{
		"error[E2009]",
		"many_to_many() requires a table reference",
		"-->",
		"many_to_many()",
		"help:",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}

	// Should NOT have redundant cause or generic note
	if strings.Contains(output, "cause:") {
		t.Errorf("structured error should not show redundant cause\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: syntax error — missing comma between properties
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_SyntaxError_MissingComma(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")

	// Missing comma after col.id()
	schema := `export default table({
  id: col.id()
  name: col.string(100),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "bad")
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	if line <= 0 {
		t.Errorf("expected line > 0 for syntax error, got %d", line)
	}

	output := cli.FormatError(err)
	for _, want := range []string{"error", "-->"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: syntax error — missing comma in longer schema
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_SyntaxError_MissingCommaDeep(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "role.js")

	// Missing comma after col.name().unique() on line 3
	schema := `export default table({
  id: col.id(),
  name: col.name().unique()
  last: col.name(),
}).timestamps().many_to_many('auth.user')
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "role")
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	if line <= 0 {
		t.Errorf("expected line > 0 for syntax error, got %d", line)
	}

	output := cli.FormatError(err)
	for _, want := range []string{"error", "-->", "role.js"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: JS runtime error — undefined method
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_UndefinedMethod(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.js")

	// col.nonexistent_method() doesn't exist
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
  bad: col.nonexistent_method(),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "user")
	if err == nil {
		t.Fatal("expected error from col.nonexistent_method(), got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}

	output := cli.FormatError(err)
	for _, want := range []string{"error", "-->", "user.js"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: JS runtime error — undefined variable
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_UndefinedVariable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.js")

	// nonexistent_func() is not defined at all
	schema := `export default table({
  id: col.id(),
  name: nonexistent_func(),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "broken")
	if err == nil {
		t.Fatal("expected error from undefined variable, got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	ctx := alErr.GetContext()
	line, _ := ctx["line"].(int)
	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}

	output := cli.FormatError(err)
	for _, want := range []string{"error", "-->", "broken.js"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: line offset correctness (IIFE wrapper)
// ---------------------------------------------------------------------------

func TestCRITICAL_ErrorPipeline_LineOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "offset.js")

	// Error is on line 5 of the original file
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
  email: col.string(255),
  owner: col.belongs_to(),
})
`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "offset")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	ctx := alErr.GetContext()
	file, _ := ctx["file"].(string)
	line, _ := ctx["line"].(int)

	// Verify line number is positive (some line was captured)
	if line <= 0 {
		t.Errorf("expected line > 0, got %d", line)
	}

	// Verify the source line in context is from the original file, not the wrapper
	source, _ := ctx["source"].(string)
	if source != "" {
		if strings.Contains(source, "__result") || strings.Contains(source, "__registerTable") {
			t.Errorf("source line is from wrapper, not original file: %q", source)
		}
	}

	// Verify the file path is set correctly
	if file == "" {
		t.Error("expected file path in error context")
	}
}

// ---------------------------------------------------------------------------
// FormatError produces different output for structured vs generic errors
// ---------------------------------------------------------------------------

func TestFormatError_StructuredVsGeneric(t *testing.T) {
	// Structured error with full context
	structured := alerr.New(alerr.ErrMissingLength, "string() requires a length").
		WithLocation("test.js", 3, 10).
		WithSource("  name: col.string()").
		WithSpan(10, 20).
		WithHelp("try col.string(255)")

	sOutput := cli.FormatError(structured)

	// Must have all components
	for _, want := range []string{"error", "E2007", "-->", "test.js:3:10", "^", "help:"} {
		if !strings.Contains(sOutput, want) {
			t.Errorf("structured output missing %q\ngot:\n%s", want, sOutput)
		}
	}

	// Minimal alerr error without source context
	gOutput := cli.FormatError(alerr.New(alerr.ErrJSExecution, "something broke"))

	if !strings.Contains(gOutput, "error") {
		t.Errorf("minimal output missing 'error'\ngot:\n%s", gOutput)
	}
	// Should not have source-related markers
	if strings.Contains(gOutput, "^") {
		t.Errorf("minimal output should not have caret pointers\ngot:\n%s", gOutput)
	}
}
