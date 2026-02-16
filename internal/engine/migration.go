package engine

import "github.com/hlop3z/astroladb/internal/ast"

// Direction indicates whether migrations run up (apply) or down (rollback).
type Direction int

const (
	// Up applies migrations (creates/modifies schema).
	Up Direction = iota
	// Down rolls back migrations (reverts schema changes).
	Down
)

// String returns the string representation of the direction.
func (d Direction) String() string {
	if d == Up {
		return "up"
	}
	return "down"
}

// Migration represents a single migration with its operations.
type Migration struct {
	// Revision is the unique identifier (e.g., "001", "20240101120000").
	Revision string

	// Name is the human-readable name (e.g., "add_users_table").
	Name string

	// Path is the file path to the migration script.
	Path string

	// Checksum is the SHA256 hash of the migration content for integrity checking.
	Checksum string

	// Operations are the forward (up) operations to apply.
	Operations []ast.Operation

	// DownOps are the reverse operations (auto-generated or explicit).
	DownOps []ast.Operation

	// Irreversible marks migrations that cannot be rolled back.
	Irreversible bool

	// Dependencies are revisions that must be applied before this one.
	Dependencies []string

	// Description is a human-readable summary of what this migration does.
	Description string

	// IsBaseline marks this as a squashed baseline migration.
	IsBaseline bool

	// SquashedThrough is the last revision that was squashed into this baseline.
	SquashedThrough string
}

// Plan represents a set of migrations to execute in a specific direction.
type Plan struct {
	// Migrations to execute, in order.
	Migrations []Migration

	// Direction of execution (Up or Down).
	Direction Direction
}

// IsEmpty returns true if the plan has no migrations to execute.
func (p *Plan) IsEmpty() bool {
	return len(p.Migrations) == 0
}

// PlanStatus represents the status of a migration.
type PlanStatus int

const (
	// StatusPending means the migration has not been applied.
	StatusPending PlanStatus = iota
	// StatusApplied means the migration has been applied.
	StatusApplied
	// StatusMissing means the migration file is missing but was previously applied.
	StatusMissing
	// StatusModified means the migration checksum doesn't match.
	StatusModified
)

// String returns the string representation of the status.
func (s PlanStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusApplied:
		return "applied"
	case StatusMissing:
		return "missing"
	case StatusModified:
		return "modified"
	default:
		return "unknown"
	}
}

// MigrationStatus provides status information about a migration.
type MigrationStatus struct {
	Revision  string
	Name      string
	Status    PlanStatus
	AppliedAt *string // nil if not applied
	Checksum  string
}
