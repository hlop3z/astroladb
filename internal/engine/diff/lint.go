package diff

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
)

// reservedSQLWords contains common SQL reserved words that should not be used as identifiers.
var reservedSQLWords = map[string]bool{
	"abort": true, "action": true, "add": true, "after": true, "all": true,
	"alter": true, "analyze": true, "and": true, "as": true, "asc": true,
	"attach": true, "autoincrement": true, "before": true, "begin": true,
	"between": true, "by": true, "cascade": true, "case": true, "cast": true,
	"check": true, "collate": true, "column": true, "commit": true, "conflict": true,
	"constraint": true, "create": true, "cross": true, "current": true, "current_date": true,
	"current_time": true, "current_timestamp": true, "database": true, "default": true,
	"deferrable": true, "deferred": true, "delete": true, "desc": true, "detach": true,
	"distinct": true, "do": true, "drop": true, "each": true, "else": true,
	"end": true, "escape": true, "except": true, "exclude": true, "exclusive": true,
	"exists": true, "explain": true, "fail": true, "filter": true, "first": true,
	"following": true, "for": true, "foreign": true, "from": true, "full": true,
	"glob": true, "group": true, "groups": true, "having": true, "if": true,
	"ignore": true, "immediate": true, "in": true, "index": true, "indexed": true,
	"initially": true, "inner": true, "insert": true, "instead": true, "intersect": true,
	"into": true, "is": true, "isnull": true, "join": true, "key": true,
	"last": true, "left": true, "like": true, "limit": true, "match": true,
	"natural": true, "no": true, "not": true, "nothing": true, "notnull": true,
	"null": true, "nulls": true, "of": true, "offset": true, "on": true,
	"or": true, "order": true, "outer": true, "over": true, "partition": true,
	"plan": true, "pragma": true, "preceding": true, "primary": true, "query": true,
	"raise": true, "range": true, "recursive": true, "references": true, "regexp": true,
	"reindex": true, "release": true, "rename": true, "replace": true, "restrict": true,
	"returning": true, "right": true, "rollback": true, "row": true, "rows": true,
	"savepoint": true, "select": true, "set": true, "table": true, "temp": true,
	"temporary": true, "then": true, "ties": true, "to": true, "transaction": true,
	"trigger": true, "unbounded": true, "union": true, "unique": true, "update": true,
	"using": true, "vacuum": true, "values": true, "view": true, "virtual": true,
	"when": true, "where": true, "window": true, "with": true, "without": true,
	// PostgreSQL-specific
	"user": true, "role": true, "grant": true, "revoke": true, "session": true,
	"authorization": true, "privileges": true, "schema": true, "sequence": true,
	"serial": true, "type": true, "enum": true, "domain": true, "extension": true,
}

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
	if reservedSQLWords[strings.ToLower(name)] {
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

	var result string
	result = "\nWARNING: Destructive operations detected\n\n"

	for _, w := range warnings {
		result += fmt.Sprintf("  - %s\n", w.Message)
	}

	result += "\n  Use --force to proceed anyway.\n"
	return result
}
