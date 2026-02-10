//go:build integration

package introspect

import (
	"context"
	"testing"

	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// -----------------------------------------------------------------------------
// TableExists Tests
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_TableExists(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	// Create a test table
	testutil.ExecSQL(t, db, `
		CREATE TABLE IF NOT EXISTS pg_test_users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL
		)
	`)

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	t.Run("existing_table", func(t *testing.T) {
		exists, err := introspector.TableExists(ctx, "pg_test_users")
		if err != nil {
			t.Fatalf("TableExists failed: %v", err)
		}
		if !exists {
			t.Error("Expected table 'pg_test_users' to exist")
		}
	})

	t.Run("non_existing_table", func(t *testing.T) {
		exists, err := introspector.TableExists(ctx, "nonexistent_pg_table")
		if err != nil {
			t.Fatalf("TableExists failed: %v", err)
		}
		if exists {
			t.Error("Expected table 'nonexistent_pg_table' to not exist")
		}
	})

	t.Run("case_sensitivity", func(t *testing.T) {
		// PostgreSQL table names are case-insensitive unless quoted
		exists, err := introspector.TableExists(ctx, "PG_TEST_USERS")
		if err != nil {
			t.Fatalf("TableExists failed: %v", err)
		}
		// Should find it (PostgreSQL folds to lowercase)
		if !exists {
			t.Log("PostgreSQL case handling: uppercase name not found (expected)")
		}
	})

	// Cleanup
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_test_users CASCADE`)
}

// -----------------------------------------------------------------------------
// listTables Tests
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_ListTables(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	// Clean up any existing test tables first
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_list_users CASCADE`)
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_list_posts CASCADE`)
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_list_comments CASCADE`)

	t.Run("with_tables", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE pg_list_users (id SERIAL PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `CREATE TABLE pg_list_posts (id SERIAL PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `CREATE TABLE pg_list_comments (id SERIAL PRIMARY KEY)`)

		tables, err := introspector.listTables(ctx)
		if err != nil {
			t.Fatalf("listTables failed: %v", err)
		}

		// Should have at least our 3 tables
		if len(tables) < 3 {
			t.Errorf("Expected at least 3 tables, got %d", len(tables))
		}

		// Check if our specific tables exist
		hasUsers := false
		hasPosts := false
		hasComments := false
		for _, table := range tables {
			switch table {
			case "pg_list_users":
				hasUsers = true
			case "pg_list_posts":
				hasPosts = true
			case "pg_list_comments":
				hasComments = true
			}
		}

		if !hasUsers {
			t.Error("Expected pg_list_users in table list")
		}
		if !hasPosts {
			t.Error("Expected pg_list_posts in table list")
		}
		if !hasComments {
			t.Error("Expected pg_list_comments in table list")
		}
	})

	// Cleanup
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_list_users CASCADE`)
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_list_posts CASCADE`)
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_list_comments CASCADE`)
}

// -----------------------------------------------------------------------------
// introspectColumns Tests
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_IntrospectColumns(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	t.Run("basic_columns", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_basic_cols CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_basic_cols (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) NOT NULL,
				age INTEGER,
				balance DECIMAL(10,2)
			)
		`)

		columns, err := introspector.introspectColumns(ctx, "pg_basic_cols")
		if err != nil {
			t.Fatalf("introspectColumns failed: %v", err)
		}

		if len(columns) != 4 {
			t.Fatalf("Expected 4 columns, got %d", len(columns))
		}

		// Check id column
		idCol := columns[0]
		if idCol.Name != "id" {
			t.Errorf("Expected column name 'id', got %s", idCol.Name)
		}
		if idCol.Type != "integer" {
			t.Errorf("Expected type 'integer', got %s", idCol.Type)
		}
		if idCol.Nullable {
			t.Error("Primary key should not be nullable")
		}
		if !idCol.PrimaryKey {
			t.Error("Expected PrimaryKey to be true")
		}

		// Check email column
		emailCol := columns[1]
		if emailCol.Name != "email" {
			t.Errorf("Expected column name 'email', got %s", emailCol.Name)
		}
		if emailCol.Type != "string" {
			t.Errorf("Expected type 'string' for VARCHAR, got %s", emailCol.Type)
		}
		if emailCol.Nullable {
			t.Error("NOT NULL column should not be nullable")
		}

		// Check age column (nullable)
		ageCol := columns[2]
		if !ageCol.Nullable {
			t.Error("Column without NOT NULL should be nullable")
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_basic_cols CASCADE`)
	})

	t.Run("column_with_default", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_defaults CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_defaults (
				id SERIAL PRIMARY KEY,
				status TEXT DEFAULT 'active',
				count INTEGER DEFAULT 0,
				created_at TIMESTAMP DEFAULT NOW()
			)
		`)

		columns, err := introspector.introspectColumns(ctx, "pg_defaults")
		if err != nil {
			t.Fatalf("introspectColumns failed: %v", err)
		}

		// Check status column default
		statusCol := columns[1]
		if !statusCol.DefaultSet {
			t.Error("Expected DefaultSet to be true")
		}
		if statusCol.ServerDefault == "" {
			t.Error("Expected non-empty ServerDefault")
		}

		// Check count column default
		countCol := columns[2]
		if !countCol.DefaultSet {
			t.Error("Expected DefaultSet to be true for count")
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_defaults CASCADE`)
	})

	t.Run("uuid_and_jsonb_types", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_types CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_types (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				data JSONB,
				tags TEXT[]
			)
		`)

		columns, err := introspector.introspectColumns(ctx, "pg_types")
		if err != nil {
			t.Fatalf("introspectColumns failed: %v", err)
		}

		// Check UUID column
		idCol := columns[0]
		if idCol.Type != "uuid" {
			t.Errorf("Expected type 'uuid', got %s", idCol.Type)
		}

		// Check JSONB column
		dataCol := columns[1]
		if dataCol.Type != "json" {
			t.Errorf("Expected type 'json' for JSONB, got %s", dataCol.Type)
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_types CASCADE`)
	})
}

// -----------------------------------------------------------------------------
// introspectIndexes Tests
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_IntrospectIndexes(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	t.Run("single_column_index", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_idx1 CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_idx1 (
				id SERIAL PRIMARY KEY,
				email TEXT NOT NULL
			)
		`)
		testutil.ExecSQL(t, db, `CREATE INDEX idx_pg_email ON pg_idx1(email)`)

		indexes, err := introspector.introspectIndexes(ctx, "pg_idx1")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		// Find the idx_pg_email index
		var found bool
		for _, idx := range indexes {
			if idx.Name == "idx_pg_email" {
				found = true
				if len(idx.Columns) != 1 {
					t.Errorf("Expected 1 column in index, got %d", len(idx.Columns))
				}
				if idx.Columns[0] != "email" {
					t.Errorf("Expected column 'email', got %s", idx.Columns[0])
				}
				if idx.Unique {
					t.Error("Expected non-unique index")
				}
				break
			}
		}
		if !found {
			t.Error("Index idx_pg_email not found")
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_idx1 CASCADE`)
	})

	t.Run("unique_index", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_unique CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_unique (
				id SERIAL PRIMARY KEY,
				username TEXT NOT NULL
			)
		`)
		testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_pg_username ON pg_unique(username)`)

		indexes, err := introspector.introspectIndexes(ctx, "pg_unique")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		var found bool
		for _, idx := range indexes {
			if idx.Name == "idx_pg_username" {
				found = true
				if !idx.Unique {
					t.Error("Expected unique index")
				}
				break
			}
		}
		if !found {
			t.Error("Unique index idx_pg_username not found")
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_unique CASCADE`)
	})

	t.Run("multi_column_index", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_multi CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_multi (
				id SERIAL PRIMARY KEY,
				first_name TEXT,
				last_name TEXT
			)
		`)
		testutil.ExecSQL(t, db, `CREATE INDEX idx_pg_name ON pg_multi(first_name, last_name)`)

		indexes, err := introspector.introspectIndexes(ctx, "pg_multi")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		var found bool
		for _, idx := range indexes {
			if idx.Name == "idx_pg_name" {
				found = true
				if len(idx.Columns) != 2 {
					t.Errorf("Expected 2 columns in index, got %d", len(idx.Columns))
				}
				if idx.Columns[0] != "first_name" {
					t.Errorf("Expected first column 'first_name', got %s", idx.Columns[0])
				}
				if idx.Columns[1] != "last_name" {
					t.Errorf("Expected second column 'last_name', got %s", idx.Columns[1])
				}
				break
			}
		}
		if !found {
			t.Error("Multi-column index idx_pg_name not found")
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_multi CASCADE`)
	})
}

// -----------------------------------------------------------------------------
// introspectForeignKeys Tests
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_IntrospectForeignKeys(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	t.Run("single_foreign_key", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_fk_child CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_fk_parent CASCADE`)
		testutil.ExecSQL(t, db, `CREATE TABLE pg_fk_parent (id SERIAL PRIMARY KEY, name TEXT)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_fk_child (
				id SERIAL PRIMARY KEY,
				parent_id INTEGER REFERENCES pg_fk_parent(id)
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "pg_fk_child")
		if err != nil {
			t.Fatalf("introspectForeignKeys failed: %v", err)
		}

		if len(fks) != 1 {
			t.Fatalf("Expected 1 foreign key, got %d", len(fks))
		}

		fk := fks[0]
		if len(fk.Columns) != 1 || fk.Columns[0] != "parent_id" {
			t.Errorf("Expected columns [parent_id], got %v", fk.Columns)
		}
		if fk.RefTable != "pg_fk_parent" {
			t.Errorf("Expected RefTable 'pg_fk_parent', got %s", fk.RefTable)
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_fk_child CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_fk_parent CASCADE`)
	})

	t.Run("foreign_key_with_cascade", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_posts CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_users CASCADE`)
		testutil.ExecSQL(t, db, `CREATE TABLE pg_users (id SERIAL PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_posts (
				id SERIAL PRIMARY KEY,
				user_id INTEGER REFERENCES pg_users(id) ON DELETE CASCADE ON UPDATE CASCADE
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "pg_posts")
		if err != nil {
			t.Fatalf("introspectForeignKeys failed: %v", err)
		}

		if len(fks) != 1 {
			t.Fatalf("Expected 1 foreign key, got %d", len(fks))
		}

		fk := fks[0]
		if fk.OnDelete != "CASCADE" {
			t.Errorf("Expected OnDelete 'CASCADE', got %s", fk.OnDelete)
		}
		if fk.OnUpdate != "CASCADE" {
			t.Errorf("Expected OnUpdate 'CASCADE', got %s", fk.OnUpdate)
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_posts CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_users CASCADE`)
	})

	t.Run("multiple_foreign_keys", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_articles CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_authors CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_categories CASCADE`)

		testutil.ExecSQL(t, db, `CREATE TABLE pg_authors (id SERIAL PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `CREATE TABLE pg_categories (id SERIAL PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_articles (
				id SERIAL PRIMARY KEY,
				author_id INTEGER REFERENCES pg_authors(id),
				category_id INTEGER REFERENCES pg_categories(id)
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "pg_articles")
		if err != nil {
			t.Fatalf("introspectForeignKeys failed: %v", err)
		}

		if len(fks) != 2 {
			t.Fatalf("Expected 2 foreign keys, got %d", len(fks))
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_articles CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_authors CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_categories CASCADE`)
	})
}

// -----------------------------------------------------------------------------
// IntrospectTable Tests (integration)
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_IntrospectTable(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	t.Run("complete_table", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_complete CASCADE`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE pg_complete (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) NOT NULL,
				username TEXT UNIQUE,
				age INTEGER DEFAULT 0
			)
		`)
		testutil.ExecSQL(t, db, `CREATE INDEX idx_pg_email_complete ON pg_complete(email)`)

		table, err := introspector.IntrospectTable(ctx, "pg_complete")
		if err != nil {
			t.Fatalf("IntrospectTable failed: %v", err)
		}

		if table == nil {
			t.Fatal("Expected non-nil table")
		}

		// Check columns
		if len(table.Columns) != 4 {
			t.Errorf("Expected 4 columns, got %d", len(table.Columns))
		}

		// Check indexes (should have at least the one we created)
		if len(table.Indexes) == 0 {
			t.Error("Expected at least one index")
		}

		// Verify specific column
		emailCol := table.GetColumn("email")
		if emailCol == nil {
			t.Fatal("Email column not found")
		}
		if emailCol.Nullable {
			t.Error("Email should not be nullable")
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_complete CASCADE`)
	})
}

// -----------------------------------------------------------------------------
// IntrospectSchema Tests (integration)
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_IntrospectSchema(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	t.Run("schema_with_tables", func(t *testing.T) {
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_schema_users CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_schema_posts CASCADE`)

		testutil.ExecSQL(t, db, `CREATE TABLE pg_schema_users (id SERIAL PRIMARY KEY, name TEXT)`)
		testutil.ExecSQL(t, db, `CREATE TABLE pg_schema_posts (id SERIAL PRIMARY KEY, title TEXT)`)

		schema, err := introspector.IntrospectSchema(ctx)
		if err != nil {
			t.Fatalf("IntrospectSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected non-nil schema")
		}

		// Should have at least our 2 tables
		if len(schema.Tables) < 2 {
			t.Errorf("Expected at least 2 tables, got %d", len(schema.Tables))
		}

		// Cleanup
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_schema_users CASCADE`)
		testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS pg_schema_posts CASCADE`)
	})
}

// -----------------------------------------------------------------------------
// IntrospectSchemaWithMapping Tests
// -----------------------------------------------------------------------------

func TestPostgresIntrospector_IntrospectSchemaWithMapping(t *testing.T) {
	db := testutil.SetupPostgres(t)
	ctx := context.Background()

	introspector := &postgresIntrospector{
		db:      db,
		dialect: dialect.Postgres(),
	}

	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS auth_user CASCADE`)
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS blog_post CASCADE`)
	testutil.ExecSQL(t, db, `CREATE TABLE auth_user (id SERIAL PRIMARY KEY, email TEXT)`)
	testutil.ExecSQL(t, db, `CREATE TABLE blog_post (id SERIAL PRIMARY KEY, title TEXT)`)

	t.Run("with_mapping", func(t *testing.T) {
		mapping := TableNameMapping{
			"auth_user": {Namespace: "auth", Name: "user"},
			"blog_post": {Namespace: "blog", Name: "post"},
		}

		schema, err := introspector.IntrospectSchemaWithMapping(ctx, mapping)
		if err != nil {
			t.Fatalf("IntrospectSchemaWithMapping failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected non-nil schema")
		}

		// Check if tables were mapped correctly
		var foundAuthUser, foundBlogPost bool
		for _, table := range schema.Tables {
			if table.Name == "user" && table.Namespace == "auth" {
				foundAuthUser = true
			}
			if table.Name == "post" && table.Namespace == "blog" {
				foundBlogPost = true
			}
		}

		if !foundAuthUser {
			t.Error("auth.user table not found with correct mapping")
		}
		if !foundBlogPost {
			t.Error("blog.post table not found with correct mapping")
		}
	})

	// Cleanup
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS auth_user CASCADE`)
	testutil.ExecSQL(t, db, `DROP TABLE IF EXISTS blog_post CASCADE`)
}
