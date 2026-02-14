package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/ui"
)

// rollbackCmd rolls back migrations.
func rollbackCmd() *cobra.Command {
	var dryRun, commit, skipLock bool
	var lockTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "rollback [steps]",
		Short: "Rollback migrations (default: 1 step)",
		Long: `Rollback applied migrations by executing their DOWN scripts.

This command reverts database changes by running the down migration scripts
for the specified number of migrations. By default, it rolls back one migration.
The rollback process removes the migration entries from the migrations table
and executes the corresponding down SQL scripts to undo schema changes.

Use --dry flag to preview the SQL that would be executed without making
actual changes to the database. This is useful for verifying the rollback
operations before executing them.

Use --commit flag to automatically commit the migration state changes to git
after a successful rollback.`,
		Example: `  # Rollback the last migration
  alab rollback

  # Rollback the last 3 migrations
  alab rollback 3

  # Preview rollback SQL without executing (dry run)
  alab rollback 2 --dry

  # Skip distributed locking (CI environments)
  alab rollback --skip-lock`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := mustConfig()

			// Show context information
			if !dryRun {
				printContextInfo(cfg)
			}

			client := mustClient()
			defer client.Close()

			steps := 1
			if len(args) > 0 {
				arg := args[0]
				var err error
				steps, err = strconv.Atoi(arg)
				if err != nil || steps < 1 {
					printRollbackStepsError(arg)
					os.Exit(1)
				}
			}

			// Show warning and get confirmation for rollback
			if !dryRun {
				fmt.Println(ui.RenderTitle(Msg.Migration.Rollback.Title))
				fmt.Println()

				list := ui.NewList()
				list.AddError(fmt.Sprintf("Rollback %s", ui.FormatCount(steps, "migration", "migrations")))
				list.AddError("This will execute DOWN migrations")
				list.AddInfo("Database state will revert to previous revision")

				warning := ui.RenderWarningPanel(
					"Destructive Operation",
					list.String()+"\n"+
						ui.Warning("Make sure you have a backup\n")+
						ui.Help(Msg.Migration.Rollback.DryHelp),
				)
				fmt.Println(warning)
				fmt.Println()

				if !confirmOrCancel(fmt.Sprintf(Msg.Migration.Rollback.Prompt, ui.FormatCount(steps, "migration", "migrations")), false, Msg.Migration.Rollback.Cancelled) {
					return nil
				}
			}

			opts := buildMigrationOpts(dryRun, skipLock, lockTimeout)

			// Execute rollback with timing
			start := time.Now()
			if err := client.MigrationRollback(steps, opts...); err != nil {
				return err
			}
			elapsed := time.Since(start)

			if !dryRun {
				// Show success with timing
				ui.ShowSuccess(
					Msg.Migration.Rollback.Complete,
					fmt.Sprintf("Rolled back %s in %s",
						ui.FormatCount(steps, "migration", "migrations"),
						ui.FormatDuration(elapsed),
					),
				)

				autoCommitIfRequested(commit, cfg.MigrationsDir, "down")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry", false, FlagDescDryRun)
	cmd.Flags().BoolVar(&commit, "commit", false, FlagDescCommit)
	cmd.Flags().BoolVar(&skipLock, "skip-lock", false, FlagDescSkipLock)
	cmd.Flags().DurationVar(&lockTimeout, "lock-timeout", 0, FlagDescLockTimeout)

	setupCommandHelp(cmd)
	return cmd
}
