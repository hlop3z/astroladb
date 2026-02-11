package cli

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// MessageType represents the type of diagnostic message.
type MessageType int

const (
	TypeError MessageType = iota
	TypeWarning
	TypeNote
	TypeHelp
)

// DiagnosticMessage represents a single diagnostic message with optional source context.
type DiagnosticMessage struct {
	Type    MessageType
	Code    string // Error code like "E2002" (empty for warnings/notes/help)
	Message string
	File    string
	Line    int
	Column  int
	Source  string   // The source line
	Span    [2]int   // [start, end] column indices for highlighting
	Label   string   // Label to show under the span
	Notes   []string // Additional notes
	Helps   []string // Help suggestions
}

// FormatError formats an error for CLI display in Cargo/rustc style.
// If the error is an *alerr.Error, it extracts structured information.
// Otherwise, it formats as a generic error.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	// Try to extract alerr.Error
	var alerror *alerr.Error
	if ae, ok := err.(*alerr.Error); ok {
		alerror = ae
	}

	if alerror != nil {
		return formatAlabError(alerror)
	}

	// Generic error fallback
	return formatGenericError(err)
}

// formatAlabError formats an *alerr.Error in Cargo style.
func formatAlabError(err *alerr.Error) string {
	var b strings.Builder

	code := string(err.GetCode())
	msg := err.GetMessage()
	ctx := err.GetContext()

	// First line: error[E1001]: message
	b.WriteString(Error("error"))
	b.WriteString("[")
	b.WriteString(Code(code))
	b.WriteString("]: ")
	b.WriteString(msg)
	b.WriteString("\n")

	// File location if available
	file, _ := ctx["file"].(string)
	line, _ := ctx["line"].(int)
	col, _ := ctx["column"].(int)

	if file != "" {
		b.WriteString("  ")
		b.WriteString(stylePipe.Render("-->"))
		b.WriteString(" ")
		loc := file
		if line > 0 {
			loc = fmt.Sprintf("%s:%d", file, line)
			if col > 0 {
				loc = fmt.Sprintf("%s:%d:%d", file, line, col)
			}
		}
		b.WriteString(FilePath(loc))
		b.WriteString("\n")
	}

	// Source context if available
	source, hasSource := ctx["source"].(string)
	var linePadding string
	if hasSource && line > 0 {
		b.WriteString(formatSourceContext(line, source, col, ctx))
		// Calculate padding for consistent pipe alignment
		lineStr := fmt.Sprintf("%d", line)
		linePadding = strings.Repeat(" ", len(lineStr)) + " "
	}

	// Context details (excluding already shown items)
	excludeKeys := map[string]bool{
		"file": true, "line": true, "column": true,
		"source": true, "span_start": true, "span_end": true,
		"notes": true, "helps": true, "label": true,
	}

	var details []string
	for k, v := range ctx {
		if excludeKeys[k] {
			continue
		}
		details = append(details, fmt.Sprintf("%s: %v", k, v))
	}

	if len(details) > 0 && !hasSource {
		b.WriteString("   ")
		b.WriteString(Pipe())
		b.WriteString("\n")
		for _, detail := range details {
			b.WriteString("   ")
			b.WriteString(Pipe())
			b.WriteString(" ")
			b.WriteString(detail)
			b.WriteString("\n")
		}
	}

	// Notes
	if notes, ok := ctx["notes"].([]string); ok {
		for _, note := range notes {
			b.WriteString("   ")
			b.WriteString(Pipe())
			b.WriteString("\n")
			b.WriteString(Note("note"))
			b.WriteString(": ")
			b.WriteString(note)
			b.WriteString("\n")
		}
	}

	// Helps
	if helps, ok := ctx["helps"].([]string); ok {
		for _, help := range helps {
			b.WriteString(Help("help"))
			b.WriteString(": ")
			b.WriteString(help)
			b.WriteString("\n")
		}
	}

	// Cause if present
	if cause := err.GetCause(); cause != nil {
		// Use line padding if available, otherwise default to 3 spaces
		if linePadding != "" {
			b.WriteString(linePadding)
		} else {
			b.WriteString("   ")
		}
		b.WriteString(Pipe())
		b.WriteString("\n")
		b.WriteString(Note("cause"))
		b.WriteString(": ")
		b.WriteString(cleanCauseMessage(cause.Error()))
		b.WriteString("\n")
	}

	return b.String()
}

// cleanCauseMessage removes Goja stack trace from error messages and styles help text.
// Strips " at github.com/.../func (native)" patterns for cleaner output.
// Parses structured errors in "CAUSE|HELP" format for proper styling.
func cleanCauseMessage(msg string) string {
	// Find Goja stack trace pattern " at github.com/..." and truncate there
	if idx := strings.Index(msg, " at github.com"); idx != -1 {
		msg = strings.TrimSpace(msg[:idx])
	}

	// Parse structured error format "CAUSE|HELP"
	if idx := strings.Index(msg, "|"); idx != -1 {
		cause := strings.TrimSpace(msg[:idx])
		help := strings.TrimSpace(msg[idx+1:])
		return cause + "\n " + Help("help") + ": " + help
	}

	return msg
}

// formatSourceContext renders source code with line numbers and highlighting.
func formatSourceContext(line int, source string, col int, ctx map[string]any) string {
	var b strings.Builder

	lineStr := fmt.Sprintf("%d", line)
	padding := strings.Repeat(" ", len(lineStr))

	// Empty line with pipe
	b.WriteString(padding)
	b.WriteString(" ")
	b.WriteString(Pipe())
	b.WriteString("\n")

	// Source line: "15 |   userName: col.string(50),"
	b.WriteString(LineNum(lineStr))
	b.WriteString(" ")
	b.WriteString(Pipe())
	b.WriteString(" ")
	b.WriteString(Source(source))
	b.WriteString("\n")

	// Pointer line: "   |   ^^^^^^^^ message"
	spanStart, _ := ctx["span_start"].(int)
	spanEnd, _ := ctx["span_end"].(int)
	label, _ := ctx["label"].(string)

	if spanStart > 0 || spanEnd > 0 || col > 0 {
		b.WriteString(padding)
		b.WriteString(" ")
		b.WriteString(Pipe())
		b.WriteString(" ")

		// Calculate pointer position
		start := spanStart
		if start == 0 && col > 0 {
			start = col
		}
		end := spanEnd
		if end == 0 {
			end = start + 1
		}

		// Add spaces before the pointer
		if start > 1 {
			b.WriteString(strings.Repeat(" ", start-1))
		}

		// Add the pointer characters
		pointerLen := end - start + 1
		if pointerLen < 1 {
			pointerLen = 1
		}
		b.WriteString(Pointer(strings.Repeat("^", pointerLen)))

		// Add label if present
		if label != "" {
			b.WriteString(" ")
			b.WriteString(label)
		}
		b.WriteString("\n")

		// Closing pipe line
		b.WriteString(padding)
		b.WriteString(" ")
		b.WriteString(Pipe())
		b.WriteString("\n")
	}

	return b.String()
}

// formatGenericError formats a non-alerr error.
func formatGenericError(err error) string {
	var b strings.Builder
	b.WriteString(Error("error"))
	b.WriteString(": ")
	b.WriteString(err.Error())
	b.WriteString("\n")
	return b.String()
}

// FormatWarning formats a warning message in Cargo style.
func FormatWarning(msg string, opts ...DiagnosticOption) string {
	diag := &DiagnosticMessage{
		Type:    TypeWarning,
		Message: msg,
	}
	for _, opt := range opts {
		opt(diag)
	}
	return formatDiagnostic(diag)
}

// FormatNote formats a note message.
func FormatNote(msg string) string {
	var b strings.Builder
	b.WriteString(Note("note"))
	b.WriteString(": ")
	b.WriteString(msg)
	b.WriteString("\n")
	return b.String()
}

// FormatHelp formats a help message.
func FormatHelp(msg string) string {
	var b strings.Builder
	b.WriteString(Help("help"))
	b.WriteString(": ")
	b.WriteString(msg)
	b.WriteString("\n")
	return b.String()
}

// FormatSuccess formats a success message.
func FormatSuccess(msg string) string {
	var b strings.Builder
	b.WriteString(Success("success"))
	b.WriteString(": ")
	b.WriteString(msg)
	b.WriteString("\n")
	return b.String()
}

// DiagnosticOption configures a diagnostic message.
type DiagnosticOption func(*DiagnosticMessage)

// WithFile sets the file location for a diagnostic.
func WithFile(file string, line, col int) DiagnosticOption {
	return func(d *DiagnosticMessage) {
		d.File = file
		d.Line = line
		d.Column = col
	}
}

// WithSource sets the source context for a diagnostic.
func WithSource(source string, spanStart, spanEnd int, label string) DiagnosticOption {
	return func(d *DiagnosticMessage) {
		d.Source = source
		d.Span = [2]int{spanStart, spanEnd}
		d.Label = label
	}
}

// WithNotes adds notes to a diagnostic.
func WithNotes(notes ...string) DiagnosticOption {
	return func(d *DiagnosticMessage) {
		d.Notes = append(d.Notes, notes...)
	}
}

// WithHelps adds help suggestions to a diagnostic.
func WithHelps(helps ...string) DiagnosticOption {
	return func(d *DiagnosticMessage) {
		d.Helps = append(d.Helps, helps...)
	}
}

// formatDiagnostic formats a DiagnosticMessage.
func formatDiagnostic(d *DiagnosticMessage) string {
	var b strings.Builder

	// Type label
	switch d.Type {
	case TypeError:
		b.WriteString(Error("error"))
		if d.Code != "" {
			b.WriteString("[")
			b.WriteString(Code(d.Code))
			b.WriteString("]")
		}
	case TypeWarning:
		b.WriteString(Warning("warning"))
	case TypeNote:
		b.WriteString(Note("note"))
	case TypeHelp:
		b.WriteString(Help("help"))
	}

	b.WriteString(": ")
	b.WriteString(d.Message)
	b.WriteString("\n")

	// File location
	if d.File != "" {
		b.WriteString("  ")
		b.WriteString(stylePipe.Render("-->"))
		b.WriteString(" ")
		loc := d.File
		if d.Line > 0 {
			loc = fmt.Sprintf("%s:%d", d.File, d.Line)
			if d.Column > 0 {
				loc = fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Column)
			}
		}
		b.WriteString(FilePath(loc))
		b.WriteString("\n")
	}

	// Source context
	if d.Source != "" && d.Line > 0 {
		ctx := map[string]any{
			"span_start": d.Span[0],
			"span_end":   d.Span[1],
			"label":      d.Label,
		}
		b.WriteString(formatSourceContext(d.Line, d.Source, d.Column, ctx))
	}

	// Notes
	for _, note := range d.Notes {
		b.WriteString("   ")
		b.WriteString(Pipe())
		b.WriteString("\n")
		b.WriteString(Note("note"))
		b.WriteString(": ")
		b.WriteString(note)
		b.WriteString("\n")
	}

	// Helps
	for _, help := range d.Helps {
		b.WriteString(Help("help"))
		b.WriteString(": ")
		b.WriteString(help)
		b.WriteString("\n")
	}

	return b.String()
}
