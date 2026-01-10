// Package engine provides migration planning and execution.
package engine

import (
	"fmt"

	"github.com/hlop3z/astroladb/internal/ast"
)

// Warning represents a safety warning for a migration operation.
type Warning struct {
	Severity string // "warning" or "error"
	Type     string // "drop_table", "drop_column", "not_null_no_default"
	Table    string
	Column   string
	Message  string
}

// LintOperations checks operations for destructive changes.
// Returns warnings that should be shown to the user.
func LintOperations(ops []ast.Operation) []Warning {
	var warnings []Warning

	for _, op := range ops {
		switch op.Type() {
		case ast.OpDropTable:
			warnings = append(warnings, Warning{
				Severity: "warning",
				Type:     "drop_table",
				Table:    op.Table(),
				Message:  fmt.Sprintf("Will DELETE ALL DATA in table '%s'", op.Table()),
			})
		case ast.OpDropColumn:
			dc := op.(*ast.DropColumn)
			warnings = append(warnings, Warning{
				Severity: "warning",
				Type:     "drop_column",
				Table:    op.Table(),
				Column:   dc.Name,
				Message:  fmt.Sprintf("Will DELETE DATA in column '%s.%s'", op.Table(), dc.Name),
			})
		}
	}

	return warnings
}

// FormatWarnings returns a human-readable string of warnings.
func FormatWarnings(warnings []Warning) string {
	if len(warnings) == 0 {
		return ""
	}

	var result string
	result = "\nWARNING: Destructive operations detected\n\n"

	for _, w := range warnings {
		result += fmt.Sprintf("  - %s\n", w.Message)
	}

	result += "\n  Use --force to proceed anyway.\n"
	return result
}
