package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/hlop3z/astroladb/pkg/astroladb"
)

// exportCmd exports the schema in various formats.
func exportCmd() *cobra.Command {
	var format, dir string
	var stdout, mik bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export schema (openapi, graphql, typescript, go, python, rust, all)",
		Long: `Export the database schema in various formats for API documentation or code generation.

Formats: openapi, graphql, typescript, go, python, rust, all.
OpenAPI/GraphQL generate single files. Other formats split by namespace into subdirectories.`,
		Example: `  # Export schema as OpenAPI specification
  alab export --format openapi

  # Export to TypeScript definitions in a custom directory
  alab export --format typescript --dir ./generated

  # Export to all formats at once
  alab export --format all --dir ./exports

  # Export GraphQL schema to stdout
  alab export --format graphql --stdout

  # Export Rust types using mik_sdk style
  alab export --format rust --mik`,
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
				formats = AllExportFormats
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

			// Track exported files as "format → path" entries
			type exportEntry struct {
				format string
				path   string
			}
			var exportedFiles []exportEntry

			for _, f := range formats {
				// OpenAPI and GraphQL always go to root (single file)
				// Other formats split by namespace
				if f == "openapi" || f == "graphql" {
					path, err := exportFormat(client, f, dir, stdout, opts)
					if err != nil {
						return err
					}
					if path != "" {
						exportedFiles = append(exportedFiles, exportEntry{f, path})
					}
				} else {
					for _, ns := range validNs {
						nsOpts := append(opts, astroladb.WithNamespace(ns))
						nsDir := filepath.Join(dir, ns)
						path, err := exportFormat(client, f, nsDir, stdout, nsOpts)
						if err != nil {
							return err
						}
						if path != "" {
							exportedFiles = append(exportedFiles, exportEntry{f, path})
						}
					}
				}
			}

			// Show summary if files were exported
			if !stdout && len(exportedFiles) > 0 {
				var lines []string
				for _, e := range exportedFiles {
					lines = append(lines, fmt.Sprintf("%s → %s", ui.Dim(e.format), ui.Primary(e.path)))
				}

				ui.ShowSuccess(
					TitleExportComplete,
					fmt.Sprintf("Exported %s:\n%s",
						ui.FormatCount(len(exportedFiles), "file", "files"),
						"  "+joinLines(lines, "\n  "),
					),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "openapi", "Export format (openapi, graphql, typescript, go, python, rust, all)")
	cmd.Flags().StringVar(&dir, "dir", DefaultExportsDir, "Output directory")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print to stdout")
	cmd.Flags().BoolVar(&mik, "mik", false, "Use mik_sdk style for Rust")

	setupCommandHelp(cmd)
	return cmd
}

// exportFormat exports a single format and returns the output path.
func exportFormat(client *astroladb.Client, format, dir string, stdout bool, opts []astroladb.ExportOption) (string, error) {
	data, err := client.SchemaExport(format, opts...)
	if err != nil {
		return "", err
	}

	if stdout {
		fmt.Println(string(data))
		return "", nil
	}

	// Auto-generate filename based on format
	filename := GetExportFilename(format)
	outputPath := filepath.Join(dir, filename)

	// Create directory if needed
	dirPath := filepath.Dir(outputPath)
	if dirPath != "." && dirPath != "" {
		if err := os.MkdirAll(dirPath, DirPerm); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}

	if err := os.WriteFile(outputPath, data, FilePerm); err != nil {
		return "", fmt.Errorf("failed to write output file: %w", err)
	}

	return outputPath, nil
}

// joinLines joins strings with a separator.
func joinLines(lines []string, sep string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += sep + lines[i]
	}
	return result
}
