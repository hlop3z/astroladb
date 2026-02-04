package main

import (
	"fmt"
	"os"

	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/internal/ui"
)

// -----------------------------------------------------------------------------
// Error Helpers
// -----------------------------------------------------------------------------

// errCreate returns a formatted error for creation failures.
// Usage: return errCreate("directory", path, err)
func errCreate(what, path string, err error) error {
	return fmt.Errorf("failed to create %s %s: %w", what, path, err)
}

// errRead returns a formatted error for read failures.
// Usage: return errRead("config file", path, err)
func errRead(what, path string, err error) error {
	return fmt.Errorf("failed to read %s %s: %w", what, path, err)
}

// errWrite returns a formatted error for write failures.
// Usage: return errWrite("output file", path, err)
func errWrite(what, path string, err error) error {
	return fmt.Errorf("failed to write %s %s: %w", what, path, err)
}

// errParse returns a formatted error for parse failures.
// Usage: return errParse("migration template", err)
func errParse(what string, err error) error {
	return fmt.Errorf("failed to parse %s: %w", what, err)
}

// printError prints an error message to stderr with consistent formatting.
// Usage: printError(err)
func printError(err error) {
	fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
}

// printWarning prints a warning message to stderr.
// Usage: printWarning("file not found")
func printWarning(msg string) {
	fmt.Fprintln(os.Stderr, ui.Warning("Warning")+": "+msg)
}

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

// OptionalSpinner wraps ui.Spinner with optional enable/disable support.
// This reduces boilerplate for commands that conditionally use spinners.
//
// Usage:
//
//	spinner := NewOptionalSpinner("Loading...", !jsonOutput && !schemaOnly)
//	defer spinner.Stop()
//	// ... do work ...
//	spinner.Update("Still loading...")
type OptionalSpinner struct {
	spinner *ui.Spinner
	enabled bool
}

// NewOptionalSpinner creates a new optional spinner.
// If enabled is false, all operations are no-ops.
func NewOptionalSpinner(message string, enabled bool) *OptionalSpinner {
	if !enabled {
		return &OptionalSpinner{enabled: false}
	}
	s := ui.NewSpinner(message)
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

// autoCommitMigrations commits uncommitted migration files after running migrate.
// direction should be "up" for migrate or "down" for rollback.
// Returns nil on success, error only for failures (not in git repo is not an error).
func autoCommitMigrations(migrationsDir, direction string) error {
	result, err := git.CommitAppliedMigrations(migrationsDir, direction)
	if err != nil {
		return err
	}

	if result != nil {
		fmt.Printf("Git: %s\n", result.Message)
	}

	return nil
}

// printContextInfo displays the config file, migrations dir, and masked database URL.
// Used by migrate and rollback commands to show context before executing.
func printContextInfo(cfg *Config) {
	dbDisplay := MaskDatabaseURL(cfg.Database.URL)

	ctx := &ui.ContextView{
		Pairs: map[string]string{
			"Config":     configFile,
			"Migrations": cfg.MigrationsDir,
			"Database":   dbDisplay,
		},
	}
	fmt.Println(ctx.Render())
	fmt.Println()
}
