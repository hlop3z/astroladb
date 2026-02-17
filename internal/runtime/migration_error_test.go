package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/cli"
)

// ---------------------------------------------------------------------------
// Helper: write a migration JS file and run it through the sandbox
// ---------------------------------------------------------------------------

func runMigrationFile(t *testing.T, code string) error {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "migration.js")
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("write migration file: %v", err)
	}
	sb := NewSandbox(nil)
	_, err := sb.RunFile(path)
	return err
}

// assertStructuredError checks that the error is an *alerr.Error with the
// expected code and that FormatError output contains all expected substrings.
func assertStructuredError(t *testing.T, err error, wantCode alerr.Code, wantSubstrings ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	alErr, ok := err.(*alerr.Error)
	if !ok {
		t.Fatalf("expected *alerr.Error, got %T: %v", err, err)
	}

	if wantCode != "" && alErr.GetCode() != wantCode {
		t.Errorf("error code = %s, want %s", alErr.GetCode(), wantCode)
	}

	output := cli.FormatError(err)
	for _, want := range wantSubstrings {
		if !strings.Contains(output, want) {
			t.Errorf("FormatError output missing %q\ngot:\n%s", want, output)
		}
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: migration() with no arguments
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_NoArgs(t *testing.T) {
	err := runMigrationFile(t, `migration()`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: migration() with non-object argument
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_NotObject(t *testing.T) {
	err := runMigrationFile(t, `migration("string")`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: migration({}) missing up()
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_MissingUp(t *testing.T) {
	err := runMigrationFile(t, `migration({})`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: migration({ up: "not a function" })
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_UpNotFunction(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: "not fn" })`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: create_table() second arg is not a function
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_CreateTableBadFn(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) { m.create_table("ns.t", "not fn") } })`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: add_column() second arg is not a function
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_AddColumnBadFn(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) { m.add_column("ns.t", "not fn") } })`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: add_column() callback defines zero columns
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_AddColumnNoCol(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) { m.add_column("ns.t", function(col) { }) } })`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: add_column() callback defines multiple columns
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_AddColumnMultiple(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) { m.add_column("ns.t", function(col) { col.string("a", 100); col.string("b", 100); }) } })`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: alter_column() third arg is not a function
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_AlterColumnBadFn(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) { m.alter_column("ns.t", "col", "not fn") } })`)
	assertStructuredError(t, err, alerr.ErrMigrationFailed, "error[MIG-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: invalid table reference (bad identifier)
// ---------------------------------------------------------------------------

func TestCRITICAL_MigrationError_InvalidRef(t *testing.T) {
	err := runMigrationFile(t, `migration({ up: function(m) { m.create_table("123invalid", function(col) {}) } })`)
	assertStructuredError(t, err, alerr.ErrInvalidReference, "error[VAL-003]", "-->", "help:")
}
