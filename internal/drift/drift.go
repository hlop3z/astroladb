package drift

import (
	"context"
	"database/sql"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/introspect"
)

// Detector performs schema drift detection by comparing expected schema
// (computed from migrations) against actual database schema.
type Detector struct {
	db      *sql.DB
	dialect dialect.Dialect
}

// NewDetector creates a new drift detector for the given database connection.
func NewDetector(db *sql.DB, d dialect.Dialect) *Detector {
	return &Detector{
		db:      db,
		dialect: d,
	}
}

// Result represents the complete drift detection result.
type Result struct {
	// HasDrift is true if any differences were found
	HasDrift bool

	// ExpectedHash is the merkle root of the expected schema
	ExpectedHash string

	// ActualHash is the merkle root of the actual database schema
	ActualHash string

	// Comparison contains detailed comparison results
	Comparison *HashComparison

	// ExpectedSchema is the schema computed from migrations
	ExpectedSchema *engine.Schema

	// ActualSchema is the schema introspected from the database
	ActualSchema *engine.Schema
}

// Detect compares expected schema against actual database schema.
// It returns a detailed result including merkle tree comparison for fast
// identification of differences.
func (d *Detector) Detect(ctx context.Context, expected *engine.Schema) (*Result, error) {
	// Introspect actual database schema
	inspector := introspect.New(d.db, d.dialect)
	if inspector == nil {
		return nil, alerr.New(alerr.EUnsupportedDialect, "dialect not supported for introspection").
			With("dialect", d.dialect.Name())
	}

	actual, err := inspector.IntrospectSchema(ctx)
	if err != nil {
		return nil, alerr.Wrap(alerr.EInternalError, err, "failed to introspect database schema")
	}

	// Compute merkle hashes for both schemas
	expectedHash, err := ComputeSchemaHash(expected)
	if err != nil {
		return nil, alerr.Wrap(alerr.EInternalError, err, "failed to compute expected schema hash")
	}

	actualHash, err := ComputeSchemaHash(actual)
	if err != nil {
		return nil, alerr.Wrap(alerr.EInternalError, err, "failed to compute actual schema hash")
	}

	// Compare hashes
	comparison := CompareHashes(expectedHash, actualHash)

	return &Result{
		HasDrift:       !comparison.Match,
		ExpectedHash:   expectedHash.Root,
		ActualHash:     actualHash.Root,
		Comparison:     comparison,
		ExpectedSchema: expected,
		ActualSchema:   actual,
	}, nil
}

// QuickCheck performs a fast drift check by comparing only root hashes.
// Use this when you just need to know if drift exists, not the details.
func (d *Detector) QuickCheck(ctx context.Context, expected *engine.Schema) (bool, error) {
	result, err := d.Detect(ctx, expected)
	if err != nil {
		return false, err
	}
	return !result.HasDrift, nil
}

// DriftSummary provides a human-readable summary of drift detection results.
type DriftSummary struct {
	// Tables is the total number of tables in expected schema
	Tables int

	// MissingTables is the count of tables missing from database
	MissingTables int

	// ExtraTables is the count of unexpected tables in database
	ExtraTables int

	// ModifiedTables is the count of tables with differences
	ModifiedTables int

	// Details contains per-table drift information
	Details []TableDriftSummary
}

// TableDriftSummary summarizes drift for a single table.
type TableDriftSummary struct {
	Name        string
	Status      string // "missing", "extra", "modified", "ok"
	Columns     DriftCounts
	Indexes     DriftCounts
	ForeignKeys DriftCounts
}

// DriftCounts tracks missing/extra/modified counts.
type DriftCounts struct {
	Missing  int
	Extra    int
	Modified int
}

// Summarize creates a human-readable summary from drift detection result.
func Summarize(result *Result) *DriftSummary {
	if result == nil || result.Comparison == nil {
		return &DriftSummary{}
	}

	summary := &DriftSummary{
		Tables:         len(result.ExpectedSchema.Tables),
		MissingTables:  len(result.Comparison.MissingTables),
		ExtraTables:    len(result.Comparison.ExtraTables),
		ModifiedTables: len(result.Comparison.TableDiffs),
		Details:        []TableDriftSummary{},
	}

	// Add missing tables
	for _, name := range result.Comparison.MissingTables {
		summary.Details = append(summary.Details, TableDriftSummary{
			Name:   name,
			Status: "missing",
		})
	}

	// Add extra tables
	for _, name := range result.Comparison.ExtraTables {
		summary.Details = append(summary.Details, TableDriftSummary{
			Name:   name,
			Status: "extra",
		})
	}

	// Add modified tables
	for name, diff := range result.Comparison.TableDiffs {
		summary.Details = append(summary.Details, TableDriftSummary{
			Name:   name,
			Status: "modified",
			Columns: DriftCounts{
				Missing:  len(diff.MissingColumns),
				Extra:    len(diff.ExtraColumns),
				Modified: len(diff.ModifiedColumns),
			},
			Indexes: DriftCounts{
				Missing:  len(diff.MissingIndexes),
				Extra:    len(diff.ExtraIndexes),
				Modified: len(diff.ModifiedIndexes),
			},
			ForeignKeys: DriftCounts{
				Missing:  len(diff.MissingFKs),
				Extra:    len(diff.ExtraFKs),
				Modified: len(diff.ModifiedFKs),
			},
		})
	}

	return summary
}
