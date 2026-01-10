package cli

import (
	"fmt"
	"strings"
)

// Table provides formatted table output.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{
		headers: headers,
		widths:  widths,
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cells ...string) {
	// Pad with empty strings if needed
	for len(cells) < len(t.headers) {
		cells = append(cells, "")
	}
	// Update column widths
	for i, cell := range cells {
		if i < len(t.widths) && len(cell) > t.widths[i] {
			t.widths[i] = len(cell)
		}
	}
	t.rows = append(t.rows, cells)
}

// String renders the table as a string.
func (t *Table) String() string {
	if len(t.headers) == 0 {
		return ""
	}

	var b strings.Builder

	// Header row
	for i, h := range t.headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(Header(padRight(h, t.widths[i])))
	}
	b.WriteString("\n")

	// Separator
	for i, w := range t.widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(Dim(strings.Repeat("─", w)))
	}
	b.WriteString("\n")

	// Data rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(t.widths) {
				break
			}
			if i > 0 {
				b.WriteString("  ")
			}
			b.WriteString(padRight(cell, t.widths[i]))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// padRight pads a string to the right with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// List provides formatted list output.
type List struct {
	items  []ListItem
	indent int
}

// ListItem represents an item in a list.
type ListItem struct {
	Marker  string
	Content string
	Style   ListStyle
}

// ListStyle determines how a list item is rendered.
type ListStyle int

const (
	ListStyleNormal ListStyle = iota
	ListStyleSuccess
	ListStyleError
	ListStyleWarning
	ListStyleInfo
)

// NewList creates a new list.
func NewList() *List {
	return &List{indent: 2}
}

// Add adds an item to the list.
func (l *List) Add(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "•",
		Content: content,
		Style:   ListStyleNormal,
	})
}

// AddSuccess adds a success item to the list.
func (l *List) AddSuccess(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "✓",
		Content: content,
		Style:   ListStyleSuccess,
	})
}

// AddError adds an error item to the list.
func (l *List) AddError(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "✗",
		Content: content,
		Style:   ListStyleError,
	})
}

// AddWarning adds a warning item to the list.
func (l *List) AddWarning(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "!",
		Content: content,
		Style:   ListStyleWarning,
	})
}

// AddInfo adds an info item to the list.
func (l *List) AddInfo(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "→",
		Content: content,
		Style:   ListStyleInfo,
	})
}

// String renders the list as a string.
func (l *List) String() string {
	var b strings.Builder
	indent := strings.Repeat(" ", l.indent)

	for _, item := range l.items {
		b.WriteString(indent)

		switch item.Style {
		case ListStyleSuccess:
			b.WriteString(Success(item.Marker))
		case ListStyleError:
			b.WriteString(Failed(item.Marker))
		case ListStyleWarning:
			b.WriteString(Warning(item.Marker))
		case ListStyleInfo:
			b.WriteString(Info(item.Marker))
		default:
			b.WriteString(item.Marker)
		}

		b.WriteString(" ")
		b.WriteString(item.Content)
		b.WriteString("\n")
	}

	return b.String()
}

// Box renders content in a box (useful for important messages).
func Box(title, content string) string {
	lines := strings.Split(content, "\n")

	// Find max width
	maxWidth := len(title)
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	var b strings.Builder

	// Top border
	b.WriteString("┌─ ")
	b.WriteString(Header(title))
	b.WriteString(" ")
	b.WriteString(strings.Repeat("─", maxWidth-len(title)+1))
	b.WriteString("┐\n")

	// Content
	for _, line := range lines {
		b.WriteString("│ ")
		b.WriteString(padRight(line, maxWidth+2))
		b.WriteString(" │\n")
	}

	// Bottom border
	b.WriteString("└")
	b.WriteString(strings.Repeat("─", maxWidth+4))
	b.WriteString("┘\n")

	return b.String()
}

// Section renders a section with a header and content.
func Section(title string, content string) string {
	var b strings.Builder
	b.WriteString(Header(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", len(title)))
	b.WriteString("\n")
	b.WriteString(content)
	return b.String()
}

// Indent indents all lines in content by the given amount.
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

// FormatKeyValue formats a key-value pair.
func FormatKeyValue(key, value string) string {
	return fmt.Sprintf("%s: %s", Dim(key), value)
}

// FormatCount formats a count with singular/plural form.
func FormatCount(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
