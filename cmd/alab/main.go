// Package main provides the CLI for the Alab database migration tool.
// Alab is a language-agnostic database migration tool that manages schema
// evolution via migrations, using a JavaScript DSL for schema definitions.
//
// Usage:
//
//	alab init                    # Create schemas/ and migrations/ dirs
//	alab status                  # Browse schema, history, drift (TUI)
//	alab schema --at <rev>       # Show schema at a specific migration revision
//	alab export --format X       # Export schema (openapi, jsonschema, typescript, go)
//	alab new <name>              # Create migration (auto-generates if schema has changes)
//	alab migrate                 # Apply pending migrations
//	alab status                  # Show applied/pending migrations
//	alab rollback [steps]        # Rollback (default: 1 step)
//	alab reset                   # Drop all tables, re-run migrations
package main

import (
	"fmt"
	"log/slog"
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
	logLevel    string
)

// initLogger configures the global slog logger based on the log level.
// Valid levels: debug, info, warn, error
// Empty string or invalid value defaults to WARN (effectively no logging for most operations)
func initLogger(level string) {
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	case "":
		// Default: WARN level (minimal logging)
		logLevel = slog.LevelWarn
	default:
		// Invalid level, default to WARN
		logLevel = slog.LevelWarn
	}

	// Create handler with the specified level
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})

	// Set as default logger
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// customHelp displays a styled help message for the root command.
func customHelp(cmd *cobra.Command) {
	categories := []CommandCategory{
		{
			Title: "Development",
			Commands: []CommandInfo{
				{"init", "Initialize project structure (schemas/, migrations/, types/)"},
				{"table", "Create a new table schema file"},
				{"http", "Start local server for live API documentation"},
				{"export", "Export schema (openapi, graphql, typescript, go, python, rust)"},
			},
		},
		{
			Title: "Migrations",
			Commands: []CommandInfo{
				{"new", "Create migration (auto-generates from schema changes)"},
				{"migrate", "Apply pending migrations"},
				{"rollback", "Rollback migrations (default: 1 step)"},
				{"reset", "Drop all tables and re-run migrations (dev only)"},
			},
		},
		{
			Title: "Toolkit",
			Commands: []CommandInfo{
				{"schema", "Show schema at a specific migration revision"},
				{"status", "Show schema, history, verify, and drift (TUI)"},
				{"types", "Regenerate TypeScript definitions for IDE"},
				{"meta", "Export schema metadata to JSON file"},
			},
		},
	}

	flags := []struct{ flag, desc string }{
		{"-c, --config", "Path to config file (default: alab.yaml)"},
		{"-d, --database-url", "Database connection URL"},
		{"--log", "Log level: debug, info, warn, error (default: warn)"},
		{"-h, --help", "Show help information"},
		{"-v, --version", "Show version information"},
	}

	renderCategoryHelp(
		MainTitle,
		MainSummary,
		categories,
		flags,
	)
}

func main() {
	// Initialize logger with a pre-command hook that parses flags
	// Default to no logging (equivalent to WARN level)
	initLogger("")

	rootCmd := &cobra.Command{
		Use:     "alab",
		Short:   "Language-agnostic database migration tool",
		Long:    `Alab is a language-agnostic database migration tool that manages schema evolution via migrations, using a JavaScript DSL for schema definitions.`,
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Reinitialize logger after flags are parsed
			initLogger(logLevel)
		},
	}

	// Set custom help function
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		customHelp(cmd)
	})

	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&databaseURL, "database-url", "d", "", "Database connection URL")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "alab.yaml", "Path to config file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "", "Log level (debug, info, warn, error)")

	// Add subcommands
	rootCmd.AddCommand(
		initCmd(),
		tableCmd(),
		typesCmd(),
		exportCmd(),
		metaCmd(),
		schemaCmd(),
		newCmd(),
		migrateCmd(),
		statusCmd(),
		rollbackCmd(),
		resetCmd(),
		httpCmd(),
		lockCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
