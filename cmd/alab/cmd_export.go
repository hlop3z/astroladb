package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hlop3z/astroladb/internal/cli"
	"github.com/hlop3z/astroladb/pkg/astroladb"
	"github.com/spf13/cobra"
)

// allFormats lists all supported export formats.
var allFormats = []string{"openapi", "graphql", "typescript", "go", "python", "rust"}

// exportCmd exports the schema in various formats.
func exportCmd() *cobra.Command {
	var format, dir string
	var stdout, mik bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export schema (openapi, graphql, typescript, go, python, rust, all)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newSchemaOnlyClient()
			if err != nil {
				return err
			}
			defer client.Close()

			var opts []astroladb.ExportOption
			if mik {
				opts = append(opts, astroladb.WithMik())
			}

			// Handle "all" format
			formats := []string{format}
			if format == "all" {
				if stdout {
					return fmt.Errorf("--stdout cannot be used with --format all")
				}
				formats = allFormats
			}

			// Get namespaces for splitting
			namespaces, err := client.GetNamespaces()
			if err != nil {
				return err
			}
			// Filter out empty namespaces
			var validNs []string
			for _, ns := range namespaces {
				if ns != "" {
					validNs = append(validNs, ns)
				}
			}

			for _, fmt := range formats {
				// OpenAPI and GraphQL always go to root (single file)
				// Other formats split by namespace
				if fmt == "openapi" || fmt == "graphql" {
					if err := exportFormat(client, fmt, dir, stdout, opts); err != nil {
						return err
					}
				} else {
					for _, ns := range validNs {
						nsOpts := append(opts, astroladb.WithNamespace(ns))
						nsDir := filepath.Join(dir, ns)
						if err := exportFormat(client, fmt, nsDir, stdout, nsOpts); err != nil {
							return err
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "openapi", "Export format (openapi, graphql, typescript, go, python, rust, all)")
	cmd.Flags().StringVar(&dir, "dir", "exports", "Output directory")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print to stdout")
	cmd.Flags().BoolVar(&mik, "mik", false, "Use mik_sdk style for Rust")

	return cmd
}

// exportFormat exports a single format.
func exportFormat(client *astroladb.Client, format, dir string, stdout bool, opts []astroladb.ExportOption) error {
	data, err := client.SchemaExport(format, opts...)
	if err != nil {
		return err
	}

	if stdout {
		fmt.Println(string(data))
		return nil
	}

	// Auto-generate filename based on format
	var filename string
	switch format {
	case "openapi":
		filename = "openapi.json"
	case "graphql", "gql":
		filename = "schema.graphql"
	case "typescript", "ts":
		filename = "types.ts"
	case "go", "golang":
		filename = "types.go"
	case "python", "py":
		filename = "types.py"
	case "rust", "rs":
		filename = "types.rs"
	default:
		filename = format + ".json"
	}

	outputPath := filepath.Join(dir, filename)

	// Create directory if needed
	dirPath := filepath.Dir(outputPath)
	if dirPath != "." && dirPath != "" {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	fmt.Printf("  %s %s → %s\n", cli.Success("Exported"), cli.Dim(format), cli.FilePath(outputPath))

	return nil
}
