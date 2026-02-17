package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Section title style - centralized for easy configuration
var sectionTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // Green + Bold

// SectionTitle renders a section title (e.g., "Global Flags", "Setup", "Schema")
func SectionTitle(text string) string { return sectionTitleStyle.Render(text) }

// Lipgloss styles for consistent terminal output
var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	primaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // Gray
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

// Themed color functions
func Success(text string) string { return successStyle.Render("✓ " + text) }
func Error(text string) string   { return errorStyle.Render("✗ " + text) }
func Warning(text string) string { return warningStyle.Render("⚠ " + text) }
func Info(text string) string    { return infoStyle.Render("ℹ " + text) }
func Primary(text string) string { return primaryStyle.Render(text) }
func Dim(text string) string     { return dimStyle.Render(text) }

// Basic color functions
func Green(text string) string  { return successStyle.Render(text) }
func Red(text string) string    { return errorStyle.Render(text) }
func Yellow(text string) string { return warningStyle.Render(text) }
func Blue(text string) string   { return primaryStyle.Render(text) }
func Cyan(text string) string   { return infoStyle.Render(text) }

// Style functions
func Bold(text string) string { return boldStyle.Render(text) }
func Header(text string, colorFuncs ...func(string) string) string {
	formatted := Bold(text)
	if len(colorFuncs) > 0 && colorFuncs[0] != nil {
		formatted = colorFuncs[0](formatted)
	}
	return formatted
}

// Aliases for compatibility
func Muted(text string) string     { return Dim(text) }
func FilePath(path string) string  { return Primary(path) }
func FormatError(err error) string { return Error(err.Error()) }
