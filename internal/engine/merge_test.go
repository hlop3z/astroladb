package engine

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// NewSchema Tests
// -----------------------------------------------------------------------------

func TestNewSchema(t *testing.T) {
	s := NewSchema()

	if s == nil {
		t.Fatal("NewSchema() returned nil")
	}

	if s.Tables == nil {
		t.Error("NewSchema().Tables is nil")
	}

	if len(s.Tables) != 0 {
		t.Errorf("NewSchema().Tables has %d entries, want 0", len(s.Tables))
	}
}

// -----------------------------------------------------------------------------
// Merge Tests
// -----------------------------------------------------------------------------

func TestSchemaMerge(t *testing.T) {
	s := NewSchema()

	tables := []*ast.TableDef{
		{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		},
		{
			Namespace: "blog",
			Name:      "posts",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		},
	}

	err := s.Merge(tables)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if s.Count() != 2 {
		t.Errorf("Count() = %d, want 2", s.Count())
	}
}

func TestSchemaMergeDuplicateError(t *testing.T) {
	s := NewSchema()

	// First merge succeeds
	err := s.Merge([]*ast.TableDef{
		{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
	})
	if err != nil {
		t.Fatalf("First Merge() error = %v", err)
	}

	// Second merge with same table should fail
	err = s.Merge([]*ast.TableDef{
		{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
	})
	if err == nil {
		t.Error("Merge() should fail for duplicate table")
	}
}

func TestSchemaMergeNilTable(t *testing.T) {
	s := NewSchema()

	err := s.Merge([]*ast.TableDef{nil})
	if err == nil {
		t.Error("Merge() should fail for nil table")
	}
}

func TestSchemaMergeOverwrite(t *testing.T) {
	s := NewSchema()

	// Add initial table
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	// Overwrite with new definition
	newTables := []*ast.TableDef{
		{
			Namespace: "auth",
			Name:      "users",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id", PrimaryKey: true},
				{Name: "email", Type: "string"},
			},
		},
	}

	s.MergeOverwrite(newTables)

	table, _ := s.GetTable("auth.users")
	if len(table.Columns) != 2 {
		t.Errorf("MergeOverwrite() table has %d columns, want 2", len(table.Columns))
	}
}

// -----------------------------------------------------------------------------
// AddTable / GetTable Tests
// -----------------------------------------------------------------------------

func TestSchemaAddTable(t *testing.T) {
	s := NewSchema()

	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	err := s.AddTable(table)
	if err != nil {
		t.Fatalf("AddTable() error = %v", err)
	}

	if s.Count() != 1 {
		t.Errorf("Count() = %d, want 1", s.Count())
	}
}

func TestSchemaGetTable(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	tests := []struct {
		name  string
		query string
		found bool
	}{
		{"existing", "auth.users", true},
		{"non_existing", "auth.roles", false},
		{"wrong_namespace", "blog.users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, ok := s.GetTable(tt.query)
			if ok != tt.found {
				t.Errorf("GetTable(%q) found = %v, want %v", tt.query, ok, tt.found)
			}
			if tt.found && table == nil {
				t.Error("GetTable() returned nil for existing table")
			}
		})
	}
}

func TestSchemaGetTableByParts(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	table, ok := s.GetTableByParts("auth", "users")
	if !ok {
		t.Error("GetTableByParts() should find existing table")
	}
	if table.Name != "users" {
		t.Errorf("GetTableByParts() table.Name = %q, want %q", table.Name, "users")
	}

	_, ok = s.GetTableByParts("blog", "users")
	if ok {
		t.Error("GetTableByParts() should not find non-existing table")
	}
}

// -----------------------------------------------------------------------------
// Schema Helper Methods Tests
// -----------------------------------------------------------------------------

func TestSchemaTableNames(t *testing.T) {
	s := NewSchema()
	s.Tables["blog.posts"] = &ast.TableDef{Namespace: "blog", Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}
	s.Tables["auth.users"] = &ast.TableDef{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}

	names := s.TableNames()

	if len(names) != 2 {
		t.Fatalf("TableNames() = %d names, want 2", len(names))
	}

	// Should be sorted alphabetically
	if names[0] != "auth.users" {
		t.Errorf("TableNames()[0] = %q, want %q", names[0], "auth.users")
	}
	if names[1] != "blog.posts" {
		t.Errorf("TableNames()[1] = %q, want %q", names[1], "blog.posts")
	}
}

func TestSchemaNamespaces(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}
	s.Tables["auth.roles"] = &ast.TableDef{Namespace: "auth", Name: "roles", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}
	s.Tables["blog.posts"] = &ast.TableDef{Namespace: "blog", Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}

	namespaces := s.Namespaces()

	if len(namespaces) != 2 {
		t.Fatalf("Namespaces() = %d, want 2", len(namespaces))
	}

	// Should be sorted
	if namespaces[0] != "auth" || namespaces[1] != "blog" {
		t.Errorf("Namespaces() = %v, want [auth, blog]", namespaces)
	}
}

func TestSchemaTablesInNamespace(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}
	s.Tables["auth.roles"] = &ast.TableDef{Namespace: "auth", Name: "roles", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}
	s.Tables["blog.posts"] = &ast.TableDef{Namespace: "blog", Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}

	authTables := s.TablesInNamespace("auth")
	if len(authTables) != 2 {
		t.Errorf("TablesInNamespace(auth) = %d tables, want 2", len(authTables))
	}

	blogTables := s.TablesInNamespace("blog")
	if len(blogTables) != 1 {
		t.Errorf("TablesInNamespace(blog) = %d tables, want 1", len(blogTables))
	}

	emptyTables := s.TablesInNamespace("nonexistent")
	if len(emptyTables) != 0 {
		t.Errorf("TablesInNamespace(nonexistent) = %d tables, want 0", len(emptyTables))
	}
}

func TestSchemaIsEmpty(t *testing.T) {
	s := NewSchema()

	if !s.IsEmpty() {
		t.Error("IsEmpty() should be true for new schema")
	}

	s.Tables["auth.users"] = &ast.TableDef{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}

	if s.IsEmpty() {
		t.Error("IsEmpty() should be false after adding table")
	}
}

func TestSchemaClone(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	clone := s.Clone()

	if clone.Count() != s.Count() {
		t.Errorf("Clone() count = %d, want %d", clone.Count(), s.Count())
	}

	// Modifying clone should not affect original
	clone.Tables["blog.posts"] = &ast.TableDef{Namespace: "blog", Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}

	if s.Count() != 1 {
		t.Error("Clone() modification should not affect original")
	}
}

// -----------------------------------------------------------------------------
// Validate Tests
// -----------------------------------------------------------------------------

func TestSchemaValidate(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	err := s.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestSchemaValidateInvalidReference(t *testing.T) {
	s := NewSchema()

	// Table with FK to non-existent table
	s.Tables["blog.posts"] = &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "user_id", Type: "uuid", Reference: &ast.Reference{
				Table:  "auth.users", // This table doesn't exist
				Column: "id",
			}},
		},
	}

	err := s.Validate()
	if err == nil {
		t.Error("Validate() should fail for invalid reference")
	}

	if !strings.Contains(err.Error(), "references unknown table") {
		t.Errorf("Validate() error should mention invalid reference, got: %v", err)
	}
}

func TestSchemaValidateCircularDependency(t *testing.T) {
	s := NewSchema()

	// Create circular dependency: A -> B -> C -> A
	s.Tables["ns.a"] = &ast.TableDef{
		Namespace: "ns",
		Name:      "a",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "c_id", Type: "uuid", Reference: &ast.Reference{Table: "ns.c", Column: "id"}},
		},
	}
	s.Tables["ns.b"] = &ast.TableDef{
		Namespace: "ns",
		Name:      "b",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "a_id", Type: "uuid", Reference: &ast.Reference{Table: "ns.a", Column: "id"}},
		},
	}
	s.Tables["ns.c"] = &ast.TableDef{
		Namespace: "ns",
		Name:      "c",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "b_id", Type: "uuid", Reference: &ast.Reference{Table: "ns.b", Column: "id"}},
		},
	}

	err := s.Validate()
	if err == nil {
		t.Error("Validate() should detect circular dependency")
	}

	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Validate() error should mention circular dependency, got: %v", err)
	}
}

func TestSchemaValidateSelfReference(t *testing.T) {
	s := NewSchema()

	// Self-referencing table (e.g., parent-child hierarchy) is allowed
	s.Tables["org.employees"] = &ast.TableDef{
		Namespace: "org",
		Name:      "employees",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "manager_id", Type: "uuid", Nullable: true, Reference: &ast.Reference{
				Table:  "org.employees", // Self-reference
				Column: "id",
			}},
		},
	}

	err := s.Validate()
	if err != nil {
		t.Errorf("Validate() should allow self-reference, got error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// GetCreationOrder Tests
// -----------------------------------------------------------------------------

func TestSchemaGetCreationOrder(t *testing.T) {
	s := NewSchema()

	// posts depends on users
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}
	s.Tables["blog.posts"] = &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "user_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
		},
	}
	// comments depends on posts
	s.Tables["blog.comments"] = &ast.TableDef{
		Namespace: "blog",
		Name:      "comments",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "post_id", Type: "uuid", Reference: &ast.Reference{Table: "blog.posts", Column: "id"}},
		},
	}

	order, err := s.GetCreationOrder()
	if err != nil {
		t.Fatalf("GetCreationOrder() error = %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("GetCreationOrder() = %d tables, want 3", len(order))
	}

	// Find positions
	var usersPos, postsPos, commentsPos int
	for i, table := range order {
		switch table.QualifiedName() {
		case "auth.users":
			usersPos = i
		case "blog.posts":
			postsPos = i
		case "blog.comments":
			commentsPos = i
		}
	}

	// Verify order: users < posts < comments
	if usersPos >= postsPos {
		t.Errorf("GetCreationOrder(): users should come before posts")
	}
	if postsPos >= commentsPos {
		t.Errorf("GetCreationOrder(): posts should come before comments")
	}
}

func TestSchemaGetCreationOrderNoDependencies(t *testing.T) {
	s := NewSchema()

	// Tables with no dependencies
	s.Tables["a.one"] = &ast.TableDef{
		Namespace: "a",
		Name:      "one",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}
	s.Tables["b.two"] = &ast.TableDef{
		Namespace: "b",
		Name:      "two",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	order, err := s.GetCreationOrder()
	if err != nil {
		t.Fatalf("GetCreationOrder() error = %v", err)
	}

	if len(order) != 2 {
		t.Errorf("GetCreationOrder() = %d tables, want 2", len(order))
	}
}

func TestSchemaGetDropOrder(t *testing.T) {
	s := NewSchema()

	// posts depends on users
	s.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}
	s.Tables["blog.posts"] = &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "user_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
		},
	}

	order, err := s.GetDropOrder()
	if err != nil {
		t.Fatalf("GetDropOrder() error = %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("GetDropOrder() = %d tables, want 2", len(order))
	}

	// Drop order is reverse of creation: posts before users
	if order[0].QualifiedName() != "blog.posts" {
		t.Errorf("GetDropOrder()[0] = %q, want %q", order[0].QualifiedName(), "blog.posts")
	}
	if order[1].QualifiedName() != "auth.users" {
		t.Errorf("GetDropOrder()[1] = %q, want %q", order[1].QualifiedName(), "auth.users")
	}
}

// -----------------------------------------------------------------------------
// SchemaFromTables Tests
// -----------------------------------------------------------------------------

func TestSchemaFromTables(t *testing.T) {
	tables := []*ast.TableDef{
		{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
		{Namespace: "blog", Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
	}

	s, err := SchemaFromTables(tables)
	if err != nil {
		t.Fatalf("SchemaFromTables() error = %v", err)
	}

	if s.Count() != 2 {
		t.Errorf("SchemaFromTables() count = %d, want 2", s.Count())
	}
}

func TestSchemaFromTablesWithDuplicate(t *testing.T) {
	tables := []*ast.TableDef{
		{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
		{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}, // Duplicate
	}

	_, err := SchemaFromTables(tables)
	if err == nil {
		t.Error("SchemaFromTables() should fail for duplicate tables")
	}
}

// -----------------------------------------------------------------------------
// RemoveTable Tests
// -----------------------------------------------------------------------------

func TestSchemaRemoveTable(t *testing.T) {
	s := NewSchema()
	s.Tables["auth.users"] = &ast.TableDef{Namespace: "auth", Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}
	s.Tables["blog.posts"] = &ast.TableDef{Namespace: "blog", Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}

	s.RemoveTable("auth.users")

	if s.Count() != 1 {
		t.Errorf("Count() after RemoveTable = %d, want 1", s.Count())
	}

	if _, ok := s.GetTable("auth.users"); ok {
		t.Error("RemoveTable() should remove the table")
	}
}
