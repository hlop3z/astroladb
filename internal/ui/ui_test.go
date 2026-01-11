package ui

import (
	"strings"
	"testing"
)

// TestStyles verifies that all style functions work correctly.
func TestStyles(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) string
		input    string
		contains string
	}{
		{"Primary", Primary, "test", "test"},
		{"Success", Success, "test", "test"},
		{"Error", Error, "test", "test"},
		{"Warning", Warning, "test", "test"},
		{"Info", Info, "test", "test"},
		{"Dim", Dim, "test", "test"},
		{"Header", Header, "test", "test"},
		{"Done", Done, "test", "✓"},
		{"Failed", Failed, "test", "✗"},
		{"Progress", Progress, "test", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.fn(tt.input)
			if !strings.Contains(output, tt.contains) {
				t.Errorf("%s() output %q does not contain %q", tt.name, output, tt.contains)
			}
		})
	}
}

// TestTable verifies table rendering.
func TestTable(t *testing.T) {
	table := NewTable("Name", "Status", "Count")
	table.AddRow("Migration 001", "Applied", "5")
	table.AddRow("Migration 002", "Pending", "3")

	output := table.String()

	// Verify headers are present
	if !strings.Contains(output, "Name") {
		t.Error("Table output missing 'Name' header")
	}
	if !strings.Contains(output, "Status") {
		t.Error("Table output missing 'Status' header")
	}
	if !strings.Contains(output, "Count") {
		t.Error("Table output missing 'Count' header")
	}

	// Verify row data is present
	if !strings.Contains(output, "Migration 001") {
		t.Error("Table output missing 'Migration 001'")
	}
	if !strings.Contains(output, "Applied") {
		t.Error("Table output missing 'Applied'")
	}
	if !strings.Contains(output, "Pending") {
		t.Error("Table output missing 'Pending'")
	}
}

// TestEmptyTable verifies empty table handling.
func TestEmptyTable(t *testing.T) {
	table := NewTable()
	output := table.String()

	if output != "" {
		t.Errorf("Empty table should return empty string, got: %q", output)
	}
}

// TestTablePadding verifies table handles different row lengths.
func TestTablePadding(t *testing.T) {
	table := NewTable("A", "B", "C")
	table.AddRow("1", "2") // Only 2 cells, should pad with empty string

	output := table.String()
	if !strings.Contains(output, "1") {
		t.Error("Table output missing padded row data")
	}
}

// TestList verifies list rendering.
func TestList(t *testing.T) {
	list := NewList()
	list.Add("Normal item")
	list.AddSuccess("Success item")
	list.AddError("Error item")
	list.AddWarning("Warning item")
	list.AddInfo("Info item")

	output := list.String()

	// Verify markers and content
	if !strings.Contains(output, "Normal item") {
		t.Error("List missing normal item")
	}
	if !strings.Contains(output, "Success item") {
		t.Error("List missing success item")
	}
	if !strings.Contains(output, "✓") {
		t.Error("List missing success marker")
	}
	if !strings.Contains(output, "✗") {
		t.Error("List missing error marker")
	}
	if !strings.Contains(output, "!") {
		t.Error("List missing warning marker")
	}
	if !strings.Contains(output, "→") {
		t.Error("List missing info marker")
	}
}

// TestFormatKeyValue verifies key-value formatting.
func TestFormatKeyValue(t *testing.T) {
	output := FormatKeyValue("revision", "001")

	if !strings.Contains(output, "revision") {
		t.Error("FormatKeyValue missing key")
	}
	if !strings.Contains(output, "001") {
		t.Error("FormatKeyValue missing value")
	}
	if !strings.Contains(output, ":") {
		t.Error("FormatKeyValue missing colon separator")
	}
}

// TestFormatCount verifies count formatting with singular/plural.
func TestFormatCount(t *testing.T) {
	tests := []struct {
		count    int
		singular string
		plural   string
		expected string
	}{
		{1, "item", "items", "1 item"},
		{0, "item", "items", "0 items"},
		{5, "item", "items", "5 items"},
		{1, "migration", "migrations", "1 migration"},
		{3, "migration", "migrations", "3 migrations"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			output := FormatCount(tt.count, tt.singular, tt.plural)
			if output != tt.expected {
				t.Errorf("FormatCount(%d, %q, %q) = %q, want %q",
					tt.count, tt.singular, tt.plural, output, tt.expected)
			}
		})
	}
}

// TestIndent verifies indentation.
func TestIndent(t *testing.T) {
	input := "line1\nline2\nline3"
	output := Indent(input, 4)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "    ") {
			t.Errorf("Line not indented correctly: %q", line)
		}
	}
}

// TestBox verifies box rendering.
func TestBox(t *testing.T) {
	output := Box("Title", "Content goes here")

	if !strings.Contains(output, "Title") {
		t.Error("Box missing title")
	}
	if !strings.Contains(output, "Content goes here") {
		t.Error("Box missing content")
	}
}

// TestSection verifies section rendering.
func TestSection(t *testing.T) {
	output := Section("Section Title", "Section content")

	if !strings.Contains(output, "Section Title") {
		t.Error("Section missing title")
	}
	if !strings.Contains(output, "Section content") {
		t.Error("Section missing content")
	}
	if !strings.Contains(output, "─") {
		t.Error("Section missing separator")
	}
}

// TestRenderTitle verifies title rendering.
func TestRenderTitle(t *testing.T) {
	output := RenderTitle("My Title")

	if !strings.Contains(output, "My Title") {
		t.Error("RenderTitle missing title text")
	}
	if !strings.Contains(output, "─") {
		t.Error("RenderTitle missing separator")
	}
}

// TestThemeCustomization verifies theme can be customized.
func TestThemeCustomization(t *testing.T) {
	// Save original theme
	original := GetTheme()
	defer SetTheme(original)

	// Set custom theme
	customTheme := DefaultTheme()
	SetTheme(customTheme)

	// Verify we can get it back
	retrieved := GetTheme()
	if retrieved != customTheme {
		t.Error("Theme customization failed")
	}
}

// TestPadRight verifies right padding.
func TestPadRight(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		output string
	}{
		{"abc", 5, "abc  "},
		{"hello", 5, "hello"},
		{"hi", 10, "hi        "},
		{"", 3, "   "},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			output := padRight(tt.input, tt.width)
			if output != tt.output {
				t.Errorf("padRight(%q, %d) = %q, want %q",
					tt.input, tt.width, output, tt.output)
			}
			if len(output) != tt.width {
				t.Errorf("padRight(%q, %d) length = %d, want %d",
					tt.input, tt.width, len(output), tt.width)
			}
		})
	}
}
