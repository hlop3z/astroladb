package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/cli"
)

// TestPreviewAllErrors is a visual test that prints every structured error.
// Run with: go test -run TestPreviewAllErrors ./internal/runtime/ -v -count=1
// Or with: task preview-errors
func TestPreviewAllErrors(t *testing.T) {
	sep := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

	// --- Migration errors ---
	migrationCases := []struct {
		name string
		code string
	}{
		{"migration() no args", `migration()`},
		{"migration() not object", `migration("string")`},
		{"migration() missing up", `migration({})`},
		{"migration() up not function", `migration({ up: "not fn" })`},
		{"create_table() bad fn", `migration({ up: function(m) { m.create_table("ns.t", "not fn") } })`},
		{"add_column() bad fn", `migration({ up: function(m) { m.add_column("ns.t", "not fn") } })`},
		{"add_column() no column", `migration({ up: function(m) { m.add_column("ns.t", function(col) { }) } })`},
		{"add_column() multiple columns", `migration({ up: function(m) { m.add_column("ns.t", function(col) { col.string("a", 100); col.string("b", 100); }) } })`},
		{"alter_column() bad fn", `migration({ up: function(m) { m.alter_column("ns.t", "col", "not fn") } })`},
		{"invalid reference", `migration({ up: function(m) { m.create_table("123invalid", function(col) {}) } })`},
	}

	fmt.Printf("\n%s\n  MIGRATION ERRORS (MIG-001 / VAL-003)\n%s\n\n", sep, sep)
	for _, tc := range migrationCases {
		err := runMigrationFile(t, tc.code)
		if err != nil {
			fmt.Printf("  [%s]\n\n%s\n\n", tc.name, cli.FormatError(err))
		}
	}

	// --- Schema errors ---
	schemaCases := []struct {
		name string
		code string
	}{
		{"table() no args", `export default table()`},
		{"table() not object", `export default table("string")`},
	}

	fmt.Printf("%s\n  SCHEMA ERRORS (SCH-001)\n%s\n\n", sep, sep)
	for _, tc := range schemaCases {
		dir := t.TempDir()
		path := filepath.Join(dir, "preview.js")
		_ = os.WriteFile(path, []byte(tc.code), 0644)
		sb := NewSandbox(nil)
		_, err := sb.EvalSchemaFile(path, "auth", "preview")
		if err != nil {
			fmt.Printf("  [%s]\n\n%s\n\n", tc.name, cli.FormatError(err))
		}
	}

	// --- Validation errors (string, decimal, belongs_to, many_to_many) ---
	validationCases := []struct {
		name string
		code string
	}{
		{"col.string() missing length", "export default table({\n  id: col.id(),\n  name: col.string(),\n})"},
		{"col.decimal() missing args", "export default table({\n  id: col.id(),\n  price: col.decimal(),\n})"},
		{"col.belongs_to() missing ref", "export default table({\n  id: col.id(),\n  owner: col.belongs_to(),\n})"},
		{"many_to_many() missing ref", "export default table({\n  id: col.id(),\n  name: col.string(100),\n}).many_to_many()"},
	}

	fmt.Printf("%s\n  VALIDATION ERRORS (VAL-007 / VAL-009)\n%s\n\n", sep, sep)
	for _, tc := range validationCases {
		dir := t.TempDir()
		path := filepath.Join(dir, "preview.js")
		_ = os.WriteFile(path, []byte(tc.code), 0644)
		sb := NewSandbox(nil)
		_, err := sb.EvalSchemaFile(path, "auth", "preview")
		if err != nil {
			fmt.Printf("  [%s]\n\n%s\n\n", tc.name, cli.FormatError(err))
		}
	}

	// --- Generator errors ---
	genCases := []struct {
		name string
		code string
	}{
		{"gen() no args", `gen()`},
		{"gen() not function", `gen("string")`},
		{"render() no args", `gen(function(s) { render() })`},
		{"render() not object", `gen(function(s) { render("string") })`},
		{"render() array", `gen(function(s) { render([1, 2]) })`},
		{"render() non-string values", `gen(function(s) { render({ "f.txt": 42 }) })`},
		{"json() no args", `gen(function(s) { json() })`},
	}

	fmt.Printf("%s\n  GENERATOR ERRORS (GEN-001)\n%s\n\n", sep, sep)
	for _, tc := range genCases {
		err := runGenerator(t, tc.code)
		if err != nil {
			fmt.Printf("  [%s]\n\n%s\n\n", tc.name, cli.FormatError(err))
		}
	}
}
