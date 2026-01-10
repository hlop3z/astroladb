package main

import (
	"fmt"
	"os"
	"strings"

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

			if !force {
				fmt.Print("This will DROP ALL TABLES and re-run migrations. Are you sure? [y/N] ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Drop all tables by executing raw SQL
			db := client.DB()
			dialect := client.Dialect()

			var dropSQL string
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

				for _, table := range tables {
					if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
						return fmt.Errorf("failed to drop table %s: %w", table, err)
					}
				}
				fmt.Println("Dropped all tables.")

				// Run migrations
				if err := client.MigrationRun(); err != nil {
					return err
				}
				fmt.Println("Database reset complete!")
				return nil
			default:
				return fmt.Errorf("unsupported dialect for db:reset: %s", dialect)
			}

			if _, err := db.Exec(dropSQL); err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}
			fmt.Println("Dropped all tables.")

			// Run migrations
			if err := client.MigrationRun(); err != nil {
				return err
			}

			fmt.Println("Database reset complete!")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}
