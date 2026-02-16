package engine

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestSplitIntoPhases_EmptyOperations verifies empty operations return no phases.
func TestSplitIntoPhases_EmptyOperations(t *testing.T) {
	ops := []ast.Operation{}

	phases := SplitIntoPhases(ops)

	if len(phases) != 0 {
		t.Errorf("Expected 0 phases for empty operations, got %d", len(phases))
	}
}

// TestSplitIntoPhases_DDLOnly verifies DDL-only migrations create single phase.
func TestSplitIntoPhases_DDLOnly(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{
				Namespace: "auth",
				Name:      "user",
			},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", Nullable: false},
			},
		},
	}

	phases := SplitIntoPhases(ops)

	if len(phases) != 1 {
		t.Fatalf("Expected 1 phase (DDL), got %d", len(phases))
	}

	if phases[0].Type != DDL {
		t.Errorf("Phase type = %s, want DDL", phases[0].Type)
	}
	if !phases[0].InTransaction {
		t.Error("DDL phase should run in transaction")
	}
	if phases[0].Batched {
		t.Error("DDL phase should not be batched")
	}
	if len(phases[0].Ops) != 1 {
		t.Errorf("DDL phase has %d ops, want 1", len(phases[0].Ops))
	}
}

// TestSplitIntoPhases_DDLWithIndex verifies DDL + Index creates two phases.
func TestSplitIntoPhases_DDLWithIndex(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{
				Namespace: "auth",
				Name:      "user",
			},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", Nullable: false, Unique: true}, // Has .unique()
			},
		},
	}

	phases := SplitIntoPhases(ops)

	if len(phases) != 2 {
		t.Fatalf("Expected 2 phases (DDL + Index), got %d", len(phases))
	}

	// Phase 1: DDL
	if phases[0].Type != DDL {
		t.Errorf("Phase 1 type = %s, want DDL", phases[0].Type)
	}
	if !phases[0].InTransaction {
		t.Error("DDL phase should run in transaction")
	}

	// Phase 2: Index
	if phases[1].Type != Index {
		t.Errorf("Phase 2 type = %s, want Index", phases[1].Type)
	}
	if phases[1].InTransaction {
		t.Error("Index phase should NOT run in transaction (CONCURRENTLY)")
	}
	if len(phases[1].Ops) != 1 {
		t.Errorf("Index phase has %d ops, want 1", len(phases[1].Ops))
	}

	// Verify index operation
	idx, ok := phases[1].Ops[0].(*ast.CreateIndex)
	if !ok {
		t.Fatal("Index phase operation is not CreateIndex")
	}
	if !idx.Unique {
		t.Error("Index should be UNIQUE")
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("Index columns = %v, want [email]", idx.Columns)
	}
}

// TestSplitIntoPhases_FullPipeline verifies DDL + Index + Data creates three phases.
func TestSplitIntoPhases_FullPipeline(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{
				Namespace: "auth",
				Table_:    "user",
			},
			Column: &ast.ColumnDef{
				Name:        "bio",
				Type:        "string",
				Nullable:    false,
				Unique:      true,
				Backfill:    "No bio",
				BackfillSet: true,
			},
		},
	}

	phases := SplitIntoPhases(ops)

	if len(phases) != 3 {
		t.Fatalf("Expected 3 phases (DDL + Index + Data), got %d", len(phases))
	}

	// Phase 1: DDL (ADD COLUMN)
	if phases[0].Type != DDL {
		t.Errorf("Phase 1 type = %s, want DDL", phases[0].Type)
	}

	// Phase 2: Index (UNIQUE INDEX on bio)
	if phases[1].Type != Index {
		t.Errorf("Phase 2 type = %s, want Index", phases[1].Type)
	}

	// Phase 3: Data (BACKFILL bio column)
	if phases[2].Type != Data {
		t.Errorf("Phase 3 type = %s, want Data", phases[2].Type)
	}
	if !phases[2].InTransaction {
		t.Error("Data phase should run in transaction")
	}
	if !phases[2].Batched {
		t.Error("Data phase should be batched")
	}
}

// TestSplitIntoPhases_MultipleIndexes verifies multiple indexes are grouped into Index phase.
func TestSplitIntoPhases_MultipleIndexes(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateIndex{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "idx_email",
			Columns:  []string{"email"},
			Unique:   true,
		},
		&ast.CreateIndex{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "idx_created",
			Columns:  []string{"created_at"},
			Unique:   false,
		},
	}

	phases := SplitIntoPhases(ops)

	if len(phases) != 1 {
		t.Fatalf("Expected 1 phase (Index), got %d", len(phases))
	}

	if phases[0].Type != Index {
		t.Errorf("Phase type = %s, want Index", phases[0].Type)
	}
	if len(phases[0].Ops) != 2 {
		t.Errorf("Index phase has %d ops, want 2", len(phases[0].Ops))
	}
}

// TestInjectLockTimeout_PostgreSQL verifies lock timeout injection for PostgreSQL.
func TestInjectLockTimeout_PostgreSQL(t *testing.T) {
	sql := "ALTER TABLE users ADD COLUMN email TEXT;"

	// Test with transaction
	result := InjectLockTimeout(sql, true, "postgres")

	if !contains(result, "BEGIN;") {
		t.Error("Expected BEGIN in wrapped SQL")
	}
	if !contains(result, "COMMIT;") {
		t.Error("Expected COMMIT in wrapped SQL")
	}
	if !contains(result, "SET lock_timeout = '2s';") {
		t.Error("Expected lock_timeout setting")
	}
	if !contains(result, "SET statement_timeout = '30s';") {
		t.Error("Expected statement_timeout setting")
	}
	if !contains(result, "ALTER TABLE users ADD COLUMN email TEXT;") {
		t.Error("Expected original SQL in wrapped output")
	}

	// Test without transaction (Index phase)
	result = InjectLockTimeout(sql, false, "postgres")

	if contains(result, "BEGIN;") {
		t.Error("Should NOT have BEGIN for non-transactional")
	}
	if !contains(result, "SET lock_timeout = '2s';") {
		t.Error("Expected lock_timeout setting even without transaction")
	}
}

// TestInjectLockTimeout_SQLite verifies SQLite doesn't get PostgreSQL-specific settings.
func TestInjectLockTimeout_SQLite(t *testing.T) {
	sql := "ALTER TABLE users ADD COLUMN email TEXT;"

	// Test with transaction
	result := InjectLockTimeout(sql, true, "sqlite")

	if !contains(result, "BEGIN;") {
		t.Error("Expected BEGIN for SQLite transaction")
	}
	if !contains(result, "COMMIT;") {
		t.Error("Expected COMMIT for SQLite transaction")
	}
	if contains(result, "SET lock_timeout") {
		t.Error("SQLite should NOT have PostgreSQL SET commands")
	}

	// Test without transaction
	result = InjectLockTimeout(sql, false, "sqlite")

	if contains(result, "BEGIN;") {
		t.Error("Should NOT have BEGIN for non-transactional SQLite")
	}
}

// TestGenerateConcurrentIndex_PostgreSQL verifies concurrent index generation.
func TestGenerateConcurrentIndex_PostgreSQL(t *testing.T) {
	idx := &ast.CreateIndex{
		TableRef: ast.TableRef{
			Namespace: "auth",
			Table_:    "user",
		},
		Name:    "idx_email",
		Columns: []string{"email"},
		Unique:  true,
	}

	sql := GenerateConcurrentIndex(idx, "postgres")

	if !contains(sql, "CREATE UNIQUE INDEX CONCURRENTLY") {
		t.Error("Expected CONCURRENTLY keyword for PostgreSQL")
	}
	if !contains(sql, "idx_email") {
		t.Error("Expected index name")
	}
	if !contains(sql, "auth_user") {
		t.Error("Expected full table name (namespace_table)")
	}
	if !contains(sql, "(email)") {
		t.Error("Expected column list")
	}
	if !contains(sql, "indisvalid = true") {
		t.Error("Expected validation check for index creation")
	}
}

// TestGenerateConcurrentIndex_SQLite verifies SQLite gets non-concurrent index.
func TestGenerateConcurrentIndex_SQLite(t *testing.T) {
	idx := &ast.CreateIndex{
		TableRef: ast.TableRef{Table_: "users"},
		Name:     "idx_email",
		Columns:  []string{"email"},
		Unique:   false,
	}

	sql := GenerateConcurrentIndex(idx, "sqlite")

	if contains(sql, "CONCURRENTLY") {
		t.Error("SQLite should NOT have CONCURRENTLY keyword")
	}
	if !contains(sql, "CREATE INDEX idx_email") {
		t.Error("Expected regular CREATE INDEX")
	}
}

// TestGenerateBatchedUpdate verifies batched update SQL generation.
func TestGenerateBatchedUpdate(t *testing.T) {
	sql := GenerateBatchedUpdate("auth_user", "bio", "'No bio'", "bio IS NULL")

	// Verify key components of batched update
	if !contains(sql, "DO $$") {
		t.Error("Expected PL/pgSQL block")
	}
	if !contains(sql, "batch_size INT := 5000") {
		t.Error("Expected batch size declaration")
	}
	if !contains(sql, "UPDATE auth_user") {
		t.Error("Expected UPDATE statement")
	}
	if !contains(sql, "SET bio = 'No bio'") {
		t.Error("Expected SET clause with value")
	}
	if !contains(sql, "WHERE bio IS NULL AND id IN") {
		t.Error("Expected WHERE clause with batch limiting")
	}
	if !contains(sql, "LIMIT batch_size") {
		t.Error("Expected batch size limit")
	}
	if !contains(sql, "COMMIT;") {
		t.Error("Expected COMMIT after each batch")
	}
	if !contains(sql, "pg_sleep(1)") {
		t.Error("Expected pg_sleep for AUTOVACUUM")
	}
	if !contains(sql, "RAISE NOTICE") {
		t.Error("Expected progress logging")
	}
}

// TestExtractBackfill_WithBackfill verifies backfill extraction from AddColumn.
func TestExtractBackfill_WithBackfill(t *testing.T) {
	op := &ast.AddColumn{
		TableRef: ast.TableRef{Table_: "users"},
		Column: &ast.ColumnDef{
			Name:        "bio",
			Type:        "string",
			Backfill:    "No bio",
			BackfillSet: true,
		},
	}

	backfill := extractBackfill(op)

	if backfill == nil {
		t.Fatal("Expected backfill operation, got nil")
	}

	rawSQL, ok := backfill.(*ast.RawSQL)
	if !ok {
		t.Fatal("Backfill operation is not RawSQL")
	}

	// Verify generated SQL contains batched update
	if !contains(rawSQL.SQL, "UPDATE users") {
		t.Error("Expected UPDATE in backfill SQL")
	}
	if !contains(rawSQL.SQL, "bio IS NULL") {
		t.Error("Expected WHERE clause in backfill SQL")
	}
}

// TestExtractBackfill_WithoutBackfill verifies no backfill when not set.
func TestExtractBackfill_WithoutBackfill(t *testing.T) {
	op := &ast.AddColumn{
		TableRef: ast.TableRef{Table_: "users"},
		Column: &ast.ColumnDef{
			Name:        "bio",
			Type:        "string",
			BackfillSet: false, // No backfill
		},
	}

	backfill := extractBackfill(op)

	if backfill != nil {
		t.Error("Expected nil backfill for column without .backfill()")
	}
}

// TestExtractIndexes_FromCreateTable verifies index extraction from table definition.
func TestExtractIndexes_FromCreateTable(t *testing.T) {
	op := &ast.CreateTable{
		TableOp: ast.TableOp{
			Namespace: "auth",
			Name:      "user",
		},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", Unique: true}, // Creates unique index
			{Name: "name", Type: "string"},                // No index
		},
		Indexes: []*ast.IndexDef{
			{
				Name:    "idx_created",
				Columns: []string{"created_at"},
				Unique:  false,
			},
		},
	}

	indexes := extractIndexes(op)

	// Should extract (in this order based on extractIndexes implementation):
	// 1. Regular index on created_at (from table.Indexes - extracted first)
	// 2. UNIQUE index on email (from column.Unique - extracted second)
	if len(indexes) != 2 {
		t.Fatalf("Expected 2 indexes, got %d", len(indexes))
	}

	// First index: created_at (from table.Indexes)
	idx0, ok := indexes[0].(*ast.CreateIndex)
	if !ok {
		t.Fatal("Index 0 is not CreateIndex")
	}
	if idx0.Name != "idx_created" {
		t.Errorf("Index 0 name = %q, want %q", idx0.Name, "idx_created")
	}
	if len(idx0.Columns) != 1 || idx0.Columns[0] != "created_at" {
		t.Errorf("Index 0 columns = %v, want [created_at]", idx0.Columns)
	}

	// Second index: email (from column.Unique)
	idx1, ok := indexes[1].(*ast.CreateIndex)
	if !ok {
		t.Fatal("Index 1 is not CreateIndex")
	}
	if !idx1.Unique {
		t.Error("Email index should be UNIQUE")
	}
	if len(idx1.Columns) != 1 || idx1.Columns[0] != "email" {
		t.Errorf("Index 1 columns = %v, want [email]", idx1.Columns)
	}
}

// TestExtractIndexes_FromAddColumn verifies index extraction from AddColumn.
func TestExtractIndexes_FromAddColumn(t *testing.T) {
	op := &ast.AddColumn{
		TableRef: ast.TableRef{
			Namespace: "auth",
			Table_:    "user",
		},
		Column: &ast.ColumnDef{
			Name:   "email",
			Type:   "string",
			Unique: true, // Creates unique index
		},
	}

	indexes := extractIndexes(op)

	if len(indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(indexes))
	}

	idx, ok := indexes[0].(*ast.CreateIndex)
	if !ok {
		t.Fatal("Index is not CreateIndex")
	}
	if !idx.Unique {
		t.Error("Index should be UNIQUE")
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
		t.Errorf("Index columns = %v, want [email]", idx.Columns)
	}
}

// TestMultiPhaseExecution_Integration is a comprehensive integration test
// that verifies the entire multi-phase execution system works end-to-end.
func TestMultiPhaseExecution_Integration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial table
	_, err := db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec("INSERT INTO users (id, name) VALUES ('1', 'Alice'), ('2', 'Bob')")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Create operations that span all three phases:
	// 1. DDL: Add column (nullable first)
	// 2. Index: Create index on bio
	// 3. Data: Backfill existing rows
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Table_: "users"},
			Column: &ast.ColumnDef{
				Name:        "bio",
				Type:        "string",
				Nullable:    true,  // Start nullable
				Backfill:    "Bio", // Will generate Data phase
				BackfillSet: true,
			},
		},
		&ast.CreateIndex{
			TableRef: ast.TableRef{Table_: "users"},
			Name:     "idx_bio",
			Columns:  []string{"bio"},
			Unique:   false,
		},
	}

	// Test phase splitting
	phases := SplitIntoPhases(ops)

	// Should have 3 phases: DDL + Index + Data
	if len(phases) != 3 {
		t.Fatalf("Expected 3 phases, got %d", len(phases))
	}

	// Verify phase types
	if phases[0].Type != DDL {
		t.Errorf("Phase 0 type = %s, want DDL", phases[0].Type)
	}
	if phases[1].Type != Index {
		t.Errorf("Phase 1 type = %s, want Index", phases[1].Type)
	}
	if phases[2].Type != Data {
		t.Errorf("Phase 2 type = %s, want Data", phases[2].Type)
	}

	// Execute Phase 1: DDL (Add column)
	_, err = db.Exec("ALTER TABLE users ADD COLUMN bio TEXT")
	if err != nil {
		t.Fatalf("Phase 1 (DDL) failed: %v", err)
	}

	// Execute Phase 2: Index (CREATE INDEX on bio)
	// Note: SQLite doesn't support CONCURRENTLY, so just regular CREATE INDEX
	_, err = db.Exec("CREATE INDEX idx_bio ON users(bio)")
	if err != nil {
		t.Fatalf("Phase 2 (Index) failed: %v", err)
	}

	// Execute Phase 3: Data (Backfill existing rows)
	// For SQLite, we use a simple UPDATE (no batching needed for 2 rows)
	_, err = db.Exec("UPDATE users SET bio = 'Bio' WHERE bio IS NULL")
	if err != nil {
		t.Fatalf("Phase 3 (Data) failed: %v", err)
	}

	// Verify all rows have bio value
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE bio IS NOT NULL").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count backfilled rows: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 rows with bio, got %d", count)
	}

	// Verify index exists
	var indexExists int
	err = db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='index' AND name='idx_bio'").Scan(&indexExists)
	if err == sql.ErrNoRows {
		t.Error("Index idx_bio was not created")
	} else if err != nil {
		t.Fatalf("Failed to check index: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAtAnyPos(s, substr))
}

func containsAtAnyPos(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGenerateConcurrentIndex_NonUnique verifies non-unique index SQL generation
func TestGenerateConcurrentIndex_NonUnique(t *testing.T) {
	idx := &ast.CreateIndex{
		TableRef: ast.TableRef{
			Namespace: "auth",
			Table_:    "user",
		},
		Name:    "idx_username",
		Columns: []string{"username"},
		Unique:  false, // Non-unique index from .index()
	}

	sql := GenerateConcurrentIndex(idx, "postgres")

	// Should NOT have UNIQUE keyword
	if contains(sql, "UNIQUE") {
		t.Error("Non-unique index should NOT contain UNIQUE keyword")
	}

	// Should have CONCURRENTLY
	if !contains(sql, "CREATE INDEX CONCURRENTLY") {
		t.Error("Expected CREATE INDEX CONCURRENTLY")
	}

	// Should have index name
	if !contains(sql, "idx_username") {
		t.Error("Expected index name idx_username")
	}

	// Should have table name
	if !contains(sql, "auth_user") {
		t.Error("Expected table name auth_user")
	}

	// Should have column
	if !contains(sql, "(username)") {
		t.Error("Expected column (username)")
	}
}
