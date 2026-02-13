package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/internal/lockfile"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
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

// writeFileEnsureDir creates parent directories if needed and writes data to path.
func writeFileEnsureDir(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, DirPerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return os.WriteFile(path, data, FilePerm)
}

// mustClient creates a new astroladb client or exits on failure.
// Use for commands that require a database connection and cannot recover from errors.
func mustClient() *astroladb.Client {
	client, err := newClient()
	if err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}
	return client
}

// mustSchemaOnlyClient creates a schema-only client or exits on failure.
// Use for commands that only read schemas and cannot recover from errors.
func mustSchemaOnlyClient() *astroladb.Client {
	client, err := newSchemaOnlyClient()
	if err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}
	return client
}

// confirmOrCancel shows a confirmation prompt and returns true if confirmed.
// If the user declines, prints a dimmed cancel message and returns false.
func confirmOrCancel(prompt string, defaultYes bool, cancelMsg string) bool {
	if !ui.Confirm(prompt, defaultYes) {
		fmt.Println(ui.Dim(cancelMsg))
		return false
	}
	fmt.Println()
	return true
}

// mustConfig loads the configuration or exits on failure.
func mustConfig() *Config {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		os.Exit(1)
	}
	return cfg
}

// executeTemplate parses an embedded template and executes it with the given data.
func executeTemplate(templatePath string, data any) (string, error) {
	tmplContent := mustReadTemplate(templatePath)
	tmpl, err := template.New(filepath.Base(templatePath)).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}
	return buf.String(), nil
}

// readMigrationFiles returns the .js filenames in the migrations directory.
// Returns nil, nil if the directory doesn't exist.
func readMigrationFiles(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		files = append(files, e.Name())
	}
	return files, nil
}

// updateLockFile regenerates the lock file from the current migrations directory.
func updateLockFile(migrationsDir string) {
	lockPath := lockfile.DefaultPath()
	if err := lockfile.Write(migrationsDir, lockPath); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", ui.Warning("Warning: failed to update lock file"), err)
	}
}

// buildMigrationOpts builds the common migration options shared by migrate and rollback.
func buildMigrationOpts(dryRun, skipLock bool, lockTimeout time.Duration) []astroladb.MigrationOption {
	var opts []astroladb.MigrationOption
	if dryRun {
		opts = append(opts, astroladb.DryRunTo(os.Stdout))
	}
	if skipLock {
		opts = append(opts, astroladb.SkipLock())
	}
	if lockTimeout > 0 {
		opts = append(opts, astroladb.LockTimeout(lockTimeout))
	}
	return opts
}

// autoCommitIfRequested commits migration files to git if the --commit flag is set.
func autoCommitIfRequested(commit bool, migrationsDir, direction string) {
	if !commit {
		return
	}
	if err := autoCommitMigrations(migrationsDir, direction); err != nil {
		fmt.Fprintf(os.Stderr, "\n"+ui.Warning("Warning")+": %v\n", err)
	} else {
		fmt.Println()
		fmt.Println(ui.Success(MsgCommittedToGit))
	}
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
