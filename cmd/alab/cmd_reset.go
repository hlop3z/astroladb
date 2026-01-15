package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/ui"
)

// resetCmd drops all tables and re-runs migrations.
func resetCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Drop all tables and re-run migrations (dev only)",
		Long: `Drop all tables and re-run migrations from scratch.

⚠️  DESTRUCTIVE - Permanently deletes ALL data! For development only.

Drops all tables, re-runs migrations, leaves clean schema with no data. Requires confirmation unless --force is used.`,
		Example: `  # Reset database with confirmation prompt (recommended)
  alab reset

  # Reset database without confirmation (dangerous)
  alab reset --force

  # Typical development workflow - reset and seed
  alab reset && alab seed`,
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
				list.AddError(WarnDropAllTables)
				list.AddError(WarnDataPermanentDelete)
				list.AddInfo("Re-run all migrations from scratch")

				warning := ui.RenderWarningPanel(
					TitleDestructiveOperation,
					list.String()+"\n"+
						ui.Warning("⚠ "+HelpCannotUndo+"\n")+
						ui.Help(HelpDevOnly),
				)
				fmt.Println(warning)
				fmt.Println()

				if !ui.Confirm(PromptContinueReset, false) {
					fmt.Println(ui.Dim(MsgResetCancelled))
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
				fmt.Println(ui.Success("Dropped " + ui.FormatCount(tableCount, "table", "tables")))
				fmt.Println()

				// Run migrations
				if err := client.MigrationRun(); err != nil {
					return err
				}

				// Show success
				ui.ShowSuccess(TitleDatabaseResetComplete, MsgAllTablesDropped)
				return nil
			default:
				return fmt.Errorf("unsupported dialect for db:reset: %s", dialect)
			}

			if _, err := db.Exec(dropSQL); err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}
			fmt.Println(ui.Success("Dropped all tables"))
			fmt.Println()

			// Run migrations
			if err := client.MigrationRun(); err != nil {
				return err
			}

			// Show success
			ui.ShowSuccess(TitleDatabaseResetComplete, MsgAllTablesDropped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	setupCommandHelp(cmd)
	return cmd
}
