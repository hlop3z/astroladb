package astroladb

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/engine/diff"
)

func (c *Client) MigrationGenerate(name string) (string, error) {
	c.log("Generating migration: %s", name)

	// Get schema diff against last migration state (not database state)
	ops, err := c.SchemaDiffFromMigrations()
	if err != nil {
		return "", err
	}

	if len(ops) == 0 {
		return "", &SchemaError{
			Message: "no schema changes detected",
		}
	}

	// Ensure migrations directory exists
	if err := os.MkdirAll(c.config.MigrationsDir, 0755); err != nil {
		return "", &MigrationError{
			Operation: "create migrations directory",
			Cause:     err,
		}
	}

	// Determine next revision number
	revision, prevRevision, err := c.nextRevision()
	if err != nil {
		return "", err
	}

	// Generate migration content
	content := c.generateMigrationContent(name, revision, prevRevision, ops)

	// Write migration file
	filename := fmt.Sprintf("%s_%s.js", revision, name)
	path := filepath.Join(c.config.MigrationsDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", &MigrationError{
			Revision:  revision,
			Operation: "write migration file",
			Cause:     err,
		}
	}

	c.log("Generated migration: %s", path)
	return path, nil
}

// DetectRenames analyzes schema diff and returns potential rename candidates.
// These candidates should be confirmed by the user before generating the migration.
func (c *Client) DetectRenames() ([]diff.RenameCandidate, error) {
	ops, err := c.SchemaDiffFromMigrations()
	if err != nil {
		return nil, err
	}
	return diff.DetectRenames(ops), nil
}

// MigrationGenerateWithRenames generates a migration with confirmed renames applied.
// The confirmedRenames should be a subset of candidates returned by DetectRenames.
func (c *Client) MigrationGenerateWithRenames(name string, confirmedRenames []diff.RenameCandidate) (string, error) {
	c.log("Generating migration with renames: %s", name)

	// Get schema diff against last migration state (not database state)
	ops, err := c.SchemaDiffFromMigrations()
	if err != nil {
		return "", err
	}

	if len(ops) == 0 {
		return "", &SchemaError{
			Message: "no schema changes detected",
		}
	}

	// Apply confirmed renames to transform drop+add into rename operations
	ops = diff.ApplyRenames(ops, confirmedRenames)

	// Ensure migrations directory exists
	if err := os.MkdirAll(c.config.MigrationsDir, 0755); err != nil {
		return "", &MigrationError{
			Operation: "create migrations directory",
			Cause:     err,
		}
	}

	// Determine next revision number
	revision, prevRevision, err := c.nextRevision()
	if err != nil {
		return "", err
	}

	// Generate migration content
	content := c.generateMigrationContent(name, revision, prevRevision, ops)

	// Write migration file
	filename := fmt.Sprintf("%s_%s.js", revision, name)
	path := filepath.Join(c.config.MigrationsDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", &MigrationError{
			Revision:  revision,
			Operation: "write migration file",
			Cause:     err,
		}
	}

	c.log("Generated migration: %s", path)
	return path, nil
}

// DetectMissingBackfills finds NOT NULL columns being added without defaults.
// Returns candidates that should be prompted to the user for backfill values.
func (c *Client) DetectMissingBackfills() ([]engine.BackfillCandidate, error) {
	ops, err := c.SchemaDiffFromMigrations()
	if err != nil {
		return nil, err
	}
	return engine.DetectMissingBackfills(ops), nil
}

// MigrationGenerateInteractive generates a migration with renames and backfills applied.
// This is the main entry point for interactive migration generation.
func (c *Client) MigrationGenerateInteractive(name string, renames []diff.RenameCandidate, backfills map[string]string) (string, error) {
	c.log("Generating interactive migration: %s", name)

	// Get schema diff against last migration state (not database state)
	ops, err := c.SchemaDiffFromMigrations()
	if err != nil {
		return "", err
	}

	if len(ops) == 0 {
		return "", &SchemaError{
			Message: "no schema changes detected",
		}
	}

	// Apply renames first
	if len(renames) > 0 {
		ops = diff.ApplyRenames(ops, renames)
	}

	// Apply backfills
	if len(backfills) > 0 {
		ops = engine.ApplyBackfills(ops, backfills)
	}

	// Ensure migrations directory exists
	if err := os.MkdirAll(c.config.MigrationsDir, 0755); err != nil {
		return "", &MigrationError{
			Operation: "create migrations directory",
			Cause:     err,
		}
	}

	// Determine next revision number
	revision, prevRevision, err := c.nextRevision()
	if err != nil {
		return "", err
	}

	// Generate migration content
	content := c.generateMigrationContent(name, revision, prevRevision, ops)

	// Write migration file
	filename := fmt.Sprintf("%s_%s.js", revision, name)
	path := filepath.Join(c.config.MigrationsDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", &MigrationError{
			Revision:  revision,
			Operation: "write migration file",
			Cause:     err,
		}
	}

	c.log("Generated migration: %s", path)
	return path, nil
}

func (c *Client) nextRevision() (string, string, error) {
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return "", "", err
	}

	if len(migrations) == 0 {
		return "001", "", nil
	}

	// Get the last revision
	lastRevision := migrations[len(migrations)-1].Revision

	// Parse as integer and increment (fail fast for determinism)
	num, err := strconv.Atoi(lastRevision)
	if err != nil {
		return "", "", alerr.New(alerr.ErrMigrationFailed, "migration revisions must be sequential numbers").
			With("last_revision", lastRevision).
			With("expected", "numeric revision (e.g., 001, 002, 003)")
	}

	return fmt.Sprintf("%03d", num+1), lastRevision, nil
}
