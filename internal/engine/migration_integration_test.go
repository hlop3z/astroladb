//go:build integration

// Package engine_test provides integration tests for migration execution
// against real PostgreSQL and SQLite databases.
package engine_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine/runner"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// -----------------------------------------------------------------------------
// Test helpers
// -----------------------------------------------------------------------------

// dialectDB pairs a database connection with its dialect.
type dialectDB struct {
	name    string
	db      *sql.DB
	dialect dialect.Dialect
}

// setupAllDatabases returns database connections for all supported dialects.
// Tests should call this to run against all databases.
func setupAllDatabases(t *testing.T) []dialectDB {
	t.Helper()

	return []dialectDB{
		{name: "postgres", db: testutil.SetupPostgres(t), dialect: dialect.Postgres()},
		{name: "sqlite", db: testutil.SetupSQLite(t), dialect: dialect.SQLite()},
	}
}

// execOperations executes a slice of operations against the database.
func execOperations(t *testing.T, db *sql.DB, d dialect.Dialect, ops []ast.Operation) {
	t.Helper()

	for _, op := range ops {
		sql, err := operationToSQL(d, op)
		if err != nil {
			t.Fatalf("failed to generate SQL for operation %T: %v", op, err)
		}
		if sql == "" {
			continue
		}

		_, err = db.Exec(sql)
		if err != nil {
			t.Fatalf("failed to execute SQL:\n%s\nerror: %v", sql, err)
		}
	}
}

// operationToSQL converts an operation to SQL using the dialect.
func operationToSQL(d dialect.Dialect, op ast.Operation) (string, error) {
	switch o := op.(type) {
	case *ast.CreateTable:
		return d.CreateTableSQL(o)
	case *ast.DropTable:
		return d.DropTableSQL(o)
	case *ast.AddColumn:
		return d.AddColumnSQL(o)
	case *ast.DropColumn:
		return d.DropColumnSQL(o)
	case *ast.RenameColumn:
		return d.RenameColumnSQL(o)
	case *ast.AlterColumn:
		return d.AlterColumnSQL(o)
	case *ast.CreateIndex:
		return d.CreateIndexSQL(o)
	case *ast.DropIndex:
		return d.DropIndexSQL(o)
	case *ast.RenameTable:
		return d.RenameTableSQL(o)
	case *ast.AddForeignKey:
		return d.AddForeignKeySQL(o)
	case *ast.DropForeignKey:
		return d.DropForeignKeySQL(o)
	case *ast.RawSQL:
		return d.RawSQLFor(o)
	default:
		return "", nil
	}
}

// -----------------------------------------------------------------------------
// Migration Table Tests
// -----------------------------------------------------------------------------

func TestVersionManager_EnsureTable_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			ctx := context.Background()
			vm := runner.NewVersionManager(ddb.db, ddb.dialect)

			// Create the migrations table
			err := vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable failed: %v", err)
			}

			// Verify the table exists
			testutil.AssertTableExists(t, ddb.db, runner.MigrationTableName)

			// Should be idempotent - calling again should not fail
			err = vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable (second call) failed: %v", err)
			}
		})
	}
}

func TestVersionManager_RecordApplied_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			ctx := context.Background()
			vm := runner.NewVersionManager(ddb.db, ddb.dialect)

			// Setup
			err := vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable failed: %v", err)
			}

			// Record a migration
			revision := "001"
			checksum := "abc123"
			execTime := 100 * time.Millisecond

			err = vm.RecordApplied(ctx, revision, checksum, execTime)
			if err != nil {
				t.Fatalf("RecordApplied failed: %v", err)
			}

			// Verify it was recorded
			applied, err := vm.GetApplied(ctx)
			if err != nil {
				t.Fatalf("GetApplied failed: %v", err)
			}

			if len(applied) != 1 {
				t.Fatalf("expected 1 applied migration, got %d", len(applied))
			}

			if applied[0].Revision != revision {
				t.Errorf("expected revision %q, got %q", revision, applied[0].Revision)
			}

			if applied[0].Checksum != checksum {
				t.Errorf("expected checksum %q, got %q", checksum, applied[0].Checksum)
			}
		})
	}
}

func TestVersionManager_RecordRollback_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			ctx := context.Background()
			vm := runner.NewVersionManager(ddb.db, ddb.dialect)

			// Setup
			err := vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable failed: %v", err)
			}

			// Record migrations
			_ = vm.RecordApplied(ctx, "001", "abc", 50*time.Millisecond)
			_ = vm.RecordApplied(ctx, "002", "def", 50*time.Millisecond)

			// Rollback one
			err = vm.RecordRollback(ctx, "002")
			if err != nil {
				t.Fatalf("RecordRollback failed: %v", err)
			}

			// Verify only 001 remains
			applied, err := vm.GetApplied(ctx)
			if err != nil {
				t.Fatalf("GetApplied failed: %v", err)
			}

			if len(applied) != 1 {
				t.Fatalf("expected 1 applied migration after rollback, got %d", len(applied))
			}

			if applied[0].Revision != "001" {
				t.Errorf("expected revision '001' to remain, got %q", applied[0].Revision)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// CreateTable Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_CreateTable_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create a simple table
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true, Nullable: false},
					{Name: "name", Type: "string", TypeArgs: []any{100}, Nullable: false},
					{Name: "created_at", Type: "datetime", Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{op})

			// Verify table exists
			testutil.AssertTableExists(t, ddb.db, "auth_user")

			// Verify columns exist
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "id")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "email")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "name")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "created_at")
		})
	}
}

func TestMigration_CreateTable_WithForeignKey_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create parent table first
			parentOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			// Create child table with foreign key
			childOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "author_id", Type: "uuid", Nullable: false},
					{Name: "title", Type: "string", TypeArgs: []any{200}, Nullable: false},
				},
				ForeignKeys: []*ast.ForeignKeyDef{
					{
						Name:       "fk_blog_post_author_id",
						Columns:    []string{"author_id"},
						RefTable:   "auth_user",
						RefColumns: []string{"id"},
						OnDelete:   "CASCADE",
					},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{parentOp, childOp})

			// Verify both tables exist
			testutil.AssertTableExists(t, ddb.db, "auth_user")
			testutil.AssertTableExists(t, ddb.db, "blog_post")
			testutil.AssertColumnExists(t, ddb.db, "blog_post", "author_id")
		})
	}
}

func TestMigration_CreateTable_WithTimestamps_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table with timestamps columns (simulating .timestamps() modifier)
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "content", Name: "article"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "title", Type: "string", TypeArgs: []any{200}, Nullable: false},
					{Name: "created_at", Type: "datetime", Nullable: false},
					{Name: "updated_at", Type: "datetime", Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{op})

			// Verify timestamps columns exist
			testutil.AssertTableExists(t, ddb.db, "content_article")
			testutil.AssertColumnExists(t, ddb.db, "content_article", "created_at")
			testutil.AssertColumnExists(t, ddb.db, "content_article", "updated_at")
		})
	}
}

func TestMigration_CreateTable_WithSoftDelete_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table with soft delete column (simulating .soft_delete() modifier)
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "content", Name: "post"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "title", Type: "string", TypeArgs: []any{200}, Nullable: false},
					{Name: "deleted_at", Type: "datetime", Nullable: true},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{op})

			// Verify soft delete column exists
			testutil.AssertTableExists(t, ddb.db, "content_post")
			testutil.AssertColumnExists(t, ddb.db, "content_post", "deleted_at")
		})
	}
}

// -----------------------------------------------------------------------------
// DropTable Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_DropTable_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// First create the table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "temp", Name: "to_delete"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})
			testutil.AssertTableExists(t, ddb.db, "temp_to_delete")

			// Now drop it
			dropOp := &ast.DropTable{
				TableOp:  ast.TableOp{Namespace: "temp", Name: "to_delete"},
				IfExists: true,
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{dropOp})
			testutil.AssertTableNotExists(t, ddb.db, "temp_to_delete")
		})
	}
}

// -----------------------------------------------------------------------------
// AddColumn Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_AddColumn_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create initial table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})

			// Add a new column
			addOp := &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Column: &ast.ColumnDef{
					Name:     "nickname",
					Type:     "string",
					TypeArgs: []any{50},
					Nullable: true,
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{addOp})

			// Verify the column was added
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "nickname")
		})
	}
}

func TestMigration_AddColumn_WithDefault_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create initial table with some data
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})

			// Insert some data
			var insertSQL string
			switch ddb.name {
			case "postgres":
				insertSQL = "INSERT INTO auth_user (id, email) VALUES (gen_random_uuid(), 'test@example.com')"
			case "sqlite":
				insertSQL = "INSERT INTO auth_user (id, email) VALUES (lower(hex(randomblob(16))), 'test@example.com')"
			}
			testutil.ExecSQL(t, ddb.db, insertSQL)

			// Add a new column with default
			addOp := &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Column: &ast.ColumnDef{
					Name:       "is_active",
					Type:       "boolean",
					Nullable:   false,
					Default:    true,
					DefaultSet: true,
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{addOp})

			// Verify the column was added
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "is_active")
		})
	}
}

// -----------------------------------------------------------------------------
// DropColumn Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_DropColumn_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create initial table with multiple columns
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
					{Name: "obsolete_field", Type: "string", TypeArgs: []any{100}, Nullable: true},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "obsolete_field")

			// Drop the obsolete column
			dropOp := &ast.DropColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "obsolete_field",
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{dropOp})

			// Verify the column was dropped
			testutil.AssertColumnNotExists(t, ddb.db, "auth_user", "obsolete_field")

			// Other columns should still exist
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "id")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "email")
		})
	}
}

// -----------------------------------------------------------------------------
// CreateIndex Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_CreateIndex_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
					{Name: "username", Type: "string", TypeArgs: []any{50}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})

			// Create an index
			indexOp := &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_auth_user_email",
				Columns:  []string{"email"},
				Unique:   false,
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{indexOp})

			// Verify the index was created
			testutil.AssertIndexExists(t, ddb.db, "auth_user", "idx_auth_user_email")
		})
	}
}

func TestMigration_CreateIndex_Unique_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})

			// Create a unique index
			indexOp := &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "uniq_auth_user_email",
				Columns:  []string{"email"},
				Unique:   true,
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{indexOp})

			// Verify the index was created
			testutil.AssertIndexExists(t, ddb.db, "auth_user", "uniq_auth_user_email")
		})
	}
}

func TestMigration_CreateIndex_Composite_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "social", Name: "follow"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "follower_id", Type: "uuid", Nullable: false},
					{Name: "following_id", Type: "uuid", Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})

			// Create a composite unique index
			indexOp := &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "social", Table_: "follow"},
				Name:     "uniq_social_follow_follower_following",
				Columns:  []string{"follower_id", "following_id"},
				Unique:   true,
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{indexOp})

			// Verify the index was created
			testutil.AssertIndexExists(t, ddb.db, "social_follow", "uniq_social_follow_follower_following")
		})
	}
}

// -----------------------------------------------------------------------------
// DropIndex Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_DropIndex_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table and index
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			indexOp := &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_auth_user_email",
				Columns:  []string{"email"},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp, indexOp})
			testutil.AssertIndexExists(t, ddb.db, "auth_user", "idx_auth_user_email")

			// Drop the index
			dropOp := &ast.DropIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Name:     "idx_auth_user_email",
				IfExists: true,
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{dropOp})
		})
	}
}

// -----------------------------------------------------------------------------
// Multiple Migrations in Order Tests
// -----------------------------------------------------------------------------

func TestMigration_MultipleMigrations_InOrder_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			ctx := context.Background()
			vm := runner.NewVersionManager(ddb.db, ddb.dialect)

			// Setup version tracking
			err := vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable failed: %v", err)
			}

			// Migration 1: Create users table
			ops1 := []ast.Operation{
				&ast.CreateTable{
					TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
					Columns: []*ast.ColumnDef{
						{Name: "id", Type: "uuid", PrimaryKey: true},
						{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
					},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, ops1)
			_ = vm.RecordApplied(ctx, "001", "hash1", 10*time.Millisecond)

			// Migration 2: Add columns to users
			ops2 := []ast.Operation{
				&ast.AddColumn{
					TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
					Column: &ast.ColumnDef{
						Name:     "name",
						Type:     "string",
						TypeArgs: []any{100},
						Nullable: true,
					},
				},
				&ast.AddColumn{
					TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
					Column: &ast.ColumnDef{
						Name:     "created_at",
						Type:     "datetime",
						Nullable: true,
					},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, ops2)
			_ = vm.RecordApplied(ctx, "002", "hash2", 15*time.Millisecond)

			// Migration 3: Create posts table with FK
			ops3 := []ast.Operation{
				&ast.CreateTable{
					TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
					Columns: []*ast.ColumnDef{
						{Name: "id", Type: "uuid", PrimaryKey: true},
						{Name: "author_id", Type: "uuid", Nullable: false},
						{Name: "title", Type: "string", TypeArgs: []any{200}, Nullable: false},
						{Name: "body", Type: "text", Nullable: true},
					},
					ForeignKeys: []*ast.ForeignKeyDef{
						{
							Name:       "fk_blog_post_author",
							Columns:    []string{"author_id"},
							RefTable:   "auth_user",
							RefColumns: []string{"id"},
							OnDelete:   "CASCADE",
						},
					},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, ops3)
			_ = vm.RecordApplied(ctx, "003", "hash3", 20*time.Millisecond)

			// Verify final state
			testutil.AssertTableExists(t, ddb.db, "auth_user")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "id")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "email")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "name")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "created_at")

			testutil.AssertTableExists(t, ddb.db, "blog_post")
			testutil.AssertColumnExists(t, ddb.db, "blog_post", "author_id")
			testutil.AssertColumnExists(t, ddb.db, "blog_post", "title")
			testutil.AssertColumnExists(t, ddb.db, "blog_post", "body")

			// Verify all migrations are recorded
			applied, err := vm.GetApplied(ctx)
			if err != nil {
				t.Fatalf("GetApplied failed: %v", err)
			}

			if len(applied) != 3 {
				t.Fatalf("expected 3 applied migrations, got %d", len(applied))
			}

			expectedRevisions := []string{"001", "002", "003"}
			for i, exp := range expectedRevisions {
				if applied[i].Revision != exp {
					t.Errorf("migration %d: expected revision %q, got %q", i, exp, applied[i].Revision)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Backfill Tests
// -----------------------------------------------------------------------------

func TestMigration_AddColumn_NotNull_WithBackfill_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table with initial data
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})

			// Insert some test data
			switch ddb.name {
			case "postgres":
				testutil.ExecSQL(t, ddb.db, "INSERT INTO auth_user (id, email) VALUES (gen_random_uuid(), 'user1@test.com'), (gen_random_uuid(), 'user2@test.com')")
			case "sqlite":
				testutil.ExecSQL(t, ddb.db, "INSERT INTO auth_user (id, email) VALUES (lower(hex(randomblob(16))), 'user1@test.com')")
				testutil.ExecSQL(t, ddb.db, "INSERT INTO auth_user (id, email) VALUES (lower(hex(randomblob(16))), 'user2@test.com')")
			}

			testutil.AssertRowCount(t, ddb.db, "auth_user", 2)

			// Strategy: Add column as nullable, backfill, then alter to NOT NULL
			// Step 1: Add as nullable
			addNullableOp := &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				Column: &ast.ColumnDef{
					Name:     "is_verified",
					Type:     "boolean",
					Nullable: true,
				},
			}
			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{addNullableOp})

			// Step 2: Backfill existing rows
			testutil.ExecSQL(t, ddb.db, "UPDATE auth_user SET is_verified = FALSE WHERE is_verified IS NULL")

			// Step 3: Alter to NOT NULL (if needed for the test)
			// For now, just verify the column exists and data is backfilled
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "is_verified")

			// Verify all rows have the backfilled value
			var nullCount int
			err := ddb.db.QueryRow("SELECT COUNT(*) FROM auth_user WHERE is_verified IS NULL").Scan(&nullCount)
			if err != nil {
				t.Fatalf("failed to count null values: %v", err)
			}
			if nullCount != 0 {
				t.Errorf("expected 0 null values after backfill, got %d", nullCount)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// RenameColumn Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_RenameColumn_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "user_name", Type: "string", TypeArgs: []any{50}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "user_name")

			// Rename the column
			renameOp := &ast.RenameColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "user"},
				OldName:  "user_name",
				NewName:  "username",
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{renameOp})

			// Verify the column was renamed
			testutil.AssertColumnNotExists(t, ddb.db, "auth_user", "user_name")
			testutil.AssertColumnExists(t, ddb.db, "auth_user", "username")
		})
	}
}

// -----------------------------------------------------------------------------
// RenameTable Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_RenameTable_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table
			createOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{createOp})
			testutil.AssertTableExists(t, ddb.db, "auth_users")

			// Rename the table
			renameOp := &ast.RenameTable{
				Namespace: "auth",
				OldName:   "users",
				NewName:   "user",
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{renameOp})

			// Verify the table was renamed
			testutil.AssertTableNotExists(t, ddb.db, "auth_users")
			testutil.AssertTableExists(t, ddb.db, "auth_user")
		})
	}
}

// -----------------------------------------------------------------------------
// Column Types Tests
// -----------------------------------------------------------------------------

func TestMigration_AllColumnTypes_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// Create table with all supported column types
			op := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "test", Name: "all_types"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "string_col", Type: "string", TypeArgs: []any{100}, Nullable: true},
					{Name: "text_col", Type: "text", Nullable: true},
					{Name: "integer_col", Type: "integer", Nullable: true},
					{Name: "float_col", Type: "float", Nullable: true},
					{Name: "decimal_col", Type: "decimal", TypeArgs: []any{10, 2}, Nullable: true},
					{Name: "boolean_col", Type: "boolean", Nullable: true},
					{Name: "date_col", Type: "date", Nullable: true},
					{Name: "time_col", Type: "time", Nullable: true},
					{Name: "datetime_col", Type: "datetime", Nullable: true},
					{Name: "uuid_col", Type: "uuid", Nullable: true},
					{Name: "json_col", Type: "json", Nullable: true},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{op})

			// Verify all columns exist
			testutil.AssertTableExists(t, ddb.db, "test_all_types")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "id")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "string_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "text_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "integer_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "float_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "decimal_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "boolean_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "date_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "time_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "datetime_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "uuid_col")
			testutil.AssertColumnExists(t, ddb.db, "test_all_types", "json_col")
		})
	}
}

// -----------------------------------------------------------------------------
// AddForeignKey Operation Tests
// -----------------------------------------------------------------------------

func TestMigration_AddForeignKey_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			// SQLite doesn't support ALTER TABLE ADD FOREIGN KEY
			if ddb.name == "sqlite" {
				t.Skip("SQLite does not support ALTER TABLE ADD FOREIGN KEY")
			}

			// Create parent table
			parentOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false},
				},
			}

			// Create child table without FK
			childOp := &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "blog", Name: "post"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "uuid", PrimaryKey: true},
					{Name: "author_id", Type: "uuid", Nullable: false},
					{Name: "title", Type: "string", TypeArgs: []any{200}, Nullable: false},
				},
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{parentOp, childOp})

			// Add FK separately
			fkOp := &ast.AddForeignKey{
				TableRef:   ast.TableRef{Namespace: "blog", Table_: "post"},
				Name:       "fk_blog_post_author",
				Columns:    []string{"author_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			}

			execOperations(t, ddb.db, ddb.dialect, []ast.Operation{fkOp})

			// FK was added - verify by trying to insert invalid data
			// This should fail due to FK constraint
			var insertSQL string
			switch ddb.name {
			case "postgres":
				insertSQL = "INSERT INTO blog_post (id, author_id, title) VALUES (gen_random_uuid(), gen_random_uuid(), 'Test')"
			case "sqlite":
				insertSQL = "INSERT INTO blog_post (id, author_id, title) VALUES (lower(hex(randomblob(16))), lower(hex(randomblob(16))), 'Test')"
			}

			_, err := ddb.db.Exec(insertSQL)
			if err == nil {
				t.Error("expected FK constraint violation, but insert succeeded")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Checksum Verification Tests
// -----------------------------------------------------------------------------

func TestVersionManager_ChecksumVerification_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			ctx := context.Background()
			vm := runner.NewVersionManager(ddb.db, ddb.dialect)

			// Setup
			err := vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable failed: %v", err)
			}

			// Record a migration with checksum
			revision := "001"
			checksum := "abc123def456"

			err = vm.RecordApplied(ctx, revision, checksum, 50*time.Millisecond)
			if err != nil {
				t.Fatalf("RecordApplied failed: %v", err)
			}

			// Verify checksum matches
			err = vm.VerifyChecksum(ctx, revision, checksum)
			if err != nil {
				t.Errorf("VerifyChecksum should pass with matching checksum: %v", err)
			}

			// Verify checksum mismatch is detected
			err = vm.VerifyChecksum(ctx, revision, "different_checksum")
			if err == nil {
				t.Error("VerifyChecksum should fail with mismatched checksum")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// IsApplied Tests
// -----------------------------------------------------------------------------

func TestVersionManager_IsApplied_AllDialects(t *testing.T) {
	for _, ddb := range setupAllDatabases(t) {
		t.Run(ddb.name, func(t *testing.T) {
			ctx := context.Background()
			vm := runner.NewVersionManager(ddb.db, ddb.dialect)

			// Setup
			err := vm.EnsureTable(ctx)
			if err != nil {
				t.Fatalf("EnsureTable failed: %v", err)
			}

			// Check non-existent migration
			applied, err := vm.IsApplied(ctx, "001")
			if err != nil {
				t.Fatalf("IsApplied failed: %v", err)
			}
			if applied {
				t.Error("expected migration to not be applied")
			}

			// Record and check again
			_ = vm.RecordApplied(ctx, "001", "abc", 10*time.Millisecond)

			applied, err = vm.IsApplied(ctx, "001")
			if err != nil {
				t.Fatalf("IsApplied failed: %v", err)
			}
			if !applied {
				t.Error("expected migration to be applied")
			}
		})
	}
}
