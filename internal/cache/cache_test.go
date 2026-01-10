package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/drift"
	"github.com/hlop3z/astroladb/internal/engine"
)

func TestCacheOpenClose(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Open cache
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	// Verify cache directory was created
	cacheDir := filepath.Join(tmpDir, CacheDir)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("cache directory was not created")
	}

	// Verify cache file exists
	cachePath := filepath.Join(cacheDir, CacheFile)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file was not created")
	}

	// Verify path getter
	if c.Path() != cachePath {
		t.Errorf("Path() = %q, want %q", c.Path(), cachePath)
	}
}

func TestSchemaSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	// Create a test schema
	schema := engine.NewSchema()
	schema.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "created_at", Type: "date_time"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_user_email", Columns: []string{"email"}, Unique: true},
		},
	}

	// Store snapshot
	if err := c.SetSchemaSnapshot("001", schema); err != nil {
		t.Fatalf("failed to set schema snapshot: %v", err)
	}

	// Retrieve snapshot
	retrieved, err := c.GetSchemaSnapshot("001")
	if err != nil {
		t.Fatalf("failed to get schema snapshot: %v", err)
	}
	if retrieved == nil {
		t.Fatal("retrieved schema is nil")
	}

	// Verify table exists
	table, exists := retrieved.Tables["auth.user"]
	if !exists {
		t.Fatal("auth.user table not found in retrieved schema")
	}

	// Verify columns
	if len(table.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(table.Columns))
	}
	if table.Columns[0].Name != "id" {
		t.Errorf("first column name = %q, want %q", table.Columns[0].Name, "id")
	}

	// Verify indexes
	if len(table.Indexes) != 1 {
		t.Errorf("expected 1 index, got %d", len(table.Indexes))
	}

	// Test non-existent revision
	notFound, err := c.GetSchemaSnapshot("999")
	if err != nil {
		t.Fatalf("unexpected error for non-existent revision: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent revision")
	}

	// Test list
	revisions, err := c.ListSchemaSnapshots()
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}
	if len(revisions) != 1 || revisions[0] != "001" {
		t.Errorf("ListSchemaSnapshots() = %v, want [001]", revisions)
	}

	// Test delete
	if err := c.DeleteSchemaSnapshot("001"); err != nil {
		t.Fatalf("failed to delete snapshot: %v", err)
	}
	deleted, _ := c.GetSchemaSnapshot("001")
	if deleted != nil {
		t.Error("snapshot should be deleted")
	}
}

func TestMerkleHash(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	// Create test hash
	hash := &drift.SchemaHash{
		Root: "abc123def456",
		Tables: map[string]*drift.TableHash{
			"auth.user": {
				Name:    "auth.user",
				Hash:    "hash1",
				Columns: map[string]string{"id": "col1", "email": "col2"},
				Indexes: map[string]string{"idx_email": "idx1"},
				FKs:     map[string]string{},
			},
		},
	}

	// Store hash
	if err := c.SetMerkleHash("001", hash); err != nil {
		t.Fatalf("failed to set merkle hash: %v", err)
	}

	// Retrieve hash
	retrieved, err := c.GetMerkleHash("001")
	if err != nil {
		t.Fatalf("failed to get merkle hash: %v", err)
	}
	if retrieved == nil {
		t.Fatal("retrieved hash is nil")
	}

	// Verify root
	if retrieved.Root != "abc123def456" {
		t.Errorf("Root = %q, want %q", retrieved.Root, "abc123def456")
	}

	// Verify table hash
	tableHash, exists := retrieved.Tables["auth.user"]
	if !exists {
		t.Fatal("auth.user hash not found")
	}
	if tableHash.Hash != "hash1" {
		t.Errorf("table Hash = %q, want %q", tableHash.Hash, "hash1")
	}

	// Test GetMerkleRootHash
	rootHash, err := c.GetMerkleRootHash("001")
	if err != nil {
		t.Fatalf("failed to get root hash: %v", err)
	}
	if rootHash != "abc123def456" {
		t.Errorf("GetMerkleRootHash() = %q, want %q", rootHash, "abc123def456")
	}

	// Test delete
	if err := c.DeleteMerkleHash("001"); err != nil {
		t.Fatalf("failed to delete hash: %v", err)
	}
}

func TestMigrationMeta(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	// Create test metadata
	meta := &MigrationMeta{
		Revision: "001_initial",
		Checksum: "sha256:abcdef123456",
		FilePath: "migrations/001_initial.js",
		FileSize: 1234,
	}

	// Store metadata
	if err := c.SetMigrationMeta(meta); err != nil {
		t.Fatalf("failed to set migration meta: %v", err)
	}

	// Retrieve metadata
	retrieved, err := c.GetMigrationMeta("001_initial")
	if err != nil {
		t.Fatalf("failed to get migration meta: %v", err)
	}
	if retrieved == nil {
		t.Fatal("retrieved meta is nil")
	}

	// Verify fields
	if retrieved.Checksum != "sha256:abcdef123456" {
		t.Errorf("Checksum = %q, want %q", retrieved.Checksum, "sha256:abcdef123456")
	}
	if retrieved.FileSize != 1234 {
		t.Errorf("FileSize = %d, want %d", retrieved.FileSize, 1234)
	}

	// Test list
	metas, err := c.ListMigrationMeta()
	if err != nil {
		t.Fatalf("failed to list metas: %v", err)
	}
	if len(metas) != 1 {
		t.Errorf("expected 1 meta, got %d", len(metas))
	}
}

func TestCacheClear(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	// Add some data
	schema := engine.NewSchema()
	c.SetSchemaSnapshot("001", schema)
	c.SetMerkleHash("001", &drift.SchemaHash{Root: "test"})
	c.SetMigrationMeta(&MigrationMeta{Revision: "001", Checksum: "test"})

	// Clear
	if err := c.Clear(); err != nil {
		t.Fatalf("failed to clear cache: %v", err)
	}

	// Verify cleared
	stats, _ := c.GetStats()
	if stats.SchemaSnapshots != 0 || stats.MerkleHashes != 0 || stats.MigrationMetas != 0 {
		t.Errorf("cache not cleared: %+v", stats)
	}
}

func TestCacheStats(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	// Get initial stats
	stats, err := c.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.SchemaSnapshots != 0 {
		t.Errorf("expected 0 snapshots, got %d", stats.SchemaSnapshots)
	}

	// Add data and verify counts
	c.SetSchemaSnapshot("001", engine.NewSchema())
	c.SetSchemaSnapshot("002", engine.NewSchema())

	stats, _ = c.GetStats()
	if stats.SchemaSnapshots != 2 {
		t.Errorf("expected 2 snapshots, got %d", stats.SchemaSnapshots)
	}
	if stats.DatabaseSize <= 0 {
		t.Error("expected positive database size")
	}
}

func TestCacheExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if Exists(tmpDir) {
		t.Error("cache should not exist initially")
	}

	// Create cache
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	c.Close()

	// Should exist now
	if !Exists(tmpDir) {
		t.Error("cache should exist after creation")
	}

	// Remove cache
	if err := Remove(tmpDir); err != nil {
		t.Fatalf("failed to remove cache: %v", err)
	}

	// Should not exist after removal
	if Exists(tmpDir) {
		t.Error("cache should not exist after removal")
	}
}

func TestSerializationRoundTrip(t *testing.T) {
	// Create a complex schema
	schema := engine.NewSchema()
	schema.Tables["core.order"] = &ast.TableDef{
		Namespace: "core",
		Name:      "order",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "user_id", Type: "uuid", Reference: &ast.Reference{
				Table:    "auth.user",
				Column:   "id",
				OnDelete: "CASCADE",
			}},
			{Name: "total", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "status", Type: "enum", Default: "pending", DefaultSet: true},
			{Name: "notes", Type: "text", Nullable: true, NullableSet: true},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_order_user", Columns: []string{"user_id"}},
			{Name: "idx_order_status", Columns: []string{"status"}, Where: "status != 'archived'"},
		},
		ForeignKeys: []*ast.ForeignKeyDef{
			{
				Name:       "fk_order_user",
				Columns:    []string{"user_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
		},
		Checks: []*ast.CheckDef{
			{Name: "chk_total_positive", Expression: "total >= 0"},
		},
		Docs:       "Customer orders",
		Deprecated: "",
	}

	// Serialize
	data, err := SerializeSchema(schema)
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Deserialize
	restored, err := DeserializeSchema(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Verify table
	table, exists := restored.Tables["core.order"]
	if !exists {
		t.Fatal("core.order not found in restored schema")
	}

	// Verify column with reference
	userIDCol := table.GetColumn("user_id")
	if userIDCol == nil {
		t.Fatal("user_id column not found")
	}
	if userIDCol.Reference == nil {
		t.Fatal("user_id reference is nil")
	}
	if userIDCol.Reference.OnDelete != "CASCADE" {
		t.Errorf("OnDelete = %q, want %q", userIDCol.Reference.OnDelete, "CASCADE")
	}

	// Verify column with default
	statusCol := table.GetColumn("status")
	if statusCol == nil {
		t.Fatal("status column not found")
	}
	if !statusCol.DefaultSet {
		t.Error("status DefaultSet should be true")
	}

	// Verify nullable column
	notesCol := table.GetColumn("notes")
	if notesCol == nil {
		t.Fatal("notes column not found")
	}
	if !notesCol.Nullable {
		t.Error("notes should be nullable")
	}

	// Verify indexes
	if len(table.Indexes) != 2 {
		t.Errorf("expected 2 indexes, got %d", len(table.Indexes))
	}

	// Verify partial index
	var partialIdx *ast.IndexDef
	for _, idx := range table.Indexes {
		if idx.Where != "" {
			partialIdx = idx
			break
		}
	}
	if partialIdx == nil {
		t.Error("partial index not found")
	}

	// Verify foreign keys
	if len(table.ForeignKeys) != 1 {
		t.Errorf("expected 1 FK, got %d", len(table.ForeignKeys))
	}

	// Verify checks
	if len(table.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(table.Checks))
	}

	// Verify docs
	if table.Docs != "Customer orders" {
		t.Errorf("Docs = %q, want %q", table.Docs, "Customer orders")
	}
}

func TestSQLExprSerialization(t *testing.T) {
	schema := engine.NewSchema()
	schema.Tables["test.table"] = &ast.TableDef{
		Namespace: "test",
		Name:      "table",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "created_at", Type: "date_time", Default: ast.NewSQLExpr("NOW()"), DefaultSet: true},
		},
	}

	// Serialize
	data, err := SerializeSchema(schema)
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Deserialize
	restored, err := DeserializeSchema(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Verify SQLExpr was preserved
	table := restored.Tables["test.table"]
	createdAt := table.GetColumn("created_at")
	if !createdAt.DefaultSet {
		t.Error("DefaultSet should be true")
	}

	expr, ok := ast.AsSQLExpr(createdAt.Default)
	if !ok {
		t.Fatalf("Default is not SQLExpr, got %T", createdAt.Default)
	}
	if expr.Expr != "NOW()" {
		t.Errorf("SQLExpr.Expr = %q, want %q", expr.Expr, "NOW()")
	}
}

func TestGetOrCompute(t *testing.T) {
	tmpDir := t.TempDir()
	c, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer c.Close()

	computeCalls := 0
	compute := func() (*engine.Schema, error) {
		computeCalls++
		s := engine.NewSchema()
		s.Tables["test.table"] = &ast.TableDef{
			Namespace: "test",
			Name:      "table",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid"}},
		}
		return s, nil
	}

	// First call should compute
	schema1, err := c.GetOrCompute("001", compute)
	if err != nil {
		t.Fatalf("GetOrCompute failed: %v", err)
	}
	if computeCalls != 1 {
		t.Errorf("expected 1 compute call, got %d", computeCalls)
	}

	// Second call should use cache
	schema2, err := c.GetOrCompute("001", compute)
	if err != nil {
		t.Fatalf("GetOrCompute failed: %v", err)
	}
	if computeCalls != 1 {
		t.Errorf("expected still 1 compute call, got %d", computeCalls)
	}

	// Both should have the table
	if _, exists := schema1.Tables["test.table"]; !exists {
		t.Error("schema1 missing test.table")
	}
	if _, exists := schema2.Tables["test.table"]; !exists {
		t.Error("schema2 missing test.table")
	}
}
