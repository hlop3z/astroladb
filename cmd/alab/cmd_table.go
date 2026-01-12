package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

// tableCmd creates a new table schema file.
func tableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "table <namespace> <table_name>",
		Short: "Create a new table schema file",
		Long: `Create a new table schema file in the schemas directory.

Both namespace and table_name are normalized to lowercase snake_case.
Examples:
  alab table auth User       -> schemas/auth/user.js
  alab table Blog post-item  -> schemas/blog/post_item.js
  alab table API UserProfile -> schemas/api/user_profile.js`,
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
			if err := os.MkdirAll(namespaceDir, 0755); err != nil {
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
			if err := os.WriteFile(schemaFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write schema file: %w", err)
			}

			fmt.Printf("Created schema: %s\n", schemaFile)
			fmt.Printf("  Namespace: %s\n", namespace)
			fmt.Printf("  Table:     %s\n", tableName)
			return nil
		},
	}

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
