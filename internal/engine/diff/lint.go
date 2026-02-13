package diff

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/validate"
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
		case ast.OpAddColumn:
			ac := op.(*ast.AddColumn)
			if ac.Column != nil && !ac.Column.Nullable && !ac.Column.DefaultSet && ac.Column.ServerDefault == "" && !ac.Column.BackfillSet && !ac.Column.PrimaryKey {
				warnings = append(warnings, Warning{
					Severity: "warning",
					Type:     "not_null_no_default",
					Table:    op.Table(),
					Column:   ac.Column.Name,
					Message:  fmt.Sprintf("NOT NULL column '%s.%s' has no default or backfill value", op.Table(), ac.Column.Name),
				})
			}
			// Check reserved words
			if ac.Column != nil {
				if w := checkReservedWord(op.Table(), ac.Column.Name); w != nil {
					warnings = append(warnings, *w)
				}
			}
		case ast.OpCreateTable:
			ct := op.(*ast.CreateTable)
			// Check full table name (namespace_tablename), not just Name
			if w := checkReservedWord("", op.Table()); w != nil {
				warnings = append(warnings, *w)
			}
			for _, col := range ct.Columns {
				if w := checkReservedWord(op.Table(), col.Name); w != nil {
					warnings = append(warnings, *w)
				}
			}
		}
	}

	return warnings
}

// checkReservedWord checks if an identifier is a SQL reserved word.
func checkReservedWord(table, name string) *Warning {
	if validate.IsReservedWord(name) {
		w := &Warning{
			Severity: "warning",
			Type:     "reserved_word",
			Table:    table,
			Column:   name,
		}
		if table != "" {
			w.Message = fmt.Sprintf("'%s.%s' is a SQL reserved word", table, name)
		} else {
			w.Message = fmt.Sprintf("'%s' is a SQL reserved word", name)
		}
		return w
	}
	return nil
}

// FormatWarnings returns a human-readable string of warnings.
func FormatWarnings(warnings []Warning) string {
	if len(warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nWARNING: Destructive operations detected\n\n")

	for _, w := range warnings {
		fmt.Fprintf(&sb, "  - %s\n", w.Message)
	}

	sb.WriteString("\n  Use --force to proceed anyway.\n")
	return sb.String()
}
