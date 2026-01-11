package main

import (
	"encoding/json"
	"fmt"
	"os"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
	"github.com/spf13/cobra"
)

// historyCmd shows applied migrations from the database with details.
func historyCmd() *cobra.Command {
	var jsonOutput bool
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show applied migrations from database with details",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			history, err := client.MigrationHistory()
			if err != nil {
				return err
			}

			if len(history) == 0 {
				if jsonOutput {
					outputJSON(map[string]any{
						"count":      0,
						"migrations": []any{},
					})
				} else {
					fmt.Println("No migrations have been applied.")
				}
				return nil
			}

			// Show newest first (reverse chronological)
			for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
				history[i], history[j] = history[j], history[i]
			}

			// Apply limit
			if limit > 0 && limit < len(history) {
				history = history[:limit]
			}

			// JSON output mode
			if jsonOutput {
				migrations := make([]map[string]any, len(history))
				var totalExecTime int64
				for i, h := range history {
					m := map[string]any{
						"revision":     h.Revision,
						"name":         h.Name,
						"applied_at":   h.AppliedAt.Format(TimeJSON),
						"checksum":     h.Checksum,
						"exec_time_ms": h.ExecTimeMs,
					}
					migrations[i] = m
					totalExecTime += int64(h.ExecTimeMs)
				}

				output := map[string]any{
					"count":         len(history),
					"total_exec_ms": totalExecTime,
					"migrations":    migrations,
				}

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(output)
			}

			// Human-readable styled table output
			fmt.Println(alabcli.RenderTitle("Migration History"))
			fmt.Println()
			fmt.Printf("  %s\n\n", alabcli.Muted(fmt.Sprintf("%d migrations applied", len(history))))

			table := alabcli.NewStyledTable("REVISION", "NAME", "APPLIED AT", "EXEC TIME", "CHECKSUM")

			var totalExecTime int64
			for _, h := range history {
				execTime := alabcli.Muted("-")
				if h.ExecTimeMs > 0 {
					if h.ExecTimeMs >= 1000 {
						execTime = alabcli.Yellow(fmt.Sprintf("%.2fs", float64(h.ExecTimeMs)/1000))
					} else {
						execTime = alabcli.Green(fmt.Sprintf("%dms", h.ExecTimeMs))
					}
					totalExecTime += int64(h.ExecTimeMs)
				}

				// Truncate checksum for display
				checksum := h.Checksum
				if len(checksum) > 12 {
					checksum = checksum[:12]
				}

				table.AddRow(
					h.Revision,
					h.Name,
					h.AppliedAt.Format(TimeDisplay),
					execTime,
					alabcli.Muted(checksum),
				)
			}

			fmt.Print(table.String())

			if totalExecTime > 0 {
				var execTimeStr string
				if totalExecTime >= 1000 {
					execTimeStr = fmt.Sprintf("%.2fs", float64(totalExecTime)/1000)
				} else {
					execTimeStr = fmt.Sprintf("%dms", totalExecTime)
				}
				fmt.Printf("\n  Total execution time: %s\n", alabcli.Cyan(execTimeStr))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().IntVarP(&limit, "limit", "n", 0, "Limit to N migrations")

	return cmd
}
