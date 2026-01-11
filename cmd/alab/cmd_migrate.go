package main

import (
	"fmt"
	"os"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// migrateCmd applies pending migrations.
func migrateCmd() *cobra.Command {
	var dryRun, force, confirmDestructive bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Pre-migration git checks
			if !force && !dryRun {
				check, err := git.CheckBeforeMigrate(cfg.MigrationsDir)
				if err != nil {
					return err
				}

				warnings := git.FormatPreMigrateWarnings(check)
				if warnings != "" {
					fmt.Fprint(os.Stderr, warnings)
				}

				// Modified migrations are errors - don't proceed
				if len(check.Errors) > 0 {
					fmt.Fprintln(os.Stderr, "Use --force to proceed anyway.")
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

			// Check for destructive operations - requires --confirm-destructive (not --force)
			if !dryRun {
				warnings, err := client.LintPendingMigrations()
				if err != nil {
					return err
				}
				if len(warnings) > 0 && !confirmDestructive {
					fmt.Fprintln(os.Stderr, alabcli.Warning("warning")+": destructive operations detected")
					fmt.Fprintln(os.Stderr, "")
					for _, w := range warnings {
						fmt.Fprintf(os.Stderr, "  %s %s\n", alabcli.Failed("•"), w)
					}
					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, alabcli.Note("note")+": these operations will permanently delete data")
					fmt.Fprintln(os.Stderr, alabcli.Help("help")+": run with --confirm-destructive to proceed")
					os.Exit(1)
				}
			}

			var opts []astroladb.MigrationOption

			if dryRun {
				opts = append(opts, astroladb.DryRunTo(os.Stdout))
			}
			if force {
				opts = append(opts, astroladb.Force())
			}

			if err := client.MigrationRun(opts...); err != nil {
				fmt.Fprint(os.Stderr, alabcli.FormatError(err))
				os.Exit(1)
			}

			if !dryRun {
				fmt.Println("Migrations applied successfully!")

				// Auto-commit migration files
				if err := autoCommitMigrations(cfg.MigrationsDir, "up"); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry", false, "Print SQL without executing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip safety warnings")
	cmd.Flags().BoolVar(&confirmDestructive, "confirm-destructive", false, "Confirm DROP operations")

	return cmd
}
