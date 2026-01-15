package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// statusCmd shows the unified status TUI with 4 tabs.
func statusCmd() *cobra.Command {
	var jsonOutput bool
	var textOutput bool
	var at string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show schema, history, and drift status",
		Long: `Display unified status with 4 tabs:

  1. Browse  - Explore tables, columns, indexes, and foreign keys
  2. History - View applied migration history with timing
  3. Verify  - Check chain integrity and git status
  4. Drift   - Compare schema vs database state

Use --json for CI/CD integration (exits with code 1 if pending migrations).
Use --text for non-interactive output.`,
		Example: `  # Open interactive status TUI
  alab status

  # View schema at a specific revision
  alab status --at 003

  # Output as plain text (non-interactive)
  alab status --text

  # Output as JSON for CI/CD
  alab status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			// Get migration status
			statuses, err := client.MigrationStatus()
			if err != nil {
				return err
			}

			// JSON output mode for CI/CD integration
			if jsonOutput {
				return outputStatusJSON(statuses)
			}

			// Build StatusData for TUI
			data := ui.StatusData{}

			// Get the revision to display (default to latest applied)
			revision := at
			if revision == "" {
				for i := len(statuses) - 1; i >= 0; i-- {
					if statuses[i].Status == ui.StatusApplied {
						revision = statuses[i].Revision
						break
					}
				}
				if revision == "" && len(statuses) > 0 {
					revision = statuses[len(statuses)-1].Revision
				}
			}
			data.Revision = revision

			// Tab 1: Browse - Get schema at revision
			if revision != "" {
				schema, err := client.SchemaAtRevision(revision)
				if err == nil {
					tables := schema.TableList()
					data.Tables = convertTablesToUI(tables)
				}
			}

			// Tab 2: History - Convert migrations
			data.Migrations = make([]ui.MigrationItem, len(statuses))
			for i, s := range statuses {
				appliedAt := ""
				if s.AppliedAt != nil {
					appliedAt = s.AppliedAt.Format(TimeDisplay)
				}
				data.Migrations[i] = ui.MigrationItem{
					Revision:  s.Revision,
					Name:      s.Name,
					Status:    s.Status,
					AppliedAt: appliedAt,
					Checksum:  s.Checksum,
				}
			}

			// Tab 3: Verify - Get git info
			data.GitBranch = getGitBranch()
			data.GitStatus = getGitMigrationStatus()

			// Tab 4: Drift - Check drift
			driftResult, err := client.CheckDrift()
			if err == nil && driftResult != nil {
				data.DriftItems = convertDriftToUI(driftResult)
			}

			// Text output mode
			if textOutput {
				ui.ShowStatusText(data)
				return nil
			}

			// No migrations found
			if len(statuses) == 0 {
				fmt.Println(ui.Info("No migrations found."))
				fmt.Println(ui.Muted("Run 'alab new <name>' to create your first migration."))
				return nil
			}

			// Show interactive TUI
			ui.ShowStatus("browse", data)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON for CI/CD")
	cmd.Flags().BoolVar(&textOutput, "text", false, "Output as plain text (non-interactive)")
	cmd.Flags().StringVar(&at, "at", "", "Migration revision (e.g., 003)")

	setupCommandHelp(cmd)
	return cmd
}

// outputStatusJSON outputs migration status as JSON.
func outputStatusJSON(statuses []astroladb.MigrationStatus) error {
	var applied, pending int
	for _, s := range statuses {
		if s.Status == ui.StatusApplied {
			applied++
		} else {
			pending++
		}
	}

	migrations := make([]map[string]any, len(statuses))
	for i, s := range statuses {
		m := map[string]any{
			"revision":   s.Revision,
			"name":       s.Name,
			"status":     s.Status,
			"applied_at": nil,
		}
		if s.AppliedAt != nil {
			m["applied_at"] = s.AppliedAt.Format(TimeJSON)
		}
		migrations[i] = m
	}

	output := map[string]any{
		"applied":    applied,
		"pending":    pending,
		"migrations": migrations,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return err
	}

	// Exit with code 1 if there are pending migrations
	if pending > 0 {
		os.Exit(1)
	}
	return nil
}

// convertTablesToUI converts ast.TableDef to ui.TableInfo.
func convertTablesToUI(tables []*ast.TableDef) []ui.TableInfo {
	result := make([]ui.TableInfo, len(tables))

	for i, table := range tables {
		t := ui.TableInfo{
			Name:      table.Name,
			Namespace: table.Namespace,
			Docs:      table.Docs,
		}

		// Convert columns
		t.Columns = make([]ui.ColumnInfo, len(table.Columns))
		for j, col := range table.Columns {
			typeArgs := ""
			if len(col.TypeArgs) > 0 {
				args := make([]string, len(col.TypeArgs))
				for k, arg := range col.TypeArgs {
					args[k] = fmt.Sprintf("%v", arg)
				}
				typeArgs = strings.Join(args, ", ")
			}

			defaultVal := ""
			if col.DefaultSet {
				defaultVal = fmt.Sprintf("%v", col.Default)
			}

			t.Columns[j] = ui.ColumnInfo{
				Name:     col.Name,
				Type:     col.Type,
				TypeArgs: typeArgs,
				Nullable: col.Nullable,
				Unique:   col.Unique,
				IsPK:     col.PrimaryKey,
				Default:  defaultVal,
				Docs:     col.Docs,
			}
		}

		// Convert indexes
		t.Indexes = make([]ui.IndexInfo, len(table.Indexes))
		for j, idx := range table.Indexes {
			t.Indexes[j] = ui.IndexInfo{
				Name:    idx.Name,
				Columns: idx.Columns,
				Unique:  idx.Unique,
			}
		}

		// Convert foreign keys (table-level)
		var fks []ui.ForeignKeyInfo
		for _, fk := range table.ForeignKeys {
			column := ""
			refColumn := ""
			if len(fk.Columns) > 0 {
				column = fk.Columns[0]
			}
			if len(fk.RefColumns) > 0 {
				refColumn = fk.RefColumns[0]
			}
			fks = append(fks, ui.ForeignKeyInfo{
				Name:      fk.Name,
				Column:    column,
				RefTable:  fk.RefTable,
				RefColumn: refColumn,
				OnDelete:  fk.OnDelete,
				OnUpdate:  fk.OnUpdate,
			})
		}

		// Also check column-level references (belongs_to)
		for _, col := range table.Columns {
			if col.Reference != nil {
				refColumn := col.Reference.Column
				if refColumn == "" {
					refColumn = "id"
				}
				fks = append(fks, ui.ForeignKeyInfo{
					Name:      fmt.Sprintf("fk_%s_%s", table.Name, col.Name),
					Column:    col.Name,
					RefTable:  col.Reference.Table,
					RefColumn: refColumn,
					OnDelete:  col.Reference.OnDelete,
					OnUpdate:  col.Reference.OnUpdate,
				})
			}
		}
		t.ForeignKeys = fks

		result[i] = t
	}

	return result
}

// convertDriftToUI converts DriftResult to ui.DriftItem slice.
func convertDriftToUI(result *astroladb.DriftResult) []ui.DriftItem {
	var items []ui.DriftItem

	// Convert missing tables
	for _, table := range result.MissingTables {
		items = append(items, ui.DriftItem{
			Type:    ui.DriftMissing,
			Object:  table,
			Details: ui.DriftMsgTableMissing,
		})
	}

	// Convert extra tables (filter out internal tables)
	for _, table := range result.ExtraTables {
		if isInternalTable(table) {
			continue
		}
		items = append(items, ui.DriftItem{
			Type:    ui.DriftExtra,
			Object:  table,
			Details: ui.DriftMsgTableExtra,
		})
	}

	// Convert table diffs
	for tableName, diff := range result.TableDiffs {
		if isInternalTable(tableName) {
			continue
		}

		for _, col := range diff.MissingColumns {
			items = append(items, ui.DriftItem{
				Type:    ui.DriftMissing,
				Object:  fmt.Sprintf("%s.%s", tableName, col),
				Details: ui.DriftMsgColumnMissing,
			})
		}

		for _, col := range diff.ExtraColumns {
			items = append(items, ui.DriftItem{
				Type:    ui.DriftExtra,
				Object:  fmt.Sprintf("%s.%s", tableName, col),
				Details: ui.DriftMsgColumnExtra,
			})
		}

		for _, col := range diff.ModifiedColumns {
			items = append(items, ui.DriftItem{
				Type:    ui.DriftModified,
				Object:  fmt.Sprintf("%s.%s", tableName, col),
				Details: ui.DriftMsgColumnModified,
			})
		}

		for _, idx := range diff.MissingIndexes {
			items = append(items, ui.DriftItem{
				Type:    ui.DriftMissing,
				Object:  fmt.Sprintf("%s.%s (index)", tableName, idx),
				Details: ui.DriftMsgIndexMissing,
			})
		}

		for _, idx := range diff.ExtraIndexes {
			items = append(items, ui.DriftItem{
				Type:    ui.DriftExtra,
				Object:  fmt.Sprintf("%s.%s (index)", tableName, idx),
				Details: ui.DriftMsgIndexExtra,
			})
		}
	}

	return items
}

// isInternalTable returns true if the table is an internal alab table.
func isInternalTable(name string) bool {
	for _, internal := range ui.InternalTables {
		if name == internal || strings.HasSuffix(name, "."+internal) {
			return true
		}
	}

	return false
}

// getGitBranch returns the current git branch name.
func getGitBranch() string {
	// Try to read from .git/HEAD
	data, err := os.ReadFile(".git/HEAD")
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}
	return ""
}

// getGitMigrationStatus returns uncommitted migration files.
func getGitMigrationStatus() []string {
	// This is a simplified check - in production you'd use git status
	return nil
}
