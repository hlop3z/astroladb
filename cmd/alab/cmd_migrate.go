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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Show context information
			if !dryRun {
				// Mask database URL for security
				dbDisplay := MaskDatabaseURL(cfg.Database.URL)

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

			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
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
					if !ui.Confirm(
						fmt.Sprintf(PromptApplyMigrations, ui.FormatCount(pendingCount, "migration", "migrations")),
						true,
					) {
						fmt.Println(ui.Dim(MsgMigrationCancelled))
						return nil
					}
					fmt.Println()
				}
			}

			var opts []astroladb.MigrationOption

			if dryRun {
				opts = append(opts, astroladb.DryRunTo(os.Stdout))
			}
			if force {
				opts = append(opts, astroladb.Force())
			}
			if skipLock {
				opts = append(opts, astroladb.SkipLock())
			}
			if lockTimeout > 0 {
				opts = append(opts, astroladb.LockTimeout(lockTimeout))
			}

			// Apply migrations with timing
			start := time.Now()
			if err := client.MigrationRun(opts...); err != nil {
				fmt.Fprint(os.Stderr, ui.FormatError(err))
				os.Exit(1)
			}
			elapsed := time.Since(start)

			if !dryRun {
				// Show success with timing
				ui.ShowSuccess(
					TitleMigrationsApplied,
					fmt.Sprintf("Applied %s in %s",
						ui.FormatCount(pendingCount, "migration", "migrations"),
						ui.FormatDuration(elapsed),
					),
				)

				// Auto-commit migration files (optional with --commit flag)
				if commit {
					if err := autoCommitMigrations(cfg.MigrationsDir, "up"); err != nil {
						fmt.Fprintf(os.Stderr, "\n"+ui.Warning("Warning")+": %v\n", err)
					} else {
						fmt.Println()
						fmt.Println(ui.Success(MsgCommittedToGit))
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry", false, "Print SQL without executing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip safety warnings")
	cmd.Flags().BoolVar(&confirmDestroy, "confirm-destroy", false, "Confirm DROP operations")
	cmd.Flags().BoolVar(&commit, "commit", false, "Auto-commit migration files to git")
	cmd.Flags().BoolVar(&skipLock, "skip-lock", false, "Skip distributed locking (use in CI)")
	cmd.Flags().DurationVar(&lockTimeout, "lock-timeout", 0, "Lock acquisition timeout (default 30s)")
	cmd.Flags().BoolVar(&verifySQL, "verify-sql", false, "Verify SQL checksums before applying")

	setupCommandHelp(cmd)
	return cmd
}
