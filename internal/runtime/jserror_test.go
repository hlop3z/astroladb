package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
)

// ---------------------------------------------------------------------------
// ParseJSError — structured error (__throwError)
// ---------------------------------------------------------------------------

func TestParseJSError_GoCallbackPanic(t *testing.T) {
	vm := goja.New()

	// Register a Go callback that panics with a structured string (like throwStructuredError)
	vm.Set("failingFunc", func(ref string) string {
		if ref == "" {
			panic(vm.ToValue("[VAL-009] missing ref|try adding a ref"))
		}
		return "ok"
	})

	// Call from JS so Goja captures the JS call site
	_, err := vm.RunString(`
var x = 1;
var y = 2;
failingFunc("");
`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	info := ParseJSError(err)
	if info == nil {
		t.Fatal("ParseJSError returned nil")
	}

	// Message should contain the structured string
	if info.Message != "[VAL-009] missing ref|try adding a ref" {
		t.Errorf("Message = %q, want %q", info.Message, "[VAL-009] missing ref|try adding a ref")
	}

	// Line should point to the JS call site (line 4), not the Go callback
	if info.Line != 4 {
		t.Errorf("Line = %d, want 4 (JS call site)", info.Line)
	}
	if info.Column <= 0 {
		t.Errorf("expected Column > 0, got %d", info.Column)
	}
}

// ---------------------------------------------------------------------------
// ParseJSError — syntax error
// ---------------------------------------------------------------------------

func TestParseJSError_SyntaxError(t *testing.T) {
	vm := goja.New()

	// RunString wraps the syntax error in a *goja.Exception,
	// which ParseJSError can parse via parseGojaErrorMessage fallback.
	_, err := vm.RunString(`var x = {;`)
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}

	info := ParseJSError(err)
	if info == nil {
		t.Fatal("ParseJSError returned nil")
	}

	if info.Line <= 0 {
		t.Errorf("expected Line > 0 for syntax error, got %d", info.Line)
	}
	if info.Column <= 0 {
		t.Errorf("expected Column > 0 for syntax error, got %d", info.Column)
	}
}

// ---------------------------------------------------------------------------
// ParseJSError — runtime exception
// ---------------------------------------------------------------------------

func TestParseJSError_RuntimeException(t *testing.T) {
	vm := goja.New()

	_, err := vm.RunString(`
var x = 1;
var y = 2;
undefined_function();
`)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}

	info := ParseJSError(err)
	if info == nil {
		t.Fatal("ParseJSError returned nil")
	}

	if info.Line <= 0 {
		t.Errorf("expected Line > 0 for runtime exception, got %d", info.Line)
	}
}

// ---------------------------------------------------------------------------
// ParseJSError — nil
// ---------------------------------------------------------------------------

func TestParseJSError_Nil(t *testing.T) {
	info := ParseJSError(nil)
	if info != nil {
		t.Errorf("expected nil for nil error, got %+v", info)
	}
}

// ---------------------------------------------------------------------------
// GetSourceLine — 1-indexed
// ---------------------------------------------------------------------------

func TestGetSourceLine_OneIndexed(t *testing.T) {
	code := "line one\nline two\nline three"

	tests := []struct {
		lineNum int
		want    string
	}{
		{1, "line one"},
		{2, "line two"},
		{3, "line three"},
	}
	for _, tt := range tests {
		got := GetSourceLine(code, tt.lineNum)
		if got != tt.want {
			t.Errorf("GetSourceLine(code, %d) = %q, want %q", tt.lineNum, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GetSourceLine — edge cases
// ---------------------------------------------------------------------------

func TestGetSourceLine_EdgeCases(t *testing.T) {
	code := "only line"

	// Line 0 — invalid
	if got := GetSourceLine(code, 0); got != "" {
		t.Errorf("GetSourceLine(code, 0) = %q, want empty", got)
	}

	// Negative line
	if got := GetSourceLine(code, -1); got != "" {
		t.Errorf("GetSourceLine(code, -1) = %q, want empty", got)
	}

	// Beyond end
	if got := GetSourceLine(code, 999); got != "" {
		t.Errorf("GetSourceLine(code, 999) = %q, want empty", got)
	}

	// Empty string
	if got := GetSourceLine("", 1); got != "" {
		t.Errorf("GetSourceLine(\"\", 1) = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// GetSourceLineFromFile — 1-indexed
// ---------------------------------------------------------------------------

func TestGetSourceLineFromFile_OneIndexed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.js")

	content := "first line\nsecond line\nthird line\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	tests := []struct {
		lineNum int
		want    string
	}{
		{1, "first line"},
		{2, "second line"},
		{3, "third line"},
	}
	for _, tt := range tests {
		got := GetSourceLineFromFile(path, tt.lineNum)
		if got != tt.want {
			t.Errorf("GetSourceLineFromFile(%q, %d) = %q, want %q", path, tt.lineNum, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GetSourceLineFromFile — edge cases
// ---------------------------------------------------------------------------

func TestGetSourceLineFromFile_EdgeCases(t *testing.T) {
	// Non-existent file
	if got := GetSourceLineFromFile("/does/not/exist.js", 1); got != "" {
		t.Errorf("expected empty for non-existent file, got %q", got)
	}

	// Empty path
	if got := GetSourceLineFromFile("", 1); got != "" {
		t.Errorf("expected empty for empty path, got %q", got)
	}

	// Line 0
	dir := t.TempDir()
	path := filepath.Join(dir, "test.js")
	_ = os.WriteFile(path, []byte("hello\n"), 0644)
	if got := GetSourceLineFromFile(path, 0); got != "" {
		t.Errorf("expected empty for line 0, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// extractStructuredError
// ---------------------------------------------------------------------------

func TestExtractStructuredError(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		wantCode string
		wantMsg  string
		wantHelp string
	}{
		{
			name:     "alerr format with Use help",
			msg:      "[VAL-009] belongs_to() requires a reference\n       Use belongs_to('namespace.table')",
			wantCode: "VAL-009",
			wantMsg:  "belongs_to() requires a reference",
			wantHelp: "Use belongs_to('namespace.table')",
		},
		{
			name:     "pipe format with code",
			msg:      "[VAL-007] string() requires a length|try `col.string(255)`",
			wantCode: "VAL-007",
			wantMsg:  "string() requires a length",
			wantHelp: "try `col.string(255)`",
		},
		{
			name:     "code without help",
			msg:      "[VAL-007] string() requires a length argument",
			wantCode: "VAL-007",
			wantMsg:  "string() requires a length argument",
			wantHelp: "",
		},
		{
			name:     "no code",
			msg:      "plain error message",
			wantCode: "",
			wantMsg:  "plain error message",
			wantHelp: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg, help := extractStructuredError(tt.msg)
			if code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
			if msg != tt.wantMsg {
				t.Errorf("msg = %q, want %q", msg, tt.wantMsg)
			}
			if help != tt.wantHelp {
				t.Errorf("help = %q, want %q", help, tt.wantHelp)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// computeSpanEnd
// ---------------------------------------------------------------------------

func TestComputeSpanEnd(t *testing.T) {
	tests := []struct {
		name   string
		source string
		col    int
		want   int
	}{
		// Case 1: Opening delimiter — find matching close
		{
			name:   "empty parens",
			source: "foo()",
			col:    4,
			want:   5, // covers "()"
		},
		{
			name:   "parens with string arg",
			source: `foo("hello")`,
			col:    4,
			want:   12, // covers `("hello")`
		},
		{
			name:   "nested parens inside string",
			source: `sql({ postgres: "NOW()" })`,
			col:    4,
			want:   26, // covers entire `({ postgres: "NOW()" })`
		},
		{
			name:   "nested braces and parens",
			source: `foo({ bar: baz() })`,
			col:    4,
			want:   19, // covers `({ bar: baz() })`
		},
		{
			name:   "square brackets",
			source: `arr[0]`,
			col:    4,
			want:   6, // covers `[0]`
		},
		{
			name:   "curly braces",
			source: `obj = { a: 1 }`,
			col:    7,
			want:   14, // covers `{ a: 1 }`
		},

		// Case 2: Non-delimiter — scan to boundary
		{
			name:   "identifier to comma",
			source: `col.belongs_to(), other`,
			col:    1,
			want:   16, // covers `col.belongs_to()`
		},
		{
			name:   "identifier to semicolon",
			source: `col.belongs_to();`,
			col:    1,
			want:   16, // covers `col.belongs_to()`
		},
		{
			name:   "identifier to space",
			source: `foo bar`,
			col:    1,
			want:   3, // covers `foo`
		},
		{
			name:   "identifier with string arg to comma",
			source: `foo("hello world"), bar`,
			col:    1,
			want:   18, // covers `foo("hello world")`
		},
		{
			name:   "stops before unmatched close",
			source: `  belongs_to()}`,
			col:    3,
			want:   14, // covers `belongs_to()`
		},

		// Edge cases
		{
			name:   "column at end of line",
			source: "abc",
			col:    3,
			want:   3, // single char `c`
		},
		{
			name:   "column past end of line",
			source: "abc",
			col:    10,
			want:   10, // fallback
		},
		{
			name:   "column zero",
			source: "abc",
			col:    0,
			want:   0, // fallback
		},
		{
			name:   "unmatched open paren falls to token scan",
			source: `foo(bar`,
			col:    4,
			want:   7, // no matching close, scans to end
		},
		{
			name:   "escaped quote in string",
			source: `foo("he\"llo")`,
			col:    4,
			want:   14, // covers `("he\"llo")`
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSpanEnd(tt.source, tt.col)
			if got != tt.want {
				t.Errorf("computeSpanEnd(%q, %d) = %d, want %d", tt.source, tt.col, got, tt.want)
			}
		})
	}
}
