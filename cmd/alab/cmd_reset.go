package main

import (
	"fmt"
	"os"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
)

// resetCmd drops all tables and re-runs migrations.
func resetCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Drop all tables and re-run migrations (dev only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			// Show warning and get confirmation
			if !force {
				list := ui.NewList()
				list.AddError("Drop ALL tables in the database")
				list.AddError("Delete ALL data permanently")
				list.AddInfo("Re-run all migrations from scratch")

				warning := ui.RenderWarningPanel(
					"Destructive Operation",
					list.String()+"\n"+
						ui.Warning("⚠ This operation cannot be undone\n")+
						ui.Help("This command is intended for development only"),
				)
				fmt.Println(warning)
				fmt.Println()

				if !ui.Confirm("Continue with database reset?", false) {
					fmt.Println(ui.Dim("Reset cancelled"))
					return nil
				}
				fmt.Println()
			}

			// Drop all tables by executing raw SQL
			db := client.DB()
			dialect := client.Dialect()

			var dropSQL string
			var tableCount int
			switch dialect {
			case "postgres":
				dropSQL = "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
			case "sqlite":
				// SQLite: get all tables and drop them
				rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
				if err != nil {
					return fmt.Errorf("failed to list tables: %w", err)
				}
				defer rows.Close()

				var tables []string
				for rows.Next() {
					var name string
					if err := rows.Scan(&name); err != nil {
						return err
					}
					tables = append(tables, name)
				}
				tableCount = len(tables)

				for _, table := range tables {
					if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
						return fmt.Errorf("failed to drop table %s: %w", table, err)
					}
				}
				fmt.Println(ui.Success("✓") + " Dropped " + ui.FormatCount(tableCount, "table", "tables"))
				fmt.Println()

				// Run migrations
				if err := client.MigrationRun(); err != nil {
					return err
				}

				// Show success
				view := ui.NewSuccessView(
					"Database Reset Complete",
					"All tables dropped and migrations re-applied",
				)
				fmt.Println(view.Render())
				return nil
			default:
				return fmt.Errorf("unsupported dialect for db:reset: %s", dialect)
			}

			if _, err := db.Exec(dropSQL); err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}
			fmt.Println(ui.Success("✓") + " Dropped all tables")
			fmt.Println()

			// Run migrations
			if err := client.MigrationRun(); err != nil {
				return err
			}

			// Show success
			view := ui.NewSuccessView(
				"Database Reset Complete",
				"All tables dropped and migrations re-applied",
			)
			fmt.Println(view.Render())
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}
