package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// ---------------------------------------------------------------------------
// .events() — basic usage with payload and topic
// ---------------------------------------------------------------------------

func TestEvents_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  total: col.decimal(10, 2),
  status: col.enum(["draft", "published"]),
}).events({
  order_created: {
    payload: ["id", "total", "status"],
    topic:   "orders.created",
  },
  order_archived: {
    payload: ["id"],
    topic:   "orders.archived",
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

	if td.Events == nil {
		t.Fatal("expected Events to be set, got nil")
	}
	if len(td.Events) != 2 {
		t.Fatalf("Events len = %d, want 2", len(td.Events))
	}

	created := td.Events["order_created"]
	if created == nil {
		t.Fatal("expected 'order_created' event")
	}
	if len(created.Payload) != 3 {
		t.Errorf("order_created payload len = %d, want 3", len(created.Payload))
	}
	if created.Topic != "orders.created" {
		t.Errorf("order_created topic = %q, want %q", created.Topic, "orders.created")
	}

	archived := td.Events["order_archived"]
	if archived == nil {
		t.Fatal("expected 'order_archived' event")
	}
	if len(archived.Payload) != 1 || archived.Payload[0] != "id" {
		t.Errorf("order_archived payload = %v, want [id]", archived.Payload)
	}
}

// ---------------------------------------------------------------------------
// .events() — without topic (optional)
// ---------------------------------------------------------------------------

func TestEvents_NoTopic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  total: col.decimal(10, 2),
}).events({
  order_created: {
    payload: ["id", "total"],
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

	if td.Events == nil {
		t.Fatal("expected Events to be set")
	}
	created := td.Events["order_created"]
	if created == nil {
		t.Fatal("expected 'order_created' event")
	}
	if created.Topic != "" {
		t.Errorf("topic = %q, want empty string", created.Topic)
	}
}

// ---------------------------------------------------------------------------
// .events() — omitted when not set
// ---------------------------------------------------------------------------

func TestEvents_OmittedWhenNotSet(t *testing.T) {
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

	if td.Events != nil {
		t.Errorf("expected Events to be nil when not called, got %+v", td.Events)
	}
}

// ---------------------------------------------------------------------------
// .events() — chains with other methods
// ---------------------------------------------------------------------------

func TestEvents_ChainsWithOtherMethods(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  name: col.string(100),
}).events({
  item_created: { payload: ["id", "name"] },
}).meta({ team: "core" }).sort_by("name")`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Events == nil || len(td.Events) != 1 {
		t.Errorf("Events not preserved after chaining")
	}
	if td.Meta == nil || td.Meta["team"] != "core" {
		t.Errorf("Meta not preserved after chaining")
	}
	if len(td.SortBy) != 1 || td.SortBy[0] != "name" {
		t.Errorf("SortBy = %v, want [name]", td.SortBy)
	}
}

// ---------------------------------------------------------------------------
// .events() — combined with all four higher-order methods
// ---------------------------------------------------------------------------

func TestEvents_AllFourMethods(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "order.js")
	schema := `export default table({
  id: col.id(),
  total: col.decimal(10, 2),
  status: col.enum(["draft", "published", "archived"]),
})
  .meta({ service: "billing" })
  .lifecycle({ column: "status", transitions: { publish: { from: "draft", to: "published" } } })
  .policy({ create: ["admin"], read: ["admin", "owner"] })
  .events({ order_created: { payload: ["id", "total"], topic: "orders.created" } })`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	td, err := sb.EvalSchemaFile(path, "billing", "order")
	if err != nil {
		t.Fatalf("EvalSchemaFile: %v", err)
	}

	if td.Meta == nil {
		t.Error("Meta is nil")
	}
	if td.Lifecycle == nil {
		t.Error("Lifecycle is nil")
	}
	if td.Policy == nil {
		t.Error("Policy is nil")
	}
	if td.Events == nil {
		t.Error("Events is nil")
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: events() payload references unknown column
// ---------------------------------------------------------------------------

func TestCRITICAL_Events_UnknownColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
  total: col.decimal(10, 2),
}).events({
  order_created: {
    payload: ["id", "totals", "status"],
  },
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	if err == nil {
		t.Fatal("expected error for unknown column, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "VAL-023") {
		t.Errorf("expected VAL-023 error code, got: %s", errStr)
	}
	if !strings.Contains(errStr, "totals") {
		t.Errorf("expected error to mention 'totals', got: %s", errStr)
	}
}

// ---------------------------------------------------------------------------
// CRITICAL: events() with empty payload
// ---------------------------------------------------------------------------

func TestCRITICAL_Events_EmptyPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	schema := `export default table({
  id: col.id(),
}).events({
  order_created: {
    payload: [],
  },
})`
	if err := os.WriteFile(path, []byte(schema), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sb := NewSandbox(nil)
	_, err := sb.EvalSchemaFile(path, "billing", "order")
	assertStructuredError(t, err, alerr.ErrEventsInvalid, "error[VAL-023]", "-->", "help:")
}
