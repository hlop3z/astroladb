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

// TestSchemaRegistry_CompleteWorkflow tests a complete registry workflow
func TestSchemaRegistry_CompleteWorkflow(t *testing.T) {
	reg := NewSchemaRegistry()

	// Register a table directly (no map parsing)
	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", Unique: true},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_email", Columns: []string{"email"}},
		},
		Docs:      "Users table",
		Auditable: true,
	}
	reg.RegisterTable(table)

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
