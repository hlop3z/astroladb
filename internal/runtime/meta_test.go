package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// .meta() — stores freeform metadata on the table
// ---------------------------------------------------------------------------

func TestMeta_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  total: col.decimal(10, 2),
}).meta({
  service: "billing",
  team: "payments",
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Meta == nil {
		t.Fatal("expected Meta to be set, got nil")
	}
	if td.Meta["service"] != "billing" {
		t.Errorf("Meta[service] = %v, want %q", td.Meta["service"], "billing")
	}
	if td.Meta["team"] != "payments" {
		t.Errorf("Meta[team] = %v, want %q", td.Meta["team"], "payments")
	}
}

func TestMeta_NestedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
}).meta({
  tags: ["important", "billing"],
  config: { nested: true, level: 2 },
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Meta == nil {
		t.Fatal("expected Meta to be set, got nil")
	}

	// Check array value
	tags, ok := td.Meta["tags"].([]any)
	if !ok {
		t.Fatalf("Meta[tags] type = %T, want []any", td.Meta["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("Meta[tags] len = %d, want 2", len(tags))
	}

	// Check nested object
	config, ok := td.Meta["config"].(map[string]any)
	if !ok {
		t.Fatalf("Meta[config] type = %T, want map[string]any", td.Meta["config"])
	}
	if config["nested"] != true {
		t.Errorf("Meta[config][nested] = %v, want true", config["nested"])
	}
}

func TestMeta_OmittedWhenNotSet(t *testing.T) {
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

	if td.Meta != nil {
		t.Errorf("expected Meta to be nil when .meta() not called, got %v", td.Meta)
	}
}

func TestMeta_ChainsWithOtherMethods(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
}).meta({ service: "billing" }).sort_by("name").searchable("name")`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	// Meta
	if td.Meta == nil || td.Meta["service"] != "billing" {
		t.Errorf("Meta not preserved after chaining, got %v", td.Meta)
	}
	// SortBy
	if len(td.SortBy) != 1 || td.SortBy[0] != "name" {
		t.Errorf("SortBy = %v, want [name]", td.SortBy)
	}
	// Searchable
	if len(td.Searchable) != 1 || td.Searchable[0] != "name" {
		t.Errorf("Searchable = %v, want [name]", td.Searchable)
	}
}
