package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

func init() {
	// Use plain mode for deterministic test output
	SetDefault(&Config{Mode: ModePlain})
}

func TestFormatError_Nil(t *testing.T) {
	result := FormatError(nil)
	if result != "" {
		t.Errorf("FormatError(nil) = %q, want empty string", result)
	}
}

func TestFormatError_GenericError(t *testing.T) {
	err := errors.New("something went wrong")
	result := FormatError(err)

	if !strings.Contains(result, "error") {
		t.Errorf("result should contain 'error': %q", result)
	}
	if !strings.Contains(result, "something went wrong") {
		t.Errorf("result should contain error message: %q", result)
	}
}

func TestFormatError_AlabError(t *testing.T) {
	err := alerr.New(alerr.ErrInvalidSnakeCase, "column name must be snake_case").
		WithTable("auth", "user").
		WithColumn("userName")

	result := FormatError(err)

	// Should contain error code
	if !strings.Contains(result, "E2002") {
		t.Errorf("result should contain error code E2002: %q", result)
	}
	// Should contain message
	if !strings.Contains(result, "column name must be snake_case") {
		t.Errorf("result should contain error message: %q", result)
	}
}

func TestFormatError_WithFileLocation(t *testing.T) {
	err := alerr.New(alerr.ErrSchemaInvalid, "invalid schema").
		WithLocation("schemas/auth/user.js", 15, 5)

	result := FormatError(err)

	// Should contain file path
	if !strings.Contains(result, "schemas/auth/user.js") {
		t.Errorf("result should contain file path: %q", result)
	}
	// Should contain line number
	if !strings.Contains(result, "15") {
		t.Errorf("result should contain line number: %q", result)
	}
}

func TestFormatError_WithSourceContext(t *testing.T) {
	err := alerr.New(alerr.ErrInvalidSnakeCase, "column name must be snake_case").
		WithLocation("schemas/auth/user.js", 15, 3).
		WithSource("  userName: col.string(50),").
		WithSpan(3, 11).
		WithLabel("should be user_name")

	result := FormatError(err)

	// Should contain source line
	if !strings.Contains(result, "userName: col.string(50)") {
		t.Errorf("result should contain source line: %q", result)
	}
	// Should contain pointer characters
	if !strings.Contains(result, "^") {
		t.Errorf("result should contain pointer: %q", result)
	}
}

func TestFormatError_WithNotes(t *testing.T) {
	err := alerr.New(alerr.ErrInvalidType, "int64 is not allowed").
		WithNote("JavaScript cannot safely represent integers > 2^53").
		WithNote("Use integer (32-bit) or string instead")

	result := FormatError(err)

	// Should contain notes
	if !strings.Contains(result, "note") {
		t.Errorf("result should contain 'note': %q", result)
	}
	if !strings.Contains(result, "JavaScript cannot safely") {
		t.Errorf("result should contain first note: %q", result)
	}
}

func TestFormatError_WithHelps(t *testing.T) {
	err := alerr.New(alerr.ErrMissingLength, "string column requires length").
		WithHelp("try col.string(255)").
		WithHelp("or use col.text() for unlimited length")

	result := FormatError(err)

	// Should contain helps
	if !strings.Contains(result, "help") {
		t.Errorf("result should contain 'help': %q", result)
	}
	if !strings.Contains(result, "col.string(255)") {
		t.Errorf("result should contain help suggestion: %q", result)
	}
}

func TestFormatError_WithCause(t *testing.T) {
	cause := errors.New("underlying database error")
	err := alerr.Wrap(alerr.ErrSQLExecution, cause, "failed to execute migration")

	result := FormatError(err)

	// Should contain cause
	if !strings.Contains(result, "cause") {
		t.Errorf("result should contain 'cause': %q", result)
	}
	if !strings.Contains(result, "underlying database error") {
		t.Errorf("result should contain cause message: %q", result)
	}
}

func TestFormatWarning(t *testing.T) {
	result := FormatWarning("destructive operation detected")

	if !strings.Contains(result, "warning") {
		t.Errorf("result should contain 'warning': %q", result)
	}
	if !strings.Contains(result, "destructive operation detected") {
		t.Errorf("result should contain message: %q", result)
	}
}

func TestFormatWarning_WithOptions(t *testing.T) {
	result := FormatWarning("destructive operation",
		WithFile("migrations/003_cleanup.js", 12, 0),
		WithSource("m.drop_table(\"legacy_data\")", 3, 15, "will delete data"),
		WithNotes("this will DELETE ALL DATA"),
		WithHelps("run with --confirm-destructive"),
	)

	if !strings.Contains(result, "warning") {
		t.Errorf("result should contain 'warning': %q", result)
	}
	if !strings.Contains(result, "migrations/003_cleanup.js") {
		t.Errorf("result should contain file: %q", result)
	}
	if !strings.Contains(result, "note") {
		t.Errorf("result should contain note: %q", result)
	}
	if !strings.Contains(result, "help") {
		t.Errorf("result should contain help: %q", result)
	}
}

func TestFormatNote(t *testing.T) {
	result := FormatNote("this is additional information")

	if !strings.Contains(result, "note") {
		t.Errorf("result should contain 'note': %q", result)
	}
	if !strings.Contains(result, "this is additional information") {
		t.Errorf("result should contain message: %q", result)
	}
}

func TestFormatHelp(t *testing.T) {
	result := FormatHelp("try running with --verbose")

	if !strings.Contains(result, "help") {
		t.Errorf("result should contain 'help': %q", result)
	}
	if !strings.Contains(result, "try running with --verbose") {
		t.Errorf("result should contain message: %q", result)
	}
}

func TestFormatSuccess(t *testing.T) {
	result := FormatSuccess("migration applied")

	if !strings.Contains(result, "success") {
		t.Errorf("result should contain 'success': %q", result)
	}
	if !strings.Contains(result, "migration applied") {
		t.Errorf("result should contain message: %q", result)
	}
}

func TestDiagnosticOptions(t *testing.T) {
	t.Run("WithFile", func(t *testing.T) {
		d := &DiagnosticMessage{}
		WithFile("test.js", 10, 5)(d)
		if d.File != "test.js" {
			t.Errorf("File = %q, want %q", d.File, "test.js")
		}
		if d.Line != 10 {
			t.Errorf("Line = %d, want %d", d.Line, 10)
		}
		if d.Column != 5 {
			t.Errorf("Column = %d, want %d", d.Column, 5)
		}
	})

	t.Run("WithSource", func(t *testing.T) {
		d := &DiagnosticMessage{}
		WithSource("source code", 1, 10, "label")(d)
		if d.Source != "source code" {
			t.Errorf("Source = %q, want %q", d.Source, "source code")
		}
		if d.Span != [2]int{1, 10} {
			t.Errorf("Span = %v, want %v", d.Span, [2]int{1, 10})
		}
		if d.Label != "label" {
			t.Errorf("Label = %q, want %q", d.Label, "label")
		}
	})

	t.Run("WithNotes", func(t *testing.T) {
		d := &DiagnosticMessage{}
		WithNotes("note1", "note2")(d)
		if len(d.Notes) != 2 {
			t.Errorf("len(Notes) = %d, want 2", len(d.Notes))
		}
		if d.Notes[0] != "note1" || d.Notes[1] != "note2" {
			t.Errorf("Notes = %v, want [note1 note2]", d.Notes)
		}
	})

	t.Run("WithHelps", func(t *testing.T) {
		d := &DiagnosticMessage{}
		WithHelps("help1", "help2")(d)
		if len(d.Helps) != 2 {
			t.Errorf("len(Helps) = %d, want 2", len(d.Helps))
		}
	})
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		msgType MessageType
		name    string
	}{
		{TypeError, "error"},
		{TypeWarning, "warning"},
		{TypeNote, "note"},
		{TypeHelp, "help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DiagnosticMessage{
				Type:    tt.msgType,
				Message: "test message",
			}
			result := formatDiagnostic(d)
			if !strings.Contains(result, tt.name) {
				t.Errorf("result should contain %q: %q", tt.name, result)
			}
		})
	}
}

func TestFormatDiagnostic_WithCode(t *testing.T) {
	d := &DiagnosticMessage{
		Type:    TypeError,
		Code:    "E1001",
		Message: "test error",
	}
	result := formatDiagnostic(d)

	if !strings.Contains(result, "E1001") {
		t.Errorf("result should contain error code: %q", result)
	}
	if !strings.Contains(result, "[") && !strings.Contains(result, "]") {
		t.Errorf("error code should be bracketed: %q", result)
	}
}

func TestFormatDiagnostic_WithFileLineColumn(t *testing.T) {
	d := &DiagnosticMessage{
		Type:    TypeError,
		Message: "test error",
		File:    "test.js",
		Line:    10,
		Column:  5,
	}
	result := formatDiagnostic(d)

	if !strings.Contains(result, "test.js:10:5") {
		t.Errorf("result should contain file:line:column: %q", result)
	}
}
