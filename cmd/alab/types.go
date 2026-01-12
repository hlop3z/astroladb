package main

import (
	"os"
	"path/filepath"
)

// Type definition template paths
const (
	templateJSConfig     = "templates/types/tsconfig.json"
	templateIndexDTS     = "templates/types/index.d.ts"
	templateGlobalsDTS   = "templates/types/globals.d.ts"
	templateColumnDTS    = "templates/types/column.d.ts"
	templateSchemaDTS    = "templates/types/schema.d.ts"
	templateMigrationDTS = "templates/types/migration.d.ts"
)

// writeTypeDefinitions creates the TypeScript definition files for IDE autocomplete.
// These are always overwritten to ensure they stay in sync.
func writeTypeDefinitions() error {
	typesDir := "types"
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		return err
	}

	// Map of output file names to their embedded template paths
	files := map[string]string{
		"index.d.ts":     templateIndexDTS,
		"globals.d.ts":   templateGlobalsDTS,
		"column.d.ts":    templateColumnDTS,
		"schema.d.ts":    templateSchemaDTS,
		"migration.d.ts": templateMigrationDTS,
	}

	for name, templatePath := range files {
		content := mustReadTemplate(templatePath)
		path := filepath.Join(typesDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Write tsconfig.json in project root for IDE support
	jsconfigContent := mustReadTemplate(templateJSConfig)
	if err := os.WriteFile("tsconfig.json", []byte(jsconfigContent), 0644); err != nil {
		return err
	}

	return nil
}
