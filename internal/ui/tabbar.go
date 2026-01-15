package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// TabBar represents a reusable tab bar component.
type TabBar struct {
	*tview.TextView
	tabs        []string
	activeIndex int
}

// NewTabBar creates a new tab bar with the given tabs.
// activeIndex specifies which tab should be highlighted (0-indexed).
func NewTabBar(tabs []string, activeIndex int) *TabBar {
	bar := &TabBar{
		TextView:    tview.NewTextView(),
		tabs:        tabs,
		activeIndex: activeIndex,
	}

	bar.SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBackgroundColor(Theme.Background)

	bar.render()
	return bar
}

// SetActiveTab sets the active tab by index and re-renders.
func (t *TabBar) SetActiveTab(index int) *TabBar {
	if index >= 0 && index < len(t.tabs) {
		t.activeIndex = index
		t.render()
	}
	return t
}

// SetActiveTabByName sets the active tab by name and re-renders.
func (t *TabBar) SetActiveTabByName(name string) *TabBar {
	for i, tab := range t.tabs {
		if strings.Contains(strings.ToLower(tab), strings.ToLower(name)) {
			t.activeIndex = i
			t.render()
			break
		}
	}
	return t
}

// GetActiveIndex returns the current active tab index.
func (t *TabBar) GetActiveIndex() int {
	return t.activeIndex
}

// render updates the tab bar display.
func (t *TabBar) render() {
	var text strings.Builder
	text.WriteString(" ")

	for i, tab := range t.tabs {
		if i == t.activeIndex {
			// Active tab: inverted colors (black on white)
			text.WriteString(fmt.Sprintf(TagSelected+" %s "+TagEnd+" ", tab))
		} else {
			// Inactive tab: normal
			text.WriteString(fmt.Sprintf(" %s  ", tab))
		}
	}

	t.Clear()
	t.SetText(text.String())
}

// TabBarFromNames creates a tab bar from simple names with numbered shortcuts.
// E.g., ["Status", "History", "Verify"] becomes ["1 Status", "2 History", "3 Verify"]
func TabBarFromNames(names []string, activeIndex int) *TabBar {
	tabs := make([]string, len(names))
	for i, name := range names {
		tabs[i] = fmt.Sprintf("%d %s", i+1, name)
	}
	return NewTabBar(tabs, activeIndex)
}
