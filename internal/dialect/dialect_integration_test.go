//go:build integration

// Package dialect_test provides integration tests for dialect SQL execution.
// These tests verify that generated SQL actually executes on real databases.
//
// Run with: go test ./internal/dialect -tags=integration
// Requires: docker-compose -f docker-compose.test.yml up -d
package dialect_test

import (
	"database/sql"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// =============================================================================
// PostgreSQL Tests
// =============================================================================

func TestCreateTable_SimpleTypes_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "simple_types"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{255}},
			{Name: "description", Type: "text"},
			{Name: "count", Type: "integer"},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "active", Type: "boolean", Default: false, DefaultSet: true},
			{Name: "created_at", Type: "date_time"},
			{Name: "external_id", Type: "uuid"},
			{Name: "metadata", Type: "json", Nullable: true, NullableSet: true},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sql)
	testutil.AssertTableExists(t, db, "test_simple_types")
	testutil.AssertColumnExists(t, db, "test_simple_types", "id")
	testutil.AssertColumnExists(t, db, "test_simple_types", "name")
	testutil.AssertColumnExists(t, db, "test_simple_types", "metadata")
}

func TestCreateTable_WithForeignKey_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// First create the referenced table
	parentOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true},
		},
	}
	parentSQL, _ := d.CreateTableSQL(parentOp)
	testutil.ExecSQL(t, db, parentSQL)

	// Now create table with FK
	childOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "content", Name: "post"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "author_id", Type: "uuid"},
			{Name: "title", Type: "string", TypeArgs: []any{200}},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_post_author",
				Columns:    []string{"author_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
		},
	}

	childSQL, err := d.CreateTableSQL(childOp)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, childSQL)
	testutil.AssertTableExists(t, db, "content_post")
	testutil.AssertColumnExists(t, db, "content_post", "author_id")
}

func TestCreateTable_WithIndex_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create table first
	tableOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "indexed"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "status", Type: "string", TypeArgs: []any{50}},
		},
	}
	tableSQL, _ := d.CreateTableSQL(tableOp)
	testutil.ExecSQL(t, db, tableSQL)

	// Create index
	indexOp := &ast.CreateIndex{
		TableRef:    ast.TableRef{Namespace: "test", Table_: "indexed"},
		Name:        "idx_indexed_email_status",
		Columns:     []string{"email", "status"},
		IfNotExists: true,
	}
	indexSQL, err := d.CreateIndexSQL(indexOp)
	if err != nil {
		t.Fatalf("failed to generate index SQL: %v", err)
	}

	testutil.ExecSQL(t, db, indexSQL)
	testutil.AssertIndexExists(t, db, "test_indexed", "idx_indexed_email_status")
}

func TestCreateTable_WithUniqueConstraint_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "unique_cols"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "code", Type: "string", TypeArgs: []any{50}, Unique: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sql)
	testutil.AssertTableExists(t, db, "test_unique_cols")

	// Verify unique constraint works by trying to insert duplicate
	testutil.ExecSQL(t, db, `INSERT INTO test_unique_cols (id, code, name) VALUES ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'ABC', 'First')`)
	_, err = db.Exec(`INSERT INTO test_unique_cols (id, code, name) VALUES ('b2c3d4e5-f6a7-8901-bcde-f12345678901', 'ABC', 'Second')`)
	if err == nil {
		t.Error("expected unique constraint violation, but insert succeeded")
	}
}

func TestAlterTable_AddColumn_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create initial table
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "alter_add"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Add column
	addOp := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "test", Table_: "alter_add"},
		Column:   &ast.ColumnDef{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: true, NullableSet: true},
	}
	addSQL, err := d.AddColumnSQL(addOp)
	if err != nil {
		t.Fatalf("failed to generate add column SQL: %v", err)
	}

	testutil.ExecSQL(t, db, addSQL)
	testutil.AssertColumnExists(t, db, "test_alter_add", "email")
}

func TestAlterTable_DropColumn_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create table with column to drop
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "alter_drop"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "temp_col", Type: "string", TypeArgs: []any{50}},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Drop column
	dropOp := &ast.DropColumn{
		TableRef: ast.TableRef{Namespace: "test", Table_: "alter_drop"},
		Name:     "temp_col",
	}
	dropSQL, err := d.DropColumnSQL(dropOp)
	if err != nil {
		t.Fatalf("failed to generate drop column SQL: %v", err)
	}

	testutil.ExecSQL(t, db, dropSQL)
	testutil.AssertColumnNotExists(t, db, "test_alter_drop", "temp_col")
}

func TestDropTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create table to drop
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "to_drop"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)
	testutil.AssertTableExists(t, db, "test_to_drop")

	// Drop it
	dropOp := &ast.DropTable{
		TableOp:  ast.TableOp{Namespace: "test", Name: "to_drop"},
		IfExists: true,
	}
	dropSQL, err := d.DropTableSQL(dropOp)
	if err != nil {
		t.Fatalf("failed to generate drop SQL: %v", err)
	}

	testutil.ExecSQL(t, db, dropSQL)
	testutil.AssertTableNotExists(t, db, "test_to_drop")
}

func TestRenameTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create table to rename
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "old_name"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Rename it
	renameOp := &ast.RenameTable{
		Namespace: "test",
		OldName:   "old_name",
		NewName:   "new_name",
	}
	renameSQL, err := d.RenameTableSQL(renameOp)
	if err != nil {
		t.Fatalf("failed to generate rename SQL: %v", err)
	}

	testutil.ExecSQL(t, db, renameSQL)
	testutil.AssertTableNotExists(t, db, "test_old_name")
	testutil.AssertTableExists(t, db, "test_new_name")
}

func TestDropIndex_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create table and index
	tableOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "drop_idx"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}
	tableSQL, _ := d.CreateTableSQL(tableOp)
	testutil.ExecSQL(t, db, tableSQL)

	indexOp := &ast.CreateIndex{
		TableRef: ast.TableRef{Namespace: "test", Table_: "drop_idx"},
		Name:     "idx_to_drop",
		Columns:  []string{"email"},
	}
	indexSQL, _ := d.CreateIndexSQL(indexOp)
	testutil.ExecSQL(t, db, indexSQL)
	testutil.AssertIndexExists(t, db, "test_drop_idx", "idx_to_drop")

	// Drop index
	dropOp := &ast.DropIndex{
		TableRef: ast.TableRef{Namespace: "test", Table_: "drop_idx"},
		Name:     "idx_to_drop",
		IfExists: true,
	}
	dropSQL, err := d.DropIndexSQL(dropOp)
	if err != nil {
		t.Fatalf("failed to generate drop index SQL: %v", err)
	}

	testutil.ExecSQL(t, db, dropSQL)
}

// =============================================================================
// SQLite Tests
// =============================================================================

func TestCreateTable_SimpleTypes_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "simple_types"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{255}},
			{Name: "description", Type: "text"},
			{Name: "count", Type: "integer"},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "active", Type: "boolean", Default: false, DefaultSet: true},
			{Name: "created_at", Type: "date_time"},
			{Name: "external_id", Type: "uuid"},
			{Name: "metadata", Type: "json", Nullable: true, NullableSet: true},
		},
	}

	sqlStmt, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sqlStmt)
	testutil.AssertTableExists(t, db, "test_simple_types")
	testutil.AssertColumnExists(t, db, "test_simple_types", "id")
	testutil.AssertColumnExists(t, db, "test_simple_types", "metadata")
}

func TestCreateTable_WithForeignKey_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// First create the referenced table
	parentOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "user"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true},
		},
	}
	parentSQL, _ := d.CreateTableSQL(parentOp)
	testutil.ExecSQL(t, db, parentSQL)

	// Now create table with FK (SQLite supports FK at creation time)
	childOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "content", Name: "post"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "author_id", Type: "uuid"},
			{Name: "title", Type: "string", TypeArgs: []any{200}},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_post_author",
				Columns:    []string{"author_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
		},
	}

	childSQL, err := d.CreateTableSQL(childOp)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, childSQL)
	testutil.AssertTableExists(t, db, "content_post")
}

func TestCreateTable_WithIndex_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create table first
	tableOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "indexed"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "status", Type: "string", TypeArgs: []any{50}},
		},
	}
	tableSQL, _ := d.CreateTableSQL(tableOp)
	testutil.ExecSQL(t, db, tableSQL)

	// Create index
	indexOp := &ast.CreateIndex{
		TableRef:    ast.TableRef{Namespace: "test", Table_: "indexed"},
		Name:        "idx_indexed_email_status",
		Columns:     []string{"email", "status"},
		IfNotExists: true,
	}
	indexSQL, err := d.CreateIndexSQL(indexOp)
	if err != nil {
		t.Fatalf("failed to generate index SQL: %v", err)
	}

	testutil.ExecSQL(t, db, indexSQL)
	testutil.AssertIndexExists(t, db, "test_indexed", "idx_indexed_email_status")
}

func TestCreateTable_WithUniqueConstraint_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "unique_cols"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "code", Type: "string", TypeArgs: []any{50}, Unique: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
		},
	}

	sqlStmt, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sqlStmt)
	testutil.AssertTableExists(t, db, "test_unique_cols")

	// Verify unique constraint works
	testutil.ExecSQL(t, db, `INSERT INTO test_unique_cols (id, code, name) VALUES ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'ABC', 'First')`)
	_, err = db.Exec(`INSERT INTO test_unique_cols (id, code, name) VALUES ('b2c3d4e5-f6a7-8901-bcde-f12345678901', 'ABC', 'Second')`)
	if err == nil {
		t.Error("expected unique constraint violation, but insert succeeded")
	}
}

func TestAlterTable_AddColumn_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create initial table
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "alter_add"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Add column
	addOp := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "test", Table_: "alter_add"},
		Column:   &ast.ColumnDef{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: true, NullableSet: true},
	}
	addSQL, err := d.AddColumnSQL(addOp)
	if err != nil {
		t.Fatalf("failed to generate add column SQL: %v", err)
	}

	testutil.ExecSQL(t, db, addSQL)
	testutil.AssertColumnExists(t, db, "test_alter_add", "email")
}

func TestAlterTable_DropColumn_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create table with column to drop
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "alter_drop"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "temp_col", Type: "string", TypeArgs: []any{50}},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Drop column (requires SQLite 3.35.0+)
	dropOp := &ast.DropColumn{
		TableRef: ast.TableRef{Namespace: "test", Table_: "alter_drop"},
		Name:     "temp_col",
	}
	dropSQL, err := d.DropColumnSQL(dropOp)
	if err != nil {
		t.Fatalf("failed to generate drop column SQL: %v", err)
	}

	testutil.ExecSQL(t, db, dropSQL)
	testutil.AssertColumnNotExists(t, db, "test_alter_drop", "temp_col")
}

func TestDropTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create table to drop
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "to_drop"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)
	testutil.AssertTableExists(t, db, "test_to_drop")

	// Drop it
	dropOp := &ast.DropTable{
		TableOp:  ast.TableOp{Namespace: "test", Name: "to_drop"},
		IfExists: true,
	}
	dropSQL, err := d.DropTableSQL(dropOp)
	if err != nil {
		t.Fatalf("failed to generate drop SQL: %v", err)
	}

	testutil.ExecSQL(t, db, dropSQL)
	testutil.AssertTableNotExists(t, db, "test_to_drop")
}

func TestRenameTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create table to rename
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "old_name"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Rename it
	renameOp := &ast.RenameTable{
		Namespace: "test",
		OldName:   "old_name",
		NewName:   "new_name",
	}
	renameSQL, err := d.RenameTableSQL(renameOp)
	if err != nil {
		t.Fatalf("failed to generate rename SQL: %v", err)
	}

	testutil.ExecSQL(t, db, renameSQL)
	testutil.AssertTableNotExists(t, db, "test_old_name")
	testutil.AssertTableExists(t, db, "test_new_name")
}

func TestDropIndex_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create table and index
	tableOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "drop_idx"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}
	tableSQL, _ := d.CreateTableSQL(tableOp)
	testutil.ExecSQL(t, db, tableSQL)

	indexOp := &ast.CreateIndex{
		TableRef: ast.TableRef{Namespace: "test", Table_: "drop_idx"},
		Name:     "idx_to_drop",
		Columns:  []string{"email"},
	}
	indexSQL, _ := d.CreateIndexSQL(indexOp)
	testutil.ExecSQL(t, db, indexSQL)
	testutil.AssertIndexExists(t, db, "test_drop_idx", "idx_to_drop")

	// Drop index
	dropOp := &ast.DropIndex{
		TableRef: ast.TableRef{Namespace: "test", Table_: "drop_idx"},
		Name:     "idx_to_drop",
		IfExists: true,
	}
	dropSQL, err := d.DropIndexSQL(dropOp)
	if err != nil {
		t.Fatalf("failed to generate drop index SQL: %v", err)
	}

	testutil.ExecSQL(t, db, dropSQL)
}

// =============================================================================
// Cross-Database Type Mapping Tests
// =============================================================================

// TestAllColumnTypes_Postgres verifies all supported column types execute correctly
func TestAllColumnTypes_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "all_types"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "col_string", Type: "string", TypeArgs: []any{100}},
			{Name: "col_text", Type: "text"},
			{Name: "col_integer", Type: "integer"},
			{Name: "col_float", Type: "float"},
			{Name: "col_decimal", Type: "decimal", TypeArgs: []any{19, 4}},
			{Name: "col_boolean", Type: "boolean"},
			{Name: "col_date", Type: "date"},
			{Name: "col_time", Type: "time"},
			{Name: "col_datetime", Type: "date_time"},
			{Name: "col_uuid", Type: "uuid"},
			{Name: "col_json", Type: "json"},
			{Name: "col_base64", Type: "base64"},
		},
	}

	sqlStmt, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sqlStmt)
	verifyAllColumns(t, db, "test_all_types", op.Columns)
}

// TestAllColumnTypes_SQLite verifies all supported column types execute correctly
func TestAllColumnTypes_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "all_types"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "col_string", Type: "string", TypeArgs: []any{100}},
			{Name: "col_text", Type: "text"},
			{Name: "col_integer", Type: "integer"},
			{Name: "col_float", Type: "float"},
			{Name: "col_decimal", Type: "decimal", TypeArgs: []any{19, 4}},
			{Name: "col_boolean", Type: "boolean"},
			{Name: "col_date", Type: "date"},
			{Name: "col_time", Type: "time"},
			{Name: "col_datetime", Type: "date_time"},
			{Name: "col_uuid", Type: "uuid"},
			{Name: "col_json", Type: "json"},
			{Name: "col_base64", Type: "base64"},
		},
	}

	sqlStmt, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sqlStmt)
	verifyAllColumns(t, db, "test_all_types", op.Columns)
}

// verifyAllColumns checks that all expected columns exist in the table
func verifyAllColumns(t *testing.T, db *sql.DB, table string, columns []*ast.ColumnDef) {
	t.Helper()
	for _, col := range columns {
		testutil.AssertColumnExists(t, db, table, col.Name)
	}
}

// =============================================================================
// Default Value Tests
// =============================================================================

func TestDefaultValues_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "defaults"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "count", Type: "integer", Default: 0, DefaultSet: true},
			{Name: "active", Type: "boolean", Default: true, DefaultSet: true},
			{Name: "name", Type: "string", TypeArgs: []any{50}, Default: "unnamed", DefaultSet: true},
		},
	}

	sqlStmt, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sqlStmt)

	// Insert row with defaults
	testutil.ExecSQL(t, db, `INSERT INTO test_defaults (id) VALUES ('a1b2c3d4-e5f6-7890-abcd-ef1234567890')`)

	// Verify defaults were applied
	var count int
	var active bool
	var name string
	err = db.QueryRow(`SELECT count, active, name FROM test_defaults WHERE id = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'`).Scan(&count, &active, &name)
	if err != nil {
		t.Fatalf("failed to query inserted row: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count=0, got %d", count)
	}
	if !active {
		t.Error("expected active=true, got false")
	}
	if name != "unnamed" {
		t.Errorf("expected name='unnamed', got '%s'", name)
	}
}

func TestDefaultValues_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "defaults"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "count", Type: "integer", Default: 0, DefaultSet: true},
			{Name: "active", Type: "boolean", Default: true, DefaultSet: true},
			{Name: "name", Type: "string", TypeArgs: []any{50}, Default: "unnamed", DefaultSet: true},
		},
	}

	sqlStmt, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate SQL: %v", err)
	}

	testutil.ExecSQL(t, db, sqlStmt)

	// Insert row with defaults
	testutil.ExecSQL(t, db, `INSERT INTO test_defaults (id) VALUES ('a1b2c3d4-e5f6-7890-abcd-ef1234567890')`)

	// Verify defaults were applied
	var count int
	var active int // SQLite returns boolean as int
	var name string
	err = db.QueryRow(`SELECT count, active, name FROM test_defaults WHERE id = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'`).Scan(&count, &active, &name)
	if err != nil {
		t.Fatalf("failed to query inserted row: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count=0, got %d", count)
	}
	if active != 1 {
		t.Errorf("expected active=1, got %d", active)
	}
	if name != "unnamed" {
		t.Errorf("expected name='unnamed', got '%s'", name)
	}
}

// =============================================================================
// Unique Index Tests
// =============================================================================

func TestCreateUniqueIndex_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	tableOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "unique_idx"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
		},
	}
	tableSQL, _ := d.CreateTableSQL(tableOp)
	testutil.ExecSQL(t, db, tableSQL)

	indexOp := &ast.CreateIndex{
		TableRef: ast.TableRef{Namespace: "test", Table_: "unique_idx"},
		Name:     "uniq_email_username",
		Columns:  []string{"email", "username"},
		Unique:   true,
	}
	indexSQL, err := d.CreateIndexSQL(indexOp)
	if err != nil {
		t.Fatalf("failed to generate index SQL: %v", err)
	}

	testutil.ExecSQL(t, db, indexSQL)
	testutil.AssertIndexExists(t, db, "test_unique_idx", "uniq_email_username")

	// Verify uniqueness
	testutil.ExecSQL(t, db, `INSERT INTO test_unique_idx (id, email, username) VALUES ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'test@example.com', 'testuser')`)
	_, err = db.Exec(`INSERT INTO test_unique_idx (id, email, username) VALUES ('b2c3d4e5-f6a7-8901-bcde-f12345678901', 'test@example.com', 'testuser')`)
	if err == nil {
		t.Error("expected unique index violation, but insert succeeded")
	}
}

func TestCreateUniqueIndex_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	tableOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "unique_idx"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
		},
	}
	tableSQL, _ := d.CreateTableSQL(tableOp)
	testutil.ExecSQL(t, db, tableSQL)

	indexOp := &ast.CreateIndex{
		TableRef: ast.TableRef{Namespace: "test", Table_: "unique_idx"},
		Name:     "uniq_email_username",
		Columns:  []string{"email", "username"},
		Unique:   true,
	}
	indexSQL, err := d.CreateIndexSQL(indexOp)
	if err != nil {
		t.Fatalf("failed to generate index SQL: %v", err)
	}

	testutil.ExecSQL(t, db, indexSQL)
	testutil.AssertIndexExists(t, db, "test_unique_idx", "uniq_email_username")

	// Verify uniqueness
	testutil.ExecSQL(t, db, `INSERT INTO test_unique_idx (id, email, username) VALUES ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'test@example.com', 'testuser')`)
	_, err = db.Exec(`INSERT INTO test_unique_idx (id, email, username) VALUES ('b2c3d4e5-f6a7-8901-bcde-f12345678901', 'test@example.com', 'testuser')`)
	if err == nil {
		t.Error("expected unique index violation, but insert succeeded")
	}
}

// =============================================================================
// Rename Column Tests
// =============================================================================

func TestRenameColumn_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create table with column to rename
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "rename_col"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "old_name", Type: "string", TypeArgs: []any{100}},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Rename column
	renameOp := &ast.RenameColumn{
		TableRef: ast.TableRef{Namespace: "test", Table_: "rename_col"},
		OldName:  "old_name",
		NewName:  "new_name",
	}
	renameSQL, err := d.RenameColumnSQL(renameOp)
	if err != nil {
		t.Fatalf("failed to generate rename column SQL: %v", err)
	}

	testutil.ExecSQL(t, db, renameSQL)
	testutil.AssertColumnNotExists(t, db, "test_rename_col", "old_name")
	testutil.AssertColumnExists(t, db, "test_rename_col", "new_name")
}

func TestRenameColumn_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// Create table with column to rename
	createOp := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "test", Name: "rename_col"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "old_name", Type: "string", TypeArgs: []any{100}},
		},
	}
	createSQL, _ := d.CreateTableSQL(createOp)
	testutil.ExecSQL(t, db, createSQL)

	// Rename column (SQLite 3.25.0+ supports RENAME COLUMN)
	renameOp := &ast.RenameColumn{
		TableRef: ast.TableRef{Namespace: "test", Table_: "rename_col"},
		OldName:  "old_name",
		NewName:  "new_name",
	}
	renameSQL, err := d.RenameColumnSQL(renameOp)
	if err != nil {
		t.Fatalf("failed to generate rename column SQL: %v", err)
	}

	testutil.ExecSQL(t, db, renameSQL)
	testutil.AssertColumnNotExists(t, db, "test_rename_col", "old_name")
	testutil.AssertColumnExists(t, db, "test_rename_col", "new_name")
}
