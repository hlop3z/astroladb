package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// ---------------------------------------------------------------------------
// .policy() — basic usage with CRUD actions
// ---------------------------------------------------------------------------

func TestPolicy_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  total: col.decimal(10, 2),
}).policy({
  create: ["admin"],
  read:   ["admin", "owner", "member"],
  update: ["admin", "owner"],
  delete: ["admin"],
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Policy == nil {
		t.Fatal("expected Policy to be set, got nil")
	}
	if len(td.Policy.Rules) != 4 {
		t.Fatalf("Policy.Rules len = %d, want 4", len(td.Policy.Rules))
	}

	// Check specific rules
	readRoles := td.Policy.Rules["read"]
	if len(readRoles) != 3 {
		t.Fatalf("read roles len = %d, want 3", len(readRoles))
	}
	wantRead := map[string]bool{"admin": true, "owner": true, "member": true}
	for _, r := range readRoles {
		if !wantRead[r] {
			t.Errorf("unexpected read role: %q", r)
		}
	}

	deleteRoles := td.Policy.Rules["delete"]
	if len(deleteRoles) != 1 || deleteRoles[0] != "admin" {
		t.Errorf("delete roles = %v, want [admin]", deleteRoles)
	}
}

// ---------------------------------------------------------------------------
// .policy() — freeform actions (not just CRUD)
// ---------------------------------------------------------------------------

func TestPolicy_FreeformActions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
}).policy({
  approve: ["admin", "manager"],
  refund:  ["admin"],
  export:  ["admin", "analyst"],
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Policy == nil {
		t.Fatal("expected Policy to be set")
	}
	if len(td.Policy.Rules) != 3 {
		t.Fatalf("Policy.Rules len = %d, want 3", len(td.Policy.Rules))
	}
	if roles := td.Policy.Rules["approve"]; len(roles) != 2 {
		t.Errorf("approve roles = %v, want 2 roles", roles)
	}
}

// ---------------------------------------------------------------------------
// .policy() — omitted when not set
// ---------------------------------------------------------------------------

func TestPolicy_OmittedWhenNotSet(t *testing.T) {
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

	if td.Policy != nil {
		t.Errorf("expected Policy to be nil when not called, got %+v", td.Policy)
	}
}

// ---------------------------------------------------------------------------
// .policy() — chains with other methods
// ---------------------------------------------------------------------------

func TestPolicy_ChainsWithOtherMethods(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
}).policy({ read: ["admin"] }).meta({ team: "core" }).sort_by("name")`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Policy == nil || len(td.Policy.Rules["read"]) != 1 {
		t.Errorf("Policy not preserved after chaining")
	}
	if td.Meta == nil || td.Meta["team"] != "core" {
		t.Errorf("Meta not preserved after chaining")
	}
	if len(td.SortBy) != 1 || td.SortBy[0] != "name" {
		t.Errorf("SortBy = %v, want [name]", td.SortBy)
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: policy() with empty role array
// ---------------------------------------------------------------------------

func TestCRITICAL_Policy_EmptyRoles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
}).policy({ read: [] })`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	assertStructuredError(t, err, alerr.ErrPolicyInvalid, "error[VAL-022]", "-->", "help:")
}
