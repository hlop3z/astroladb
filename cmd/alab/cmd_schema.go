package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/strutil"
	"github.com/hlop3z/astroladb/internal/ui"
)

// schemaCmd shows the schema at a specific migration revision.
func schemaCmd() *cobra.Command {
	var at, format string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Show schema at a specific migration revision",
		Long: `Show complete database schema at a specific migration revision.

Displays tables, columns, indexes, and foreign keys. Output formats: table (default), json, or sql.`,
		Example: `  # Show schema at revision 003 in table format
  alab schema --at 003

  # Export schema at revision 005 as JSON
  alab schema --at 005 --format json > schema.json

  # Generate SQL CREATE statements for the schema at revision 010
  alab schema --at 010 --format sql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if at == "" {
				printSchemaAtError()
				os.Exit(1)
			}

			client := mustClient()
			defer client.Close()

			// Get schema at the specified revision
			schema, err := client.SchemaAtRevision(at)
			if err != nil {
				return err
			}

			// Get the list of tables
			tables := schema.TableList()

			// Output in the requested format
			switch strings.ToLower(format) {
			case "tui", "":
				// Default to interactive TUI
				data := convertToStatusData(at, tables)
				ui.ShowStatus("browse", data)
			case "table":
				printSchemaTable(at, tables)
			case "json":
				printSchemaJSON(at, tables)
			case "sql":
				printSchemaSQL(client, at, tables)
			default:
				return fmt.Errorf("unknown format: %s (valid formats: tui, table, json, sql)", format)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&at, "at", "", "Migration revision (e.g., 003)")
	cmd.Flags().StringVarP(&format, "format", "f", "tui", "Output format: tui (interactive), table, json, sql")
	cmd.MarkFlagRequired("at")

	setupCommandHelp(cmd)
	return cmd
}

// printSchemaTable prints the schema in a human-readable table format.
func printSchemaTable(revision string, tables []*ast.TableDef) {
	fmt.Println(ui.RenderTitle(fmt.Sprintf("Schema at revision %s", revision)))
	fmt.Println()

	if len(tables) == 0 {
		fmt.Println(ui.Info("No tables defined."))
		return
	}

	fmt.Printf("  %s\n\n", ui.Muted(ui.FormatCount(len(tables), "table", "tables")))

	for _, table := range tables {
		// Table header with description
		header := fmt.Sprintf("Table: %s", table.QualifiedName())
		content := fmt.Sprintf("  SQL Name: %s", ui.Dim(table.FullName()))
		if table.Docs != "" {
			content += "\n  " + formatTableDescription(table.Docs)
		}

		fmt.Println(ui.Section(header, content))
		fmt.Println()

		// Columns
		fmt.Println("  " + ui.Primary("Columns:"))
		colTable := ui.NewStyledTable("NAME", "TYPE", "NULLABLE", "UNIQUE", "NOTES")

		for _, col := range table.Columns {
			nullable := ui.Error("NOT NULL")
			if col.Nullable {
				nullable = ui.Dim("NULL")
			}

			unique := ""
			if col.Unique {
				unique = ui.Warning("UNIQUE")
			}

			colTable.AddRow(
				ui.Bold(col.Name),
				formatColumnType(col),
				nullable,
				unique,
				formatColumnNotes(col),
			)
		}

		fmt.Print(strutil.Indent(colTable.String(), 2))
		fmt.Println()

		// Indexes
		if len(table.Indexes) > 0 {
			fmt.Println("  " + ui.Primary("Indexes:"))
			list := ui.NewList()
			for _, idx := range table.Indexes {
				uniqueStr := ""
				if idx.Unique {
					uniqueStr = ui.Warning(" (UNIQUE)")
				}
				list.AddInfo(fmt.Sprintf("%s: [%s]%s",
					ui.Bold(idx.Name),
					strings.Join(idx.Columns, ", "),
					uniqueStr))
			}
			fmt.Println(list.String())
			fmt.Println()
		}

		// Foreign Keys
		if len(table.ForeignKeys) > 0 {
			fmt.Println("  " + ui.Primary("Foreign Keys:"))
			list := ui.NewList()
			for _, fk := range table.ForeignKeys {
				fkStr := fmt.Sprintf("%s: [%s] → %s [%s]",
					ui.Bold(fk.Name),
					strings.Join(fk.Columns, ", "),
					ui.Primary(fk.RefTable),
					strings.Join(fk.RefColumns, ", "))

				if fk.OnDelete != "" || fk.OnUpdate != "" {
					fkStr += ui.Dim(fmt.Sprintf(" (ON DELETE: %s, ON UPDATE: %s)",
						fk.OnDelete, fk.OnUpdate))
				}
				list.AddInfo(fkStr)
			}
			fmt.Println(list.String())
			fmt.Println()
		}

		fmt.Println()
	}
}

// printSchemaJSON prints the schema in JSON format.
func printSchemaJSON(revision string, tables []*ast.TableDef) {
	type ColumnOutput struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		TypeArgs   []any  `json:"type_args,omitempty"`
		Nullable   bool   `json:"nullable"`
		Unique     bool   `json:"unique,omitempty"`
		PrimaryKey bool   `json:"primary_key,omitempty"`
		Default    any    `json:"default,omitempty"`
		Reference  string `json:"reference,omitempty"`
		Docs       string `json:"docs,omitempty"`
	}

	type IndexOutput struct {
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
		Unique  bool     `json:"unique,omitempty"`
	}

	type ForeignKeyOutput struct {
		Name       string   `json:"name"`
		Columns    []string `json:"columns"`
		RefTable   string   `json:"ref_table"`
		RefColumns []string `json:"ref_columns"`
		OnDelete   string   `json:"on_delete,omitempty"`
		OnUpdate   string   `json:"on_update,omitempty"`
	}

	type TableOutput struct {
		Namespace   string             `json:"namespace"`
		Name        string             `json:"name"`
		SQLName     string             `json:"sql_name"`
		Columns     []ColumnOutput     `json:"columns"`
		Indexes     []IndexOutput      `json:"indexes,omitempty"`
		ForeignKeys []ForeignKeyOutput `json:"foreign_keys,omitempty"`
		Docs        string             `json:"docs,omitempty"`
	}

	type SchemaOutput struct {
		Revision string        `json:"revision"`
		Tables   []TableOutput `json:"tables"`
	}

	output := SchemaOutput{
		Revision: revision,
		Tables:   make([]TableOutput, len(tables)),
	}

	for i, table := range tables {
		t := TableOutput{
			Namespace: table.Namespace,
			Name:      table.Name,
			SQLName:   table.FullName(),
			Columns:   make([]ColumnOutput, len(table.Columns)),
			Docs:      table.Docs,
		}

		for j, col := range table.Columns {
			c := ColumnOutput{
				Name:       col.Name,
				Type:       col.Type,
				TypeArgs:   col.TypeArgs,
				Nullable:   col.Nullable,
				Unique:     col.Unique,
				PrimaryKey: col.PrimaryKey,
				Docs:       col.Docs,
			}
			if col.DefaultSet {
				c.Default = col.Default
			}
			if col.Reference != nil {
				c.Reference = col.Reference.Table
			}
			t.Columns[j] = c
		}

		if len(table.Indexes) > 0 {
			t.Indexes = make([]IndexOutput, len(table.Indexes))
			for j, idx := range table.Indexes {
				t.Indexes[j] = IndexOutput{
					Name:    idx.Name,
					Columns: idx.Columns,
					Unique:  idx.Unique,
				}
			}
		}

		if len(table.ForeignKeys) > 0 {
			t.ForeignKeys = make([]ForeignKeyOutput, len(table.ForeignKeys))
			for j, fk := range table.ForeignKeys {
				t.ForeignKeys[j] = ForeignKeyOutput{
					Name:       fk.Name,
					Columns:    fk.Columns,
					RefTable:   fk.RefTable,
					RefColumns: fk.RefColumns,
					OnDelete:   fk.OnDelete,
					OnUpdate:   fk.OnUpdate,
				}
			}
		}

		output.Tables[i] = t
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}

// printSchemaSQL prints the schema as SQL CREATE statements.
func printSchemaSQL(client interface{}, revision string, tables []*ast.TableDef) {
	fmt.Printf("-- Schema at revision %s\n", revision)
	fmt.Printf("-- Generated by alab schema --at %s --format sql\n\n", revision)

	for i, table := range tables {
		if i > 0 {
			fmt.Println()
		}

		// Convert table to CreateTable operation
		createOp := &ast.CreateTable{
			TableOp: ast.TableOp{
				Namespace: table.Namespace,
				Name:      table.Name,
			},
			Columns:     table.Columns,
			Indexes:     table.Indexes,
			ForeignKeys: table.ForeignKeys,
		}

		// Use a generic SQL format since we might not have dialect access
		// In a full implementation, we would use the dialect from the client
		fmt.Printf("-- Table: %s\n", table.QualifiedName())
		printCreateTableSQL(createOp)
	}
}

// printCreateTableSQL prints a CREATE TABLE statement.
// This is a simplified version that doesn't require a dialect.
func printCreateTableSQL(op *ast.CreateTable) {
	tableName := op.Table()

	fmt.Printf("CREATE TABLE %s (\n", tableName)

	for i, col := range op.Columns {
		colSQL := fmt.Sprintf("  %s %s", col.Name, mapTypeToSQL(col.Type, col.TypeArgs))

		if col.PrimaryKey {
			colSQL += " PRIMARY KEY"
		}
		if !col.Nullable {
			colSQL += " NOT NULL"
		}
		if col.Unique && !col.PrimaryKey {
			colSQL += " UNIQUE"
		}
		if col.DefaultSet {
			colSQL += fmt.Sprintf(" DEFAULT %s", formatDefaultValue(col.Default))
		}

		if i < len(op.Columns)-1 {
			colSQL += ","
		}
		fmt.Println(colSQL)
	}

	fmt.Println(");")

	// Print indexes
	for _, idx := range op.Indexes {
		uniqueStr := ""
		if idx.Unique {
			uniqueStr = "UNIQUE "
		}
		indexName := idx.Name
		if indexName == "" {
			indexName = strutil.IndexName(tableName, idx.Columns...)
		}
		fmt.Printf("CREATE %sINDEX %s ON %s (%s);\n",
			uniqueStr, indexName, tableName, strings.Join(idx.Columns, ", "))
	}
}

// mapTypeToSQL maps DSL types to SQL types.
func mapTypeToSQL(typ string, args []any) string {
	switch typ {
	case "id", "uuid":
		return "UUID"
	case "string":
		length := 255
		if len(args) > 0 {
			if l, ok := args[0].(int); ok {
				length = l
			} else if l, ok := args[0].(float64); ok {
				length = int(l)
			}
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case "text":
		return "TEXT"
	case "integer":
		return "INTEGER"
	case "float":
		return "REAL"
	case "decimal":
		precision, scale := 10, 2
		if len(args) >= 2 {
			if p, ok := args[0].(int); ok {
				precision = p
			} else if p, ok := args[0].(float64); ok {
				precision = int(p)
			}
			if s, ok := args[1].(int); ok {
				scale = s
			} else if s, ok := args[1].(float64); ok {
				scale = int(s)
			}
		}
		return fmt.Sprintf("DECIMAL(%d, %d)", precision, scale)
	case "boolean":
		return "BOOLEAN"
	case "date":
		return "DATE"
	case "time":
		return "TIME"
	case "datetime":
		return "TIMESTAMP"
	case "json":
		return "JSONB"
	case "base64":
		return "TEXT"
	case "enum":
		return "VARCHAR(255)"
	default:
		return strings.ToUpper(typ)
	}
}

// formatDefaultValue formats a default value for SQL.
func formatDefaultValue(val any) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case *ast.SQLExpr:
		return v.Postgres
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

func formatTableDescription(docs string) string {
	return "Description: " + ui.Dim(docs)
}

func formatColumnType(col *ast.ColumnDef) string {
	colType := col.Type
	if len(col.TypeArgs) > 0 {
		args := make([]string, len(col.TypeArgs))
		for i, arg := range col.TypeArgs {
			args[i] = fmt.Sprintf("%v", arg)
		}
		colType = fmt.Sprintf("%s(%s)", col.Type, strings.Join(args, ", "))
	}
	return ui.Info(colType)
}

func formatColumnNotes(col *ast.ColumnDef) string {
	var notes []string

	if col.PrimaryKey {
		notes = append(notes, ui.Success("PRIMARY KEY"))
	}
	if col.Reference != nil {
		notes = append(notes, ui.Info(fmt.Sprintf("FK → %s", col.Reference.Table)))
	}
	if col.DefaultSet {
		notes = append(notes, ui.Dim(fmt.Sprintf("DEFAULT: %v", col.Default)))
	}

	if len(notes) == 0 {
		return ""
	}
	return strings.Join(notes, ", ")
}

// convertToStatusData converts ast.TableDef slices to ui.StatusData for the schema command.
func convertToStatusData(revision string, tables []*ast.TableDef) ui.StatusData {
	return ui.StatusData{
		Revision: revision,
		Tables:   convertTablesToUI(tables),
	}
}
