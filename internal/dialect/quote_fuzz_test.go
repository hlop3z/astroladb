package dialect

import (
	"strings"
	"testing"
)

// FuzzQuoteIdentPostgres tests that QuoteIdent on Postgres never produces
// an unescaped double-quote inside the identifier, and always wraps the
// result in double-quotes.
func FuzzQuoteIdentPostgres(f *testing.F) {
	seeds := []string{
		"users",
		"my_table",
		`col"name`,
		`"already_quoted"`,
		`""`,
		"",
		"SELECT 1; DROP TABLE users--",
		"tab\x00le",
		"hello\nworld",
		"café",
		"日本語",
		strings.Repeat("a", 100),
		`a""b""c`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	d := Postgres()

	f.Fuzz(func(t *testing.T, name string) {
		result := d.QuoteIdent(name)

		// Must start and end with double-quote
		if !strings.HasPrefix(result, `"`) || !strings.HasSuffix(result, `"`) {
			t.Errorf("QuoteIdent(%q) = %q — not wrapped in double-quotes", name, result)
		}

		// Strip outer quotes and check that all internal double-quotes are escaped (paired)
		inner := result[1 : len(result)-1]
		i := 0
		for i < len(inner) {
			if inner[i] == '"' {
				if i+1 >= len(inner) || inner[i+1] != '"' {
					t.Errorf("QuoteIdent(%q) = %q — unescaped double-quote at position %d", name, result, i+1)
					break
				}
				i += 2 // skip the escaped pair
			} else {
				i++
			}
		}
	})
}

// FuzzQuoteIdentSQLite tests that QuoteIdent on SQLite never produces
// an unescaped double-quote inside the identifier.
func FuzzQuoteIdentSQLite(f *testing.F) {
	seeds := []string{
		"users",
		"my_table",
		`col"name`,
		`"already_quoted"`,
		`""`,
		"",
		"SELECT 1; DROP TABLE users--",
		"tab\x00le",
		"hello\nworld",
		"café",
		"日本語",
		strings.Repeat("a", 100),
		`a""b""c`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	d := SQLite()

	f.Fuzz(func(t *testing.T, name string) {
		result := d.QuoteIdent(name)

		// Must start and end with double-quote
		if !strings.HasPrefix(result, `"`) || !strings.HasSuffix(result, `"`) {
			t.Errorf("QuoteIdent(%q) = %q — not wrapped in double-quotes", name, result)
		}

		// Strip outer quotes and check that all internal double-quotes are escaped (paired)
		inner := result[1 : len(result)-1]
		i := 0
		for i < len(inner) {
			if inner[i] == '"' {
				if i+1 >= len(inner) || inner[i+1] != '"' {
					t.Errorf("QuoteIdent(%q) = %q — unescaped double-quote at position %d", name, result, i+1)
					break
				}
				i += 2
			} else {
				i++
			}
		}
	})
}
