package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// typesCmd regenerates the TypeScript definition files for IDE autocomplete.
func typesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "Regenerate TypeScript definitions for IDE autocomplete",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := writeTypeDefinitions(); err != nil {
				return fmt.Errorf("failed to write type definitions: %w", err)
			}
			fmt.Println("Regenerated types/*.d.ts and jsconfig.json")
			fmt.Println("These files are auto-generated - do not edit manually.")
			return nil
		},
	}
}
