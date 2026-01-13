package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// RenderSuccessPanel renders a success panel with a message.
func RenderSuccessPanel(title, content string) string {
	return renderPanel(title, content, "✓", Theme.Success)
}

// RenderErrorPanel renders an error panel with a message.
func RenderErrorPanel(title, content string) string {
	return renderPanel(title, content, "✗", Theme.Error)
}

// RenderWarningPanel renders a warning panel with a message.
func RenderWarningPanel(title, content string) string {
	return renderPanel(title, content, "⚠", Theme.Warning)
}

// RenderInfoPanel renders an info panel with a message.
func RenderInfoPanel(title, content string) string {
	return renderPanel(title, content, "ℹ", Theme.Info)
}

// renderPanel renders a styled panel with title and content.
func renderPanel(title, content, icon string, color tcell.Color) string {
	width := 80
	var output strings.Builder

	// Top border
	output.WriteString("╭")
	output.WriteString(strings.Repeat("─", width-2))
	output.WriteString("╮\n")

	// Title line
	titleText := fmt.Sprintf(" %s %s ", icon, title)
	padding := width - len(titleText) - 2
	output.WriteString("│")
	output.WriteString(Colorize(titleText, color))
	output.WriteString(strings.Repeat(" ", padding))
	output.WriteString("│\n")

	// Empty line
	output.WriteString("│")
	output.WriteString(strings.Repeat(" ", width-2))
	output.WriteString("│\n")

	// Content lines
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		// Wrap long lines if needed
		if len(line) > width-4 {
			wrapped := wrapText(line, width-4)
			for _, wl := range wrapped {
				output.WriteString("│ ")
				output.WriteString(wl)
				output.WriteString(strings.Repeat(" ", width-len(wl)-3))
				output.WriteString(" │\n")
			}
		} else {
			output.WriteString("│ ")
			output.WriteString(line)
			output.WriteString(strings.Repeat(" ", width-len(line)-3))
			output.WriteString(" │\n")
		}
	}

	// Bottom border
	output.WriteString("╰")
	output.WriteString(strings.Repeat("─", width-2))
	output.WriteString("╯")

	return output.String()
}

// wrapText wraps text to fit within a given width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	for len(text) > width {
		// Find last space before width (start from width-1 for safety)
		breakPoint := width
		maxIndex := width - 1
		if maxIndex >= len(text) {
			maxIndex = len(text) - 1
		}
		for i := maxIndex; i > 0; i-- {
			if text[i] == ' ' {
				breakPoint = i
				break
			}
		}
		// If no space found, break at width to avoid infinite loop
		if breakPoint > len(text) {
			breakPoint = len(text)
		}
		lines = append(lines, text[:breakPoint])
		text = strings.TrimSpace(text[breakPoint:])
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return lines
}
