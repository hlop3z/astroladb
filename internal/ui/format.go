package ui

import (
	"fmt"
	"time"
)

// FormatCount formats a count with singular/plural forms.
// Example: FormatCount(1, "file", "files") => "1 file"
// Example: FormatCount(5, "file", "files") => "5 files"
func FormatCount(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// FormatDuration formats a time.Duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// FormatChange formats an old → new value change with colors.
// Example: FormatChange("", "old_name", "new_name") => "old_name → new_name"
func FormatChange(prefix, oldValue, newValue string) string {
	arrow := Dim("→")
	old := Red(oldValue)
	new := Green(newValue)

	if prefix != "" {
		return fmt.Sprintf("%s %s %s %s", prefix, old, arrow, new)
	}
	return fmt.Sprintf("%s %s %s", old, arrow, new)
}

// padRight pads a string to the right with spaces to reach the desired width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + fmt.Sprintf("%*s", width-len(s), "")
}

// Help returns help text with dimmed styling.
func Help(text string) string {
	return Dim(text)
}

// Note returns note text with special formatting.
func Note(text string) string {
	return Dim("Note: " + text)
}
