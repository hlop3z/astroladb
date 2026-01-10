package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/git"
	"github.com/spf13/cobra"
)

// newCmd creates a migration file. If schema has changes, auto-generates from diff.
func newCmd() *cobra.Command {
	var empty, noInteractive, noCommit, push bool

	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create migration (auto-generates if schema has changes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			var migrationPath string

			// If --empty flag, create empty migration without checking schema diff
			if empty {
				if err := createEmptyMigration(name); err != nil {
					return err
				}
				// Get the path (last created file)
				revision, _ := nextRevision(cfg.MigrationsDir)
				// Revision is next, so decrement for the one we just created
				if rev, _ := strconv.Atoi(revision); rev > 1 {
					revision = fmt.Sprintf("%03d", rev-1)
				}
				migrationPath = filepath.Join(cfg.MigrationsDir, fmt.Sprintf("%s_%s.js", revision, name))
			} else {
				// Try to auto-generate from schema diff
				client, err := newClient()
				if err != nil {
					// If we can't connect to DB, fall back to empty migration
					if err := createEmptyMigration(name); err != nil {
						return err
					}
					revision, _ := nextRevision(cfg.MigrationsDir)
					if rev, _ := strconv.Atoi(revision); rev > 1 {
						revision = fmt.Sprintf("%03d", rev-1)
					}
					migrationPath = filepath.Join(cfg.MigrationsDir, fmt.Sprintf("%s_%s.js", revision, name))
				} else {
					defer client.Close()

					interactive := !noInteractive

					// Collect user inputs for renames and backfills
					var confirmedRenames []engine.RenameCandidate
					var backfills map[string]string

					if interactive {
						// Check for potential renames
						renameCandidates, err := client.DetectRenames()
						if err == nil && len(renameCandidates) > 0 {
							confirmedRenames = promptForRenames(renameCandidates)
						}

						// Check for missing backfills
						backfillCandidates, err := client.DetectMissingBackfills()
						if err == nil && len(backfillCandidates) > 0 {
							backfills = promptForBackfills(backfillCandidates)
						}
					}

					// Generate migration with user inputs applied
					if len(confirmedRenames) > 0 || len(backfills) > 0 {
						path, err := client.MigrationGenerateInteractive(name, confirmedRenames, backfills)
						if err != nil {
							if err := createEmptyMigration(name); err != nil {
								return err
							}
							revision, _ := nextRevision(cfg.MigrationsDir)
							if rev, _ := strconv.Atoi(revision); rev > 1 {
								revision = fmt.Sprintf("%03d", rev-1)
							}
							migrationPath = filepath.Join(cfg.MigrationsDir, fmt.Sprintf("%s_%s.js", revision, name))
						} else {
							fmt.Printf("Generated migration: %s\n", path)
							migrationPath = path
						}
					} else {
						// Standard generation (no interactive changes)
						path, err := client.MigrationGenerate(name)
						if err != nil {
							// If generation fails (e.g., no changes), create empty migration
							if err := createEmptyMigration(name); err != nil {
								return err
							}
							revision, _ := nextRevision(cfg.MigrationsDir)
							if rev, _ := strconv.Atoi(revision); rev > 1 {
								revision = fmt.Sprintf("%03d", rev-1)
							}
							migrationPath = filepath.Join(cfg.MigrationsDir, fmt.Sprintf("%s_%s.js", revision, name))
						} else {
							fmt.Printf("Generated migration: %s\n", path)
							migrationPath = path
						}
					}
				}
			}

			// Auto-commit unless disabled
			if !noCommit && migrationPath != "" {
				result, err := git.CommitMigration(cfg.MigrationsDir, migrationPath, name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to commit: %v\n", err)
				} else if result != nil {
					fmt.Println()
					fmt.Println(git.FormatCommitSuccess(result.Files, result.CommitHash, "Add migration: "+name))

					// Push if requested
					if push {
						pushResult, err := git.PushIfReady(cfg.MigrationsDir)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Warning: Failed to push: %v\n", err)
						} else if pushResult != nil && pushResult.Pushed {
							fmt.Println(pushResult.Message)
						}
					} else if result.ShouldPush {
						fmt.Println("\nTip: Run 'git push' or use 'alab new --push' to push automatically")
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&empty, "empty", false, "Force creation of empty migration (skip auto-generation)")
	cmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Skip interactive prompts (renames, backfills)")
	cmd.Flags().BoolVar(&noCommit, "no-commit", false, "Skip auto-commit of migration file")
	cmd.Flags().BoolVar(&push, "push", false, "Push after committing")

	return cmd
}

// promptForRenames asks the user to confirm potential renames.
func promptForRenames(candidates []engine.RenameCandidate) []engine.RenameCandidate {
	if len(candidates) == 0 {
		return nil
	}

	fmt.Println("\nDetected possible renames:")
	fmt.Println()

	var confirmed []engine.RenameCandidate
	reader := bufio.NewReader(os.Stdin)

	for _, c := range candidates {
		var prompt string
		if c.Type == "column" {
			tableRef := c.Table
			if c.Namespace != "" {
				tableRef = c.Namespace + "." + c.Table
			}
			prompt = fmt.Sprintf("  Did you rename column '%s.%s' to '%s'? [y/N] ",
				tableRef, c.OldName, c.NewName)
		} else {
			tableRef := c.OldName
			newRef := c.NewName
			if c.Namespace != "" {
				tableRef = c.Namespace + "." + c.OldName
				newRef = c.Namespace + "." + c.NewName
			}
			prompt = fmt.Sprintf("  Did you rename table '%s' to '%s'? [y/N] ",
				tableRef, newRef)
		}

		fmt.Print(prompt)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "y" || answer == "yes" {
			confirmed = append(confirmed, c)
		}
	}

	if len(confirmed) > 0 {
		fmt.Println()
	}

	return confirmed
}

// promptForBackfills asks the user for backfill values for NOT NULL columns.
func promptForBackfills(candidates []engine.BackfillCandidate) map[string]string {
	if len(candidates) == 0 {
		return nil
	}

	fmt.Println("\nYou are adding non-nullable columns without defaults.")
	fmt.Println("Existing rows need a value. Please provide a backfill value for each:")
	fmt.Println()

	backfills := make(map[string]string)
	reader := bufio.NewReader(os.Stdin)

	for _, c := range candidates {
		tableRef := c.Table
		if c.Namespace != "" {
			tableRef = c.Namespace + "." + c.Table
		}

		suggested := engine.SuggestDefault(c.ColType)

		fmt.Printf("  Column '%s.%s' (%s)\n", tableRef, c.Column, c.ColType)
		fmt.Printf("    1) Provide a value now\n")
		fmt.Printf("    2) Use suggested: %s\n", suggested)
		fmt.Printf("    3) Skip (I'll add .backfill() manually)\n")
		fmt.Print("  Select [1/2/3]: ")

		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		key := c.Namespace + "." + c.Table + "." + c.Column

		switch answer {
		case "1":
			fmt.Print("  Enter value: ")
			value, _ := reader.ReadString('\n')
			value = strings.TrimSpace(value)
			if value != "" {
				backfills[key] = value
			}
		case "2":
			backfills[key] = suggested
		case "3":
			// Skip - user will handle manually
		default:
			// Default to suggested
			backfills[key] = suggested
		}
		fmt.Println()
	}

	return backfills
}

// createEmptyMigration creates an empty migration file.
func createEmptyMigration(name string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Ensure migrations directory exists
	if err := os.MkdirAll(cfg.MigrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Determine next revision number
	revision, err := nextRevision(cfg.MigrationsDir)
	if err != nil {
		return err
	}

	// Generate migration content
	content := fmt.Sprintf(`// Migration: %s
// Generated at: %s

export function up(m) {
  // Examples:
  // m.create_table("namespace.table", t => { t.id(); t.string("name", 100); })
  // m.add_column("namespace.table", c => c.integer("count").default(0))
  // m.create_index("namespace.table", ["column1", "column2"])
  // m.rename_column("namespace.table", "old_name", "new_name")
  // m.drop_column("namespace.table", "column_name")
}

export function down(m) {
  // Reverse the operations in up() in reverse order
}
`, name, time.Now().Format(TimeJSON))

	// Write migration file
	filename := fmt.Sprintf("%s_%s.js", revision, name)
	path := filepath.Join(cfg.MigrationsDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("Created migration: %s\n", path)
	return nil
}

// nextRevision determines the next migration revision number.
func nextRevision(migrationsDir string) (string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "001", nil
		}
		return "", err
	}

	maxNum := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if num, err := strconv.Atoi(parts[0]); err == nil && num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("%03d", maxNum+1), nil
}
