package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Table wraps bubbles/table with a simple API matching internal/cli.Table.
type Table struct {
	model       table.Model
	headers     []string
	rows        [][]string
	fixedWidths bool
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

// NewTableWithWidths creates a new table with exact total width.
// The width parameters are the desired total width and proportions.
func NewTableWithWidths(header1, header2 string, totalWidth, col1Percent int) *Table {
	// Calculate column widths as percentages of total width
	// Account for table padding/spacing (approximately 4 characters)
	usableWidth := totalWidth - 4
	col1Width := (usableWidth * col1Percent) / 100
	col2Width := usableWidth - col1Width

	// Create columns with fixed widths
	columns := []table.Column{
		{Title: header1, Width: col1Width},
		{Title: header2, Width: col2Width},
	}

	// Create table model with our theme
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false), // No selection for static output
		table.WithStyles(tableStyles()),
	)

	// Set exact table width
	t.SetWidth(totalWidth)

	return &Table{
		model:   t,
		headers: []string{header1, header2},
		rows:    [][]string{},
		fixedWidths: true, // Flag to skip width recalculation
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

	// Pre-process rows with styling to calculate accurate widths
	boldStyle := lipgloss.NewStyle().Bold(true)
	styledRows := make([][]string, len(t.rows))
	for i, row := range t.rows {
		styledRow := make([]string, len(row))
		for j, cell := range row {
			if j == 0 {
				// Make first column bold
				styledRow[j] = boldStyle.Render(cell)
			} else {
				styledRow[j] = cell
			}
		}
		styledRows[i] = styledRow
	}

	// Only recalculate widths if not using fixed widths
	if !t.fixedWidths {
		// Calculate column widths based on STYLED content
		widths := make([]int, len(t.headers))
		for i, h := range t.headers {
			widths[i] = lipgloss.Width(h)
		}
		for _, row := range styledRows {
			for i, cell := range row {
				cellWidth := lipgloss.Width(cell)
				if i < len(widths) && cellWidth > widths[i] {
					widths[i] = cellWidth
				}
			}
		}

		// Add generous padding to prevent truncation
		// The first column needs extra space because bold characters can render wider
		for i := range widths {
			if i == 0 {
				// First column (command names) - extra padding for bold text
				widths[i] += 5
			} else {
				// Description column - larger padding to prevent wrapping
				widths[i] += 5
			}
		}

		// Update column widths
		columns := t.model.Columns()
		totalWidth := 0
		for i := range columns {
			if i < len(widths) {
				columns[i].Width = widths[i]
				totalWidth += widths[i]
			}
		}
		t.model.SetColumns(columns)

		// Set table width to sum of column widths plus borders and padding
		t.model.SetWidth(totalWidth + len(columns) + 4)
	}

	// Convert styled rows to bubbles format
	bubblesRows := make([]table.Row, len(styledRows))
	for i, row := range styledRows {
		bubblesRows[i] = table.Row(row)
	}
	t.model.SetRows(bubblesRows)

	// Set height to match the number of rows
	// Note: SetHeight is the viewport height in lines.
	// We need: header (1) + header border (1) + data rows
	if len(t.rows) > 0 {
		t.model.SetHeight(len(t.rows) + 2)
	}

	return t.model.View()
}

// tableStyles returns table styling using our theme.
func tableStyles() table.Styles {
	s := table.DefaultStyles()

	// Create a custom border using the same character as the main Divider constant (─)
	// This ensures consistent divider characters throughout the CLI
	customBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        " ",
		Right:       " ",
		TopLeft:     " ",
		TopRight:    " ",
		BottomLeft:  " ",
		BottomRight: " ",
	}

	// Use our theme colors
	s.Header = s.Header.
		BorderStyle(customBorder).
		BorderForeground(theme.Border.GetForeground()).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.Header.GetForeground())

	// Remove all selection/highlight styling for static tables
	s.Selected = s.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)

	return s
}

// NewStyledTable is an alias for NewTable for backward compatibility.
func NewStyledTable(headers ...string) *Table {
	return NewTable(headers...)
}

// Grid provides a simple two-column grid layout with exact width control.
type Grid struct {
	header1    string
	header2    string
	rows       [][]string
	totalWidth int
	col1Width  int
}

// NewGrid creates a new grid with exact total width and first column width.
func NewGrid(header1, header2 string, totalWidth, col1Width int) *Grid {
	return &Grid{
		header1:    header1,
		header2:    header2,
		rows:       [][]string{},
		totalWidth: totalWidth,
		col1Width:  col1Width,
	}
}

// AddRow adds a row to the grid.
func (g *Grid) AddRow(col1, col2 string) {
	g.rows = append(g.rows, []string{col1, col2})
}

// String renders the grid as a string with top and bottom dividers.
func (g *Grid) String() string {
	if len(g.rows) == 0 {
		return ""
	}

	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Header.GetForeground())
	boldStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := theme.Dim

	var output strings.Builder

	// Top divider
	output.WriteString(dimStyle.Render(strings.Repeat("─", g.totalWidth)))
	output.WriteString("\n")

	// Header row - pad first column, let second flow
	output.WriteString(" ")
	output.WriteString(padRight(headerStyle.Render(g.header1), g.col1Width))
	output.WriteString(" ")
	output.WriteString(headerStyle.Render(g.header2))
	output.WriteString("\n")

	// Middle divider
	output.WriteString(dimStyle.Render(strings.Repeat("─", g.totalWidth)))
	output.WriteString("\n")

	// Data rows - pad first column only
	for _, row := range g.rows {
		output.WriteString(" ")
		output.WriteString(padRight(boldStyle.Render(row[0]), g.col1Width))
		output.WriteString(" ")
		output.WriteString(row[1])
		output.WriteString("\n")
	}

	// Bottom divider
	output.WriteString(dimStyle.Render(strings.Repeat("─", g.totalWidth)))
	output.WriteString("\n")

	return output.String()
}
