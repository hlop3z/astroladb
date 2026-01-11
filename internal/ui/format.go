package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Box renders content in a bordered box with a title.
func Box(title, content string) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border.GetForeground()).
		Padding(0, 1)

	titleRendered := theme.Header.Render(title)
	return boxStyle.Render(titleRendered + "\n\n" + content)
}

// Section renders a section with a header and separator.
func Section(title, content string) string {
	header := theme.Header.Render(title)
	separator := theme.Dim.Render(strings.Repeat("─", len(title)))
	return lipgloss.JoinVertical(lipgloss.Left, header, separator, content)
}

// RenderTitle renders a large title with separator.
func RenderTitle(title string) string {
	return theme.Header.Render(title) + "\n" + theme.Dim.Render(strings.Repeat("─", len(title)))
}

// FormatKeyValue formats a key-value pair.
func FormatKeyValue(key, value string) string {
	return theme.Dim.Render(key+": ") + value
}

// FormatCount formats a count with singular/plural form.
func FormatCount(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// Indent indents all lines in content by the given number of spaces.
func Indent(content string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// padRight pads a string to the right with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
