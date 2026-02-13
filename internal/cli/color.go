package cli

import "github.com/charmbracelet/lipgloss"

// Color scheme inspired by Cargo/rustc.
// Uses ANSI 256 colors for broad terminal compatibility.
var (
	// Message type styles
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	styleNote    = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	styleHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	// Error code style (e.g., E1001)
	styleCode = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)

	// Source code display styles
	styleLineNum  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	stylePipe     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	stylePointer  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleSource   = lipgloss.NewStyle()
	styleFilePath = lipgloss.NewStyle().Bold(true)

	// Progress styles
	styleProgress = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleFailed   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	// Table styles
	styleHeader    = lipgloss.NewStyle().Bold(true)
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

// applyStyle renders text with the given style, respecting color settings.
func applyStyle(s string, style lipgloss.Style) string {
	if !EnableColors() {
		return s
	}
	return style.Render(s)
}

// Styled text functions - these check EnableColors() internally.

func Error(s string) string     { return applyStyle(s, styleError) }
func Warning(s string) string   { return applyStyle(s, styleWarning) }
func Note(s string) string      { return applyStyle(s, styleNote) }
func Help(s string) string      { return applyStyle(s, styleHelp) }
func Success(s string) string   { return applyStyle(s, styleSuccess) }
func Info(s string) string      { return applyStyle(s, styleInfo) }
func Code(s string) string      { return applyStyle(s, styleCode) }
func LineNum(s string) string   { return applyStyle(s, styleLineNum) }
func Pointer(s string) string   { return applyStyle(s, stylePointer) }
func Source(s string) string    { return applyStyle(s, styleSource) }
func FilePath(s string) string  { return applyStyle(s, styleFilePath) }
func Progress(s string) string  { return applyStyle(s, styleProgress) }
func Done(s string) string      { return applyStyle(s, styleDone) }
func Failed(s string) string    { return applyStyle(s, styleFailed) }
func Header(s string) string    { return applyStyle(s, styleHeader) }
func Dim(s string) string       { return applyStyle(s, styleDim) }
func Highlight(s string) string { return applyStyle(s, styleHighlight) }

// Pipe returns a pipe character styled for source display.
func Pipe() string {
	if !EnableColors() {
		return "|"
	}
	return stylePipe.Render("|")
}
