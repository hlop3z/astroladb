package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// diffCmd shows the diff between schema and database.
func diffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show diff between schema files and database",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			ops, err := client.SchemaDiff()
			if err != nil {
				return err
			}

			if len(ops) == 0 {
				fmt.Println("No changes detected.")
				return nil
			}

			fmt.Printf("Found %d operations:\n", len(ops))
			for i, op := range ops {
				table := op.Table()
				if table != "" {
					fmt.Printf("  %d. %s on %s\n", i+1, op.Type(), table)
				} else {
					fmt.Printf("  %d. %s\n", i+1, op.Type())
				}
			}
			return nil
		},
	}
}
