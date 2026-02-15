package dsl

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestNewColumnBuilder(t *testing.T) {
	tests := []struct {
		name     string
		colName  string
		typeName string
		args     []any
		want     *ast.ColumnDef
	}{
		{
			name:     "simple string column",
			colName:  "name",
			typeName: "string",
			args:     []any{255},
			want: &ast.ColumnDef{
				Name:        "name",
				Type:        "string",
				TypeArgs:    []any{255},
				Nullable:    false,
				NullableSet: false,
			},
		},
		{
			name:     "text column no args",
			colName:  "description",
			typeName: "text",
			args:     nil,
			want: &ast.ColumnDef{
				Name:        "description",
				Type:        "text",
				TypeArgs:    nil,
				Nullable:    false,
				NullableSet: false,
			},
		},
		{
			name:     "decimal with precision and scale",
			colName:  "price",
			typeName: "decimal",
			args:     []any{10, 2},
			want: &ast.ColumnDef{
				Name:        "price",
				Type:        "decimal",
				TypeArgs:    []any{10, 2},
				Nullable:    false,
				NullableSet: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewColumnBuilder(tt.colName, tt.typeName, tt.args...)
			got := cb.Build()

			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Nullable != tt.want.Nullable {
				t.Errorf("Nullable = %v, want %v", got.Nullable, tt.want.Nullable)
			}
			if got.NullableSet != tt.want.NullableSet {
				t.Errorf("NullableSet = %v, want %v", got.NullableSet, tt.want.NullableSet)
			}
		})
	}
}

func TestColumnBuilder_Nullable(t *testing.T) {
	cb := NewColumnBuilder("email", "string", 255)
	cb.Nullable()
	got := cb.Build()

	if !got.Nullable {
		t.Error("Nullable() should set Nullable to true")
	}
	if !got.NullableSet {
		t.Error("Nullable() should set NullableSet to true")
	}
}

func TestColumnBuilder_NotNull(t *testing.T) {
	cb := NewColumnBuilder("email", "string", 255)
	cb.NotNull()
	got := cb.Build()

	if got.Nullable {
		t.Error("NotNull() should set Nullable to false")
	}
	if !got.NullableSet {
		t.Error("NotNull() should set NullableSet to true")
	}
}

func TestColumnBuilder_Unique(t *testing.T) {
	cb := NewColumnBuilder("email", "string", 255)
	cb.Unique()
	got := cb.Build()

	if !got.Unique {
		t.Error("Unique() should set Unique to true")
	}
}

func TestColumnBuilder_PrimaryKey(t *testing.T) {
	cb := NewColumnBuilder("id", "uuid")
	cb.PrimaryKey()
	got := cb.Build()

	if !got.PrimaryKey {
		t.Error("PrimaryKey() should set PrimaryKey to true")
	}
}

func TestColumnBuilder_Default(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"string default", "active"},
		{"integer default", 0},
		{"boolean default", false},
		{"sql expression", &ast.SQLExpr{Expr: "NOW()"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewColumnBuilder("status", "string", 50)
			cb.Default(tt.value)
			got := cb.Build()

			if got.Default != tt.value {
				t.Errorf("Default = %v, want %v", got.Default, tt.value)
			}
			if !got.DefaultSet {
				t.Error("Default() should set DefaultSet to true")
			}
		})
	}
}

func TestColumnBuilder_Backfill(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"string backfill", "unknown"},
		{"integer backfill", 0},
		{"sql expression", &ast.SQLExpr{Expr: "COALESCE(old_column, 'default')"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewColumnBuilder("category", "string", 100)
			cb.Backfill(tt.value)
			got := cb.Build()

			if got.Backfill != tt.value {
				t.Errorf("Backfill = %v, want %v", got.Backfill, tt.value)
			}
			if !got.BackfillSet {
				t.Error("Backfill() should set BackfillSet to true")
			}
		})
	}
}

func TestColumnBuilder_Min(t *testing.T) {
	cb := NewColumnBuilder("age", "integer")
	cb.Min(0)
	got := cb.Build()

	if got.Min == nil {
		t.Fatal("Min() should set Min to non-nil")
	}
	if *got.Min != 0.0 {
		t.Errorf("Min = %v, want 0.0", *got.Min)
	}
}

func TestColumnBuilder_Max(t *testing.T) {
	cb := NewColumnBuilder("age", "integer")
	cb.Max(150)
	got := cb.Build()

	if got.Max == nil {
		t.Fatal("Max() should set Max to non-nil")
	}
	if *got.Max != 150.0 {
		t.Errorf("Max = %v, want 150.0", *got.Max)
	}
}

func TestColumnBuilder_MinMax_Combined(t *testing.T) {
	cb := NewColumnBuilder("quantity", "integer")
	cb.Min(1).Max(100)
	got := cb.Build()

	if got.Min == nil || *got.Min != 1.0 {
		t.Errorf("Min = %v, want 1.0", got.Min)
	}
	if got.Max == nil || *got.Max != 100.0 {
		t.Errorf("Max = %v, want 100.0", got.Max)
	}
}

func TestColumnBuilder_Pattern(t *testing.T) {
	pattern := `^[A-Z]{2}-\d{4}$`
	cb := NewColumnBuilder("code", "string", 10)
	cb.Pattern(pattern)
	got := cb.Build()

	if got.Pattern != pattern {
		t.Errorf("Pattern = %q, want %q", got.Pattern, pattern)
	}
}

func TestColumnBuilder_Format(t *testing.T) {
	cb := NewColumnBuilder("email", "string", 255)
	cb.Format("email")
	got := cb.Build()

	if got.Format != "email" {
		t.Errorf("Format = %q, want %q", got.Format, "email")
	}
}

func TestColumnBuilder_FormatDef(t *testing.T) {
	cb := NewColumnBuilder("email", "string", 255)
	cb.FormatDef(FormatEmail)
	got := cb.Build()

	if got.Format != "email" {
		t.Errorf("Format = %q, want %q", got.Format, "email")
	}
	if got.Pattern != FormatEmail.Pattern {
		t.Errorf("Pattern = %q, want %q", got.Pattern, FormatEmail.Pattern)
	}
}

func TestColumnBuilder_FormatDef_DoesNotOverwritePattern(t *testing.T) {
	customPattern := `^[a-z]+@example\.com$`
	cb := NewColumnBuilder("email", "string", 255)
	cb.Pattern(customPattern).FormatDef(FormatEmail)
	got := cb.Build()

	if got.Format != "email" {
		t.Errorf("Format = %q, want %q", got.Format, "email")
	}
	// Pattern should remain as custom pattern, not overwritten
	if got.Pattern != customPattern {
		t.Errorf("Pattern = %q, want %q (custom pattern should not be overwritten)", got.Pattern, customPattern)
	}
}

func TestColumnBuilder_Docs(t *testing.T) {
	description := "User's email address for login"
	cb := NewColumnBuilder("email", "string", 255)
	cb.Docs(description)
	got := cb.Build()

	if got.Docs != description {
		t.Errorf("Docs = %q, want %q", got.Docs, description)
	}
}

func TestColumnBuilder_Deprecated(t *testing.T) {
	reason := "Use email_address instead"
	cb := NewColumnBuilder("email", "string", 255)
	cb.Deprecated(reason)
	got := cb.Build()

	if got.Deprecated != reason {
		t.Errorf("Deprecated = %q, want %q", got.Deprecated, reason)
	}
}

func TestColumnBuilder_References(t *testing.T) {
	cb := NewColumnBuilder("user_id", "uuid")
	cb.References("auth.users")
	got := cb.Build()

	if got.Reference == nil {
		t.Fatal("References() should set Reference to non-nil")
	}
	if got.Reference.Table != "auth.users" {
		t.Errorf("Reference.Table = %q, want %q", got.Reference.Table, "auth.users")
	}
	if got.Reference.Column != "id" {
		t.Errorf("Reference.Column = %q, want %q", got.Reference.Column, "id")
	}
}

func TestColumnBuilder_References_WithOptions(t *testing.T) {
	cb := NewColumnBuilder("author_id", "uuid")
	cb.References("blog.users", RefColumn("user_id"), OnDelete(Cascade), OnUpdate(SetNull))
	got := cb.Build()

	if got.Reference == nil {
		t.Fatal("References() should set Reference to non-nil")
	}
	if got.Reference.Column != "user_id" {
		t.Errorf("Reference.Column = %q, want %q", got.Reference.Column, "user_id")
	}
	if got.Reference.OnDelete != Cascade {
		t.Errorf("Reference.OnDelete = %q, want %q", got.Reference.OnDelete, Cascade)
	}
	if got.Reference.OnUpdate != SetNull {
		t.Errorf("Reference.OnUpdate = %q, want %q", got.Reference.OnUpdate, SetNull)
	}
}

func TestColumnBuilder_FluentChaining(t *testing.T) {
	cb := NewColumnBuilder("email", "string", 255)
	got := cb.
		Unique().
		FormatDef(FormatEmail).
		Min(5).
		Max(255).
		Docs("User email address").
		Build()

	if !got.Unique {
		t.Error("Expected Unique to be true")
	}
	if got.Format != "email" {
		t.Error("Expected Format to be 'email'")
	}
	if got.Min == nil || *got.Min != 5 {
		t.Error("Expected Min to be 5")
	}
	if got.Max == nil || *got.Max != 255 {
		t.Error("Expected Max to be 255")
	}
	if got.Docs != "User email address" {
		t.Error("Expected Docs to be set")
	}
}

func TestColumnBuilder_IsStringType(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"string", true},
		{"text", true},
		{"integer", false},
		{"boolean", false},
		{"uuid", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			cb := NewColumnBuilder("test", tt.typeName)
			if got := cb.IsStringType(); got != tt.want {
				t.Errorf("IsStringType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumnBuilder_IsNumericType(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"integer", true},
		{"float", true},
		{"decimal", true},
		{"string", false},
		{"boolean", false},
		{"uuid", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			cb := NewColumnBuilder("test", tt.typeName)
			if got := cb.IsNumericType(); got != tt.want {
				t.Errorf("IsNumericType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumnBuilder_Def(t *testing.T) {
	cb := NewColumnBuilder("test", "string", 100)
	def := cb.Def()

	if def == nil {
		t.Fatal("Def() should return non-nil")
	}
	if def != cb.Build() {
		t.Error("Def() should return the same underlying ColumnDef")
	}
}

func TestRefOptionConstants(t *testing.T) {
	// Test that constants have expected values
	if Cascade != "CASCADE" {
		t.Errorf("Cascade = %q, want %q", Cascade, "CASCADE")
	}
	if SetNull != "SET NULL" {
		t.Errorf("SetNull = %q, want %q", SetNull, "SET NULL")
	}
	if Restrict != "RESTRICT" {
		t.Errorf("Restrict = %q, want %q", Restrict, "RESTRICT")
	}
	if NoAction != "NO ACTION" {
		t.Errorf("NoAction = %q, want %q", NoAction, "NO ACTION")
	}
	if SetDefault != "SET DEFAULT" {
		t.Errorf("SetDefault = %q, want %q", SetDefault, "SET DEFAULT")
	}
}
