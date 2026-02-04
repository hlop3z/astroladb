package introspect

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// -----------------------------------------------------------------------------
// parseTableName Tests
// -----------------------------------------------------------------------------

func TestParseTableName(t *testing.T) {
	tests := []struct {
		name      string
		sqlName   string
		mapping   TableNameMapping
		wantNS    string
		wantTable string
	}{
		{
			name:      "simple_namespace_table",
			sqlName:   "auth_users",
			mapping:   nil,
			wantNS:    "auth",
			wantTable: "users",
		},
		{
			name:      "no_underscore",
			sqlName:   "users",
			mapping:   nil,
			wantNS:    "",
			wantTable: "users",
		},
		{
			name:      "multiple_underscores_fallback",
			sqlName:   "my_app_users",
			mapping:   nil,
			wantNS:    "my", // Naive parsing splits at first underscore
			wantTable: "app_users",
		},
		{
			name:    "with_mapping_resolves_correctly",
			sqlName: "my_app_users",
			mapping: TableNameMapping{
				"my_app_users": {Namespace: "my_app", Name: "users"},
			},
			wantNS:    "my_app",
			wantTable: "users",
		},
		{
			name:    "mapping_not_found_falls_back",
			sqlName: "other_table",
			mapping: TableNameMapping{
				"my_app_users": {Namespace: "my_app", Name: "users"},
			},
			wantNS:    "other",
			wantTable: "table",
		},
		{
			name:      "underscore_in_table_name",
			sqlName:   "blog_user_posts",
			mapping:   nil,
			wantNS:    "blog",
			wantTable: "user_posts",
		},
		{
			name:    "mapping_with_underscore_in_table",
			sqlName: "blog_user_posts",
			mapping: TableNameMapping{
				"blog_user_posts": {Namespace: "blog", Name: "user_posts"},
			},
			wantNS:    "blog",
			wantTable: "user_posts",
		},
		{
			name:      "empty_string",
			sqlName:   "",
			mapping:   nil,
			wantNS:    "",
			wantTable: "",
		},
		{
			name:      "single_underscore",
			sqlName:   "_",
			mapping:   nil,
			wantNS:    "",
			wantTable: "",
		},
		{
			name:      "leading_underscore",
			sqlName:   "_users",
			mapping:   nil,
			wantNS:    "",
			wantTable: "users",
		},
		{
			name:      "trailing_underscore",
			sqlName:   "auth_",
			mapping:   nil,
			wantNS:    "auth",
			wantTable: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, table := parseTableName(tt.sqlName, tt.mapping)
			if ns != tt.wantNS {
				t.Errorf("parseTableName(%q) namespace = %q, want %q", tt.sqlName, ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("parseTableName(%q) table = %q, want %q", tt.sqlName, table, tt.wantTable)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// normalizeAction Tests
// -----------------------------------------------------------------------------

func TestNormalizeAction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CASCADE", "CASCADE"},
		{"cascade", "CASCADE"},
		{"Cascade", "CASCADE"},
		{"SET NULL", "SET NULL"},
		{"set null", "SET NULL"},
		{"RESTRICT", "RESTRICT"},
		{"restrict", "RESTRICT"},
		{"NO ACTION", ""},
		{"no action", ""},
		{"", ""},
		{"INVALID", ""},
		{"SET DEFAULT", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeAction(tt.input)
			if got != tt.want {
				t.Errorf("normalizeAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// isInternalTable Tests
// -----------------------------------------------------------------------------

func TestIsInternalTable(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"alab_migrations", true},
		{"users", false},
		{"auth_users", false},
		{"migrations", false},
		{"alab_migrations_lock", true}, // Also internal
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInternalTable(tt.name)
			if got != tt.want {
				t.Errorf("isInternalTable(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// BuildTableNameMapping Tests
// -----------------------------------------------------------------------------

func TestBuildTableNameMapping(t *testing.T) {
	t.Run("nil_schema", func(t *testing.T) {
		mapping := BuildTableNameMapping(nil)
		if mapping != nil {
			t.Errorf("BuildTableNameMapping(nil) = %v, want nil", mapping)
		}
	})

	t.Run("empty_schema", func(t *testing.T) {
		schema := engine.NewSchema()
		mapping := BuildTableNameMapping(schema)
		if mapping == nil {
			t.Fatal("BuildTableNameMapping() returned nil")
		}
		if len(mapping) != 0 {
			t.Errorf("BuildTableNameMapping() = %d entries, want 0", len(mapping))
		}
	})

	t.Run("schema_with_tables", func(t *testing.T) {
		schema := engine.NewSchema()
		schema.Tables["auth.users"] = &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
		}
		schema.Tables["blog.posts"] = &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
		}

		mapping := BuildTableNameMapping(schema)
		if mapping == nil {
			t.Fatal("BuildTableNameMapping() returned nil")
		}
		if len(mapping) != 2 {
			t.Errorf("BuildTableNameMapping() = %d entries, want 2", len(mapping))
		}

		// Check auth_users mapping
		if entry, ok := mapping["auth_users"]; ok {
			if entry.Namespace != "auth" {
				t.Errorf("mapping['auth_users'].Namespace = %q, want %q", entry.Namespace, "auth")
			}
			if entry.Name != "users" {
				t.Errorf("mapping['auth_users'].Name = %q, want %q", entry.Name, "users")
			}
		} else {
			t.Error("mapping missing 'auth_users' entry")
		}

		// Check blog_posts mapping
		if entry, ok := mapping["blog_posts"]; ok {
			if entry.Namespace != "blog" {
				t.Errorf("mapping['blog_posts'].Namespace = %q, want %q", entry.Namespace, "blog")
			}
			if entry.Name != "posts" {
				t.Errorf("mapping['blog_posts'].Name = %q, want %q", entry.Name, "posts")
			}
		} else {
			t.Error("mapping missing 'blog_posts' entry")
		}
	})

	t.Run("namespace_with_underscore", func(t *testing.T) {
		schema := engine.NewSchema()
		schema.Tables["my_app.users"] = &ast.TableDef{
			Namespace: "my_app",
			Name:      "users",
		}

		mapping := BuildTableNameMapping(schema)

		// The SQL name would be "my_app_users"
		if entry, ok := mapping["my_app_users"]; ok {
			if entry.Namespace != "my_app" {
				t.Errorf("mapping['my_app_users'].Namespace = %q, want %q", entry.Namespace, "my_app")
			}
			if entry.Name != "users" {
				t.Errorf("mapping['my_app_users'].Name = %q, want %q", entry.Name, "users")
			}
		} else {
			t.Error("mapping missing 'my_app_users' entry")
		}
	})
}

// -----------------------------------------------------------------------------
// Round-trip Tests
// -----------------------------------------------------------------------------

func TestParseTableNameRoundTrip(t *testing.T) {
	// Test that a schema can be converted to mapping and back
	schema := engine.NewSchema()
	schema.Tables["auth.users"] = &ast.TableDef{Namespace: "auth", Name: "users"}
	schema.Tables["blog.posts"] = &ast.TableDef{Namespace: "blog", Name: "posts"}
	schema.Tables["my_app.user_settings"] = &ast.TableDef{Namespace: "my_app", Name: "user_settings"}

	mapping := BuildTableNameMapping(schema)

	testCases := []struct {
		sqlName   string
		wantNS    string
		wantTable string
	}{
		{"auth_users", "auth", "users"},
		{"blog_posts", "blog", "posts"},
		{"my_app_user_settings", "my_app", "user_settings"},
	}

	for _, tc := range testCases {
		t.Run(tc.sqlName, func(t *testing.T) {
			ns, table := parseTableName(tc.sqlName, mapping)
			if ns != tc.wantNS {
				t.Errorf("namespace = %q, want %q", ns, tc.wantNS)
			}
			if table != tc.wantTable {
				t.Errorf("table = %q, want %q", table, tc.wantTable)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// RawColumn Tests (struct functionality)
// -----------------------------------------------------------------------------

func TestRawColumn(t *testing.T) {
	col := RawColumn{
		Name:         "email",
		DataType:     "VARCHAR",
		IsNullable:   false,
		IsPrimaryKey: false,
		IsUnique:     true,
	}

	if col.Name != "email" {
		t.Errorf("RawColumn.Name = %q, want %q", col.Name, "email")
	}
	if col.DataType != "VARCHAR" {
		t.Errorf("RawColumn.DataType = %q, want %q", col.DataType, "VARCHAR")
	}
	if col.IsNullable {
		t.Error("RawColumn.IsNullable should be false")
	}
	if col.IsPrimaryKey {
		t.Error("RawColumn.IsPrimaryKey should be false")
	}
	if !col.IsUnique {
		t.Error("RawColumn.IsUnique should be true")
	}
}

// -----------------------------------------------------------------------------
// RawIndex Tests (struct functionality)
// -----------------------------------------------------------------------------

func TestRawIndex(t *testing.T) {
	idx := RawIndex{
		Name:    "idx_users_email",
		Columns: []string{"email"},
		Unique:  true,
	}

	if idx.Name != "idx_users_email" {
		t.Errorf("RawIndex.Name = %q, want %q", idx.Name, "idx_users_email")
	}
	if len(idx.Columns) != 1 {
		t.Errorf("RawIndex.Columns = %d, want 1", len(idx.Columns))
	}
	if idx.Columns[0] != "email" {
		t.Errorf("RawIndex.Columns[0] = %q, want %q", idx.Columns[0], "email")
	}
	if !idx.Unique {
		t.Error("RawIndex.Unique should be true")
	}
}

// -----------------------------------------------------------------------------
// RawForeignKey Tests (struct functionality)
// -----------------------------------------------------------------------------

func TestRawForeignKey(t *testing.T) {
	fk := RawForeignKey{
		Name:       "fk_posts_author",
		Columns:    []string{"author_id"},
		RefTable:   "auth_users",
		RefColumns: []string{"id"},
		OnDelete:   "CASCADE",
		OnUpdate:   "NO ACTION",
	}

	if fk.Name != "fk_posts_author" {
		t.Errorf("RawForeignKey.Name = %q, want %q", fk.Name, "fk_posts_author")
	}
	if len(fk.Columns) != 1 {
		t.Errorf("RawForeignKey.Columns = %d, want 1", len(fk.Columns))
	}
	if fk.RefTable != "auth_users" {
		t.Errorf("RawForeignKey.RefTable = %q, want %q", fk.RefTable, "auth_users")
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("RawForeignKey.OnDelete = %q, want %q", fk.OnDelete, "CASCADE")
	}
}

// -----------------------------------------------------------------------------
// Edge Cases
// -----------------------------------------------------------------------------

func TestParseTableNameEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		sqlName   string
		wantNS    string
		wantTable string
	}{
		{"consecutive_underscores", "a__b", "a", "_b"},
		{"many_underscores", "a_b_c_d_e", "a", "b_c_d_e"},
		{"only_underscores", "___", "", "__"},
		{"unicode", "ns_täble", "ns", "täble"},
		{"numbers", "ns1_table2", "ns1", "table2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, table := parseTableName(tt.sqlName, nil)
			if ns != tt.wantNS {
				t.Errorf("parseTableName(%q) namespace = %q, want %q", tt.sqlName, ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("parseTableName(%q) table = %q, want %q", tt.sqlName, table, tt.wantTable)
			}
		})
	}
}
