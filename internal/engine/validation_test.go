package engine

import (
	"context"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestValidateConstraints_AddNotNullColumnToEmptyTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create empty table
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:     "email",
				Type:     "string",
				Nullable: false, // NOT NULL
			},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	// Empty table - no violations
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for empty table, got %d", len(violations))
	}
}

func TestValidateConstraints_AddNotNullColumnWithData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id) VALUES ('1'), ('2')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:     "email",
				Type:     "string",
				Nullable: false, // NOT NULL without default
			},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "not_null" {
		t.Errorf("Type = %q, want %q", v.Type, "not_null")
	}
	if v.ViolationCount != 2 {
		t.Errorf("ViolationCount = %d, want 2", v.ViolationCount)
	}
}

func TestValidateConstraints_AddNotNullColumnWithDefault(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id) VALUES ('1'), ('2')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:       "email",
				Type:       "string",
				Nullable:   false,
				Default:    "user@example.com", // Has default
				DefaultSet: true,
			},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	// Has default - no violation
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations (has default), got %d", len(violations))
	}
}

func TestValidateConstraints_AddNotNullColumnWithBackfill(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id) VALUES ('1'), ('2')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:        "email",
				Type:        "string",
				Nullable:    false,
				Backfill:    "user@example.com", // Has backfill
				BackfillSet: true,
			},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	// Has backfill - no violation
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations (has backfill), got %d", len(violations))
	}
}

func TestValidateConstraints_AddUniqueColumnWithDefault(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id) VALUES ('1'), ('2')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:       "email",
				Type:       "string",
				Unique:     true,         // UNIQUE
				Default:    "same@email", // Same default for all rows!
				DefaultSet: true,
			},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	// UNIQUE column with default on existing rows = violation
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "unique" {
		t.Errorf("Type = %q, want %q", v.Type, "unique")
	}
	if v.ViolationCount != 2 {
		t.Errorf("ViolationCount = %d, want 2", v.ViolationCount)
	}
}

func TestValidateConstraints_AlterColumnToNotNull(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with nullable column containing NULLs
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, email) VALUES ('1', NULL), ('2', 'user@example.com')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	notNullable := false
	ops := []ast.Operation{
		&ast.AlterColumn{
			TableRef:    ast.TableRef{Table_: "users"},
			Name:        "email",
			SetNullable: &notNullable, // Making NOT NULL
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "not_null" {
		t.Errorf("Type = %q, want %q", v.Type, "not_null")
	}
	if v.ViolationCount != 1 {
		t.Errorf("ViolationCount = %d, want 1 (one NULL row)", v.ViolationCount)
	}
	if len(v.SampleValues) == 0 {
		t.Error("Expected sample values to be populated")
	}
}

func TestValidateConstraints_AlterColumnToNotNullNoViolations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with no NULLs
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, email) VALUES ('1', 'alice@example.com'), ('2', 'bob@example.com')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	notNullable := false
	ops := []ast.Operation{
		&ast.AlterColumn{
			TableRef:    ast.TableRef{Table_: "users"},
			Name:        "email",
			SetNullable: &notNullable,
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	// No NULLs - no violation
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations (no NULLs), got %d", len(violations))
	}
}

func TestValidateConstraints_CreateUniqueIndexWithDuplicates(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with duplicate emails
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, email) VALUES ('1', 'alice@example.com'), ('2', 'alice@example.com')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.CreateIndex{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "idx_email",
			Columns:  []string{"email"},
			Unique:   true,
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "unique" {
		t.Errorf("Type = %q, want %q", v.Type, "unique")
	}
	if v.ViolationCount != 1 {
		t.Errorf("ViolationCount = %d, want 1 (one duplicate group)", v.ViolationCount)
	}
}

func TestValidateConstraints_CreateUniqueIndexNoDuplicates(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with unique emails
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, email) VALUES ('1', 'alice@example.com'), ('2', 'bob@example.com')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	ops := []ast.Operation{
		&ast.CreateIndex{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "idx_email",
			Columns:  []string{"email"},
			Unique:   true,
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	// All unique - no violation
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations (all unique), got %d", len(violations))
	}
}

func TestValidateConstraints_AddForeignKeyMissingRefTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create source table but not referenced table
	_, err := db.Exec("CREATE TABLE posts (id TEXT PRIMARY KEY, user_id TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Table_: "posts"},
			Columns:    []string{"user_id"},
			RefTable:   "users", // Doesn't exist!
			RefColumns: []string{"id"},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "foreign_key" {
		t.Errorf("Type = %q, want %q", v.Type, "foreign_key")
	}
}

func TestValidateConstraints_AddForeignKeyOrphanedRows(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create both tables
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id) VALUES ('user1')")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	_, err = db.Exec("CREATE TABLE posts (id TEXT PRIMARY KEY, user_id TEXT)")
	if err != nil {
		t.Fatalf("Failed to create posts table: %v", err)
	}
	// Insert post with invalid user_id
	_, err = db.Exec("INSERT INTO posts (id, user_id) VALUES ('post1', 'invalid_user')")
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}

	ops := []ast.Operation{
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Table_: "posts"},
			Columns:    []string{"user_id"},
			RefTable:   "users",
			RefColumns: []string{"id"},
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "foreign_key" {
		t.Errorf("Type = %q, want %q", v.Type, "foreign_key")
	}
	if v.ViolationCount != 1 {
		t.Errorf("ViolationCount = %d, want 1 (one orphaned row)", v.ViolationCount)
	}
}

func TestValidateConstraints_MultipleViolations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create table with data
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, email) VALUES ('1', NULL), ('2', 'user@example.com')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	notNullable := false
	ops := []ast.Operation{
		// Violation 1: NOT NULL column without default
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:     "phone",
				Type:     "string",
				Nullable: false,
			},
		},
		// Violation 2: Make email NOT NULL (has NULL values)
		&ast.AlterColumn{
			TableRef:    ast.TableRef{Table_: "users"},
			Name:        "email",
			SetNullable: &notNullable,
		},
	}

	violations, err := ValidateConstraints(ctx, db, ops, "sqlite")
	if err != nil {
		t.Fatalf("ValidateConstraints failed: %v", err)
	}

	if len(violations) != 2 {
		t.Fatalf("Expected 2 violations, got %d", len(violations))
	}

	// Both should be not_null violations
	for _, v := range violations {
		if v.Type != "not_null" {
			t.Errorf("Expected all violations to be not_null, got %q", v.Type)
		}
	}
}
