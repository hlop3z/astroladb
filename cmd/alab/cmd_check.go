package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/lockfile"
	"github.com/hlop3z/astroladb/internal/ui"
)

// checkCmd validates schema files or lints migration files.
func checkCmd() *cobra.Command {
	var migrations, determinism bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate schemas or lint migrations",
		Long: `Validate schema files for correctness or lint migration files for safety issues.

Default mode validates schema files:
  - Parses all schema files in the schemas directory
  - Checks for syntax errors, invalid types, and other issues
  - Verifies deterministic evaluation

With --migrations, lints migration files:
  - Checks for destructive operations (DROP TABLE, DROP COLUMN)
  - Detects NOT NULL columns without defaults
  - Flags reserved SQL word usage in identifiers

With --determinism, verifies SQL output reproducibility:
  - Re-parses each applied migration file
  - Regenerates SQL and compares against stored SQL checksum
  - Reports non-deterministic migrations`,
		Example: `  # Validate all schema files
  alab check

  # Lint migration files for safety issues
  alab check --migrations

  # Verify SQL determinism for applied migrations
  alab check --determinism`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if determinism {
				return checkDeterminism()
			}
			if migrations {
				return checkMigrations()
			}
			return checkSchemas()
		},
	}

	cmd.Flags().BoolVar(&migrations, "migrations", false, "Lint migration files instead of validating schemas")
	cmd.Flags().BoolVar(&determinism, "determinism", false, "Verify SQL checksum determinism for applied migrations")

	setupCommandHelp(cmd)
	return cmd
}

// checkSchemas validates all schema files.
func checkSchemas() error {
	client, err := newSchemaOnlyClient()
	if err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}
	defer client.Close()

	if err := client.SchemaCheck(); err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}

	fmt.Println(ui.Success("All schemas valid"))
	return nil
}

// checkDeterminism verifies that applied migrations produce the same SQL checksum.
func checkDeterminism() error {
	client, err := newClient()
	if err != nil {
		if handleClientError(err) {
			os.Exit(1)
		}
		return err
	}
	defer client.Close()

	results, err := client.VerifySQLDeterminism()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println(ui.Success("No applied migrations with SQL checksums to verify"))
		return nil
	}

	mismatches := 0
	for _, r := range results {
		if r.Match {
			fmt.Printf("  %s %s %s\n", ui.Success("✓"), r.Revision, r.Name)
		} else {
			mismatches++
			fmt.Fprintf(os.Stderr, "  %s %s %s\n", ui.Error("✗"), r.Revision, r.Name)
			fmt.Fprintf(os.Stderr, "    stored:  %s\n", r.StoredChecksum)
			fmt.Fprintf(os.Stderr, "    current: %s\n", r.CurrentChecksum)
		}
	}

	if mismatches > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, ui.Error(fmt.Sprintf("%d non-deterministic migration(s) detected", mismatches)))
		os.Exit(1)
	}

	fmt.Println(ui.Success(fmt.Sprintf("All %d migration(s) produce deterministic SQL", len(results))))
	return nil
}

// checkMigrations lints all migration files for safety issues.
func checkMigrations() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(cfg.MigrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(ui.Success("No migrations to check"))
			return nil
		}
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	client, err := newSchemaOnlyClient()
	if err != nil {
		if !handleClientError(err) {
			fmt.Fprintln(os.Stderr, ui.Error("error")+": "+err.Error())
		}
		os.Exit(1)
	}
	defer client.Close()

	// Verify lock file if it exists
	lockPath := lockfile.DefaultPath()
	if err := lockfile.Verify(cfg.MigrationsDir, lockPath); err != nil {
		fmt.Fprintf(os.Stderr, "  %s Lock file: %v\n", ui.Warning("WARN"), err)
	}

	var allWarnings []engine.Warning
	fileCount := 0
	hasErrors := false

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		fileCount++

		path := filepath.Join(cfg.MigrationsDir, entry.Name())
		ops, err := client.ParseMigrationFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %s: %v\n", ui.Error("ERROR"), entry.Name(), err)
			hasErrors = true
			continue
		}

		warnings := engine.LintOperations(ops)
		for i := range warnings {
			warnings[i].Message = fmt.Sprintf("[%s] %s", entry.Name(), warnings[i].Message)
		}
		allWarnings = append(allWarnings, warnings...)
	}

	if hasErrors {
		os.Exit(1)
	}

	if len(allWarnings) > 0 {
		fmt.Println(ui.RenderTitle("Migration Lint Results"))
		fmt.Println()
		for _, w := range allWarnings {
			switch w.Severity {
			case "error":
				fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Error("ERROR"), w.Message)
			default:
				fmt.Fprintf(os.Stderr, "  %s %s\n", ui.Warning("WARN"), w.Message)
			}
		}
		fmt.Println()
		fmt.Fprintf(os.Stderr, "%s\n", ui.Warning(fmt.Sprintf("%d warning(s) in %d migration(s)", len(allWarnings), fileCount)))
		os.Exit(1)
	}

	fmt.Println(ui.Success(fmt.Sprintf("All %d migration(s) passed lint checks", fileCount)))
	return nil
}
