package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// rollbackCmd rolls back migrations.
func rollbackCmd() *cobra.Command {
	var dryRun, commit bool

	cmd := &cobra.Command{
		Use:   "rollback [steps]",
		Short: "Rollback migrations (default: 1 step)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Show context information
			if !dryRun {
				dbDisplay := cfg.Database.URL
				if len(dbDisplay) > 40 {
					dbDisplay = dbDisplay[:40] + "..."
				}

				ctx := &ui.ContextView{
					Pairs: map[string]string{
						"Config":     configFile,
						"Migrations": cfg.MigrationsDir,
						"Database":   dbDisplay,
					},
				}
				fmt.Println(ctx.Render())
				fmt.Println()
			}

			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			steps := 1
			if len(args) > 0 {
				arg := args[0]
				steps, err = strconv.Atoi(arg)
				if err != nil || steps < 1 {
					printRollbackStepsError(arg)
					os.Exit(1)
				}
			}

			// Show warning and get confirmation for rollback
			if !dryRun {
				fmt.Println(ui.RenderTitle("Rollback Migrations"))
				fmt.Println()

				list := ui.NewList()
				list.AddError(fmt.Sprintf("Rollback %s", ui.FormatCount(steps, "migration", "migrations")))
				list.AddError("This will execute DOWN migrations")
				list.AddInfo("Database state will revert to previous revision")

				warning := ui.RenderWarningPanel(
					"Destructive Operation",
					list.String()+"\n"+
						ui.Warning("⚠ Make sure you have a backup\n")+
						ui.Help("Use --dry to preview SQL without executing"),
				)
				fmt.Println(warning)
				fmt.Println()

				if !ui.Confirm(fmt.Sprintf("Rollback %s?", ui.FormatCount(steps, "migration", "migrations")), false) {
					fmt.Println(ui.Dim("Rollback cancelled"))
					return nil
				}
				fmt.Println()
			}

			var opts []astroladb.MigrationOption
			if dryRun {
				opts = append(opts, astroladb.DryRunTo(os.Stdout))
			}

			// Execute rollback with timing
			start := time.Now()
			if err := client.MigrationRollback(steps, opts...); err != nil {
				return err
			}
			elapsed := time.Since(start)

			if !dryRun {
				// Show success with timing
				fmt.Println(ui.RenderSuccessPanel(
					"Rollback Complete",
					fmt.Sprintf("Rolled back %s in %s",
						ui.FormatCount(steps, "migration", "migrations"),
						ui.FormatDuration(elapsed),
					),
				))

				// Auto-commit migration files (optional with --commit flag)
				if commit {
					if err := autoCommitMigrations(cfg.MigrationsDir, "down"); err != nil {
						fmt.Fprintf(os.Stderr, "\n"+ui.Warning("Warning")+": %v\n", err)
					} else {
						fmt.Println()
						fmt.Println(ui.Success("✓") + " Migration files committed to git")
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry", false, "Print SQL without executing")
	cmd.Flags().BoolVar(&commit, "commit", false, "Auto-commit migration files to git")

	return cmd
}
