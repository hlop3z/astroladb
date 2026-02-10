package runtime

import (
	"testing"
)

// TestNewReferenceBuilder tests the constructor
func TestNewReferenceBuilder(t *testing.T) {
	rb := NewReferenceBuilder()
	if rb == nil {
		t.Fatal("NewReferenceBuilder should return a non-nil builder")
	}
}

// TestBuildFK_BasicOptions tests building FK with basic options
func TestBuildFK_BasicOptions(t *testing.T) {
	rb := NewReferenceBuilder()

	col, idx := rb.BuildFK("auth.users", "author", ReferenceOpts{})

	// Verify column
	if col == nil {
		t.Fatal("Expected non-nil column")
	}
	if col.Name != "author_id" {
		t.Errorf("Expected column name 'author_id', got '%s'", col.Name)
	}
	if col.Type != "uuid" {
		t.Errorf("Expected type 'uuid', got '%s'", col.Type)
	}
	if col.Reference == nil {
		t.Fatal("Expected reference to be set")
	}
	if col.Reference.Table != "auth.users" {
		t.Errorf("Expected reference table 'auth.users', got '%s'", col.Reference.Table)
	}
	if col.Reference.Column != "id" {
		t.Errorf("Expected reference column 'id', got '%s'", col.Reference.Column)
	}
	if col.XRef != "auth.users" {
		t.Errorf("Expected XRef 'auth.users', got '%s'", col.XRef)
	}
	if !col.IsRelationship {
		t.Error("Expected IsRelationship to be true")
	}

	// Verify index
	if idx == nil {
		t.Fatal("Expected non-nil index")
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "author_id" {
		t.Errorf("Expected index columns ['author_id'], got %v", idx.Columns)
	}
	if idx.Unique {
		t.Error("Expected index to not be unique by default")
	}
}

// TestBuildFK_WithNullable tests building nullable FK
func TestBuildFK_WithNullable(t *testing.T) {
	rb := NewReferenceBuilder()

	col, _ := rb.BuildFK("users", "manager", ReferenceOpts{Nullable: true})

	if !col.Nullable {
		t.Error("Expected column to be nullable")
	}
}

// TestBuildFK_WithUnique tests building unique FK
func TestBuildFK_WithUnique(t *testing.T) {
	rb := NewReferenceBuilder()

	col, idx := rb.BuildFK("profiles", "user", ReferenceOpts{Unique: true})

	if !col.Unique {
		t.Error("Expected column to be unique")
	}
	if !idx.Unique {
		t.Error("Expected index to be unique")
	}
}

// TestBuildFK_WithCascade tests building FK with cascade delete
func TestBuildFK_WithCascade(t *testing.T) {
	rb := NewReferenceBuilder()

	col, _ := rb.BuildFK("organizations", "org", ReferenceOpts{
		OnDelete: "CASCADE",
		OnUpdate: "RESTRICT",
	})

	if col.Reference.OnDelete != "CASCADE" {
		t.Errorf("Expected on_delete 'CASCADE', got '%s'", col.Reference.OnDelete)
	}
	if col.Reference.OnUpdate != "RESTRICT" {
		t.Errorf("Expected on_update 'RESTRICT', got '%s'", col.Reference.OnUpdate)
	}
}

// TestBuildFK_WithAllOptions tests building FK with all options
func TestBuildFK_WithAllOptions(t *testing.T) {
	rb := NewReferenceBuilder()

	col, idx := rb.BuildFK("core.users", "created_by", ReferenceOpts{
		Nullable: true,
		Unique:   true,
		OnDelete: "SET NULL",
		OnUpdate: "CASCADE",
	})

	// Verify all options applied
	if col.Name != "created_by_id" {
		t.Errorf("Expected column name 'created_by_id', got '%s'", col.Name)
	}
	if !col.Nullable {
		t.Error("Expected nullable to be true")
	}
	if !col.Unique {
		t.Error("Expected unique to be true")
	}
	if col.Reference.OnDelete != "SET NULL" {
		t.Errorf("Expected on_delete 'SET NULL', got '%s'", col.Reference.OnDelete)
	}
	if col.Reference.OnUpdate != "CASCADE" {
		t.Errorf("Expected on_update 'CASCADE', got '%s'", col.Reference.OnUpdate)
	}
	if !idx.Unique {
		t.Error("Expected index to be unique")
	}
}

// TestBuildBelongsTo tests building standard belongs_to relationship
func TestBuildBelongsTo(t *testing.T) {
	rb := NewReferenceBuilder()

	col, idx := rb.BuildBelongsTo("blog.posts", "post")

	// Verify it creates standard non-nullable, non-unique FK
	if col.Name != "post_id" {
		t.Errorf("Expected column name 'post_id', got '%s'", col.Name)
	}
	if col.Type != "uuid" {
		t.Errorf("Expected type 'uuid', got '%s'", col.Type)
	}
	if col.Nullable {
		t.Error("Expected nullable to be false for belongs_to")
	}
	if col.Unique {
		t.Error("Expected unique to be false for belongs_to")
	}
	if col.Reference == nil {
		t.Fatal("Expected reference to be set")
	}
	if col.Reference.Table != "blog.posts" {
		t.Errorf("Expected reference table 'blog.posts', got '%s'", col.Reference.Table)
	}

	// Verify index
	if idx == nil {
		t.Fatal("Expected non-nil index")
	}
	if idx.Unique {
		t.Error("Expected index to not be unique for belongs_to")
	}
}

// TestBuildOneToOne tests building one_to_one relationship
func TestBuildOneToOne(t *testing.T) {
	rb := NewReferenceBuilder()

	col, idx := rb.BuildOneToOne("auth.users", "user")

	// Verify it creates unique FK
	if col.Name != "user_id" {
		t.Errorf("Expected column name 'user_id', got '%s'", col.Name)
	}
	if !col.Unique {
		t.Error("Expected unique to be true for one_to_one")
	}
	if col.Nullable {
		t.Error("Expected nullable to be false for one_to_one")
	}

	// Verify unique index
	if !idx.Unique {
		t.Error("Expected index to be unique for one_to_one")
	}
}

// TestBuildPolymorphic_Basic tests building basic polymorphic relationship
func TestBuildPolymorphic_Basic(t *testing.T) {
	rb := NewReferenceBuilder()

	targets := []string{"posts", "comments"}
	result := rb.BuildPolymorphic(targets, "commentable", false)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify type column
	if result.TypeColumn == nil {
		t.Fatal("Expected type column")
	}
	if result.TypeColumn.Name != "commentable_type" {
		t.Errorf("Expected type column 'commentable_type', got '%s'", result.TypeColumn.Name)
	}
	if result.TypeColumn.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", result.TypeColumn.Type)
	}
	if len(result.TypeColumn.TypeArgs) != 1 || result.TypeColumn.TypeArgs[0] != 100 {
		t.Errorf("Expected type_args [100], got %v", result.TypeColumn.TypeArgs)
	}
	if result.TypeColumn.Nullable {
		t.Error("Expected type column to not be nullable")
	}

	// Verify ID column
	if result.IDColumn == nil {
		t.Fatal("Expected ID column")
	}
	if result.IDColumn.Name != "commentable_id" {
		t.Errorf("Expected ID column 'commentable_id', got '%s'", result.IDColumn.Name)
	}
	if result.IDColumn.Type != "uuid" {
		t.Errorf("Expected type 'uuid', got '%s'", result.IDColumn.Type)
	}
	if result.IDColumn.Nullable {
		t.Error("Expected ID column to not be nullable")
	}

	// Verify index
	if result.Index == nil {
		t.Fatal("Expected index")
	}
	if len(result.Index.Columns) != 2 {
		t.Errorf("Expected 2 index columns, got %d", len(result.Index.Columns))
	}
	if result.Index.Columns[0] != "commentable_type" || result.Index.Columns[1] != "commentable_id" {
		t.Errorf("Expected index columns ['commentable_type', 'commentable_id'], got %v", result.Index.Columns)
	}
	if result.Index.Unique {
		t.Error("Expected index to not be unique")
	}
}

// TestBuildPolymorphic_Nullable tests building nullable polymorphic relationship
func TestBuildPolymorphic_Nullable(t *testing.T) {
	rb := NewReferenceBuilder()

	targets := []string{"users", "teams"}
	result := rb.BuildPolymorphic(targets, "owner", true)

	// Verify both columns are nullable
	if !result.TypeColumn.Nullable {
		t.Error("Expected type column to be nullable")
	}
	if !result.IDColumn.Nullable {
		t.Error("Expected ID column to be nullable")
	}
}

// TestBuildPolymorphic_DifferentAliases tests various alias names
func TestBuildPolymorphic_DifferentAliases(t *testing.T) {
	rb := NewReferenceBuilder()

	tests := []struct {
		alias        string
		expectedType string
		expectedID   string
	}{
		{"taggable", "taggable_type", "taggable_id"},
		{"parent", "parent_type", "parent_id"},
		{"imageable", "imageable_type", "imageable_id"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			result := rb.BuildPolymorphic([]string{"posts", "users"}, tt.alias, false)

			if result.TypeColumn.Name != tt.expectedType {
				t.Errorf("Expected type column '%s', got '%s'", tt.expectedType, result.TypeColumn.Name)
			}
			if result.IDColumn.Name != tt.expectedID {
				t.Errorf("Expected ID column '%s', got '%s'", tt.expectedID, result.IDColumn.Name)
			}
		})
	}
}

// TestReferenceExtractTableName tests extracting table names from references
func TestReferenceExtractTableName(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{"with namespace", "auth.users", "users"},
		{"with deep namespace", "core.admin.users", "users"},
		{"without namespace", "posts", "posts"},
		{"single char table", "auth.u", "u"},
		{"single char namespace", "a.users", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTableName(tt.ref)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestDefaultColumnName tests generating default FK column names
func TestDefaultColumnName(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{"with namespace", "auth.users", "users_id"},
		{"with deep namespace", "core.admin.teams", "teams_id"},
		{"without namespace", "organizations", "organizations_id"},
		{"singular table", "auth.user", "user_id"},
		{"short name", "blog.post", "post_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultColumnName(tt.ref)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestNewManyToMany tests creating many_to_many metadata
func TestNewManyToMany(t *testing.T) {
	meta := NewManyToMany("tags", "tags")

	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}
	if meta.Type != "many_to_many" {
		t.Errorf("Expected type 'many_to_many', got '%s'", meta.Type)
	}
	if meta.Target != "tags" {
		t.Errorf("Expected target 'tags', got '%s'", meta.Target)
	}
	if meta.Alias != "tags" {
		t.Errorf("Expected alias 'tags', got '%s'", meta.Alias)
	}
}

// TestNewManyToMany_WithNamespace tests many_to_many with namespaced target
func TestNewManyToMany_WithNamespace(t *testing.T) {
	meta := NewManyToMany("blog.categories", "categories")

	if meta.Target != "blog.categories" {
		t.Errorf("Expected target 'blog.categories', got '%s'", meta.Target)
	}
}

// TestNewManyToMany_DifferentAlias tests many_to_many with different alias
func TestNewManyToMany_DifferentAlias(t *testing.T) {
	meta := NewManyToMany("auth.users", "collaborators")

	if meta.Target != "auth.users" {
		t.Errorf("Expected target 'auth.users', got '%s'", meta.Target)
	}
	if meta.Alias != "collaborators" {
		t.Errorf("Expected alias 'collaborators', got '%s'", meta.Alias)
	}
}

// TestNewPolymorphic tests creating polymorphic metadata
func TestNewPolymorphic(t *testing.T) {
	targets := []string{"posts", "comments", "videos"}
	meta := NewPolymorphic(targets, "commentable")

	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}
	if meta.Type != "polymorphic" {
		t.Errorf("Expected type 'polymorphic', got '%s'", meta.Type)
	}
	if len(meta.Targets) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(meta.Targets))
	}
	if meta.Targets[0] != "posts" || meta.Targets[1] != "comments" || meta.Targets[2] != "videos" {
		t.Errorf("Expected targets [posts, comments, videos], got %v", meta.Targets)
	}
	if meta.Alias != "commentable" {
		t.Errorf("Expected alias 'commentable', got '%s'", meta.Alias)
	}
}

// TestNewPolymorphic_SingleTarget tests polymorphic with single target
func TestNewPolymorphic_SingleTarget(t *testing.T) {
	targets := []string{"users"}
	meta := NewPolymorphic(targets, "owner")

	if len(meta.Targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(meta.Targets))
	}
	if meta.Targets[0] != "users" {
		t.Errorf("Expected target 'users', got '%s'", meta.Targets[0])
	}
}

// TestNewPolymorphic_WithNamespaces tests polymorphic with namespaced targets
func TestNewPolymorphic_WithNamespaces(t *testing.T) {
	targets := []string{"blog.posts", "core.pages", "media.videos"}
	meta := NewPolymorphic(targets, "attachable")

	if len(meta.Targets) != 3 {
		t.Errorf("Expected 3 targets, got %d", len(meta.Targets))
	}
	if meta.Targets[0] != "blog.posts" {
		t.Errorf("Expected first target 'blog.posts', got '%s'", meta.Targets[0])
	}
	if meta.Targets[1] != "core.pages" {
		t.Errorf("Expected second target 'core.pages', got '%s'", meta.Targets[1])
	}
	if meta.Targets[2] != "media.videos" {
		t.Errorf("Expected third target 'media.videos', got '%s'", meta.Targets[2])
	}
}

// TestReferenceBuilder_CompleteWorkflow tests a complete reference building workflow
func TestReferenceBuilder_CompleteWorkflow(t *testing.T) {
	rb := NewReferenceBuilder()

	// Build belongs_to
	authorCol, authorIdx := rb.BuildBelongsTo("auth.users", "author")
	if authorCol.Name != "author_id" {
		t.Errorf("Expected author_id, got %s", authorCol.Name)
	}
	if authorIdx.Unique {
		t.Error("BelongsTo index should not be unique")
	}

	// Build one_to_one
	profileCol, profileIdx := rb.BuildOneToOne("profiles", "profile")
	if profileCol.Name != "profile_id" {
		t.Errorf("Expected profile_id, got %s", profileCol.Name)
	}
	if !profileIdx.Unique {
		t.Error("OneToOne index should be unique")
	}

	// Build polymorphic
	polyResult := rb.BuildPolymorphic([]string{"posts", "comments"}, "commentable", false)
	if polyResult.TypeColumn.Name != "commentable_type" {
		t.Errorf("Expected commentable_type, got %s", polyResult.TypeColumn.Name)
	}
	if polyResult.IDColumn.Name != "commentable_id" {
		t.Errorf("Expected commentable_id, got %s", polyResult.IDColumn.Name)
	}

	// Create metadata
	m2mMeta := NewManyToMany("tags", "tags")
	if m2mMeta.Type != "many_to_many" {
		t.Errorf("Expected many_to_many, got %s", m2mMeta.Type)
	}

	polyMeta := NewPolymorphic([]string{"users", "teams"}, "owner")
	if polyMeta.Type != "polymorphic" {
		t.Errorf("Expected polymorphic, got %s", polyMeta.Type)
	}
}
