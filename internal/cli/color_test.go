package cli

import (
	"strings"
	"testing"
)

func TestColorFunctionsPlainMode(t *testing.T) {
	// Save original and set to plain mode
	original := defaultCfg
	defer func() { defaultCfg = original }()
	SetDefault(&Config{Mode: ModePlain})

	tests := []struct {
		name  string
		fn    func(string) string
		input string
	}{
		{"Error", Error, "error text"},
		{"Warning", Warning, "warning text"},
		{"Note", Note, "note text"},
		{"Help", Help, "help text"},
		{"Success", Success, "success text"},
		{"Info", Info, "info text"},
		{"Code", Code, "E1001"},
		{"LineNum", LineNum, "42"},
		{"Pointer", Pointer, "^^^^"},
		{"Source", Source, "code here"},
		{"FilePath", FilePath, "file.js"},
		{"Progress", Progress, "50%"},
		{"Done", Done, "done"},
		{"Failed", Failed, "failed"},
		{"Header", Header, "COLUMN"},
		{"Dim", Dim, "muted"},
		{"Highlight", Highlight, "important"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			// In plain mode, output should equal input (no ANSI codes)
			if result != tt.input {
				t.Errorf("%s(%q) = %q, want %q (plain mode)", tt.name, tt.input, result, tt.input)
			}
		})
	}
}

func TestColorFunctionsTTYMode(t *testing.T) {
	// Save original and set to TTY mode
	original := defaultCfg
	defer func() { defaultCfg = original }()
	SetDefault(&Config{Mode: ModeTTY})

	tests := []struct {
		name  string
		fn    func(string) string
		input string
	}{
		{"Error", Error, "error text"},
		{"Warning", Warning, "warning text"},
		{"Note", Note, "note text"},
		{"Help", Help, "help text"},
		{"Success", Success, "success text"},
		{"Info", Info, "info text"},
		{"Code", Code, "E1001"},
		{"LineNum", LineNum, "42"},
		{"Pointer", Pointer, "^^^^"},
		{"Source", Source, "code here"},
		{"FilePath", FilePath, "file.js"},
		{"Progress", Progress, "50%"},
		{"Done", Done, "done"},
		{"Failed", Failed, "failed"},
		{"Header", Header, "COLUMN"},
		{"Dim", Dim, "muted"},
		{"Highlight", Highlight, "important"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			// In TTY mode, output should contain the input text
			if !strings.Contains(result, tt.input) {
				t.Errorf("%s(%q) = %q, should contain input", tt.name, tt.input, result)
			}
			// Note: lipgloss automatically detects when not in a real TTY
			// and disables colors, so in test environment the output may
			// be the same as input. This is correct behavior.
			// We just verify the input is preserved.
		})
	}
}

func TestPipe(t *testing.T) {
	// Save original
	original := defaultCfg
	defer func() { defaultCfg = original }()

	// Plain mode
	SetDefault(&Config{Mode: ModePlain})
	if got := Pipe(); got != "|" {
		t.Errorf("Pipe() in plain mode = %q, want %q", got, "|")
	}

	// TTY mode
	SetDefault(&Config{Mode: ModeTTY})
	result := Pipe()
	if !strings.Contains(result, "|") {
		t.Errorf("Pipe() in TTY mode = %q, should contain |", result)
	}
}

func TestEmptyInput(t *testing.T) {
	// Save original and set to plain mode
	original := defaultCfg
	defer func() { defaultCfg = original }()
	SetDefault(&Config{Mode: ModePlain})

	// Test with empty string
	if got := Error(""); got != "" {
		t.Errorf("Error(\"\") = %q, want \"\"", got)
	}
	if got := Warning(""); got != "" {
		t.Errorf("Warning(\"\") = %q, want \"\"", got)
	}
}

func TestSpecialCharacters(t *testing.T) {
	// Save original and set to plain mode
	original := defaultCfg
	defer func() { defaultCfg = original }()
	SetDefault(&Config{Mode: ModePlain})

	special := "special < > & \" ' chars"
	if got := Error(special); got != special {
		t.Errorf("Error with special chars = %q, want %q", got, special)
	}
}

func TestUnicodeInput(t *testing.T) {
	// Save original and set to plain mode
	original := defaultCfg
	defer func() { defaultCfg = original }()
	SetDefault(&Config{Mode: ModePlain})

	unicode := "Unicode: 日本語 émojis 🎉"
	if got := Info(unicode); got != unicode {
		t.Errorf("Info with unicode = %q, want %q", got, unicode)
	}
}
