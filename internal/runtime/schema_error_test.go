package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// ---------------------------------------------------------------------------
// CRITICAL: table() with no arguments
// ---------------------------------------------------------------------------

func TestCRITICAL_SchemaError_TableNoArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	if err := os.WriteFile(path, []byte(`export default table()`), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "bad")
	assertStructuredError(t, err, alerr.ErrSchemaInvalid, "error[SCH-001]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: table() with non-object argument
// ---------------------------------------------------------------------------

func TestCRITICAL_SchemaError_TableNotObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	if err := os.WriteFile(path, []byte(`export default table("string")`), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "auth", "bad")
	assertStructuredError(t, err, alerr.ErrSchemaInvalid, "error[SCH-001]", "-->", "help:")
}
