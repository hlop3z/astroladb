package builder

import "fmt"

// BuilderError represents a structured error message with cause and help.
type BuilderError struct {
	Cause string
	Help  string
}

// String formats the error as "CAUSE|HELP" for easy parsing.
func (e BuilderError) String() string {
	return fmt.Sprintf("%s|%s", e.Cause, e.Help)
}

// Builder error messages - centralized for consistency and maintainability.
// These messages follow Rust-style error formatting with helpful examples.
var (
	// Column type errors
	ErrMsgStringRequiresLength = BuilderError{
		Cause: "string() requires a length argument",
		Help:  "try col.string(255) for VARCHAR(255)",
	}

	ErrMsgDecimalRequiresArgs = BuilderError{
		Cause: "decimal() requires precision and scale arguments",
		Help:  "try col.decimal(10, 2) for DECIMAL(10,2)",
	}

	// Table builder (migration) errors
	ErrMsgTableDecimalRequiresArgs = BuilderError{
		Cause: "decimal() requires precision and scale arguments",
		Help:  "try c.decimal(\"price\", 10, 2) for DECIMAL(10,2)",
	}

	// Enum errors
	ErrMsgEnumRequiresValues = BuilderError{
		Cause: "enum() requires at least one value",
		Help:  "try col.enum(['active', 'inactive']) or c.enum(\"status\", ['active', 'inactive'])",
	}

	// Relationship errors (column-level, schema API)
	ErrMsgBelongsToRequiresRef = BuilderError{
		Cause: "belongs_to() requires a table reference",
		Help:  "try col.belongs_to('namespace.table') or col.belongs_to('.table') for same namespace",
	}
)
