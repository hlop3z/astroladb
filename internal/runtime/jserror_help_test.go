package runtime

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// addJSErrorHelp Tests
// -----------------------------------------------------------------------------

func TestAddJSErrorHelp(t *testing.T) {
	t.Run("restricted_global_date", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "Date is not defined")
		addJSErrorHelp(err, "Date is not defined")

		notes := err.Notes()
		helps := err.Helps()

		if len(notes) == 0 || !strings.Contains(notes[0], "declarative") {
			t.Errorf("Expected note about declarative schemas, got: %v", notes)
		}
		if len(helps) == 0 || !strings.Contains(helps[0], "col.datetime()") {
			t.Errorf("Expected help about col.datetime(), got: %v", helps)
		}
	})

	t.Run("restricted_global_math", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "Math is not defined")
		addJSErrorHelp(err, "Math is not defined")

		helps := err.Helps()
		if len(helps) == 0 {
			t.Error("Expected help message")
		} else if !strings.Contains(helps[0], "sql()") && !strings.Contains(helps[0], "fn.*()") {
			t.Errorf("Expected help about sql() or fn.*(), got: %s", helps[0])
		}
	})

	t.Run("restricted_global_json", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "JSON is not defined")
		addJSErrorHelp(err, "JSON is not defined")

		helps := err.Helps()
		if len(helps) == 0 || !strings.Contains(helps[0], "col.json()") {
			t.Errorf("Expected help about col.json(), got: %v", helps)
		}
	})

	t.Run("restricted_global_map", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "Map is not defined")
		addJSErrorHelp(err, "Map is not defined")

		helps := err.Helps()
		if len(helps) == 0 || !strings.Contains(helps[0], "plain objects {}") {
			t.Errorf("Expected help about plain objects, got: %v", helps)
		}
	})

	t.Run("restricted_global_set", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "Set is not defined")
		addJSErrorHelp(err, "Set is not defined")

		helps := err.Helps()
		if len(helps) == 0 || !strings.Contains(helps[0], "arrays []") {
			t.Errorf("Expected help about arrays, got: %v", helps)
		}
	})

	t.Run("restricted_global_parseint", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "parseInt is not defined")
		addJSErrorHelp(err, "parseInt is not defined")

		helps := err.Helps()
		if len(helps) == 0 || !strings.Contains(helps[0], "integer literals") {
			t.Errorf("Expected help about integer literals, got: %v", helps)
		}
	})

	t.Run("case_insensitive_matching", func(t *testing.T) {
		// Should match "date" case-insensitively
		err := alerr.New(alerr.ErrJSExecution, "DATE is not defined")
		addJSErrorHelp(err, "DATE is not defined")

		helps := err.Helps()
		if len(helps) == 0 || !strings.Contains(helps[0], "col.datetime()") {
			t.Errorf("Expected case-insensitive match for DATE, got: %v", helps)
		}
	})

	t.Run("undefined_error", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "foo is undefined")
		addJSErrorHelp(err, "foo is undefined")

		notes := err.Notes()
		if len(notes) == 0 || !strings.Contains(notes[0], "undefined") {
			t.Errorf("Expected note about undefined, got: %v", notes)
		}
	})

	t.Run("undefined_error_with_col", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "col is undefined")
		addJSErrorHelp(err, "col is undefined")

		notes := err.Notes()
		helps := err.Helps()

		if len(notes) == 0 || !strings.Contains(notes[0], "undefined") {
			t.Errorf("Expected note about undefined, got: %v", notes)
		}
		if len(helps) == 0 || !strings.Contains(helps[0], "col") {
			t.Errorf("Expected help about 'col', got: %v", helps)
		}
	})

	t.Run("not_a_function_error", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "foo.bar is not a function")
		addJSErrorHelp(err, "foo.bar is not a function")

		notes := err.Notes()
		helps := err.Helps()

		if len(notes) == 0 || !strings.Contains(notes[0], "not a function") {
			t.Errorf("Expected note about 'not a function', got: %v", notes)
		}
		if len(helps) == 0 || !strings.Contains(helps[0], "method name") {
			t.Errorf("Expected help about method name, got: %v", helps)
		}
	})

	t.Run("syntax_error", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "SyntaxError: Unexpected token }")
		addJSErrorHelp(err, "SyntaxError: Unexpected token }")

		notes := err.Notes()
		if len(notes) == 0 || !strings.Contains(notes[0], "syntax") {
			t.Errorf("Expected note about syntax, got: %v", notes)
		}
	})

	t.Run("unexpected_token_error", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "Unexpected token ,")
		addJSErrorHelp(err, "Unexpected token ,")

		notes := err.Notes()
		helps := err.Helps()

		if len(notes) == 0 || !strings.Contains(notes[0], "unexpected") {
			t.Errorf("Expected note about unexpected token, got: %v", notes)
		}
		if len(helps) == 0 || !strings.Contains(helps[0], "typos") {
			t.Errorf("Expected help about typos, got: %v", helps)
		}
	})

	t.Run("reference_error", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "ReferenceError: x is not defined")
		addJSErrorHelp(err, "ReferenceError: x is not defined")

		notes := err.Notes()
		if len(notes) == 0 || !strings.Contains(notes[0], "reference") {
			t.Errorf("Expected note about reference, got: %v", notes)
		}
	})

	t.Run("type_error", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "TypeError: Cannot read property")
		addJSErrorHelp(err, "TypeError: Cannot read property")

		notes := err.Notes()
		if len(notes) == 0 || !strings.Contains(notes[0], "type") {
			t.Errorf("Expected note about type, got: %v", notes)
		}
	})

	t.Run("no_matching_pattern", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "Some random error message")
		addJSErrorHelp(err, "Some random error message")

		// Should not panic, just no help added
		notes := err.Notes()
		helps := err.Helps()

		// Note and help should be empty or unchanged
		_ = notes
		_ = helps
	})

	t.Run("empty_message", func(t *testing.T) {
		err := alerr.New(alerr.ErrJSExecution, "")
		addJSErrorHelp(err, "")

		// Should not panic with empty message
		notes := err.Notes()
		helps := err.Helps()

		_ = notes
		_ = helps
	})

	t.Run("restricted_global_takes_precedence", func(t *testing.T) {
		// Message contains both "date" and "undefined", but restricted global should win
		err := alerr.New(alerr.ErrJSExecution, "Date is undefined")
		addJSErrorHelp(err, "Date is undefined")

		helps := err.Helps()
		// Should get date-specific help, not generic undefined help
		if len(helps) == 0 || !strings.Contains(helps[0], "col.datetime()") {
			t.Errorf("Expected restricted global help to take precedence, got: %v", helps)
		}
	})

	t.Run("multiple_keywords_first_match_wins", func(t *testing.T) {
		// Contains both "syntax" and "unexpected token"
		// Should match first pattern it encounters in switch statement
		err := alerr.New(alerr.ErrJSExecution, "SyntaxError: Unexpected token")
		addJSErrorHelp(err, "SyntaxError: Unexpected token")

		notes := err.Notes()
		// Should match syntax error since it comes first in message
		if len(notes) == 0 || (!strings.Contains(notes[0], "syntax") && !strings.Contains(notes[0], "unexpected")) {
			t.Errorf("Expected either syntax or unexpected token note, got: %v", notes)
		}
	})
}
