// Package main provides the CLI for the Alab database migration tool.
// Alab is a language-agnostic database migration tool that manages schema
// evolution via migrations, using a JavaScript DSL for schema definitions.
//
// Usage:
//
//	alab init                    # Create schemas/ and migrations/ dirs
//	alab check                   # Validate all schema files
//	alab diff                    # Show diff between schema and DB
//	alab export --format X       # Export schema (openapi, jsonschema, typescript, go)
//	alab schema --at <rev>       # Show schema at a specific migration revision
//	alab new <name>              # Create migration (auto-generates if schema has changes)
//	alab migrate                 # Apply pending migrations
//	alab status                  # Show applied/pending migrations
//	alab rollback [steps]        # Rollback (default: 1 step)
//	alab reset                   # Drop all tables, re-run migrations
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// Database drivers
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// version is set via ldflags during build: -ldflags="-X main.version=v1.0.0"
var version = "dev"

// Global flags
var (
	databaseURL   string
	schemasDir    string
	migrationsDir string
	configFile    string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "alab",
		Short:   "Language-agnostic database migration tool",
		Long:    `Alab is a language-agnostic database migration tool that manages schema evolution via migrations, using a JavaScript DSL for schema definitions.`,
		Version: version,
	}

	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&databaseURL, "database-url", "d", "", "Database connection URL")
	rootCmd.PersistentFlags().StringVar(&schemasDir, "schemas-dir", "./schemas", "Path to schemas directory")
	rootCmd.PersistentFlags().StringVar(&migrationsDir, "migrations-dir", "./migrations", "Path to migrations directory")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "alab.yaml", "Path to config file")

	// Add subcommands
	rootCmd.AddCommand(
		initCmd(),
		typesCmd(),
		checkCmd(),
		diffCmd(),
		exportCmd(),
		schemaCmd(),
		newCmd(),
		migrateCmd(),
		statusCmd(),
		historyCmd(),
		rollbackCmd(),
		resetCmd(),
		verifyCmd(),
		rebuildCacheCmd(),
		httpCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
