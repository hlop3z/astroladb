package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hlop3z/astroladb/internal/ui"
	"github.com/spf13/cobra"
)

// initCmd creates the schemas/ and migrations/ directories.
func initCmd() *cobra.Command {
	var withDemo bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project structure (creates schemas/, migrations/, types/ dirs)",
		Long: `Initialize a new Alab project by creating the necessary directory structure and configuration file.

This command creates:
- schemas/ directory for your schema definitions
- migrations/ directory for migration files
- types/ directory for TypeScript type definitions
- alab.yaml configuration file (if it doesn't exist)

Use --demo to include example schemas (auth.user, auth.role, blog.post).`,
		Example: `  # Initialize a new project
  alab init

  # Initialize with demo schemas
  alab init --demo

  # This creates the following structure:
  # ├── alab.yaml
  # ├── schemas/
  # ├── migrations/
  # └── types/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Track created files/directories
			created := ui.NewList()

			dirs := []string{cfg.SchemasDir, cfg.MigrationsDir, "types"}
			gitkeepDirs := []string{cfg.SchemasDir, cfg.MigrationsDir}
			for _, dir := range dirs {
				if err := os.MkdirAll(dir, DirPerm); err != nil {
					return fmt.Errorf("failed to create %s: %w", dir, err)
				}
				created.AddSuccess(dir + "/")
			}
			// Create .gitkeep to ensure empty directories are tracked by git
			for _, dir := range gitkeepDirs {
				gitkeepPath := filepath.Join(dir, ".gitkeep")
				if _, err := os.Stat(gitkeepPath); os.IsNotExist(err) {
					if err := os.WriteFile(gitkeepPath, []byte{}, FilePerm); err != nil {
						return fmt.Errorf("failed to create %s: %w", gitkeepPath, err)
					}
				}
			}

			// Create alab.yaml if it doesn't exist
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				content := mustReadTemplate("templates/alab.yaml.tmpl")
				if err := os.WriteFile(configFile, []byte(content), FilePerm); err != nil {
					return fmt.Errorf("failed to create config file: %w", err)
				}
				created.AddSuccess(configFile)
			}

			// Create type definition files for IDE autocomplete
			if err := writeTypeDefinitions(); err != nil {
				return fmt.Errorf("failed to create type definitions: %w", err)
			}
			created.AddSuccess("types/*.d.ts")

			// Create tsconfig.json for IDE autocomplete if it doesn't exist
			jsconfigPath := "tsconfig.json"
			if _, err := os.Stat(jsconfigPath); os.IsNotExist(err) {
				jsconfigContent := mustReadTemplate("templates/tsconfig.json.tmpl")
				if err := os.WriteFile(jsconfigPath, []byte(jsconfigContent), FilePerm); err != nil {
					return fmt.Errorf("failed to create tsconfig.json: %w", err)
				}
				created.AddSuccess(jsconfigPath)
			}

			// Copy demo schemas if --demo flag is set
			if withDemo {
				if err := copyDemoSchemas(cfg.SchemasDir, created); err != nil {
					return fmt.Errorf("failed to copy demo schemas: %w", err)
				}
			}

			// Show success panel with created files
			nextSteps := "\nNext steps:\n  1. Edit alab.yaml to configure your database\n"
			if withDemo {
				nextSteps += "  2. Review demo schemas in " + cfg.SchemasDir + "/\n  3. Run 'alab new init' to create your first migration"
			} else {
				nextSteps += "  2. Create schema files in " + cfg.SchemasDir + "/\n  3. Run 'alab new <name>' to create your first migration"
			}

			ui.ShowSuccess(
				TitleProjectInitialized,
				"Created:\n"+created.String()+"\n"+ui.Help(nextSteps),
			)
			return nil
		},
	}

	cmd.Flags().BoolVar(&withDemo, "demo", false, "Include demo schemas (auth.user, auth.role, blog.post)")

	setupCommandHelp(cmd)
	return cmd
}

// copyDemoSchemas copies demo schema files from embedded templates to the project's schemas directory.
func copyDemoSchemas(schemasDir string, created *ui.List) error {
	// Walk through all files in templates/schemas
	return fs.WalkDir(templates, "templates/schemas", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root schemas directory itself
		if path == "templates/schemas" {
			return nil
		}

		// Get relative path from templates/schemas
		relPath := strings.TrimPrefix(path, "templates/schemas/")

		// Create full destination path
		destPath := filepath.Join(schemasDir, relPath)

		if d.IsDir() {
			// Create directory
			if err := os.MkdirAll(destPath, DirPerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
		} else {
			// Read file content from embedded FS
			content, err := templates.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read template file %s: %w", path, err)
			}

			// Write file to destination
			if err := os.WriteFile(destPath, content, FilePerm); err != nil {
				return fmt.Errorf("failed to write file %s: %w", destPath, err)
			}

			created.AddSuccess(destPath)
		}

		return nil
	})
}
