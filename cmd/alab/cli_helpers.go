package main

import (
	"fmt"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
)

// printDriftSection prints a section of drift results with consistent formatting.
// Used by cmd_check.go to display missing/extra tables, columns, indexes, etc.
//
// Parameters:
//   - title: The section title (e.g., "Missing tables")
//   - symbol: The prefix symbol (e.g., "-", "+", "~")
//   - items: The list of items to display
//   - styleFn: A styling function from alabcli (e.g., alabcli.Failed, alabcli.Warning)
func printDriftSection(title, symbol string, items []string, styleFn func(string) string) {
	if len(items) == 0 {
		return
	}
	fmt.Println("  " + title + ":")
	for _, item := range items {
		fmt.Printf("    %s %s\n", styleFn(symbol), item)
	}
	fmt.Println()
}

// OptionalSpinner wraps alabcli.Spinner with optional enable/disable support.
// This reduces boilerplate for commands that conditionally use spinners.
//
// Usage:
//
//	spinner := NewOptionalSpinner("Loading...", !jsonOutput && !schemaOnly)
//	defer spinner.Stop()
//	// ... do work ...
//	spinner.Update("Still loading...")
type OptionalSpinner struct {
	spinner *alabcli.Spinner
	enabled bool
}

// NewOptionalSpinner creates a new optional spinner.
// If enabled is false, all operations are no-ops.
func NewOptionalSpinner(message string, enabled bool) *OptionalSpinner {
	if !enabled {
		return &OptionalSpinner{enabled: false}
	}
	s := alabcli.NewSpinner(message)
	s.Start()
	return &OptionalSpinner{spinner: s, enabled: true}
}

// Update changes the spinner message. No-op if disabled.
func (o *OptionalSpinner) Update(message string) {
	if o.enabled && o.spinner != nil {
		o.spinner.Update(message)
	}
}

// Stop stops the spinner. No-op if disabled. Safe to call multiple times.
func (o *OptionalSpinner) Stop() {
	if o.enabled && o.spinner != nil {
		o.spinner.Stop()
	}
}
