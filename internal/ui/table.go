package ui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Table wraps bubbles/table with a simple API matching internal/cli.Table.
type Table struct {
	model   table.Model
	headers []string
	rows    [][]string
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	// Create columns from headers
	columns := make([]table.Column, len(headers))
	for i, h := range headers {
		columns[i] = table.Column{
			Title: h,
			Width: 20, // Will be auto-calculated when rows are added
		}
	}

	// Create table model with our theme
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false), // No selection for static output
		table.WithStyles(tableStyles()),
	)

	return &Table{
		model:   t,
		headers: headers,
		rows:    [][]string{},
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cells ...string) {
	// Pad with empty strings if needed
	for len(cells) < len(t.headers) {
		cells = append(cells, "")
	}
	t.rows = append(t.rows, cells)
}

// String renders the table as a string.
func (t *Table) String() string {
	if len(t.headers) == 0 {
		return ""
	}

	// Calculate column widths based on content
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Update column widths
	columns := t.model.Columns()
	for i := range columns {
		if i < len(widths) {
			columns[i].Width = widths[i]
		}
	}
	t.model.SetColumns(columns)

	// Convert rows to bubbles format
	bubblesRows := make([]table.Row, len(t.rows))
	for i, row := range t.rows {
		bubblesRows[i] = table.Row(row)
	}
	t.model.SetRows(bubblesRows)

	return t.model.View()
}

// tableStyles returns table styling using our theme.
func tableStyles() table.Styles {
	s := table.DefaultStyles()

	// Use our theme colors
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Border.GetForeground()).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.Header.GetForeground())

	s.Selected = s.Selected.
		Foreground(theme.Primary.GetForeground()).
		Background(lipgloss.Color("0")).
		Bold(false)

	return s
}
