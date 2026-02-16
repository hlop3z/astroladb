package schema

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/runtime/builder"
)

// TestNewColumnConverter tests the constructor
func TestNewColumnConverter(t *testing.T) {
	converter := NewColumnConverter()
	if converter == nil {
		t.Fatal("Expected non-nil converter")
	}
}

// TestToAST tests column conversion to AST
func TestToAST(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("nil column", func(t *testing.T) {
		result := converter.ToAST(nil)
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("basic column", func(t *testing.T) {
		col := &builder.ColumnDef{
			Name:     "username",
			Type:     "string",
			TypeArgs: []any{255},
			Nullable: false,
			Unique:   true,
		}

		result := converter.ToAST(col)
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Name != "username" {
			t.Errorf("Name = %q, want %q", result.Name, "username")
		}
		if result.Type != "string" {
			t.Errorf("Type = %q, want %q", result.Type, "string")
		}
		if !result.Unique {
			t.Error("Expected Unique to be true")
		}
		if result.Nullable {
			t.Error("Expected Nullable to be false")
		}
	})

	t.Run("column with default value", func(t *testing.T) {
		defaultValue := "active"
		col := &builder.ColumnDef{
			Name:    "status",
			Type:    "string",
			Default: defaultValue,
		}

		result := converter.ToAST(col)
		if !result.DefaultSet {
			t.Error("Expected DefaultSet to be true")
		}
		if result.Default != defaultValue {
			t.Errorf("Default = %v, want %v", result.Default, defaultValue)
		}
	})

	t.Run("column with backfill", func(t *testing.T) {
		backfillValue := "unknown"
		col := &builder.ColumnDef{
			Name:     "category",
			Type:     "string",
			Backfill: backfillValue,
		}

		result := converter.ToAST(col)
		if !result.BackfillSet {
			t.Error("Expected BackfillSet to be true")
		}
		if result.Backfill != backfillValue {
			t.Errorf("Backfill = %v, want %v", result.Backfill, backfillValue)
		}
	})

	t.Run("column with min and max", func(t *testing.T) {
		min := 18.0
		max := 100.0
		col := &builder.ColumnDef{
			Name: "age",
			Type: "integer",
			Min:  &min,
			Max:  &max,
		}

		result := converter.ToAST(col)
		if result.Min == nil {
			t.Fatal("Expected Min to be set")
		}
		if *result.Min != 18 {
			t.Errorf("Min = %v, want %v", *result.Min, 18.0)
		}
		if result.Max == nil {
			t.Fatal("Expected Max to be set")
		}
		if *result.Max != 100 {
			t.Errorf("Max = %v, want %v", *result.Max, 100.0)
		}
	})

	t.Run("column with reference", func(t *testing.T) {
		col := &builder.ColumnDef{
			Name: "user_id",
			Type: "uuid",
			Reference: &builder.RefDef{
				Table:    "users",
				Column:   "id",
				OnDelete: "CASCADE",
			},
		}

		result := converter.ToAST(col)
		if result.Reference == nil {
			t.Fatal("Expected Reference to be set")
		}
		if result.Reference.Table != "users" {
			t.Errorf("Reference.Table = %q, want %q", result.Reference.Table, "users")
		}
		if result.Reference.Column != "id" {
			t.Errorf("Reference.Column = %q, want %q", result.Reference.Column, "id")
		}
		if result.Reference.OnDelete != "CASCADE" {
			t.Errorf("Reference.OnDelete = %q, want %q", result.Reference.OnDelete, "CASCADE")
		}
	})

	t.Run("column with computed expression", func(t *testing.T) {
		computed := "UPPER(name)"
		col := &builder.ColumnDef{
			Name:     "name_upper",
			Type:     "string",
			Computed: &computed,
		}

		result := converter.ToAST(col)
		if result.Computed == nil {
			t.Fatal("Expected Computed to be set")
		}
		// Computed is type 'any' in AST, should be the same pointer
		if result.Computed != col.Computed {
			t.Errorf("Computed = %v, want %v", result.Computed, col.Computed)
		}
	})

	t.Run("NullableSet always true", func(t *testing.T) {
		col := &builder.ColumnDef{
			Name:     "status",
			Type:     "string",
			Nullable: false,
		}

		result := converter.ToAST(col)
		if !result.NullableSet {
			t.Error("NullableSet should always be true (builder explicitly sets Nullable)")
		}
		if result.Nullable {
			t.Error("Nullable should be false")
		}

		// Also test with nullable = true
		col2 := &builder.ColumnDef{
			Name:     "bio",
			Type:     "text",
			Nullable: true,
		}
		result2 := converter.ToAST(col2)
		if !result2.NullableSet {
			t.Error("NullableSet should be true for nullable column too")
		}
		if !result2.Nullable {
			t.Error("Nullable should be true")
		}
	})

	t.Run("Virtual column", func(t *testing.T) {
		col := &builder.ColumnDef{
			Name:    "full_name",
			Type:    "string",
			Virtual: true,
		}

		result := converter.ToAST(col)
		if !result.Virtual {
			t.Error("Virtual should be true")
		}

		// Non-virtual column
		col2 := &builder.ColumnDef{
			Name:    "email",
			Type:    "string",
			Virtual: false,
		}
		result2 := converter.ToAST(col2)
		if result2.Virtual {
			t.Error("Virtual should be false")
		}
	})

	t.Run("column with metadata", func(t *testing.T) {
		col := &builder.ColumnDef{
			Name:       "email",
			Type:       "string",
			Format:     "email",
			Pattern:    "^[a-z]+@[a-z]+\\.[a-z]+$",
			Docs:       "User email address",
			Deprecated: "Use contact_email instead",
		}

		result := converter.ToAST(col)
		if result.Format != "email" {
			t.Errorf("Format = %q, want %q", result.Format, "email")
		}
		if result.Pattern != col.Pattern {
			t.Errorf("Pattern = %q, want %q", result.Pattern, col.Pattern)
		}
		if result.Docs != "User email address" {
			t.Errorf("Docs = %q, want %q", result.Docs, "User email address")
		}
		if result.Deprecated != "Use contact_email instead" {
			t.Errorf("Deprecated = %q, want %q", result.Deprecated, "Use contact_email instead")
		}
	})
}

// TestRefToAST tests reference conversion
func TestRefToAST(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("nil reference", func(t *testing.T) {
		result := converter.RefToAST(nil)
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("reference with column", func(t *testing.T) {
		ref := &builder.RefDef{
			Table:    "users",
			Column:   "id",
			OnDelete: "CASCADE",
			OnUpdate: "RESTRICT",
		}

		result := converter.RefToAST(ref)
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Table != "users" {
			t.Errorf("Table = %q, want %q", result.Table, "users")
		}
		if result.Column != "id" {
			t.Errorf("Column = %q, want %q", result.Column, "id")
		}
		if result.OnDelete != "CASCADE" {
			t.Errorf("OnDelete = %q, want %q", result.OnDelete, "CASCADE")
		}
		if result.OnUpdate != "RESTRICT" {
			t.Errorf("OnUpdate = %q, want %q", result.OnUpdate, "RESTRICT")
		}
	})

	t.Run("reference without column defaults to id", func(t *testing.T) {
		ref := &builder.RefDef{
			Table:    "posts",
			OnDelete: "SET NULL",
		}

		result := converter.RefToAST(ref)
		if result.Column != "id" {
			t.Errorf("Column = %q, want %q (should default to 'id')", result.Column, "id")
		}
	})
}

// TestIndexToAST tests index conversion
func TestIndexToAST(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("nil index", func(t *testing.T) {
		result := converter.IndexToAST(nil)
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("basic index", func(t *testing.T) {
		idx := &builder.IndexDef{
			Name:    "idx_email",
			Columns: []string{"email"},
			Unique:  true,
		}

		result := converter.IndexToAST(idx)
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Name != "idx_email" {
			t.Errorf("Name = %q, want %q", result.Name, "idx_email")
		}
		if len(result.Columns) != 1 || result.Columns[0] != "email" {
			t.Errorf("Columns = %v, want [email]", result.Columns)
		}
		if !result.Unique {
			t.Error("Expected Unique to be true")
		}
	})

	t.Run("composite index with where clause", func(t *testing.T) {
		where := "deleted_at IS NULL"
		idx := &builder.IndexDef{
			Name:    "idx_active_users",
			Columns: []string{"email", "created_at"},
			Unique:  false,
			Where:   where,
		}

		result := converter.IndexToAST(idx)
		if len(result.Columns) != 2 {
			t.Errorf("Columns length = %d, want 2", len(result.Columns))
		}
		if result.Where == "" {
			t.Fatal("Expected Where to be set")
		}
		if result.Where != where {
			t.Errorf("Where = %q, want %q", result.Where, where)
		}
	})
}

// TestColumnsToAST tests batch column conversion
func TestColumnsToAST(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("empty slice", func(t *testing.T) {
		result := converter.ColumnsToAST([]*builder.ColumnDef{})
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(result))
		}
	})

	t.Run("multiple columns", func(t *testing.T) {
		cols := []*builder.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", Unique: true},
			{Name: "created_at", Type: "datetime"},
		}

		result := converter.ColumnsToAST(cols)
		if len(result) != 3 {
			t.Errorf("Expected 3 columns, got %d", len(result))
		}
		if result[0].Name != "id" || !result[0].PrimaryKey {
			t.Error("First column should be id with PrimaryKey=true")
		}
		if result[1].Name != "email" || !result[1].Unique {
			t.Error("Second column should be email with Unique=true")
		}
		if result[2].Name != "created_at" {
			t.Error("Third column should be created_at")
		}
	})

	t.Run("filters nil columns", func(t *testing.T) {
		cols := []*builder.ColumnDef{
			{Name: "id", Type: "uuid"},
			nil,
			{Name: "name", Type: "string"},
		}

		result := converter.ColumnsToAST(cols)
		// Nil columns should be filtered out by the nil check in ToAST
		if len(result) != 2 {
			t.Errorf("Expected 2 columns (nil filtered), got %d", len(result))
		}
	})
}

// TestIndexesToAST tests batch index conversion
func TestIndexesToAST(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("empty slice", func(t *testing.T) {
		result := converter.IndexesToAST([]*builder.IndexDef{})
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got length %d", len(result))
		}
	})

	t.Run("multiple indexes", func(t *testing.T) {
		idxs := []*builder.IndexDef{
			{Name: "idx_email", Columns: []string{"email"}, Unique: true},
			{Name: "idx_created", Columns: []string{"created_at"}},
		}

		result := converter.IndexesToAST(idxs)
		if len(result) != 2 {
			t.Errorf("Expected 2 indexes, got %d", len(result))
		}
		if result[0].Name != "idx_email" {
			t.Error("First index should be idx_email")
		}
		if result[1].Name != "idx_created" {
			t.Error("Second index should be idx_created")
		}
	})

	t.Run("filters nil indexes", func(t *testing.T) {
		idxs := []*builder.IndexDef{
			{Name: "idx_1", Columns: []string{"col1"}},
			nil,
			{Name: "idx_2", Columns: []string{"col2"}},
		}

		result := converter.IndexesToAST(idxs)
		if len(result) != 2 {
			t.Errorf("Expected 2 indexes (nil filtered), got %d", len(result))
		}
	})
}

// TestConvertValue tests value conversion including SQL expressions
func TestConvertValue(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("regular string value", func(t *testing.T) {
		result := converter.convertValue("hello")
		if result != "hello" {
			t.Errorf("Expected 'hello', got %v", result)
		}
	})

	t.Run("regular integer value", func(t *testing.T) {
		result := converter.convertValue(42)
		if result != 42 {
			t.Errorf("Expected 42, got %v", result)
		}
	})

	t.Run("SQL expression marker", func(t *testing.T) {
		sqlExpr := map[string]any{
			"_type": "sql_expr",
			"expr":  "NOW()",
		}

		result := converter.convertValue(sqlExpr)
		astExpr, ok := result.(*ast.SQLExpr)
		if !ok {
			t.Fatalf("Expected *ast.SQLExpr, got %T", result)
		}
		if astExpr.Expr != "NOW()" {
			t.Errorf("Expr = %q, want %q", astExpr.Expr, "NOW()")
		}
	})

	t.Run("regular map not SQL expression", func(t *testing.T) {
		regularMap := map[string]any{
			"foo": "bar",
			"baz": 123,
		}

		result := converter.convertValue(regularMap)
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Expected map[string]any, got %T", result)
		}
		if resultMap["foo"] != "bar" {
			t.Error("Expected map to be unchanged")
		}
	})

	t.Run("malformed SQL expression missing expr", func(t *testing.T) {
		sqlExpr := map[string]any{
			"_type": "sql_expr",
			// Missing "expr" field
		}

		result := converter.convertValue(sqlExpr)
		// Should return the map unchanged since expr field is missing
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Expected map[string]any, got %T", result)
		}
		if resultMap["_type"] != "sql_expr" {
			t.Error("Expected map to be unchanged")
		}
	})
}

// TestTableBuilderToAST tests table builder conversion
func TestTableBuilderToAST(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("basic table builder", func(t *testing.T) {
		tb := &builder.TableBuilder{
			Columns: []*builder.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "name", Type: "string"},
			},
			Indexes: []*builder.IndexDef{
				{Name: "idx_name", Columns: []string{"name"}},
			},
			Docs:       "User table",
			Deprecated: "",
		}

		result := converter.TableBuilderToAST(tb, "auth", "user")
		if result.Namespace != "auth" {
			t.Errorf("Namespace = %q, want %q", result.Namespace, "auth")
		}
		if result.Name != "user" {
			t.Errorf("Name = %q, want %q", result.Name, "user")
		}
		if len(result.Columns) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(result.Columns))
		}
		if len(result.Indexes) != 1 {
			t.Errorf("Expected 1 index, got %d", len(result.Indexes))
		}
		if result.Docs != "User table" {
			t.Errorf("Docs = %q, want %q", result.Docs, "User table")
		}
	})

	t.Run("empty table builder", func(t *testing.T) {
		tb := &builder.TableBuilder{
			Columns: []*builder.ColumnDef{},
			Indexes: []*builder.IndexDef{},
		}

		result := converter.TableBuilderToAST(tb, "app", "settings")
		if len(result.Columns) != 0 {
			t.Error("Expected empty columns")
		}
		if len(result.Indexes) != 0 {
			t.Error("Expected empty indexes")
		}
		if len(result.Checks) != 0 {
			t.Error("Expected empty checks")
		}
	})
}

// TestTableBuilderToAST_SchemaFields tests that TableBuilderToAST includes schema-only fields.
func TestTableBuilderToAST_SchemaFields(t *testing.T) {
	converter := NewColumnConverter()

	t.Run("table builder with all schema features", func(t *testing.T) {
		tb := &builder.TableBuilder{
			Columns: []*builder.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "title", Type: "string"},
			},
			Indexes: []*builder.IndexDef{
				{Name: "idx_title", Columns: []string{"title"}},
			},
			Docs:       "Blog post",
			Deprecated: "Use v2 API",
			Auditable:  true,
			SortBy:     []string{"created_at"},
			Searchable: []string{"title", "content"},
			Filterable: []string{"status"},
		}

		result := converter.TableBuilderToAST(tb, "blog", "post")
		if result.Namespace != "blog" {
			t.Errorf("Namespace = %q, want %q", result.Namespace, "blog")
		}
		if result.Name != "post" {
			t.Errorf("Name = %q, want %q", result.Name, "post")
		}
		if !result.Auditable {
			t.Error("Expected Auditable to be true")
		}
		if len(result.SortBy) != 1 || result.SortBy[0] != "created_at" {
			t.Errorf("SortBy = %v, want [created_at]", result.SortBy)
		}
		if len(result.Searchable) != 2 {
			t.Errorf("Expected 2 searchable fields, got %d", len(result.Searchable))
		}
		if len(result.Filterable) != 1 {
			t.Errorf("Expected 1 filterable field, got %d", len(result.Filterable))
		}
		if result.Docs != "Blog post" {
			t.Errorf("Docs = %q, want %q", result.Docs, "Blog post")
		}
	})

	t.Run("table builder without schema fields", func(t *testing.T) {
		tb := &builder.TableBuilder{
			Columns:    []*builder.ColumnDef{{Name: "id", Type: "uuid"}},
			Indexes:    []*builder.IndexDef{},
			Auditable:  false,
			SortBy:     nil,
			Searchable: nil,
			Filterable: nil,
		}

		result := converter.TableBuilderToAST(tb, "test", "table")
		if result.Auditable {
			t.Error("Expected Auditable to be false")
		}
		if result.SortBy != nil {
			t.Error("Expected SortBy to be nil")
		}
		if result.Searchable != nil {
			t.Error("Expected Searchable to be nil")
		}
		if result.Filterable != nil {
			t.Error("Expected Filterable to be nil")
		}
	})
}
