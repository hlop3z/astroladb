package astroladb

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hlop3z/astroladb/internal/chain"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/engine/runner"
)

// Migration execution and rollback operations.

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
	plan, err := runner.PlanMigrations(migrations, applied, cfg.Target, engine.Up)
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
	plan, err := runner.PlanMigrations(migrations, applied, cfg.Target, engine.Down)
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
				Checksum: link.Checksum,
			})
			continue
		}

		hooks := c.sandbox.GetHooks()
		meta := c.sandbox.GetMigrationMeta()

		m := engine.Migration{
			Revision:        link.Revision,
			Name:            link.Name,
			Path:            path,
			Checksum:        link.Checksum,
			Operations:      ops,
			BeforeHooks:     hooks.Before,
			AfterHooks:      hooks.After,
			DownBeforeHooks: hooks.DownBefore,
			DownAfterHooks:  hooks.DownAfter,
			Description:     meta.Description,
		}

		// Detect baseline migrations by name and parse squashed_through from file
		if link.Name == "baseline" {
			m.IsBaseline = true
			m.SquashedThrough = parseSquashedThrough(path)
		}

		migrations = append(migrations, m)
	}

	return migrations, nil
}

// parseSquashedThrough reads the first line of a baseline migration file
// and extracts the squashed_through revision from the header comment.
// Expected format: // Baseline: squashed from N migrations (through revision XXX)
func parseSquashedThrough(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	firstLine := strings.SplitN(string(data), "\n", 2)[0]
	// Look for "(through revision XXX)"
	const marker = "(through revision "
	idx := strings.Index(firstLine, marker)
	if idx < 0 {
		return ""
	}
	rest := firstLine[idx+len(marker):]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

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
