package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// ===========================================================================
// Color Tests
// ===========================================================================

func TestColorize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		color    tcell.Color
		contains string
	}{
		{"red text", "error", tcell.ColorRed, "\033[31m"},
		{"green text", "success", tcell.ColorGreen, "\033[32m"},
		{"blue text", "info", tcell.ColorBlue, "\033[34m"},
		{"yellow text", "warning", tcell.ColorYellow, "\033[33m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Colorize(tt.text, tt.color)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Colorize(%q, %v) should contain %q", tt.text, tt.color, tt.contains)
			}
			if !strings.Contains(result, tt.text) {
				t.Errorf("Colorize should contain original text %q", tt.text)
			}
			if !strings.HasSuffix(result, "\033[0m") {
				t.Error("Colorize should end with reset sequence")
			}
		})
	}
}

func TestColorize_UnknownColor(t *testing.T) {
	// Use a color not in the basic 16-color map
	result := Colorize("test", tcell.ColorTeal)
	if result != "test" {
		t.Errorf("Colorize with unmapped color should return original text, got %q", result)
	}
}

// ===========================================================================
// Format Tests
// ===========================================================================

func TestSuccess(t *testing.T) {
	result := Success("test message")
	if !strings.Contains(result, "test message") {
		t.Error("Success should contain the message")
	}
}

func TestError(t *testing.T) {
	result := Error("test error")
	if !strings.Contains(result, "test error") {
		t.Error("Error should contain the message")
	}
}

func TestWarning(t *testing.T) {
	result := Warning("test warning")
	if !strings.Contains(result, "test warning") {
		t.Error("Warning should contain the message")
	}
}

func TestInfo(t *testing.T) {
	result := Info("test info")
	if !strings.Contains(result, "test info") {
		t.Error("Info should contain the message")
	}
}

func TestDim(t *testing.T) {
	result := Dim("dim text")
	if !strings.Contains(result, "dim text") {
		t.Error("Dim should contain the text")
	}
}

func TestBold(t *testing.T) {
	result := Bold("bold text")
	if !strings.Contains(result, "bold text") {
		t.Error("Bold should contain the text")
	}
	if !strings.Contains(result, "\033[1m") {
		t.Error("Bold should contain bold ANSI sequence")
	}
}

// ===========================================================================
// WrapText Tests
// ===========================================================================

func TestWrapText_ShortText(t *testing.T) {
	result := wrapText("short", 40)
	if len(result) != 1 {
		t.Errorf("Short text should not wrap, got %d lines", len(result))
	}
	if result[0] != "short" {
		t.Errorf("wrapText should return original text, got %q", result[0])
	}
}

func TestWrapText_LongText(t *testing.T) {
	text := "This is a very long line that should be wrapped into multiple lines"
	result := wrapText(text, 20)
	if len(result) < 2 {
		t.Errorf("Long text should wrap, got %d lines", len(result))
	}
	for i, line := range result {
		if len(line) > 20 {
			t.Errorf("Line %d exceeds width: %q (len=%d)", i, line, len(line))
		}
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	result := wrapText("test", 0)
	if len(result) != 1 || result[0] != "test" {
		t.Error("Zero width should return original text")
	}
}

func TestWrapText_NegativeWidth(t *testing.T) {
	result := wrapText("test", -5)
	if len(result) != 1 || result[0] != "test" {
		t.Error("Negative width should return original text")
	}
}

func TestWrapText_NoSpaces(t *testing.T) {
	text := "verylongwordwithoutanyspaces"
	result := wrapText(text, 10)
	if len(result) == 0 {
		t.Error("wrapText should return at least one line")
	}
}

// ===========================================================================
// Component Tests
// ===========================================================================

func TestBadge(t *testing.T) {
	result := Badge("success", BadgeSuccess)
	if !strings.Contains(result, "success") {
		t.Errorf("Badge should contain the text")
	}
}

func TestHeader(t *testing.T) {
	result := Header("Test Header")
	if !strings.Contains(result, "Test Header") {
		t.Error("Header should contain the title")
	}
}

// ===========================================================================
// Panel Tests
// ===========================================================================

func TestRenderSuccessPanel(t *testing.T) {
	result := RenderSuccessPanel("Success", "Operation completed")
	if !strings.Contains(result, "Success") {
		t.Error("Panel should contain title")
	}
	if !strings.Contains(result, "Operation completed") {
		t.Error("Panel should contain content")
	}
}

func TestRenderErrorPanel(t *testing.T) {
	result := RenderErrorPanel("Error", "Something went wrong")
	if !strings.Contains(result, "Error") {
		t.Error("Panel should contain title")
	}
}

func TestRenderInfoPanel(t *testing.T) {
	result := RenderInfoPanel("Info", "Information message")
	if !strings.Contains(result, "Info") {
		t.Error("Panel should contain title")
	}
}

// ===========================================================================
// List Tests
// ===========================================================================

func TestList(t *testing.T) {
	list := NewList()
	list.Add("Item 1")
	list.Add("Item 2")
	list.Add("Item 3")

	result := list.String()
	if !strings.Contains(result, "Item 1") {
		t.Error("List should contain Item 1")
	}
	if !strings.Contains(result, "Item 2") {
		t.Error("List should contain Item 2")
	}
}

func TestList_WithStatus(t *testing.T) {
	list := NewList()
	list.AddSuccess("Success item")
	list.AddError("Error item")

	result := list.String()
	if !strings.Contains(result, "Success item") {
		t.Error("List should contain success item")
	}
	if !strings.Contains(result, "Error item") {
		t.Error("List should contain error item")
	}
}

// ===========================================================================
// Grid Tests
// ===========================================================================

func TestGrid(t *testing.T) {
	grid := NewGrid("Key", "Value", 40, 15)
	grid.AddRow("Name", "Test")
	grid.AddRow("Version", "1.0.0")

	result := grid.String()
	if !strings.Contains(result, "Name") {
		t.Error("Grid should contain Name key")
	}
	if !strings.Contains(result, "Test") {
		t.Error("Grid should contain Test value")
	}
}

// ===========================================================================
// Table Tests
// ===========================================================================

func TestTable(t *testing.T) {
	table := NewTable("Col1", "Col2", "Col3")
	table.AddRow("A", "B", "C")
	table.AddRow("D", "E", "F")

	result := table.Render()
	if result == nil {
		t.Error("Table.Render() should not return nil")
	}
}

// ===========================================================================
// Theme Tests
// ===========================================================================

func TestTheme(t *testing.T) {
	if Theme.Success == 0 {
		t.Error("Theme.Success should be defined")
	}
	if Theme.Error == 0 {
		t.Error("Theme.Error should be defined")
	}
	if Theme.Warning == 0 {
		t.Error("Theme.Warning should be defined")
	}
	if Theme.Info == 0 {
		t.Error("Theme.Info should be defined")
	}
	if Theme.Primary == 0 {
		t.Error("Theme.Primary should be defined")
	}
}

// ===========================================================================
// Context Tests
// ===========================================================================

func TestContextView(t *testing.T) {
	ctx := &ContextView{
		Pairs: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	result := ctx.String()
	if !strings.Contains(result, "key1") {
		t.Error("Context should contain key1")
	}
	if !strings.Contains(result, "value1") {
		t.Error("Context should contain value1")
	}
}
