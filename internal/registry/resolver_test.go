package registry

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name         string
		ref          string
		wantNS       string
		wantTable    string
		wantRelative bool
	}{
		{
			name:         "fully qualified",
			ref:          "auth.users",
			wantNS:       "auth",
			wantTable:    "users",
			wantRelative: false,
		},
		{
			name:         "relative with dot",
			ref:          ".roles",
			wantNS:       "",
			wantTable:    "roles",
			wantRelative: true,
		},
		{
			name:         "unqualified",
			ref:          "posts",
			wantNS:       "",
			wantTable:    "posts",
			wantRelative: false,
		},
		{
			name:         "empty string",
			ref:          "",
			wantNS:       "",
			wantTable:    "",
			wantRelative: false,
		},
		{
			name:         "nested namespace",
			ref:          "app.auth.users",
			wantNS:       "app",
			wantTable:    "auth.users",
			wantRelative: false,
		},
		{
			name:         "just dot",
			ref:          ".",
			wantNS:       "",
			wantTable:    "",
			wantRelative: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, table, isRelative := ParseReference(tt.ref)
			if ns != tt.wantNS {
				t.Errorf("namespace = %q, want %q", ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("table = %q, want %q", table, tt.wantTable)
			}
			if isRelative != tt.wantRelative {
				t.Errorf("isRelative = %v, want %v", isRelative, tt.wantRelative)
			}
		})
	}
}

func TestNormalizeReference(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		currentNS string
		want      string
		wantErr   bool
		errCode   alerr.Code
	}{
		{
			name:      "fully qualified unchanged",
			ref:       "blog.posts",
			currentNS: "auth",
			want:      "blog.posts",
			wantErr:   false,
		},
		{
			name:      "relative resolves to current namespace",
			ref:       ".roles",
			currentNS: "auth",
			want:      "auth.roles",
			wantErr:   false,
		},
		{
			name:      "unqualified resolves to current namespace",
			ref:       "users",
			currentNS: "auth",
			want:      "auth.users",
			wantErr:   false,
		},
		{
			name:      "empty table name error",
			ref:       "auth.",
			currentNS: "test",
			want:      "",
			wantErr:   true,
			errCode:   alerr.ErrInvalidReference,
		},
		{
			name:      "relative without current namespace error",
			ref:       ".roles",
			currentNS: "",
			want:      "",
			wantErr:   true,
			errCode:   alerr.ErrInvalidReference,
		},
		{
			name:      "unqualified without current namespace error",
			ref:       "users",
			currentNS: "",
			want:      "",
			wantErr:   true,
			errCode:   alerr.ErrInvalidReference,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeReference(tt.ref, tt.currentNS)
			if tt.wantErr {
				if err == nil {
					t.Error("NormalizeReference() expected error, got nil")
				}
				if tt.errCode != "" && !alerr.Is(err, tt.errCode) {
					t.Errorf("Expected error code %v, got %v", tt.errCode, err)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeReference() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeReference() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModelRegistry_Resolve(t *testing.T) {
	r := NewModelRegistry()

	// Setup test tables
	authUsers := &ast.TableDef{
		Name:    "users",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	authRoles := &ast.TableDef{
		Name:    "roles",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	blogPosts := &ast.TableDef{
		Name:    "posts",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", authUsers)
	_ = r.Register("auth", "roles", authRoles)
	_ = r.Register("blog", "posts", blogPosts)

	tests := []struct {
		name      string
		ref       string
		currentNS string
		want      *ast.TableDef
		wantErr   bool
	}{
		{
			name:      "fully qualified reference",
			ref:       "auth.users",
			currentNS: "blog",
			want:      authUsers,
			wantErr:   false,
		},
		{
			name:      "relative reference",
			ref:       ".roles",
			currentNS: "auth",
			want:      authRoles,
			wantErr:   false,
		},
		{
			name:      "unqualified reference",
			ref:       "posts",
			currentNS: "blog",
			want:      blogPosts,
			wantErr:   false,
		},
		{
			name:      "cross-namespace fully qualified",
			ref:       "blog.posts",
			currentNS: "auth",
			want:      blogPosts,
			wantErr:   false,
		},
		{
			name:      "not found",
			ref:       "auth.nonexistent",
			currentNS: "auth",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "relative not found",
			ref:       ".nonexistent",
			currentNS: "auth",
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.ref, tt.currentNS)
			if tt.wantErr {
				if err == nil {
					t.Error("Resolve() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Resolve() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelRegistry_SQLName(t *testing.T) {
	r := NewModelRegistry()

	tests := []struct {
		name      string
		ref       string
		currentNS string
		want      string
		wantErr   bool
	}{
		{
			name:      "fully qualified",
			ref:       "blog.posts",
			currentNS: "auth",
			want:      "blog_posts",
			wantErr:   false,
		},
		{
			name:      "relative reference",
			ref:       ".roles",
			currentNS: "auth",
			want:      "auth_roles",
			wantErr:   false,
		},
		{
			name:      "unqualified reference",
			ref:       "users",
			currentNS: "auth",
			want:      "auth_users",
			wantErr:   false,
		},
		{
			name:      "unqualified without namespace error",
			ref:       "users",
			currentNS: "",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.SQLName(tt.ref, tt.currentNS)
			if tt.wantErr {
				if err == nil {
					t.Error("SQLName() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("SQLName() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("SQLName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModelRegistry_SQLNameChecked(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", def)

	// Should succeed for registered table
	got, err := r.SQLNameChecked("auth.users", "")
	if err != nil {
		t.Errorf("SQLNameChecked() error = %v", err)
	}
	if got != "auth_users" {
		t.Errorf("SQLNameChecked() = %q, want %q", got, "auth_users")
	}

	// Should fail for unregistered table
	_, err = r.SQLNameChecked("auth.nonexistent", "")
	if err == nil {
		t.Error("SQLNameChecked() should error for unregistered table")
	}
}

func TestModelRegistry_ValidateReferences(t *testing.T) {
	r := NewModelRegistry()

	// Setup test tables
	users := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", users)

	// Table with valid reference
	validTable := &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid"},
			{
				Name: "user_id",
				Type: "uuid",
				Reference: &ast.Reference{
					Table:  "auth.users",
					Column: "id",
				},
			},
		},
	}

	// Table with invalid reference
	invalidTable := &ast.TableDef{
		Namespace: "blog",
		Name:      "comments",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid"},
			{
				Name: "user_id",
				Type: "uuid",
				Reference: &ast.Reference{
					Table:  "auth.nonexistent",
					Column: "id",
				},
			},
		},
	}

	// Table with empty reference table
	emptyRefTable := &ast.TableDef{
		Namespace: "blog",
		Name:      "items",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid"},
			{
				Name: "ref_id",
				Type: "uuid",
				Reference: &ast.Reference{
					Table:  "",
					Column: "id",
				},
			},
		},
	}

	t.Run("valid references", func(t *testing.T) {
		err := r.ValidateReferences(validTable)
		if err != nil {
			t.Errorf("ValidateReferences() should pass for valid references: %v", err)
		}
	})

	t.Run("invalid references", func(t *testing.T) {
		err := r.ValidateReferences(invalidTable)
		if err == nil {
			t.Error("ValidateReferences() should fail for invalid references")
		}
		if !alerr.Is(err, alerr.ErrInvalidReference) {
			t.Errorf("Expected ErrInvalidReference, got %v", err)
		}
	})

	t.Run("empty reference table", func(t *testing.T) {
		err := r.ValidateReferences(emptyRefTable)
		if err == nil {
			t.Error("ValidateReferences() should fail for empty reference table")
		}
	})

	t.Run("nil table definition", func(t *testing.T) {
		err := r.ValidateReferences(nil)
		if err == nil {
			t.Error("ValidateReferences() should fail for nil table")
		}
	})

	t.Run("table with no references", func(t *testing.T) {
		noRefsTable := &ast.TableDef{
			Namespace: "test",
			Name:      "simple",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid"},
				{Name: "name", Type: "string"},
			},
		}
		err := r.ValidateReferences(noRefsTable)
		if err != nil {
			t.Errorf("ValidateReferences() should pass for table with no references: %v", err)
		}
	})
}

func TestModelRegistry_ValidateAllReferences(t *testing.T) {
	r := NewModelRegistry()

	// Setup test tables
	users := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	posts := &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid"},
			{
				Name: "user_id",
				Type: "uuid",
				Reference: &ast.Reference{
					Table:  "auth.users",
					Column: "id",
				},
			},
		},
	}
	_ = r.Register("auth", "users", users)
	_ = r.Register("blog", "posts", posts)

	t.Run("all valid", func(t *testing.T) {
		errs := r.ValidateAllReferences()
		if len(errs) != 0 {
			t.Errorf("ValidateAllReferences() returned %d errors, want 0", len(errs))
		}
	})

	// Add a table with invalid reference
	badTable := &ast.TableDef{
		Namespace: "test",
		Name:      "bad",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid"},
			{
				Name: "ref_id",
				Type: "uuid",
				Reference: &ast.Reference{
					Table:  "nonexistent.table",
					Column: "id",
				},
			},
		},
	}
	_ = r.Register("test", "bad", badTable)

	t.Run("some invalid", func(t *testing.T) {
		errs := r.ValidateAllReferences()
		if len(errs) != 1 {
			t.Errorf("ValidateAllReferences() returned %d errors, want 1", len(errs))
		}
	})
}

func TestModelRegistry_ResolveColumnReference(t *testing.T) {
	r := NewModelRegistry()

	users := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", users)

	t.Run("valid column reference", func(t *testing.T) {
		col := &ast.ColumnDef{
			Name: "user_id",
			Type: "uuid",
			Reference: &ast.Reference{
				Table:  "auth.users",
				Column: "id",
			},
		}
		got, err := r.ResolveColumnReference(col, "blog")
		if err != nil {
			t.Errorf("ResolveColumnReference() error = %v", err)
		}
		if got != users {
			t.Error("ResolveColumnReference() returned wrong table")
		}
	})

	t.Run("relative column reference", func(t *testing.T) {
		col := &ast.ColumnDef{
			Name: "author_id",
			Type: "uuid",
			Reference: &ast.Reference{
				Table:  ".users",
				Column: "id",
			},
		}
		got, err := r.ResolveColumnReference(col, "auth")
		if err != nil {
			t.Errorf("ResolveColumnReference() error = %v", err)
		}
		if got != users {
			t.Error("ResolveColumnReference() returned wrong table")
		}
	})

	t.Run("nil column", func(t *testing.T) {
		_, err := r.ResolveColumnReference(nil, "auth")
		if err == nil {
			t.Error("ResolveColumnReference() should error on nil column")
		}
	})

	t.Run("column without reference", func(t *testing.T) {
		col := &ast.ColumnDef{
			Name: "name",
			Type: "string",
		}
		_, err := r.ResolveColumnReference(col, "auth")
		if err == nil {
			t.Error("ResolveColumnReference() should error on column without reference")
		}
	})
}

func TestParseReferenceEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		ref          string
		wantNS       string
		wantTable    string
		wantRelative bool
	}{
		{
			name:         "single character namespace",
			ref:          "a.users",
			wantNS:       "a",
			wantTable:    "users",
			wantRelative: false,
		},
		{
			name:         "single character table",
			ref:          "auth.u",
			wantNS:       "auth",
			wantTable:    "u",
			wantRelative: false,
		},
		{
			name:         "underscore in names",
			ref:          "my_app.user_profiles",
			wantNS:       "my_app",
			wantTable:    "user_profiles",
			wantRelative: false,
		},
		{
			name:         "numbers in names",
			ref:          "app2.users123",
			wantNS:       "app2",
			wantTable:    "users123",
			wantRelative: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, table, isRelative := ParseReference(tt.ref)
			if ns != tt.wantNS {
				t.Errorf("namespace = %q, want %q", ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("table = %q, want %q", table, tt.wantTable)
			}
			if isRelative != tt.wantRelative {
				t.Errorf("isRelative = %v, want %v", isRelative, tt.wantRelative)
			}
		})
	}
}

func TestNormalizeReference_FullyQualifiedPassthrough(t *testing.T) {
	// Fully qualified references should pass through unchanged
	// even with different currentNS values
	tests := []struct {
		ref       string
		currentNS string
	}{
		{"auth.users", "blog"},
		{"auth.users", ""},
		{"auth.users", "auth"},
		{"billing.invoices", "shop"},
	}

	for _, tt := range tests {
		t.Run(tt.ref+"_in_"+tt.currentNS, func(t *testing.T) {
			got, err := NormalizeReference(tt.ref, tt.currentNS)
			if err != nil {
				t.Errorf("NormalizeReference() error = %v", err)
			}
			if got != tt.ref {
				t.Errorf("NormalizeReference() = %q, want %q (unchanged)", got, tt.ref)
			}
		})
	}
}
