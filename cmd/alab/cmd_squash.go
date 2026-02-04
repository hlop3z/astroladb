package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/chain"
	"github.com/hlop3z/astroladb/internal/lockfile"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
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

	// Load schema and generate baseline content (schema-only client for baseline generation)
	schemaClient, err := newSchemaOnlyClient()
	if err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}
	defer schemaClient.Close()

	ops, err := schemaClient.GenerateBaselineOps()
	if err != nil {
		return fmt.Errorf("failed to generate baseline: %w", err)
	}

	if len(ops) == 0 {
		fmt.Println(ui.Warning("Schema has no tables — nothing to squash"))
		return nil
	}

	// Try to get database client for history archival (optional - squash works without DB)
	var dbClient *astroladb.Client
	var migrationHistory []astroladb.MigrationHistoryEntry
	dbClient, err = newClient()
	if err == nil {
		defer dbClient.Close()
		// Get migration history for archival
		migrationHistory, _ = dbClient.MigrationHistory()
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
		if len(migrationHistory) > 0 {
			fmt.Println()
			fmt.Printf("  Database records to archive: %s\n",
				ui.FormatCount(len(migrationHistory), "record", "records"))
		}
		return nil
	}

	baselineContent, err := schemaClient.GenerateBaselineContent(lastRevision, len(migFiles))
	if err != nil {
		return fmt.Errorf("failed to generate baseline content: %w", err)
	}

	// Archive old migrations
	archiveDir := filepath.Join(".alab", "archive", time.Now().UTC().Format("20060102_150405"))
	if err := os.MkdirAll(archiveDir, DirPerm); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Archive migration files
	for _, f := range migFiles {
		src := filepath.Join(cfg.MigrationsDir, f)
		dst := filepath.Join(archiveDir, f)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to archive %s: %w", f, err)
		}
	}

	// Archive database migration history to JSON file (if available)
	var dbRecordsArchived int
	if len(migrationHistory) > 0 {
		historyPath := filepath.Join(archiveDir, "migration_history.json")
		if err := archiveMigrationHistory(historyPath, migrationHistory); err != nil {
			// Non-fatal: warn but continue
			fmt.Fprintf(os.Stderr, "  %s: could not archive migration history: %v\n", ui.Warning("warning"), err)
		} else {
			dbRecordsArchived = len(migrationHistory)
		}
	}

	// Write baseline migration file
	baselinePath := filepath.Join(cfg.MigrationsDir, "001_baseline.js")
	if err := os.WriteFile(baselinePath, []byte(baselineContent), FilePerm); err != nil {
		return fmt.Errorf("failed to write baseline: %w", err)
	}

	// Compute checksum for the new baseline file
	baselineChecksum := ""
	migrationChain, err := chain.ComputeFromDir(cfg.MigrationsDir)
	if err == nil && len(migrationChain.Links) > 0 {
		baselineChecksum = migrationChain.Links[0].Checksum
	}

	// Update database: clear old records and insert baseline record
	if dbClient != nil && len(migrationHistory) > 0 {
		ctx := context.Background()

		// Clear old migration records
		_, err := dbClient.ClearMigrationHistory(ctx)
		if err != nil {
			// Non-fatal: warn but continue
			fmt.Fprintf(os.Stderr, "  %s: could not clear migration history: %v\n", ui.Warning("warning"), err)
		} else {
			// Record the new baseline
			err = dbClient.RecordSquashedBaseline(ctx, "001", baselineChecksum, lastRevision, len(migFiles))
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s: could not record baseline: %v\n", ui.Warning("warning"), err)
			}
		}
	}

	// Update lock file
	lockPath := lockfile.DefaultPath()
	_ = lockfile.Write(cfg.MigrationsDir, lockPath)

	// Build success message
	successMsg := fmt.Sprintf("Squashed %s into baseline\n  Archived to: %s\n  Baseline: %s\n  Squashed through: %s",
		ui.FormatCount(len(migFiles), "migration", "migrations"),
		ui.Dim(archiveDir),
		ui.Primary(baselinePath),
		lastRevision,
	)
	if dbRecordsArchived > 0 {
		successMsg += fmt.Sprintf("\n  DB records archived: %d", dbRecordsArchived)
	}

	ui.ShowSuccess("Squash Complete", successMsg)

	return nil
}

// archiveMigrationHistory writes migration history to a JSON file.
func archiveMigrationHistory(path string, history []astroladb.MigrationHistoryEntry) error {
	// Convert to a serializable format
	type archivedEntry struct {
		Revision     string `json:"revision"`
		Name         string `json:"name"`
		AppliedAt    string `json:"applied_at"`
		Checksum     string `json:"checksum,omitempty"`
		ExecTimeMs   int    `json:"exec_time_ms,omitempty"`
		Description  string `json:"description,omitempty"`
		SQLChecksum  string `json:"sql_checksum,omitempty"`
		AppliedOrder int    `json:"applied_order,omitempty"`
	}

	archived := make([]archivedEntry, len(history))
	for i, h := range history {
		archived[i] = archivedEntry{
			Revision:     h.Revision,
			Name:         h.Name,
			AppliedAt:    h.AppliedAt.Format(time.RFC3339),
			Checksum:     h.Checksum,
			ExecTimeMs:   h.ExecTimeMs,
			Description:  h.Description,
			SQLChecksum:  h.SQLChecksum,
			AppliedOrder: h.AppliedOrder,
		}
	}

	data, err := json.MarshalIndent(archived, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, FilePerm)
}
