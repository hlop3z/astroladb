package astroladb

import (
	"time"

	"github.com/hlop3z/astroladb/internal/chain"
	"github.com/hlop3z/astroladb/internal/engine"
)

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
			Revision:     a.Revision,
			Name:         name,
			AppliedAt:    a.AppliedAt,
			Checksum:     a.Checksum,
			ExecTimeMs:   a.ExecTimeMs,
			Description:  a.Description,
			SQLChecksum:  a.SQLChecksum,
			AppliedOrder: a.AppliedOrder,
		}
	}

	return result, nil
}

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

// VerifySQLDeterminism re-parses each applied migration file, regenerates SQL,
// and compares the SQL checksum against the stored value. Returns mismatches.
func (c *Client) VerifySQLDeterminism() ([]SQLDeterminismResult, error) {
	ctx, cancel := c.context()
	defer cancel()

	// Get applied migrations with stored SQL checksums
	if err := c.runner.VersionManager().EnsureTable(ctx); err != nil {
		return nil, &MigrationError{Operation: "ensure migration table", Cause: err}
	}
	applied, err := c.runner.VersionManager().GetApplied(ctx)
	if err != nil {
		return nil, &MigrationError{Operation: "get applied migrations", Cause: err}
	}

	// Build map of applied sql checksums
	appliedMap := make(map[string]engine.AppliedMigration)
	for _, a := range applied {
		if a.SQLChecksum != "" {
			appliedMap[a.Revision] = a
		}
	}

	if len(appliedMap) == 0 {
		return nil, nil
	}

	// Load migration files
	migrations, err := c.loadMigrationFiles()
	if err != nil {
		return nil, err
	}

	// For each migration that has a stored SQL checksum, regenerate SQL and compare
	var results []SQLDeterminismResult
	for _, m := range migrations {
		stored, ok := appliedMap[m.Revision]
		if !ok {
			continue
		}

		// Create a plan with just this migration to generate SQL
		plan := &engine.Plan{
			Migrations: []engine.Migration{m},
			Direction:  engine.Up,
		}
		sqls, err := c.runner.RunDryRun(ctx, plan)
		if err != nil {
			continue
		}
		currentChecksum := engine.ComputeSQLChecksum(sqls)

		results = append(results, SQLDeterminismResult{
			Revision:        m.Revision,
			Name:            m.Name,
			StoredChecksum:  stored.SQLChecksum,
			CurrentChecksum: currentChecksum,
			Match:           stored.SQLChecksum == currentChecksum,
		})
	}

	return results, nil
}
