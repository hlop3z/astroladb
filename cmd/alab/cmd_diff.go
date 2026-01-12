package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
)

// diffCmd shows the diff between schema and database.
func diffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff between schema files and database",
		Long: `Shows the differences between your schema files and the current database state.

This command analyzes your schema definitions and compares them against the actual
database structure. It identifies all necessary operations (create, alter, drop) that
would need to be performed to bring the database in sync with your schema files.

The diff output helps you understand what changes need to be migrated before you
create a new migration file. No database changes are made by this command - it only
reports the differences.`,
		Example: `  # Show schema differences
  alab diff

  # Check for differences after modifying schema files
  alab diff

  # Verify database is in sync with schema
  alab diff`,
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

			// No changes case
			if len(ops) == 0 {
				fmt.Println(ui.RenderSuccessPanel(
					"Schema in sync",
					"No changes detected between schema files and database",
				))
				return nil
			}

			// Changes detected case
			fmt.Println(ui.RenderTitle("Schema Diff"))
			fmt.Println()
			fmt.Printf("  %s\n\n", ui.Warning(ui.FormatCount(len(ops), "operation", "operations")))

			// Create styled table
			table := ui.NewStyledTable("#", "OPERATION", "TABLE", "DETAILS")
			for i, op := range ops {
				opType := op.Type().String()
				tableName := op.Table()
				if tableName == "" {
					tableName = ui.Dim("-")
				}

				// Color-code operation types
				var styledType string
				switch {
				case strings.HasPrefix(opType, "Create"):
					styledType = ui.Success(opType)
				case strings.HasPrefix(opType, "Drop"):
					styledType = ui.Error(opType)
				case strings.HasPrefix(opType, "Alter"), strings.HasPrefix(opType, "Modify"):
					styledType = ui.Warning(opType)
				default:
					styledType = ui.Info(opType)
				}

				details := getOperationDetails(op)

				table.AddRow(
					ui.Dim(fmt.Sprintf("%d", i+1)),
					styledType,
					tableName,
					details,
				)
			}

			fmt.Print(table.String())
			fmt.Println()
			fmt.Println(ui.Help("Run 'alab new <name>' to create a migration for these changes"))
			return nil
		},
	}

	setupCommandHelp(cmd)
	return cmd
}

// getOperationDetails extracts human-readable details from an operation
func getOperationDetails(op interface{}) string {
	// Extract column names, index names, or other relevant details
	// based on operation type (may need type assertions)
	// Return "-" if no details available
	return ui.Dim("-") // Placeholder - enhance based on Operation interface
}
