package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// List provides simple bullet-point lists.
type List struct {
	items  []ListItem
	indent int
}

// ListItem represents an item in a list.
type ListItem struct {
	Marker  string
	Content string
	Style   lipgloss.Style
}

// ListStyle determines how a list item is rendered (kept for compatibility).
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
	return &List{
		items:  []ListItem{},
		indent: 2,
	}
}

// Add adds a normal item to the list.
func (l *List) Add(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "•",
		Content: content,
		Style:   theme.Primary,
	})
}

// AddSuccess adds a success item to the list.
func (l *List) AddSuccess(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "✓",
		Content: content,
		Style:   theme.Success,
	})
}

// AddError adds an error item to the list.
func (l *List) AddError(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "✗",
		Content: content,
		Style:   theme.Error,
	})
}

// AddWarning adds a warning item to the list.
func (l *List) AddWarning(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "!",
		Content: content,
		Style:   theme.Warning,
	})
}

// AddInfo adds an info item to the list.
func (l *List) AddInfo(content string) {
	l.items = append(l.items, ListItem{
		Marker:  "→",
		Content: content,
		Style:   theme.Info,
	})
}

// String renders the list as a string using lipgloss.
func (l *List) String() string {
	var lines []string
	indentStr := strings.Repeat(" ", l.indent)

	for _, item := range l.items {
		line := indentStr + item.Style.Render(item.Marker+" "+item.Content)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
