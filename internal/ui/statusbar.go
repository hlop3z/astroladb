package ui

import (
	"strings"

	"github.com/rivo/tview"
)

// StatusBar represents a reusable status bar component with keyboard hints.
type StatusBar struct {
	*tview.TextView
}

// NewStatusBar creates a new status bar with the given hints text.
// The hints text should describe keyboard shortcuts, e.g., "q quit  ↑↓ navigate"
func NewStatusBar(hints string) *StatusBar {
	bar := &StatusBar{
		TextView: tview.NewTextView(),
	}

	bar.SetText(" " + hints + " ").
		SetTextColor(Theme.TextDim).
		SetTextAlign(tview.AlignCenter).
		SetBackgroundColor(Theme.Background)

	return bar
}

// SetHints updates the hints text.
func (s *StatusBar) SetHints(hints string) *StatusBar {
	s.SetText(" " + hints + " ")
	return s
}

// DefaultStatusHints returns standard hints for the unified Status view.
func DefaultStatusHints() string {
	return strings.TrimSpace(HintsStatus)
}

// HeaderBar creates a header bar with title and optional context info.
type HeaderBar struct {
	*tview.TextView
}

// NewHeaderBar creates a new header bar.
func NewHeaderBar(title string) *HeaderBar {
	h := &HeaderBar{
		TextView: tview.NewTextView(),
	}

	h.SetText(" " + title + " ").
		SetTextColor(Theme.Text).
		SetTextAlign(tview.AlignLeft).
		SetBackgroundColor(Theme.Primary)

	return h
}

// SetContext adds context information to the right side of the header.
func (h *HeaderBar) SetContext(context string) *HeaderBar {
	// Pad the title to push context to the right
	// This is a simple implementation; for true right-alignment,
	// you might need to calculate terminal width
	h.SetText(" " + h.GetText(false) + "                                     " + context + " ")
	return h
}
