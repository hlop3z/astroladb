package runner

import (
	"testing"
)

func FuzzSplitStatements(f *testing.F) {
	f.Add("SELECT 1; SELECT 2")
	f.Add("SELECT 'hello; world'")
	f.Add("SELECT $$body$$; SELECT 1")
	f.Add("SELECT $func$hello; world$func$; SELECT 2")
	f.Add("SELECT 'it''s'; SELECT 2")
	f.Add("")
	f.Add(";;;")
	// Additional edge cases
	f.Add("SELECT $tag$nested $$dollar$$ signs$tag$; SELECT 2")
	f.Add("SELECT ''; SELECT ''")
	f.Add("SELECT ''''; SELECT 1")
	f.Add("SELECT $$ $$ ; SELECT $$$$")
	f.Add("CREATE TABLE t (id INT); INSERT INTO t VALUES (1); DROP TABLE t")
	f.Add("SELECT 'line1\nline2'; SELECT 1")
	f.Add("SELECT $invalid tag$; SELECT 1")

	// Use a Runner with nil db — splitStatements doesn't use db
	runner := &Runner{}

	f.Fuzz(func(t *testing.T, sql string) {
		// Should never panic
		_ = runner.splitStatements(sql)
	})
}
