package main

import (
	"errors"
	"fmt"
	"os"

	alabcli "github.com/hlop3z/astroladb/internal/cli"
	"github.com/hlop3z/astroladb/internal/git"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// migrateCmd applies pending migrations.
func migrateCmd() *cobra.Command {
	var dryRun, force, confirmDestructive, ignoreChainError, noProgress bool
	var target string
	var steps int

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
			if target != "" {
				opts = append(opts, astroladb.Target(target))
			}
			if steps > 0 {
				opts = append(opts, astroladb.Steps(steps))
			}

			if err := client.MigrationRun(opts...); err != nil {
				// Check if it's a chain error and format nicely
				var chainErr *astroladb.ChainError
				if errors.As(err, &chainErr) {
					if !ignoreChainError {
						fmt.Fprintln(os.Stderr, alabcli.Error("error")+": migration chain verification failed")
						fmt.Fprintln(os.Stderr, "")
						fmt.Fprintln(os.Stderr, chainErr.Error())
						fmt.Fprintln(os.Stderr, "")
						fmt.Fprintln(os.Stderr, alabcli.Note("note")+": chain errors indicate migrations may have been modified after being applied")
						fmt.Fprintln(os.Stderr, alabcli.Help("help")+": run with --ignore-chain-error to proceed (dangerous)")
						os.Exit(1)
					}
				}
				// Format other errors using cli package
				fmt.Fprint(os.Stderr, alabcli.FormatError(err))
				os.Exit(1)
			}

			if !dryRun {
				fmt.Println("Migrations applied successfully!")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print SQL without executing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip non-destructive warnings (git checks, etc.)")
	cmd.Flags().BoolVar(&confirmDestructive, "confirm-destructive", false, "Confirm execution of DROP TABLE/COLUMN operations")
	cmd.Flags().BoolVar(&ignoreChainError, "ignore-chain-error", false, "Proceed despite migration chain verification failures")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress indicators")
	cmd.Flags().StringVar(&target, "target", "", "Migrate to specific version")
	cmd.Flags().IntVar(&steps, "steps", 0, "Apply at most N migrations")

	return cmd
}
