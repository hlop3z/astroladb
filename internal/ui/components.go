package ui

import (
	"fmt"
	"strings"
)

// RenderTitle renders a title with a separator line.
func RenderTitle(title string) string {
	return Header(title) + "\n" + Dim("────────────────")
}

// RenderSubtitle renders a subtitle with visual separation.
func RenderSubtitle(text string) string {
	return fmt.Sprintf("%s\n%s", Bold(text), Dim(strings.Repeat("─", len(text))))
}

// Section renders a section with a header and content.
func Section(header, content string) string {
	var output strings.Builder
	output.WriteString(Bold(header))
	output.WriteString("\n")
	output.WriteString(Dim(strings.Repeat("─", len(header))))
	output.WriteString("\n")
	output.WriteString(content)
	return output.String()
}

// Badge styles for colored badges.
type BadgeStyle int

const (
	BadgeSuccess BadgeStyle = iota
	BadgeWarning
	BadgeError
	BadgeInfo
	BadgePending
	BadgeApplied
)

// Badge renders a styled badge with background color.
func Badge(text string, style BadgeStyle) string {
	var bg, fg int

	switch style {
	case BadgeSuccess, BadgeApplied:
		bg, fg = 42, 30 // Green background, black text
	case BadgeWarning, BadgePending:
		bg, fg = 43, 30 // Yellow background, black text
	case BadgeError:
		bg, fg = 41, 37 // Red background, white text
	case BadgeInfo:
		bg, fg = 46, 30 // Cyan background, black text
	default:
		bg, fg = 47, 30 // White background, black text
	}

	return fmt.Sprintf("\033[%d;%dm %s \033[0m", bg, fg, text)
}

// Badge convenience functions
func RenderAppliedBadge() string { return Badge("Applied", BadgeApplied) }
func RenderPendingBadge() string { return Badge("Pending", BadgePending) }
func RenderErrorBadge() string   { return Badge("Error", BadgeError) }

// List represents a list of items with status indicators.
type List struct {
	items []string
}

// NewList creates a new list.
func NewList() *List {
	return &List{items: []string{}}
}

// AddSuccess adds a success item to the list.
func (l *List) AddSuccess(item string) {
	l.items = append(l.items, Success(item))
}

// AddError adds an error item to the list.
func (l *List) AddError(item string) {
	l.items = append(l.items, Error(item))
}

// AddWarning adds a warning item to the list.
func (l *List) AddWarning(item string) {
	l.items = append(l.items, Warning(item))
}

// AddInfo adds an info item to the list.
func (l *List) AddInfo(item string) {
	l.items = append(l.items, Info(item))
}

// Add adds a plain item to the list.
func (l *List) Add(item string) {
	l.items = append(l.items, "  • "+item)
}

// String returns the list as a formatted string.
func (l *List) String() string {
	return strings.Join(l.items, "\n")
}
