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
