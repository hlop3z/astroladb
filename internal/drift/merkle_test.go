package drift

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

func TestComputeSchemaHash_Empty(t *testing.T) {
	schema := engine.NewSchema()

	hash, err := ComputeSchemaHash(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash.Root == "" {
		t.Error("expected non-empty root hash for empty schema")
	}

	if len(hash.Tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(hash.Tables))
	}
}

func TestComputeSchemaHash_SingleTable(t *testing.T) {
	schema := engine.NewSchema()
	schema.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true},
			{Name: "created_at", Type: "datetime"},
		},
		Indexes: []*ast.IndexDef{
			{Name: "idx_user_email", Columns: []string{"email"}, Unique: true},
		},
	}

	hash, err := ComputeSchemaHash(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash.Root == "" {
		t.Error("expected non-empty root hash")
	}

	if len(hash.Tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(hash.Tables))
	}

	tableHash, ok := hash.Tables["auth.user"]
	if !ok {
		t.Fatal("expected auth.user table hash")
	}

	if len(tableHash.Columns) != 3 {
		t.Errorf("expected 3 column hashes, got %d", len(tableHash.Columns))
	}

	if len(tableHash.Indexes) != 1 {
		t.Errorf("expected 1 index hash, got %d", len(tableHash.Indexes))
	}
}

func TestComputeSchemaHash_Deterministic(t *testing.T) {
	// Create same schema twice
	makeSchema := func() *engine.Schema {
		schema := engine.NewSchema()
		schema.Tables["auth.user"] = &ast.TableDef{
			Namespace: "auth",
			Name:      "user",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", TypeArgs: []any{255}},
			},
		}
		schema.Tables["core.post"] = &ast.TableDef{
			Namespace: "core",
			Name:      "post",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "title", Type: "string", TypeArgs: []any{200}},
			},
		}
		return schema
	}

	hash1, err := ComputeSchemaHash(makeSchema())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash2, err := ComputeSchemaHash(makeSchema())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1.Root != hash2.Root {
		t.Errorf("hashes should be deterministic: %s != %s", hash1.Root, hash2.Root)
	}
}

func TestComputeSchemaHash_DifferentSchemas(t *testing.T) {
	schema1 := engine.NewSchema()
	schema1.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	schema2 := engine.NewSchema()
	schema2.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{100}}, // Different length
		},
	}

	hash1, err := ComputeSchemaHash(schema1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash2, err := ComputeSchemaHash(schema2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1.Root == hash2.Root {
		t.Error("different schemas should have different hashes")
	}

	// Column hashes should also differ
	if hash1.Tables["auth.user"].Columns["email"] == hash2.Tables["auth.user"].Columns["email"] {
		t.Error("column hashes should differ for different type args")
	}
}

func TestCompareHashes_Match(t *testing.T) {
	schema := engine.NewSchema()
	schema.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
		},
	}

	hash1, _ := ComputeSchemaHash(schema)
	hash2, _ := ComputeSchemaHash(schema)

	comparison := CompareHashes(hash1, hash2)

	if !comparison.Match {
		t.Error("identical schemas should match")
	}

	if len(comparison.MissingTables) != 0 {
		t.Error("no missing tables expected")
	}

	if len(comparison.ExtraTables) != 0 {
		t.Error("no extra tables expected")
	}

	if len(comparison.TableDiffs) != 0 {
		t.Error("no table diffs expected")
	}
}

func TestCompareHashes_MissingTable(t *testing.T) {
	expected := engine.NewSchema()
	expected.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
	}
	expected.Tables["core.post"] = &ast.TableDef{
		Namespace: "core",
		Name:      "post",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
	}

	actual := engine.NewSchema()
	actual.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
	}

	expectedHash, _ := ComputeSchemaHash(expected)
	actualHash, _ := ComputeSchemaHash(actual)

	comparison := CompareHashes(expectedHash, actualHash)

	if comparison.Match {
		t.Error("schemas should not match")
	}

	if len(comparison.MissingTables) != 1 {
		t.Errorf("expected 1 missing table, got %d", len(comparison.MissingTables))
	}

	if comparison.MissingTables[0] != "core.post" {
		t.Errorf("expected core.post missing, got %s", comparison.MissingTables[0])
	}
}

func TestCompareHashes_ExtraTable(t *testing.T) {
	expected := engine.NewSchema()
	expected.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
	}

	actual := engine.NewSchema()
	actual.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
	}
	actual.Tables["temp.debug"] = &ast.TableDef{
		Namespace: "temp",
		Name:      "debug",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "uuid", PrimaryKey: true}},
	}

	expectedHash, _ := ComputeSchemaHash(expected)
	actualHash, _ := ComputeSchemaHash(actual)

	comparison := CompareHashes(expectedHash, actualHash)

	if comparison.Match {
		t.Error("schemas should not match")
	}

	if len(comparison.ExtraTables) != 1 {
		t.Errorf("expected 1 extra table, got %d", len(comparison.ExtraTables))
	}

	if comparison.ExtraTables[0] != "temp.debug" {
		t.Errorf("expected temp.debug extra, got %s", comparison.ExtraTables[0])
	}
}

func TestCompareHashes_ModifiedColumn(t *testing.T) {
	expected := engine.NewSchema()
	expected.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	actual := engine.NewSchema()
	actual.Tables["auth.user"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{100}}, // Different!
		},
	}

	expectedHash, _ := ComputeSchemaHash(expected)
	actualHash, _ := ComputeSchemaHash(actual)

	comparison := CompareHashes(expectedHash, actualHash)

	if comparison.Match {
		t.Error("schemas should not match")
	}

	diff, ok := comparison.TableDiffs["auth.user"]
	if !ok {
		t.Fatal("expected diff for auth.user")
	}

	if len(diff.ModifiedColumns) != 1 {
		t.Errorf("expected 1 modified column, got %d", len(diff.ModifiedColumns))
	}

	if diff.ModifiedColumns[0] != "email" {
		t.Errorf("expected email modified, got %s", diff.ModifiedColumns[0])
	}
}

func TestTableDiff_HasDifferences(t *testing.T) {
	tests := []struct {
		name     string
		diff     TableDiff
		expected bool
	}{
		{
			name:     "empty diff",
			diff:     TableDiff{},
			expected: false,
		},
		{
			name:     "missing column",
			diff:     TableDiff{MissingColumns: []string{"foo"}},
			expected: true,
		},
		{
			name:     "extra index",
			diff:     TableDiff{ExtraIndexes: []string{"idx"}},
			expected: true,
		},
		{
			name:     "modified fk",
			diff:     TableDiff{ModifiedFKs: []string{"fk"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.HasDifferences(); got != tt.expected {
				t.Errorf("HasDifferences() = %v, want %v", got, tt.expected)
			}
		})
	}
}
