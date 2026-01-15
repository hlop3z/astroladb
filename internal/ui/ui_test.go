package ui

import (
	"strings"
	"testing"
)

// ===========================================================================
// Color Tests
// ===========================================================================

func TestSuccess(t *testing.T) {
	result := Success("test message")
	if !strings.Contains(result, "test message") {
		t.Error("Success should contain the message")
	}
	if !strings.Contains(result, "✓") {
		t.Error("Success should contain checkmark icon")
	}
}

func TestError(t *testing.T) {
	result := Error("test error")
	if !strings.Contains(result, "test error") {
		t.Error("Error should contain the message")
	}
	if !strings.Contains(result, "✗") {
		t.Error("Error should contain X icon")
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
}

func TestPrimary(t *testing.T) {
	result := Primary("primary text")
	if !strings.Contains(result, "primary text") {
		t.Error("Primary should contain the text")
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
	if !strings.Contains(result, "╭") {
		t.Error("Panel should have rounded border")
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

func TestRenderWarningPanel(t *testing.T) {
	result := RenderWarningPanel("Warning", "Warning message")
	if !strings.Contains(result, "Warning") {
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
