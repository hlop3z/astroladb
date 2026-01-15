package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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

// setupCommandHelp sets up custom help for a specific command.
func setupCommandHelp(cmd *cobra.Command) {
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		renderCommandHelp(c)
	})
}

// CommandCategory represents a category of related commands.
type CommandCategory struct {
	Title    string
	Commands []CommandInfo
}

// CommandInfo represents a command with its name and description.
type CommandInfo struct {
	Name string
	Desc string
}

// setupCategoryHelp sets up help that displays commands organized by categories.
func setupCategoryHelp(cmd *cobra.Command, title, subtitle string, categories []CommandCategory, flags []struct{ flag, desc string }) {
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		renderCategoryHelp(title, subtitle, categories, flags)
	})
}

// renderCategoryHelp renders help with commands organized by categories.
func renderCategoryHelp(title, subtitle string, categories []CommandCategory, flags []struct{ flag, desc string }) {
	dividerWidth := 80
	divider := ui.Dim(strings.Repeat("─", dividerWidth))

	// Header with app title
	fmt.Println()
	fmt.Println(ui.SectionTitle(title))
	fmt.Println()
	// Format subtitle: yellow star + dim text
	if strings.HasPrefix(subtitle, "★") {
		fmt.Println(ui.Yellow("★") + ui.Dim(strings.TrimPrefix(subtitle, "★")))
	} else {
		fmt.Println(ui.Dim(subtitle))
	}
	fmt.Println()

	// Flags section first
	if len(flags) > 0 {
		fmt.Println()
		fmt.Println(ui.SectionTitle("Global Flags"))
		fmt.Println(divider)
		fmt.Printf(" %s %s\n", ui.Blue(padStr("Flag", 22)), ui.Blue("Description"))
		fmt.Println(divider)
		for _, f := range flags {
			fmt.Printf(" %s %s\n", ui.Cyan(padStr(f.flag, 22)), f.desc)
		}
		fmt.Println(divider)
	}

	// Display each category with commands
	for _, cat := range categories {
		fmt.Println()
		fmt.Println(ui.SectionTitle(cat.Title))
		fmt.Println(divider)
		fmt.Printf(" %s %s\n", ui.Blue(padStr("Command", 12)), ui.Blue("Description"))
		fmt.Println(divider)
		for _, cmd := range cat.Commands {
			fmt.Printf(" %s %s\n", ui.Yellow(padStr(cmd.Name, 12)), cmd.Desc)
		}
		fmt.Println(divider)
	}

	// Footer
	fmt.Println()
	fmt.Println(ui.Help("Use 'alab [command] --help' for more information about a command"))
}

// padStr pads a string to a fixed width with spaces.
func padStr(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// renderCommandHelp renders help for a specific command.
func renderCommandHelp(cmd *cobra.Command) {
	dividerWidth := 80

	fmt.Println()
	fmt.Println(ui.Header(fmt.Sprintf("⏳ alab %s", cmd.Name()), ui.Success))
	fmt.Println()

	// Description
	if cmd.Short != "" {
		fmt.Println(ui.Dim(cmd.Short))
		fmt.Println()
	}

	// Usage
	if cmd.Use != "" {
		fmt.Println(ui.Header("Usage", ui.Success))
		fmt.Println(ui.Dim(strings.Repeat("─", dividerWidth)))
		fmt.Printf("  alab %s\n", cmd.Use)
		fmt.Println(ui.Dim(strings.Repeat("─", dividerWidth)))
		fmt.Println()
	}

	// Long description
	if cmd.Long != "" {
		fmt.Println(ui.Header("Description", ui.Success))
		fmt.Println(ui.Dim(strings.Repeat("─", dividerWidth)))
		fmt.Println(wrapText(cmd.Long, dividerWidth-4))
		fmt.Println(ui.Dim(strings.Repeat("─", dividerWidth)))
		fmt.Println()
	}

	// Flags
	if cmd.HasAvailableLocalFlags() {
		fmt.Println(ui.Header("Flags", ui.Success))

		grid := ui.NewGrid("Flag", "Description", dividerWidth, 24)

		cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			if !flag.Hidden {
				flagName := fmt.Sprintf("--%s", flag.Name)
				if flag.Shorthand != "" {
					flagName = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
				}
				grid.AddRow(flagName, flag.Usage)
			}
		})

		fmt.Print(grid.String())
	}

	// Global flags
	if cmd.HasAvailableInheritedFlags() {
		fmt.Println(ui.Header("Global Flags", ui.Success))

		grid := ui.NewGrid("Flag", "Description", dividerWidth, 24)

		cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
			if !flag.Hidden {
				flagName := fmt.Sprintf("--%s", flag.Name)
				if flag.Shorthand != "" {
					flagName = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
				}
				grid.AddRow(flagName, flag.Usage)
			}
		})

		fmt.Print(grid.String())
	}

	// Examples
	if cmd.Example != "" {
		fmt.Println(ui.Header("Examples", ui.Success))
		fmt.Println(ui.Dim(strings.Repeat("─", dividerWidth)))
		fmt.Println(cmd.Example)
		fmt.Println(ui.Dim(strings.Repeat("─", dividerWidth)))
		fmt.Println()
	}

	fmt.Println(ui.Help("Use 'alab --help' to see all available commands"))
}

// wrapText wraps text to a specific width.
func wrapText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = "  " + word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = "  " + word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}
