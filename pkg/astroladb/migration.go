package astroladb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/chain"
	"github.com/hlop3z/astroladb/internal/engine"
)

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	// Revision is the migration identifier (e.g., "001", "20240101120000").
	Revision string

	// Name is the human-readable migration name.
	Name string

	// Status is the current status (pending, applied, missing, modified).
	Status string

	// AppliedAt is when the migration was applied (nil if not applied).
	AppliedAt *time.Time

	// Checksum is the SHA256 hash of the migration file.
	Checksum string
}

// MigrationHistoryEntry represents a migration that has been applied to the database.
// This is returned by MigrationHistory() and contains detailed information about
// each applied migration including execution timing.
type MigrationHistoryEntry struct {
	// Revision is the migration identifier (e.g., "001", "20240101120000").
	Revision string

	// Name is the human-readable migration name.
	Name string

	// AppliedAt is the timestamp when the migration was applied.
	AppliedAt time.Time

	// Checksum is the SHA256 hash of the migration file at time of application.
	Checksum string

	// ExecTimeMs is the execution time in milliseconds (0 if not recorded).
	ExecTimeMs int
}

// MigrationRun applies all pending migrations to the database.
// By default, it applies all pending migrations. Use options to customize behavior.
//
// Options:
//   - DryRun(): Show SQL without executing
//   - Target(revision): Stop at a specific revision
//   - Steps(n): Apply at most n migrations
//
// Example:
//
//	// Apply all pending migrations
//	err := client.MigrationRun()
//
//	// Dry-run to see SQL
//	err := client.MigrationRun(astroladb.DryRun())
//
//	// Apply up to 3 migrations
//	err := client.MigrationRun(astroladb.Steps(3))
func (c *Client) MigrationRun(opts ...MigrationOption) error {
	cfg := applyMigrationOptions(opts)
	ctx, cancel := c.context()
	defer cancel()

	c.log("Running migrations from %s", c.config.MigrationsDir)

	// Compute migration chain from files
	migrationChain, err := chain.ComputeFromDir(c.config.MigrationsDir)
	if err != nil {
		return &MigrationError{
			Operation: "compute migration chain",
			Cause:     err,
		}
	}

	if len(migrationChain.Links) == 0 {
		c.log("No migrations found")
		return nil
	}

	// Get applied migrations from database
	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		// Table might not exist yet - try to create it
		if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
			return &MigrationError{
				Operation: "ensure migration table",
				Cause:     err,
			}
		}
		applied = nil
	}

	// Convert applied migrations to chain format for verification
	appliedChain := make([]chain.AppliedMigration, len(applied))
	for i, a := range applied {
		appliedChain[i] = chain.AppliedMigration{
			Revision:  a.Revision,
			Checksum:  a.Checksum,
			AppliedAt: a.AppliedAt.Format(time.RFC3339),
		}
	}

	// Verify chain integrity (unless force mode)
	if !cfg.Force {
		result := migrationChain.Verify(appliedChain)
		if !result.Valid {
			return &ChainError{
				Result: result,
			}
		}
	}

	// Load migrations with operations for execution
	migrations, err := c.loadMigrationFilesWithChain(migrationChain)
	if err != nil {
		return err
	}

	// Create execution plan
	plan, err := engine.PlanMigrations(migrations, applied, cfg.Target, engine.Up)
	if err != nil {
		return &MigrationError{
			Operation: "plan migrations",
			Cause:     err,
		}
	}

	// Apply steps limit
	if cfg.Steps > 0 && cfg.Steps < len(plan.Migrations) {
		plan.Migrations = plan.Migrations[:cfg.Steps]
	}

	if plan.IsEmpty() {
		c.log("No pending migrations")
		return nil
	}

	c.log("Found %d pending migrations", len(plan.Migrations))

	// Handle dry-run
	if cfg.DryRun {
		return c.dryRunMigrations(ctx, plan, cfg.Output)
	}

	// Execute the plan (with or without locking)
	var execErr error
	if cfg.SkipLock {
		execErr = c.runner.Run(ctx, plan)
	} else {
		execErr = c.runner.RunWithLock(ctx, plan, cfg.LockTimeout)
	}

	if execErr != nil {
		return &MigrationError{
			Operation: "execute migrations",
			Cause:     execErr,
		}
	}

	c.log("Applied %d migrations", len(plan.Migrations))
	return nil
}

// MigrationRollback reverts the last n migrations.
// By default, it rolls back 1 migration. Use options to customize.
//
// Options:
//   - DryRun(): Show SQL without executing
//   - Steps(n): Rollback n migrations
//   - Target(revision): Rollback to a specific revision (exclusive)
//
// Example:
//
//	// Rollback last migration
//	err := client.MigrationRollback(1)
//
//	// Rollback last 3 migrations
//	err := client.MigrationRollback(3)
//
//	// Dry-run
//	err := client.MigrationRollback(1, astroladb.DryRun())
func (c *Client) MigrationRollback(steps int, opts ...MigrationOption) error {
	cfg := applyMigrationOptions(opts)
	ctx, cancel := c.context()
	defer cancel()

	if steps <= 0 {
		steps = 1
	}

	c.log("Rolling back %d migrations", steps)

	// Load all migrations
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return err
	}

	// Get applied migrations
	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		return &MigrationError{
			Operation: "get applied migrations",
			Cause:     err,
		}
	}

	if len(applied) == 0 {
		c.log("No migrations to rollback")
		return nil
	}

	// Create rollback plan
	plan, err := engine.PlanMigrations(migrations, applied, cfg.Target, engine.Down)
	if err != nil {
		return &MigrationError{
			Operation: "plan rollback",
			Cause:     err,
		}
	}

	// Apply steps limit
	if steps < len(plan.Migrations) {
		plan.Migrations = plan.Migrations[:steps]
	}

	if plan.IsEmpty() {
		c.log("No migrations to rollback")
		return nil
	}

	c.log("Rolling back %d migrations", len(plan.Migrations))

	// Handle dry-run
	if cfg.DryRun {
		return c.dryRunMigrations(ctx, plan, cfg.Output)
	}

	// Execute the rollback (with or without locking)
	var execErr error
	if cfg.SkipLock {
		execErr = c.runner.Run(ctx, plan)
	} else {
		execErr = c.runner.RunWithLock(ctx, plan, cfg.LockTimeout)
	}

	if execErr != nil {
		return &MigrationError{
			Operation: "execute rollback",
			Cause:     execErr,
		}
	}

	c.log("Rolled back %d migrations", len(plan.Migrations))
	return nil
}

// MigrationStatus returns the status of all migrations.
// Each migration is marked as pending, applied, missing, or modified.
func (c *Client) MigrationStatus() ([]MigrationStatus, error) {
	ctx, cancel := c.context()
	defer cancel()

	// Load all migrations
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, err
	}

	// Ensure version table exists
	if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
		return nil, &MigrationError{
			Operation: "ensure migration table",
			Cause:     err,
		}
	}

	// Get applied migrations
	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		return nil, &MigrationError{
			Operation: "get applied migrations",
			Cause:     err,
		}
	}

	// Get status from engine
	statuses := engine.GetStatus(migrations, applied)

	// Convert to public type
	result := make([]MigrationStatus, len(statuses))
	for i, s := range statuses {
		result[i] = MigrationStatus{
			Revision: s.Revision,
			Name:     s.Name,
			Status:   s.Status.String(),
			Checksum: s.Checksum,
		}
		if s.AppliedAt != nil {
			t, _ := time.Parse("2006-01-02 15:04:05", *s.AppliedAt)
			result[i].AppliedAt = &t
		}
	}

	return result, nil
}

// MigrationHistory returns the history of applied migrations from the database.
// Unlike MigrationStatus which compares files to DB, this returns only what's in the DB.
// Results are ordered by revision (oldest first by default).
//
// This is useful for auditing, debugging, and CI/CD pipelines that need to know
// exactly what migrations have been applied to a database.
//
// Example:
//
//	history, err := client.MigrationHistory()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, entry := range history {
//	    fmt.Printf("%s: %s (applied %s, took %dms)\n",
//	        entry.Revision, entry.Name, entry.AppliedAt, entry.ExecTimeMs)
//	}
func (c *Client) MigrationHistory() ([]MigrationHistoryEntry, error) {
	ctx, cancel := c.context()
	defer cancel()

	// Ensure version table exists
	if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
		return nil, &MigrationError{
			Operation: "ensure migration table",
			Cause:     err,
		}
	}

	// Get applied migrations directly from the database
	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		return nil, &MigrationError{
			Operation: "get applied migrations",
			Cause:     err,
		}
	}

	if len(applied) == 0 {
		return []MigrationHistoryEntry{}, nil
	}

	// Load migration files to get names (revision -> name mapping)
	migrations, _ := c.loadMigrationFiles()
	nameMap := make(map[string]string)
	for _, m := range migrations {
		nameMap[m.Revision] = m.Name
	}

	// Convert to history entries
	result := make([]MigrationHistoryEntry, len(applied))
	for i, a := range applied {
		name := nameMap[a.Revision]
		if name == "" {
			// If file doesn't exist, use revision as name
			name = "(file missing)"
		}

		result[i] = MigrationHistoryEntry{
			Revision:   a.Revision,
			Name:       name,
			AppliedAt:  a.AppliedAt,
			Checksum:   a.Checksum,
			ExecTimeMs: a.ExecTimeMs,
		}
	}

	return result, nil
}

// VerifyChain verifies the integrity of the migration chain.
// Returns the verification result with details about any issues found.
func (c *Client) VerifyChain() (*chain.VerificationResult, error) {
	ctx, cancel := c.context()
	defer cancel()

	// Compute chain from files
	migrationChain, err := chain.ComputeFromDir(c.config.MigrationsDir)
	if err != nil {
		return nil, &MigrationError{
			Operation: "compute migration chain",
			Cause:     err,
		}
	}

	// Get applied migrations from database
	if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
		return nil, &MigrationError{
			Operation: "ensure migration table",
			Cause:     err,
		}
	}

	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		return nil, &MigrationError{
			Operation: "get applied migrations",
			Cause:     err,
		}
	}

	// Convert to chain format
	appliedChain := make([]chain.AppliedMigration, len(applied))
	for i, a := range applied {
		appliedChain[i] = chain.AppliedMigration{
			Revision:  a.Revision,
			Checksum:  a.Checksum,
			AppliedAt: a.AppliedAt.Format(time.RFC3339),
		}
	}

	return migrationChain.Verify(appliedChain), nil
}

// GetChain returns the computed migration chain from files.
// This is useful for displaying chain status.
func (c *Client) GetChain() (*chain.Chain, error) {
	return chain.ComputeFromDir(c.config.MigrationsDir)
}

// MigrationPlan returns the execution plan for pending migrations.
// This is useful for previewing what would be applied.
func (c *Client) MigrationPlan() (*engine.Plan, error) {
	ctx, cancel := c.context()
	defer cancel()

	// Load all migrations
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		// Table might not exist
		applied = nil
	}

	// Create execution plan
	return engine.PlanMigrations(migrations, applied, "", engine.Up)
}

// -----------------------------------------------------------------------------
// Migration Lock Management
// -----------------------------------------------------------------------------

// MigrationLockInfo contains information about the current migration lock state.
type MigrationLockInfo struct {
	// Locked indicates whether the lock is currently held.
	Locked bool

	// LockedAt is when the lock was acquired (nil if not locked).
	LockedAt *time.Time

	// LockedBy identifies which process holds the lock (hostname:pid).
	LockedBy string
}

// MigrationLockStatus returns the current state of the migration lock.
// Use this to check if another process is running migrations.
func (c *Client) MigrationLockStatus() (*MigrationLockInfo, error) {
	ctx, cancel := c.context()
	defer cancel()

	// Ensure lock table exists
	if err := c.runner.VersionManager().EnsureLockTable(ctx); err != nil {
		return nil, &MigrationError{
			Operation: "ensure lock table",
			Cause:     err,
		}
	}

	info, err := c.runner.VersionManager().GetLockInfo(ctx)
	if err != nil {
		return nil, &MigrationError{
			Operation: "get lock info",
			Cause:     err,
		}
	}

	if info == nil {
		return &MigrationLockInfo{Locked: false}, nil
	}

	return &MigrationLockInfo{
		Locked:   info.Locked,
		LockedAt: info.LockedAt,
		LockedBy: info.LockedBy,
	}, nil
}

// MigrationReleaseLock forcefully releases a stuck migration lock.
// Use this only to recover from stuck locks when a process crashed
// while holding the lock.
//
// WARNING: Only use this when you are certain no other migration is running.
// Forcefully releasing an active lock can cause concurrent migrations.
func (c *Client) MigrationReleaseLock() error {
	ctx, cancel := c.context()
	defer cancel()

	if err := c.runner.VersionManager().ForceReleaseLock(ctx); err != nil {
		return &MigrationError{
			Operation: "release lock",
			Cause:     err,
		}
	}

	c.log("Migration lock released")
	return nil
}

// LintPendingMigrations checks pending migrations for destructive operations.
// Returns a list of warning messages if any destructive operations are found.
// This is used by the migrate command to warn users before applying migrations.
func (c *Client) LintPendingMigrations() ([]string, error) {
	// Get the diff of what would be applied
	ops, err := c.SchemaDiff()
	if err != nil {
		// If we can't get the diff, just return no warnings
		// (the migration will fail on its own if there's an issue)
		return nil, nil
	}

	warnings := engine.LintOperations(ops)
	if len(warnings) == 0 {
		return nil, nil
	}

	// Convert to simple string messages
	messages := make([]string, len(warnings))
	for i, w := range warnings {
		messages[i] = w.Message
	}
	return messages, nil
}

// MigrationGenerate generates a new migration file based on schema changes.
// The name should be descriptive (e.g., "add_users_table", "add_email_to_users").
// Returns the path to the generated migration file.
//
// Example:
//
//	path, err := client.MigrationGenerate("add_users_table")
//	// Creates: migrations/001_add_users_table.js
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
	revision, err := c.nextRevision()
	if err != nil {
		return "", err
	}

	// Generate migration content
	content := c.generateMigrationContent(name, ops)

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
func (c *Client) DetectRenames() ([]engine.RenameCandidate, error) {
	ops, err := c.SchemaDiffFromMigrations()
	if err != nil {
		return nil, err
	}
	return engine.DetectRenames(ops), nil
}

// MigrationGenerateWithRenames generates a migration with confirmed renames applied.
// The confirmedRenames should be a subset of candidates returned by DetectRenames.
func (c *Client) MigrationGenerateWithRenames(name string, confirmedRenames []engine.RenameCandidate) (string, error) {
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
	ops = engine.ApplyRenames(ops, confirmedRenames)

	// Ensure migrations directory exists
	if err := os.MkdirAll(c.config.MigrationsDir, 0755); err != nil {
		return "", &MigrationError{
			Operation: "create migrations directory",
			Cause:     err,
		}
	}

	// Determine next revision number
	revision, err := c.nextRevision()
	if err != nil {
		return "", err
	}

	// Generate migration content
	content := c.generateMigrationContent(name, ops)

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
func (c *Client) MigrationGenerateInteractive(name string, renames []engine.RenameCandidate, backfills map[string]string) (string, error) {
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
		ops = engine.ApplyRenames(ops, renames)
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
	revision, err := c.nextRevision()
	if err != nil {
		return "", err
	}

	// Generate migration content
	content := c.generateMigrationContent(name, ops)

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

// loadMigrationFiles loads migration files from the migrations directory.
// Computes the migration chain with checksums for integrity verification.
func (c *Client) loadMigrationFiles() ([]engine.Migration, error) {
	// Compute the chain first
	migrationChain, err := chain.ComputeFromDir(c.config.MigrationsDir)
	if err != nil {
		return nil, &MigrationError{
			Operation: "compute migration chain",
			Cause:     err,
		}
	}

	return c.loadMigrationFilesWithChain(migrationChain)
}

// loadMigrationFilesWithChain loads migrations using pre-computed chain checksums.
// The chain checksums are used for integrity verification.
func (c *Client) loadMigrationFilesWithChain(migrationChain *chain.Chain) ([]engine.Migration, error) {
	if len(migrationChain.Links) == 0 {
		return nil, nil
	}

	var migrations []engine.Migration
	for _, link := range migrationChain.Links {
		path := filepath.Join(c.config.MigrationsDir, link.Filename)

		// Evaluate the migration to get operations
		ops, err := c.sandbox.RunFile(path)
		if err != nil {
			// If evaluation fails, still include the migration with empty ops
			// The error will be caught when running
			migrations = append(migrations, engine.Migration{
				Revision: link.Revision,
				Name:     link.Name,
				Path:     path,
				Checksum: link.Checksum, // Use chain checksum!
			})
			continue
		}

		migrations = append(migrations, engine.Migration{
			Revision:   link.Revision,
			Name:       link.Name,
			Path:       path,
			Checksum:   link.Checksum, // Use chain checksum!
			Operations: ops,
		})
	}

	return migrations, nil
}

// nextRevision determines the next migration revision number.
func (c *Client) nextRevision() (string, error) {
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return "", err
	}

	if len(migrations) == 0 {
		return "001", nil
	}

	// Get the last revision
	lastRevision := migrations[len(migrations)-1].Revision

	// Parse as integer and increment (fail fast for determinism)
	num, err := strconv.Atoi(lastRevision)
	if err != nil {
		return "", alerr.New(alerr.ErrMigrationFailed, "migration revisions must be sequential numbers").
			With("last_revision", lastRevision).
			With("expected", "numeric revision (e.g., 001, 002, 003)")
	}

	return fmt.Sprintf("%03d", num+1), nil
}

// generateMigrationContent generates the JavaScript content for a migration.
func (c *Client) generateMigrationContent(name string, ops []ast.Operation) string {
	var sb strings.Builder

	sb.WriteString("// Migration: ")
	sb.WriteString(name)
	sb.WriteString("\n\n")

	// Write migration wrapper with up function
	sb.WriteString("export default migration({\n")
	sb.WriteString("  up(m) {\n")

	// Track operation sections for organizing comments
	var lastOpType string
	hasIndexes := false

	// First pass: check if we have standalone indexes
	for _, op := range ops {
		if _, ok := op.(*ast.CreateIndex); ok {
			hasIndexes = true
			break
		}
	}

	for _, op := range ops {
		currentOpType := getOperationType(op)

		// Add section headers when switching operation types
		if currentOpType != lastOpType && currentOpType != "" {
			if lastOpType != "" {
				sb.WriteString("\n")
			}
			switch currentOpType {
			case "table":
				// Don't add a section header for tables, we'll comment each table individually
			case "index":
				if hasIndexes {
					sb.WriteString("    // Indexes\n")
				}
			case "column":
				sb.WriteString("    // Columns\n")
			case "foreignkey":
				sb.WriteString("    // Foreign Keys\n")
			}
			lastOpType = currentOpType
		}

		switch o := op.(type) {
		case *ast.CreateTable:
			c.writeCreateTable(&sb, o)
		case *ast.DropTable:
			c.writeDropTable(&sb, o)
		case *ast.RenameTable:
			c.writeRenameTable(&sb, o)
		case *ast.AddColumn:
			c.writeAddColumn(&sb, o)
		case *ast.DropColumn:
			c.writeDropColumn(&sb, o)
		case *ast.RenameColumn:
			c.writeRenameColumn(&sb, o)
		case *ast.AlterColumn:
			c.writeAlterColumn(&sb, o)
		case *ast.CreateIndex:
			c.writeCreateIndex(&sb, o)
		case *ast.DropIndex:
			c.writeDropIndex(&sb, o)
		case *ast.AddForeignKey:
			c.writeAddForeignKey(&sb, o)
		case *ast.DropForeignKey:
			c.writeDropForeignKey(&sb, o)
		default:
			sb.WriteString("    // Unsupported operation\n")
		}
	}

	sb.WriteString("  },\n\n")

	// Write down function (reverse operations)
	sb.WriteString("  down(m) {\n")
	for i := len(ops) - 1; i >= 0; i-- {
		switch o := ops[i].(type) {
		case *ast.CreateTable:
			ref := o.Name
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Name
			}
			sb.WriteString(fmt.Sprintf("    m.drop_table(\"%s\")\n", ref))
		case *ast.DropTable:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate table %s (data cannot be recovered automatically)\n", o.Name))
		case *ast.RenameTable:
			oldRef := o.OldName
			newRef := o.NewName
			if o.Namespace != "" {
				oldRef = o.Namespace + "." + o.OldName
				newRef = o.Namespace + "." + o.NewName
			}
			sb.WriteString(fmt.Sprintf("    m.rename_table(\"%s\", \"%s\")\n", newRef, oldRef))
		case *ast.AddColumn:
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    m.drop_column(\"%s\", \"%s\")\n", ref, o.Column.Name))
		case *ast.DropColumn:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate column %s (data cannot be recovered automatically)\n", o.Name))
		case *ast.RenameColumn:
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    m.rename_column(\"%s\", \"%s\", \"%s\")\n", ref, o.NewName, o.OldName))
		case *ast.AlterColumn:
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    // Manual: Reverse alter_column on %s.%s (original state unknown)\n", ref, o.Name))
		case *ast.CreateIndex:
			indexName := o.Name
			if indexName == "" {
				indexName = fmt.Sprintf("idx_%s_%s", o.Table_, strings.Join(o.Columns, "_"))
			}
			sb.WriteString(fmt.Sprintf("    m.drop_index(\"%s\")\n", indexName))
		case *ast.DropIndex:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate index %s (original definition unknown)\n", o.Name))
		case *ast.AddForeignKey:
			fkName := o.Name
			if fkName == "" {
				fkName = fmt.Sprintf("fk_%s_%s", o.Table_, strings.Join(o.Columns, "_"))
			}
			ref := o.Table_
			if o.Namespace != "" {
				ref = o.Namespace + "." + o.Table_
			}
			sb.WriteString(fmt.Sprintf("    m.drop_foreign_key(\"%s\", \"%s\")\n", ref, fkName))
		case *ast.DropForeignKey:
			sb.WriteString(fmt.Sprintf("    // Manual: Recreate foreign key %s (original definition unknown)\n", o.Name))
		}
	}
	sb.WriteString("  }\n")
	sb.WriteString("})\n")

	// Beautify the generated JavaScript code
	return beautifyJavaScript(sb.String())
}

// getOperationType returns the operation type category for section grouping.
func getOperationType(op ast.Operation) string {
	switch op.(type) {
	case *ast.CreateTable, *ast.DropTable, *ast.RenameTable:
		return "table"
	case *ast.CreateIndex, *ast.DropIndex:
		return "index"
	case *ast.AddColumn, *ast.DropColumn, *ast.RenameColumn, *ast.AlterColumn:
		return "column"
	case *ast.AddForeignKey, *ast.DropForeignKey:
		return "foreignkey"
	default:
		return ""
	}
}

// beautifyJavaScript formats JavaScript code using Prettier for professional-quality formatting.
// It tries to use npx prettier, and falls back to unformatted code if Prettier is unavailable.
func beautifyJavaScript(code string) string {
	// Try to format with prettier via npx
	cmd := exec.Command("npx", "--yes", "prettier@latest",
		"--parser", "babel",
		"--tab-width", "2",
		"--no-semi",
		"--single-quote=false",
		"--trailing-comma", "none",
		"--arrow-parens", "avoid",
		"--print-width", "80")

	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(code)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Notify user that Prettier is not available
		fmt.Fprintln(os.Stderr, "⚠️  Warning: Prettier not found. Migration generated without formatting.")
		fmt.Fprintln(os.Stderr, "   To enable beautiful formatting, install Node.js from: https://nodejs.org")
		fmt.Fprintln(os.Stderr, "   Or run: npm install -g prettier")
		fmt.Fprintln(os.Stderr)

		// Return unformatted code - migration will still work
		return code
	}

	return stdout.String()
}

// writeCreateTable writes a create_table DSL call.
func (c *Client) writeCreateTable(sb *strings.Builder, op *ast.CreateTable) {
	ref := op.Name
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Name
	}

	// Add table comment
	sb.WriteString(fmt.Sprintf("\n    // Table: %s\n", ref))
	sb.WriteString(fmt.Sprintf("    m.create_table(\"%s\", t => {\n", ref))

	// Sort columns: id first, timestamps last, others alphabetically
	sortedColumns := c.sortColumnsForDatabase(op.Columns)

	// Track which columns to skip (for timestamps detection)
	skip := make(map[int]bool)
	timestampIndex := -1

	// Detect timestamps() pattern: created_at and updated_at datetime columns
	for i := 0; i < len(sortedColumns)-1; i++ {
		col1 := sortedColumns[i]
		col2 := sortedColumns[i+1]
		if col1.Name == "created_at" && col1.Type == "datetime" &&
			col2.Name == "updated_at" && col2.Type == "datetime" {
			skip[i] = true
			skip[i+1] = true
			timestampIndex = i
		}
	}

	// Write columns
	for i, col := range sortedColumns {
		if skip[i] {
			continue
		}
		c.writeColumn(sb, col)
	}

	// Write timestamps() at the end if detected
	if timestampIndex >= 0 {
		sb.WriteString("    t.timestamps()\n")
	}

	sb.WriteString("  })\n")

	// Output embedded indexes as separate create_index calls
	for _, idx := range op.Indexes {
		c.writeCreateIndex(sb, &ast.CreateIndex{
			TableRef: ast.TableRef{
				Namespace: op.Namespace,
				Table_:    op.Name,
			},
			Name:        idx.Name,
			Columns:     idx.Columns,
			Unique:      idx.Unique,
			IfNotExists: true,
		})
	}
}

// typeToDSLMethod converts internal type names to DSL method names.
// Some internal types have different names than their DSL methods.
func typeToDSLMethod(internalType string) string {
	switch internalType {
	case "datetime":
		return "datetime"
	default:
		return internalType
	}
}

// writeColumn writes a column definition.
func (c *Client) writeColumn(sb *strings.Builder, col *ast.ColumnDef) {
	// Handle special column types

	// Detect t.id() pattern: uuid primary key named "id"
	if col.Type == "uuid" && col.Name == "id" && col.PrimaryKey {
		sb.WriteString("    t.id()\n")
		return
	}

	// Detect belongs_to pattern: uuid column with a Reference
	if col.Reference != nil {
		c.writeBelongsTo(sb, col)
		return
	}

	// Build the column call
	var call strings.Builder
	dslMethod := typeToDSLMethod(col.Type)
	call.WriteString(fmt.Sprintf("    t.%s(\"%s\"", dslMethod, col.Name))

	// Add type arguments
	for _, arg := range col.TypeArgs {
		switch v := arg.(type) {
		case int:
			call.WriteString(fmt.Sprintf(", %d", v))
		case float64:
			call.WriteString(fmt.Sprintf(", %v", v))
		case string:
			call.WriteString(fmt.Sprintf(", \"%s\"", v))
		case []string:
			// Enum values array
			quoted := make([]string, len(v))
			for i, s := range v {
				quoted[i] = fmt.Sprintf("\"%s\"", s)
			}
			call.WriteString(fmt.Sprintf(", [%s]", strings.Join(quoted, ", ")))
		case []any:
			// Generic array (e.g., enum values passed as []any)
			quoted := make([]string, len(v))
			for i, s := range v {
				quoted[i] = fmt.Sprintf("\"%v\"", s)
			}
			call.WriteString(fmt.Sprintf(", [%s]", strings.Join(quoted, ", ")))
		}
	}
	call.WriteString(")")

	// Add modifiers
	if col.Nullable {
		call.WriteString(".optional()")
	}
	if col.Unique {
		call.WriteString(".unique()")
	}
	// Handle default values (same internal type handling as in writeAddColumn)
	if col.DefaultSet {
		switch v := col.Default.(type) {
		case bool:
			call.WriteString(fmt.Sprintf(".default(%t)", v))
		case int:
			call.WriteString(fmt.Sprintf(".default(%d)", v))
		case int64: // Internal: JavaScript numbers from Goja
			call.WriteString(fmt.Sprintf(".default(%d)", v))
		case float64: // Internal: JavaScript numbers from Goja
			// Check if it's actually an integer (write without decimal point)
			if v == float64(int64(v)) {
				call.WriteString(fmt.Sprintf(".default(%d)", int64(v)))
			} else {
				call.WriteString(fmt.Sprintf(".default(%v)", v))
			}
		case string:
			call.WriteString(fmt.Sprintf(".default(\"%s\")", v))
		case *ast.SQLExpr:
			call.WriteString(fmt.Sprintf(".default(sql(\"%s\"))", v.Expr))
		}
	}

	call.WriteString("\n")
	sb.WriteString(call.String())
}

// writeBelongsTo writes a belongs_to relationship column.
func (c *Client) writeBelongsTo(sb *strings.Builder, col *ast.ColumnDef) {
	var call strings.Builder
	call.WriteString(fmt.Sprintf("    t.belongs_to(\"%s\")", col.Reference.Table))

	// Determine if an alias is needed by checking if column name differs from default
	// Default column name for belongs_to("ns.table") is "table_id"
	expectedColName := extractTableName(col.Reference.Table) + "_id"
	if col.Name != expectedColName {
		// Extract alias from column name (remove _id suffix)
		alias := strings.TrimSuffix(col.Name, "_id")
		call.WriteString(fmt.Sprintf(".as(\"%s\")", alias))
	}

	// Add modifiers
	if col.Nullable {
		call.WriteString(".optional()")
	}
	if col.Reference.OnDelete != "" {
		call.WriteString(fmt.Sprintf(".on_delete(\"%s\")", strings.ToLower(col.Reference.OnDelete)))
	}
	if col.Reference.OnUpdate != "" {
		call.WriteString(fmt.Sprintf(".on_update(\"%s\")", strings.ToLower(col.Reference.OnUpdate)))
	}

	call.WriteString("\n")
	sb.WriteString(call.String())
}

// extractTableName extracts the table name from a reference like "ns.table" or "table".
func extractTableName(ref string) string {
	parts := strings.Split(ref, ".")
	return parts[len(parts)-1]
}

// writeDropTable writes a drop_table DSL call.
func (c *Client) writeDropTable(sb *strings.Builder, op *ast.DropTable) {
	ref := op.Name
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Name
	}
	sb.WriteString(fmt.Sprintf("    m.drop_table(\"%s\")\n", ref))
}

// writeRenameTable writes a rename_table DSL call.
func (c *Client) writeRenameTable(sb *strings.Builder, op *ast.RenameTable) {
	oldRef := op.OldName
	newRef := op.NewName
	if op.Namespace != "" {
		oldRef = op.Namespace + "." + op.OldName
		newRef = op.Namespace + "." + op.NewName
	}
	sb.WriteString(fmt.Sprintf("    m.rename_table(\"%s\", \"%s\")\n", oldRef, newRef))
}

// writeRenameColumn writes a rename_column DSL call.
func (c *Client) writeRenameColumn(sb *strings.Builder, op *ast.RenameColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}
	sb.WriteString(fmt.Sprintf("    m.rename_column(\"%s\", \"%s\", \"%s\")\n", ref, op.OldName, op.NewName))
}

// writeAddColumn writes an add_column DSL call.
func (c *Client) writeAddColumn(sb *strings.Builder, op *ast.AddColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	dslMethod := typeToDSLMethod(op.Column.Type)
	sb.WriteString(fmt.Sprintf("    m.add_column(\"%s\", c => c.%s(\"%s\"", ref, dslMethod, op.Column.Name))

	// Add type args
	for _, arg := range op.Column.TypeArgs {
		switch v := arg.(type) {
		case int:
			sb.WriteString(fmt.Sprintf(", %d", v))
		case float64:
			sb.WriteString(fmt.Sprintf(", %v", v))
		case string:
			sb.WriteString(fmt.Sprintf(", \"%s\"", v))
		}
	}
	sb.WriteString(")")

	// Add modifiers
	if op.Column.Nullable {
		sb.WriteString(".optional()")
	}
	if op.Column.Unique {
		sb.WriteString(".unique()")
	}
	if op.Column.DefaultSet {
		// IMPORTANT: int64 and float64 here are INTERNAL Go types from Goja's JavaScript engine.
		// Users NEVER write these types in schemas or migrations (see CONTRIBUTING.md).
		// The actual schema DSL only exposes JS-safe types: integer() for 32-bit, decimal() for precision.
		// When JavaScript passes default values like `0` or `4.5` through Goja, they arrive as
		// int64/float64 in Go. We convert them back to JavaScript literals for the migration file.
		switch v := op.Column.Default.(type) {
		case bool:
			sb.WriteString(fmt.Sprintf(".default(%t)", v))
		case int:
			sb.WriteString(fmt.Sprintf(".default(%d)", v))
		case int64: // Internal: JavaScript numbers come through Goja as int64
			sb.WriteString(fmt.Sprintf(".default(%d)", v))
		case float64: // Internal: JavaScript numbers come through Goja as float64
			// Check if it's actually an integer (write without decimal point)
			if v == float64(int64(v)) {
				sb.WriteString(fmt.Sprintf(".default(%d)", int64(v)))
			} else {
				sb.WriteString(fmt.Sprintf(".default(%v)", v))
			}
		case string:
			sb.WriteString(fmt.Sprintf(".default(\"%s\")", v))
		case *ast.SQLExpr: // SQL expressions like NOW(), CURRENT_TIMESTAMP, etc.
			sb.WriteString(fmt.Sprintf(".default(sql(\"%s\"))", v.Expr))
		}
	}
	// Add backfill for existing rows
	// Same internal type handling as default values (see comment above)
	if op.Column.BackfillSet {
		switch v := op.Column.Backfill.(type) {
		case bool:
			sb.WriteString(fmt.Sprintf(".backfill(%t)", v))
		case int:
			sb.WriteString(fmt.Sprintf(".backfill(%d)", v))
		case int64: // Internal: JavaScript numbers come through Goja as int64
			sb.WriteString(fmt.Sprintf(".backfill(%d)", v))
		case float64: // Internal: JavaScript numbers come through Goja as float64
			if v == float64(int64(v)) {
				sb.WriteString(fmt.Sprintf(".backfill(%d)", int64(v)))
			} else {
				sb.WriteString(fmt.Sprintf(".backfill(%v)", v))
			}
		case string:
			// Check if it's a sql() expression
			if strings.HasPrefix(v, "sql(") {
				sb.WriteString(fmt.Sprintf(".backfill(%s)", v))
			} else {
				sb.WriteString(fmt.Sprintf(".backfill(\"%s\")", v))
			}
		case *ast.SQLExpr:
			sb.WriteString(fmt.Sprintf(".backfill(sql(\"%s\"))", v.Expr))
		}
	}

	sb.WriteString(")\n")
}

// writeDropColumn writes a drop_column DSL call.
func (c *Client) writeDropColumn(sb *strings.Builder, op *ast.DropColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}
	sb.WriteString(fmt.Sprintf("    m.drop_column(\"%s\", \"%s\")\n", ref, op.Name))
}

// writeCreateIndex writes a create_index DSL call.
func (c *Client) writeCreateIndex(sb *strings.Builder, op *ast.CreateIndex) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	cols := make([]string, len(op.Columns))
	for i, col := range op.Columns {
		cols[i] = fmt.Sprintf("\"%s\"", col)
	}

	sb.WriteString(fmt.Sprintf("    // Index: %s\n", ref))
	sb.WriteString(fmt.Sprintf("    m.create_index(\"%s\", [%s]", ref, strings.Join(cols, ", ")))

	if op.Unique || op.Name != "" {
		sb.WriteString(", {")
		parts := []string{}
		if op.Unique {
			parts = append(parts, "unique: true")
		}
		if op.Name != "" {
			parts = append(parts, fmt.Sprintf("name: \"%s\"", op.Name))
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("}")
	}

	sb.WriteString(")\n")
}

// writeDropIndex writes a drop_index DSL call.
func (c *Client) writeDropIndex(sb *strings.Builder, op *ast.DropIndex) {
	sb.WriteString(fmt.Sprintf("    m.drop_index(\"%s\")\n", op.Name))
}

// writeAlterColumn writes an alter_column DSL call.
func (c *Client) writeAlterColumn(sb *strings.Builder, op *ast.AlterColumn) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	sb.WriteString(fmt.Sprintf("    m.alter_column(\"%s\", \"%s\", c => c", ref, op.Name))

	if op.NewType != "" {
		dslMethod := typeToDSLMethod(op.NewType)
		sb.WriteString(fmt.Sprintf(".set_type(\"%s\"", dslMethod))
		for _, arg := range op.NewTypeArgs {
			switch v := arg.(type) {
			case int:
				sb.WriteString(fmt.Sprintf(", %d", v))
			case float64:
				sb.WriteString(fmt.Sprintf(", %v", v))
			}
		}
		sb.WriteString(")")
	}
	if op.SetNullable != nil {
		if *op.SetNullable {
			sb.WriteString(".set_nullable()")
		} else {
			sb.WriteString(".set_not_null()")
		}
	}
	if op.DropDefault {
		sb.WriteString(".drop_default()")
	} else if op.SetDefault != nil {
		switch v := op.SetDefault.(type) {
		case bool:
			sb.WriteString(fmt.Sprintf(".set_default(%t)", v))
		case int:
			sb.WriteString(fmt.Sprintf(".set_default(%d)", v))
		case float64:
			sb.WriteString(fmt.Sprintf(".set_default(%v)", v))
		case string:
			sb.WriteString(fmt.Sprintf(".set_default(\"%s\")", v))
		}
	}
	if op.ServerDefault != "" {
		sb.WriteString(fmt.Sprintf(".set_server_default(\"%s\")", op.ServerDefault))
	}

	sb.WriteString(")\n")
}

// writeAddForeignKey writes an add_foreign_key DSL call.
func (c *Client) writeAddForeignKey(sb *strings.Builder, op *ast.AddForeignKey) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}

	// Format columns
	cols := make([]string, len(op.Columns))
	for i, col := range op.Columns {
		cols[i] = fmt.Sprintf("\"%s\"", col)
	}

	// Format ref columns
	refCols := make([]string, len(op.RefColumns))
	for i, col := range op.RefColumns {
		refCols[i] = fmt.Sprintf("\"%s\"", col)
	}

	sb.WriteString(fmt.Sprintf("    m.add_foreign_key(\"%s\", [%s], \"%s\", [%s]",
		ref, strings.Join(cols, ", "), op.RefTable, strings.Join(refCols, ", ")))

	// Add options if present
	opts := []string{}
	if op.Name != "" {
		opts = append(opts, fmt.Sprintf("name: \"%s\"", op.Name))
	}
	if op.OnDelete != "" {
		opts = append(opts, fmt.Sprintf("on_delete: \"%s\"", strings.ToLower(op.OnDelete)))
	}
	if op.OnUpdate != "" {
		opts = append(opts, fmt.Sprintf("on_update: \"%s\"", strings.ToLower(op.OnUpdate)))
	}
	if len(opts) > 0 {
		sb.WriteString(fmt.Sprintf(", {%s}", strings.Join(opts, ", ")))
	}

	sb.WriteString(")\n")
}

// writeDropForeignKey writes a drop_foreign_key DSL call.
func (c *Client) writeDropForeignKey(sb *strings.Builder, op *ast.DropForeignKey) {
	ref := op.Table_
	if op.Namespace != "" {
		ref = op.Namespace + "." + op.Table_
	}
	sb.WriteString(fmt.Sprintf("    m.drop_foreign_key(\"%s\", \"%s\")\n", ref, op.Name))
}

// dryRunMigrations outputs the SQL that would be executed without running it.
func (c *Client) dryRunMigrations(ctx context.Context, plan *engine.Plan, output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	sqls, err := c.runner.RunDryRun(ctx, plan)
	if err != nil {
		return &MigrationError{
			Operation: "generate dry-run SQL",
			Cause:     err,
		}
	}

	direction := "UP"
	if plan.Direction == engine.Down {
		direction = "DOWN"
	}

	fmt.Fprintf(output, "-- Dry-run: %s\n", direction)
	fmt.Fprintf(output, "-- Migrations: %d\n\n", len(plan.Migrations))

	for i, m := range plan.Migrations {
		fmt.Fprintf(output, "-- [%d] %s: %s\n", i+1, m.Revision, m.Name)
	}
	fmt.Fprintln(output)

	for i, sql := range sqls {
		if i > 0 {
			fmt.Fprintln(output)
		}
		fmt.Fprintln(output, sql+";")
	}

	return nil
}
