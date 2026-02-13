//go:build integration

package engine_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/engine/diff"
	"github.com/hlop3z/astroladb/internal/introspect"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// executeSQL runs SQL statements against the database.
func executeSQL(t *testing.T, db *sql.DB, sql string) {
	t.Helper()
	if _, err := db.Exec(sql); err != nil {
		t.Fatalf("failed to execute SQL:\n%s\nerror: %v", sql, err)
	}
}

// createTableFromDef generates and executes CREATE TABLE SQL from a TableDef.
func createTableFromDef(t *testing.T, db *sql.DB, d dialect.Dialect, table *ast.TableDef) {
	t.Helper()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{
			Namespace: table.Namespace,
			Name:      table.Name,
		},
		Columns:     table.Columns,
		Indexes:     table.Indexes,
		ForeignKeys: table.ForeignKeys,
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("failed to generate CREATE TABLE SQL: %v", err)
	}

	executeSQL(t, db, sql)
}

// createIndexFromDef generates and executes CREATE INDEX SQL.
func createIndexFromDef(t *testing.T, db *sql.DB, d dialect.Dialect, table *ast.TableDef, idx *ast.IndexDef) {
	t.Helper()

	op := &ast.CreateIndex{
		TableRef: ast.TableRef{
			Namespace: table.Namespace,
			Table_:    table.Name,
		},
		Name:    idx.Name,
		Columns: idx.Columns,
		Unique:  idx.Unique,
	}

	sql, err := d.CreateIndexSQL(op)
	if err != nil {
		t.Fatalf("failed to generate CREATE INDEX SQL: %v", err)
	}

	executeSQL(t, db, sql)
}

// introspectAndCompare introspects a table and compares to expected definition.
func introspectAndCompare(t *testing.T, db *sql.DB, d dialect.Dialect, expected *ast.TableDef) {
	t.Helper()
	ctx := context.Background()

	intro := introspect.New(db, d)
	if intro == nil {
		t.Fatalf("failed to create introspector for dialect %s", d.Name())
	}

	actual, err := intro.IntrospectTable(ctx, expected.FullName())
	if err != nil {
		t.Fatalf("failed to introspect table %s: %v", expected.FullName(), err)
	}
	if actual == nil {
		t.Fatalf("table %s not found after creation", expected.FullName())
	}

	// Compare table name
	if actual.Name != expected.Name {
		t.Errorf("table name mismatch: got %q, want %q", actual.Name, expected.Name)
	}

	// Compare columns
	compareColumns(t, actual.Columns, expected.Columns, d)

	// Compare indexes
	compareIndexes(t, actual.Indexes, expected.Indexes)

	// Compare foreign keys
	compareForeignKeys(t, actual.ForeignKeys, expected.ForeignKeys)
}

// compareColumns compares introspected columns to expected columns.
func compareColumns(t *testing.T, actual, expected []*ast.ColumnDef, d dialect.Dialect) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Errorf("column count mismatch: got %d, want %d", len(actual), len(expected))
		for _, col := range actual {
			t.Logf("  actual: %s (%s)", col.Name, col.Type)
		}
		for _, col := range expected {
			t.Logf("  expected: %s (%s)", col.Name, col.Type)
		}
		return
	}

	// Build map for expected columns
	expectedMap := make(map[string]*ast.ColumnDef)
	for _, col := range expected {
		expectedMap[col.Name] = col
	}

	for _, actualCol := range actual {
		expectedCol, ok := expectedMap[actualCol.Name]
		if !ok {
			t.Errorf("unexpected column %q in introspected table", actualCol.Name)
			continue
		}

		// Compare type (with dialect-specific normalization)
		compareColumnType(t, actualCol, expectedCol, d)

		// Compare nullability
		if actualCol.Nullable != expectedCol.Nullable {
			t.Errorf("column %q nullable mismatch: got %v, want %v",
				actualCol.Name, actualCol.Nullable, expectedCol.Nullable)
		}

		// Compare primary key
		if actualCol.PrimaryKey != expectedCol.PrimaryKey {
			t.Errorf("column %q primary key mismatch: got %v, want %v",
				actualCol.Name, actualCol.PrimaryKey, expectedCol.PrimaryKey)
		}
	}
}

// compareColumnType compares types with dialect-specific normalization.
func compareColumnType(t *testing.T, actual, expected *ast.ColumnDef, d dialect.Dialect) {
	t.Helper()

	// Handle type normalization
	actualType := normalizeType(actual.Type, actual.TypeArgs, d)
	expectedType := normalizeType(expected.Type, expected.TypeArgs, d)

	if actualType != expectedType {
		// SQLite is especially loose with types, so be more lenient
		if d.Name() == "sqlite" {
			// SQLite often maps everything to TEXT, INTEGER, REAL, BLOB
			if isSQLiteTypeEquivalent(actual.Type, expected.Type) {
				return
			}
		}
		t.Errorf("column %q type mismatch: got %q, want %q",
			actual.Name, actualType, expectedType)
	}
}

// normalizeType creates a comparable type string.
func normalizeType(typeName string, typeArgs []any, d dialect.Dialect) string {
	switch typeName {
	case "id":
		return "id"
	case "string":
		if len(typeArgs) > 0 {
			if length, ok := typeArgs[0].(int); ok {
				return "string(" + itoa(length) + ")"
			}
			if length, ok := typeArgs[0].(float64); ok {
				return "string(" + itoa(int(length)) + ")"
			}
		}
		return "string(255)"
	case "decimal":
		p, s := 10, 2
		if len(typeArgs) > 0 {
			if prec, ok := typeArgs[0].(int); ok {
				p = prec
			} else if prec, ok := typeArgs[0].(float64); ok {
				p = int(prec)
			}
		}
		if len(typeArgs) > 1 {
			if scale, ok := typeArgs[1].(int); ok {
				s = scale
			} else if scale, ok := typeArgs[1].(float64); ok {
				s = int(scale)
			}
		}
		return "decimal(" + itoa(p) + "," + itoa(s) + ")"
	default:
		return typeName
	}
}

// isSQLiteTypeEquivalent checks if SQLite type mappings are equivalent.
func isSQLiteTypeEquivalent(actual, expected string) bool {
	// SQLite uses TEXT for many types
	textTypes := map[string]bool{
		"string": true, "text": true, "uuid": true, "id": true,
		"date": true, "time": true, "datetime": true, "json": true, "decimal": true,
	}
	if actual == "text" && textTypes[expected] {
		return true
	}
	if expected == "text" && textTypes[actual] {
		return true
	}

	// SQLite uses INTEGER for boolean
	if (actual == "integer" && expected == "boolean") ||
		(actual == "boolean" && expected == "integer") {
		return true
	}

	return actual == expected
}

// itoa is a simple int to string converter.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var result string
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n = n / 10
	}
	return result
}

// compareIndexes compares introspected indexes to expected indexes.
func compareIndexes(t *testing.T, actual, expected []*ast.IndexDef) {
	t.Helper()

	// Build map by columns (name may be auto-generated)
	expectedByKey := make(map[string]*ast.IndexDef)
	for _, idx := range expected {
		key := indexKey(idx)
		expectedByKey[key] = idx
	}

	for _, actualIdx := range actual {
		key := indexKey(actualIdx)
		expectedIdx, ok := expectedByKey[key]
		if !ok {
			// May be auto-generated index (FK, unique), skip for now
			continue
		}

		// Compare uniqueness
		if actualIdx.Unique != expectedIdx.Unique {
			t.Errorf("index %q unique mismatch: got %v, want %v",
				actualIdx.Name, actualIdx.Unique, expectedIdx.Unique)
		}
	}
}

// indexKey creates a unique key for an index based on columns.
func indexKey(idx *ast.IndexDef) string {
	key := ""
	if idx.Unique {
		key = "unique:"
	}
	for i, col := range idx.Columns {
		if i > 0 {
			key += ","
		}
		key += col
	}
	return key
}

// compareForeignKeys compares introspected FKs to expected FKs.
func compareForeignKeys(t *testing.T, actual, expected []*ast.ForeignKeyDef) {
	t.Helper()

	// Build map by local columns
	expectedByKey := make(map[string]*ast.ForeignKeyDef)
	for _, fk := range expected {
		key := fkKey(fk)
		expectedByKey[key] = fk
	}

	for _, actualFK := range actual {
		key := fkKey(actualFK)
		expectedFK, ok := expectedByKey[key]
		if !ok {
			continue // May be auto-generated
		}

		// Compare reference table
		if actualFK.RefTable != expectedFK.RefTable {
			t.Errorf("FK %q ref table mismatch: got %q, want %q",
				actualFK.Name, actualFK.RefTable, expectedFK.RefTable)
		}
	}
}

// fkKey creates a unique key for a FK based on columns.
func fkKey(fk *ast.ForeignKeyDef) string {
	key := ""
	for i, col := range fk.Columns {
		if i > 0 {
			key += ","
		}
		key += col
	}
	key += "->"
	for i, col := range fk.RefColumns {
		if i > 0 {
			key += ","
		}
		key += col
	}
	return key
}

// -----------------------------------------------------------------------------
// Simple Table Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_SimpleTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Define a simple table
	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "created_at", Type: "datetime"},
		},
	}

	// Create table
	createTableFromDef(t, db, d, expected)

	// Introspect and compare
	introspectAndCompare(t, db, d, expected)
}

func TestRoundTrip_SimpleTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "created_at", Type: "datetime"},
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// Column Type Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_AllColumnTypes_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "all_types",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "str_col", Type: "string", TypeArgs: []any{50}},
			{Name: "text_col", Type: "text"},
			{Name: "int_col", Type: "integer"},
			{Name: "float_col", Type: "float"},
			{Name: "decimal_col", Type: "decimal", TypeArgs: []any{10, 4}},
			{Name: "bool_col", Type: "boolean"},
			{Name: "date_col", Type: "date"},
			{Name: "time_col", Type: "time"},
			{Name: "datetime_col", Type: "datetime"},
			{Name: "uuid_col", Type: "uuid"},
			{Name: "json_col", Type: "json"},
			{Name: "bytes_col", Type: "base64"},
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}

func TestRoundTrip_AllColumnTypes_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "all_types",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "str_col", Type: "string", TypeArgs: []any{50}},
			{Name: "text_col", Type: "text"},
			{Name: "int_col", Type: "integer"},
			{Name: "float_col", Type: "float"},
			{Name: "decimal_col", Type: "decimal", TypeArgs: []any{10, 4}},
			{Name: "bool_col", Type: "boolean"},
			{Name: "date_col", Type: "date"},
			{Name: "time_col", Type: "time"},
			{Name: "datetime_col", Type: "datetime"},
			{Name: "uuid_col", Type: "uuid"},
			{Name: "json_col", Type: "json"},
			{Name: "bytes_col", Type: "base64"},
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// Nullable Column Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_NullableColumns_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "nullable",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "required_col", Type: "string", TypeArgs: []any{100}, Nullable: false},
			{Name: "optional_col", Type: "string", TypeArgs: []any{100}, Nullable: true, NullableSet: true},
			{Name: "int_optional", Type: "integer", Nullable: true, NullableSet: true},
			{Name: "date_optional", Type: "date", Nullable: true, NullableSet: true},
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}

func TestRoundTrip_NullableColumns_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "nullable",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "required_col", Type: "string", TypeArgs: []any{100}, Nullable: false},
			{Name: "optional_col", Type: "string", TypeArgs: []any{100}, Nullable: true, NullableSet: true},
			{Name: "int_optional", Type: "integer", Nullable: true, NullableSet: true},
			{Name: "date_optional", Type: "date", Nullable: true, NullableSet: true},
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// Index Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_WithIndexes_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "indexed",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
			{Name: "created_at", Type: "datetime"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_test_indexed_email", Columns: []string{"email"}, Unique: true},
			{Name: "idx_test_indexed_created_at", Columns: []string{"created_at"}, Unique: false},
		},
	}

	createTableFromDef(t, db, d, expected)

	// Create indexes separately
	for _, idx := range expected.Indexes {
		createIndexFromDef(t, db, d, expected, idx)
	}

	introspectAndCompare(t, db, d, expected)
}

func TestRoundTrip_WithIndexes_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "indexed",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "username", Type: "string", TypeArgs: []any{50}},
			{Name: "created_at", Type: "datetime"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_test_indexed_email", Columns: []string{"email"}, Unique: true},
			{Name: "idx_test_indexed_created_at", Columns: []string{"created_at"}, Unique: false},
		},
	}

	createTableFromDef(t, db, d, expected)

	for _, idx := range expected.Indexes {
		createIndexFromDef(t, db, d, expected, idx)
	}

	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// Composite Index Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_CompositeIndexes_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "composite",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "user_id", Type: "uuid"},
			{Name: "product_id", Type: "uuid"},
			{Name: "quantity", Type: "integer"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_test_composite_user_product", Columns: []string{"user_id", "product_id"}, Unique: true},
		},
	}

	createTableFromDef(t, db, d, expected)

	for _, idx := range expected.Indexes {
		createIndexFromDef(t, db, d, expected, idx)
	}

	introspectAndCompare(t, db, d, expected)
}

func TestRoundTrip_CompositeIndexes_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "composite",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "user_id", Type: "uuid"},
			{Name: "product_id", Type: "uuid"},
			{Name: "quantity", Type: "integer"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_test_composite_user_product", Columns: []string{"user_id", "product_id"}, Unique: true},
		},
	}

	createTableFromDef(t, db, d, expected)

	for _, idx := range expected.Indexes {
		createIndexFromDef(t, db, d, expected, idx)
	}

	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// Foreign Key Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_ForeignKey_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// First, create the referenced table
	userTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}
	createTableFromDef(t, db, d, userTable)

	// Then create the referencing table
	postTable := &ast.TableDef{
		Namespace: "content",
		Name:      "post",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "author_id", Type: "uuid"},
			{Name: "title", Type: "string", TypeArgs: []any{200}},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_content_post_author_id",
				Columns:    []string{"author_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
		},
	}
	createTableFromDef(t, db, d, postTable)

	// Verify foreign key
	ctx := context.Background()
	intro := introspect.New(db, d)

	actual, err := intro.IntrospectTable(ctx, postTable.FullName())
	if err != nil {
		t.Fatalf("failed to introspect table: %v", err)
	}

	if len(actual.ForeignKeys) == 0 {
		t.Error("expected at least one foreign key, got none")
	} else {
		fk := actual.ForeignKeys[0]
		if fk.RefTable != "auth_user" {
			t.Errorf("FK ref table mismatch: got %q, want %q", fk.RefTable, "auth_user")
		}
		if len(fk.Columns) == 0 || fk.Columns[0] != "author_id" {
			t.Errorf("FK column mismatch: got %v, want [author_id]", fk.Columns)
		}
	}
}

func TestRoundTrip_ForeignKey_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	// First, create the referenced table
	userTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}
	createTableFromDef(t, db, d, userTable)

	// Then create the referencing table
	postTable := &ast.TableDef{
		Namespace: "content",
		Name:      "post",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "author_id", Type: "uuid"},
			{Name: "title", Type: "string", TypeArgs: []any{200}},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_content_post_author_id",
				Columns:    []string{"author_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
		},
	}
	createTableFromDef(t, db, d, postTable)

	// Verify foreign key
	ctx := context.Background()
	intro := introspect.New(db, d)

	actual, err := intro.IntrospectTable(ctx, postTable.FullName())
	if err != nil {
		t.Fatalf("failed to introspect table: %v", err)
	}

	if len(actual.ForeignKeys) == 0 {
		t.Error("expected at least one foreign key, got none")
	} else {
		fk := actual.ForeignKeys[0]
		if fk.RefTable != "auth_user" {
			t.Errorf("FK ref table mismatch: got %q, want %q", fk.RefTable, "auth_user")
		}
	}
}

// -----------------------------------------------------------------------------
// Self-Referential FK Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_SelfReferentialFK_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Table with self-referential FK (e.g., parent-child)
	expected := &ast.TableDef{
		Namespace: "org",
		Name:      "category",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "parent_id", Type: "uuid", Nullable: true, NullableSet: true},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_org_category_parent_id",
				Columns:    []string{"parent_id"},
				RefTable:   "org_category",
				RefColumns: []string{"id"},
				OnDelete:   "SET NULL",
			},
		},
	}

	createTableFromDef(t, db, d, expected)

	ctx := context.Background()
	intro := introspect.New(db, d)

	actual, err := intro.IntrospectTable(ctx, expected.FullName())
	if err != nil {
		t.Fatalf("failed to introspect table: %v", err)
	}

	if len(actual.ForeignKeys) == 0 {
		t.Error("expected self-referential foreign key")
	} else {
		fk := actual.ForeignKeys[0]
		if fk.RefTable != "org_category" {
			t.Errorf("self-ref FK table mismatch: got %q, want %q", fk.RefTable, "org_category")
		}
	}
}

// -----------------------------------------------------------------------------
// Complex Schema Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_ComplexSchema_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	// Create a complex table with all features
	expected := &ast.TableDef{
		Namespace: "ecommerce",
		Name:      "order",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "order_number", Type: "string", TypeArgs: []any{50}, Unique: true},
			{Name: "customer_email", Type: "string", TypeArgs: []any{255}},
			{Name: "total_amount", Type: "decimal", TypeArgs: []any{19, 4}},
			{Name: "currency", Type: "string", TypeArgs: []any{3}},
			{Name: "status", Type: "string", TypeArgs: []any{20}},
			{Name: "notes", Type: "text", Nullable: true, NullableSet: true},
			{Name: "metadata", Type: "json", Nullable: true, NullableSet: true},
			{Name: "is_paid", Type: "boolean"},
			{Name: "shipped_at", Type: "datetime", Nullable: true, NullableSet: true},
			{Name: "created_at", Type: "datetime"},
			{Name: "updated_at", Type: "datetime"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_ecommerce_order_customer_email", Columns: []string{"customer_email"}},
			{Name: "idx_ecommerce_order_status_created", Columns: []string{"status", "created_at"}},
		},
	}

	createTableFromDef(t, db, d, expected)

	for _, idx := range expected.Indexes {
		createIndexFromDef(t, db, d, expected, idx)
	}

	introspectAndCompare(t, db, d, expected)
}

func TestRoundTrip_ComplexSchema_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()

	expected := &ast.TableDef{
		Namespace: "ecommerce",
		Name:      "order",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "order_number", Type: "string", TypeArgs: []any{50}, Unique: true},
			{Name: "customer_email", Type: "string", TypeArgs: []any{255}},
			{Name: "total_amount", Type: "decimal", TypeArgs: []any{19, 4}},
			{Name: "currency", Type: "string", TypeArgs: []any{3}},
			{Name: "status", Type: "string", TypeArgs: []any{20}},
			{Name: "notes", Type: "text", Nullable: true, NullableSet: true},
			{Name: "metadata", Type: "json", Nullable: true, NullableSet: true},
			{Name: "is_paid", Type: "boolean"},
			{Name: "shipped_at", Type: "datetime", Nullable: true, NullableSet: true},
			{Name: "created_at", Type: "datetime"},
			{Name: "updated_at", Type: "datetime"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_ecommerce_order_customer_email", Columns: []string{"customer_email"}},
			{Name: "idx_ecommerce_order_status_created", Columns: []string{"status", "created_at"}},
		},
	}

	createTableFromDef(t, db, d, expected)

	for _, idx := range expected.Indexes {
		createIndexFromDef(t, db, d, expected, idx)
	}

	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// Full Schema Round-Trip with Diff
// -----------------------------------------------------------------------------

func TestRoundTrip_SchemaDiff_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()
	ctx := context.Background()

	// Create initial schema
	userTable := &ast.TableDef{
		Namespace: "app",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
		},
	}
	createTableFromDef(t, db, d, userTable)

	// Introspect current state
	intro := introspect.New(db, d)
	currentSchema, err := intro.IntrospectSchema(ctx)
	if err != nil {
		t.Fatalf("failed to introspect schema: %v", err)
	}

	// Define target schema (add a column)
	targetSchema := engine.NewSchema()
	targetSchema.Tables["app.user"] = &ast.TableDef{
		Namespace: "app",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "created_at", Type: "datetime", Nullable: true, NullableSet: true},
		},
	}

	// Compute diff
	ops, err := diff.Diff(currentSchema, targetSchema)
	if err != nil {
		t.Fatalf("failed to compute diff: %v", err)
	}

	// Should have one AddColumn operation
	if len(ops) != 1 {
		t.Errorf("expected 1 operation, got %d", len(ops))
		for _, op := range ops {
			t.Logf("  op: %v", op.Type())
		}
	}

	// Verify it's an AddColumn
	if len(ops) > 0 {
		addCol, ok := ops[0].(*ast.AddColumn)
		if !ok {
			t.Errorf("expected AddColumn operation, got %T", ops[0])
		} else if addCol.Column.Name != "created_at" {
			t.Errorf("expected column 'created_at', got %q", addCol.Column.Name)
		}
	}
}

func TestRoundTrip_SchemaDiff_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	d := dialect.SQLite()
	ctx := context.Background()

	// Empty database -> Create new table
	intro := introspect.New(db, d)
	currentSchema, err := intro.IntrospectSchema(ctx)
	if err != nil {
		t.Fatalf("failed to introspect schema: %v", err)
	}

	// Define target schema (new table)
	targetSchema := engine.NewSchema()
	targetSchema.Tables["app.user"] = &ast.TableDef{
		Namespace: "app",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	// Compute diff
	ops, err := diff.Diff(currentSchema, targetSchema)
	if err != nil {
		t.Fatalf("failed to compute diff: %v", err)
	}

	// Should have one CreateTable operation
	if len(ops) != 1 {
		t.Errorf("expected 1 operation, got %d", len(ops))
	}

	if len(ops) > 0 {
		createOp, ok := ops[0].(*ast.CreateTable)
		if !ok {
			t.Errorf("expected CreateTable operation, got %T", ops[0])
		} else if createOp.Name != "user" {
			t.Errorf("expected table 'user', got %q", createOp.Name)
		}
	}
}

// -----------------------------------------------------------------------------
// Decimal Precision Round-Trip Tests
// -----------------------------------------------------------------------------

func TestRoundTrip_DecimalPrecision_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	expected := &ast.TableDef{
		Namespace: "finance",
		Name:      "transaction",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "amount_small", Type: "decimal", TypeArgs: []any{5, 2}},  // 999.99
			{Name: "amount_large", Type: "decimal", TypeArgs: []any{19, 4}}, // money
			{Name: "percentage", Type: "decimal", TypeArgs: []any{5, 2}},    // 100.00
			{Name: "rate", Type: "decimal", TypeArgs: []any{10, 8}},         // high precision rate
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}

// -----------------------------------------------------------------------------
// String Length Variations
// -----------------------------------------------------------------------------

func TestRoundTrip_StringLengths_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	d := dialect.Postgres()

	expected := &ast.TableDef{
		Namespace: "test",
		Name:      "strings",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "tiny", Type: "string", TypeArgs: []any{10}},
			{Name: "small", Type: "string", TypeArgs: []any{50}},
			{Name: "medium", Type: "string", TypeArgs: []any{255}},
			{Name: "large", Type: "string", TypeArgs: []any{1000}},
			{Name: "max", Type: "string", TypeArgs: []any{4000}},
		},
	}

	createTableFromDef(t, db, d, expected)
	introspectAndCompare(t, db, d, expected)
}
