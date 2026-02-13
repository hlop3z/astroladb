package astroladb

import (
	"context"
	"fmt"
	"time"

	"github.com/hlop3z/astroladb/internal/chain"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/engine/diff"
	"github.com/hlop3z/astroladb/internal/engine/runner"
)

// Note: Migration implementations split across multiple files:
// - migration_run.go: Execution (MigrationRun, MigrationRollback)
// - migration_status.go: Status, history, verification queries
// - migration_generate.go: Migration generation orchestration
// - migration_dsl_content.go: Content generation logic
// - migration_dsl_writers.go: Pure DSL string writers

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

	// Description is a human-readable summary of the migration.
	Description string

	// SQLChecksum is the SHA256 hash of the generated SQL statements.
	SQLChecksum string

	// AppliedOrder is the sequential execution order (1-based).
	AppliedOrder int
}

// SQLDeterminismResult holds the result of a SQL determinism check for one migration.
type SQLDeterminismResult struct {
	Revision        string
	Name            string
	StoredChecksum  string
	CurrentChecksum string
	Match           bool
}

// MigrationLockInfo contains information about the current migration lock state.
type MigrationLockInfo struct {
	// Locked indicates whether the lock is currently held.
	Locked bool

	// LockedAt is when the lock was acquired (nil if not locked).
	LockedAt *time.Time

	// LockedBy identifies which process holds the lock (hostname:pid).
	LockedBy string
}

func (c *Client) ClearMigrationHistory(ctx context.Context) (int, error) {
	if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
		return 0, &MigrationError{
			Operation: "ensure migration table",
			Cause:     err,
		}
	}

	deleted, err := c.runner.VersionManager().DeleteAll(ctx)
	if err != nil {
		return 0, &MigrationError{
			Operation: "clear migration history",
			Cause:     err,
		}
	}

	return deleted, nil
}

func (c *Client) RecordSquashedBaseline(ctx context.Context, revision, checksum, squashedThrough string, squashedCount int) error {
	if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
		return &MigrationError{
			Operation: "ensure migration table",
			Cause:     err,
		}
	}

	description := fmt.Sprintf("Baseline: squashed from %d migrations", squashedCount)

	rec := runner.ApplyRecord{
		Revision:        revision,
		Checksum:        checksum,
		ExecTime:        0, // No actual execution time for squash
		Description:     description,
		SquashedThrough: squashedThrough,
	}

	if err := c.runner.VersionManager().RecordApplied(ctx, rec); err != nil {
		return &MigrationError{
			Operation: "record baseline migration",
			Cause:     err,
		}
	}

	return nil
}

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
	return runner.PlanMigrations(migrations, applied, "", engine.Up)
}

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

func (c *Client) LintPendingMigrations() ([]string, error) {
	// Get the diff of what would be applied
	ops, err := c.SchemaDiff()
	if err != nil {
		// If we can't get the diff, just return no warnings
		// (the migration will fail on its own if there's an issue)
		return nil, nil
	}

	warnings := diff.LintOperations(ops)
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
