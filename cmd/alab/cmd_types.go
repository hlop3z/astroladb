package main

import (
	"fmt"

	"github.com/hlop3z/astroladb/internal/ui"
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

			// Show success with list of regenerated files
			list := ui.NewList()
			list.AddSuccess("types/base.d.ts")
			list.AddSuccess("types/schema.d.ts")
			list.AddSuccess("types/migration.d.ts")

			view := ui.NewSuccessView(
				"Type Definitions Regenerated",
				list.String()+"\n"+
					ui.Help("These files are auto-generated - do not edit manually"),
			)
			fmt.Println(view.Render())
			return nil
		},
	}
}
