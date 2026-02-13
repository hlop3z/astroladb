package runner

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
)

// newTestRunner creates a Runner with a dialect for testing formatBackfillValue.
func newTestRunner(d dialect.Dialect) *Runner {
	return &Runner{dialect: d}
}

// TestFormatBackfillValue_StringEscaping tests proper escaping of string values.
func TestFormatBackfillValue_StringEscaping(t *testing.T) {
	r := newTestRunner(dialect.Postgres())

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{name: "simple string", input: "hello", expected: "'hello'"},
		{name: "empty string", input: "", expected: "''"},
		{name: "single quote", input: "O'Reilly", expected: "'O''Reilly'"},
		{name: "multiple single quotes", input: "it''s a ''test''", expected: "'it''''s a ''''test'''''"},
		{name: "double quotes", input: `say "hello"`, expected: `'say "hello"'`},
		{name: "backslash", input: `path\to\file`, expected: `'path\to\file'`},
		{name: "newline", input: "line1\nline2", expected: "'line1\nline2'"},
		{name: "carriage return", input: "line1\rline2", expected: "'line1\rline2'"},
		{name: "tab", input: "col1\tcol2", expected: "'col1\tcol2'"},
		{name: "null byte", input: "before\x00after", expected: "'before\x00after'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.formatBackfillValue(tt.input)
			if got != tt.expected {
				t.Errorf("formatBackfillValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestFormatBackfillValue_SQLInjection tests that SQL injection attempts are neutralized.
func TestFormatBackfillValue_SQLInjection(t *testing.T) {
	r := newTestRunner(dialect.Postgres())

	tests := []struct {
		name             string
		input            any
		shouldNotContain []string
	}{
		{
			name:             "basic injection",
			input:            "'; DROP TABLE users; --",
			shouldNotContain: []string{"'; DROP"},
		},
		{
			name:             "comment injection",
			input:            "value' -- comment",
			shouldNotContain: []string{"' --"},
		},
		{
			name:             "union injection",
			input:            "' UNION SELECT * FROM passwords --",
			shouldNotContain: []string{"' UNION"},
		},
		{
			name:             "multi-statement injection",
			input:            "value'; INSERT INTO admin VALUES('hacker','pass'); --",
			shouldNotContain: []string{"'; INSERT"},
		},
		{
			name:             "nested quotes injection",
			input:            "''''; DROP TABLE users; --",
			shouldNotContain: nil, // just verify it doesn't produce unquoted SQL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.formatBackfillValue(tt.input)

			// Result must start and end with single quotes (it's a string literal)
			if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
				t.Errorf("result not properly quoted: %q", got)
			}

			// Verify the inner content has all single quotes escaped
			inner := got[1 : len(got)-1]
			// After unescaping '', there should be no lone single quotes
			unescaped := strings.ReplaceAll(inner, "''", "")
			if strings.Contains(unescaped, "'") {
				t.Errorf("unescaped single quote found in result: %q (inner: %q)", got, inner)
			}

			for _, banned := range tt.shouldNotContain {
				// Check the raw output doesn't contain the unescaped injection
				// The check is: after removing escaped quotes, the injection pattern shouldn't work
				if strings.Contains(unescaped, banned) {
					t.Errorf("injection pattern %q found in unescaped result", banned)
				}
			}
		})
	}
}

// TestFormatBackfillValue_Types tests all supported value types.
func TestFormatBackfillValue_Types(t *testing.T) {
	pgRunner := newTestRunner(dialect.Postgres())
	sqliteRunner := newTestRunner(dialect.SQLite())

	t.Run("bool_postgres", func(t *testing.T) {
		if got := pgRunner.formatBackfillValue(true); got != "TRUE" {
			t.Errorf("got %q, want TRUE", got)
		}
		if got := pgRunner.formatBackfillValue(false); got != "FALSE" {
			t.Errorf("got %q, want FALSE", got)
		}
	})

	t.Run("bool_sqlite", func(t *testing.T) {
		if got := sqliteRunner.formatBackfillValue(true); got != "1" {
			t.Errorf("got %q, want 1", got)
		}
		if got := sqliteRunner.formatBackfillValue(false); got != "0" {
			t.Errorf("got %q, want 0", got)
		}
	})

	t.Run("nil", func(t *testing.T) {
		if got := pgRunner.formatBackfillValue(nil); got != "NULL" {
			t.Errorf("got %q, want NULL", got)
		}
	})

	t.Run("integer", func(t *testing.T) {
		got := pgRunner.formatBackfillValue(42)
		if got != "'42'" {
			t.Errorf("got %q, want '42'", got)
		}
	})

	t.Run("float", func(t *testing.T) {
		got := pgRunner.formatBackfillValue(3.14)
		if got != "'3.14'" {
			t.Errorf("got %q, want '3.14'", got)
		}
	})

	t.Run("sql_expr", func(t *testing.T) {
		expr := &ast.SQLExpr{Expr: "NOW()"}
		got := pgRunner.formatBackfillValue(expr)
		if got != "NOW()" {
			t.Errorf("got %q, want NOW()", got)
		}
	})
}

// TestFormatBackfillValue_SQLExpr_Passthrough tests that SQLExpr values are returned as-is.
// This is by design but means SQLExpr sources must be validated upstream.
func TestFormatBackfillValue_SQLExpr_Passthrough(t *testing.T) {
	r := newTestRunner(dialect.Postgres())

	tests := []struct {
		name string
		expr string
	}{
		{name: "now", expr: "NOW()"},
		{name: "gen_uuid", expr: "gen_random_uuid()"},
		{name: "literal_default", expr: "0"},
		{name: "complex_expr", expr: "COALESCE(old_col, 'default')"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.formatBackfillValue(&ast.SQLExpr{Expr: tt.expr})
			if got != tt.expr {
				t.Errorf("SQLExpr passthrough failed: got %q, want %q", got, tt.expr)
			}
		})
	}
}

// TestFormatBackfillValue_UnicodeAndSpecialChars tests Unicode and special character handling.
func TestFormatBackfillValue_UnicodeAndSpecialChars(t *testing.T) {
	r := newTestRunner(dialect.Postgres())

	tests := []struct {
		name  string
		input string
	}{
		{name: "emoji", input: "Hello 🌍"},
		{name: "chinese", input: "你好世界"},
		{name: "arabic", input: "مرحبا"},
		{name: "japanese", input: "こんにちは"},
		{name: "mixed_unicode", input: "café résumé naïve"},
		{name: "zero_width_space", input: "hello\u200Bworld"},
		{name: "right_to_left_override", input: "test\u202Eevil"},
		{name: "very_long_string", input: strings.Repeat("a", 10000)},
		{name: "sql_keywords_as_value", input: "SELECT DROP TABLE INSERT DELETE UPDATE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.formatBackfillValue(tt.input)

			// Must be properly quoted
			if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
				t.Errorf("result not properly quoted: %q", got)
			}

			// Inner content must not have unescaped single quotes
			inner := got[1 : len(got)-1]
			unescaped := strings.ReplaceAll(inner, "''", "")
			if strings.Contains(unescaped, "'") {
				t.Errorf("unescaped single quote in result for input %q", tt.input)
			}
		})
	}
}
