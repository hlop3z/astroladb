package main

import (
	"encoding/json"
	"fmt"
	"os"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// checkCmd validates schema files and detects drift.
func checkCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate schema and detect drift",
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
					fmt.Fprint(os.Stderr, alabcli.FormatError(err))
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
					fmt.Fprintln(os.Stderr, alabcli.Error("error")+": drift check failed")
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
					fmt.Println(alabcli.RenderWarningPanel("Schema drift detected", content))
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
					fmt.Println(alabcli.RenderSuccessPanel("Schema check passed", content))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

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
	fmt.Printf("  Expected: %s\n", alabcli.Dim(truncateHash(result.ExpectedHash)))
	fmt.Printf("  Actual:   %s\n", alabcli.Dim(truncateHash(result.ActualHash)))
	fmt.Println()

	printDriftSection(alabcli.Error("Missing tables")+" (in migrations but not in database)", "-", result.MissingTables, alabcli.Failed)
	printDriftSection(alabcli.Warning("Extra tables")+" (in database but not in migrations)", "+", result.ExtraTables, alabcli.Warning)

	if len(result.TableDiffs) > 0 {
		fmt.Println("  " + alabcli.Info("Modified tables") + ":")
		for name, diff := range result.TableDiffs {
			fmt.Printf("\n    %s:\n", alabcli.Header(name))
			printTableDiffItems("column", diff.MissingColumns, diff.ExtraColumns, diff.ModifiedColumns)
			printTableDiffItems("index", diff.MissingIndexes, diff.ExtraIndexes, nil)
		}
		fmt.Println()
	}

	fmt.Println(alabcli.Help("help") + ": create a migration to reconcile differences:")
	fmt.Println("  alab new reconcile_drift")
}

// printTableDiffItems prints missing/extra/modified items for a table diff.
func printTableDiffItems(itemType string, missing, extra, modified []string) {
	for _, item := range missing {
		fmt.Printf("      %s %s %s\n", alabcli.Failed("-"), itemType, item)
	}
	for _, item := range extra {
		fmt.Printf("      %s %s %s\n", alabcli.Warning("+"), itemType, item)
	}
	for _, item := range modified {
		fmt.Printf("      %s %s %s\n", alabcli.Info("~"), itemType, item)
	}
}

// truncateHash returns the first 12 characters of a hash.
func truncateHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
