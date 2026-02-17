package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

func init() {
	// Force plain mode in tests so style functions return raw text (no ANSI codes).
	SetDefault(&Config{Mode: ModePlain})
}

// ---------------------------------------------------------------------------
// FormatError — full source context
// ---------------------------------------------------------------------------

func TestFormatError_FullSourceContext(t *testing.T) {
	err := alerr.New(alerr.ErrMissingReference, "belongs_to() requires a table reference").
		WithLocation("schemas/auth/role.js", 5, 18).
		WithSource("  owner: col.belongs_to()").
		WithSpan(10, 25).
		WithHelp("try col.belongs_to('namespace.table') or col.belongs_to('.table') for same namespace")

	output := FormatError(err)

	checks := []string{
		"error",
		"E2009",
		"belongs_to() requires a table reference",
		"-->",
		"schemas/auth/role.js:5:18",
		"5",              // line number
		"col.belongs_to", // source text
		"^",              // caret pointer
		"help:",
		"col.belongs_to('namespace.table')",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// FormatError — file only (no line number)
// ---------------------------------------------------------------------------

func TestFormatError_FileOnly(t *testing.T) {
	err := alerr.New(alerr.ErrInvalidReference, "foreign key references unknown table").
		WithLocation("schemas/auth/role.js", 0, 0).
		With("table", "_auth_role").
		With("referenced_table", "auth.user")

	output := FormatError(err)

	checks := []string{
		"error",
		"E2003",
		"foreign key references unknown table",
		"-->",
		"schemas/auth/role.js",
		"table: _auth_role",
		"referenced_table: auth.user",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}

	// Should NOT have ":0" or source/caret lines when line==0
	if strings.Contains(output, "schemas/auth/role.js:0") {
		t.Errorf("FormatError should not include :0 for line 0\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// FormatError — notes and helps
// ---------------------------------------------------------------------------

func TestFormatError_NotesAndHelps(t *testing.T) {
	err := alerr.New(alerr.ErrJSExecution, "undefined variable").
		WithLocation("schemas/test.js", 3, 1).
		WithSource("  foo.bar()").
		WithNote("JavaScript 'undefined' error - a variable or function was not found").
		WithHelp("ensure 'col' is available in your schema file")

	output := FormatError(err)

	if !strings.Contains(output, "note:") {
		t.Errorf("expected 'note:' in output\ngot:\n%s", output)
	}
	if !strings.Contains(output, "help:") {
		t.Errorf("expected 'help:' in output\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// FormatError — wrapped cause
// ---------------------------------------------------------------------------

func TestFormatError_WithCause(t *testing.T) {
	cause := errors.New("column 'email' does not exist")
	err := alerr.Wrap(alerr.ErrSQLExecution, cause, "failed to alter table").
		WithLocation("migrations/001.js", 10, 0)

	output := FormatError(err)

	if !strings.Contains(output, "cause:") {
		t.Errorf("expected 'cause:' in output\ngot:\n%s", output)
	}
	if !strings.Contains(output, "column 'email' does not exist") {
		t.Errorf("expected cause message in output\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// cleanCauseMessage tests
// ---------------------------------------------------------------------------

func TestFormatError_CleanCause_GojaStack(t *testing.T) {
	msg := "some error at github.com/dop251/goja/runtime.go:42 func (native)"
	got := cleanCauseMessage(msg)
	if strings.Contains(got, "github.com") {
		t.Errorf("Goja stack trace not stripped\ngot: %s", got)
	}
	if !strings.Contains(got, "some error") {
		t.Errorf("expected cause message preserved\ngot: %s", got)
	}
}

func TestFormatError_CleanCause_ErrorCode(t *testing.T) {
	msg := "[E2009] belongs_to() requires a reference"
	got := cleanCauseMessage(msg)
	if strings.Contains(got, "[E2009]") {
		t.Errorf("error code not stripped\ngot: %s", got)
	}
	if !strings.Contains(got, "belongs_to() requires a reference") {
		t.Errorf("cause text lost\ngot: %s", got)
	}
}

func TestFormatError_CleanCause_PipeFormat(t *testing.T) {
	msg := "string() requires a length argument|try col.string(255) for VARCHAR(255)"
	got := cleanCauseMessage(msg)
	if !strings.Contains(got, "string() requires a length argument") {
		t.Errorf("cause portion lost\ngot: %s", got)
	}
	if !strings.Contains(got, "help:") {
		t.Errorf("pipe-format help not rendered\ngot: %s", got)
	}
	if !strings.Contains(got, "col.string(255)") {
		t.Errorf("help text lost\ngot: %s", got)
	}
}

func TestFormatError_CleanCause_SyntaxError(t *testing.T) {
	msg := "SyntaxError: SyntaxError: (anonymous): Line 7:3 Unexpected identifier (and 6 more errors)"
	got := cleanCauseMessage(msg)
	if strings.Contains(got, "SyntaxError: SyntaxError:") {
		t.Errorf("double SyntaxError prefix not cleaned\ngot: %s", got)
	}
	if strings.Contains(got, "Line 7:3") {
		t.Errorf("Line X:Y prefix not stripped\ngot: %s", got)
	}
	if !strings.Contains(got, "unexpected identifier") {
		t.Errorf("core message lost\ngot: %s", got)
	}
}

// ---------------------------------------------------------------------------
// FormatError — generic (non-alerr) error
// ---------------------------------------------------------------------------

func TestFormatError_GenericError(t *testing.T) {
	err := errors.New("something went wrong")
	output := FormatError(err)

	if !strings.Contains(output, "error:") {
		t.Errorf("expected 'error:' in output\ngot:\n%s", output)
	}
	if !strings.Contains(output, "something went wrong") {
		t.Errorf("expected message in output\ngot:\n%s", output)
	}
	// Should NOT contain brackets since it's not structured
	if strings.Contains(output, "[E") {
		t.Errorf("generic error should not have error code\ngot:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// FormatError — nil error
// ---------------------------------------------------------------------------

func TestFormatError_Nil(t *testing.T) {
	output := FormatError(nil)
	if output != "" {
		t.Errorf("FormatError(nil) should return empty string\ngot: %q", output)
	}
}
