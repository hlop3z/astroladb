package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// ---------------------------------------------------------------------------
// .lifecycle() — basic usage with column + transitions
// ---------------------------------------------------------------------------

func TestLifecycle_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  status: col.enum(["draft", "published", "archived"]),
}).lifecycle({
  column: "status",
  transitions: {
    publish:   { from: "draft",     to: "published" },
    archive:   { from: "published", to: "archived" },
    unpublish: { from: "published", to: "draft" },
  },
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Lifecycle == nil {
		t.Fatal("expected Lifecycle to be set, got nil")
	}
	if td.Lifecycle.Column != "status" {
		t.Errorf("Lifecycle.Column = %q, want %q", td.Lifecycle.Column, "status")
	}
	if len(td.Lifecycle.Transitions) != 3 {
		t.Fatalf("Lifecycle.Transitions len = %d, want 3", len(td.Lifecycle.Transitions))
	}

	// Check a specific transition
	pub := td.Lifecycle.Transitions["publish"]
	if pub == nil {
		t.Fatal("expected 'publish' transition")
	}
	if pub.From != "draft" || pub.To != "published" {
		t.Errorf("publish transition = {%q, %q}, want {draft, published}", pub.From, pub.To)
	}
}

// ---------------------------------------------------------------------------
// .lifecycle() — column only, no transitions
// ---------------------------------------------------------------------------

func TestLifecycle_ColumnOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  status: col.enum(["active", "inactive"]),
}).lifecycle({
  column: "status",
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Lifecycle == nil {
		t.Fatal("expected Lifecycle to be set")
	}
	if td.Lifecycle.Column != "status" {
		t.Errorf("Lifecycle.Column = %q, want %q", td.Lifecycle.Column, "status")
	}
	if len(td.Lifecycle.Transitions) != 0 {
		t.Errorf("Lifecycle.Transitions len = %d, want 0", len(td.Lifecycle.Transitions))
	}
}

// ---------------------------------------------------------------------------
// .lifecycle() — omitted when not set
// ---------------------------------------------------------------------------

func TestLifecycle_OmittedWhenNotSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Lifecycle != nil {
		t.Errorf("expected Lifecycle to be nil when not called, got %+v", td.Lifecycle)
	}
}

// ---------------------------------------------------------------------------
// .lifecycle() — chains with other methods
// ---------------------------------------------------------------------------

func TestLifecycle_ChainsWithOtherMethods(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
  status: col.enum(["draft", "published"]),
}).lifecycle({ column: "status" }).meta({ service: "billing" }).sort_by("name")`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Lifecycle == nil || td.Lifecycle.Column != "status" {
		t.Errorf("Lifecycle not preserved after chaining")
	}
	if td.Meta == nil || td.Meta["service"] != "billing" {
		t.Errorf("Meta not preserved after chaining")
	}
	if len(td.SortBy) != 1 || td.SortBy[0] != "name" {
		t.Errorf("SortBy = %v, want [name]", td.SortBy)
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: lifecycle() with missing column key
// ---------------------------------------------------------------------------

func TestCRITICAL_Lifecycle_MissingColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
  status: col.enum(["draft", "published"]),
}).lifecycle({})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	assertStructuredError(t, err, alerr.ErrLifecycleInvalid, "error[VAL-020]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: lifecycle() column is not an enum
// ---------------------------------------------------------------------------

func TestCRITICAL_Lifecycle_ColumnNotEnum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
}).lifecycle({ column: "name" })`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	assertStructuredError(t, err, alerr.ErrLifecycleColumn, "error[VAL-021]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: lifecycle() column does not exist
// ---------------------------------------------------------------------------

func TestCRITICAL_Lifecycle_ColumnNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
}).lifecycle({ column: "status" })`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	assertStructuredError(t, err, alerr.ErrLifecycleColumn, "error[VAL-021]", "-->", "help:")
}

// ---------------------------------------------------------------------------
// CRITICAL: lifecycle() transition references unknown state
// ---------------------------------------------------------------------------

func TestCRITICAL_Lifecycle_UnknownState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
  status: col.enum(["draft", "published", "archived"]),
}).lifecycle({
  column: "status",
  transitions: {
    publish: { from: "draft", to: "publshed" },
  },
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	if err == nil {
		t.Fatal("expected error for unknown state, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "VAL-021") {
		t.Errorf("expected VAL-021 error code, got: %s", errStr)
	}
	if !strings.Contains(errStr, "publshed") {
		t.Errorf("expected error to mention 'publshed', got: %s", errStr)
	}
}
