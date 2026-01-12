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
	databaseURL string
	configFile  string
)

// customHelp displays a styled help message for the root command.
func customHelp(cmd *cobra.Command) {
	categories := []CommandCategory{
		{
			Title: "Setup",
			Commands: []CommandInfo{
				{"init", "Initialize project structure (schemas/, migrations/, types/)"},
				{"table", "Create a new table schema file"},
			},
		},
		{
			Title: "Schema Management",
			Commands: []CommandInfo{
				{"check", "Validate schema files and detect issues"},
				{"diff", "Show differences between schema and database"},
				{"schema", "Show schema at a specific migration revision"},
			},
		},
		{
			Title: "Migrations",
			Commands: []CommandInfo{
				{"new", "Create migration (auto-generates from schema changes)"},
				{"migrate", "Apply pending migrations"},
				{"rollback", "Rollback migrations (default: 1 step)"},
				{"reset", "Drop all tables and re-run migrations (dev only)"},
				{"status", "Show applied/pending migrations"},
				{"history", "Show applied migrations with details"},
			},
		},
		{
			Title: "Development",
			Commands: []CommandInfo{
				{"http", "Start local server for live API documentation"},
				{"export", "Export schema (openapi, graphql, typescript, go, python, rust)"},
				{"meta", "Export schema metadata to JSON file"},
				{"types", "Regenerate TypeScript definitions for IDE"},
			},
		},
		{
			Title: "Verification",
			Commands: []CommandInfo{
				{"verify", "Verify migration chain integrity and git status"},
			},
		},
	}

	flags := []struct{ flag, desc string }{
		{"-c, --config", "Path to config file (default: alab.yaml)"},
		{"-d, --database-url", "Database connection URL"},
		{"-h, --help", "Show help information"},
		{"-v, --version", "Show version information"},
	}

	renderCategoryHelp(
		"⏳ Alab - Database Migration Tool",
		"★  Language-agnostic database migration tool using JavaScript DSL",
		categories,
		flags,
	)
}

func main() {
	rootCmd := &cobra.Command{
		Use:     "alab",
		Short:   "Language-agnostic database migration tool",
		Long:    `Alab is a language-agnostic database migration tool that manages schema evolution via migrations, using a JavaScript DSL for schema definitions.`,
		Version: version,
	}

	// Set custom help function
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		customHelp(cmd)
	})

	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&databaseURL, "database-url", "d", "", "Database connection URL")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "alab.yaml", "Path to config file")

	// Add subcommands
	rootCmd.AddCommand(
		initCmd(),
		tableCmd(),
		typesCmd(),
		checkCmd(),
		diffCmd(),
		exportCmd(),
		metaCmd(),
		schemaCmd(),
		newCmd(),
		migrateCmd(),
		statusCmd(),
		historyCmd(),
		rollbackCmd(),
		resetCmd(),
		verifyCmd(),
		httpCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
