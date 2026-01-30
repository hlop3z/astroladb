package ast

import "testing"

func FuzzValidateSQLExpression(f *testing.F) {
	// Seed corpus
	f.Add("column > 0")
	f.Add("name IS NOT NULL")
	f.Add("age BETWEEN 0 AND 150")
	f.Add("status IN ('active', 'inactive')")
	f.Add("'; DROP TABLE users; --")
	f.Add("1; EXEC xp_cmdshell('whoami')")
	f.Add("UNION SELECT * FROM passwords")
	f.Add("pg_sleep(10)")
	f.Add("")

	f.Fuzz(func(t *testing.T, expr string) {
		// Should never panic
		_ = ValidateSQLExpression(expr)
	})
}

func FuzzValidateIdentifier(f *testing.F) {
	f.Add("users")
	f.Add("created_at")
	f.Add("a123")
	f.Add("")
	f.Add("Robert'; DROP TABLE students;--")
	f.Add("UNION")

	f.Fuzz(func(t *testing.T, name string) {
		_ = ValidateIdentifier(name)
	})
}
