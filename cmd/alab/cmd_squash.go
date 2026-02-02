package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/lockfile"
	"github.com/hlop3z/astroladb/internal/ui"
)

// squashCmd squashes all existing migrations into a single baseline migration.
func squashCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "squash",
		Short: "Squash migrations into a baseline",
		Long: `Collapse all existing migrations into a single baseline migration.

The baseline captures the current schema state as create_table + create_index
operations. Old migration files are archived to .alab/archive/.

Existing environments (already migrated) are unaffected — the runner detects
the baseline and skips it. New environments apply only the baseline.`,
		Example: `  # Preview what would be squashed
  alab squash --dry-run

  # Squash all migrations into a baseline
  alab squash`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSquash(dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without writing")

	setupCommandHelp(cmd)
	return cmd
}

func runSquash(dryRun bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Read existing migrations
	entries, err := os.ReadDir(cfg.MigrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(ui.Success("No migrations to squash"))
			return nil
		}
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migFiles []string
	var lastRevision string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		migFiles = append(migFiles, e.Name())
		name := strings.TrimSuffix(e.Name(), ".js")
		parts := strings.SplitN(name, "_", 2)
		if len(parts) > 0 {
			lastRevision = parts[0]
		}
	}

	if len(migFiles) == 0 {
		fmt.Println(ui.Success("No migrations to squash"))
		return nil
	}

	if len(migFiles) == 1 {
		fmt.Println(ui.Warning("Only 1 migration — nothing to squash"))
		return nil
	}

	// Load schema and generate baseline content
	client, err := newSchemaOnlyClient()
	if err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}
	defer client.Close()

	ops, err := client.GenerateBaselineOps()
	if err != nil {
		return fmt.Errorf("failed to generate baseline: %w", err)
	}

	if len(ops) == 0 {
		fmt.Println(ui.Warning("Schema has no tables — nothing to squash"))
		return nil
	}

	if dryRun {
		fmt.Println(ui.RenderTitle("Squash Preview"))
		fmt.Println()
		fmt.Printf("  Would squash %s into a single baseline\n",
			ui.FormatCount(len(migFiles), "migration", "migrations"))
		fmt.Printf("  Baseline contains %s\n",
			ui.FormatCount(len(ops), "operation", "operations"))
		fmt.Printf("  Squashed through revision: %s\n", lastRevision)
		fmt.Println()
		fmt.Println("  Files to archive:")
		for _, f := range migFiles {
			fmt.Printf("    %s\n", ui.Dim(f))
		}
		return nil
	}

	baselineContent, err := client.GenerateBaselineContent(lastRevision, len(migFiles))
	if err != nil {
		return fmt.Errorf("failed to generate baseline content: %w", err)
	}

	// Archive old migrations
	archiveDir := filepath.Join(".alab", "archive", time.Now().UTC().Format("20060102_150405"))
	if err := os.MkdirAll(archiveDir, DirPerm); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	for _, f := range migFiles {
		src := filepath.Join(cfg.MigrationsDir, f)
		dst := filepath.Join(archiveDir, f)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to archive %s: %w", f, err)
		}
	}

	// Write baseline migration file
	baselinePath := filepath.Join(cfg.MigrationsDir, "001_baseline.js")
	if err := os.WriteFile(baselinePath, []byte(baselineContent), FilePerm); err != nil {
		return fmt.Errorf("failed to write baseline: %w", err)
	}

	// Update lock file
	lockPath := lockfile.DefaultPath()
	_ = lockfile.Write(cfg.MigrationsDir, lockPath)

	ui.ShowSuccess(
		"Squash Complete",
		fmt.Sprintf("Squashed %s into baseline\n  Archived to: %s\n  Baseline: %s\n  Squashed through: %s",
			ui.FormatCount(len(migFiles), "migration", "migrations"),
			ui.Dim(archiveDir),
			ui.Primary(baselinePath),
			lastRevision,
		),
	)

	return nil
}
