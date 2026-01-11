package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hlop3z/astroladb/internal/ui"
)

// HelpMessage represents a structured help message for error conditions.
type HelpMessage struct {
	Title string   // Error title (e.g., "No database configuration found")
	Lines []string // Help content lines
}

// helpMessages contains data-driven help messages for common error conditions.
// Using a data-driven approach reduces boilerplate and ensures consistency.
var helpMessages = map[string]HelpMessage{
	"missing_db_url": {
		Title: "No database configuration found",
		Lines: []string{
			"To fix this, do ONE of the following:",
			"",
			"  1. Set the DATABASE_URL environment variable:",
			"     export DATABASE_URL=\"postgres://user:pass@localhost:5432/mydb\"",
			"",
			"  2. Use the --database-url flag:",
			"     alab migrate --database-url \"postgres://user:pass@localhost:5432/mydb\"",
			"",
			"  3. Create alab.yaml with your config:",
			"     alab init",
			"     # Then edit alab.yaml to set database_url",
			"",
			"Supported URL formats:",
			"  PostgreSQL: postgres://user:pass@localhost:5432/dbname",
			"  SQLite:     ./mydb.db  or  /absolute/path/to/mydb.db",
		},
	},
	"schemas_dir_not_found": {
		Title: "Schemas directory not found",
		Lines: []string{
			"To fix this:",
			"",
			"  1. Initialize a new project:",
			"     alab init",
			"",
			"  2. Or create the directory manually:",
			"     mkdir -p %s",
			"",
			"  3. Or specify a different location:",
			"     alab check --schemas-dir /path/to/schemas",
		},
	},
	"migrations_dir_not_found": {
		Title: "Migrations directory not found",
		Lines: []string{
			"To fix this:",
			"",
			"  1. Initialize a new project:",
			"     alab init",
			"",
			"  2. Or create the directory manually:",
			"     mkdir -p %s",
		},
	},
	"migration_name_required": {
		Title: "Migration name is required",
		Lines: []string{
			"Usage:",
			"  alab new <name>",
			"",
			"Examples:",
			"  alab new create_users              # Creates 001_create_users.js",
			"  alab new add_email_to_users        # Creates 002_add_email_to_users.js",
			"  alab new --empty manual_changes    # Creates empty migration",
			"",
			"Tips:",
			"  - Use snake_case for migration names",
			"  - Start with a verb: create_, add_, remove_, update_",
			"  - Be descriptive: add_index_on_users_email",
		},
	},
	"rollback_steps_invalid": {
		Title: "Invalid steps argument",
		Lines: []string{
			"  Got: %s",
			"  Expected: A positive integer",
			"",
			"Usage:",
			"  alab rollback        # Rollback 1 migration (default)",
			"  alab rollback 3      # Rollback 3 migrations",
			"  alab rollback --dry    # Preview without executing",
		},
	},
	"schema_at_required": {
		Title: "Missing --at flag",
		Lines: []string{
			"The schema command requires a revision to display.",
			"",
			"Usage:",
			"  alab schema --at <revision>",
			"",
			"Examples:",
			"  alab schema --at 001                    # Schema after migration 001",
			"  alab schema --at 005 --format json      # Output as JSON",
			"  alab schema --at 003 --table auth.user  # Show specific table",
			"",
			"To see available revisions:",
			"  alab status",
		},
	},
}

// printHelp prints a help message by key.
// Supports optional format args for messages with placeholders.
func printHelp(key string, args ...any) {
	msg, ok := helpMessages[key]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown help message key: %s\n", key)
		return
	}

	fmt.Fprintln(os.Stderr, ui.Error("Error")+": "+msg.Title)
	fmt.Fprintln(os.Stderr)

	for _, line := range msg.Lines {
		// Apply format args if the line contains placeholders
		if strings.Contains(line, "%") && len(args) > 0 {
			fmt.Fprintf(os.Stderr, line+"\n", args...)
			// Consume the used arg
			if len(args) > 1 {
				args = args[1:]
			} else {
				args = nil
			}
		} else {
			fmt.Fprintln(os.Stderr, line)
		}
	}
}

// connectionHelpPostgres provides PostgreSQL-specific connection troubleshooting.
var connectionHelpPostgres = map[string][]string{
	"connection refused": {
		"- Is PostgreSQL running? Check: pg_isready -h localhost -p 5432",
		"- Verify the host and port in your URL",
	},
	"password": {
		"- Check your username and password",
		"- Verify pg_hba.conf allows your connection method",
	},
	"authentication": {
		"- Check your username and password",
		"- Verify pg_hba.conf allows your connection method",
	},
	"does not exist": {
		"- Database does not exist. Create it with:",
		"  createdb mydbname",
	},
	"timeout": {
		"- Connection timed out. Check network/firewall settings",
	},
	"default": {
		"- Verify the database server is running and accessible",
		"- Check your connection URL format:",
		"  postgres://user:pass@host:5432/dbname",
	},
}

// connectionHelpSQLite provides SQLite-specific connection troubleshooting.
var connectionHelpSQLite = map[string][]string{
	"no such file": {
		"- Database file path does not exist",
		"- Check the directory exists and is writable",
	},
	"unable to open": {
		"- Database file path does not exist",
		"- Check the directory exists and is writable",
	},
	"permission": {
		"- Permission denied. Check file/directory permissions",
	},
	"read-only": {
		"- Permission denied. Check file/directory permissions",
	},
	"default": {
		"- Verify the file path is correct",
		"- Check your database URL format:",
		"  ./path/to/database.db",
	},
}

// getConnectionHelp returns troubleshooting advice for a connection error.
func getConnectionHelp(dialect, causeStr string) []string {
	causeStr = strings.ToLower(causeStr)

	var helpMap map[string][]string
	switch dialect {
	case "postgres":
		helpMap = connectionHelpPostgres
	case "sqlite":
		helpMap = connectionHelpSQLite
	default:
		return []string{
			"- Verify the database server is running",
			"- Check your connection URL format",
			"- Ensure credentials are correct",
		}
	}

	// Check each key for a match in the cause string
	for key, help := range helpMap {
		if key != "default" && strings.Contains(causeStr, key) {
			return help
		}
	}

	// Return default help for this dialect
	if defaultHelp, ok := helpMap["default"]; ok {
		return defaultHelp
	}
	return nil
}
