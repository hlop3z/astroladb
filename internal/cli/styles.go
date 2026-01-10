package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Modern color palette
var (
	colorPrimary   = lipgloss.Color("12")  // Blue
	colorSuccess   = lipgloss.Color("10")  // Green
	colorWarning   = lipgloss.Color("11")  // Yellow
	colorError     = lipgloss.Color("9")   // Red
	colorMuted     = lipgloss.Color("8")   // Gray
	colorHighlight = lipgloss.Color("14")  // Cyan
	colorWhite     = lipgloss.Color("15")  // White
	colorBg        = lipgloss.Color("236") // Dark gray background
)

// Badge styles for status indicators
var (
	badgeApplied = lipgloss.NewStyle().
			Background(colorSuccess).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true)

	badgePending = lipgloss.NewStyle().
			Background(colorWarning).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true)

	badgeError = lipgloss.NewStyle().
			Background(colorError).
			Foreground(colorWhite).
			Padding(0, 1).
			Bold(true)

	badgeInfo = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorWhite).
			Padding(0, 1).
			Bold(true)
)

// Panel styles for boxes and containers
var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)

	panelSuccess = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSuccess).
			Padding(1, 2)

	panelWarning = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorWarning).
			Padding(1, 2)

	panelError = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(1, 2)

	panelInfo = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)
)

// Title styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHighlight).
			MarginBottom(1)
)

// Render functions that respect EnableColors()

// RenderBadge renders a styled badge.
func RenderBadge(text string, style lipgloss.Style) string {
	if !EnableColors() {
		return "[" + text + "]"
	}
	return style.Render(text)
}

// RenderAppliedBadge renders an "applied" status badge.
func RenderAppliedBadge() string {
	return RenderBadge("APPLIED", badgeApplied)
}

// RenderPendingBadge renders a "pending" status badge.
func RenderPendingBadge() string {
	return RenderBadge("PENDING", badgePending)
}

// RenderErrorBadge renders an "error" badge.
func RenderErrorBadge() string {
	return RenderBadge("ERROR", badgeError)
}

// RenderInfoBadge renders an "info" badge.
func RenderInfoBadge(text string) string {
	return RenderBadge(text, badgeInfo)
}

// RenderPanel renders content in a bordered panel.
func RenderPanel(content string) string {
	if !EnableColors() {
		return content
	}
	return panelStyle.Render(content)
}

// RenderSuccessPanel renders content in a success-styled panel.
func RenderSuccessPanel(title, content string) string {
	if !EnableColors() {
		return fmt.Sprintf("✓ %s\n%s", title, content)
	}

	titleRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSuccess).
		Render("✓ " + title)

	return panelSuccess.Render(titleRendered + "\n\n" + content)
}

// RenderWarningPanel renders content in a warning-styled panel.
func RenderWarningPanel(title, content string) string {
	if !EnableColors() {
		return fmt.Sprintf("! %s\n%s", title, content)
	}

	titleRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWarning).
		Render("⚠ " + title)

	return panelWarning.Render(titleRendered + "\n\n" + content)
}

// RenderErrorPanel renders content in an error-styled panel.
func RenderErrorPanel(title, content string) string {
	if !EnableColors() {
		return fmt.Sprintf("✗ %s\n%s", title, content)
	}

	titleRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorError).
		Render("✗ " + title)

	return panelError.Render(titleRendered + "\n\n" + content)
}

// RenderInfoPanel renders content in an info-styled panel.
func RenderInfoPanel(title, content string) string {
	if !EnableColors() {
		return fmt.Sprintf("→ %s\n%s", title, content)
	}

	titleRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render("→ " + title)

	return panelInfo.Render(titleRendered + "\n\n" + content)
}

// RenderTitle renders a styled title.
func RenderTitle(text string) string {
	if !EnableColors() {
		return "═══ " + text + " ═══"
	}
	return titleStyle.Render(text)
}

// RenderSubtitle renders a styled subtitle.
func RenderSubtitle(text string) string {
	if !EnableColors() {
		return "── " + text + " ──"
	}
	return subtitleStyle.Render(text)
}

// StyledTable provides a modern table with borders.
type StyledTable struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewStyledTable creates a new styled table.
func NewStyledTable(headers ...string) *StyledTable {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &StyledTable{
		headers: headers,
		widths:  widths,
	}
}

// AddRow adds a row to the styled table.
func (t *StyledTable) AddRow(cells ...string) {
	for len(cells) < len(t.headers) {
		cells = append(cells, "")
	}
	for i, cell := range cells {
		if i < len(t.widths) {
			// Account for ANSI codes when measuring width
			plainCell := stripAnsi(cell)
			if len(plainCell) > t.widths[i] {
				t.widths[i] = len(plainCell)
			}
		}
	}
	t.rows = append(t.rows, cells)
}

// String renders the styled table.
func (t *StyledTable) String() string {
	if len(t.headers) == 0 {
		return ""
	}

	if !EnableColors() {
		return t.renderPlain()
	}

	return t.renderStyled()
}

func (t *StyledTable) renderPlain() string {
	var b strings.Builder

	// Header
	for i, h := range t.headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(padRight(h, t.widths[i]))
	}
	b.WriteString("\n")

	// Separator
	for i, w := range t.widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("-", w))
	}
	b.WriteString("\n")

	// Rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(t.widths) {
				break
			}
			if i > 0 {
				b.WriteString("  ")
			}
			b.WriteString(padRightAnsi(cell, t.widths[i]))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (t *StyledTable) renderStyled() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorHighlight)
	borderStyle := lipgloss.NewStyle().Foreground(colorMuted)
	cellStyle := lipgloss.NewStyle().Foreground(colorWhite)

	// Calculate total width
	totalWidth := 0
	for _, w := range t.widths {
		totalWidth += w + 3 // cell + padding
	}
	totalWidth-- // Remove last padding

	// Top border
	b.WriteString(borderStyle.Render("╭" + strings.Repeat("─", totalWidth+2) + "╮"))
	b.WriteString("\n")

	// Header row
	b.WriteString(borderStyle.Render("│") + " ")
	for i, h := range t.headers {
		if i > 0 {
			b.WriteString(borderStyle.Render(" │ "))
		}
		b.WriteString(headerStyle.Render(padRight(h, t.widths[i])))
	}
	b.WriteString(" " + borderStyle.Render("│"))
	b.WriteString("\n")

	// Header separator
	b.WriteString(borderStyle.Render("├" + strings.Repeat("─", totalWidth+2) + "┤"))
	b.WriteString("\n")

	// Data rows
	for _, row := range t.rows {
		b.WriteString(borderStyle.Render("│") + " ")
		for i, cell := range row {
			if i >= len(t.widths) {
				break
			}
			if i > 0 {
				b.WriteString(borderStyle.Render(" │ "))
			}
			// Cell might already have styling, so use padRightAnsi
			b.WriteString(cellStyle.Render(padRightAnsi(cell, t.widths[i])))
		}
		b.WriteString(" " + borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Bottom border
	b.WriteString(borderStyle.Render("╰" + strings.Repeat("─", totalWidth+2) + "╯"))
	b.WriteString("\n")

	return b.String()
}

// StatusLine renders a status line with icon and message.
type StatusLine struct {
	items []statusItem
}

type statusItem struct {
	icon    string
	message string
	style   lipgloss.Style
}

// NewStatusLine creates a new status line.
func NewStatusLine() *StatusLine {
	return &StatusLine{}
}

// AddSuccess adds a success item.
func (s *StatusLine) AddSuccess(msg string) *StatusLine {
	s.items = append(s.items, statusItem{
		icon:    "✓",
		message: msg,
		style:   lipgloss.NewStyle().Foreground(colorSuccess),
	})
	return s
}

// AddWarning adds a warning item.
func (s *StatusLine) AddWarning(msg string) *StatusLine {
	s.items = append(s.items, statusItem{
		icon:    "!",
		message: msg,
		style:   lipgloss.NewStyle().Foreground(colorWarning),
	})
	return s
}

// AddError adds an error item.
func (s *StatusLine) AddError(msg string) *StatusLine {
	s.items = append(s.items, statusItem{
		icon:    "✗",
		message: msg,
		style:   lipgloss.NewStyle().Foreground(colorError),
	})
	return s
}

// AddInfo adds an info item.
func (s *StatusLine) AddInfo(msg string) *StatusLine {
	s.items = append(s.items, statusItem{
		icon:    "→",
		message: msg,
		style:   lipgloss.NewStyle().Foreground(colorPrimary),
	})
	return s
}

// String renders the status line.
func (s *StatusLine) String() string {
	var parts []string
	for _, item := range s.items {
		if EnableColors() {
			parts = append(parts, item.style.Render(item.icon+" "+item.message))
		} else {
			parts = append(parts, item.icon+" "+item.message)
		}
	}
	return strings.Join(parts, "  ")
}

// Helper to strip ANSI codes for width calculation
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// padRightAnsi pads a string that may contain ANSI codes.
func padRightAnsi(s string, width int) string {
	plainLen := len(stripAnsi(s))
	if plainLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-plainLen)
}

// KeyValue renders a key-value pair nicely.
func KeyValue(key, value string) string {
	if !EnableColors() {
		return fmt.Sprintf("%s: %s", key, value)
	}
	keyStyle := lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(colorWhite)
	return keyStyle.Render(key+":") + " " + valueStyle.Render(value)
}

// Muted renders muted/dim text.
func Muted(s string) string {
	if !EnableColors() {
		return s
	}
	return lipgloss.NewStyle().Foreground(colorMuted).Render(s)
}

// Bold renders bold text.
func Bold(s string) string {
	if !EnableColors() {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Render(s)
}

// Cyan renders cyan text.
func Cyan(s string) string {
	if !EnableColors() {
		return s
	}
	return lipgloss.NewStyle().Foreground(colorHighlight).Render(s)
}

// Green renders green text.
func Green(s string) string {
	if !EnableColors() {
		return s
	}
	return lipgloss.NewStyle().Foreground(colorSuccess).Render(s)
}

// Yellow renders yellow text.
func Yellow(s string) string {
	if !EnableColors() {
		return s
	}
	return lipgloss.NewStyle().Foreground(colorWarning).Render(s)
}

// Red renders red text.
func Red(s string) string {
	if !EnableColors() {
		return s
	}
	return lipgloss.NewStyle().Foreground(colorError).Render(s)
}
