package main

import (
	"fmt"

	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/internal/ui"
)

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
