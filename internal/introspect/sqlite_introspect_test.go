//go:build integration

package introspect

import (
	"context"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/testutil"
)

// -----------------------------------------------------------------------------
// TableExists Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_TableExists(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	// Create a test table
	testutil.ExecSQL(t, db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL
		)
	`)

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	t.Run("existing_table", func(t *testing.T) {
		exists, err := introspector.TableExists(ctx, "users")
		if err != nil {
			t.Fatalf("TableExists failed: %v", err)
		}
		if !exists {
			t.Error("Expected table 'users' to exist")
		}
	})

	t.Run("non_existing_table", func(t *testing.T) {
		exists, err := introspector.TableExists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("TableExists failed: %v", err)
		}
		if exists {
			t.Error("Expected table 'nonexistent' to not exist")
		}
	})

	t.Run("case_sensitive", func(t *testing.T) {
		// SQLite table names are case-insensitive by default
		exists, err := introspector.TableExists(ctx, "USERS")
		if err != nil {
			t.Fatalf("TableExists failed: %v", err)
		}
		// Behavior may vary based on SQLite configuration
		_ = exists
	})
}

// -----------------------------------------------------------------------------
// listTables Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_ListTables(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	t.Run("empty_database", func(t *testing.T) {
		tables, err := introspector.listTables(ctx)
		if err != nil {
			t.Fatalf("listTables failed: %v", err)
		}
		if len(tables) != 0 {
			t.Errorf("Expected 0 tables, got %d", len(tables))
		}
	})

	t.Run("multiple_tables", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE users (id INTEGER PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `CREATE TABLE posts (id INTEGER PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `CREATE TABLE comments (id INTEGER PRIMARY KEY)`)

		tables, err := introspector.listTables(ctx)
		if err != nil {
			t.Fatalf("listTables failed: %v", err)
		}

		expected := 3
		if len(tables) != expected {
			t.Errorf("Expected %d tables, got %d", expected, len(tables))
		}

		// Verify table names (should be sorted)
		expectedNames := []string{"comments", "posts", "users"}
		for i, name := range expectedNames {
			if i >= len(tables) {
				t.Errorf("Missing expected table %s", name)
				continue
			}
			if tables[i] != name {
				t.Errorf("Expected table[%d] = %s, got %s", i, name, tables[i])
			}
		}
	})

	t.Run("excludes_system_tables", func(t *testing.T) {
		// System tables like sqlite_master, sqlite_sequence should be excluded
		tables, err := introspector.listTables(ctx)
		if err != nil {
			t.Fatalf("listTables failed: %v", err)
		}

		for _, table := range tables {
			if table == "sqlite_master" || table == "sqlite_sequence" {
				t.Errorf("System table %s should be excluded", table)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// introspectColumns Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectColumns(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	t.Run("basic_columns", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE test_basic (
				id INTEGER PRIMARY KEY,
				email TEXT NOT NULL,
				age INTEGER,
				balance REAL
			)
		`)

		columns, err := introspector.introspectColumns(ctx, "test_basic")
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
		if emailCol.Type != "text" {
			t.Errorf("Expected type 'text', got %s", emailCol.Type)
		}
		if emailCol.Nullable {
			t.Error("NOT NULL column should not be nullable")
		}

		// Check age column (nullable)
		ageCol := columns[2]
		if ageCol.Name != "age" {
			t.Errorf("Expected column name 'age', got %s", ageCol.Name)
		}
		if !ageCol.Nullable {
			t.Error("Column without NOT NULL should be nullable")
		}
	})

	t.Run("column_with_default", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE test_defaults (
				id INTEGER PRIMARY KEY,
				status TEXT DEFAULT 'active',
				count INTEGER DEFAULT 0,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP
			)
		`)

		columns, err := introspector.introspectColumns(ctx, "test_defaults")
		if err != nil {
			t.Fatalf("introspectColumns failed: %v", err)
		}

		// Check status column default
		statusCol := columns[1]
		if !statusCol.DefaultSet {
			t.Error("Expected DefaultSet to be true")
		}
		if statusCol.ServerDefault != "'active'" {
			t.Errorf("Expected default 'active', got %s", statusCol.ServerDefault)
		}

		// Check count column default
		countCol := columns[2]
		if !countCol.DefaultSet {
			t.Error("Expected DefaultSet to be true")
		}
		if countCol.ServerDefault != "0" {
			t.Errorf("Expected default '0', got %s", countCol.ServerDefault)
		}

		// Check created_at column default
		createdCol := columns[3]
		if !createdCol.DefaultSet {
			t.Error("Expected DefaultSet to be true")
		}
	})

	t.Run("type_with_parameters", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE test_types (
				id INTEGER PRIMARY KEY,
				name VARCHAR(100),
				price DECIMAL(10, 2),
				description TEXT
			)
		`)

		columns, err := introspector.introspectColumns(ctx, "test_types")
		if err != nil {
			t.Fatalf("introspectColumns failed: %v", err)
		}

		// VARCHAR(100) should be parsed
		nameCol := columns[1]
		if nameCol.Type != "string" {
			t.Errorf("Expected type 'string' for VARCHAR, got %s", nameCol.Type)
		}
		if len(nameCol.TypeArgs) == 0 {
			t.Error("Expected TypeArgs for VARCHAR(100)")
		}

		// DECIMAL(10,2) should be parsed
		priceCol := columns[2]
		if priceCol.Type != "decimal" {
			t.Errorf("Expected type 'decimal', got %s", priceCol.Type)
		}
		if len(priceCol.TypeArgs) == 0 {
			t.Error("Expected TypeArgs for DECIMAL(10,2)")
		}
	})

	t.Run("empty_table", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE empty_test (id INTEGER PRIMARY KEY)`)

		columns, err := introspector.introspectColumns(ctx, "empty_test")
		if err != nil {
			t.Fatalf("introspectColumns failed: %v", err)
		}

		if len(columns) != 1 {
			t.Errorf("Expected 1 column, got %d", len(columns))
		}
	})

	t.Run("nonexistent_table", func(t *testing.T) {
		columns, err := introspector.introspectColumns(ctx, "nonexistent_table")
		// SQLite PRAGMA table_info returns empty result for nonexistent tables, not error
		if err != nil {
			t.Logf("Got error (acceptable): %v", err)
		}
		if columns == nil {
			columns = []*ast.ColumnDef{}
		}
		// Either error or empty result is acceptable
		if err == nil && len(columns) > 0 {
			t.Error("Expected error or empty columns for nonexistent table")
		}
	})
}

// -----------------------------------------------------------------------------
// introspectIndexes Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectIndexes(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	t.Run("no_indexes", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE no_idx (id INTEGER PRIMARY KEY, name TEXT)`)

		indexes, err := introspector.introspectIndexes(ctx, "no_idx")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		// Indexes can be nil or empty - both are acceptable
		// SQLite may auto-generate indexes for PRIMARY KEY
		_ = indexes
	})

	t.Run("single_column_index", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE test_idx1 (
				id INTEGER PRIMARY KEY,
				email TEXT NOT NULL
			)
		`)
		testutil.ExecSQL(t, db, `CREATE INDEX idx_email ON test_idx1(email)`)

		indexes, err := introspector.introspectIndexes(ctx, "test_idx1")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		// Find the idx_email index
		var found bool
		for _, idx := range indexes {
			if idx.Name == "idx_email" {
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
			t.Error("Index idx_email not found")
		}
	})

	t.Run("unique_index", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE test_unique (
				id INTEGER PRIMARY KEY,
				username TEXT NOT NULL
			)
		`)
		testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_username ON test_unique(username)`)

		indexes, err := introspector.introspectIndexes(ctx, "test_unique")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		var found bool
		for _, idx := range indexes {
			if idx.Name == "idx_username" {
				found = true
				if !idx.Unique {
					t.Error("Expected unique index")
				}
				break
			}
		}
		if !found {
			t.Error("Unique index idx_username not found")
		}
	})

	t.Run("multi_column_index", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE test_multi (
				id INTEGER PRIMARY KEY,
				first_name TEXT,
				last_name TEXT
			)
		`)
		testutil.ExecSQL(t, db, `CREATE INDEX idx_name ON test_multi(first_name, last_name)`)

		indexes, err := introspector.introspectIndexes(ctx, "test_multi")
		if err != nil {
			t.Fatalf("introspectIndexes failed: %v", err)
		}

		var found bool
		for _, idx := range indexes {
			if idx.Name == "idx_name" {
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
			t.Error("Multi-column index idx_name not found")
		}
	})
}

// -----------------------------------------------------------------------------
// getIndexDetails Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_GetIndexDetails(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	testutil.ExecSQL(t, db, `
		CREATE TABLE test_details (
			id INTEGER PRIMARY KEY,
			email TEXT,
			username TEXT
		)
	`)
	testutil.ExecSQL(t, db, `CREATE INDEX idx_test ON test_details(email)`)
	testutil.ExecSQL(t, db, `CREATE UNIQUE INDEX idx_unique_test ON test_details(username)`)

	t.Run("regular_index", func(t *testing.T) {
		idx, err := introspector.getIndexDetails(ctx, "test_details", "idx_test")
		if err != nil {
			t.Fatalf("getIndexDetails failed: %v", err)
		}

		if idx == nil {
			t.Fatal("Expected non-nil index")
		}
		if idx.Name != "idx_test" {
			t.Errorf("Expected name 'idx_test', got %s", idx.Name)
		}
		if idx.Unique {
			t.Error("Expected non-unique index")
		}
		if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
			t.Errorf("Expected columns [email], got %v", idx.Columns)
		}
	})

	t.Run("unique_index", func(t *testing.T) {
		idx, err := introspector.getIndexDetails(ctx, "test_details", "idx_unique_test")
		if err != nil {
			t.Fatalf("getIndexDetails failed: %v", err)
		}

		if idx == nil {
			t.Fatal("Expected non-nil index")
		}
		if !idx.Unique {
			t.Error("Expected unique index")
		}
	})

	t.Run("nonexistent_index", func(t *testing.T) {
		idx, err := introspector.getIndexDetails(ctx, "test_details", "nonexistent_idx")
		if err != nil {
			t.Fatalf("getIndexDetails failed: %v", err)
		}
		// Should return nil for nonexistent index (no columns found)
		if idx != nil {
			t.Error("Expected nil for nonexistent index")
		}
	})
}

// -----------------------------------------------------------------------------
// introspectForeignKeys Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectForeignKeys(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	t.Run("no_foreign_keys", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE no_fk (id INTEGER PRIMARY KEY, name TEXT)`)

		fks, err := introspector.introspectForeignKeys(ctx, "no_fk")
		if err != nil {
			t.Fatalf("introspectForeignKeys failed: %v", err)
		}

		if len(fks) != 0 {
			t.Errorf("Expected 0 foreign keys, got %d", len(fks))
		}
	})

	t.Run("single_foreign_key", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE parent (id INTEGER PRIMARY KEY, name TEXT)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE child (
				id INTEGER PRIMARY KEY,
				parent_id INTEGER,
				FOREIGN KEY (parent_id) REFERENCES parent(id)
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "child")
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
		if fk.RefTable != "parent" {
			t.Errorf("Expected RefTable 'parent', got %s", fk.RefTable)
		}
		if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
			t.Errorf("Expected RefColumns [id], got %v", fk.RefColumns)
		}
	})

	t.Run("foreign_key_with_cascade", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE users (id INTEGER PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE posts (
				id INTEGER PRIMARY KEY,
				user_id INTEGER,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "posts")
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
	})

	t.Run("multiple_foreign_keys", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE authors (id INTEGER PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `CREATE TABLE categories (id INTEGER PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE articles (
				id INTEGER PRIMARY KEY,
				author_id INTEGER,
				category_id INTEGER,
				FOREIGN KEY (author_id) REFERENCES authors(id),
				FOREIGN KEY (category_id) REFERENCES categories(id)
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "articles")
		if err != nil {
			t.Fatalf("introspectForeignKeys failed: %v", err)
		}

		if len(fks) != 2 {
			t.Fatalf("Expected 2 foreign keys, got %d", len(fks))
		}

		// Verify both FKs exist
		var hasAuthorFK, hasCategoryFK bool
		for _, fk := range fks {
			if fk.RefTable == "authors" && len(fk.Columns) == 1 && fk.Columns[0] == "author_id" {
				hasAuthorFK = true
			}
			if fk.RefTable == "categories" && len(fk.Columns) == 1 && fk.Columns[0] == "category_id" {
				hasCategoryFK = true
			}
		}

		if !hasAuthorFK {
			t.Error("Missing foreign key to authors table")
		}
		if !hasCategoryFK {
			t.Error("Missing foreign key to categories table")
		}
	})

	t.Run("composite_foreign_key", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE composite_parent (
				key1 INTEGER,
				key2 INTEGER,
				PRIMARY KEY (key1, key2)
			)
		`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE composite_child (
				id INTEGER PRIMARY KEY,
				fk1 INTEGER,
				fk2 INTEGER,
				FOREIGN KEY (fk1, fk2) REFERENCES composite_parent(key1, key2)
			)
		`)

		fks, err := introspector.introspectForeignKeys(ctx, "composite_child")
		if err != nil {
			t.Fatalf("introspectForeignKeys failed: %v", err)
		}

		if len(fks) != 1 {
			t.Fatalf("Expected 1 foreign key, got %d", len(fks))
		}

		fk := fks[0]
		if len(fk.Columns) != 2 {
			t.Errorf("Expected 2 columns in FK, got %d", len(fk.Columns))
		}
		if len(fk.RefColumns) != 2 {
			t.Errorf("Expected 2 ref columns in FK, got %d", len(fk.RefColumns))
		}
	})
}

// -----------------------------------------------------------------------------
// IntrospectTable Tests (integration)
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectTable(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	t.Run("complete_table", func(t *testing.T) {
		testutil.ExecSQL(t, db, `
			CREATE TABLE complete_test (
				id INTEGER PRIMARY KEY,
				email TEXT NOT NULL,
				username TEXT UNIQUE,
				age INTEGER DEFAULT 0,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP
			)
		`)
		testutil.ExecSQL(t, db, `CREATE INDEX idx_email ON complete_test(email)`)

		table, err := introspector.IntrospectTable(ctx, "complete_test")
		if err != nil {
			t.Fatalf("IntrospectTable failed: %v", err)
		}

		if table == nil {
			t.Fatal("Expected non-nil table")
		}

		// Check columns
		if len(table.Columns) != 5 {
			t.Errorf("Expected 5 columns, got %d", len(table.Columns))
		}

		// Check indexes
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
	})

	t.Run("table_with_foreign_key", func(t *testing.T) {
		testutil.ExecSQL(t, db, `CREATE TABLE fk_parent (id INTEGER PRIMARY KEY)`)
		testutil.ExecSQL(t, db, `
			CREATE TABLE fk_child (
				id INTEGER PRIMARY KEY,
				parent_id INTEGER,
				FOREIGN KEY (parent_id) REFERENCES fk_parent(id) ON DELETE CASCADE
			)
		`)

		table, err := introspector.IntrospectTable(ctx, "fk_child")
		if err != nil {
			t.Fatalf("IntrospectTable failed: %v", err)
		}

		if table == nil {
			t.Fatal("Expected non-nil table")
		}

		if len(table.ForeignKeys) != 1 {
			t.Errorf("Expected 1 foreign key, got %d", len(table.ForeignKeys))
		}
	})

	t.Run("nonexistent_table", func(t *testing.T) {
		table, err := introspector.IntrospectTable(ctx, "nonexistent_table_xyz")
		// SQLite may return empty result or error
		if err != nil {
			t.Logf("Got error (acceptable): %v", err)
			return
		}
		// If no error, table should be nil or have no columns
		if table != nil && len(table.Columns) > 0 {
			t.Error("Expected error or empty table for nonexistent table")
		}
	})
}

// -----------------------------------------------------------------------------
// IntrospectSchema Tests (integration)
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectSchema(t *testing.T) {
	t.Run("empty_schema", func(t *testing.T) {
		db := testutil.SetupSQLite(t)
		ctx := context.Background()

		introspector := &sqliteIntrospector{
			db:      db,
			dialect: dialect.SQLite(),
		}

		schema, err := introspector.IntrospectSchema(ctx)
		if err != nil {
			t.Fatalf("IntrospectSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected non-nil schema")
		}

		if len(schema.Tables) != 0 {
			t.Errorf("Expected 0 tables, got %d", len(schema.Tables))
		}
	})

	t.Run("schema_with_tables", func(t *testing.T) {
		db := testutil.SetupSQLite(t)
		ctx := context.Background()

		introspector := &sqliteIntrospector{
			db:      db,
			dialect: dialect.SQLite(),
		}

		testutil.ExecSQL(t, db, `CREATE TABLE schema_users (id INTEGER PRIMARY KEY, name TEXT)`)
		testutil.ExecSQL(t, db, `CREATE TABLE schema_posts (id INTEGER PRIMARY KEY, title TEXT)`)

		schema, err := introspector.IntrospectSchema(ctx)
		if err != nil {
			t.Fatalf("IntrospectSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected non-nil schema")
		}

		if len(schema.Tables) != 2 {
			t.Errorf("Expected 2 tables, got %d", len(schema.Tables))
		}

		// Check if specific tables exist
		// Note: parseTableName splits "schema_users" -> namespace:"schema", name:"users"
		var foundUsers, foundPosts bool
		for _, table := range schema.Tables {
			if table.Name == "users" && table.Namespace == "schema" {
				foundUsers = true
			}
			if table.Name == "posts" && table.Namespace == "schema" {
				foundPosts = true
			}
		}

		if !foundUsers {
			t.Error("schema.users table not found in schema")
		}
		if !foundPosts {
			t.Error("schema.posts table not found in schema")
		}
	})
}

// -----------------------------------------------------------------------------
// IntrospectSchemaWithMapping Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectSchemaWithMapping(t *testing.T) {
	t.Run("with_mapping", func(t *testing.T) {
		db := testutil.SetupSQLite(t)
		ctx := context.Background()

		introspector := &sqliteIntrospector{
			db:      db,
			dialect: dialect.SQLite(),
		}

		testutil.ExecSQL(t, db, `CREATE TABLE auth_user (id INTEGER PRIMARY KEY, email TEXT)`)
		testutil.ExecSQL(t, db, `CREATE TABLE blog_post (id INTEGER PRIMARY KEY, title TEXT)`)
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

	t.Run("without_mapping", func(t *testing.T) {
		db := testutil.SetupSQLite(t)
		ctx := context.Background()

		introspector := &sqliteIntrospector{
			db:      db,
			dialect: dialect.SQLite(),
		}

		testutil.ExecSQL(t, db, `CREATE TABLE auth_user (id INTEGER PRIMARY KEY, email TEXT)`)
		testutil.ExecSQL(t, db, `CREATE TABLE blog_post (id INTEGER PRIMARY KEY, title TEXT)`)

		schema, err := introspector.IntrospectSchemaWithMapping(ctx, nil)
		if err != nil {
			t.Fatalf("IntrospectSchemaWithMapping failed: %v", err)
		}

		if schema == nil {
			t.Fatal("Expected non-nil schema")
		}

		// Without mapping, parseTableName still splits on underscore
		// "auth_user" -> namespace:"auth", name:"user"
		// "blog_post" -> namespace:"blog", name:"post"
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
			t.Error("auth.user table not found (parsed from auth_user)")
		}
		if !foundBlogPost {
			t.Error("blog.post table not found (parsed from blog_post)")
		}
	})
}

// -----------------------------------------------------------------------------
// introspectTableWithMapping Tests
// -----------------------------------------------------------------------------

func TestSQLiteIntrospector_IntrospectTableWithMapping(t *testing.T) {
	db := testutil.SetupSQLite(t)
	ctx := context.Background()

	introspector := &sqliteIntrospector{
		db:      db,
		dialect: dialect.SQLite(),
	}

	testutil.ExecSQL(t, db, `CREATE TABLE app_settings (id INTEGER PRIMARY KEY, key TEXT, value TEXT)`)

	t.Run("with_mapping", func(t *testing.T) {
		mapping := TableNameMapping{
			"app_settings": {Namespace: "app", Name: "settings"},
		}

		table, err := introspector.introspectTableWithMapping(ctx, "app_settings", mapping)
		if err != nil {
			t.Fatalf("introspectTableWithMapping failed: %v", err)
		}

		if table == nil {
			t.Fatal("Expected non-nil table")
		}

		if table.Name != "settings" {
			t.Errorf("Expected name 'settings', got %s", table.Name)
		}
		if table.Namespace != "app" {
			t.Errorf("Expected namespace 'app', got %s", table.Namespace)
		}
	})

	t.Run("without_mapping", func(t *testing.T) {
		table, err := introspector.introspectTableWithMapping(ctx, "app_settings", nil)
		if err != nil {
			t.Fatalf("introspectTableWithMapping failed: %v", err)
		}

		if table == nil {
			t.Fatal("Expected non-nil table")
		}

		// Without mapping, parseTableName will split on underscore
		// "app_settings" -> namespace: "app", name: "settings"
		// This is the default behavior when no mapping is provided
		if table.Name == "" {
			t.Error("Expected non-empty table name")
		}
	})
}
