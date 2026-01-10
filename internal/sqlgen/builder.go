// Package sqlgen provides dialect-agnostic SQL building helpers to reduce string concatenation.
package sqlgen

import (
	"strconv"
	"strings"
)

// Dialect represents a supported SQL database dialect.
type Dialect int

const (
	// Postgres represents PostgreSQL dialect.
	Postgres Dialect = iota
	// SQLite represents SQLite dialect.
	SQLite
)

// String returns the string representation of the dialect.
func (d Dialect) String() string {
	switch d {
	case Postgres:
		return "postgres"
	case SQLite:
		return "sqlite"
	default:
		return "unknown"
	}
}

// Builder provides fluent SQL construction with dialect awareness.
type Builder struct {
	dialect Dialect
	buf     strings.Builder
}

// New creates a new Builder for the specified dialect.
func New(dialect Dialect) *Builder {
	return &Builder{
		dialect: dialect,
	}
}

// Dialect returns the dialect of this builder.
func (b *Builder) Dialect() Dialect {
	return b.dialect
}

// ----------------------------------------------------------------------------
// DDL Helpers
// ----------------------------------------------------------------------------

// CreateTable appends "CREATE TABLE <name>" to the buffer.
func (b *Builder) CreateTable(name string) *Builder {
	b.buf.WriteString("CREATE TABLE ")
	b.buf.WriteString(QuoteIdent(b.dialect, name))
	return b
}

// DropTable appends "DROP TABLE <name>" to the buffer.
func (b *Builder) DropTable(name string) *Builder {
	b.buf.WriteString("DROP TABLE ")
	b.buf.WriteString(QuoteIdent(b.dialect, name))
	return b
}

// AlterTable appends "ALTER TABLE <name>" to the buffer.
func (b *Builder) AlterTable(name string) *Builder {
	b.buf.WriteString("ALTER TABLE ")
	b.buf.WriteString(QuoteIdent(b.dialect, name))
	return b
}

// AddColumn appends "<name> <typ>" to the buffer (for use inside CREATE TABLE or ALTER TABLE ADD COLUMN).
func (b *Builder) AddColumn(name, typ string) *Builder {
	b.buf.WriteString(QuoteIdent(b.dialect, name))
	b.buf.WriteString(" ")
	b.buf.WriteString(typ)
	return b
}

// DropColumn appends "DROP COLUMN <name>" to the buffer.
func (b *Builder) DropColumn(name string) *Builder {
	b.buf.WriteString("DROP COLUMN ")
	b.buf.WriteString(QuoteIdent(b.dialect, name))
	return b
}

// RenameColumn appends dialect-specific column rename syntax.
// PostgreSQL: RENAME COLUMN <old> TO <new>
// SQLite: RENAME COLUMN <old> TO <new>
func (b *Builder) RenameColumn(old, new string) *Builder {
	b.buf.WriteString("RENAME COLUMN ")
	b.buf.WriteString(QuoteIdent(b.dialect, old))
	b.buf.WriteString(" TO ")
	b.buf.WriteString(QuoteIdent(b.dialect, new))
	return b
}

// ----------------------------------------------------------------------------
// Column Modifiers
// ----------------------------------------------------------------------------

// NotNull appends "NOT NULL" to the buffer.
func (b *Builder) NotNull() *Builder {
	b.buf.WriteString(" NOT NULL")
	return b
}

// Null appends "NULL" to the buffer.
func (b *Builder) Null() *Builder {
	b.buf.WriteString(" NULL")
	return b
}

// Default appends "DEFAULT <expr>" to the buffer.
// The expression is written as-is (not quoted).
func (b *Builder) Default(expr string) *Builder {
	b.buf.WriteString(" DEFAULT ")
	b.buf.WriteString(expr)
	return b
}

// PrimaryKey appends "PRIMARY KEY" to the buffer.
func (b *Builder) PrimaryKey() *Builder {
	b.buf.WriteString(" PRIMARY KEY")
	return b
}

// Unique appends "UNIQUE" to the buffer.
func (b *Builder) Unique() *Builder {
	b.buf.WriteString(" UNIQUE")
	return b
}

// References appends "REFERENCES <table>(<column>)" to the buffer.
func (b *Builder) References(table, column string) *Builder {
	b.buf.WriteString(" REFERENCES ")
	b.buf.WriteString(QuoteIdent(b.dialect, table))
	b.buf.WriteString("(")
	b.buf.WriteString(QuoteIdent(b.dialect, column))
	b.buf.WriteString(")")
	return b
}

// OnDelete appends "ON DELETE <action>" to the buffer.
// Common actions: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION.
func (b *Builder) OnDelete(action string) *Builder {
	b.buf.WriteString(" ON DELETE ")
	b.buf.WriteString(action)
	return b
}

// OnUpdate appends "ON UPDATE <action>" to the buffer.
// Common actions: CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION.
func (b *Builder) OnUpdate(action string) *Builder {
	b.buf.WriteString(" ON UPDATE ")
	b.buf.WriteString(action)
	return b
}

// ----------------------------------------------------------------------------
// Constraints
// ----------------------------------------------------------------------------

// Constraint appends "CONSTRAINT <name>" to the buffer.
func (b *Builder) Constraint(name string) *Builder {
	b.buf.WriteString("CONSTRAINT ")
	b.buf.WriteString(QuoteIdent(b.dialect, name))
	return b
}

// ForeignKey appends "FOREIGN KEY (<cols>)" to the buffer.
func (b *Builder) ForeignKey(cols ...string) *Builder {
	b.buf.WriteString(" FOREIGN KEY (")
	for i, col := range cols {
		if i > 0 {
			b.buf.WriteString(", ")
		}
		b.buf.WriteString(QuoteIdent(b.dialect, col))
	}
	b.buf.WriteString(")")
	return b
}

// Check appends "CHECK (<expr>)" to the buffer.
func (b *Builder) Check(expr string) *Builder {
	b.buf.WriteString(" CHECK (")
	b.buf.WriteString(expr)
	b.buf.WriteString(")")
	return b
}

// ----------------------------------------------------------------------------
// Utilities
// ----------------------------------------------------------------------------

// Raw appends raw SQL to the buffer without any modification.
func (b *Builder) Raw(sql string) *Builder {
	b.buf.WriteString(sql)
	return b
}

// Comma appends "," to the buffer.
func (b *Builder) Comma() *Builder {
	b.buf.WriteString(",")
	return b
}

// OpenParen appends " (" to the buffer.
func (b *Builder) OpenParen() *Builder {
	b.buf.WriteString(" (")
	return b
}

// CloseParen appends ")" to the buffer.
func (b *Builder) CloseParen() *Builder {
	b.buf.WriteString(")")
	return b
}

// Newline appends a newline character to the buffer.
func (b *Builder) Newline() *Builder {
	b.buf.WriteString("\n")
	return b
}

// Space appends a space character to the buffer.
func (b *Builder) Space() *Builder {
	b.buf.WriteString(" ")
	return b
}

// String returns the accumulated SQL string.
func (b *Builder) String() string {
	return b.buf.String()
}

// Reset clears the buffer so the builder can be reused.
func (b *Builder) Reset() *Builder {
	b.buf.Reset()
	return b
}

// ----------------------------------------------------------------------------
// Standalone Helpers
// ----------------------------------------------------------------------------

// QuoteIdent returns the identifier quoted according to the dialect.
// PostgreSQL and SQLite use double quotes: "name"
func QuoteIdent(dialect Dialect, s string) string {
	// PostgreSQL and SQLite use double quotes. Escape by doubling them.
	escaped := strings.ReplaceAll(s, `"`, `""`)
	return `"` + escaped + `"`
}

// Columns returns a comma-separated list of quoted column names using double quotes.
// Example: Columns("a", "b", "c") -> `"a", "b", "c"`
// For dialect-specific quoting, use QuoteIdent directly or Builder methods.
func Columns(cols ...string) string {
	if len(cols) == 0 {
		return ""
	}
	parts := make([]string, len(cols))
	for i, col := range cols {
		// Use double quotes (PostgreSQL/SQLite style) as the standard.
		escaped := strings.ReplaceAll(col, `"`, `""`)
		parts[i] = `"` + escaped + `"`
	}
	return strings.Join(parts, ", ")
}

// Placeholders returns a comma-separated list of placeholders for the given count.
// PostgreSQL uses numbered placeholders: $1, $2, $3
// SQLite uses question marks: ?, ?, ?
func Placeholders(dialect Dialect, n int) string {
	if n <= 0 {
		return ""
	}
	switch dialect {
	case Postgres:
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			parts[i] = "$" + strconv.Itoa(i+1)
		}
		return strings.Join(parts, ", ")
	case SQLite:
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			parts[i] = "?"
		}
		return strings.Join(parts, ", ")
	default:
		// Default to positional placeholders.
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			parts[i] = "$" + strconv.Itoa(i+1)
		}
		return strings.Join(parts, ", ")
	}
}

// List returns a comma-separated list of items without quoting.
// Example: List("a", "b", "c") -> "a, b, c"
func List(items ...string) string {
	return strings.Join(items, ", ")
}
