package dsl

import (
	"testing"
)

func TestNewTableBuilder(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	got := tb.Build()

	if got.Namespace != "auth" {
		t.Errorf("Namespace = %q, want %q", got.Namespace, "auth")
	}
	if got.Name != "users" {
		t.Errorf("Name = %q, want %q", got.Name, "users")
	}
	if got.Columns == nil {
		t.Error("Columns should be initialized")
	}
	if got.Indexes == nil {
		t.Error("Indexes should be initialized")
	}
	if got.Checks == nil {
		t.Error("Checks should be initialized")
	}
}

func TestTableBuilder_ID(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.ID()
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "id" {
		t.Errorf("Column name = %q, want %q", col.Name, "id")
	}
	if col.Type != "uuid" {
		t.Errorf("Column type = %q, want %q", col.Type, "uuid")
	}
	if !col.PrimaryKey {
		t.Error("Column should be primary key")
	}
	if !col.DefaultSet {
		t.Error("Column should have default set")
	}
}

func TestTableBuilder_String(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.String("email", 255)
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "email" {
		t.Errorf("Column name = %q, want %q", col.Name, "email")
	}
	if col.Type != "string" {
		t.Errorf("Column type = %q, want %q", col.Type, "string")
	}
	if len(col.TypeArgs) != 1 || col.TypeArgs[0] != 255 {
		t.Errorf("TypeArgs = %v, want [255]", col.TypeArgs)
	}
	if col.Nullable {
		t.Error("Column should be NOT NULL by default")
	}
}

func TestTableBuilder_Text(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.Text("content")
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "content" {
		t.Errorf("Column name = %q, want %q", col.Name, "content")
	}
	if col.Type != "text" {
		t.Errorf("Column type = %q, want %q", col.Type, "text")
	}
}

func TestTableBuilder_Integer(t *testing.T) {
	tb := NewTableBuilder("shop", "products")
	tb.Integer("quantity")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "integer" {
		t.Errorf("Column type = %q, want %q", col.Type, "integer")
	}
}

func TestTableBuilder_Float(t *testing.T) {
	tb := NewTableBuilder("shop", "products")
	tb.Float("rating")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "float" {
		t.Errorf("Column type = %q, want %q", col.Type, "float")
	}
}

func TestTableBuilder_Decimal(t *testing.T) {
	tb := NewTableBuilder("shop", "products")
	tb.Decimal("price", 10, 2)
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "decimal" {
		t.Errorf("Column type = %q, want %q", col.Type, "decimal")
	}
	if len(col.TypeArgs) != 2 {
		t.Fatalf("Expected 2 type args, got %d", len(col.TypeArgs))
	}
	if col.TypeArgs[0] != 10 || col.TypeArgs[1] != 2 {
		t.Errorf("TypeArgs = %v, want [10, 2]", col.TypeArgs)
	}
}

func TestTableBuilder_Boolean(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.Boolean("is_active")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "boolean" {
		t.Errorf("Column type = %q, want %q", col.Type, "boolean")
	}
}

func TestTableBuilder_Date(t *testing.T) {
	tb := NewTableBuilder("hr", "employees")
	tb.Date("birth_date")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "date" {
		t.Errorf("Column type = %q, want %q", col.Type, "date")
	}
}

func TestTableBuilder_Time(t *testing.T) {
	tb := NewTableBuilder("schedule", "slots")
	tb.Time("start_time")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "time" {
		t.Errorf("Column type = %q, want %q", col.Type, "time")
	}
}

func TestTableBuilder_DateTime(t *testing.T) {
	tb := NewTableBuilder("log", "events")
	tb.DateTime("occurred_at")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "date_time" {
		t.Errorf("Column type = %q, want %q", col.Type, "date_time")
	}
}

func TestTableBuilder_UUID(t *testing.T) {
	tb := NewTableBuilder("auth", "sessions")
	tb.UUID("token")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "uuid" {
		t.Errorf("Column type = %q, want %q", col.Type, "uuid")
	}
}

func TestTableBuilder_JSON(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.JSON("preferences")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "json" {
		t.Errorf("Column type = %q, want %q", col.Type, "json")
	}
}

func TestTableBuilder_Base64(t *testing.T) {
	tb := NewTableBuilder("files", "attachments")
	tb.Base64("data")
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "base64" {
		t.Errorf("Column type = %q, want %q", col.Type, "base64")
	}
}

func TestTableBuilder_Enum(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	values := []string{"active", "pending", "suspended"}
	tb.Enum("status", values)
	got := tb.Build()

	col := got.Columns[0]
	if col.Type != "enum" {
		t.Errorf("Column type = %q, want %q", col.Type, "enum")
	}
	if len(col.TypeArgs) != 1 {
		t.Fatalf("Expected 1 type arg, got %d", len(col.TypeArgs))
	}
	enumValues, ok := col.TypeArgs[0].([]string)
	if !ok {
		t.Fatal("TypeArgs[0] should be []string")
	}
	if len(enumValues) != 3 {
		t.Errorf("Enum values count = %d, want 3", len(enumValues))
	}
}

func TestTableBuilder_Timestamps(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.Timestamps()
	got := tb.Build()

	if len(got.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(got.Columns))
	}

	// Check created_at
	createdAt := got.Columns[0]
	if createdAt.Name != "created_at" {
		t.Errorf("First column name = %q, want %q", createdAt.Name, "created_at")
	}
	if createdAt.Type != "date_time" {
		t.Errorf("First column type = %q, want %q", createdAt.Type, "date_time")
	}
	if !createdAt.DefaultSet {
		t.Error("created_at should have default set")
	}

	// Check updated_at
	updatedAt := got.Columns[1]
	if updatedAt.Name != "updated_at" {
		t.Errorf("Second column name = %q, want %q", updatedAt.Name, "updated_at")
	}
	if updatedAt.Type != "date_time" {
		t.Errorf("Second column type = %q, want %q", updatedAt.Type, "date_time")
	}
	if !updatedAt.DefaultSet {
		t.Error("updated_at should have default set")
	}
}

func TestTableBuilder_SoftDelete(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.SoftDelete()
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "deleted_at" {
		t.Errorf("Column name = %q, want %q", col.Name, "deleted_at")
	}
	if col.Type != "date_time" {
		t.Errorf("Column type = %q, want %q", col.Type, "date_time")
	}
	if !col.Nullable {
		t.Error("deleted_at should be nullable")
	}
}

func TestTableBuilder_Sortable(t *testing.T) {
	tb := NewTableBuilder("cms", "items")
	tb.Sortable()
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "position" {
		t.Errorf("Column name = %q, want %q", col.Name, "position")
	}
	if col.Type != "integer" {
		t.Errorf("Column type = %q, want %q", col.Type, "integer")
	}
	if col.Default != 0 {
		t.Errorf("Default = %v, want 0", col.Default)
	}
}

func TestTableBuilder_Slugged(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.String("title", 100)
	tb.Slugged("title")
	got := tb.Build()

	if len(got.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(got.Columns))
	}

	slug := got.Columns[1]
	if slug.Name != "slug" {
		t.Errorf("Column name = %q, want %q", slug.Name, "slug")
	}
	if slug.Type != "string" {
		t.Errorf("Column type = %q, want %q", slug.Type, "string")
	}
	if !slug.Unique {
		t.Error("Slug column should be unique")
	}
	// Should use source column's max length (100)
	if len(slug.TypeArgs) != 1 || slug.TypeArgs[0] != 100 {
		t.Errorf("TypeArgs = %v, want [100]", slug.TypeArgs)
	}
}

func TestTableBuilder_Slugged_DefaultLength(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	// Slugged without matching source column
	tb.Slugged("nonexistent")
	got := tb.Build()

	slug := got.Columns[0]
	// Should use default length (200)
	if len(slug.TypeArgs) != 1 || slug.TypeArgs[0] != 200 {
		t.Errorf("TypeArgs = %v, want [200]", slug.TypeArgs)
	}
}

func TestTableBuilder_BelongsTo(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.BelongsTo("auth.users")
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "user_id" {
		t.Errorf("Column name = %q, want %q", col.Name, "user_id")
	}
	if col.Type != "uuid" {
		t.Errorf("Column type = %q, want %q", col.Type, "uuid")
	}
	if col.Reference == nil {
		t.Fatal("Reference should not be nil")
	}
	if col.Reference.Table != "auth.users" {
		t.Errorf("Reference.Table = %q, want %q", col.Reference.Table, "auth.users")
	}
	if col.Nullable {
		t.Error("BelongsTo should be NOT NULL by default")
	}

	// Check that index was auto-created
	if len(got.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(got.Indexes))
	}
	idx := got.Indexes[0]
	if len(idx.Columns) != 1 || idx.Columns[0] != "user_id" {
		t.Errorf("Index columns = %v, want [user_id]", idx.Columns)
	}
}

func TestTableBuilder_BelongsTo_WithOptions(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.BelongsTo("auth.users", As("author"), BelongsToNullable(), BelongsToOnDelete(Cascade))
	got := tb.Build()

	col := got.Columns[0]
	if col.Name != "author_id" {
		t.Errorf("Column name = %q, want %q", col.Name, "author_id")
	}
	if !col.Nullable {
		t.Error("Column should be nullable when BelongsToNullable() is used")
	}
	if col.Reference.OnDelete != Cascade {
		t.Errorf("OnDelete = %q, want %q", col.Reference.OnDelete, Cascade)
	}
}

func TestTableBuilder_BelongsTo_NoIndex(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.BelongsTo("auth.users", BelongsToNoIndex())
	got := tb.Build()

	if len(got.Indexes) != 0 {
		t.Errorf("Expected 0 indexes when BelongsToNoIndex() is used, got %d", len(got.Indexes))
	}
}

func TestTableBuilder_OneToOne(t *testing.T) {
	tb := NewTableBuilder("auth", "profiles")
	tb.OneToOne("auth.users")
	got := tb.Build()

	if len(got.Columns) != 1 {
		t.Fatalf("Expected 1 column, got %d", len(got.Columns))
	}

	col := got.Columns[0]
	if col.Name != "user_id" {
		t.Errorf("Column name = %q, want %q", col.Name, "user_id")
	}
	if !col.Unique {
		t.Error("OneToOne should create a unique column")
	}
}

func TestTableBuilder_ManyToMany(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	m2m := tb.ManyToMany("auth.roles")

	if m2m == nil {
		t.Fatal("ManyToMany should return a ManyToManyDef")
	}
	if m2m.OtherTable != "auth.roles" {
		t.Errorf("OtherTable = %q, want %q", m2m.OtherTable, "auth.roles")
	}
}

func TestTableBuilder_ManyToMany_Through(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	m2m := tb.ManyToMany("auth.roles", Through("user_roles"))

	if m2m.Through != "user_roles" {
		t.Errorf("Through = %q, want %q", m2m.Through, "user_roles")
	}
}

func TestTableBuilder_BelongsToAny(t *testing.T) {
	tb := NewTableBuilder("cms", "comments")
	tb.BelongsToAny([]string{"blog.posts", "cms.pages"})
	got := tb.Build()

	if len(got.Columns) != 2 {
		t.Fatalf("Expected 2 columns (type + id), got %d", len(got.Columns))
	}

	// Check type column
	typeCol := got.Columns[0]
	if typeCol.Name != "parent_type" {
		t.Errorf("Type column name = %q, want %q", typeCol.Name, "parent_type")
	}
	if typeCol.Type != "string" {
		t.Errorf("Type column type = %q, want %q", typeCol.Type, "string")
	}

	// Check id column
	idCol := got.Columns[1]
	if idCol.Name != "parent_id" {
		t.Errorf("ID column name = %q, want %q", idCol.Name, "parent_id")
	}
	if idCol.Type != "uuid" {
		t.Errorf("ID column type = %q, want %q", idCol.Type, "uuid")
	}

	// Check composite index
	if len(got.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(got.Indexes))
	}
	idx := got.Indexes[0]
	if len(idx.Columns) != 2 {
		t.Errorf("Expected 2 columns in index, got %d", len(idx.Columns))
	}
}

func TestTableBuilder_BelongsToAny_CustomName(t *testing.T) {
	tb := NewTableBuilder("cms", "comments")
	tb.BelongsToAny([]string{"blog.posts"}, PolymorphicAs("commentable"))
	got := tb.Build()

	typeCol := got.Columns[0]
	if typeCol.Name != "commentable_type" {
		t.Errorf("Type column name = %q, want %q", typeCol.Name, "commentable_type")
	}

	idCol := got.Columns[1]
	if idCol.Name != "commentable_id" {
		t.Errorf("ID column name = %q, want %q", idCol.Name, "commentable_id")
	}
}

func TestTableBuilder_Index(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.Index("email")
	got := tb.Build()

	if len(got.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(got.Indexes))
	}

	idx := got.Indexes[0]
	if idx.Unique {
		t.Error("Index() should not create a unique index")
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("Columns = %v, want [email]", idx.Columns)
	}
}

func TestTableBuilder_Index_MultiColumn(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.Index("first_name", "last_name")
	got := tb.Build()

	idx := got.Indexes[0]
	if len(idx.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(idx.Columns))
	}
	if idx.Columns[0] != "first_name" || idx.Columns[1] != "last_name" {
		t.Errorf("Columns = %v, want [first_name, last_name]", idx.Columns)
	}
}

func TestTableBuilder_Unique(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.Unique("email")
	got := tb.Build()

	if len(got.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(got.Indexes))
	}

	idx := got.Indexes[0]
	if !idx.Unique {
		t.Error("Unique() should create a unique index")
	}
}

func TestTableBuilder_Unique_Composite(t *testing.T) {
	tb := NewTableBuilder("blog", "subscriptions")
	tb.Unique("user_id", "blog_id")
	got := tb.Build()

	idx := got.Indexes[0]
	if !idx.Unique {
		t.Error("Unique() should create a unique index")
	}
	if len(idx.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(idx.Columns))
	}
}

func TestTableBuilder_Docs(t *testing.T) {
	tb := NewTableBuilder("auth", "users")
	tb.Docs("Application users table")
	got := tb.Build()

	if got.Docs != "Application users table" {
		t.Errorf("Docs = %q, want %q", got.Docs, "Application users table")
	}
}

func TestTableBuilder_Deprecated(t *testing.T) {
	tb := NewTableBuilder("auth", "old_users")
	tb.Deprecated("Use auth.users instead")
	got := tb.Build()

	if got.Deprecated != "Use auth.users instead" {
		t.Errorf("Deprecated = %q, want %q", got.Deprecated, "Use auth.users instead")
	}
}

func TestTableBuilder_CompleteTable(t *testing.T) {
	tb := NewTableBuilder("blog", "posts")
	tb.ID()
	tb.String("title", 200)
	tb.Text("content")
	tb.BelongsTo("auth.users", As("author"))
	tb.Timestamps()
	tb.SoftDelete()
	tb.Index("created_at")
	tb.Unique("author_id", "title")
	tb.Docs("Blog posts")
	got := tb.Build()

	// Check column count: id + title + content + author_id + created_at + updated_at + deleted_at = 7
	if len(got.Columns) != 7 {
		t.Errorf("Expected 7 columns, got %d", len(got.Columns))
	}

	// Check indexes: 1 from BelongsTo + 1 Index + 1 Unique = 3
	if len(got.Indexes) != 3 {
		t.Errorf("Expected 3 indexes, got %d", len(got.Indexes))
	}

	// Check docs
	if got.Docs != "Blog posts" {
		t.Errorf("Docs = %q, want %q", got.Docs, "Blog posts")
	}
}

func TestParseRefParts(t *testing.T) {
	tests := []struct {
		ref          string
		wantNS       string
		wantTable    string
		wantRelative bool
	}{
		{"auth.users", "auth", "users", false},
		{".roles", "", "roles", true},
		{"posts", "", "posts", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			ns, table, isRelative := parseRefParts(tt.ref)
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
