// Package main provides the CLI for Astroladb.
// Astroladb is a schema-centric tooling language that generates database migrations,
// OpenAPI specs, GraphQL schemas, and type definitions from a single JavaScript schema.
//
// Usage:
//
//	alab init                    # Initialize project structure
//	alab table <ns> <name>       # Create a new table schema file
//	alab new <name>              # Generate migration from schema changes
//	alab migrate                 # Apply pending migrations
//	alab rollback [n]            # Rollback last n migrations
//	alab status                  # Show migration status
//	alab export -f <format>      # Export schema (openapi, graphql, typescript, go, python, rust)
//	alab live                    # Start live documentation server
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
				{"live", "Start local server for live API documentation"},
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
				{"check", "Validate schemas or lint migrations"},
			},
		},
		{
			Title: "Generators",
			Commands: []CommandInfo{
				{"gen run", "Run a code generator against the schema"},
				{"gen add", "Download a shared generator from a URL"},
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
		Short:   "Schema-first database migrations",
		Long:    `Astroladb is a schema-centric tooling language that generates database migrations, OpenAPI specs, GraphQL schemas, and type definitions from a single JavaScript schema.`,
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
		genCmd(),
		checkCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
