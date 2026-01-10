package main

import "time"

// Time format constants for consistent formatting across the CLI.
// Use these instead of inline format strings.
const (
	// TimeJSON is RFC3339 format for JSON output (e.g., "2006-01-02T15:04:05Z07:00").
	TimeJSON = time.RFC3339

	// TimeDisplay is a human-friendly format for table output (e.g., "2006-01-02 15:04").
	TimeDisplay = "2006-01-02 15:04"

	// TimeFull is a detailed format with seconds (e.g., "2006-01-02 15:04:05").
	TimeFull = "2006-01-02 15:04:05"
)
