package runtime

import (
	"encoding/json"
	"testing"
)

func TestJunctionDebug(t *testing.T) {
	schema := `
export default table({
  post_id: col.belongs_to("blog.post"),
  tag_id: col.belongs_to("blog.tag"),
}).junction("blog.post", "blog.tag");
	`

	sb := NewSandbox(nil)
	tableDef, err := sb.EvalSchema(schema, "test", "junction_table")
	if err != nil {
		t.Fatalf("schema error: %v", err)
	}

	// Debug: print table definition
	t.Logf("Table: %s.%s", tableDef.Namespace, tableDef.Name)
	t.Logf("Columns: %d", len(tableDef.Columns))
	for i, col := range tableDef.Columns {
		t.Logf("  [%d] %s: %s (ref=%v)", i, col.Name, col.Type, col.Reference)
	}

	// Debug: print metadata
	meta := sb.Metadata()
	t.Logf("M2M relationships: %d", len(meta.ManyToMany))
	for i, m2m := range meta.ManyToMany {
		metaJSON, _ := json.MarshalIndent(m2m, "  ", "  ")
		t.Logf("  [%d] %s", i, string(metaJSON))
	}

	// Debug: check all tables
	allTables := sb.GetTables()
	t.Logf("All tables: %d", len(allTables))
}
