package dialect_test

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/dialect"
)

// TestPostgresImplementsAllInterfaces verifies that the PostgreSQL dialect
// satisfies all sub-interfaces via Go's structural typing.
func TestPostgresImplementsAllInterfaces(t *testing.T) {
	var d dialect.Dialect = dialect.Postgres()

	var _ dialect.TypeMapper = d
	var _ dialect.DDLGenerator = d
	var _ dialect.SQLFormatter = d
	var _ dialect.FeatureDetector = d
}

// TestSQLiteImplementsAllInterfaces verifies that the SQLite dialect
// satisfies all sub-interfaces via Go's structural typing.
func TestSQLiteImplementsAllInterfaces(t *testing.T) {
	var d dialect.Dialect = dialect.SQLite()

	var _ dialect.TypeMapper = d
	var _ dialect.DDLGenerator = d
	var _ dialect.SQLFormatter = d
	var _ dialect.FeatureDetector = d
}

// TestNarrowInterfaceUsage verifies that a full Dialect can be passed
// where only a narrow sub-interface is required.
func TestNarrowInterfaceUsage(t *testing.T) {
	pg := dialect.Postgres()
	sq := dialect.SQLite()

	// Accept Dialect as TypeMapper
	requireStringType := func(m dialect.TypeMapper, expected string) {
		t.Helper()
		if got := m.StringType(100); got != expected {
			t.Errorf("StringType(100) = %q, want %q", got, expected)
		}
	}
	requireStringType(pg, "VARCHAR(100)")
	requireStringType(sq, "TEXT")

	// Accept Dialect as SQLFormatter
	requireQuote := func(f dialect.SQLFormatter, name, expected string) {
		t.Helper()
		if got := f.QuoteIdent(name); got != expected {
			t.Errorf("QuoteIdent(%q) = %q, want %q", name, got, expected)
		}
	}
	requireQuote(pg, "users", `"users"`)
	requireQuote(sq, "users", `"users"`)

	// Accept Dialect as FeatureDetector
	requireFeature := func(f dialect.FeatureDetector) {
		t.Helper()
		if !f.SupportsIfExists() {
			t.Error("expected SupportsIfExists() = true")
		}
	}
	requireFeature(pg)
	requireFeature(sq)
}
