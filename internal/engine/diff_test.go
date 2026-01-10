package engine

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// Diff Tests
// -----------------------------------------------------------------------------

func TestDiffEmptySchemas(t *testing.T) {
	tests := []struct {
		name string
		old  *Schema
		new  *Schema
	}{
		{"both_nil", nil, nil},
		{"old_nil", nil, NewSchema()},
		{"new_nil", NewSchema(), nil},
		{"both_empty", NewSchema(), NewSchema()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, err := Diff(tt.old, tt.new)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}
			if len(ops) != 0 {
				t.Errorf("Diff() = %d operations, want 0", len(ops))
			}
		})
	}
}

func TestDiffTableCreation(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	// Add a new table to the new schema
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("Diff() = %d operations, want 1", len(ops))
	}

	createOp, ok := ops[0].(*ast.CreateTable)
	if !ok {
		t.Fatalf("Diff() op[0] type = %T, want *ast.CreateTable", ops[0])
	}

	if createOp.Table() != "auth_users" {
		t.Errorf("CreateTable.Table() = %q, want %q", createOp.Table(), "auth_users")
	}
}

func TestDiffMultipleTableCreation(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	// Add multiple new tables
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}
	new.Tables["blog.posts"] = &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	// Should have 2 CreateTable operations
	createCount := 0
	for _, op := range ops {
		if op.Type() == ast.OpCreateTable {
			createCount++
		}
	}

	if createCount != 2 {
		t.Errorf("Diff() CreateTable count = %d, want 2", createCount)
	}
}

func TestDiffTableDrop(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	// Old schema has a table, new schema doesn't
	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("Diff() = %d operations, want 1", len(ops))
	}

	dropOp, ok := ops[0].(*ast.DropTable)
	if !ok {
		t.Fatalf("Diff() op[0] type = %T, want *ast.DropTable", ops[0])
	}

	if dropOp.Table() != "auth_users" {
		t.Errorf("DropTable.Table() = %q, want %q", dropOp.Table(), "auth_users")
	}
}

func TestDiffColumnAdd(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	// Both have the same table, but new has an additional column
	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
		},
	}
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	// Should have 1 AddColumn operation
	var addOp *ast.AddColumn
	for _, op := range ops {
		if add, ok := op.(*ast.AddColumn); ok {
			addOp = add
		}
	}

	if addOp == nil {
		t.Fatal("Diff() missing AddColumn operation")
	}

	if addOp.Column.Name != "email" {
		t.Errorf("AddColumn.Column.Name = %q, want %q", addOp.Column.Name, "email")
	}
}

func TestDiffColumnDrop(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	// Old has more columns than new
	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
			{Name: "phone", Type: "string", TypeArgs: []any{20}},
		},
	}
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	// Should have 1 DropColumn operation
	var dropOp *ast.DropColumn
	for _, op := range ops {
		if drop, ok := op.(*ast.DropColumn); ok {
			dropOp = drop
		}
	}

	if dropOp == nil {
		t.Fatal("Diff() missing DropColumn operation")
	}

	if dropOp.Name != "phone" {
		t.Errorf("DropColumn.Name = %q, want %q", dropOp.Name, "phone")
	}
}

func TestDiffColumnAlter(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	// Column type changes
	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
		},
	}
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{255}}, // Changed length
		},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	// Should have 1 AlterColumn operation
	var alterOp *ast.AlterColumn
	for _, op := range ops {
		if alter, ok := op.(*ast.AlterColumn); ok {
			alterOp = alter
		}
	}

	if alterOp == nil {
		t.Fatal("Diff() missing AlterColumn operation")
	}

	if alterOp.Name != "name" {
		t.Errorf("AlterColumn.Name = %q, want %q", alterOp.Name, "name")
	}
}

func TestDiffColumnNullabilityChange(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "phone", Type: "string", TypeArgs: []any{20}, Nullable: false},
		},
	}
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "phone", Type: "string", TypeArgs: []any{20}, Nullable: true}, // Now nullable
		},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	var alterOp *ast.AlterColumn
	for _, op := range ops {
		if alter, ok := op.(*ast.AlterColumn); ok {
			alterOp = alter
		}
	}

	if alterOp == nil {
		t.Fatal("Diff() missing AlterColumn operation for nullability change")
	}

	if alterOp.SetNullable == nil || *alterOp.SetNullable != true {
		t.Error("AlterColumn.SetNullable should be true")
	}
}

func TestDiffIndexAdd(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		Indexes:   []*ast.IndexDef{},
	}
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		Indexes: []*ast.IndexDef{
			{Name: "idx_users_email", Columns: []string{"email"}},
		},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	var createIndexOp *ast.CreateIndex
	for _, op := range ops {
		if ci, ok := op.(*ast.CreateIndex); ok {
			createIndexOp = ci
		}
	}

	if createIndexOp == nil {
		t.Fatal("Diff() missing CreateIndex operation")
	}
}

func TestDiffIndexDrop(t *testing.T) {
	old := NewSchema()
	new := NewSchema()

	old.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		Indexes: []*ast.IndexDef{
			{Name: "idx_users_email", Columns: []string{"email"}},
		},
	}
	new.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		Indexes:   []*ast.IndexDef{},
	}

	ops, err := Diff(old, new)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	var dropIndexOp *ast.DropIndex
	for _, op := range ops {
		if di, ok := op.(*ast.DropIndex); ok {
			dropIndexOp = di
		}
	}

	if dropIndexOp == nil {
		t.Fatal("Diff() missing DropIndex operation")
	}
}

// -----------------------------------------------------------------------------
// sortByDependency Tests
// -----------------------------------------------------------------------------

func TestSortByDependencyEmpty(t *testing.T) {
	ops := []ast.Operation{}
	schema := NewSchema()

	sorted := sortByDependency(ops, schema)

	if len(sorted) != 0 {
		t.Errorf("sortByDependency() = %d operations, want 0", len(sorted))
	}
}

func TestSortByDependencyCreateTablesWithFK(t *testing.T) {
	schema := NewSchema()

	// posts depends on users (posts.user_id -> users.id)
	schema.Tables["auth.users"] = &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns:   []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
	}
	schema.Tables["blog.posts"] = &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{Name: "user_id", Type: "uuid", Reference: &ast.Reference{Table: "auth_users", Column: "id"}},
		},
	}

	// Create operations in wrong order (dependent first)
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "posts"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id", PrimaryKey: true},
				{Name: "user_id", Type: "uuid", Reference: &ast.Reference{Table: "auth_users", Column: "id"}},
			},
		},
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
			Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		},
	}

	sorted := sortByDependency(ops, schema)

	// Users should come before posts
	var usersIdx, postsIdx int
	for i, op := range sorted {
		if op.Table() == "auth_users" {
			usersIdx = i
		}
		if op.Table() == "blog_posts" {
			postsIdx = i
		}
	}

	if usersIdx >= postsIdx {
		t.Errorf("sortByDependency(): users (idx=%d) should come before posts (idx=%d)", usersIdx, postsIdx)
	}
}

func TestSortByDependencyOperationOrder(t *testing.T) {
	schema := NewSchema()

	// Mix of different operation types
	ops := []ast.Operation{
		&ast.CreateIndex{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Columns: []string{"email"}},
		&ast.DropIndex{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "old_index"},
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "old_table"}, Name: "col"},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Column: &ast.ColumnDef{Name: "new_col", Type: "string"}},
		&ast.DropTable{TableOp: ast.TableOp{Namespace: "old", Name: "table"}},
	}

	sorted := sortByDependency(ops, schema)

	// Check operation type order:
	// 1. DropForeignKey (not in our test)
	// 2. DropIndex
	// 3. DropColumn
	// 4. DropTable
	// 5. CreateTable
	// 6. AddColumn
	// 7. AlterColumn (not in our test)
	// 8. CreateIndex
	// 9. AddForeignKey (not in our test)

	findIndex := func(opType ast.OpType) int {
		for i, op := range sorted {
			if op.Type() == opType {
				return i
			}
		}
		return -1
	}

	dropIdxPos := findIndex(ast.OpDropIndex)
	dropColPos := findIndex(ast.OpDropColumn)
	dropTablePos := findIndex(ast.OpDropTable)
	createTablePos := findIndex(ast.OpCreateTable)
	addColPos := findIndex(ast.OpAddColumn)
	createIdxPos := findIndex(ast.OpCreateIndex)

	// Verify ordering
	if dropIdxPos >= dropColPos {
		t.Error("DropIndex should come before DropColumn")
	}
	if dropColPos >= dropTablePos {
		t.Error("DropColumn should come before DropTable")
	}
	if dropTablePos >= createTablePos {
		t.Error("DropTable should come before CreateTable")
	}
	if createTablePos >= addColPos {
		t.Error("CreateTable should come before AddColumn")
	}
	if addColPos >= createIdxPos {
		t.Error("AddColumn should come before CreateIndex")
	}
}

// -----------------------------------------------------------------------------
// DiffSummary Tests
// -----------------------------------------------------------------------------

func TestSummarize(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "blog", Name: "posts"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
		&ast.DropTable{TableOp: ast.TableOp{Namespace: "old", Name: "table"}},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Column: &ast.ColumnDef{Name: "email", Type: "string"}},
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "old_col"},
		&ast.CreateIndex{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Columns: []string{"email"}},
	}

	summary := Summarize(ops)

	if summary.TablesToCreate != 2 {
		t.Errorf("TablesToCreate = %d, want 2", summary.TablesToCreate)
	}
	if summary.TablesToDrop != 1 {
		t.Errorf("TablesToDrop = %d, want 1", summary.TablesToDrop)
	}
	if summary.ColumnsToAdd != 1 {
		t.Errorf("ColumnsToAdd = %d, want 1", summary.ColumnsToAdd)
	}
	if summary.ColumnsToDrop != 1 {
		t.Errorf("ColumnsToDrop = %d, want 1", summary.ColumnsToDrop)
	}
	if summary.IndexesToAdd != 1 {
		t.Errorf("IndexesToAdd = %d, want 1", summary.IndexesToAdd)
	}
	if summary.TotalOps != 6 {
		t.Errorf("TotalOps = %d, want 6", summary.TotalOps)
	}
}

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name string
		ops  []ast.Operation
		want bool
	}{
		{"empty", []ast.Operation{}, false},
		{"one_op", []ast.Operation{&ast.CreateTable{TableOp: ast.TableOp{Namespace: "a", Name: "b"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasChanges(tt.ops); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// parseSimpleRef Tests
// -----------------------------------------------------------------------------

func TestParseSimpleRef(t *testing.T) {
	tests := []struct {
		input     string
		wantNS    string
		wantTable string
		wantRel   bool
	}{
		{"users", "", "users", false},
		{"auth.users", "auth", "users", false},
		{".users", "", "users", true},
		{"blog.posts", "blog", "posts", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ns, table, isRel := parseSimpleRef(tt.input)
			if ns != tt.wantNS {
				t.Errorf("parseSimpleRef(%q) ns = %q, want %q", tt.input, ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("parseSimpleRef(%q) table = %q, want %q", tt.input, table, tt.wantTable)
			}
			if isRel != tt.wantRel {
				t.Errorf("parseSimpleRef(%q) isRelative = %v, want %v", tt.input, isRel, tt.wantRel)
			}
		})
	}
}
