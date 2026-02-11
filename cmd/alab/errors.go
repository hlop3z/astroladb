package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// printMissingDatabaseURLError prints a helpful error message when no database URL is configured.
func printMissingDatabaseURLError() {
	printHelp("missing_db_url")
}

// printConnectionError prints a helpful error message for database connection failures.
func printConnectionError(connErr *astroladb.ConnectionError) {
	fmt.Fprintln(os.Stderr, "Error: Failed to connect to database")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  Dialect: %s\n", connErr.Dialect)
	fmt.Fprintf(os.Stderr, "  URL:     %s\n", connErr.URL)
	fmt.Fprintf(os.Stderr, "  Cause:   %v\n", connErr.Cause)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Troubleshooting:")

	causeStr := strings.ToLower(connErr.Cause.Error())

	switch connErr.Dialect {
	case "postgres":
		if strings.Contains(causeStr, "connection refused") {
			fmt.Fprintln(os.Stderr, "  - Is PostgreSQL running? Check: pg_isready -h localhost -p 5432")
			fmt.Fprintln(os.Stderr, "  - Verify the host and port in your URL")
		} else if strings.Contains(causeStr, "password") || strings.Contains(causeStr, "authentication") {
			fmt.Fprintln(os.Stderr, "  - Check your username and password")
			fmt.Fprintln(os.Stderr, "  - Verify pg_hba.conf allows your connection method")
		} else if strings.Contains(causeStr, "does not exist") {
			fmt.Fprintln(os.Stderr, "  - Database does not exist. Create it with:")
			fmt.Fprintln(os.Stderr, "    createdb mydbname")
		} else if strings.Contains(causeStr, "timeout") {
			fmt.Fprintln(os.Stderr, "  - Connection timed out. Check network/firewall settings")
		} else {
			fmt.Fprintln(os.Stderr, "  - Verify the database server is running and accessible")
			fmt.Fprintln(os.Stderr, "  - Check your connection URL format:")
			fmt.Fprintln(os.Stderr, "    postgres://user:pass@host:5432/dbname")
		}

	case "sqlite":
		if strings.Contains(causeStr, "no such file") || strings.Contains(causeStr, "unable to open") {
			fmt.Fprintln(os.Stderr, "  - Database file path does not exist")
			fmt.Fprintln(os.Stderr, "  - Check the directory exists and is writable")
		} else if strings.Contains(causeStr, "permission") || strings.Contains(causeStr, "read-only") {
			fmt.Fprintln(os.Stderr, "  - Permission denied. Check file/directory permissions")
		} else {
			fmt.Fprintln(os.Stderr, "  - Verify the file path is correct")
			fmt.Fprintln(os.Stderr, "  - Check your database URL format:")
			fmt.Fprintln(os.Stderr, "    ./path/to/database.db")
		}

	default:
		fmt.Fprintln(os.Stderr, "  - Verify the database server is running")
		fmt.Fprintln(os.Stderr, "  - Check your connection URL format")
		fmt.Fprintln(os.Stderr, "  - Ensure credentials are correct")
	}
}

// printRollbackStepsError prints a helpful error message for invalid rollback steps.
func printRollbackStepsError(input string) {
	fmt.Fprintln(os.Stderr, "Error: Invalid steps argument")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  Got: %s\n", input)
	fmt.Fprintln(os.Stderr, "  Expected: A positive integer")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  alab rollback        # Rollback 1 migration (default)")
	fmt.Fprintln(os.Stderr, "  alab rollback 3      # Rollback 3 migrations")
	fmt.Fprintln(os.Stderr, "  alab rollback --dry    # Preview without executing")
}

// printSchemaAtError prints a helpful error message for the schema command.
func printSchemaAtError() {
	fmt.Fprintln(os.Stderr, "Error: Missing --at flag")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "The schema command requires a revision to display.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  alab schema --at <revision>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  alab schema --at 001                    # Schema after migration 001")
	fmt.Fprintln(os.Stderr, "  alab schema --at 005 --format json      # Output as JSON")
	fmt.Fprintln(os.Stderr, "  alab schema --at 003 --format sql       # Generate SQL CREATE statements")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "To see available revisions:")
	fmt.Fprintln(os.Stderr, "  alab status")
}

// handleClientError checks for common error types and prints helpful messages.
// Returns true if the error was handled (and a message was printed), false otherwise.
func handleClientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for missing database URL
	if errors.Is(err, astroladb.ErrMissingDatabaseURL) {
		printMissingDatabaseURLError()
		return true
	}

	// Check for connection errors
	var connErr *astroladb.ConnectionError
	if errors.As(err, &connErr) {
		printConnectionError(connErr)
		return true
	}

	// Check for schema errors
	var schemaErr *astroladb.SchemaError
	if errors.As(err, &schemaErr) {
		fmt.Fprintln(os.Stderr, ui.Error("error")+": schema validation failed")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  %v\n", err)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Note("note")+": check the syntax of your schema file")
		fmt.Fprintln(os.Stderr, ui.Help("help")+": run `alab check` to validate schemas")
		return true
	}

	return false
}
