package ui

import "fmt"

// App defines application-level configuration.
var App = struct {
	// App info
	Name    string
	Version string

	// Status header format (use %s for revision, %d for applied, %d for pending)
	HeaderFormat   string
	HeaderNoRevFmt string
}{
	Name:           "Astrola DB Center",
	Version:        "",
	HeaderFormat:   " %s | rev: %s | A:%d | P:%d ",
	HeaderNoRevFmt: " %s | A:%d | P:%d ",
}

// FormatHeader returns the formatted header with revision info.
func FormatHeader(revision string, applied, pending int) string {
	if revision == "" {
		return fmt.Sprintf(App.HeaderNoRevFmt, App.Name, applied, pending)
	}
	return fmt.Sprintf(App.HeaderFormat, App.Name, revision, applied, pending)
}

// Layout defines all UI sizing and layout configuration.
// Centralizes layout values for easy modification and consistency.
var Layout = struct {
	// Browse tab panel widths
	BrowseTablesWidth  int
	BrowseDetailsWidth int

	// History tab panel proportions (flex ratios)
	HistoryListRatio    int
	HistoryDetailsRatio int

	// Verify tab panel proportions
	VerifyChainRatio int
	VerifyGitRatio   int

	// Drift tab panel proportions
	DriftListRatio    int
	DriftDetailsRatio int

	// Header and status bar heights
	HeaderHeight    int
	StatusBarHeight int
	TabBarHeight    int
	SpacerHeight    int

	// Summary bar height (for History tab)
	SummaryHeight int

	// Table column expansion (0 = no expand, 1+ = flex weight)
	TableExpandMain  int // Main/name column expansion
	TableExpandOther int // Other columns expansion

	// Checksum display truncation
	ChecksumDisplayLen int

	// Execution time threshold (ms) for warning color
	ExecTimeWarningMs int

	// SQL preview truncation length
	SQLPreviewMaxLen int
}{
	// Browse tab: Tables | Columns (flex) | Details
	BrowseTablesWidth:  45,
	BrowseDetailsWidth: 60,

	// History tab: List (70%) | Details (30%)
	HistoryListRatio:    7,
	HistoryDetailsRatio: 3,

	// Verify tab: Chain (50%) | Git (50%)
	VerifyChainRatio: 1,
	VerifyGitRatio:   1,

	// Drift tab: List (70%) | Details (30%)
	DriftListRatio:    7,
	DriftDetailsRatio: 3,

	// Common heights
	HeaderHeight:    1,
	StatusBarHeight: 1,
	TabBarHeight:    1,
	SpacerHeight:    1,
	SummaryHeight:   1,

	// Table columns - expand to fill available space
	TableExpandMain:  2, // Name columns get more space
	TableExpandOther: 1, // Other columns get less

	// Checksums: show first N characters
	ChecksumDisplayLen: 8,

	// Show warning color if exec time >= this value
	ExecTimeWarningMs: 1000,

	// SQL preview: truncate after N characters
	SQLPreviewMaxLen: 500,
}
