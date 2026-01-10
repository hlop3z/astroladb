package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// initCmd creates the schemas/ and migrations/ directories.
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize project structure (creates schemas/, migrations/, types/ dirs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			dirs := []string{cfg.SchemasDir, cfg.MigrationsDir, "types"}
			for _, dir := range dirs {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create %s: %w", dir, err)
				}
				fmt.Printf("Created %s/\n", dir)
			}

			// Create alab.yaml if it doesn't exist
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				content := `# Alab configuration
database_url: ${DATABASE_URL}
schemas_dir: ./schemas
migrations_dir: ./migrations
`
				if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to create config file: %w", err)
				}
				fmt.Printf("Created %s\n", configFile)
			}

			// Create type definition files for IDE autocomplete
			if err := writeTypeDefinitions(); err != nil {
				return fmt.Errorf("failed to create type definitions: %w", err)
			}
			fmt.Println("Created types/*.d.ts (for IDE autocomplete)")

			// Create jsconfig.json for IDE autocomplete if it doesn't exist
			jsconfigPath := "jsconfig.json"
			if _, err := os.Stat(jsconfigPath); os.IsNotExist(err) {
				jsconfigContent := `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ES2020",
    "moduleResolution": "node",
    "checkJs": false,
    "strict": false,
    "noEmit": true
  },
  "include": [
    "types/**/*.d.ts",
    "schemas/**/*.js",
    "migrations/**/*.js"
  ],
  "exclude": [
    "node_modules",
    ".alab"
  ]
}
`
				if err := os.WriteFile(jsconfigPath, []byte(jsconfigContent), 0644); err != nil {
					return fmt.Errorf("failed to create jsconfig.json: %w", err)
				}
				fmt.Printf("Created %s\n", jsconfigPath)
			}

			fmt.Println("Project initialized successfully!")
			return nil
		},
	}
}
