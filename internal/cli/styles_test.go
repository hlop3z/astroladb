package cli

import (
	"strings"
	"testing"
)

func init() {
	// Use plain mode for deterministic test output
	SetDefault(&Config{Mode: ModePlain})
}

func TestRenderBadge_PlainMode(t *testing.T) {
	result := RenderBadge("TEST", badgeApplied)
	if result != "[TEST]" {
		t.Errorf("RenderBadge in plain mode = %q, want %q", result, "[TEST]")
	}
}

func TestRenderAppliedBadge_PlainMode(t *testing.T) {
	result := RenderAppliedBadge()
	if result != "[APPLIED]" {
		t.Errorf("RenderAppliedBadge in plain mode = %q, want %q", result, "[APPLIED]")
	}
}

func TestRenderPendingBadge_PlainMode(t *testing.T) {
	result := RenderPendingBadge()
	if result != "[PENDING]" {
		t.Errorf("RenderPendingBadge in plain mode = %q, want %q", result, "[PENDING]")
	}
}

func TestRenderErrorBadge_PlainMode(t *testing.T) {
	result := RenderErrorBadge()
	if result != "[ERROR]" {
		t.Errorf("RenderErrorBadge in plain mode = %q, want %q", result, "[ERROR]")
	}
}

func TestRenderInfoBadge_PlainMode(t *testing.T) {
	result := RenderInfoBadge("INFO")
	if result != "[INFO]" {
		t.Errorf("RenderInfoBadge in plain mode = %q, want %q", result, "[INFO]")
	}
}

func TestRenderPanel_PlainMode(t *testing.T) {
	content := "Panel content"
	result := RenderPanel(content)
	if result != content {
		t.Errorf("RenderPanel in plain mode = %q, want %q", result, content)
	}
}

func TestRenderSuccessPanel_PlainMode(t *testing.T) {
	result := RenderSuccessPanel("Title", "Content")
	if !strings.Contains(result, "Title") {
		t.Errorf("RenderSuccessPanel should contain title: %q", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("RenderSuccessPanel should contain content: %q", result)
	}
	if !strings.Contains(result, "✓") {
		t.Errorf("RenderSuccessPanel should contain checkmark: %q", result)
	}
}

func TestRenderWarningPanel_PlainMode(t *testing.T) {
	result := RenderWarningPanel("Title", "Content")
	if !strings.Contains(result, "Title") {
		t.Errorf("RenderWarningPanel should contain title: %q", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("RenderWarningPanel should contain content: %q", result)
	}
	if !strings.Contains(result, "!") {
		t.Errorf("RenderWarningPanel should contain exclamation: %q", result)
	}
}

func TestRenderErrorPanel_PlainMode(t *testing.T) {
	result := RenderErrorPanel("Title", "Content")
	if !strings.Contains(result, "Title") {
		t.Errorf("RenderErrorPanel should contain title: %q", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("RenderErrorPanel should contain content: %q", result)
	}
	if !strings.Contains(result, "✗") {
		t.Errorf("RenderErrorPanel should contain X: %q", result)
	}
}

func TestRenderInfoPanel_PlainMode(t *testing.T) {
	result := RenderInfoPanel("Title", "Content")
	if !strings.Contains(result, "Title") {
		t.Errorf("RenderInfoPanel should contain title: %q", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("RenderInfoPanel should contain content: %q", result)
	}
	if !strings.Contains(result, "→") {
		t.Errorf("RenderInfoPanel should contain arrow: %q", result)
	}
}

func TestRenderTitle_PlainMode(t *testing.T) {
	result := RenderTitle("My Title")
	if !strings.Contains(result, "My Title") {
		t.Errorf("RenderTitle should contain title: %q", result)
	}
	if !strings.Contains(result, "═══") {
		t.Errorf("RenderTitle should contain decorations: %q", result)
	}
}

func TestRenderSubtitle_PlainMode(t *testing.T) {
	result := RenderSubtitle("Subtitle")
	if !strings.Contains(result, "Subtitle") {
		t.Errorf("RenderSubtitle should contain subtitle: %q", result)
	}
	if !strings.Contains(result, "──") {
		t.Errorf("RenderSubtitle should contain decorations: %q", result)
	}
}

func TestNewStyledTable(t *testing.T) {
	table := NewStyledTable("NAME", "VALUE")
	if table == nil {
		t.Fatal("NewStyledTable returned nil")
	}
	if len(table.headers) != 2 {
		t.Errorf("len(headers) = %d, want 2", len(table.headers))
	}
}

func TestStyledTable_AddRow(t *testing.T) {
	table := NewStyledTable("A", "B")
	table.AddRow("1", "2")
	table.AddRow("long value", "x")

	if len(table.rows) != 2 {
		t.Errorf("len(rows) = %d, want 2", len(table.rows))
	}
}

func TestStyledTable_String_PlainMode(t *testing.T) {
	table := NewStyledTable("REVISION", "NAME", "STATUS")
	table.AddRow("001", "create_users", "[APPLIED]")
	table.AddRow("002", "add_posts", "[PENDING]")

	result := table.String()

	// Should contain headers
	if !strings.Contains(result, "REVISION") {
		t.Errorf("result should contain header REVISION: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("result should contain header NAME: %q", result)
	}

	// Should contain data
	if !strings.Contains(result, "001") {
		t.Errorf("result should contain data: %q", result)
	}
	if !strings.Contains(result, "create_users") {
		t.Errorf("result should contain data: %q", result)
	}
}

func TestStyledTable_Empty(t *testing.T) {
	table := &StyledTable{}
	result := table.String()
	if result != "" {
		t.Errorf("empty table should return empty string, got %q", result)
	}
}

func TestStyledTable_RowPadding(t *testing.T) {
	table := NewStyledTable("A", "B", "C")
	table.AddRow("1", "2") // Missing third column

	if len(table.rows) != 1 {
		t.Fatal("row should be added")
	}
	if len(table.rows[0]) != 3 {
		t.Errorf("row should be padded to 3 columns, got %d", len(table.rows[0]))
	}
}

func TestNewStatusLine(t *testing.T) {
	line := NewStatusLine()
	if line == nil {
		t.Fatal("NewStatusLine returned nil")
	}
}

func TestStatusLine_AddSuccess(t *testing.T) {
	line := NewStatusLine()
	line.AddSuccess("OK")
	if len(line.items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(line.items))
	}
	if line.items[0].icon != "✓" {
		t.Errorf("icon = %q, want %q", line.items[0].icon, "✓")
	}
}

func TestStatusLine_AddWarning(t *testing.T) {
	line := NewStatusLine()
	line.AddWarning("Caution")
	if len(line.items) != 1 {
		t.Fatal("item should be added")
	}
	if line.items[0].icon != "!" {
		t.Errorf("icon = %q, want %q", line.items[0].icon, "!")
	}
}

func TestStatusLine_AddError(t *testing.T) {
	line := NewStatusLine()
	line.AddError("Failed")
	if len(line.items) != 1 {
		t.Fatal("item should be added")
	}
	if line.items[0].icon != "✗" {
		t.Errorf("icon = %q, want %q", line.items[0].icon, "✗")
	}
}

func TestStatusLine_AddInfo(t *testing.T) {
	line := NewStatusLine()
	line.AddInfo("Note")
	if len(line.items) != 1 {
		t.Fatal("item should be added")
	}
	if line.items[0].icon != "→" {
		t.Errorf("icon = %q, want %q", line.items[0].icon, "→")
	}
}

func TestStatusLine_String(t *testing.T) {
	line := NewStatusLine()
	line.AddSuccess("OK").AddWarning("Warn").AddError("Fail")

	result := line.String()

	if !strings.Contains(result, "OK") {
		t.Errorf("result should contain OK: %q", result)
	}
	if !strings.Contains(result, "Warn") {
		t.Errorf("result should contain Warn: %q", result)
	}
	if !strings.Contains(result, "Fail") {
		t.Errorf("result should contain Fail: %q", result)
	}
}

func TestStatusLine_Empty(t *testing.T) {
	line := NewStatusLine()
	result := line.String()
	if result != "" {
		t.Errorf("empty status line should return empty string, got %q", result)
	}
}

func TestKeyValue_PlainMode(t *testing.T) {
	result := KeyValue("Name", "Value")
	if !strings.Contains(result, "Name:") {
		t.Errorf("result should contain key: %q", result)
	}
	if !strings.Contains(result, "Value") {
		t.Errorf("result should contain value: %q", result)
	}
}

func TestMuted_PlainMode(t *testing.T) {
	input := "muted text"
	result := Muted(input)
	if result != input {
		t.Errorf("Muted in plain mode = %q, want %q", result, input)
	}
}

func TestBold_PlainMode(t *testing.T) {
	input := "bold text"
	result := Bold(input)
	if result != input {
		t.Errorf("Bold in plain mode = %q, want %q", result, input)
	}
}

func TestCyan_PlainMode(t *testing.T) {
	input := "cyan text"
	result := Cyan(input)
	if result != input {
		t.Errorf("Cyan in plain mode = %q, want %q", result, input)
	}
}

func TestGreen_PlainMode(t *testing.T) {
	input := "green text"
	result := Green(input)
	if result != input {
		t.Errorf("Green in plain mode = %q, want %q", result, input)
	}
}

func TestYellow_PlainMode(t *testing.T) {
	input := "yellow text"
	result := Yellow(input)
	if result != input {
		t.Errorf("Yellow in plain mode = %q, want %q", result, input)
	}
}

func TestRed_PlainMode(t *testing.T) {
	input := "red text"
	result := Red(input)
	if result != input {
		t.Errorf("Red in plain mode = %q, want %q", result, input)
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello", "hello"},
		{"with color", "\x1b[31mred\x1b[0m", "red"},
		{"multiple codes", "\x1b[1m\x1b[32mbold green\x1b[0m", "bold green"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAnsi(tt.input)
			if got != tt.want {
				t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPadRightAnsi(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  int // expected length after stripping ANSI
	}{
		{"plain text", "hello", 10, 10},
		{"already at width", "hello", 5, 5},
		{"with ansi", "\x1b[31mred\x1b[0m", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padRightAnsi(tt.input, tt.width)
			gotPlain := stripAnsi(got)
			if len(gotPlain) != tt.want {
				t.Errorf("padRightAnsi(%q, %d) plain length = %d, want %d",
					tt.input, tt.width, len(gotPlain), tt.want)
			}
		})
	}
}
