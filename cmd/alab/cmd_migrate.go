package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// migrateCmd applies pending migrations.
func migrateCmd() *cobra.Command {
	var dryRun, force, confirmDestroy, commit, skipLock, verifySQL bool
	var lockTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending migrations",
		Long: `Apply pending migrations to the database.

Safety features: Git integration checks uncommitted changes, destructive operations require confirmation,
dry run mode previews SQL without executing, and distributed locking prevents concurrent migrations.
Use --commit to auto-commit after successful migration.`,
		Example: `  # Apply all pending migrations with confirmation
  alab migrate

  # Preview SQL that would be executed without applying
  alab migrate --dry

  # Apply migrations and auto-commit to git
  alab migrate --commit

  # Skip safety checks and confirmation prompts
  alab migrate --force

  # Apply migrations that include DROP operations
  alab migrate --confirm-destroy

  # Skip distributed locking (CI environments)
  alab migrate --skip-lock

  # Set custom lock timeout
  alab migrate --lock-timeout 60s

  # Verify SQL checksums before applying
  alab migrate --verify-sql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := mustConfig()

			// Show context information
			if !dryRun {
				printContextInfo(cfg)
			}

			// Pre-migration git checks
			if !force && !dryRun {
				check, err := git.CheckBeforeMigrate(cfg.MigrationsDir)
				if err != nil {
					return err
				}

				if len(check.Warnings) > 0 || len(check.Errors) > 0 {
					warnings := git.FormatPreMigrateWarnings(check)
					if warnings != "" {
						fmt.Fprint(os.Stderr, warnings)
					}
				}

				// Modified migrations are errors - don't proceed
				if len(check.Errors) > 0 {
					fmt.Fprintln(os.Stderr, ui.Help("hint")+": use --force to proceed anyway")
					os.Exit(1)
				}
			}

			client := mustClient()
			defer client.Close()

			// Get pending migrations to show count
			statuses, err := client.MigrationStatus()
			if err != nil {
				return err
			}

			pendingCount := 0
			for _, s := range statuses {
				if s.Status == ui.StatusPending {
					pendingCount++
				}
			}

			// If no pending migrations, show success and exit
			if pendingCount == 0 {
				ui.ShowSuccess(MsgMigrationsUpToDate, MsgNoUpToDate)
				return nil
			}

			// Show pending migrations count
			if !dryRun {
				fmt.Println(ui.RenderTitle(TitleApplyMigrations))
				fmt.Println()
				fmt.Printf("  %s\n\n", ui.Warning(ui.FormatCount(pendingCount, "pending migration", "pending migrations")))
			}

			// Check for destructive operations - requires --confirm-destroy (not --force)
			if !dryRun {
				warnings, err := client.LintPendingMigrations()
				if err != nil {
					return err
				}
				if len(warnings) > 0 && !confirmDestroy {
					list := ui.NewList()
					for _, w := range warnings {
						list.AddError(w)
					}

					fmt.Println(ui.RenderWarningPanel(
						TitleDestructiveOpsDetected,
						list.String()+"\n"+
							ui.Note(WarnDestructiveOps+"\n")+
							ui.Help(HelpUseConfirmDestroy),
					))
					os.Exit(1)
				}

				// Verify SQL determinism if requested
				if verifySQL {
					results, err := client.VerifySQLDeterminism()
					if err != nil {
						fmt.Fprintf(os.Stderr, "%s: %v\n", ui.Warning("Warning: SQL verification failed"), err)
					} else {
						for _, r := range results {
							if !r.Match {
								fmt.Fprintf(os.Stderr, "  %s SQL mismatch: %s %s\n", ui.Warning("WARN"), r.Revision, r.Name)
							}
						}
					}
				}

				// Ask for confirmation (unless dry run or force)
				if !force {
					if !confirmOrCancel(
						fmt.Sprintf(PromptApplyMigrations, ui.FormatCount(pendingCount, "migration", "migrations")),
						true, MsgMigrationCancelled,
					) {
						return nil
					}
				}
			}

			opts := buildMigrationOpts(dryRun, skipLock, lockTimeout)
			if force {
				opts = append(opts, astroladb.Force())
			}

			// Apply migrations with timing
			start := time.Now()
			if err := client.MigrationRun(opts...); err != nil {
				fmt.Fprint(os.Stderr, ui.FormatError(err))
				os.Exit(1)
			}
			elapsed := time.Since(start)

			if !dryRun {
				// Update lock file
				updateLockFile(cfg.MigrationsDir)

				// Show success with timing
				ui.ShowSuccess(
					TitleMigrationsApplied,
					fmt.Sprintf("Applied %s in %s",
						ui.FormatCount(pendingCount, "migration", "migrations"),
						ui.FormatDuration(elapsed),
					),
				)

				autoCommitIfRequested(commit, cfg.MigrationsDir, "up")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry", false, FlagDescDryRun)
	cmd.Flags().BoolVar(&force, "force", false, FlagDescForce)
	cmd.Flags().BoolVar(&confirmDestroy, "confirm-destroy", false, FlagDescConfirmDestroy)
	cmd.Flags().BoolVar(&commit, "commit", false, FlagDescCommit)
	cmd.Flags().BoolVar(&skipLock, "skip-lock", false, FlagDescSkipLock)
	cmd.Flags().DurationVar(&lockTimeout, "lock-timeout", 0, FlagDescLockTimeout)
	cmd.Flags().BoolVar(&verifySQL, "verify-sql", false, FlagDescVerifySQL)

	setupCommandHelp(cmd)
	return cmd
}
