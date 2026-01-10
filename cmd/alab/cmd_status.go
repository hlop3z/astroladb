package main

import (
	"encoding/json"
	"fmt"
	"os"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
	"github.com/spf13/cobra"
)

// statusCmd shows the status of all migrations.
func statusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show applied/pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			statuses, err := client.MigrationStatus()
			if err != nil {
				return err
			}

			// JSON output mode for CI/CD integration
			if jsonOutput {
				// Count applied and pending
				var applied, pending int
				for _, s := range statuses {
					if s.Status == "applied" {
						applied++
					} else {
						pending++
					}
				}

				// Build migration list with proper JSON formatting
				migrations := make([]map[string]any, len(statuses))
				for i, s := range statuses {
					m := map[string]any{
						"revision":   s.Revision,
						"name":       s.Name,
						"status":     s.Status,
						"applied_at": nil,
					}
					if s.AppliedAt != nil {
						m["applied_at"] = s.AppliedAt.Format(TimeJSON)
					}
					migrations[i] = m
				}

				output := map[string]any{
					"applied":    applied,
					"pending":    pending,
					"migrations": migrations,
				}

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(output); err != nil {
					return err
				}

				// Exit with code 1 if there are pending migrations
				// This helps CI/CD pipelines detect when migrations are needed
				if pending > 0 {
					os.Exit(1)
				}
				return nil
			}

			// Human-readable output
			if len(statuses) == 0 {
				fmt.Println(alabcli.Info("No migrations found."))
				return nil
			}

			// Count applied and pending for summary
			var appliedCount, pendingCount int
			for _, s := range statuses {
				if s.Status == "applied" {
					appliedCount++
				} else {
					pendingCount++
				}
			}

			// Print title and summary
			fmt.Println(alabcli.RenderTitle("Migration Status"))
			fmt.Println()

			if pendingCount > 0 {
				fmt.Printf("  %s  %s\n\n",
					alabcli.Green(fmt.Sprintf("%d applied", appliedCount)),
					alabcli.Yellow(fmt.Sprintf("%d pending", pendingCount)))
			} else {
				fmt.Printf("  %s\n\n",
					alabcli.Green(fmt.Sprintf("%d applied", appliedCount)))
			}

			// Build styled table with borders
			table := alabcli.NewStyledTable("REVISION", "NAME", "STATUS", "APPLIED AT")
			for _, s := range statuses {
				appliedAt := ""
				if s.AppliedAt != nil {
					appliedAt = s.AppliedAt.Format(TimeDisplay)
				}

				// Use styled badges for status
				var statusBadge string
				if s.Status == "applied" {
					statusBadge = alabcli.RenderAppliedBadge()
				} else {
					statusBadge = alabcli.RenderPendingBadge()
				}

				table.AddRow(s.Revision, s.Name, statusBadge, appliedAt)
			}

			fmt.Print(table.String())
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON for CI/CD")

	return cmd
}
