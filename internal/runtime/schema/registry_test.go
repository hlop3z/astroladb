package schema

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestNewSchemaRegistry tests the constructor
func TestNewSchemaRegistry(t *testing.T) {
	reg := NewSchemaRegistry()

	if reg == nil {
		t.Fatal("NewSchemaRegistry should return a non-nil registry")
	}
	if reg.Tables() == nil {
		t.Error("Tables should be initialized")
	}
	if reg.Operations() == nil {
		t.Error("Operations should be initialized")
	}
	if reg.Metadata() == nil {
		t.Error("Metadata should be initialized")
	}
	if reg.Parser() == nil {
		t.Error("Parser should be initialized")
	}
}

// TestRegisterTable tests registering a table
func TestRegisterTable(t *testing.T) {
	reg := NewSchemaRegistry()

	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
	}

	reg.RegisterTable(table)

	tables := reg.Tables()
	if len(tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "users" {
		t.Errorf("Expected table name 'users', got '%s'", tables[0].Name)
	}
}

// TestRegisterTable_Nil tests registering nil table (should be ignored)
func TestRegisterTable_Nil(t *testing.T) {
	reg := NewSchemaRegistry()

	reg.RegisterTable(nil)

	tables := reg.Tables()
	if len(tables) != 0 {
		t.Errorf("Expected 0 tables when registering nil, got %d", len(tables))
	}
}

// TestRegisterTable_Multiple tests registering multiple tables
func TestRegisterTable_Multiple(t *testing.T) {
	reg := NewSchemaRegistry()

	table1 := &ast.TableDef{Namespace: "auth", Name: "users"}
	table2 := &ast.TableDef{Namespace: "blog", Name: "posts"}
	table3 := &ast.TableDef{Namespace: "blog", Name: "comments"}

	reg.RegisterTable(table1)
	reg.RegisterTable(table2)
	reg.RegisterTable(table3)

	tables := reg.Tables()
	if len(tables) != 3 {
		t.Fatalf("Expected 3 tables, got %d", len(tables))
	}
}

// TestRegisterOperation tests registering an operation
func TestRegisterOperation(t *testing.T) {
	reg := NewSchemaRegistry()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{
			Namespace: "auth",
			Name:      "users",
		},
	}

	reg.RegisterOperation(op)

	ops := reg.Operations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(ops))
	}
}

// TestRegisterOperation_Nil tests registering nil operation (should be ignored)
func TestRegisterOperation_Nil(t *testing.T) {
	reg := NewSchemaRegistry()

	reg.RegisterOperation(nil)

	ops := reg.Operations()
	if len(ops) != 0 {
		t.Errorf("Expected 0 operations when registering nil, got %d", len(ops))
	}
}

// TestRegisterOperation_Multiple tests registering multiple operations
func TestRegisterOperation_Multiple(t *testing.T) {
	reg := NewSchemaRegistry()

	op1 := &ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}}
	op2 := &ast.CreateTable{TableOp: ast.TableOp{Namespace: "blog", Name: "posts"}}
	op3 := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
		Column:   &ast.ColumnDef{Name: "email"},
	}

	reg.RegisterOperation(op1)
	reg.RegisterOperation(op2)
	reg.RegisterOperation(op3)

	ops := reg.Operations()
	if len(ops) != 3 {
		t.Fatalf("Expected 3 operations, got %d", len(ops))
	}
}

// TestTables tests retrieving tables
func TestTables(t *testing.T) {
	reg := NewSchemaRegistry()

	// Empty initially
	if len(reg.Tables()) != 0 {
		t.Error("Expected empty tables initially")
	}

	// Add some tables
	reg.RegisterTable(&ast.TableDef{Name: "users"})
	reg.RegisterTable(&ast.TableDef{Name: "posts"})

	tables := reg.Tables()
	if len(tables) != 2 {
		t.Fatalf("Expected 2 tables, got %d", len(tables))
	}
}

// TestOperations tests retrieving operations
func TestOperations(t *testing.T) {
	reg := NewSchemaRegistry()

	// Empty initially
	if len(reg.Operations()) != 0 {
		t.Error("Expected empty operations initially")
	}

	// Add some operations
	reg.RegisterOperation(&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}})
	reg.RegisterOperation(&ast.CreateTable{TableOp: ast.TableOp{Namespace: "blog", Name: "posts"}})

	ops := reg.Operations()
	if len(ops) != 2 {
		t.Fatalf("Expected 2 operations, got %d", len(ops))
	}
}

// TestMetadata tests retrieving metadata
func TestMetadata(t *testing.T) {
	reg := NewSchemaRegistry()

	meta := reg.Metadata()
	if meta == nil {
		t.Fatal("Metadata should not be nil")
	}
}

// TestParser tests retrieving parser
func TestParser(t *testing.T) {
	reg := NewSchemaRegistry()

	parser := reg.Parser()
	if parser == nil {
		t.Fatal("Parser should not be nil")
	}
}

// TestClearTables tests clearing tables
func TestClearTables(t *testing.T) {
	reg := NewSchemaRegistry()

	// Add some tables
	reg.RegisterTable(&ast.TableDef{Name: "users"})
	reg.RegisterTable(&ast.TableDef{Name: "posts"})

	if len(reg.Tables()) != 2 {
		t.Fatalf("Expected 2 tables before clear, got %d", len(reg.Tables()))
	}

	// Clear tables
	reg.ClearTables()

	if len(reg.Tables()) != 0 {
		t.Errorf("Expected 0 tables after clear, got %d", len(reg.Tables()))
	}
}

// TestClearOperations tests clearing operations
func TestClearOperations(t *testing.T) {
	reg := NewSchemaRegistry()

	// Add some operations
	reg.RegisterOperation(&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}})
	reg.RegisterOperation(&ast.CreateTable{TableOp: ast.TableOp{Namespace: "blog", Name: "posts"}})

	if len(reg.Operations()) != 2 {
		t.Fatalf("Expected 2 operations before clear, got %d", len(reg.Operations()))
	}

	// Clear operations
	reg.ClearOperations()

	if len(reg.Operations()) != 0 {
		t.Errorf("Expected 0 operations after clear, got %d", len(reg.Operations()))
	}
}

// TestClearMetadata tests clearing metadata
func TestClearMetadata(t *testing.T) {
	reg := NewSchemaRegistry()

	originalMeta := reg.Metadata()
	if originalMeta == nil {
		t.Fatal("Original metadata should not be nil")
	}

	// Clear metadata
	reg.ClearMetadata()

	newMeta := reg.Metadata()
	if newMeta == nil {
		t.Fatal("New metadata should not be nil after clear")
	}

	// Should be a new instance
	if originalMeta == newMeta {
		t.Error("ClearMetadata should create a new metadata instance")
	}
}

// TestClear tests clearing all state
func TestClear(t *testing.T) {
	reg := NewSchemaRegistry()

	// Add tables, operations
	reg.RegisterTable(&ast.TableDef{Name: "users"})
	reg.RegisterTable(&ast.TableDef{Name: "posts"})
	reg.RegisterOperation(&ast.CreateTable{TableOp: ast.TableOp{Namespace: "test", Name: "users"}})

	if len(reg.Tables()) == 0 || len(reg.Operations()) == 0 {
		t.Fatal("Expected tables and operations to be set")
	}

	// Clear all
	reg.Clear()

	// Check all cleared
	if len(reg.Tables()) != 0 {
		t.Errorf("Expected 0 tables after Clear(), got %d", len(reg.Tables()))
	}
	if len(reg.Operations()) != 0 {
		t.Errorf("Expected 0 operations after Clear(), got %d", len(reg.Operations()))
	}
}

// TestAddManyToMany tests adding many-to-many relationship
func TestAddManyToMany(t *testing.T) {
	reg := NewSchemaRegistry()

	reg.AddManyToMany("blog", "posts", "tags", "")

	// Verify it's added to metadata
	meta := reg.Metadata()
	if meta == nil {
		t.Fatal("Metadata should not be nil")
	}
	// The metadata tracks this internally, we just verify no panic
}

// TestAddPolymorphic tests adding polymorphic relationship
func TestAddPolymorphic(t *testing.T) {
	reg := NewSchemaRegistry()

	targets := []string{"posts", "comments", "videos"}
	reg.AddPolymorphic("social", "likes", "likeable", targets)

	// Verify it's added to metadata
	meta := reg.Metadata()
	if meta == nil {
		t.Fatal("Metadata should not be nil")
	}
	// The metadata tracks this internally, we just verify no panic
}

// TestGetJoinTables tests retrieving join tables
func TestGetJoinTables(t *testing.T) {
	reg := NewSchemaRegistry()

	// Should return empty initially
	joinTables := reg.GetJoinTables()
	if joinTables == nil {
		t.Fatal("GetJoinTables should return non-nil slice")
	}
}

// TestParseAndRegisterTable_Basic tests basic table parsing and registration
func TestParseAndRegisterTable_Basic(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns": []any{
			map[string]any{"name": "id", "type": "uuid"},
			map[string]any{"name": "email", "type": "string"},
		},
	}

	table := reg.ParseAndRegisterTable("auth", "users", defObj)

	if table == nil {
		t.Fatal("Expected non-nil table")
	}
	if table.Namespace != "auth" {
		t.Errorf("Expected namespace 'auth', got '%s'", table.Namespace)
	}
	if table.Name != "users" {
		t.Errorf("Expected name 'users', got '%s'", table.Name)
	}
	if len(table.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(table.Columns))
	}

	// Verify it's registered
	tables := reg.Tables()
	if len(tables) != 1 {
		t.Errorf("Expected 1 registered table, got %d", len(tables))
	}
}

// TestParseAndRegisterTable_WithIndexes tests parsing table with indexes
func TestParseAndRegisterTable_WithIndexes(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns": []any{
			map[string]any{"name": "email", "type": "string"},
		},
		"indexes": []any{
			map[string]any{"name": "idx_email", "columns": []any{"email"}},
		},
	}

	table := reg.ParseAndRegisterTable("auth", "users", defObj)

	if len(table.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(table.Indexes))
	}
	if table.Indexes[0].Name != "idx_email" {
		t.Errorf("Expected index name 'idx_email', got '%s'", table.Indexes[0].Name)
	}
}

// TestParseAndRegisterTable_WithDocs tests parsing table with documentation
func TestParseAndRegisterTable_WithDocs(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns":    []any{},
		"docs":       "User authentication table",
		"deprecated": "Use new_users instead",
	}

	table := reg.ParseAndRegisterTable("auth", "users", defObj)

	if table.Docs != "User authentication table" {
		t.Errorf("Expected docs 'User authentication table', got '%s'", table.Docs)
	}
	if table.Deprecated != "Use new_users instead" {
		t.Errorf("Expected deprecated 'Use new_users instead', got '%s'", table.Deprecated)
	}
}

// TestParseAndRegisterTable_WithMetadata tests parsing table with x-db metadata
func TestParseAndRegisterTable_WithMetadata(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns":    []any{},
		"auditable":  true,
		"sort_by":    []any{"created_at", "updated_at"},
		"searchable": []any{"name", "email"},
		"filterable": []any{"status", "role"},
	}

	table := reg.ParseAndRegisterTable("auth", "users", defObj)

	if !table.Auditable {
		t.Error("Expected auditable to be true")
	}
	if len(table.SortBy) != 2 {
		t.Errorf("Expected 2 sort_by fields, got %d", len(table.SortBy))
	}
	if len(table.Searchable) != 2 {
		t.Errorf("Expected 2 searchable fields, got %d", len(table.Searchable))
	}
	if len(table.Filterable) != 2 {
		t.Errorf("Expected 2 filterable fields, got %d", len(table.Filterable))
	}
}

// TestParseAndRegisterTable_WithManyToMany tests parsing table with many_to_many
func TestParseAndRegisterTable_WithManyToMany(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns": []any{
			map[string]any{"name": "id", "type": "uuid"},
			map[string]any{
				"_type": "relationship",
				"relationship": map[string]any{
					"type":   "many_to_many",
					"target": "tags",
				},
			},
		},
	}

	table := reg.ParseAndRegisterTable("blog", "posts", defObj)

	// Should only have 1 column (the relationship is metadata)
	if len(table.Columns) != 1 {
		t.Errorf("Expected 1 column (relationship filtered), got %d", len(table.Columns))
	}
	if table.Columns[0].Name != "id" {
		t.Errorf("Expected first column 'id', got '%s'", table.Columns[0].Name)
	}
}

// TestParseAndRegisterTable_WithPolymorphic tests parsing table with polymorphic relationship
func TestParseAndRegisterTable_WithPolymorphic(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns": []any{
			map[string]any{"name": "id", "type": "uuid"},
			map[string]any{
				"_type": "polymorphic",
				"polymorphic": map[string]any{
					"as":      "commentable",
					"targets": []any{"posts", "videos"},
				},
			},
		},
	}

	table := reg.ParseAndRegisterTable("social", "comments", defObj)

	// Should only have 1 column (the polymorphic is metadata)
	if len(table.Columns) != 1 {
		t.Errorf("Expected 1 column (polymorphic filtered), got %d", len(table.Columns))
	}
}

// TestParseAndRegisterTable_InvalidInput tests error handling
func TestParseAndRegisterTable_InvalidInput(t *testing.T) {
	reg := NewSchemaRegistry()

	tests := []struct {
		name   string
		defObj any
	}{
		{"nil", nil},
		{"string", "not a map"},
		{"number", 42},
		{"array", []string{"invalid"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := reg.ParseAndRegisterTable("test", "table", tt.defObj)
			if table != nil {
				t.Error("Should return nil for invalid input")
			}
		})
	}
}

// TestParseAndRegisterTable_EmptyDefinition tests parsing empty definition
func TestParseAndRegisterTable_EmptyDefinition(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{}

	table := reg.ParseAndRegisterTable("test", "empty", defObj)

	if table == nil {
		t.Fatal("Expected non-nil table for empty definition")
	}
	if len(table.Columns) != 0 {
		t.Errorf("Expected 0 columns, got %d", len(table.Columns))
	}
	if len(table.Indexes) != 0 {
		t.Errorf("Expected 0 indexes, got %d", len(table.Indexes))
	}
}

// TestParseColumnsWithRelationships_InvalidColumns tests handling invalid column types
func TestParseColumnsWithRelationships_InvalidColumns(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns": "invalid", // Not an array
	}

	table := reg.ParseAndRegisterTable("test", "table", defObj)

	if table == nil {
		t.Fatal("Expected non-nil table")
	}
	// Invalid columns should result in empty columns array
	if table.Columns != nil && len(table.Columns) > 0 {
		t.Errorf("Expected empty or nil columns for invalid input, got %d", len(table.Columns))
	}
}

// TestParseColumnsWithRelationships_TypedSlice tests handling []map[string]any columns
func TestParseColumnsWithRelationships_TypedSlice(t *testing.T) {
	reg := NewSchemaRegistry()

	defObj := map[string]any{
		"columns": []map[string]any{
			{"name": "id", "type": "uuid"},
			{"name": "email", "type": "string"},
		},
	}

	table := reg.ParseAndRegisterTable("auth", "users", defObj)

	if len(table.Columns) != 2 {
		t.Errorf("Expected 2 columns from typed slice, got %d", len(table.Columns))
	}
}

// TestSchemaRegistry_CompleteWorkflow tests a complete registry workflow
func TestSchemaRegistry_CompleteWorkflow(t *testing.T) {
	reg := NewSchemaRegistry()

	// Parse and register table
	defObj := map[string]any{
		"columns": []any{
			map[string]any{"name": "id", "type": "uuid", "primary_key": true},
			map[string]any{"name": "email", "type": "string", "unique": true},
		},
		"indexes": []any{
			map[string]any{"name": "idx_email", "columns": []any{"email"}},
		},
		"docs":      "Users table",
		"auditable": true,
	}

	table := reg.ParseAndRegisterTable("auth", "users", defObj)
	if table == nil {
		t.Fatal("Expected table to be parsed")
	}

	// Register an operation
	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: table.Namespace, Name: table.Name},
	}
	reg.RegisterOperation(op)

	// Add relationships
	reg.AddManyToMany("blog", "posts", "tags", "")
	reg.AddPolymorphic("social", "comments", "commentable", []string{"posts", "videos"})

	// Verify state
	if len(reg.Tables()) != 1 {
		t.Errorf("Expected 1 table, got %d", len(reg.Tables()))
	}
	if len(reg.Operations()) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(reg.Operations()))
	}

	// Get join tables
	joinTables := reg.GetJoinTables()
	if joinTables == nil {
		t.Error("GetJoinTables should return non-nil")
	}

	// Clear everything
	reg.Clear()

	if len(reg.Tables()) != 0 {
		t.Errorf("Expected 0 tables after clear, got %d", len(reg.Tables()))
	}
	if len(reg.Operations()) != 0 {
		t.Errorf("Expected 0 operations after clear, got %d", len(reg.Operations()))
	}
}

// TestIndexModifier_SchemaExtraction verifies .index() is extracted from schemas
func TestIndexModifier_SchemaExtraction(t *testing.T) {
	registry := NewSchemaRegistry()

	// Simulate a table with .index() and .unique() modifiers
	tableDef := registry.ParseAndRegisterTable("test", "user", map[string]any{
		"columns": []any{
			map[string]any{
				"name":   "id",
				"type":   "uuid",
				"unique": false,
				"index":  false,
			},
			map[string]any{
				"name":   "username",
				"type":   "string",
				"unique": false,
				"index":  true, // .index() modifier
			},
			map[string]any{
				"name":   "email",
				"type":   "string",
				"unique": true, // .unique() modifier
				"index":  false,
			},
		},
	})

	// Should have 2 indexes: 1 non-unique (username), 1 unique (email)
	if len(tableDef.Indexes) != 2 {
		t.Fatalf("Expected 2 indexes, got %d", len(tableDef.Indexes))
	}

	// Check non-unique index
	foundNonUnique := false
	foundUnique := false
	for _, idx := range tableDef.Indexes {
		if len(idx.Columns) == 1 && idx.Columns[0] == "username" && !idx.Unique {
			foundNonUnique = true
		}
		if len(idx.Columns) == 1 && idx.Columns[0] == "email" && idx.Unique {
			foundUnique = true
		}
	}

	if !foundNonUnique {
		t.Error("Non-unique index on username not found")
	}
	if !foundUnique {
		t.Error("Unique index on email not found")
	}
}
