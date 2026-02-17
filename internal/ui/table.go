package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Table wraps tview.Table with simplified API for static table output.
type Table struct {
	table   *tview.Table
	headers []string
	rows    [][]string
}

// newTable creates a new table with the given headers.
func newTable(headers ...string) *Table {
	table := tview.NewTable().
		SetBorders(false).
		SetSeparator(tview.Borders.Vertical)

	return &Table{
		table:   table,
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

// Render builds and returns the configured tview table for display.
// For static output, call String() instead.
func (t *Table) Render() *tview.Table {
	// Add header row
	for c, header := range t.headers {
		cell := tview.NewTableCell(header).
			SetTextColor(Theme.Header).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		t.table.SetCell(0, c, cell)
	}

	// Add data rows
	for r, row := range t.rows {
		for c, cellText := range row {
			color := Theme.Text
			attrs := tcell.AttrNone

			// First column is bold
			if c == 0 {
				attrs = tcell.AttrBold
			}

			cell := tview.NewTableCell(cellText).
				SetTextColor(color).
				SetAttributes(attrs).
				SetSelectable(false).
				SetAlign(tview.AlignLeft)

			// Row index is offset by 1 because of header
			t.table.SetCell(r+1, c, cell)
		}
	}

	// Fix the header row
	t.table.SetFixed(1, 0)

	return t.table
}

// String renders the table to a string for non-interactive output.
// This is useful for piping output or simple display.
func (t *Table) String() string {
	if len(t.headers) == 0 {
		return ""
	}

	// Calculate column widths
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

	// Add padding
	for i := range widths {
		widths[i] += 4
	}

	var output string

	// Separator line
	sepLine := ""
	for _, w := range widths {
		sepLine += "─"
		for i := 0; i < w; i++ {
			sepLine += "─"
		}
	}
	sepLine += "─\n"

	output += sepLine

	// Header row
	output += " "
	for i, h := range t.headers {
		output += padRight(h, widths[i])
	}
	output += "\n"
	output += sepLine

	// Data rows
	for _, row := range t.rows {
		output += " "
		for i, cell := range row {
			if i < len(widths) {
				output += padRight(cell, widths[i])
			}
		}
		output += "\n"
	}

	return output
}

// NewStyledTable creates a new styled table with the given headers.
func NewStyledTable(headers ...string) *Table {
	return newTable(headers...)
}
