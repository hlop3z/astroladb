package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestDetectRenames_ColumnRename(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "name",
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name: "full_name",
				Type: "string",
			},
		},
	}

	candidates := DetectRenames(ops)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]
	if c.Type != "column" {
		t.Errorf("expected type 'column', got %q", c.Type)
	}
	if c.Table != "user" {
		t.Errorf("expected table 'user', got %q", c.Table)
	}
	if c.Namespace != "auth" {
		t.Errorf("expected namespace 'auth', got %q", c.Namespace)
	}
	if c.OldName != "name" {
		t.Errorf("expected old name 'name', got %q", c.OldName)
	}
	if c.NewName != "full_name" {
		t.Errorf("expected new name 'full_name', got %q", c.NewName)
	}
}

func TestDetectRenames_TableRename(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "article"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
	}

	candidates := DetectRenames(ops)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	c := candidates[0]
	if c.Type != "table" {
		t.Errorf("expected type 'table', got %q", c.Type)
	}
	if c.Namespace != "blog" {
		t.Errorf("expected namespace 'blog', got %q", c.Namespace)
	}
	if c.OldName != "post" {
		t.Errorf("expected old name 'post', got %q", c.OldName)
	}
	if c.NewName != "article" {
		t.Errorf("expected new name 'article', got %q", c.NewName)
	}
}

func TestDetectRenames_NoRename(t *testing.T) {
	// Different tables - not a rename
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "name",
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "profile"}, // Different table
			Column: &ast.ColumnDef{
				Name: "full_name",
				Type: "string",
			},
		},
	}

	candidates := DetectRenames(ops)

	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates, got %d", len(candidates))
	}
}

func TestApplyRenames_ColumnRename(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "name",
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name: "full_name",
				Type: "string",
			},
		},
	}

	confirmed := []RenameCandidate{
		{
			Type:      "column",
			Namespace: "auth",
			Table:     "user",
			OldName:   "name",
			NewName:   "full_name",
		},
	}

	result := ApplyRenames(ops, confirmed)

	if len(result) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(result))
	}

	rename, ok := result[0].(*ast.RenameColumn)
	if !ok {
		t.Fatalf("expected RenameColumn, got %T", result[0])
	}

	if rename.Namespace != "auth" {
		t.Errorf("expected namespace 'auth', got %q", rename.Namespace)
	}
	if rename.Table_ != "user" {
		t.Errorf("expected table 'user', got %q", rename.Table_)
	}
	if rename.OldName != "name" {
		t.Errorf("expected old name 'name', got %q", rename.OldName)
	}
	if rename.NewName != "full_name" {
		t.Errorf("expected new name 'full_name', got %q", rename.NewName)
	}
}

func TestApplyRenames_TableRename(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "article"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
			},
		},
	}

	confirmed := []RenameCandidate{
		{
			Type:      "table",
			Namespace: "blog",
			OldName:   "post",
			NewName:   "article",
		},
	}

	result := ApplyRenames(ops, confirmed)

	if len(result) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(result))
	}

	rename, ok := result[0].(*ast.RenameTable)
	if !ok {
		t.Fatalf("expected RenameTable, got %T", result[0])
	}

	if rename.Namespace != "blog" {
		t.Errorf("expected namespace 'blog', got %q", rename.Namespace)
	}
	if rename.OldName != "post" {
		t.Errorf("expected old name 'post', got %q", rename.OldName)
	}
	if rename.NewName != "article" {
		t.Errorf("expected new name 'article', got %q", rename.NewName)
	}
}

func TestApplyRenames_NoConfirmation(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Name:     "name",
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
			Column: &ast.ColumnDef{
				Name: "full_name",
				Type: "string",
			},
		},
	}

	// No confirmations - operations should remain unchanged
	result := ApplyRenames(ops, nil)

	if len(result) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(result))
	}

	if _, ok := result[0].(*ast.DropColumn); !ok {
		t.Errorf("expected DropColumn, got %T", result[0])
	}
	if _, ok := result[1].(*ast.AddColumn); !ok {
		t.Errorf("expected AddColumn, got %T", result[1])
	}
}

func TestJaroWinkler_Identical(t *testing.T) {
	if s := JaroWinkler("hello", "hello"); s != 1.0 {
		t.Errorf("expected 1.0, got %f", s)
	}
}

func TestJaroWinkler_Empty(t *testing.T) {
	if s := JaroWinkler("", "hello"); s != 0.0 {
		t.Errorf("expected 0.0, got %f", s)
	}
	if s := JaroWinkler("hello", ""); s != 0.0 {
		t.Errorf("expected 0.0, got %f", s)
	}
	if s := JaroWinkler("", ""); s != 1.0 {
		t.Errorf("expected 1.0, got %f", s)
	}
}

func TestJaroWinkler_Similar(t *testing.T) {
	// Common prefix should boost score
	s := JaroWinkler("username", "user_name")
	if s < 0.7 {
		t.Errorf("expected high similarity for username/user_name, got %f", s)
	}
}

func TestJaroWinkler_Dissimilar(t *testing.T) {
	s := JaroWinkler("abc", "xyz")
	if s > 0.5 {
		t.Errorf("expected low similarity for abc/xyz, got %f", s)
	}
}

func TestJaroWinkler_CommonPrefix(t *testing.T) {
	// Winkler boost for common prefix
	withPrefix := JaroWinkler("email", "email_address")
	noPrefix := JaroWinkler("email", "address_email")
	if withPrefix <= noPrefix {
		t.Errorf("common prefix should boost score: %f <= %f", withPrefix, noPrefix)
	}
}

func TestScoreColumnRename_SameType(t *testing.T) {
	drop := colInfo{name: "old_col", colType: "string", index: 0}
	add := colInfo{name: "new_col", colType: "string", index: 1}
	score := scoreColumnRename(drop, add, 2)
	// Type match (0.4) + name sim (~0.3*x) + constraints (0.2) + position (0.1*y)
	if score < 0.6 {
		t.Errorf("same type columns should score >= 0.6, got %f", score)
	}
}

func TestScoreColumnRename_DifferentType(t *testing.T) {
	drop := colInfo{name: "old_col", colType: "string", index: 0}
	add := colInfo{name: "new_col", colType: "integer", index: 1}
	score := scoreColumnRename(drop, add, 2)
	// No type match bonus
	if score >= 0.6 {
		t.Errorf("different type columns should score < 0.6, got %f", score)
	}
}

func TestDetectRenames_HighSimilarityColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "core", Table_: "account"},
			Name:     "username",
		},
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "core", Table_: "account"},
			Column:   &ast.ColumnDef{Name: "user_name", Type: "string"},
		},
	}
	candidates := DetectRenames(ops)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Score == 0 {
		t.Error("expected non-zero score")
	}
}

func TestDetectRenames_DifferentNamespace(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropTable{TableOp: ast.TableOp{Namespace: "auth", Name: "user"}},
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "blog", Name: "user"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}},
	}
	candidates := DetectRenames(ops)
	if len(candidates) != 0 {
		t.Errorf("different namespaces should not match, got %d candidates", len(candidates))
	}
}

func TestDetectRenames_MultipleColumns(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "core", Table_: "t"}, Name: "a"},
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "core", Table_: "t"}, Name: "b"},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "core", Table_: "t"}, Column: &ast.ColumnDef{Name: "x", Type: "string"}},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "core", Table_: "t"}, Column: &ast.ColumnDef{Name: "y", Type: "string"}},
	}
	candidates := DetectRenames(ops)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	// Each drop should map to exactly one add
	names := map[string]bool{}
	for _, c := range candidates {
		names[c.OldName+"->"+c.NewName] = true
	}
	if len(names) != 2 {
		t.Errorf("expected 2 unique rename pairs, got %d", len(names))
	}
}
