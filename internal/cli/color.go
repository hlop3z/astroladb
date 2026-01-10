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

// Styled text functions - these check EnableColors() internally.

// Error returns text styled as an error label.
func Error(s string) string {
	if !EnableColors() {
		return s
	}
	return styleError.Render(s)
}

// Warning returns text styled as a warning label.
func Warning(s string) string {
	if !EnableColors() {
		return s
	}
	return styleWarning.Render(s)
}

// Note returns text styled as a note label.
func Note(s string) string {
	if !EnableColors() {
		return s
	}
	return styleNote.Render(s)
}

// Help returns text styled as a help label.
func Help(s string) string {
	if !EnableColors() {
		return s
	}
	return styleHelp.Render(s)
}

// Success returns text styled as a success message.
func Success(s string) string {
	if !EnableColors() {
		return s
	}
	return styleSuccess.Render(s)
}

// Info returns text styled as informational text.
func Info(s string) string {
	if !EnableColors() {
		return s
	}
	return styleInfo.Render(s)
}

// Code returns text styled as an error code.
func Code(s string) string {
	if !EnableColors() {
		return s
	}
	return styleCode.Render(s)
}

// LineNum returns text styled as a line number.
func LineNum(s string) string {
	if !EnableColors() {
		return s
	}
	return styleLineNum.Render(s)
}

// Pipe returns a pipe character styled for source display.
func Pipe() string {
	if !EnableColors() {
		return "|"
	}
	return stylePipe.Render("|")
}

// Pointer returns text styled as a pointer (^^^^).
func Pointer(s string) string {
	if !EnableColors() {
		return s
	}
	return stylePointer.Render(s)
}

// Source returns text styled as source code.
func Source(s string) string {
	if !EnableColors() {
		return s
	}
	return styleSource.Render(s)
}

// FilePath returns text styled as a file path.
func FilePath(s string) string {
	if !EnableColors() {
		return s
	}
	return styleFilePath.Render(s)
}

// Progress returns text styled for progress display.
func Progress(s string) string {
	if !EnableColors() {
		return s
	}
	return styleProgress.Render(s)
}

// Done returns text styled as "done" (success).
func Done(s string) string {
	if !EnableColors() {
		return s
	}
	return styleDone.Render(s)
}

// Failed returns text styled as "failed" (error).
func Failed(s string) string {
	if !EnableColors() {
		return s
	}
	return styleFailed.Render(s)
}

// Header returns text styled as a table header.
func Header(s string) string {
	if !EnableColors() {
		return s
	}
	return styleHeader.Render(s)
}

// Dim returns text styled as dim/muted.
func Dim(s string) string {
	if !EnableColors() {
		return s
	}
	return styleDim.Render(s)
}

// Highlight returns text styled as highlighted.
func Highlight(s string) string {
	if !EnableColors() {
		return s
	}
	return styleHighlight.Render(s)
}
