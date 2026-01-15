package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
)

// tableCmd creates a new table schema file.
func tableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "table <namespace> <table_name>",
		Short: "Create a new table schema file",
		Long: `Create a new table schema file in the schemas directory.

Both namespace and table_name are normalized to lowercase snake_case.`,
		Example: `  # Create a user table in the auth namespace
  alab table auth User
  # Output: schemas/auth/user.js

  # Create a post_item table in the blog namespace
  alab table Blog post-item
  # Output: schemas/blog/post_item.js

  # Create a user_profile table in the API namespace
  alab table API UserProfile
  # Output: schemas/api/user_profile.js`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Normalize namespace and table name to lowercase snake_case
			namespace := toSnakeCase(args[0])
			tableName := toSnakeCase(args[1])

			// Validate names
			if namespace == "" {
				return fmt.Errorf("namespace cannot be empty")
			}
			if tableName == "" {
				return fmt.Errorf("table name cannot be empty")
			}

			// Load config to get schemas directory
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Create the namespace directory
			namespaceDir := filepath.Join(cfg.SchemasDir, namespace)
			if err := os.MkdirAll(namespaceDir, DirPerm); err != nil {
				return fmt.Errorf("failed to create namespace directory: %w", err)
			}

			// Create the schema file
			schemaFile := filepath.Join(namespaceDir, tableName+".js")

			// Check if file already exists
			if _, err := os.Stat(schemaFile); err == nil {
				return fmt.Errorf("schema file already exists: %s", schemaFile)
			}

			// Generate schema content
			content, err := generateTableSchema(namespace, tableName)
			if err != nil {
				return err
			}

			// Write the file
			if err := os.WriteFile(schemaFile, []byte(content), FilePerm); err != nil {
				return fmt.Errorf("failed to write schema file: %w", err)
			}

			// Show success with details
			list := ui.NewList()
			list.AddInfo(fmt.Sprintf("File:      %s", ui.FilePath(schemaFile)))
			list.AddInfo(fmt.Sprintf("Namespace: %s", ui.Primary(namespace)))
			list.AddInfo(fmt.Sprintf("Table:     %s", ui.Primary(tableName)))

			ui.ShowSuccess(
				TitleSchemaFileCreated,
				list.String()+"\n"+
					ui.Help("Next steps:\n  1. Edit the schema file to define columns\n  2. Run 'alab new <name>' to generate a migration"),
			)
			return nil
		},
	}

	setupCommandHelp(cmd)
	return cmd
}

// generateTableSchema generates the content for a new table schema file.
func generateTableSchema(namespace, tableName string) (string, error) {
	tmplContent := mustReadTemplate("templates/table.js.tmpl")
	tmpl, err := template.New("table").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse table template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		Namespace string
		TableName string
	}{
		Namespace: namespace,
		TableName: tableName,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute table template: %w", err)
	}
	return buf.String(), nil
}
