package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/ui"
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
		Long: `Create a new migration file.

If your schema has changes, the migration will be auto-generated from the diff between your schema and database. If there are no changes, an empty migration will be created.

Migration names are automatically normalized to snake_case and prefixed with a sequential number.`,
		Example: `  # Auto-generate from schema changes
  alab new add_users_table

  # Create empty migration for manual changes
  alab new --empty custom_index

  # Migration naming (auto-normalized):
  alab new CreateUsers    # -> 001_create_users.js
  alab new add-email      # -> 002_add_email.js`,
		Args: cobra.ExactArgs(1),
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

			// Show success
			fmt.Println(ui.Success("Generated migration: " + ui.FilePath(path)))
			return nil
		},
	}

	cmd.Flags().BoolVar(&empty, "empty", false, "Create empty migration")

	setupCommandHelp(cmd)
	return cmd
}

// createEmptyMigration creates an empty migration file.
func createEmptyMigration(name string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Ensure migrations directory exists
	if err := os.MkdirAll(cfg.MigrationsDir, DirPerm); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Determine next revision number
	revision, err := nextRevision(cfg.MigrationsDir)
	if err != nil {
		return err
	}

	// Generate migration content using template
	tmplContent := mustReadTemplate("templates/migration.js.tmpl")
	tmpl, err := template.New("migration").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse migration template: %w", err)
	}

	// Find previous revision for down_revision linkage
	prevRevision := previousRevision(cfg.MigrationsDir)

	var buf bytes.Buffer
	data := struct {
		Name         string
		Timestamp    string
		UpRevision   string
		DownRevision string
	}{
		Name:         name,
		Timestamp:    time.Now().UTC().Format(TimeJSON),
		UpRevision:   revision,
		DownRevision: prevRevision,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute migration template: %w", err)
	}
	content := buf.String()

	// Write migration file
	filename := fmt.Sprintf("%s_%s.js", revision, name)
	path := filepath.Join(cfg.MigrationsDir, filename)

	if err := os.WriteFile(path, []byte(content), FilePerm); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	// Show success
	fmt.Println(ui.Success("Created migration: " + ui.FilePath(path)))
	return nil
}

// nextRevision determines the next migration revision number.
// Uses format NNN_YYYYMMDD_HHMMSS (e.g., "003_20260201_143022").
func nextRevision(migrationsDir string) (string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("001_%s", time.Now().UTC().Format("20060102_150405")), nil
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

	return fmt.Sprintf("%03d_%s", maxNum+1, time.Now().UTC().Format("20060102_150405")), nil
}

// previousRevision returns the revision string of the latest existing migration,
// or an empty string if there are no migrations yet.
func previousRevision(migrationsDir string) string {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return ""
	}

	var latest string
	maxNum := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		// Extract the full revision (everything before the last underscore-separated name)
		base := strings.TrimSuffix(entry.Name(), ".js")
		parts := strings.SplitN(base, "_", 2)
		if num, err := strconv.Atoi(parts[0]); err == nil && num > maxNum {
			maxNum = num
			// Revision is the prefix up to the migration name
			// e.g. "003_20260201_143022_add_users.js" → revision is "003_20260201_143022"
			// Find the revision by stripping the name suffix
			if idx := strings.LastIndex(base, "_"); idx > 0 {
				candidate := base[:idx]
				// If candidate still has underscore-separated parts ending in a name, keep trimming
				// The revision format is NNN_YYYYMMDD_HHMMSS
				// Simple approach: take first 3 underscore-separated segments
				segs := strings.SplitN(base, "_", 4)
				if len(segs) >= 3 {
					latest = segs[0] + "_" + segs[1] + "_" + segs[2]
				} else {
					latest = candidate
				}
			}
		}
	}

	return latest
}

// promptForRenames prompts the user to confirm rename candidates.
// Returns the list of confirmed renames.
func promptForRenames(candidates []engine.RenameCandidate) []engine.RenameCandidate {
	if len(candidates) == 0 {
		return nil
	}

	var confirmed []engine.RenameCandidate

	fmt.Println()
	fmt.Println(ui.RenderSubtitle("Detected Potential Renames"))
	fmt.Println()

	for _, c := range candidates {
		var message string
		if c.Type == "column" {
			tableRef := c.Table
			if c.Namespace != "" {
				tableRef = c.Namespace + "." + c.Table
			}
			message = fmt.Sprintf("Column %s in table %s",
				ui.FormatChange("", c.OldName, c.NewName),
				ui.Bold(tableRef))
		} else {
			oldRef := c.OldName
			if c.Namespace != "" {
				oldRef = c.Namespace + "." + c.OldName
			}
			newRef := c.NewName
			if c.Namespace != "" {
				newRef = c.Namespace + "." + c.NewName
			}
			message = fmt.Sprintf("Table %s", ui.FormatChange("", oldRef, newRef))
		}

		if ui.Confirm(message+"\n  Is this a rename?", false) {
			confirmed = append(confirmed, c)
		}
		fmt.Println()
	}

	if len(confirmed) > 0 {
		fmt.Printf("%s Confirmed %s\n\n",
			ui.Success(""),
			ui.FormatCount(len(confirmed), "rename", "renames"))
	}

	return confirmed
}

// promptForBackfills prompts the user for backfill values for NOT NULL columns.
// Returns a map of "namespace.table.column" -> backfill value.
func promptForBackfills(candidates []engine.BackfillCandidate) map[string]string {
	if len(candidates) == 0 {
		return nil
	}

	backfills := make(map[string]string)

	fmt.Println()
	fmt.Println(ui.RenderSubtitle("Backfill Values Required"))
	fmt.Println()
	fmt.Println(ui.Info("The following NOT NULL columns need backfill values for existing rows"))
	fmt.Println()

	for _, c := range candidates {
		tableRef := c.Table
		if c.Namespace != "" {
			tableRef = c.Namespace + "." + c.Table
		}

		suggested := engine.SuggestDefault(c.ColType)

		// Show column info
		fmt.Printf("  Column: %s.%s (%s)\n",
			ui.Bold(tableRef),
			ui.Primary(c.Column),
			ui.Dim(c.ColType))

		// Prompt for value
		input, err := ui.Prompt(ui.PromptConfig{
			Message:      "Backfill value (or 'skip')",
			DefaultValue: suggested,
			Required:     false,
		})
		if err != nil {
			continue
		}

		// Skip this column
		if strings.ToLower(input) == "skip" || strings.ToLower(input) == "s" {
			fmt.Println(ui.Dim("  Skipped"))
			fmt.Println()
			continue
		}

		key := c.Namespace + "." + c.Table + "." + c.Column
		backfills[key] = input
		fmt.Println(ui.Success("  Set to: ") + input)
		fmt.Println()
	}

	if len(backfills) > 0 {
		fmt.Printf("%s Set %s\n\n",
			ui.Success(""),
			ui.FormatCount(len(backfills), "backfill value", "backfill values"))
	}

	return backfills
}
