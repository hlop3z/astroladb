package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Panel styles using lipgloss
var (
	// Base panel style with rounded border
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Width(78)

	// Title styles
	successTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")). // Green
				Bold(true)

	errorTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Bold(true)

	warningTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")). // Yellow
				Bold(true)

	infoTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Blue
			Bold(true)
)

// RenderSuccessPanel renders a success panel with a message.
func RenderSuccessPanel(title, content string) string {
	titleLine := successTitleStyle.Render("✓ " + title)
	style := panelStyle.BorderForeground(lipgloss.Color("10"))
	return style.Render(titleLine + "\n\n" + content)
}

// RenderErrorPanel renders an error panel with a message.
func RenderErrorPanel(title, content string) string {
	titleLine := errorTitleStyle.Render("✗ " + title)
	style := panelStyle.BorderForeground(lipgloss.Color("9"))
	return style.Render(titleLine + "\n\n" + content)
}

// RenderWarningPanel renders a warning panel with a message.
func RenderWarningPanel(title, content string) string {
	titleLine := warningTitleStyle.Render("⚠ " + title)
	style := panelStyle.BorderForeground(lipgloss.Color("11"))
	return style.Render(titleLine + "\n\n" + content)
}

// RenderInfoPanel renders an info panel with a message.
func RenderInfoPanel(title, content string) string {
	titleLine := infoTitleStyle.Render("ℹ " + title)
	style := panelStyle.BorderForeground(lipgloss.Color("12"))
	return style.Render(titleLine + "\n\n" + content)
}
