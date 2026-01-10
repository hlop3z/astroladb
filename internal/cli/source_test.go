package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	// Use plain mode for deterministic test output
	SetDefault(&Config{Mode: ModePlain})
}

func TestNewSourceSnippetFromString(t *testing.T) {
	snippet := NewSourceSnippetFromString("test.js", 10, "const x = 42;")

	if snippet.File != "test.js" {
		t.Errorf("File = %q, want %q", snippet.File, "test.js")
	}
	if snippet.StartLine != 10 {
		t.Errorf("StartLine = %d, want %d", snippet.StartLine, 10)
	}
	if len(snippet.Lines) != 1 {
		t.Errorf("len(Lines) = %d, want 1", len(snippet.Lines))
	}
	if snippet.Lines[0] != "const x = 42;" {
		t.Errorf("Lines[0] = %q, want %q", snippet.Lines[0], "const x = 42;")
	}
}

func TestSourceSnippet_AddHighlight(t *testing.T) {
	snippet := NewSourceSnippetFromString("test.js", 10, "const x = 42;")
	snippet.AddHighlight(10, 7, 8, "variable name", StyleError)

	if len(snippet.SourceHighlights) != 1 {
		t.Fatalf("len(Highlights) = %d, want 1", len(snippet.SourceHighlights))
	}

	h := snippet.SourceHighlights[0]
	if h.Line != 10 {
		t.Errorf("Highlight.Line = %d, want 10", h.Line)
	}
	if h.Start != 7 {
		t.Errorf("Highlight.Start = %d, want 7", h.Start)
	}
	if h.End != 8 {
		t.Errorf("Highlight.End = %d, want 8", h.End)
	}
	if h.Label != "variable name" {
		t.Errorf("Highlight.Label = %q, want %q", h.Label, "variable name")
	}
	if h.Style != StyleError {
		t.Errorf("Highlight.Style = %v, want StyleError", h.Style)
	}
}

func TestSourceSnippet_Render(t *testing.T) {
	snippet := NewSourceSnippetFromString("test.js", 15, "  userName: col.string(50),")
	snippet.AddHighlight(15, 3, 11, "should be user_name", StyleError)

	result := snippet.Render()

	// Should contain line number
	if !strings.Contains(result, "15") {
		t.Errorf("result should contain line number: %q", result)
	}
	// Should contain pipe
	if !strings.Contains(result, "|") {
		t.Errorf("result should contain pipe: %q", result)
	}
	// Should contain source code
	if !strings.Contains(result, "userName: col.string(50)") {
		t.Errorf("result should contain source code: %q", result)
	}
	// Should contain pointer
	if !strings.Contains(result, "^") {
		t.Errorf("result should contain pointer: %q", result)
	}
	// Should contain label
	if !strings.Contains(result, "should be user_name") {
		t.Errorf("result should contain label: %q", result)
	}
}

func TestSourceSnippet_RenderMultipleLines(t *testing.T) {
	snippet := &SourceSnippet{
		File:      "test.js",
		StartLine: 10,
		Lines: []string{
			"function foo() {",
			"  return 42;",
			"}",
		},
	}

	result := snippet.Render()

	// Should contain all line numbers
	if !strings.Contains(result, "10") {
		t.Errorf("result should contain line 10: %q", result)
	}
	if !strings.Contains(result, "11") {
		t.Errorf("result should contain line 11: %q", result)
	}
	if !strings.Contains(result, "12") {
		t.Errorf("result should contain line 12: %q", result)
	}
}

func TestSourceSnippet_RenderEmptyLines(t *testing.T) {
	snippet := &SourceSnippet{
		File:      "test.js",
		StartLine: 1,
		Lines:     []string{},
	}

	result := snippet.Render()
	if result != "" {
		t.Errorf("empty snippet should render empty string, got %q", result)
	}
}

func TestSourceSnippet_MultipleHighlights(t *testing.T) {
	snippet := &SourceSnippet{
		File:      "test.js",
		StartLine: 10,
		Lines: []string{
			"const x = 1, y = 2;",
		},
	}
	snippet.AddHighlight(10, 7, 8, "first", StyleError)
	snippet.AddHighlight(10, 14, 15, "second", StyleWarning)

	result := snippet.Render()

	// Should have multiple pointer lines
	pointerCount := strings.Count(result, "^")
	if pointerCount < 2 {
		t.Errorf("expected at least 2 pointers, got %d", pointerCount)
	}
}

func TestSourceHighlightStyles(t *testing.T) {
	tests := []struct {
		name  string
		style SourceHighlightStyle
	}{
		{"StyleError", StyleError},
		{"StyleWarning", StyleWarning},
		{"StyleNote", StyleNote},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snippet := NewSourceSnippetFromString("test.js", 1, "code")
			snippet.AddHighlight(1, 1, 4, "", tt.style)
			result := snippet.Render()
			// Should render without panic
			if result == "" {
				t.Error("render should produce output")
			}
		})
	}
}

func TestRenderFileHeader(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		line     int
		col      int
		expected []string
	}{
		{
			name:     "file only",
			file:     "test.js",
			line:     0,
			col:      0,
			expected: []string{"-->", "test.js"},
		},
		{
			name:     "file and line",
			file:     "test.js",
			line:     15,
			col:      0,
			expected: []string{"-->", "test.js:15"},
		},
		{
			name:     "file, line and column",
			file:     "test.js",
			line:     15,
			col:      5,
			expected: []string{"-->", "test.js:15:5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderFileHeader(tt.file, tt.line, tt.col)
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("result should contain %q: %q", exp, result)
				}
			}
		})
	}
}

func TestRenderSourceLine(t *testing.T) {
	result := RenderSourceLine(42, "const answer = 42;", 7, 13, "magic number")

	if !strings.Contains(result, "42") {
		t.Errorf("result should contain line number: %q", result)
	}
	if !strings.Contains(result, "const answer = 42;") {
		t.Errorf("result should contain source: %q", result)
	}
	if !strings.Contains(result, "^") {
		t.Errorf("result should contain pointer: %q", result)
	}
	if !strings.Contains(result, "magic number") {
		t.Errorf("result should contain label: %q", result)
	}
}

func TestRenderSourceLine_NoHighlight(t *testing.T) {
	result := RenderSourceLine(1, "plain code", 0, 0, "")

	if !strings.Contains(result, "plain code") {
		t.Errorf("result should contain source: %q", result)
	}
	// Should not have pointer without highlight
	// (unless default behavior adds one)
}

func TestNewSourceSnippet_FromFile(t *testing.T) {
	// Create a temp file
	dir := t.TempDir()
	file := filepath.Join(dir, "test.js")
	content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	snippet, err := NewSourceSnippet(file, 4, 1, 1)
	if err != nil {
		t.Fatalf("NewSourceSnippet error: %v", err)
	}

	// Should have 3 lines (line 3, 4, 5 with context)
	if len(snippet.Lines) != 3 {
		t.Errorf("len(Lines) = %d, want 3", len(snippet.Lines))
	}
	if snippet.StartLine != 3 {
		t.Errorf("StartLine = %d, want 3", snippet.StartLine)
	}
}

func TestNewSourceSnippet_FileNotFound(t *testing.T) {
	_, err := NewSourceSnippet("/nonexistent/file.js", 1, 0, 0)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestNewSourceSnippet_BoundaryConditions(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.js")
	content := "line 1\nline 2\nline 3"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("target at start", func(t *testing.T) {
		snippet, err := NewSourceSnippet(file, 1, 2, 0)
		if err != nil {
			t.Fatal(err)
		}
		// StartLine should be 1 (can't go below)
		if snippet.StartLine != 1 {
			t.Errorf("StartLine = %d, want 1", snippet.StartLine)
		}
	})

	t.Run("target at end", func(t *testing.T) {
		snippet, err := NewSourceSnippet(file, 3, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		// Should only have lines that exist
		if len(snippet.Lines) != 1 {
			t.Errorf("len(Lines) = %d, want 1", len(snippet.Lines))
		}
	})
}
