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

	// Use a Runner with nil db — splitStatements doesn't use db
	runner := &Runner{}

	f.Fuzz(func(t *testing.T, sql string) {
		// Should never panic
		_ = runner.splitStatements(sql)
	})
}
