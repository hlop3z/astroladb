package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// New Tests
// -----------------------------------------------------------------------------

func TestNew(t *testing.T) {
	m := New()

	if m == nil {
		t.Fatal("New() returned nil")
	}

	if m.Version != "1.0" {
		t.Errorf("New().Version = %q, want %q", m.Version, "1.0")
	}

	if m.Tables == nil {
		t.Error("New().Tables is nil")
	}

	if m.ManyToMany == nil {
		t.Error("New().ManyToMany is nil")
	}

	if m.Polymorphic == nil {
		t.Error("New().Polymorphic is nil")
	}

	if m.JoinTables == nil {
		t.Error("New().JoinTables is nil")
	}

	if m.GeneratedAt.IsZero() {
		t.Error("New().GeneratedAt is zero")
	}
}

// -----------------------------------------------------------------------------
// AddTable Tests
// -----------------------------------------------------------------------------

func TestAddTable(t *testing.T) {
	t.Run("basic_table", func(t *testing.T) {
		m := New()

		table := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string"},
				{Name: "name", Type: "string"},
			},
		}

		m.AddTable(table)

		meta, ok := m.Tables["auth.users"]
		if !ok {
			t.Fatal("AddTable() did not add table to map")
		}

		if meta.Namespace != "auth" {
			t.Errorf("AddTable().Namespace = %q, want %q", meta.Namespace, "auth")
		}

		if meta.Name != "users" {
			t.Errorf("AddTable().Name = %q, want %q", meta.Name, "users")
		}

		if meta.SQLName != "auth_users" {
			t.Errorf("AddTable().SQLName = %q, want %q", meta.SQLName, "auth_users")
		}

		if len(meta.Columns) != 3 {
			t.Errorf("AddTable().Columns = %d, want 3", len(meta.Columns))
		}
	})

	t.Run("table_with_foreign_keys", func(t *testing.T) {
		m := New()

		table := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "title", Type: "string"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
			},
		}

		m.AddTable(table)

		meta := m.Tables["blog.posts"]
		if len(meta.ForeignKeys) != 1 {
			t.Errorf("AddTable().ForeignKeys = %d, want 1", len(meta.ForeignKeys))
		}

		if meta.ForeignKeys[0] != "author_id" {
			t.Errorf("AddTable().ForeignKeys[0] = %q, want %q", meta.ForeignKeys[0], "author_id")
		}
	})

	t.Run("table_with_indexes", func(t *testing.T) {
		m := New()

		table := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string"},
			},
			Indexes: []*ast.IndexDef{
				{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
			},
		}

		m.AddTable(table)

		meta := m.Tables["auth.users"]
		if len(meta.Indexes) != 1 {
			t.Errorf("AddTable().Indexes = %d, want 1", len(meta.Indexes))
		}
	})
}

// -----------------------------------------------------------------------------
// AddManyToMany Tests
// -----------------------------------------------------------------------------

func TestAddManyToMany(t *testing.T) {
	t.Run("creates_join_table", func(t *testing.T) {
		m := New()

		jt := m.AddManyToMany("auth", "users", "auth.roles", "")

		if jt == nil {
			t.Fatal("AddManyToMany() returned nil")
		}

		// Check join table name is alphabetically sorted
		if jt.Name != "auth_roles_auth_users" {
			t.Errorf("AddManyToMany().Name = %q, want %q", jt.Name, "auth_roles_auth_users")
		}
	})

	t.Run("records_relationship", func(t *testing.T) {
		m := New()

		m.AddManyToMany("auth", "users", "auth.roles", "")

		if len(m.ManyToMany) != 1 {
			t.Fatalf("AddManyToMany() m.ManyToMany = %d, want 1", len(m.ManyToMany))
		}

		rel := m.ManyToMany[0]
		if rel.Source != "auth.users" {
			t.Errorf("AddManyToMany().Source = %q, want %q", rel.Source, "auth.users")
		}

		if rel.Target != "auth.roles" {
			t.Errorf("AddManyToMany().Target = %q, want %q", rel.Target, "auth.roles")
		}
	})

	t.Run("generates_fk_columns", func(t *testing.T) {
		m := New()

		jt := m.AddManyToMany("auth", "users", "auth.roles", "")

		rel := m.ManyToMany[0]
		if rel.SourceFK != "users_id" {
			t.Errorf("AddManyToMany().SourceFK = %q, want %q", rel.SourceFK, "users_id")
		}

		if rel.TargetFK != "roles_id" {
			t.Errorf("AddManyToMany().TargetFK = %q, want %q", rel.TargetFK, "roles_id")
		}

		// Check definition has correct columns
		if jt.Definition == nil {
			t.Fatal("AddManyToMany().Definition is nil")
		}

		if len(jt.Definition.Columns) != 2 {
			t.Errorf("AddManyToMany().Definition.Columns = %d, want 2", len(jt.Definition.Columns))
		}
	})

	t.Run("generates_indexes", func(t *testing.T) {
		m := New()

		jt := m.AddManyToMany("auth", "users", "auth.roles", "")

		if jt.Definition == nil {
			t.Fatal("AddManyToMany().Definition is nil")
		}

		// Should have 3 indexes: source FK, target FK, and composite unique
		if len(jt.Definition.Indexes) != 3 {
			t.Errorf("AddManyToMany().Definition.Indexes = %d, want 3", len(jt.Definition.Indexes))
		}

		// Check for unique composite index
		hasUnique := false
		for _, idx := range jt.Definition.Indexes {
			if idx.Unique && len(idx.Columns) == 2 {
				hasUnique = true
				break
			}
		}

		if !hasUnique {
			t.Error("AddManyToMany() missing unique composite index")
		}
	})

	t.Run("stores_in_join_tables_map", func(t *testing.T) {
		m := New()

		jt := m.AddManyToMany("auth", "users", "auth.roles", "")

		stored, ok := m.JoinTables[jt.Name]
		if !ok {
			t.Fatal("AddManyToMany() did not store in JoinTables map")
		}

		if stored != jt {
			t.Error("AddManyToMany() stored different pointer")
		}
	})
}

// -----------------------------------------------------------------------------
// AddPolymorphic Tests
// -----------------------------------------------------------------------------

func TestAddPolymorphic(t *testing.T) {
	m := New()

	m.AddPolymorphic("blog", "comments", "commentable", []string{"blog.posts", "blog.pages"})

	if len(m.Polymorphic) != 1 {
		t.Fatalf("AddPolymorphic() m.Polymorphic = %d, want 1", len(m.Polymorphic))
	}

	poly := m.Polymorphic[0]

	if poly.Table != "blog.comments" {
		t.Errorf("AddPolymorphic().Table = %q, want %q", poly.Table, "blog.comments")
	}

	if poly.Alias != "commentable" {
		t.Errorf("AddPolymorphic().Alias = %q, want %q", poly.Alias, "commentable")
	}

	if poly.TypeColumn != "commentable_type" {
		t.Errorf("AddPolymorphic().TypeColumn = %q, want %q", poly.TypeColumn, "commentable_type")
	}

	if poly.IDColumn != "commentable_id" {
		t.Errorf("AddPolymorphic().IDColumn = %q, want %q", poly.IDColumn, "commentable_id")
	}

	if len(poly.Targets) != 2 {
		t.Errorf("AddPolymorphic().Targets = %d, want 2", len(poly.Targets))
	}
}

// -----------------------------------------------------------------------------
// GetJoinTables Tests
// -----------------------------------------------------------------------------

func TestGetJoinTables(t *testing.T) {
	t.Run("returns_all_definitions", func(t *testing.T) {
		m := New()

		m.AddManyToMany("auth", "users", "auth.roles", "")
		m.AddManyToMany("blog", "posts", "blog.tags", "")

		tables := m.GetJoinTables()

		if len(tables) != 2 {
			t.Errorf("GetJoinTables() = %d, want 2", len(tables))
		}
	})

	t.Run("empty_when_no_relationships", func(t *testing.T) {
		m := New()

		tables := m.GetJoinTables()

		if len(tables) != 0 {
			t.Errorf("GetJoinTables() = %d, want 0", len(tables))
		}
	})
}

// -----------------------------------------------------------------------------
// Save and Load Tests
// -----------------------------------------------------------------------------

func TestSaveAndLoad(t *testing.T) {
	t.Run("save_creates_directory", func(t *testing.T) {
		dir := t.TempDir()

		m := New()
		m.AddTable(&ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
		})

		err := m.Save(dir)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Check .alab directory was created
		alabDir := filepath.Join(dir, ".alab")
		if _, err := os.Stat(alabDir); os.IsNotExist(err) {
			t.Error("Save() did not create .alab directory")
		}

		// Check metadata.json was created
		metaFile := filepath.Join(alabDir, "metadata.json")
		if _, err := os.Stat(metaFile); os.IsNotExist(err) {
			t.Error("Save() did not create metadata.json")
		}
	})

	t.Run("load_returns_saved_data", func(t *testing.T) {
		dir := t.TempDir()

		// Create and save metadata
		m := New()
		m.AddTable(&ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
		})
		m.AddManyToMany("auth", "users", "auth.roles", "")
		m.AddPolymorphic("blog", "comments", "commentable", []string{"blog.posts"})

		if err := m.Save(dir); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Load and verify
		loaded, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if len(loaded.Tables) != 1 {
			t.Errorf("Load().Tables = %d, want 1", len(loaded.Tables))
		}

		if len(loaded.ManyToMany) != 1 {
			t.Errorf("Load().ManyToMany = %d, want 1", len(loaded.ManyToMany))
		}

		if len(loaded.Polymorphic) != 1 {
			t.Errorf("Load().Polymorphic = %d, want 1", len(loaded.Polymorphic))
		}
	})

	t.Run("load_nonexistent_returns_empty", func(t *testing.T) {
		dir := t.TempDir()

		loaded, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded == nil {
			t.Fatal("Load() returned nil for nonexistent file")
		}

		if loaded.Tables == nil {
			t.Error("Load().Tables is nil")
		}
	})

	t.Run("load_invalid_json_returns_error", func(t *testing.T) {
		dir := t.TempDir()

		// Create invalid JSON file
		alabDir := filepath.Join(dir, ".alab")
		if err := os.MkdirAll(alabDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		metaFile := filepath.Join(alabDir, "metadata.json")
		if err := os.WriteFile(metaFile, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		_, err := Load(dir)
		if err == nil {
			t.Error("Load() expected error for invalid JSON")
		}
	})
}

// -----------------------------------------------------------------------------
// SaveToFile Tests
// -----------------------------------------------------------------------------

func TestSaveToFile(t *testing.T) {
	t.Run("creates_parent_directories", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "nested", "dir", "metadata.json")

		m := New()
		err := m.SaveToFile(filePath)
		if err != nil {
			t.Fatalf("SaveToFile() error = %v", err)
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("SaveToFile() did not create file")
		}
	})

	t.Run("writes_valid_json", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.json")

		m := New()
		m.AddTable(&ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
		})

		if err := m.SaveToFile(filePath); err != nil {
			t.Fatalf("SaveToFile() error = %v", err)
		}

		// Read and parse JSON
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("SaveToFile() wrote invalid JSON: %v", err)
		}
	})
}

// -----------------------------------------------------------------------------
// parseRef Tests
// -----------------------------------------------------------------------------

func TestParseRef(t *testing.T) {
	tests := []struct {
		ref       string
		wantNS    string
		wantTable string
	}{
		{"auth.users", "auth", "users"},
		{"blog.posts", "blog", "posts"},
		{"users", "", "users"},
		{"a.b.c", "a", "b.c"}, // Only first dot is split
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			ns, table := parseRef(tt.ref)
			if ns != tt.wantNS {
				t.Errorf("parseRef(%q) namespace = %q, want %q", tt.ref, ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("parseRef(%q) table = %q, want %q", tt.ref, table, tt.wantTable)
			}
		})
	}
}
