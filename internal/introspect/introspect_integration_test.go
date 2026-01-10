//go:build integration

package introspect_test

import (
	"context"
	"testing"

	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/introspect"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// =============================================================================
// PostgreSQL Tests
// =============================================================================

func TestIntrospectTable_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			age INTEGER
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "test_users")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	if table == nil {
		t.Fatal("expected table to be non-nil")
	}

	if table.Name != "users" {
		t.Errorf("expected table name 'users', got %q", table.Name)
	}

	if table.Namespace != "test" {
		t.Errorf("expected namespace 'test', got %q", table.Namespace)
	}

	if len(table.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(table.Columns))
	}
}

func TestIntrospectColumns_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE auth_user (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(100),
			age INTEGER NOT NULL DEFAULT 0,
			balance DECIMAL(10,2),
			is_active BOOLEAN NOT NULL DEFAULT true,
			bio TEXT,
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "auth_user")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	tests := []struct {
		name       string
		colName    string
		colType    string
		nullable   bool
		primaryKey bool
		hasDefault bool
	}{
		{"id column", "id", "uuid", false, true, false},
		{"email column", "email", "string", false, false, false},
		{"name column", "name", "string", true, false, false},
		{"age column", "age", "integer", false, false, true},
		{"balance column", "balance", "decimal", true, false, false},
		{"is_active column", "is_active", "boolean", false, false, true},
		{"bio column", "bio", "text", true, false, false},
		{"metadata column", "metadata", "json", true, false, false},
		{"created_at column", "created_at", "date_time", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := table.GetColumn(tt.colName)
			if col == nil {
				t.Fatalf("column %q not found", tt.colName)
			}

			if col.Type != tt.colType {
				t.Errorf("expected type %q, got %q", tt.colType, col.Type)
			}

			if col.Nullable != tt.nullable {
				t.Errorf("expected nullable=%v, got %v", tt.nullable, col.Nullable)
			}

			if col.PrimaryKey != tt.primaryKey {
				t.Errorf("expected primaryKey=%v, got %v", tt.primaryKey, col.PrimaryKey)
			}

			if tt.hasDefault && !col.DefaultSet && col.ServerDefault == "" {
				t.Error("expected column to have default, but it doesn't")
			}
		})
	}
}

func TestIntrospectColumnTypeArgs_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_types (
			id UUID PRIMARY KEY,
			short_name VARCHAR(50),
			long_name VARCHAR(500),
			money DECIMAL(19,4),
			percentage DECIMAL(5,2)
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "test_types")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	tests := []struct {
		colName  string
		typeArgs []any
	}{
		{"short_name", []any{50}},
		{"long_name", []any{500}},
		{"money", []any{19, 4}},
		{"percentage", []any{5, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.colName, func(t *testing.T) {
			col := table.GetColumn(tt.colName)
			if col == nil {
				t.Fatalf("column %q not found", tt.colName)
			}

			if len(col.TypeArgs) != len(tt.typeArgs) {
				t.Fatalf("expected %d type args, got %d", len(tt.typeArgs), len(col.TypeArgs))
			}

			for i, want := range tt.typeArgs {
				got := col.TypeArgs[i]
				if got != want {
					t.Errorf("type arg %d: expected %v, got %v", i, want, got)
				}
			}
		})
	}
}

func TestIntrospectIndexes_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_indexed (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			username VARCHAR(100) NOT NULL,
			country VARCHAR(2),
			created_at TIMESTAMP NOT NULL
		)
	`)
	testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_email ON test_indexed (email)`)
	testutil.ExecSQL(t, db, `CREATE INDEX idx_username ON test_indexed (username)`)
	testutil.ExecSQL(t, db, `CREATE INDEX idx_country_created ON test_indexed (country, created_at)`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "test_indexed")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	if len(table.Indexes) != 3 {
		t.Fatalf("expected 3 indexes, got %d", len(table.Indexes))
	}

	indexMap := make(map[string]bool)
	for _, idx := range table.Indexes {
		indexMap[idx.Name] = true
	}

	expectedIndexes := []string{"idx_email", "idx_username", "idx_country_created"}
	for _, name := range expectedIndexes {
		if !indexMap[name] {
			t.Errorf("expected index %q not found", name)
		}
	}

	// Check unique index
	for _, idx := range table.Indexes {
		if idx.Name == "idx_email" {
			if !idx.Unique {
				t.Error("expected idx_email to be unique")
			}
			if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
				t.Errorf("expected idx_email columns [email], got %v", idx.Columns)
			}
		}
		if idx.Name == "idx_country_created" {
			if idx.Unique {
				t.Error("expected idx_country_created to not be unique")
			}
			if len(idx.Columns) != 2 {
				t.Errorf("expected 2 columns in composite index, got %d", len(idx.Columns))
			}
		}
	}
}

func TestIntrospectForeignKeys_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE auth_user (
			id UUID PRIMARY KEY,
			name VARCHAR(100)
		)
	`)
	testutil.ExecSQL(t, db, `
		CREATE TABLE blog_post (
			id UUID PRIMARY KEY,
			author_id UUID NOT NULL REFERENCES auth_user(id) ON DELETE CASCADE,
			editor_id UUID REFERENCES auth_user(id) ON DELETE SET NULL,
			title VARCHAR(200) NOT NULL
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "blog_post")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	if len(table.ForeignKeys) != 2 {
		t.Fatalf("expected 2 foreign keys, got %d", len(table.ForeignKeys))
	}

	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) != 1 {
			t.Errorf("expected 1 column per FK, got %d", len(fk.Columns))
		}
		if fk.RefTable != "auth_user" {
			t.Errorf("expected ref table 'auth_user', got %q", fk.RefTable)
		}
		if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
			t.Errorf("expected ref column 'id', got %v", fk.RefColumns)
		}

		// Check ON DELETE actions
		if fk.Columns[0] == "author_id" && fk.OnDelete != "CASCADE" {
			t.Errorf("expected author_id ON DELETE CASCADE, got %q", fk.OnDelete)
		}
		if fk.Columns[0] == "editor_id" && fk.OnDelete != "SET NULL" {
			t.Errorf("expected editor_id ON DELETE SET NULL, got %q", fk.OnDelete)
		}
	}
}

func TestIntrospectSchema_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `CREATE TABLE auth_user (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE auth_role (id UUID PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE blog_post (id UUID PRIMARY KEY)`)

	inspector := introspect.New(db, dialect.Postgres())
	schema, err := inspector.IntrospectSchema(ctx)
	if err != nil {
		t.Fatalf("IntrospectSchema failed: %v", err)
	}

	if len(schema.Tables) != 3 {
		t.Errorf("expected 3 tables, got %d", len(schema.Tables))
	}

	expectedTables := []string{"auth.user", "auth.role", "blog.post"}
	for _, name := range expectedTables {
		if _, ok := schema.Tables[name]; !ok {
			t.Errorf("expected table %q not found in schema", name)
		}
	}
}

func TestTableExists_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `CREATE TABLE existing_table (id UUID PRIMARY KEY)`)

	inspector := introspect.New(db, dialect.Postgres())

	exists, err := inspector.TableExists(ctx, "existing_table")
	if err != nil {
		t.Fatalf("TableExists failed: %v", err)
	}
	if !exists {
		t.Error("expected existing_table to exist")
	}

	exists, err = inspector.TableExists(ctx, "nonexistent_table")
	if err != nil {
		t.Fatalf("TableExists failed: %v", err)
	}
	if exists {
		t.Error("expected nonexistent_table to not exist")
	}
}

// =============================================================================
// SQLite Tests
// =============================================================================

func TestIntrospectTable_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			age INTEGER
		)
	`)

	inspector := introspect.New(db, dialect.SQLite())
	table, err := inspector.IntrospectTable(ctx, "test_users")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	if table == nil {
		t.Fatal("expected table to be non-nil")
	}

	if table.Name != "users" {
		t.Errorf("expected table name 'users', got %q", table.Name)
	}

	if len(table.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(table.Columns))
	}
}

func TestIntrospectColumns_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE auth_user (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			name TEXT,
			age INTEGER NOT NULL DEFAULT 0,
			balance NUMERIC,
			is_active INTEGER NOT NULL DEFAULT 1,
			bio TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)

	inspector := introspect.New(db, dialect.SQLite())
	table, err := inspector.IntrospectTable(ctx, "auth_user")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	tests := []struct {
		name       string
		colName    string
		colType    string
		nullable   bool
		primaryKey bool
	}{
		{"id column", "id", "id", false, true},
		{"email column", "email", "text", false, false},
		{"name column", "name", "text", true, false},
		{"age column", "age", "integer", false, false},
		{"balance column", "balance", "decimal", true, false},
		{"is_active column", "is_active", "boolean", false, false},
		{"bio column", "bio", "text", true, false},
		{"created_at column", "created_at", "date_time", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := table.GetColumn(tt.colName)
			if col == nil {
				t.Fatalf("column %q not found", tt.colName)
			}

			if col.Type != tt.colType {
				t.Errorf("expected type %q, got %q", tt.colType, col.Type)
			}

			if col.Nullable != tt.nullable {
				t.Errorf("expected nullable=%v, got %v", tt.nullable, col.Nullable)
			}

			if col.PrimaryKey != tt.primaryKey {
				t.Errorf("expected primaryKey=%v, got %v", tt.primaryKey, col.PrimaryKey)
			}
		})
	}
}

func TestIntrospectIndexes_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_indexed (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			username TEXT NOT NULL,
			country TEXT,
			created_at TEXT NOT NULL
		)
	`)
	testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_email ON test_indexed (email)`)
	testutil.ExecSQL(t, db, `CREATE INDEX idx_username ON test_indexed (username)`)
	testutil.ExecSQL(t, db, `CREATE INDEX idx_country_created ON test_indexed (country, created_at)`)

	inspector := introspect.New(db, dialect.SQLite())
	table, err := inspector.IntrospectTable(ctx, "test_indexed")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	if len(table.Indexes) != 3 {
		t.Fatalf("expected 3 indexes, got %d", len(table.Indexes))
	}

	for _, idx := range table.Indexes {
		if idx.Name == "idx_email" {
			if !idx.Unique {
				t.Error("expected idx_email to be unique")
			}
			if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
				t.Errorf("expected idx_email columns [email], got %v", idx.Columns)
			}
		}
		if idx.Name == "idx_country_created" {
			if idx.Unique {
				t.Error("expected idx_country_created to not be unique")
			}
			if len(idx.Columns) != 2 {
				t.Errorf("expected 2 columns in composite index, got %d", len(idx.Columns))
			}
		}
	}
}

func TestIntrospectForeignKeys_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE auth_user (
			id TEXT PRIMARY KEY,
			name TEXT
		)
	`)
	testutil.ExecSQL(t, db, `
		CREATE TABLE blog_post (
			id TEXT PRIMARY KEY,
			author_id TEXT NOT NULL REFERENCES auth_user(id) ON DELETE CASCADE,
			editor_id TEXT REFERENCES auth_user(id) ON DELETE SET NULL,
			title TEXT NOT NULL
		)
	`)

	inspector := introspect.New(db, dialect.SQLite())
	table, err := inspector.IntrospectTable(ctx, "blog_post")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	if len(table.ForeignKeys) != 2 {
		t.Fatalf("expected 2 foreign keys, got %d", len(table.ForeignKeys))
	}

	for _, fk := range table.ForeignKeys {
		if fk.RefTable != "auth_user" {
			t.Errorf("expected ref table 'auth_user', got %q", fk.RefTable)
		}

		if fk.Columns[0] == "author_id" && fk.OnDelete != "CASCADE" {
			t.Errorf("expected author_id ON DELETE CASCADE, got %q", fk.OnDelete)
		}
		if fk.Columns[0] == "editor_id" && fk.OnDelete != "SET NULL" {
			t.Errorf("expected editor_id ON DELETE SET NULL, got %q", fk.OnDelete)
		}
	}
}

func TestIntrospectSchema_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `CREATE TABLE auth_user (id TEXT PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE auth_role (id TEXT PRIMARY KEY)`)
	testutil.ExecSQL(t, db, `CREATE TABLE blog_post (id TEXT PRIMARY KEY)`)

	inspector := introspect.New(db, dialect.SQLite())
	schema, err := inspector.IntrospectSchema(ctx)
	if err != nil {
		t.Fatalf("IntrospectSchema failed: %v", err)
	}

	if len(schema.Tables) != 3 {
		t.Errorf("expected 3 tables, got %d", len(schema.Tables))
	}
}

func TestTableExists_SQLite(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `CREATE TABLE existing_table (id TEXT PRIMARY KEY)`)

	inspector := introspect.New(db, dialect.SQLite())

	exists, err := inspector.TableExists(ctx, "existing_table")
	if err != nil {
		t.Fatalf("TableExists failed: %v", err)
	}
	if !exists {
		t.Error("expected existing_table to exist")
	}

	exists, err = inspector.TableExists(ctx, "nonexistent_table")
	if err != nil {
		t.Fatalf("TableExists failed: %v", err)
	}
	if exists {
		t.Error("expected nonexistent_table to not exist")
	}
}

// =============================================================================
// Edge Cases and Special Scenarios
// =============================================================================

func TestIntrospectEmptyTable_AllDialects(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (*testing.T, context.Context, introspect.Introspector, string)
	}{
		{
			name: "Postgres",
			setup: func(t *testing.T) (*testing.T, context.Context, introspect.Introspector, string) {
				db := testutil.SetupPostgres(t)
				testutil.ExecSQL(t, db, `CREATE TABLE empty_table (id UUID PRIMARY KEY)`)
				return t, context.Background(), introspect.New(db, dialect.Postgres()), "empty_table"
			},
		},
		{
			name: "SQLite",
			setup: func(t *testing.T) (*testing.T, context.Context, introspect.Introspector, string) {
				db := testutil.SetupSQLite(t)
				testutil.ExecSQL(t, db, `CREATE TABLE empty_table (id TEXT PRIMARY KEY)`)
				return t, context.Background(), introspect.New(db, dialect.SQLite()), "empty_table"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx, inspector, tableName := tt.setup(t)

			table, err := inspector.IntrospectTable(ctx, tableName)
			if err != nil {
				t.Fatalf("IntrospectTable failed: %v", err)
			}

			if table == nil {
				t.Fatal("expected table to be non-nil")
			}

			if len(table.Columns) != 1 {
				t.Errorf("expected 1 column, got %d", len(table.Columns))
			}
		})
	}
}

func TestIntrospectNonexistentTable_AllDialects(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (context.Context, introspect.Introspector)
	}{
		{
			name: "Postgres",
			setup: func(t *testing.T) (context.Context, introspect.Introspector) {
				db := testutil.SetupPostgres(t)
				return context.Background(), introspect.New(db, dialect.Postgres())
			},
		},
		{
			name: "SQLite",
			setup: func(t *testing.T) (context.Context, introspect.Introspector) {
				db := testutil.SetupSQLite(t)
				return context.Background(), introspect.New(db, dialect.SQLite())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, inspector := tt.setup(t)

			table, err := inspector.IntrospectTable(ctx, "nonexistent_table")
			if err != nil {
				t.Fatalf("IntrospectTable failed: %v", err)
			}

			if table != nil {
				t.Error("expected nil table for nonexistent table")
			}
		})
	}
}

func TestIntrospectCompositePrimaryKey_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE user_role (
			user_id UUID NOT NULL,
			role_id UUID NOT NULL,
			assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, role_id)
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "user_role")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	// Both columns should be marked as primary key
	userIdCol := table.GetColumn("user_id")
	roleIdCol := table.GetColumn("role_id")

	if userIdCol == nil || roleIdCol == nil {
		t.Fatal("expected both user_id and role_id columns")
	}

	if !userIdCol.PrimaryKey || !roleIdCol.PrimaryKey {
		t.Error("expected both columns to be marked as primary key")
	}
}

func TestIntrospectInternalTablesSkipped_AllDialects(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (context.Context, introspect.Introspector)
	}{
		{
			name: "Postgres",
			setup: func(t *testing.T) (context.Context, introspect.Introspector) {
				db := testutil.SetupPostgres(t)
				testutil.ExecSQL(t, db, `CREATE TABLE alab_migrations (id SERIAL PRIMARY KEY, name TEXT)`)
				testutil.ExecSQL(t, db, `CREATE TABLE user_data (id UUID PRIMARY KEY)`)
				return context.Background(), introspect.New(db, dialect.Postgres())
			},
		},
		{
			name: "SQLite",
			setup: func(t *testing.T) (context.Context, introspect.Introspector) {
				db := testutil.SetupSQLite(t)
				testutil.ExecSQL(t, db, `CREATE TABLE alab_migrations (id INTEGER PRIMARY KEY, name TEXT)`)
				testutil.ExecSQL(t, db, `CREATE TABLE user_data (id TEXT PRIMARY KEY)`)
				return context.Background(), introspect.New(db, dialect.SQLite())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, inspector := tt.setup(t)

			schema, err := inspector.IntrospectSchema(ctx)
			if err != nil {
				t.Fatalf("IntrospectSchema failed: %v", err)
			}

			if len(schema.Tables) != 1 {
				t.Errorf("expected 1 table (alab_migrations should be skipped), got %d", len(schema.Tables))
			}

			if _, ok := schema.Tables["alab_migrations"]; ok {
				t.Error("alab_migrations should not be in introspected schema")
			}
		})
	}
}

func TestIntrospectUniqueConstraint_AllDialects(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (context.Context, introspect.Introspector, string)
	}{
		{
			name: "Postgres",
			setup: func(t *testing.T) (context.Context, introspect.Introspector, string) {
				db := testutil.SetupPostgres(t)
				testutil.ExecSQL(t, db, `
					CREATE TABLE test_unique (
						id UUID PRIMARY KEY,
						email VARCHAR(255) UNIQUE,
						username VARCHAR(100) NOT NULL
					)
				`)
				testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_username ON test_unique (username)`)
				return context.Background(), introspect.New(db, dialect.Postgres()), "test_unique"
			},
		},
		{
			name: "SQLite",
			setup: func(t *testing.T) (context.Context, introspect.Introspector, string) {
				db := testutil.SetupSQLite(t)
				testutil.ExecSQL(t, db, `
					CREATE TABLE test_unique (
						id TEXT PRIMARY KEY,
						email TEXT UNIQUE,
						username TEXT NOT NULL
					)
				`)
				testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_username ON test_unique (username)`)
				return context.Background(), introspect.New(db, dialect.SQLite()), "test_unique"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, inspector, tableName := tt.setup(t)

			table, err := inspector.IntrospectTable(ctx, tableName)
			if err != nil {
				t.Fatalf("IntrospectTable failed: %v", err)
			}

			// Check that we have unique indexes
			uniqueIndexCount := 0
			for _, idx := range table.Indexes {
				if idx.Unique {
					uniqueIndexCount++
				}
			}

			if uniqueIndexCount < 1 {
				t.Error("expected at least 1 unique index")
			}
		})
	}
}

func TestIntrospectCompositeUniqueConstraint_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE user_follow (
			id UUID PRIMARY KEY,
			follower_id UUID NOT NULL,
			following_id UUID NOT NULL,
			UNIQUE (follower_id, following_id)
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "user_follow")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	foundCompositeUnique := false
	for _, idx := range table.Indexes {
		if idx.Unique && len(idx.Columns) == 2 {
			foundCompositeUnique = true
			break
		}
	}

	if !foundCompositeUnique {
		t.Error("expected to find composite unique constraint")
	}
}

func TestIntrospectDefaultValues_Postgres(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_defaults (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			counter INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)

	inspector := introspect.New(db, dialect.Postgres())
	table, err := inspector.IntrospectTable(ctx, "test_defaults")
	if err != nil {
		t.Fatalf("IntrospectTable failed: %v", err)
	}

	// Check that columns with defaults have DefaultSet or ServerDefault
	defaultCols := []string{"status", "counter", "is_active", "created_at"}
	for _, colName := range defaultCols {
		col := table.GetColumn(colName)
		if col == nil {
			t.Fatalf("column %q not found", colName)
		}

		if !col.DefaultSet && col.ServerDefault == "" {
			t.Errorf("expected column %q to have default value", colName)
		}
	}
}

func TestIntrospectNullableVsNotNull_AllDialects(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (context.Context, introspect.Introspector, string)
	}{
		{
			name: "Postgres",
			setup: func(t *testing.T) (context.Context, introspect.Introspector, string) {
				db := testutil.SetupPostgres(t)
				testutil.ExecSQL(t, db, `
					CREATE TABLE test_nulls (
						id UUID PRIMARY KEY,
						required_field VARCHAR(100) NOT NULL,
						optional_field VARCHAR(100),
						required_int INTEGER NOT NULL,
						optional_int INTEGER
					)
				`)
				return context.Background(), introspect.New(db, dialect.Postgres()), "test_nulls"
			},
		},
		{
			name: "SQLite",
			setup: func(t *testing.T) (context.Context, introspect.Introspector, string) {
				db := testutil.SetupSQLite(t)
				testutil.ExecSQL(t, db, `
					CREATE TABLE test_nulls (
						id TEXT PRIMARY KEY,
						required_field TEXT NOT NULL,
						optional_field TEXT,
						required_int INTEGER NOT NULL,
						optional_int INTEGER
					)
				`)
				return context.Background(), introspect.New(db, dialect.SQLite()), "test_nulls"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, inspector, tableName := tt.setup(t)

			table, err := inspector.IntrospectTable(ctx, tableName)
			if err != nil {
				t.Fatalf("IntrospectTable failed: %v", err)
			}

			nullTests := []struct {
				colName  string
				nullable bool
			}{
				{"id", false},
				{"required_field", false},
				{"optional_field", true},
				{"required_int", false},
				{"optional_int", true},
			}

			for _, nt := range nullTests {
				col := table.GetColumn(nt.colName)
				if col == nil {
					t.Fatalf("column %q not found", nt.colName)
				}

				if col.Nullable != nt.nullable {
					t.Errorf("column %q: expected nullable=%v, got %v", nt.colName, nt.nullable, col.Nullable)
				}
			}
		})
	}
}
