package engine

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hlop3z/astroladb/internal/ast"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestDetectDestructiveOperations_DropTableEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create empty table
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	ops := []ast.Operation{
		&ast.DropTable{
			TableOp: ast.TableOp{Name: "users"},
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	if len(destructive) != 1 {
		t.Fatalf("Expected 1 destructive operation, got %d", len(destructive))
	}

	d := destructive[0]
	if d.Type != "drop_table" {
		t.Errorf("Type = %q, want %q", d.Type, "drop_table")
	}
	if d.Target != "users" {
		t.Errorf("Target = %q, want %q", d.Target, "users")
	}
	if d.HasData {
		t.Error("HasData = true, want false (table is empty)")
	}
	if d.RowCount != 0 {
		t.Errorf("RowCount = %d, want 0", d.RowCount)
	}
	if d.IsDestructive() {
		t.Error("IsDestructive() = true, want false (table is empty)")
	}
}

func TestDetectDestructiveOperations_DropTableWithData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, name) VALUES ('1', 'Alice'), ('2', 'Bob')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.DropTable{
			TableOp: ast.TableOp{Name: "users"},
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	if len(destructive) != 1 {
		t.Fatalf("Expected 1 destructive operation, got %d", len(destructive))
	}

	d := destructive[0]
	if d.Type != "drop_table" {
		t.Errorf("Type = %q, want %q", d.Type, "drop_table")
	}
	if !d.HasData {
		t.Error("HasData = false, want true")
	}
	if d.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", d.RowCount)
	}
	if !d.IsDestructive() {
		t.Error("IsDestructive() = false, want true")
	}
}

func TestDetectDestructiveOperations_DropNonexistentTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	ops := []ast.Operation{
		&ast.DropTable{
			TableOp: ast.TableOp{Name: "nonexistent"},
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	// Non-existent table should not be flagged as destructive
	if len(destructive) != 0 {
		t.Errorf("Expected 0 destructive operations for non-existent table, got %d", len(destructive))
	}
}

func TestDetectDestructiveOperations_DropColumnEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with empty column
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, bio TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	// Insert row with NULL bio
	_, err = db.Exec("INSERT INTO users (id, bio) VALUES ('1', NULL)")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "bio",
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	if len(destructive) != 1 {
		t.Fatalf("Expected 1 destructive operation, got %d", len(destructive))
	}

	d := destructive[0]
	if d.Type != "drop_column" {
		t.Errorf("Type = %q, want %q", d.Type, "drop_column")
	}
	if d.HasData {
		t.Error("HasData = true, want false (column has only NULLs)")
	}
}

func TestDetectDestructiveOperations_DropColumnWithData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with column data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, bio TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, bio) VALUES ('1', 'Software engineer')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "bio",
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	if len(destructive) != 1 {
		t.Fatalf("Expected 1 destructive operation, got %d", len(destructive))
	}

	d := destructive[0]
	if d.Type != "drop_column" {
		t.Errorf("Type = %q, want %q", d.Type, "drop_column")
	}
	if !d.HasData {
		t.Error("HasData = false, want true")
	}
	if !d.IsDestructive() {
		t.Error("IsDestructive() = false, want true")
	}
}

func TestDetectDestructiveOperations_DropIndex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table and index
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("CREATE INDEX idx_email ON users(email)")
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	ops := []ast.Operation{
		&ast.DropIndex{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "idx_email",
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	if len(destructive) != 1 {
		t.Fatalf("Expected 1 destructive operation, got %d", len(destructive))
	}

	d := destructive[0]
	if d.Type != "drop_index" {
		t.Errorf("Type = %q, want %q", d.Type, "drop_index")
	}
	if d.HasData {
		t.Error("HasData = true, want false (indexes don't contain data)")
	}
	if d.IsDestructive() {
		t.Error("IsDestructive() = true, want false (dropping index is safe)")
	}
}

func TestDetectDestructiveOperations_MultipleOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create tables with varying data
	_, err := db.Exec(`
		CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT);
		CREATE TABLE posts (id TEXT PRIMARY KEY, title TEXT);
		CREATE TABLE comments (id TEXT PRIMARY KEY, body TEXT);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// users: has data
	_, err = db.Exec("INSERT INTO users (id, name) VALUES ('1', 'Alice')")
	if err != nil {
		t.Fatalf("Failed to insert into users: %v", err)
	}

	// posts: empty
	// comments: has data
	_, err = db.Exec("INSERT INTO comments (id, body) VALUES ('1', 'Great post!')")
	if err != nil {
		t.Fatalf("Failed to insert into comments: %v", err)
	}

	ops := []ast.Operation{
		&ast.DropTable{TableOp: ast.TableOp{Name: "users"}},    // Has data
		&ast.DropTable{TableOp: ast.TableOp{Name: "posts"}},    // Empty
		&ast.DropTable{TableOp: ast.TableOp{Name: "comments"}}, // Has data
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	if len(destructive) != 3 {
		t.Fatalf("Expected 3 destructive operations, got %d", len(destructive))
	}

	// Check which ones are truly destructive
	var truelyDestructive int
	for _, d := range destructive {
		if d.IsDestructive() {
			truelyDestructive++
		}
	}

	if truelyDestructive != 2 {
		t.Errorf("Expected 2 truly destructive operations (users, comments), got %d", truelyDestructive)
	}
}

func TestDetectDestructiveOperations_MixedOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, bio TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, name, bio) VALUES ('1', 'Alice', 'Engineer')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		// Non-destructive operations
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column:   &ast.ColumnDef{Name: "email", Type: "string"},
		},
		// Destructive operations
		&ast.DropColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "bio",
		},
		// More non-destructive
		&ast.RenameColumn{
			TableRef: ast.TableRef{Table_: "users"},
			OldName:  "name",
			NewName:  "full_name",
		},
	}

	destructive, err := DetectDestructiveOperations(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("DetectDestructiveOperations failed: %v", err)
	}

	// Should only detect the DropColumn operation
	if len(destructive) != 1 {
		t.Fatalf("Expected 1 destructive operation, got %d", len(destructive))
	}

	if destructive[0].Type != "drop_column" {
		t.Errorf("Expected drop_column, got %s", destructive[0].Type)
	}
}
