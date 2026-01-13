package ui

import (
	"fmt"
	"strings"
)

// ContextView displays key-value pairs in a formatted layout.
type ContextView struct {
	Pairs     map[string]string
	MaxKeyLen int
}

// Render renders the context view as a formatted string.
func (cv *ContextView) Render() string {
	if len(cv.Pairs) == 0 {
		return ""
	}

	// Calculate max key length if not set
	maxLen := cv.MaxKeyLen
	if maxLen == 0 {
		for key := range cv.Pairs {
			if len(key) > maxLen {
				maxLen = len(key)
			}
		}
	}

	var output strings.Builder

	// Sort keys for consistent output
	keys := make([]string, 0, len(cv.Pairs))
	for key := range cv.Pairs {
		keys = append(keys, key)
	}

	// Display each pair
	for _, key := range keys {
		value := cv.Pairs[key]
		padding := maxLen - len(key)
		output.WriteString(fmt.Sprintf("  %s:%s %s\n",
			Dim(key),
			strings.Repeat(" ", padding),
			value,
		))
	}

	return output.String()
}

// String returns the rendered context view.
func (cv *ContextView) String() string {
	return cv.Render()
}
