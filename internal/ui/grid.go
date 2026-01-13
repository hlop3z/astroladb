package ui

import (
	"strings"
)

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

	var output strings.Builder

	// Top divider
	output.WriteString(Dim(strings.Repeat("─", g.totalWidth)))
	output.WriteString("\n")

	// Header row - pad first column, let second flow
	output.WriteString(" ")
	output.WriteString(padRight(Bold(g.header1), g.col1Width))
	output.WriteString(" ")
	output.WriteString(Bold(g.header2))
	output.WriteString("\n")

	// Middle divider
	output.WriteString(Dim(strings.Repeat("─", g.totalWidth)))
	output.WriteString("\n")

	// Data rows - pad first column only
	for _, row := range g.rows {
		output.WriteString(" ")
		output.WriteString(padRight(Bold(row[0]), g.col1Width))
		output.WriteString(" ")
		output.WriteString(row[1])
		output.WriteString("\n")
	}

	// Bottom divider
	output.WriteString(Dim(strings.Repeat("─", g.totalWidth)))
	output.WriteString("\n")

	return output.String()
}
