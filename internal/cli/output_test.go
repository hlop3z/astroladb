package cli

import (
	"strings"
	"testing"
)

func init() {
	// Use plain mode for deterministic test output
	SetDefault(&Config{Mode: ModePlain})
}

func TestNewTable(t *testing.T) {
	table := NewTable("NAME", "AGE", "CITY")

	if table == nil {
		t.Fatal("NewTable returned nil")
	}
	if len(table.headers) != 3 {
		t.Errorf("len(headers) = %d, want 3", len(table.headers))
	}
	if len(table.widths) != 3 {
		t.Errorf("len(widths) = %d, want 3", len(table.widths))
	}
}

func TestTable_AddRow(t *testing.T) {
	table := NewTable("NAME", "VALUE")
	table.AddRow("foo", "bar")
	table.AddRow("longer_name", "x")

	if len(table.rows) != 2 {
		t.Errorf("len(rows) = %d, want 2", len(table.rows))
	}

	// Width should be updated
	if table.widths[0] < len("longer_name") {
		t.Errorf("width[0] = %d, should be >= %d", table.widths[0], len("longer_name"))
	}
}

func TestTable_AddRow_Padding(t *testing.T) {
	table := NewTable("A", "B", "C")
	table.AddRow("1", "2") // Missing third column

	if len(table.rows) != 1 {
		t.Fatal("row should be added")
	}
	if len(table.rows[0]) != 3 {
		t.Errorf("row should be padded to 3 columns, got %d", len(table.rows[0]))
	}
	if table.rows[0][2] != "" {
		t.Errorf("padded column should be empty string, got %q", table.rows[0][2])
	}
}

func TestTable_String(t *testing.T) {
	table := NewTable("REVISION", "NAME", "STATUS")
	table.AddRow("001", "create_users", "applied")
	table.AddRow("002", "add_posts", "pending")

	result := table.String()

	// Should contain headers
	if !strings.Contains(result, "REVISION") {
		t.Errorf("result should contain header: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("result should contain header: %q", result)
	}
	if !strings.Contains(result, "STATUS") {
		t.Errorf("result should contain header: %q", result)
	}

	// Should contain data
	if !strings.Contains(result, "001") {
		t.Errorf("result should contain data: %q", result)
	}
	if !strings.Contains(result, "create_users") {
		t.Errorf("result should contain data: %q", result)
	}
	if !strings.Contains(result, "applied") {
		t.Errorf("result should contain data: %q", result)
	}

	// Should contain separator
	if !strings.Contains(result, "─") {
		t.Errorf("result should contain separator: %q", result)
	}
}

func TestTable_String_Empty(t *testing.T) {
	table := &Table{}
	result := table.String()

	if result != "" {
		t.Errorf("empty table should return empty string, got %q", result)
	}
}

func TestNewList(t *testing.T) {
	list := NewList()

	if list == nil {
		t.Fatal("NewList returned nil")
	}
	if len(list.items) != 0 {
		t.Errorf("new list should be empty, got %d items", len(list.items))
	}
}

func TestList_Add(t *testing.T) {
	list := NewList()
	list.Add("Item 1")
	list.Add("Item 2")

	if len(list.items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(list.items))
	}
	if list.items[0].Content != "Item 1" {
		t.Errorf("items[0].Content = %q, want %q", list.items[0].Content, "Item 1")
	}
	if list.items[0].Style != ListStyleNormal {
		t.Errorf("items[0].Style = %v, want ListStyleNormal", list.items[0].Style)
	}
}

func TestList_AddSuccess(t *testing.T) {
	list := NewList()
	list.AddSuccess("Success item")

	if len(list.items) != 1 {
		t.Fatal("item should be added")
	}
	if list.items[0].Style != ListStyleSuccess {
		t.Errorf("style = %v, want ListStyleSuccess", list.items[0].Style)
	}
	if list.items[0].Marker != "✓" {
		t.Errorf("marker = %q, want %q", list.items[0].Marker, "✓")
	}
}

func TestList_AddError(t *testing.T) {
	list := NewList()
	list.AddError("Error item")

	if len(list.items) != 1 {
		t.Fatal("item should be added")
	}
	if list.items[0].Style != ListStyleError {
		t.Errorf("style = %v, want ListStyleError", list.items[0].Style)
	}
	if list.items[0].Marker != "✗" {
		t.Errorf("marker = %q, want %q", list.items[0].Marker, "✗")
	}
}

func TestList_AddWarning(t *testing.T) {
	list := NewList()
	list.AddWarning("Warning item")

	if len(list.items) != 1 {
		t.Fatal("item should be added")
	}
	if list.items[0].Style != ListStyleWarning {
		t.Errorf("style = %v, want ListStyleWarning", list.items[0].Style)
	}
}

func TestList_AddInfo(t *testing.T) {
	list := NewList()
	list.AddInfo("Info item")

	if len(list.items) != 1 {
		t.Fatal("item should be added")
	}
	if list.items[0].Style != ListStyleInfo {
		t.Errorf("style = %v, want ListStyleInfo", list.items[0].Style)
	}
}

func TestList_String(t *testing.T) {
	list := NewList()
	list.Add("Normal item")
	list.AddSuccess("Success item")
	list.AddError("Error item")

	result := list.String()

	if !strings.Contains(result, "Normal item") {
		t.Errorf("result should contain normal item: %q", result)
	}
	if !strings.Contains(result, "Success item") {
		t.Errorf("result should contain success item: %q", result)
	}
	if !strings.Contains(result, "Error item") {
		t.Errorf("result should contain error item: %q", result)
	}
}

func TestList_String_Empty(t *testing.T) {
	list := NewList()
	result := list.String()

	if result != "" {
		t.Errorf("empty list should return empty string, got %q", result)
	}
}

func TestBox(t *testing.T) {
	result := Box("Warning", "This is important\nPay attention")

	// Should contain title
	if !strings.Contains(result, "Warning") {
		t.Errorf("result should contain title: %q", result)
	}
	// Should contain content
	if !strings.Contains(result, "This is important") {
		t.Errorf("result should contain content: %q", result)
	}
	if !strings.Contains(result, "Pay attention") {
		t.Errorf("result should contain content: %q", result)
	}
	// Should have box characters
	if !strings.Contains(result, "┌") {
		t.Errorf("result should contain top-left corner: %q", result)
	}
	if !strings.Contains(result, "┘") {
		t.Errorf("result should contain bottom-right corner: %q", result)
	}
}

func TestSection(t *testing.T) {
	result := Section("Title", "Content here")

	if !strings.Contains(result, "Title") {
		t.Errorf("result should contain title: %q", result)
	}
	if !strings.Contains(result, "Content here") {
		t.Errorf("result should contain content: %q", result)
	}
	// Should have underline
	if !strings.Contains(result, "─") {
		t.Errorf("result should contain underline: %q", result)
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		spaces  int
		want    string
	}{
		{
			name:    "single line",
			content: "hello",
			spaces:  2,
			want:    "  hello",
		},
		{
			name:    "multiple lines",
			content: "line1\nline2",
			spaces:  4,
			want:    "    line1\n    line2",
		},
		{
			name:    "with empty line",
			content: "line1\n\nline2",
			spaces:  2,
			want:    "  line1\n\n  line2",
		},
		{
			name:    "zero indent",
			content: "hello",
			spaces:  0,
			want:    "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Indent(tt.content, tt.spaces)
			if got != tt.want {
				t.Errorf("Indent(%q, %d) = %q, want %q", tt.content, tt.spaces, got, tt.want)
			}
		})
	}
}

func TestFormatKeyValue(t *testing.T) {
	result := FormatKeyValue("Status", "active")

	if !strings.Contains(result, "Status") {
		t.Errorf("result should contain key: %q", result)
	}
	if !strings.Contains(result, "active") {
		t.Errorf("result should contain value: %q", result)
	}
	if !strings.Contains(result, ":") {
		t.Errorf("result should contain colon: %q", result)
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		count    int
		singular string
		plural   string
		want     string
	}{
		{0, "item", "items", "0 items"},
		{1, "item", "items", "1 item"},
		{2, "item", "items", "2 items"},
		{100, "migration", "migrations", "100 migrations"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatCount(tt.count, tt.singular, tt.plural)
			if got != tt.want {
				t.Errorf("FormatCount(%d, %q, %q) = %q, want %q",
					tt.count, tt.singular, tt.plural, got, tt.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input string
		width int
		want  string
	}{
		{"hello", 10, "hello     "},
		{"hello", 5, "hello"},
		{"hello", 3, "hello"}, // No truncation
		{"", 5, "     "},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := padRight(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestTable_LongContent(t *testing.T) {
	table := NewTable("ID", "DESCRIPTION")
	table.AddRow("1", "This is a very long description that exceeds normal width")
	table.AddRow("2", "Short")

	result := table.String()

	// Should not panic and should contain all content
	if !strings.Contains(result, "very long description") {
		t.Errorf("result should contain long content: %q", result)
	}
}

func TestListStyles(t *testing.T) {
	styles := []ListStyle{
		ListStyleNormal,
		ListStyleSuccess,
		ListStyleError,
		ListStyleWarning,
		ListStyleInfo,
	}

	for _, style := range styles {
		list := NewList()
		list.items = append(list.items, ListItem{
			Marker:  "•",
			Content: "Test",
			Style:   style,
		})
		result := list.String()
		if result == "" {
			t.Errorf("style %v should render output", style)
		}
	}
}
