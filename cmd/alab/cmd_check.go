package main

import (
	"encoding/json"
	"fmt"
	"os"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// checkCmd validates schema files and optionally detects drift.
func checkCmd() *cobra.Command {
	var schemaOnly, driftOnly, quick, jsonOutput bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate schema files and detect drift (chain integrity + database comparison)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Schema-only mode doesn't need database connection
			var client *astroladb.Client
			var err error
			if schemaOnly {
				client, err = newSchemaOnlyClient()
			} else {
				client, err = newClient()
			}
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			// Use optional spinner for drift check (non-JSON mode)
			spinner := NewOptionalSpinner("Checking schema...", !jsonOutput && !schemaOnly)
			defer spinner.Stop()

			// Validate schema files unless drift-only
			if !driftOnly {
				spinner.Update("Validating schema files...")
				if err := client.SchemaCheck(); err != nil {
					spinner.Stop()
					if jsonOutput {
						outputJSON(map[string]any{
							"schema_valid": false,
							"error":        err.Error(),
						})
					} else {
						fmt.Fprint(os.Stderr, alabcli.FormatError(err))
					}
					os.Exit(1)
				}
			}

			// Check drift unless schema-only
			if !schemaOnly {
				spinner.Update("Checking for schema drift...")

				if quick {
					// Quick check - just compare root hashes
					match, err := client.QuickDriftCheck()
					spinner.Stop()
					if err != nil {
						if jsonOutput {
							outputJSON(map[string]any{
								"schema_valid": true,
								"drift_check":  "error",
								"error":        err.Error(),
							})
						} else {
							fmt.Fprintln(os.Stderr, alabcli.Error("error")+": drift check failed")
							fmt.Fprintf(os.Stderr, "  %v\n", err)
						}
						os.Exit(1)
					}

					if jsonOutput {
						outputJSON(map[string]any{
							"schema_valid": true,
							"has_drift":    !match,
						})
					} else {
						if match {
							fmt.Println(alabcli.RenderSuccessPanel("No drift detected", "Schema hashes match"))
						} else {
							fmt.Println(alabcli.RenderWarningPanel("Drift detected", "Run 'alab check' without --quick for details"))
							os.Exit(1)
						}
					}
				} else {
					// Full drift check with details
					result, err := client.CheckDrift()
					spinner.Stop()
					if err != nil {
						if jsonOutput {
							outputJSON(map[string]any{
								"schema_valid": true,
								"drift_check":  "error",
								"error":        err.Error(),
							})
						} else {
							fmt.Fprintln(os.Stderr, alabcli.Error("error")+": drift check failed")
							fmt.Fprintf(os.Stderr, "  %v\n", err)
						}
						os.Exit(1)
					}

					if jsonOutput {
						outputJSON(map[string]any{
							"schema_valid":    true,
							"has_drift":       result.HasDrift,
							"expected_hash":   result.ExpectedHash,
							"actual_hash":     result.ActualHash,
							"missing_tables":  result.MissingTables,
							"extra_tables":    result.ExtraTables,
							"modified_tables": len(result.TableDiffs),
							"summary":         result.Summary,
						})
						if result.HasDrift {
							os.Exit(1)
						}
					} else {
						if result.HasDrift {
							// Use warning panel for drift
							content := fmt.Sprintf("Expected: %s\nActual:   %s",
								truncateHash(result.ExpectedHash),
								truncateHash(result.ActualHash))
							fmt.Println(alabcli.RenderWarningPanel("Schema drift detected", content))
							fmt.Println()
							printDriftDetails(result)
							os.Exit(1)
						} else {
							// Use success panel for passed check
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
				}
			} else {
				spinner.Stop()
				if jsonOutput {
					outputJSON(map[string]any{
						"schema_valid": true,
					})
				} else {
					fmt.Println(alabcli.RenderSuccessPanel("Schema validation passed", "All schema files are valid"))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Only validate schema files (skip drift detection)")
	cmd.Flags().BoolVar(&driftOnly, "drift-only", false, "Only check for drift (skip schema validation)")
	cmd.Flags().BoolVar(&quick, "quick", false, "Quick drift check (compare merkle roots only)")
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
	// Hash comparison
	fmt.Printf("  Expected: %s\n", alabcli.Dim(truncateHash(result.ExpectedHash)))
	fmt.Printf("  Actual:   %s\n", alabcli.Dim(truncateHash(result.ActualHash)))
	fmt.Println()

	// Missing tables (in migrations but not in DB)
	printDriftSection(alabcli.Error("Missing tables")+" (in migrations but not in database)", "-", result.MissingTables, alabcli.Failed)

	// Extra tables (in DB but not in migrations)
	printDriftSection(alabcli.Warning("Extra tables")+" (in database but not in migrations)", "+", result.ExtraTables, alabcli.Warning)

	// Modified tables (more complex structure, keep inline)
	if len(result.TableDiffs) > 0 {
		fmt.Println("  " + alabcli.Info("Modified tables") + ":")
		for name, diff := range result.TableDiffs {
			fmt.Printf("\n    %s:\n", alabcli.Header(name))
			printTableDiffItems("column", diff.MissingColumns, diff.ExtraColumns, diff.ModifiedColumns)
			printTableDiffItems("index", diff.MissingIndexes, diff.ExtraIndexes, nil)
		}
		fmt.Println()
	}

	// Help
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
