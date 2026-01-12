package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// checkCmd validates schema files and detects drift.
func checkCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate schema and detect drift",
		Long: `Validate schema files and detect drift between migrations and database state.

This command performs two key operations:
1. Validates all schema files in the migrations directory for syntax errors
2. Compares the expected schema (from migrations) against the actual database state

If drift is detected, the command outputs detailed information about:
- Missing tables (defined in migrations but not in database)
- Extra tables (present in database but not in migrations)
- Modified tables (differences in columns, indexes, or constraints)

The command exits with status code 1 if validation fails or drift is detected.`,
		Example: `  # Check all schema files
  alab check

  # Check with JSON output for automation
  alab check --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			spinner := NewOptionalSpinner("Checking schema...", !jsonOutput)
			defer spinner.Stop()

			// Validate schema files
			spinner.Update("Validating schema files...")
			if err := client.SchemaCheck(); err != nil {
				spinner.Stop()
				if jsonOutput {
					outputJSON(map[string]any{
						"valid": false,
						"error": err.Error(),
					})
				} else {
					fmt.Fprint(os.Stderr, ui.FormatError(err))
				}
				os.Exit(1)
			}

			// Check drift
			spinner.Update("Checking for drift...")
			result, err := client.CheckDrift()
			spinner.Stop()
			if err != nil {
				if jsonOutput {
					outputJSON(map[string]any{
						"valid": true,
						"error": err.Error(),
					})
				} else {
					fmt.Fprintln(os.Stderr, ui.Error("error")+": drift check failed")
					fmt.Fprintf(os.Stderr, "  %v\n", err)
				}
				os.Exit(1)
			}

			if jsonOutput {
				outputJSON(map[string]any{
					"valid":     true,
					"has_drift": result.HasDrift,
					"hash":      result.ExpectedHash,
				})
				if result.HasDrift {
					os.Exit(1)
				}
			} else {
				if result.HasDrift {
					content := fmt.Sprintf("Expected: %s\nActual:   %s",
						truncateHash(result.ExpectedHash),
						truncateHash(result.ActualHash))
					fmt.Println(ui.RenderWarningPanel("Schema drift detected", content))
					fmt.Println()
					printDriftDetails(result)
					os.Exit(1)
				} else {
					var content string
					if result.Summary != nil {
						content = fmt.Sprintf("Tables: %d\nHash:   %s",
							result.Summary.Tables,
							truncateHash(result.ExpectedHash))
					} else {
						content = fmt.Sprintf("Hash: %s", truncateHash(result.ExpectedHash))
					}
					fmt.Println(ui.RenderSuccessPanel("Schema check passed", content))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	setupCommandHelp(cmd)
	return cmd
}

// outputJSON writes a JSON object to stdout.
func outputJSON(data map[string]any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

// printDriftDetails prints colored drift details.
func printDriftDetails(result *astroladb.DriftResult) {
	fmt.Printf("  Expected: %s\n", ui.Dim(truncateHash(result.ExpectedHash)))
	fmt.Printf("  Actual:   %s\n", ui.Dim(truncateHash(result.ActualHash)))
	fmt.Println()

	printDriftSection(ui.Error("Missing tables")+" (in migrations but not in database)", "-", result.MissingTables, ui.Failed)
	printDriftSection(ui.Warning("Extra tables")+" (in database but not in migrations)", "+", result.ExtraTables, ui.Warning)

	if len(result.TableDiffs) > 0 {
		fmt.Println("  " + ui.Info("Modified tables") + ":")
		for name, diff := range result.TableDiffs {
			fmt.Printf("\n    %s:\n", ui.Header(name))
			printTableDiffItems("column", diff.MissingColumns, diff.ExtraColumns, diff.ModifiedColumns)
			printTableDiffItems("index", diff.MissingIndexes, diff.ExtraIndexes, nil)
		}
		fmt.Println()
	}

	fmt.Println(ui.Help("help") + ": create a migration to reconcile differences:")
	fmt.Println("  alab new reconcile_drift")
}

// printTableDiffItems prints missing/extra/modified items for a table diff.
func printTableDiffItems(itemType string, missing, extra, modified []string) {
	for _, item := range missing {
		fmt.Printf("      %s %s %s\n", ui.Failed("-"), itemType, item)
	}
	for _, item := range extra {
		fmt.Printf("      %s %s %s\n", ui.Warning("+"), itemType, item)
	}
	for _, item := range modified {
		fmt.Printf("      %s %s %s\n", ui.Info("~"), itemType, item)
	}
}

// truncateHash returns the first 12 characters of a hash.
func truncateHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
