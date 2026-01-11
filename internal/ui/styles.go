// Package ui provides terminal UI components built on Charm Bubbles.
// This package replaces internal/cli output components with better, tested, and maintainable alternatives.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme is the single source of truth for all UI styling.
type Theme struct {
	Primary lipgloss.Style
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Dim     lipgloss.Style
	Header  lipgloss.Style
	Border  lipgloss.Style
	Done    lipgloss.Style
}

// DefaultTheme returns the default color theme.
func DefaultTheme() *Theme {
	return &Theme{
		Primary: lipgloss.NewStyle().Foreground(lipgloss.Color("12")), // Blue
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // Green
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Red
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("11")), // Yellow
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("14")), // Cyan
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),  // Gray
		Header:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		Border:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Done:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
	}
}

// Global theme instance (single source of truth).
var theme = DefaultTheme()

// SetTheme allows changing the global theme (useful for testing or future theme support).
func SetTheme(t *Theme) {
	theme = t
}

// GetTheme returns the current global theme.
func GetTheme() *Theme {
	return theme
}

// Style helper functions for consistent text formatting across the CLI.

// Primary renders text in the primary color.
func Primary(text string) string {
	return theme.Primary.Render(text)
}

// Success renders text in the success color (green).
func Success(text string) string {
	return theme.Success.Render(text)
}

// Error renders text in the error color (red).
func Error(text string) string {
	return theme.Error.Render(text)
}

// Warning renders text in the warning color (yellow).
func Warning(text string) string {
	return theme.Warning.Render(text)
}

// Info renders text in the info color (cyan).
func Info(text string) string {
	return theme.Info.Render(text)
}

// Dim renders text in a dimmed color (gray).
func Dim(text string) string {
	return theme.Dim.Render(text)
}

// Header renders text as a header (bold primary).
func Header(text string) string {
	return theme.Header.Render(text)
}

// Done renders text with a success checkmark.
func Done(text string) string {
	return theme.Done.Render("✓ " + text)
}

// Failed renders text with an error cross.
func Failed(text string) string {
	return theme.Error.Render("✗ " + text)
}

// Progress renders text in the info color (used for progress indicators).
func Progress(text string) string {
	return theme.Info.Render(text)
}

// Note renders a note prefix.
func Note(text string) string {
	return theme.Info.Render(text)
}

// Help renders a help prefix.
func Help(text string) string {
	return theme.Dim.Render(text)
}
