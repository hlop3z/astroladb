package ui

import (
	"fmt"
	"strings"
)

// CommandView represents a complete command output with consistent structure
type CommandView struct {
	Title   string
	Summary string
	Content string
	Footer  string
	Type    ViewType
}

// ViewType defines the semantic type of view
type ViewType int

const (
	ViewSuccess ViewType = iota
	ViewWarning
	ViewError
	ViewInfo
)

// Render renders the complete view with consistent spacing
func (v *CommandView) Render() string {
	var b strings.Builder

	// Title section
	if v.Title != "" {
		b.WriteString(RenderTitle(v.Title))
		b.WriteString("\n\n")
	}

	// Summary section (typically a count or status line)
	if v.Summary != "" {
		b.WriteString("  ")
		b.WriteString(Muted(v.Summary))
		b.WriteString("\n\n")
	}

	// Main content
	b.WriteString(v.Content)

	// Footer section (typically help text or next steps)
	if v.Footer != "" {
		b.WriteString("\n\n")
		b.WriteString(v.Footer)
	}

	return b.String()
}

// NewSuccessView creates a success-styled view with a panel
func NewSuccessView(title, message string) *CommandView {
	return &CommandView{
		Title:   title,
		Content: RenderSuccessPanel("Success", message),
		Type:    ViewSuccess,
	}
}

// NewWarningView creates a warning-styled view with a panel
func NewWarningView(title, message string) *CommandView {
	return &CommandView{
		Title:   title,
		Content: RenderWarningPanel("Warning", message),
		Type:    ViewWarning,
	}
}

// NewErrorView creates an error-styled view with a panel
func NewErrorView(title, message string) *CommandView {
	return &CommandView{
		Title:   title,
		Content: RenderErrorPanel("Error", message),
		Type:    ViewError,
	}
}

// NewTableView creates a table-based view
func NewTableView(title, summary string, table *Table) *CommandView {
	return &CommandView{
		Title:   title,
		Summary: summary,
		Content: table.String(),
		Type:    ViewInfo,
	}
}

// NewListView creates a list-based view
func NewListView(title, summary string, list *List) *CommandView {
	return &CommandView{
		Title:   title,
		Summary: summary,
		Content: list.String(),
		Type:    ViewInfo,
	}
}

// NewEmptyView creates a view for "nothing to show" scenarios
func NewEmptyView(title, emptyMessage string) *CommandView {
	return &CommandView{
		Title:   title,
		Content: "  " + Dim(emptyMessage),
		Type:    ViewInfo,
	}
}

// NewComparisonView creates a before/after comparison view
func NewComparisonView(title string, before, after string) *CommandView {
	var b strings.Builder

	b.WriteString(RenderSubtitle("Before"))
	b.WriteString("\n")
	b.WriteString(Indent(before, 1))
	b.WriteString("\n\n")

	b.WriteString(RenderSubtitle("After"))
	b.WriteString("\n")
	b.WriteString(Indent(after, 1))

	return &CommandView{
		Title:   title,
		Content: b.String(),
		Type:    ViewInfo,
	}
}

// NewProgressView creates a view with progress tracking
type ProgressView struct {
	Title string
	Total int
	items []progressItem
}

type progressItem struct {
	Name     string
	Status   string // "pending", "running", "success", "error"
	Duration string
	Error    string
}

// AddItem adds an item to the progress view
func (pv *ProgressView) AddItem(name, status, duration, errMsg string) {
	pv.items = append(pv.items, progressItem{
		Name:     name,
		Status:   status,
		Duration: duration,
		Error:    errMsg,
	})
}

// Render renders the progress view
func (pv *ProgressView) Render() string {
	var b strings.Builder

	if pv.Title != "" {
		b.WriteString(RenderTitle(pv.Title))
		b.WriteString("\n\n")
	}

	for i, item := range pv.items {
		counter := fmt.Sprintf("[%d/%d]", i+1, pv.Total)

		var statusIcon, statusText string
		switch item.Status {
		case "success":
			statusIcon = Success("✓")
			statusText = Dim(item.Duration)
		case "error":
			statusIcon = Error("✗")
			statusText = Error(item.Error)
		case "running":
			statusIcon = Info("→")
			statusText = Dim("processing...")
		case "pending":
			statusIcon = Dim("○")
			statusText = Dim("pending")
		}

		b.WriteString(fmt.Sprintf("  %s %s ... %s %s\n",
			Dim(counter),
			Primary(item.Name),
			statusIcon,
			statusText,
		))
	}

	return b.String()
}

// SQLPreviewView creates a view showing SQL statements
type SQLPreviewView struct {
	Title      string
	Statements []string
	Warning    string
}

// Render renders the SQL preview
func (v *SQLPreviewView) Render() string {
	var b strings.Builder

	if v.Title != "" {
		b.WriteString(RenderSubtitle(v.Title))
		b.WriteString("\n\n")
	}

	if v.Warning != "" {
		b.WriteString("  ")
		b.WriteString(Warning("⚠ " + v.Warning))
		b.WriteString("\n\n")
	}

	// Box with SQL statements
	for _, stmt := range v.Statements {
		b.WriteString("  ")
		b.WriteString(Dim(stmt))
		b.WriteString("\n")
	}

	return b.String()
}

// ContextView displays key-value context information
type ContextView struct {
	Pairs map[string]string
}

// Render renders the context view
func (v *ContextView) Render() string {
	if len(v.Pairs) == 0 {
		return ""
	}

	// Find max key length for alignment
	maxLen := 0
	for key := range v.Pairs {
		if len(key) > maxLen {
			maxLen = len(key)
		}
	}

	var b strings.Builder
	for key, value := range v.Pairs {
		padding := strings.Repeat(" ", maxLen-len(key))
		b.WriteString(fmt.Sprintf("  %s:%s %s\n",
			Dim(key),
			padding,
			Primary(value),
		))
	}

	return b.String()
}
