package runtime

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestDirectPath_BasicSchema verifies that the direct TableBuilder path
// produces the same result as the legacy map-based path.
func TestDirectPath_BasicSchema(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		email: col.string(255),
		name: col.string(100),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "users")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}
	if tableDef == nil {
		t.Fatal("EvalSchema() returned nil table")
	}

	if tableDef.Namespace != "auth" {
		t.Errorf("Namespace = %q, want %q", tableDef.Namespace, "auth")
	}
	if tableDef.Name != "users" {
		t.Errorf("Name = %q, want %q", tableDef.Name, "users")
	}
	if len(tableDef.Columns) != 3 {
		t.Fatalf("Expected 3 columns, got %d", len(tableDef.Columns))
	}

	// Verify column names (sorted alphabetically by key)
	expectedNames := map[string]bool{"id": true, "email": true, "name": true}
	for _, col := range tableDef.Columns {
		if !expectedNames[col.Name] {
			t.Errorf("Unexpected column name: %q", col.Name)
		}
	}
}

// TestDirectPath_WithTimestamps verifies timestamps chain method works.
func TestDirectPath_WithTimestamps(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		title: col.string(200),
	}).timestamps()`

	tableDef, err := sb.EvalSchema(code, "blog", "post")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	// Should have 4 columns: id, title, created_at, updated_at
	if len(tableDef.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(tableDef.Columns))
	}

	hasCreatedAt := false
	hasUpdatedAt := false
	for _, col := range tableDef.Columns {
		if col.Name == "created_at" {
			hasCreatedAt = true
			if col.Type != "datetime" {
				t.Errorf("created_at type = %q, want datetime", col.Type)
			}
		}
		if col.Name == "updated_at" {
			hasUpdatedAt = true
		}
	}
	if !hasCreatedAt {
		t.Error("Missing created_at column")
	}
	if !hasUpdatedAt {
		t.Error("Missing updated_at column")
	}
}

// TestDirectPath_WithRelationship verifies belongs_to works.
func TestDirectPath_WithRelationship(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		author: col.belongs_to("auth.user"),
	})`

	tableDef, err := sb.EvalSchema(code, "blog", "post")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	// Find the author_id column (belongs_to auto-appends _id)
	var authorCol = findColumn(tableDef.Columns, "author_id")
	if authorCol == nil {
		t.Fatal("Missing author_id column from belongs_to")
	}
	if authorCol.Type != "uuid" {
		t.Errorf("author_id type = %q, want uuid", authorCol.Type)
	}
	if authorCol.Reference == nil {
		t.Fatal("author_id should have a Reference")
	}
	if authorCol.Reference.Table != "auth.user" {
		t.Errorf("Reference.Table = %q, want %q", authorCol.Reference.Table, "auth.user")
	}
}

// TestDirectPath_NullableSet verifies NullableSet is set correctly.
func TestDirectPath_NullableSet(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		bio: col.text().optional(),
		name: col.string(100),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	bioCol := findColumn(tableDef.Columns, "bio")
	if bioCol == nil {
		t.Fatal("Missing bio column")
	}
	if !bioCol.Nullable {
		t.Error("bio should be nullable (.optional())")
	}
	if !bioCol.NullableSet {
		t.Error("bio NullableSet should be true")
	}

	nameCol := findColumn(tableDef.Columns, "name")
	if nameCol == nil {
		t.Fatal("Missing name column")
	}
	if nameCol.Nullable {
		t.Error("name should not be nullable")
	}
	if !nameCol.NullableSet {
		t.Error("name NullableSet should be true (builder always sets it)")
	}
}

// TestDirectPath_WithDefault verifies default values work.
func TestDirectPath_WithDefault(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		status: col.string(50).default("active"),
		is_admin: col.flag(false),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	statusCol := findColumn(tableDef.Columns, "status")
	if statusCol == nil {
		t.Fatal("Missing status column")
	}
	if !statusCol.DefaultSet {
		t.Error("status DefaultSet should be true")
	}
	if statusCol.Default != "active" {
		t.Errorf("status Default = %v, want %q", statusCol.Default, "active")
	}
}

// TestDirectPath_ManyToMany verifies many_to_many metadata is registered.
func TestDirectPath_ManyToMany(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		title: col.string(200),
	}).many_to_many("blog.tag")`

	tableDef, err := sb.EvalSchema(code, "blog", "post")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	// The many_to_many is metadata, not a column
	if len(tableDef.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(tableDef.Columns))
	}

	// Check metadata was registered
	meta := sb.Metadata()
	if meta == nil {
		t.Fatal("Metadata should not be nil")
	}
	// GetJoinTables should have created join table entries
	joinTables := meta.GetJoinTables()
	if len(joinTables) == 0 {
		t.Error("Expected join table to be created from many_to_many")
	}
}

// TestDirectPath_Docs verifies docs/deprecated chain methods work.
func TestDirectPath_Docs(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
	}).docs("User authentication table").deprecated("Use v2 API")`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	if tableDef.Docs != "User authentication table" {
		t.Errorf("Docs = %q, want %q", tableDef.Docs, "User authentication table")
	}
	if tableDef.Deprecated != "Use v2 API" {
		t.Errorf("Deprecated = %q, want %q", tableDef.Deprecated, "Use v2 API")
	}
}

// TestDirectPath_Auditable verifies auditable chain method works.
func TestDirectPath_Auditable(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
	}).auditable()`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	if !tableDef.Auditable {
		t.Error("Expected Auditable to be true")
	}

	// Should have created_by and updated_by columns
	createdBy := findColumn(tableDef.Columns, "created_by")
	updatedBy := findColumn(tableDef.Columns, "updated_by")
	if createdBy == nil {
		t.Error("Missing created_by column from .auditable()")
	}
	if updatedBy == nil {
		t.Error("Missing updated_by column from .auditable()")
	}
}

// TestDirectPath_SortBySearchableFilterable verifies x-db metadata.
func TestDirectPath_SortBySearchableFilterable(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		title: col.string(200),
		status: col.string(50),
	}).sort_by("-created_at", "title").searchable("title").filterable("status")`

	tableDef, err := sb.EvalSchema(code, "blog", "post")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	if len(tableDef.SortBy) != 2 {
		t.Errorf("SortBy length = %d, want 2", len(tableDef.SortBy))
	}
	if len(tableDef.Searchable) != 1 || tableDef.Searchable[0] != "title" {
		t.Errorf("Searchable = %v, want [title]", tableDef.Searchable)
	}
	if len(tableDef.Filterable) != 1 || tableDef.Filterable[0] != "status" {
		t.Errorf("Filterable = %v, want [status]", tableDef.Filterable)
	}
}

// TestDirectPath_ColumnModifiers verifies column modifier methods work.
func TestDirectPath_ColumnModifiers(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		email: col.string(255).unique().docs("User email"),
		bio: col.text().optional().deprecated("Use profile table"),
		age: col.integer().min(0).max(150),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	emailCol := findColumn(tableDef.Columns, "email")
	if emailCol == nil {
		t.Fatal("Missing email column")
	}
	if !emailCol.Unique {
		t.Error("email should be unique")
	}
	if emailCol.Docs != "User email" {
		t.Errorf("email.Docs = %q, want %q", emailCol.Docs, "User email")
	}

	bioCol := findColumn(tableDef.Columns, "bio")
	if bioCol == nil {
		t.Fatal("Missing bio column")
	}
	if !bioCol.Nullable {
		t.Error("bio should be nullable")
	}
	if bioCol.Deprecated != "Use profile table" {
		t.Errorf("bio.Deprecated = %q, want %q", bioCol.Deprecated, "Use profile table")
	}

	ageCol := findColumn(tableDef.Columns, "age")
	if ageCol == nil {
		t.Fatal("Missing age column")
	}
	if ageCol.Min == nil || *ageCol.Min != 0 {
		t.Error("age.Min should be 0")
	}
	if ageCol.Max == nil || *ageCol.Max != 150 {
		t.Error("age.Max should be 150")
	}
}

// TestDirectPath_EnumValidation verifies that col.enum() requires values.
func TestDirectPath_EnumValidation(t *testing.T) {
	sb := NewSandbox(nil)

	// Valid enum
	code := `export default table({
		id: col.id(),
		status: col.enum(["active", "inactive"]),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("Valid enum should not error: %v", err)
	}

	statusCol := findColumn(tableDef.Columns, "status")
	if statusCol == nil {
		t.Fatal("Missing status column")
	}
	if statusCol.Type != "enum" {
		t.Errorf("status.Type = %q, want enum", statusCol.Type)
	}
}

// TestDirectPath_EnumEmpty verifies that empty enum panics.
func TestDirectPath_EnumEmpty(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		status: col.enum([]),
	})`

	_, err := sb.EvalSchema(code, "auth", "user")
	if err == nil {
		t.Error("Empty enum should error")
	}
}

// TestDirectPath_SourceFile verifies source file is set.
func TestDirectPath_SourceFile(t *testing.T) {
	sb := NewSandbox(nil)
	sb.SetCurrentFile("/tmp/test_schema.js")

	code := `export default table({
		id: col.id(),
	})`

	tableDef, err := sb.EvalSchema(code, "test", "table")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	if tableDef.SourceFile != "/tmp/test_schema.js" {
		t.Errorf("SourceFile = %q, want %q", tableDef.SourceFile, "/tmp/test_schema.js")
	}
}

// TestDirectPath_Virtual verifies virtual column flag is preserved.
func TestDirectPath_Virtual(t *testing.T) {
	sb := NewSandbox(nil)

	code := `export default table({
		id: col.id(),
		full_name: col.string(500).virtual(),
	})`

	tableDef, err := sb.EvalSchema(code, "auth", "user")
	if err != nil {
		t.Fatalf("EvalSchema() error = %v", err)
	}

	fullNameCol := findColumn(tableDef.Columns, "full_name")
	if fullNameCol == nil {
		t.Fatal("Missing full_name column")
	}
	if !fullNameCol.Virtual {
		t.Error("full_name should have Virtual=true")
	}
}

// findColumn is a test helper that finds a column by name.
func findColumn(cols []*ast.ColumnDef, name string) *ast.ColumnDef {
	for _, col := range cols {
		if col.Name == name {
			return col
		}
	}
	return nil
}
