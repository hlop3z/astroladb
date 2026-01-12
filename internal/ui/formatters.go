package ui

import (
	"fmt"
	"strings"
	"time"
)

// FormatDuration formats a duration in human-readable form
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatBytes formats bytes in human-readable form
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatList formats a slice as a bulleted list
func FormatList(items []string, bulletStyle string) string {
	if bulletStyle == "" {
		bulletStyle = "•"
	}

	var b strings.Builder
	for _, item := range items {
		b.WriteString(fmt.Sprintf("  %s %s\n", Dim(bulletStyle), item))
	}
	return b.String()
}

// FormatKeyValuePairs formats key-value pairs in aligned columns
func FormatKeyValuePairs(pairs map[string]string) string {
	if len(pairs) == 0 {
		return ""
	}

	// Find max key length for alignment
	maxLen := 0
	for key := range pairs {
		if len(key) > maxLen {
			maxLen = len(key)
		}
	}

	var b strings.Builder
	for key, value := range pairs {
		padding := strings.Repeat(" ", maxLen-len(key))
		b.WriteString(fmt.Sprintf("  %s:%s %s\n",
			Dim(key),
			padding,
			Primary(value),
		))
	}

	return b.String()
}

// FormatDiff formats a before/after diff with color coding
func FormatDiff(before, after string) string {
	var b strings.Builder

	if before != "" {
		b.WriteString(Error("- ") + Dim(before) + "\n")
	}
	if after != "" {
		b.WriteString(Success("+ ") + Primary(after) + "\n")
	}

	return b.String()
}

// FormatChange formats a modification with arrow notation
func FormatChange(name, before, after string) string {
	return fmt.Sprintf("%s: %s %s %s",
		Bold(name),
		Dim(before),
		Dim("→"),
		Primary(after),
	)
}

// FormatSQLPreview formats SQL with basic syntax highlighting
func FormatSQLPreview(sql string) string {
	keywords := []string{
		"CREATE", "DROP", "ALTER", "TABLE", "COLUMN", "INDEX",
		"ADD", "REMOVE", "RENAME", "TO", "FROM",
		"PRIMARY", "KEY", "FOREIGN", "REFERENCES",
		"NOT", "NULL", "DEFAULT", "UNIQUE",
	}

	result := sql
	for _, keyword := range keywords {
		// Case-insensitive replacement
		result = strings.ReplaceAll(result, keyword, Bold(keyword))
		result = strings.ReplaceAll(result, strings.ToLower(keyword), Bold(keyword))
	}

	return Dim(result)
}

// FormatTimestamp formats a timestamp in human-readable form
func FormatTimestamp(t time.Time) string {
	now := time.Now()

	// If today, show time only
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04:05")
	}

	// If this year, show month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 02 15:04")
	}

	// Otherwise, show full date
	return t.Format("2006-01-02 15:04")
}

// FormatFileSize formats file size with appropriate units
func FormatFileSize(size int64) string {
	return FormatBytes(size)
}

// FormatPlural returns singular or plural form based on count
func FormatPlural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// FormatChangeMarker returns a colored change marker
func FormatChangeMarker(changeType string) string {
	switch changeType {
	case "added", "create", "new":
		return Success("+")
	case "modified", "changed", "alter":
		return Warning("~")
	case "removed", "deleted", "drop":
		return Error("-")
	default:
		return Dim("•")
	}
}

// FormatStatus formats a status badge
func FormatStatus(status string) string {
	switch strings.ToLower(status) {
	case "applied", "success", "complete", "done":
		return Success("✓ " + status)
	case "pending", "waiting":
		return Warning("○ " + status)
	case "failed", "error":
		return Error("✗ " + status)
	case "running", "processing":
		return Info("→ " + status)
	default:
		return Dim(status)
	}
}
