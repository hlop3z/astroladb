package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// rollbackCmd rolls back migrations.
func rollbackCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "rollback [steps]",
		Short: "Rollback migrations (default: 1 step)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			var opts []astroladb.MigrationOption
			if dryRun {
				opts = append(opts, astroladb.DryRunTo(os.Stdout))
			}

			if err := client.MigrationRollback(steps, opts...); err != nil {
				return err
			}

			if !dryRun {
				fmt.Printf("Rolled back %d migration(s) successfully!\n", steps)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry", false, "Print SQL without executing")

	return cmd
}
