package runner

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/dialect"
)

// FuzzFormatBackfillValue fuzzes the string escaping path of formatBackfillValue.
// Verifies that no input can produce an unquoted or improperly escaped result.
func FuzzFormatBackfillValue(f *testing.F) {
	// Seed corpus with injection patterns
	f.Add("hello")
	f.Add("")
	f.Add("O'Reilly")
	f.Add("'; DROP TABLE users; --")
	f.Add("value' OR '1'='1")
	f.Add("' UNION SELECT * FROM secrets --")
	f.Add("'''")
	f.Add("''''")
	f.Add("\x00")
	f.Add("line1\nline2")
	f.Add("tab\there")
	f.Add("emoji 🌍")
	f.Add(strings.Repeat("'", 1000))

	r := &Runner{dialect: dialect.Postgres()}

	f.Fuzz(func(t *testing.T, input string) {
		got := r.formatBackfillValue(input)

		// Invariant 1: Result must start and end with single quotes
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Fatalf("result not properly quoted for input %q: %q", input, got)
		}

		// Invariant 2: Inner content must not contain unescaped single quotes.
		// After removing all '' (escaped quote pairs), no ' should remain.
		inner := got[1 : len(got)-1]
		unescaped := strings.ReplaceAll(inner, "''", "")
		if strings.Contains(unescaped, "'") {
			t.Fatalf("unescaped single quote for input %q: result=%q, inner=%q", input, got, inner)
		}
	})
}
