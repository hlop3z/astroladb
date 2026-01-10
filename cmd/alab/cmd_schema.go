package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/spf13/cobra"
)

// schemaCmd shows the schema at a specific migration revision.
func schemaCmd() *cobra.Command {
	var at, format, tableFilter string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Show schema at a specific migration revision",
		RunE: func(cmd *cobra.Command, args []string) error {
			if at == "" {
				printSchemaAtError()
				os.Exit(1)
			}

			client, err := newClient()
			if err != nil {
				if handleClientError(err) {
					os.Exit(1)
				}
				return err
			}
			defer client.Close()

			// Get schema at the specified revision
			schema, err := client.SchemaAtRevision(at)
			if err != nil {
				return err
			}

			// Get the list of tables (optionally filtered)
			tables := schema.TableList()
			if tableFilter != "" {
				var filtered []*ast.TableDef
				for _, t := range tables {
					if t.QualifiedName() == tableFilter || t.Name == tableFilter || t.SQLName() == tableFilter {
						filtered = append(filtered, t)
					}
				}
				if len(filtered) == 0 {
					return fmt.Errorf("table not found: %s", tableFilter)
				}
				tables = filtered
			}

			// Output in the requested format
			switch strings.ToLower(format) {
			case "table", "":
				printSchemaTable(at, tables)
			case "json":
				printSchemaJSON(at, tables)
			case "sql":
				printSchemaSQL(client, at, tables)
			default:
				return fmt.Errorf("unknown format: %s (valid formats: table, json, sql)", format)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&at, "at", "", "Migration revision (e.g., 003)")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format: table (default), json, sql")
	cmd.Flags().StringVarP(&tableFilter, "table", "t", "", "Show only a specific table (e.g., auth.user)")

	cmd.MarkFlagRequired("at")

	return cmd
}

// printSchemaTable prints the schema in a human-readable table format.
func printSchemaTable(revision string, tables []*ast.TableDef) {
	fmt.Printf("Schema at revision %s\n", revision)
	fmt.Printf("%s\n\n", strings.Repeat("=", 50))

	if len(tables) == 0 {
		fmt.Println("No tables defined.")
		return
	}

	fmt.Printf("Tables: %d\n\n", len(tables))

	for _, table := range tables {
		fmt.Printf("Table: %s\n", table.QualifiedName())
		fmt.Printf("  SQL Name: %s\n", table.SQLName())
		if table.Docs != "" {
			fmt.Printf("  Description: %s\n", table.Docs)
		}
		fmt.Println()

		// Print columns
		fmt.Println("  Columns:")
		fmt.Printf("    %-20s %-15s %-10s %-10s %s\n", "NAME", "TYPE", "NULLABLE", "UNIQUE", "NOTES")
		fmt.Printf("    %s\n", strings.Repeat("-", 70))

		for _, col := range table.Columns {
			nullable := "NOT NULL"
			if col.Nullable {
				nullable = "NULL"
			}

			unique := ""
			if col.Unique {
				unique = "UNIQUE"
			}

			notes := ""
			if col.PrimaryKey {
				notes = "PRIMARY KEY"
			} else if col.Reference != nil {
				notes = fmt.Sprintf("FK -> %s", col.Reference.Table)
			}
			if col.DefaultSet {
				if notes != "" {
					notes += ", "
				}
				notes += fmt.Sprintf("DEFAULT: %v", col.Default)
			}

			colType := col.Type
			if len(col.TypeArgs) > 0 {
				args := make([]string, len(col.TypeArgs))
				for i, arg := range col.TypeArgs {
					args[i] = fmt.Sprintf("%v", arg)
				}
				colType = fmt.Sprintf("%s(%s)", col.Type, strings.Join(args, ", "))
			}

			fmt.Printf("    %-20s %-15s %-10s %-10s %s\n", col.Name, colType, nullable, unique, notes)
		}

		// Print indexes
		if len(table.Indexes) > 0 {
			fmt.Println()
			fmt.Println("  Indexes:")
			for _, idx := range table.Indexes {
				uniqueStr := ""
				if idx.Unique {
					uniqueStr = " (UNIQUE)"
				}
				fmt.Printf("    - %s: [%s]%s\n", idx.Name, strings.Join(idx.Columns, ", "), uniqueStr)
			}
		}

		// Print foreign keys
		if len(table.ForeignKeys) > 0 {
			fmt.Println()
			fmt.Println("  Foreign Keys:")
			for _, fk := range table.ForeignKeys {
				fmt.Printf("    - %s: [%s] -> %s [%s]\n",
					fk.Name, strings.Join(fk.Columns, ", "),
					fk.RefTable, strings.Join(fk.RefColumns, ", "))
				if fk.OnDelete != "" || fk.OnUpdate != "" {
					fmt.Printf("      ON DELETE: %s, ON UPDATE: %s\n", fk.OnDelete, fk.OnUpdate)
				}
			}
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
			SQLName:   table.SQLName(),
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
	tableName := op.Name
	if op.Namespace != "" {
		tableName = op.Namespace + "_" + op.Name
	}

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
			indexName = "idx_" + tableName + "_" + strings.Join(idx.Columns, "_")
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
	case "date_time":
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
		return v.Expr
	default:
		return fmt.Sprintf("'%v'", v)
	}
}
