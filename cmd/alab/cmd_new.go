package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/spf13/cobra"
)

// toSnakeCase converts a string to lowercase snake_case.
// Examples: "CreateUser" -> "create_user", "create-user" -> "create_user"
func toSnakeCase(s string) string {
	// Replace hyphens, spaces, and dots with underscores
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")

	// Insert underscore before uppercase letters (for camelCase/PascalCase)
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(s[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				result.WriteRune('_')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}

	// Clean up multiple underscores and trim
	cleaned := regexp.MustCompile(`_+`).ReplaceAllString(result.String(), "_")
	cleaned = strings.Trim(cleaned, "_")

	return cleaned
}

// newCmd creates a migration file. If schema has changes, auto-generates from diff.
func newCmd() *cobra.Command {
	var empty bool

	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create migration (auto-generates if schema has changes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Normalize name to lowercase snake_case
			name := toSnakeCase(args[0])

			// If --empty flag, create empty migration without checking schema diff
			if empty {
				return createEmptyMigration(name)
			}

			// Try to auto-generate from schema diff
			client, err := newClient()
			if err != nil {
				// If we can't connect to DB, fall back to empty migration
				return createEmptyMigration(name)
			}
			defer client.Close()

			// Detect renames (drop + add pairs that might be renames)
			renameCandidates, err := client.DetectRenames()
			if err != nil {
				// Ignore errors, continue without renames
				renameCandidates = nil
			}

			// Prompt for renames if any detected
			var confirmedRenames []engine.RenameCandidate
			if len(renameCandidates) > 0 {
				confirmedRenames = promptForRenames(renameCandidates)
			}

			// Detect missing backfills (NOT NULL columns without defaults)
			backfillCandidates, err := client.DetectMissingBackfills()
			if err != nil {
				// Ignore errors, continue without backfills
				backfillCandidates = nil
			}

			// Prompt for backfills if any detected
			var backfills map[string]string
			if len(backfillCandidates) > 0 {
				backfills = promptForBackfills(backfillCandidates)
			}

			// Generate migration with renames and backfills
			var path string
			if len(confirmedRenames) > 0 || len(backfills) > 0 {
				path, err = client.MigrationGenerateInteractive(name, confirmedRenames, backfills)
			} else {
				path, err = client.MigrationGenerate(name)
			}

			if err != nil {
				// If generation fails (e.g., no changes), create empty migration
				return createEmptyMigration(name)
			}
			fmt.Printf("Generated migration: %s\n", path)
			return nil
		},
	}

	cmd.Flags().BoolVar(&empty, "empty", false, "Create empty migration")

	return cmd
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

// promptForRenames prompts the user to confirm rename candidates.
// Returns the list of confirmed renames.
func promptForRenames(candidates []engine.RenameCandidate) []engine.RenameCandidate {
	if len(candidates) == 0 {
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	var confirmed []engine.RenameCandidate

	fmt.Println()
	fmt.Println("Detected potential renames:")
	fmt.Println(strings.Repeat("-", 50))

	for _, c := range candidates {
		var prompt string
		if c.Type == "column" {
			tableRef := c.Table
			if c.Namespace != "" {
				tableRef = c.Namespace + "." + c.Table
			}
			prompt = fmt.Sprintf("  Column '%s' -> '%s' in table %s", c.OldName, c.NewName, tableRef)
		} else {
			tableRef := c.OldName
			if c.Namespace != "" {
				tableRef = c.Namespace + "." + c.OldName
			}
			newRef := c.NewName
			if c.Namespace != "" {
				newRef = c.Namespace + "." + c.NewName
			}
			prompt = fmt.Sprintf("  Table '%s' -> '%s'", tableRef, newRef)
		}

		fmt.Printf("%s\n", prompt)
		fmt.Print("  Is this a rename? [y/N]: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "y" || input == "yes" {
			confirmed = append(confirmed, c)
		}
	}

	if len(confirmed) > 0 {
		fmt.Printf("\nConfirmed %d rename(s).\n", len(confirmed))
	}

	return confirmed
}

// promptForBackfills prompts the user for backfill values for NOT NULL columns.
// Returns a map of "namespace.table.column" -> backfill value.
func promptForBackfills(candidates []engine.BackfillCandidate) map[string]string {
	if len(candidates) == 0 {
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	backfills := make(map[string]string)

	fmt.Println()
	fmt.Println("The following NOT NULL columns need backfill values for existing rows:")
	fmt.Println(strings.Repeat("-", 60))

	for _, c := range candidates {
		tableRef := c.Table
		if c.Namespace != "" {
			tableRef = c.Namespace + "." + c.Table
		}

		suggested := engine.SuggestDefault(c.ColType)

		fmt.Printf("\n  Column: %s.%s (%s)\n", tableRef, c.Column, c.ColType)
		fmt.Printf("  Suggested: %s\n", suggested)
		fmt.Printf("  Enter backfill value (or press Enter for suggested, 's' to skip): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		input = strings.TrimSpace(input)

		// Skip this column
		if strings.ToLower(input) == "s" || strings.ToLower(input) == "skip" {
			continue
		}

		// Use suggested value if empty
		if input == "" {
			input = suggested
		}

		key := c.Namespace + "." + c.Table + "." + c.Column
		backfills[key] = input
	}

	if len(backfills) > 0 {
		fmt.Printf("\nSet %d backfill value(s).\n", len(backfills))
	}

	return backfills
}
