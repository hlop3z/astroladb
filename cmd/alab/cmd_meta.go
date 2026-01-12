package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
)

// metaCmd exports schema metadata to a JSON file.
func metaCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "meta",
		Short: "Export schema metadata to JSON file",
		Long: `Export complete schema metadata to a JSON file for use by external tools.

The metadata includes:
- Table definitions (columns, types, constraints)
- Relationships (foreign keys, many-to-many, polymorphic)
- Indexes and unique constraints
- Table-level metadata (searchable, filterable, sort_by, auditable)

External tools can use this metadata to generate ORMs, query builders,
GraphQL schemas, or other code without needing to evaluate JavaScript schema files.`,
		Example: `  # Export metadata to default location (alab.meta.json)
  alab meta

  # Export to custom file
  alab meta --output ./metadata/schema.json

  # Short form
  alab meta -o schema-metadata.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newSchemaOnlyClient()
			if err != nil {
				return err
			}
			defer client.Close()

			// Resolve absolute path for output file
			absPath, err := filepath.Abs(output)
			if err != nil {
				return fmt.Errorf("failed to resolve output path: %w", err)
			}

			// Save metadata to file
			if err := client.SaveMetadataToFile(absPath); err != nil {
				return fmt.Errorf("failed to save metadata: %w", err)
			}

			// Get file size for display
			info, err := os.Stat(absPath)
			if err != nil {
				return fmt.Errorf("failed to read metadata file: %w", err)
			}

			// Show success message
			view := ui.NewSuccessView(
				"Metadata Exported",
				fmt.Sprintf("Saved to: %s\nSize: %d bytes\n\n%s",
					absPath,
					info.Size(),
					ui.Help("This file contains all schema metadata for external tools to generate code, ORMs, queries, etc.")),
			)
			fmt.Println(view.Render())

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "alab.meta.json", "Output file path")

	setupCommandHelp(cmd)
	return cmd
}
