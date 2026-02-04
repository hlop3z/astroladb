package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hlop3z/astroladb/internal/runtime"
	"github.com/hlop3z/astroladb/internal/ui"
)

// DefaultGeneratorsDir is the default directory for generator files.
const DefaultGeneratorsDir = "generators"

// TitleGeneratorComplete is the success title after running a generator.
const TitleGeneratorComplete = "Generator Complete"

// TitleGeneratorAdded is the success title after adding a generator.
const TitleGeneratorAdded = "Generator Added"

// genCmd is the root command for code generators.
func genCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Run or manage code generators",
		Long:  `Run JavaScript code generators against your schema, or download shared generators.`,
		Example: `  # Run a generator
  alab gen run generators/models.js -o ./generated

  # Download a shared generator
  alab gen add https://example.com/generators/openapi_to_code.js`,
	}

	cmd.AddCommand(genRunCmd())
	cmd.AddCommand(genAddCmd())

	setupCommandHelp(cmd)
	return cmd
}

// genRunCmd runs a code generator against the schema.
func genRunCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:     "run <generator.js>",
		Short:   "Run a code generator against the schema",
		Args:    cobra.ExactArgs(1),
		Example: `  alab gen run generators/models.js -o ./generated`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputDir == "" {
				return fmt.Errorf("--output flag is required")
			}

			generatorFile := args[0]
			if !strings.HasSuffix(generatorFile, ".js") {
				generatorFile += ".js"
			}

			// Read generator file
			code, err := os.ReadFile(generatorFile)
			if err != nil {
				return fmt.Errorf("failed to read generator file: %w", err)
			}

			// Load schema
			schema, err := loadGeneratorSchema()
			if err != nil {
				return err
			}

			// Run generator in sandbox
			sandbox := runtime.NewSandbox(nil)
			sandbox.SetTimeout(5 * time.Minute)
			sandbox.SetCurrentFile(generatorFile)

			result, err := sandbox.RunGenerator(string(code), schema)
			if err != nil {
				return fmt.Errorf("generator failed: %w", err)
			}

			// Validate output
			if err := runtime.ValidateRenderOutput(result.Files, 500*1024*1024, 10000); err != nil {
				return fmt.Errorf("generator output invalid: %w", err)
			}

			// Write files atomically
			// Sort keys for deterministic output
			keys := make([]string, 0, len(result.Files))
			for k := range result.Files {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			var written []string
			for _, relPath := range keys {
				content := result.Files[relPath]
				fullPath := filepath.Join(outputDir, filepath.FromSlash(relPath))

				// Safety: ensure resolved path is inside output dir
				absOutput, _ := filepath.Abs(outputDir)
				absFile, _ := filepath.Abs(fullPath)
				if !strings.HasPrefix(absFile, absOutput+string(filepath.Separator)) && absFile != absOutput {
					return fmt.Errorf("path escape detected: %q resolves outside output directory", relPath)
				}

				// Create parent dirs
				if err := os.MkdirAll(filepath.Dir(fullPath), DirPerm); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}

				// Write to temp then rename (atomic)
				tmpFile := fullPath + ".tmp"
				if err := os.WriteFile(tmpFile, []byte(content), FilePerm); err != nil {
					return fmt.Errorf("failed to write file: %w", err)
				}
				if err := os.Rename(tmpFile, fullPath); err != nil {
					_ = os.Remove(tmpFile)
					return fmt.Errorf("failed to rename temp file: %w", err)
				}

				written = append(written, relPath)
			}

			// Summary
			var lines []string
			for _, f := range written {
				lines = append(lines, fmt.Sprintf("%s → %s", ui.Dim("gen"), ui.Primary(filepath.Join(outputDir, f))))
			}

			ui.ShowSuccess(
				TitleGeneratorComplete,
				fmt.Sprintf("Generated %s from %s:\n%s",
					ui.FormatCount(len(written), "file", "files"),
					ui.Primary(generatorFile),
					"  "+joinLines(lines, "\n  "),
				),
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (required)")
	_ = cmd.MarkFlagRequired("output")

	setupCommandHelp(cmd)
	return cmd
}

// genAddCmd downloads a shared generator from a URL.
func genAddCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "add <url>",
		Short:   "Download a shared generator",
		Long:    `Downloads a .js generator file from a URL and saves it to the generators/ directory.`,
		Args:    cobra.ExactArgs(1),
		Example: `  alab gen add https://raw.githubusercontent.com/hlop3z/astroladb/main/examples/parsing_openapi/openapi_to_code.js`,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			// Extract filename from URL
			parts := strings.Split(url, "/")
			filename := parts[len(parts)-1]

			// Ensure .js extension
			if !strings.HasSuffix(filename, ".js") {
				filename += ".js"
			}

			destPath := filepath.Join(DefaultGeneratorsDir, filename)

			// Check if file exists
			if _, err := os.Stat(destPath); err == nil && !force {
				return fmt.Errorf("generator %q already exists (use --force to overwrite)", destPath)
			}

			// Download
			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("failed to download generator: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to download generator: HTTP %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			// Create generators dir
			if err := os.MkdirAll(DefaultGeneratorsDir, DirPerm); err != nil {
				return fmt.Errorf("failed to create generators directory: %w", err)
			}

			// Write file
			if err := os.WriteFile(destPath, body, FilePerm); err != nil {
				return fmt.Errorf("failed to write generator file: %w", err)
			}

			ui.ShowSuccess(
				TitleGeneratorAdded,
				fmt.Sprintf("Downloaded %s to %s",
					ui.Primary(filename),
					ui.Primary(destPath),
				),
			)

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, FlagDescOverwrite)

	setupCommandHelp(cmd)
	return cmd
}

// loadGeneratorSchema loads the schema data for generators.
// Extracts the models map from the OpenAPI export so generators receive
// a clean { models: { namespace: tables[] }, tables: table[] } object.
func loadGeneratorSchema() (map[string]any, error) {
	client, err := newSchemaOnlyClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	data, err := client.SchemaExport("openapi")
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}
	var openapi map[string]any
	if err := json.Unmarshal(data, &openapi); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI schema: %w", err)
	}

	// Extract models from: openapi.paths["/schemas"].get.responses["200"].content[ContentTypeJSON].example.models
	models, err := extractModels(openapi)
	if err != nil {
		return nil, err
	}

	// Build flat tables array from the models map
	var tables []any
	modelsMap, _ := models.(map[string]any)
	for _, nsTables := range modelsMap {
		if arr, ok := nsTables.([]any); ok {
			tables = append(tables, arr...)
		}
	}

	return map[string]any{
		"models": models,
		"tables": tables,
	}, nil
}

// extractModels navigates the OpenAPI structure to get the models map.
func extractModels(openapi map[string]any) (any, error) {
	paths, _ := openapi["paths"].(map[string]any)
	schemas, _ := paths["/schemas"].(map[string]any)
	get, _ := schemas["get"].(map[string]any)
	responses, _ := get["responses"].(map[string]any)
	resp200, _ := responses["200"].(map[string]any)
	content, _ := resp200["content"].(map[string]any)
	appJSON, _ := content[ContentTypeJSON].(map[string]any)
	example, _ := appJSON["example"].(map[string]any)
	models, ok := example["models"]
	if !ok {
		return nil, fmt.Errorf("could not extract models from OpenAPI schema")
	}
	return models, nil
}
