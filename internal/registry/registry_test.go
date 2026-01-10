package registry

import (
	"sync"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
	"github.com/hlop3z/astroladb/internal/ast"
)

func TestNewModelRegistry(t *testing.T) {
	r := NewModelRegistry()
	if r == nil {
		t.Fatal("NewModelRegistry() returned nil")
	}
	if r.tables == nil {
		t.Error("tables map should be initialized")
	}
	if r.Count() != 0 {
		t.Errorf("New registry should be empty, got %d tables", r.Count())
	}
}

func TestModelRegistry_Register(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{
		Name:    "users",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}

	err := r.Register("auth", "users", def)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if r.Count() != 1 {
		t.Errorf("Count() = %d, want 1", r.Count())
	}

	// Verify namespace and name are set on the definition
	if def.Namespace != "auth" {
		t.Errorf("Namespace = %q, want %q", def.Namespace, "auth")
	}
	if def.Name != "users" {
		t.Errorf("Name = %q, want %q", def.Name, "users")
	}
}

func TestModelRegistry_Register_EmptyNamespace(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{Name: "users"}

	err := r.Register("", "users", def)
	if err == nil {
		t.Error("Register() should error on empty namespace")
	}
	if !alerr.Is(err, alerr.ErrInvalidIdentifier) {
		t.Errorf("Expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestModelRegistry_Register_EmptyTableName(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{Namespace: "auth"}

	err := r.Register("auth", "", def)
	if err == nil {
		t.Error("Register() should error on empty table name")
	}
	if !alerr.Is(err, alerr.ErrInvalidIdentifier) {
		t.Errorf("Expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestModelRegistry_Register_NilDefinition(t *testing.T) {
	r := NewModelRegistry()

	err := r.Register("auth", "users", nil)
	if err == nil {
		t.Error("Register() should error on nil definition")
	}
	if !alerr.Is(err, alerr.ErrSchemaInvalid) {
		t.Errorf("Expected ErrSchemaInvalid, got %v", err)
	}
}

func TestModelRegistry_Register_Duplicate(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}

	err := r.Register("auth", "users", def1)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	err = r.Register("auth", "users", def2)
	if err == nil {
		t.Error("Register() should error on duplicate registration")
	}
	if !alerr.Is(err, alerr.ErrSchemaDuplicate) {
		t.Errorf("Expected ErrSchemaDuplicate, got %v", err)
	}
}

func TestModelRegistry_Register_DifferentNamespaces(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}

	err := r.Register("auth", "users", def1)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Same table name in different namespace should succeed
	err = r.Register("billing", "users", def2)
	if err != nil {
		t.Errorf("Register() should allow same table name in different namespace: %v", err)
	}

	if r.Count() != 2 {
		t.Errorf("Count() = %d, want 2", r.Count())
	}
}

func TestModelRegistry_Get(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{
		Name:    "users",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", def)

	got, ok := r.Get("auth", "users")
	if !ok {
		t.Error("Get() should find registered table")
	}
	if got != def {
		t.Error("Get() returned different definition")
	}
}

func TestModelRegistry_Get_NotFound(t *testing.T) {
	r := NewModelRegistry()

	got, ok := r.Get("auth", "users")
	if ok {
		t.Error("Get() should not find unregistered table")
	}
	if got != nil {
		t.Error("Get() should return nil for unregistered table")
	}
}

func TestModelRegistry_Get_WrongNamespace(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def)

	got, ok := r.Get("billing", "users")
	if ok {
		t.Error("Get() should not find table in wrong namespace")
	}
	if got != nil {
		t.Error("Get() should return nil for wrong namespace")
	}
}

func TestModelRegistry_GetByRef(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{
		Name:    "users",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", def)

	got, err := r.GetByRef("auth.users")
	if err != nil {
		t.Fatalf("GetByRef() error = %v", err)
	}
	if got != def {
		t.Error("GetByRef() returned different definition")
	}
}

func TestModelRegistry_GetByRef_RelativeReference(t *testing.T) {
	r := NewModelRegistry()

	_, err := r.GetByRef(".users")
	if err == nil {
		t.Error("GetByRef() should error on relative reference")
	}
	if !alerr.Is(err, alerr.ErrInvalidReference) {
		t.Errorf("Expected ErrInvalidReference, got %v", err)
	}
}

func TestModelRegistry_GetByRef_UnqualifiedReference(t *testing.T) {
	r := NewModelRegistry()

	_, err := r.GetByRef("users")
	if err == nil {
		t.Error("GetByRef() should error on unqualified reference")
	}
	if !alerr.Is(err, alerr.ErrInvalidReference) {
		t.Errorf("Expected ErrInvalidReference, got %v", err)
	}
}

func TestModelRegistry_GetByRef_NotFound(t *testing.T) {
	r := NewModelRegistry()

	_, err := r.GetByRef("auth.users")
	if err == nil {
		t.Error("GetByRef() should error when table not found")
	}
	if !alerr.Is(err, alerr.ErrSchemaNotFound) {
		t.Errorf("Expected ErrSchemaNotFound, got %v", err)
	}
}

func TestModelRegistry_All(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def1)
	_ = r.Register("blog", "posts", def2)

	all := r.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d tables, want 2", len(all))
	}

	if _, ok := all["auth.users"]; !ok {
		t.Error("All() missing auth.users")
	}
	if _, ok := all["blog.posts"]; !ok {
		t.Error("All() missing blog.posts")
	}
}

func TestModelRegistry_All_ReturnsCopy(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def)

	all := r.All()
	// Modify the returned map
	delete(all, "auth.users")

	// Original registry should be unchanged
	if r.Count() != 1 {
		t.Error("Modifying All() result should not affect registry")
	}
}

func TestModelRegistry_AllInNamespace(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "roles", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def3 := &ast.TableDef{Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def1)
	_ = r.Register("auth", "roles", def2)
	_ = r.Register("blog", "posts", def3)

	authTables := r.AllInNamespace("auth")
	if len(authTables) != 2 {
		t.Errorf("AllInNamespace(\"auth\") returned %d tables, want 2", len(authTables))
	}

	blogTables := r.AllInNamespace("blog")
	if len(blogTables) != 1 {
		t.Errorf("AllInNamespace(\"blog\") returned %d tables, want 1", len(blogTables))
	}

	emptyTables := r.AllInNamespace("nonexistent")
	if len(emptyTables) != 0 {
		t.Errorf("AllInNamespace(\"nonexistent\") returned %d tables, want 0", len(emptyTables))
	}
}

func TestModelRegistry_AllInNamespace_SortedByName(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "zebra", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "alpha", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def3 := &ast.TableDef{Name: "middle", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("test", "zebra", def1)
	_ = r.Register("test", "alpha", def2)
	_ = r.Register("test", "middle", def3)

	tables := r.AllInNamespace("test")
	if len(tables) != 3 {
		t.Fatalf("Expected 3 tables, got %d", len(tables))
	}

	// Should be sorted alphabetically
	expectedOrder := []string{"alpha", "middle", "zebra"}
	for i, expected := range expectedOrder {
		if tables[i].Name != expected {
			t.Errorf("Table %d = %q, want %q", i, tables[i].Name, expected)
		}
	}
}

func TestModelRegistry_Namespaces(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def3 := &ast.TableDef{Name: "roles", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def1)
	_ = r.Register("blog", "posts", def2)
	_ = r.Register("auth", "roles", def3)

	namespaces := r.Namespaces()
	if len(namespaces) != 2 {
		t.Errorf("Namespaces() returned %d, want 2", len(namespaces))
	}

	// Should be sorted
	if namespaces[0] != "auth" || namespaces[1] != "blog" {
		t.Errorf("Namespaces() = %v, want [auth, blog]", namespaces)
	}
}

func TestModelRegistry_Clear(t *testing.T) {
	r := NewModelRegistry()
	def := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def)

	if r.Count() != 1 {
		t.Fatal("Registry should have 1 table before Clear()")
	}

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", r.Count())
	}
}

func TestModelRegistry_Count(t *testing.T) {
	r := NewModelRegistry()

	if r.Count() != 0 {
		t.Errorf("Empty registry Count() = %d, want 0", r.Count())
	}

	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def1)
	_ = r.Register("blog", "posts", def2)

	if r.Count() != 2 {
		t.Errorf("Count() = %d, want 2", r.Count())
	}
}

func TestModelRegistry_CountInNamespace(t *testing.T) {
	r := NewModelRegistry()
	def1 := &ast.TableDef{Name: "users", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def2 := &ast.TableDef{Name: "roles", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	def3 := &ast.TableDef{Name: "posts", Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}}}
	_ = r.Register("auth", "users", def1)
	_ = r.Register("auth", "roles", def2)
	_ = r.Register("blog", "posts", def3)

	if r.CountInNamespace("auth") != 2 {
		t.Errorf("CountInNamespace(\"auth\") = %d, want 2", r.CountInNamespace("auth"))
	}
	if r.CountInNamespace("blog") != 1 {
		t.Errorf("CountInNamespace(\"blog\") = %d, want 1", r.CountInNamespace("blog"))
	}
	if r.CountInNamespace("nonexistent") != 0 {
		t.Errorf("CountInNamespace(\"nonexistent\") = %d, want 0", r.CountInNamespace("nonexistent"))
	}
}

func TestModelRegistry_ThreadSafety(t *testing.T) {
	r := NewModelRegistry()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			def := &ast.TableDef{
				Name:    "table",
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
			}
			// Each goroutine writes to a unique namespace
			ns := "ns" + string(rune('a'+i%26)) + string(rune('0'+i/26))
			_ = r.Register(ns, "table", def)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.All()
			_ = r.Count()
			_ = r.Namespaces()
		}()
	}
	wg.Wait()

	// Mixed reads and writes
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			_ = r.All()
			_ = r.Namespaces()
		}(i)
		go func(i int) {
			defer wg.Done()
			def := &ast.TableDef{
				Name:    "table2",
				Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
			}
			ns := "mixed" + string(rune('a'+i%26))
			_ = r.Register(ns, "table2", def)
		}(i)
	}
	wg.Wait()

	// If we got here without data race or panic, the test passes
	if r.Count() == 0 {
		t.Error("Registry should have some tables after concurrent access")
	}
}

func TestModelRegistry_ConcurrentGetAndRegister(t *testing.T) {
	r := NewModelRegistry()

	// Pre-register a table
	def := &ast.TableDef{
		Name:    "users",
		Columns: []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
	}
	_ = r.Register("auth", "users", def)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent Get operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				got, ok := r.Get("auth", "users")
				if !ok || got == nil {
					t.Error("Concurrent Get() should find the table")
				}
			}
		}()
	}

	// Concurrent GetByRef operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				got, err := r.GetByRef("auth.users")
				if err != nil || got == nil {
					t.Error("Concurrent GetByRef() should succeed")
				}
			}
		}()
	}

	wg.Wait()
}
