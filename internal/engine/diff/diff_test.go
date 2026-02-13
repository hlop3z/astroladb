package diff

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// -----------------------------------------------------------------------------
// Diff Tests
// -----------------------------------------------------------------------------

func TestDiffEmptySchemas(t *testing.T) {
	tests := []struct {
		name string
		old  *engine.Schema
		new  *engine.Schema
	}{
		{"both_nil", nil, nil},
		{"old_nil", nil, engine.NewSchema()},
		{"new_nil", engine.NewSchema(), nil},
		{"both_empty", engine.NewSchema(), engine.NewSchema()},
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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	old := engine.NewSchema()
	new := engine.NewSchema()

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
	schema := engine.NewSchema()

	sorted := sortByDependency(ops, schema)

	if len(sorted) != 0 {
		t.Errorf("sortByDependency() = %d operations, want 0", len(sorted))
	}
}

func TestSortByDependencyCreateTablesWithFK(t *testing.T) {
	schema := engine.NewSchema()

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
	schema := engine.NewSchema()

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
// engine.ParseSimpleRef Tests
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
			ns, table, isRel := engine.ParseSimpleRef(tt.input)
			if ns != tt.wantNS {
				t.Errorf("engine.ParseSimpleRef(%q) ns = %q, want %q", tt.input, ns, tt.wantNS)
			}
			if table != tt.wantTable {
				t.Errorf("engine.ParseSimpleRef(%q) table = %q, want %q", tt.input, table, tt.wantTable)
			}
			if isRel != tt.wantRel {
				t.Errorf("engine.ParseSimpleRef(%q) isRelative = %v, want %v", tt.input, isRel, tt.wantRel)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// defaultsEqual Tests
// -----------------------------------------------------------------------------

func TestDefaultsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
		want bool
	}{
		{"both_nil", nil, nil, true},
		{"a_nil", nil, "default", false},
		{"b_nil", "default", nil, false},
		{"equal_strings", "default", "default", true},
		{"different_strings", "a", "b", false},
		{"equal_ints", 0, 0, true},
		{"different_ints", 1, 2, false},
		{"equal_bools", true, true, true},
		{"different_bools", true, false, false},
		{"equal_floats", 1.5, 1.5, true},
		{"different_floats", 1.5, 2.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("defaultsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// checkKey and sanitizeCheckName Tests
// -----------------------------------------------------------------------------

func TestCheckKey(t *testing.T) {
	check := &ast.CheckDef{
		Name:       "chk_positive",
		Expression: "amount > 0",
	}

	key := checkKey(check)
	if key != "amount > 0" {
		t.Errorf("checkKey() = %q, want %q", key, "amount > 0")
	}
}

func TestSanitizeCheckName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"amount > 0", "amount_0"},
		{"status IN ('a', 'b')", "status_IN_a_b"}, // Trailing underscore is trimmed
		{"age >= 18", "age_18"},
		{"simple", "simple"},
		{"", "constraint"}, // Empty returns "constraint"
		{"a_very_long_expression_that_exceeds_twenty_characters", "a_very_long_expressi"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeCheckName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeCheckName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// diffCheckConstraints Tests
// -----------------------------------------------------------------------------

func TestDiffCheckConstraints(t *testing.T) {
	t.Run("add_check", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks:    []*ast.CheckDef{},
		}
		newTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks: []*ast.CheckDef{
				{Name: "chk_age", Expression: "age >= 0"},
			},
		}

		ops := diffCheckConstraints(oldTable, newTable)
		if len(ops) != 1 {
			t.Fatalf("diffCheckConstraints() = %d ops, want 1", len(ops))
		}

		addOp, ok := ops[0].(*ast.AddCheck)
		if !ok {
			t.Fatalf("ops[0] type = %T, want *ast.AddCheck", ops[0])
		}
		if addOp.Name != "chk_age" {
			t.Errorf("AddCheck.Name = %q, want %q", addOp.Name, "chk_age")
		}
	})

	t.Run("drop_check", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks: []*ast.CheckDef{
				{Name: "chk_old", Expression: "old_field > 0"},
			},
		}
		newTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks:    []*ast.CheckDef{},
		}

		ops := diffCheckConstraints(oldTable, newTable)
		if len(ops) != 1 {
			t.Fatalf("diffCheckConstraints() = %d ops, want 1", len(ops))
		}

		dropOp, ok := ops[0].(*ast.DropCheck)
		if !ok {
			t.Fatalf("ops[0] type = %T, want *ast.DropCheck", ops[0])
		}
		if dropOp.Name != "chk_old" {
			t.Errorf("DropCheck.Name = %q, want %q", dropOp.Name, "chk_old")
		}
	})

	t.Run("no_change", func(t *testing.T) {
		table := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks: []*ast.CheckDef{
				{Name: "chk_age", Expression: "age >= 0"},
			},
		}

		ops := diffCheckConstraints(table, table)
		if len(ops) != 0 {
			t.Errorf("diffCheckConstraints() = %d ops, want 0 (no change)", len(ops))
		}
	})

	t.Run("auto_generate_name", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks:    []*ast.CheckDef{},
		}
		newTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Checks: []*ast.CheckDef{
				{Name: "", Expression: "amount > 0"}, // No name provided
			},
		}

		ops := diffCheckConstraints(oldTable, newTable)
		if len(ops) != 1 {
			t.Fatalf("diffCheckConstraints() = %d ops, want 1", len(ops))
		}

		addOp := ops[0].(*ast.AddCheck)
		// Name should be auto-generated
		if addOp.Name == "" {
			t.Error("AddCheck.Name should be auto-generated, got empty")
		}
	})
}

// -----------------------------------------------------------------------------
// diffForeignKeys Tests
// -----------------------------------------------------------------------------

func TestDiffForeignKeys(t *testing.T) {
	// diffForeignKeys extracts FKs from column.Reference, not from ForeignKeys slice
	t.Run("add_foreign_key_via_column_reference", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid"}, // No reference
			},
		}
		newTable := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth_users", Column: "id"}},
			},
		}

		ops := diffForeignKeys(oldTable, newTable)
		addCount := 0
		for _, op := range ops {
			if _, ok := op.(*ast.AddForeignKey); ok {
				addCount++
			}
		}
		if addCount != 1 {
			t.Errorf("diffForeignKeys() add count = %d, want 1", addCount)
		}
	})

	t.Run("drop_foreign_key_via_column_reference", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "old_table", Column: "id"}},
			},
		}
		newTable := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid"}, // No reference anymore
			},
		}

		ops := diffForeignKeys(oldTable, newTable)
		dropCount := 0
		for _, op := range ops {
			if _, ok := op.(*ast.DropForeignKey); ok {
				dropCount++
			}
		}
		if dropCount != 1 {
			t.Errorf("diffForeignKeys() drop count = %d, want 1", dropCount)
		}
	})

	t.Run("no_change", func(t *testing.T) {
		table := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth_users", Column: "id"}},
			},
		}

		ops := diffForeignKeys(table, table)
		if len(ops) != 0 {
			t.Errorf("diffForeignKeys() = %d ops, want 0 (no change)", len(ops))
		}
	})
}

// -----------------------------------------------------------------------------
// diffIndexes Tests
// -----------------------------------------------------------------------------

func TestDiffIndexes(t *testing.T) {
	t.Run("add_index", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id"}, {Name: "email", Type: "string"}},
			Indexes:   []*ast.IndexDef{},
		}
		newTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id"}, {Name: "email", Type: "string"}},
			Indexes: []*ast.IndexDef{
				{Name: "idx_email", Columns: []string{"email"}, Unique: true},
			},
		}

		ops := diffIndexes(oldTable, newTable)
		if len(ops) != 1 {
			t.Fatalf("diffIndexes() = %d ops, want 1", len(ops))
		}

		createOp, ok := ops[0].(*ast.CreateIndex)
		if !ok {
			t.Fatalf("ops[0] type = %T, want *ast.CreateIndex", ops[0])
		}
		if createOp.Name != "idx_email" {
			t.Errorf("CreateIndex.Name = %q, want %q", createOp.Name, "idx_email")
		}
	})

	t.Run("drop_index", func(t *testing.T) {
		oldTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id"}, {Name: "email", Type: "string"}},
			Indexes: []*ast.IndexDef{
				{Name: "idx_old", Columns: []string{"email"}, Unique: false},
			},
		}
		newTable := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id"}, {Name: "email", Type: "string"}},
			Indexes:   []*ast.IndexDef{},
		}

		ops := diffIndexes(oldTable, newTable)
		if len(ops) != 1 {
			t.Fatalf("diffIndexes() = %d ops, want 1", len(ops))
		}

		dropOp, ok := ops[0].(*ast.DropIndex)
		if !ok {
			t.Fatalf("ops[0] type = %T, want *ast.DropIndex", ops[0])
		}
		if dropOp.Name != "idx_old" {
			t.Errorf("DropIndex.Name = %q, want %q", dropOp.Name, "idx_old")
		}
	})
}

// -----------------------------------------------------------------------------
// Summarize additional tests
// -----------------------------------------------------------------------------

func TestSummarizeAllOperationTypes(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "a", Name: "b"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id"}}},
		&ast.DropTable{TableOp: ast.TableOp{Namespace: "a", Name: "c"}},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Column: &ast.ColumnDef{Name: "x", Type: "string"}},
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "y"},
		&ast.AlterColumn{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "z", NewType: "text"},
		&ast.CreateIndex{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Columns: []string{"x"}},
		&ast.DropIndex{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "idx"},
		&ast.AddForeignKey{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "fk", Columns: []string{"x"}, RefTable: "c", RefColumns: []string{"id"}},
		&ast.DropForeignKey{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "fk2"},
		&ast.AddCheck{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "chk", Expression: "x > 0"},
		&ast.DropCheck{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "chk_old"},
	}

	summary := Summarize(ops)

	if summary.TablesToCreate != 1 {
		t.Errorf("TablesToCreate = %d, want 1", summary.TablesToCreate)
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
	if summary.ColumnsToAlter != 1 {
		t.Errorf("ColumnsToAlter = %d, want 1", summary.ColumnsToAlter)
	}
	if summary.IndexesToAdd != 1 {
		t.Errorf("IndexesToAdd = %d, want 1", summary.IndexesToAdd)
	}
	if summary.IndexesToDrop != 1 {
		t.Errorf("IndexesToDrop = %d, want 1", summary.IndexesToDrop)
	}
	if summary.FKsToAdd != 1 {
		t.Errorf("FKsToAdd = %d, want 1", summary.FKsToAdd)
	}
	if summary.FKsToDrop != 1 {
		t.Errorf("FKsToDrop = %d, want 1", summary.FKsToDrop)
	}
	if summary.ChecksToAdd != 1 {
		t.Errorf("ChecksToAdd = %d, want 1", summary.ChecksToAdd)
	}
	if summary.ChecksToDrop != 1 {
		t.Errorf("ChecksToDrop = %d, want 1", summary.ChecksToDrop)
	}
	if summary.TotalOps != 11 {
		t.Errorf("TotalOps = %d, want 11", summary.TotalOps)
	}
}

// -----------------------------------------------------------------------------
// diffColumnDef Tests
// -----------------------------------------------------------------------------

func TestDiffColumnDefTypeChange(t *testing.T) {
	table := &ast.TableDef{Namespace: "auth", Name: "users"}
	oldCol := &ast.ColumnDef{Name: "bio", Type: "string", TypeArgs: []any{100}}
	newCol := &ast.ColumnDef{Name: "bio", Type: "text"}

	ops := diffColumnDef(table, oldCol, newCol)

	if len(ops) != 1 {
		t.Fatalf("diffColumnDef() = %d ops, want 1", len(ops))
	}

	alterOp, ok := ops[0].(*ast.AlterColumn)
	if !ok {
		t.Fatalf("ops[0] type = %T, want *ast.AlterColumn", ops[0])
	}

	if alterOp.NewType != "text" {
		t.Errorf("AlterColumn.NewType = %q, want %q", alterOp.NewType, "text")
	}
}

func TestDiffColumnDefTypeArgsChange(t *testing.T) {
	table := &ast.TableDef{Namespace: "auth", Name: "users"}
	oldCol := &ast.ColumnDef{Name: "name", Type: "string", TypeArgs: []any{100}}
	newCol := &ast.ColumnDef{Name: "name", Type: "string", TypeArgs: []any{255}}

	ops := diffColumnDef(table, oldCol, newCol)

	if len(ops) != 1 {
		t.Fatalf("diffColumnDef() = %d ops, want 1", len(ops))
	}

	alterOp := ops[0].(*ast.AlterColumn)
	if alterOp.NewTypeArgs == nil || alterOp.NewTypeArgs[0] != 255 {
		t.Errorf("AlterColumn.NewTypeArgs = %v, want [255]", alterOp.NewTypeArgs)
	}
}

func TestDiffColumnDefDefaultChange(t *testing.T) {
	t.Run("add_default", func(t *testing.T) {
		table := &ast.TableDef{Namespace: "auth", Name: "users"}
		oldCol := &ast.ColumnDef{Name: "status", Type: "string", Default: nil}
		newCol := &ast.ColumnDef{Name: "status", Type: "string", Default: "active", DefaultSet: true}

		ops := diffColumnDef(table, oldCol, newCol)

		if len(ops) != 1 {
			t.Fatalf("diffColumnDef() = %d ops, want 1", len(ops))
		}

		alterOp := ops[0].(*ast.AlterColumn)
		if alterOp.SetDefault != "active" {
			t.Errorf("AlterColumn.SetDefault = %v, want %q", alterOp.SetDefault, "active")
		}
	})

	t.Run("change_default", func(t *testing.T) {
		table := &ast.TableDef{Namespace: "auth", Name: "users"}
		oldCol := &ast.ColumnDef{Name: "status", Type: "string", Default: "pending", DefaultSet: true}
		newCol := &ast.ColumnDef{Name: "status", Type: "string", Default: "active", DefaultSet: true}

		ops := diffColumnDef(table, oldCol, newCol)

		if len(ops) != 1 {
			t.Fatalf("diffColumnDef() = %d ops, want 1", len(ops))
		}

		alterOp := ops[0].(*ast.AlterColumn)
		if alterOp.SetDefault != "active" {
			t.Errorf("AlterColumn.SetDefault = %v, want %q", alterOp.SetDefault, "active")
		}
	})

	t.Run("drop_default", func(t *testing.T) {
		table := &ast.TableDef{Namespace: "auth", Name: "users"}
		oldCol := &ast.ColumnDef{Name: "status", Type: "string", Default: "active", DefaultSet: true}
		newCol := &ast.ColumnDef{Name: "status", Type: "string", Default: nil, DefaultSet: false}

		ops := diffColumnDef(table, oldCol, newCol)

		if len(ops) != 1 {
			t.Fatalf("diffColumnDef() = %d ops, want 1", len(ops))
		}

		alterOp := ops[0].(*ast.AlterColumn)
		if !alterOp.DropDefault {
			t.Error("AlterColumn.DropDefault should be true")
		}
	})
}

func TestDiffColumnDefNoChange(t *testing.T) {
	table := &ast.TableDef{Namespace: "auth", Name: "users"}
	col := &ast.ColumnDef{Name: "email", Type: "string", TypeArgs: []any{255}, Nullable: false}

	ops := diffColumnDef(table, col, col)

	if len(ops) != 0 {
		t.Errorf("diffColumnDef() = %d ops, want 0 (no change)", len(ops))
	}
}

func TestDiffColumnDefMultipleChanges(t *testing.T) {
	table := &ast.TableDef{Namespace: "auth", Name: "users"}
	oldCol := &ast.ColumnDef{Name: "bio", Type: "string", TypeArgs: []any{100}, Nullable: false, Default: nil}
	newCol := &ast.ColumnDef{Name: "bio", Type: "text", Nullable: true, Default: "Hello", DefaultSet: true}

	ops := diffColumnDef(table, oldCol, newCol)

	if len(ops) != 1 {
		t.Fatalf("diffColumnDef() = %d ops, want 1 (single AlterColumn with multiple changes)", len(ops))
	}

	alterOp := ops[0].(*ast.AlterColumn)
	if alterOp.NewType != "text" {
		t.Errorf("AlterColumn.NewType = %q, want %q", alterOp.NewType, "text")
	}
	if alterOp.SetNullable == nil || *alterOp.SetNullable != true {
		t.Error("AlterColumn.SetNullable should be true")
	}
	if alterOp.SetDefault != "Hello" {
		t.Errorf("AlterColumn.SetDefault = %v, want %q", alterOp.SetDefault, "Hello")
	}
}

// -----------------------------------------------------------------------------
// diffColumns Tests
// -----------------------------------------------------------------------------

func TestDiffColumnsAddAndDrop(t *testing.T) {
	oldTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "email", Type: "string"},
			{Name: "old_column", Type: "string"},
		},
	}
	newTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "email", Type: "string"},
			{Name: "new_column", Type: "string"},
		},
	}

	ops := diffColumns(oldTable, newTable)

	addCount, dropCount := 0, 0
	for _, op := range ops {
		switch op.(type) {
		case *ast.AddColumn:
			addCount++
		case *ast.DropColumn:
			dropCount++
		}
	}

	if addCount != 1 {
		t.Errorf("diffColumns() AddColumn count = %d, want 1", addCount)
	}
	if dropCount != 1 {
		t.Errorf("diffColumns() DropColumn count = %d, want 1", dropCount)
	}
}

func TestDiffColumnsNoChange(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "email", Type: "string", TypeArgs: []any{255}},
		},
	}

	ops := diffColumns(table, table)

	if len(ops) != 0 {
		t.Errorf("diffColumns() = %d ops, want 0 (no change)", len(ops))
	}
}

// -----------------------------------------------------------------------------
// sortByDependency with AddCheck/DropCheck Tests
// -----------------------------------------------------------------------------

func TestSortByDependencyWithChecks(t *testing.T) {
	schema := engine.NewSchema()

	ops := []ast.Operation{
		&ast.AddCheck{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "chk_age", Expression: "age > 0"},
		&ast.DropCheck{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "chk_old"},
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id"}}},
	}

	sorted := sortByDependency(ops, schema)

	// Find positions
	var dropCheckPos, createTablePos, addCheckPos int
	for i, op := range sorted {
		switch op.Type() {
		case ast.OpDropCheck:
			dropCheckPos = i
		case ast.OpCreateTable:
			createTablePos = i
		case ast.OpAddCheck:
			addCheckPos = i
		}
	}

	// DropCheck should come before CreateTable
	if dropCheckPos >= createTablePos {
		t.Error("DropCheck should come before CreateTable")
	}
	// CreateTable should come before AddCheck
	if createTablePos >= addCheckPos {
		t.Error("CreateTable should come before AddCheck")
	}
}

func TestSortByDependencyWithRawSQL(t *testing.T) {
	schema := engine.NewSchema()

	ops := []ast.Operation{
		&ast.RawSQL{SQL: "SELECT 1"},
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id"}}},
	}

	sorted := sortByDependency(ops, schema)

	// RawSQL should be at the end
	if len(sorted) != 2 {
		t.Fatalf("sorted = %d ops, want 2", len(sorted))
	}
	if _, ok := sorted[1].(*ast.RawSQL); !ok {
		t.Error("RawSQL should be at the end")
	}
}

// -----------------------------------------------------------------------------
// indexKey Tests
// -----------------------------------------------------------------------------

func TestIndexKey(t *testing.T) {
	table := &ast.TableDef{Namespace: "auth", Name: "users"}

	t.Run("single_column", func(t *testing.T) {
		idx := &ast.IndexDef{Name: "idx_email", Columns: []string{"email"}, Unique: false}
		key := indexKey(table, idx)
		if key != "auth.users:email" {
			t.Errorf("indexKey() = %q, want %q", key, "auth.users:email")
		}
	})

	t.Run("unique_index", func(t *testing.T) {
		idx := &ast.IndexDef{Name: "idx_email", Columns: []string{"email"}, Unique: true}
		key := indexKey(table, idx)
		if key != "auth.users:unique:email" {
			t.Errorf("indexKey() = %q, want %q", key, "auth.users:unique:email")
		}
	})

	t.Run("multi_column", func(t *testing.T) {
		idx := &ast.IndexDef{Name: "idx_name", Columns: []string{"first", "last"}, Unique: false}
		key := indexKey(table, idx)
		if key != "auth.users:first,last" {
			t.Errorf("indexKey() = %q, want %q", key, "auth.users:first,last")
		}
	})
}

// -----------------------------------------------------------------------------
// tableColumnsToFKDefs Tests
// -----------------------------------------------------------------------------

func TestTableColumnsToFKDefs(t *testing.T) {
	t.Run("no_fks", func(t *testing.T) {
		table := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "name", Type: "string"},
			},
		}

		fks := tableColumnsToFKDefs(table)
		if len(fks) != 0 {
			t.Errorf("tableColumnsToFKDefs() = %d FKs, want 0", len(fks))
		}
	})

	t.Run("with_fk", func(t *testing.T) {
		table := &ast.TableDef{
			Namespace: "blog",
			Name:      "posts",
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{
					Table:    "auth.users",
					Column:   "id",
					OnDelete: "CASCADE",
					OnUpdate: "NO ACTION",
				}},
			},
		}

		fks := tableColumnsToFKDefs(table)
		if len(fks) != 1 {
			t.Fatalf("tableColumnsToFKDefs() = %d FKs, want 1", len(fks))
		}

		fk := fks[0]
		if fk.RefTable != "auth.users" {
			t.Errorf("FK.RefTable = %q, want %q", fk.RefTable, "auth.users")
		}
		if fk.OnDelete != "CASCADE" {
			t.Errorf("FK.OnDelete = %q, want %q", fk.OnDelete, "CASCADE")
		}
	})
}

// -----------------------------------------------------------------------------
// tableIndexesToIndexDefs Tests
// -----------------------------------------------------------------------------

func TestTableIndexesToIndexDefs(t *testing.T) {
	t.Run("nil_indexes", func(t *testing.T) {
		table := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Indexes:   nil,
		}

		result := tableIndexesToIndexDefs(table)
		if result != nil {
			t.Errorf("tableIndexesToIndexDefs() = %v, want nil", result)
		}
	})

	t.Run("with_indexes", func(t *testing.T) {
		table := &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Indexes: []*ast.IndexDef{
				{Name: "idx_email", Columns: []string{"email"}, Unique: true},
			},
		}

		result := tableIndexesToIndexDefs(table)
		if len(result) != 1 {
			t.Errorf("tableIndexesToIndexDefs() = %d indexes, want 1", len(result))
		}
	})
}

// -----------------------------------------------------------------------------
// extractFKsFromColumns Tests
// -----------------------------------------------------------------------------

func TestExtractFKsFromColumns(t *testing.T) {
	table := &ast.TableDef{
		Namespace: "blog",
		Name:      "posts",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
			{Name: "editor_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
			{Name: "category_id", Type: "uuid", Reference: &ast.Reference{Table: "blog.categories", Column: "id"}},
		},
	}

	fks := extractFKsFromColumns(table)

	if len(fks) != 3 {
		t.Errorf("extractFKsFromColumns() = %d FKs, want 3", len(fks))
	}
}

// -----------------------------------------------------------------------------
// getCreateTableDependencies Tests
// -----------------------------------------------------------------------------

func TestGetCreateTableDependencies(t *testing.T) {
	schema := engine.NewSchema()

	t.Run("no_dependencies", func(t *testing.T) {
		op := &ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "name", Type: "string"},
			},
		}

		deps := getCreateTableDependencies(op, schema)
		if len(deps) != 0 {
			t.Errorf("getCreateTableDependencies() = %d deps, want 0", len(deps))
		}
	})

	t.Run("with_dependencies", func(t *testing.T) {
		op := &ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "blog", Name: "posts"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
			},
		}

		deps := getCreateTableDependencies(op, schema)
		if len(deps) != 1 {
			t.Errorf("getCreateTableDependencies() = %d deps, want 1", len(deps))
		}
	})

	t.Run("skip_self_reference", func(t *testing.T) {
		op := &ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "org", Name: "categories"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "id"},
				{Name: "parent_id", Type: "uuid", Reference: &ast.Reference{Table: "org.categories", Column: "id"}},
			},
		}

		deps := getCreateTableDependencies(op, schema)
		// Self-reference should NOT be counted as dependency
		if len(deps) != 0 {
			t.Errorf("getCreateTableDependencies() = %d deps, want 0 (self-ref excluded)", len(deps))
		}
	})
}

// -----------------------------------------------------------------------------
// sortDropTablesByDependency Tests
// -----------------------------------------------------------------------------

func TestSortDropTablesByDependency(t *testing.T) {
	t.Run("sorts_reverse_alphabetically", func(t *testing.T) {
		ops := []*ast.DropTable{
			{TableOp: ast.TableOp{Namespace: "a", Name: "first"}},
			{TableOp: ast.TableOp{Namespace: "b", Name: "second"}},
			{TableOp: ast.TableOp{Namespace: "a", Name: "third"}},
		}

		sorted := sortDropTablesByDependency(ops)

		if len(sorted) != 3 {
			t.Fatalf("sortDropTablesByDependency() = %d ops, want 3", len(sorted))
		}

		// Should be sorted in reverse alphabetical order
		// Table() returns namespace_name format
		// b_second, a_third, a_first
		if sorted[0].Table() != "b_second" {
			t.Errorf("sorted[0] = %q, want %q", sorted[0].Table(), "b_second")
		}
		if sorted[1].Table() != "a_third" {
			t.Errorf("sorted[1] = %q, want %q", sorted[1].Table(), "a_third")
		}
		if sorted[2].Table() != "a_first" {
			t.Errorf("sorted[2] = %q, want %q", sorted[2].Table(), "a_first")
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		ops := []*ast.DropTable{}
		sorted := sortDropTablesByDependency(ops)
		if len(sorted) != 0 {
			t.Errorf("sortDropTablesByDependency() = %d ops, want 0", len(sorted))
		}
	})

	t.Run("single_item", func(t *testing.T) {
		ops := []*ast.DropTable{
			{TableOp: ast.TableOp{Namespace: "a", Name: "only"}},
		}
		sorted := sortDropTablesByDependency(ops)
		if len(sorted) != 1 {
			t.Errorf("sortDropTablesByDependency() = %d ops, want 1", len(sorted))
		}
	})
}

// -----------------------------------------------------------------------------
// engine.ParseSimpleRef Extended Tests
// -----------------------------------------------------------------------------

func TestParseSimpleRefNestedNamespace(t *testing.T) {
	ns, table, isRelative := engine.ParseSimpleRef("app.auth.users")
	if ns != "app.auth" {
		t.Errorf("ns = %q, want %q", ns, "app.auth")
	}
	if table != "users" {
		t.Errorf("table = %q, want %q", table, "users")
	}
	if isRelative {
		t.Error("isRelative should be false")
	}
}

// -----------------------------------------------------------------------------
// sortCreateTablesByDependency Tests
// -----------------------------------------------------------------------------

func TestSortCreateTablesByDependency(t *testing.T) {
	schema := engine.NewSchema()

	t.Run("no_dependencies", func(t *testing.T) {
		ops := []*ast.CreateTable{
			{TableOp: ast.TableOp{Namespace: "b", Name: "second"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id"}}},
			{TableOp: ast.TableOp{Namespace: "a", Name: "first"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id"}}},
		}

		sorted := sortCreateTablesByDependency(ops, schema)
		if len(sorted) != 2 {
			t.Fatalf("sortCreateTablesByDependency() = %d ops, want 2", len(sorted))
		}
	})

	t.Run("with_dependencies", func(t *testing.T) {
		ops := []*ast.CreateTable{
			{
				TableOp: ast.TableOp{Namespace: "blog", Name: "posts"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id"},
					{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "auth.users", Column: "id"}},
				},
			},
			{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{{Name: "id", Type: "id"}},
			},
		}

		sorted := sortCreateTablesByDependency(ops, schema)
		if len(sorted) != 2 {
			t.Fatalf("sortCreateTablesByDependency() = %d ops, want 2", len(sorted))
		}

		// auth.users should come before blog.posts
		// Table() returns namespace_name format
		if sorted[0].Table() != "auth_users" {
			t.Errorf("sorted[0] = %q, want %q", sorted[0].Table(), "auth_users")
		}
		if sorted[1].Table() != "blog_posts" {
			t.Errorf("sorted[1] = %q, want %q", sorted[1].Table(), "blog_posts")
		}
	})
}

// -----------------------------------------------------------------------------
// diffColumns Edge Cases
// -----------------------------------------------------------------------------

func TestDiffColumnsNullableChange(t *testing.T) {
	oldTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "bio", Type: "string", Nullable: false},
		},
	}
	newTable := &ast.TableDef{
		Namespace: "auth",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "bio", Type: "string", Nullable: true},
		},
	}

	ops := diffColumns(oldTable, newTable)

	if len(ops) != 1 {
		t.Fatalf("diffColumns() = %d ops, want 1", len(ops))
	}

	alterOp, ok := ops[0].(*ast.AlterColumn)
	if !ok {
		t.Fatalf("ops[0] type = %T, want *ast.AlterColumn", ops[0])
	}

	if alterOp.SetNullable == nil || *alterOp.SetNullable != true {
		t.Error("AlterColumn.SetNullable should be true")
	}
}

// -----------------------------------------------------------------------------
// Diff Integration Tests
// -----------------------------------------------------------------------------

func TestDiffIntegration(t *testing.T) {
	t.Run("empty_schemas", func(t *testing.T) {
		oldSchema := engine.NewSchema()
		newSchema := engine.NewSchema()

		ops, err := Diff(oldSchema, newSchema)
		if err != nil {
			t.Fatalf("Diff() error = %v", err)
		}
		if len(ops) != 0 {
			t.Errorf("Diff() = %d ops, want 0", len(ops))
		}
	})

	t.Run("add_table", func(t *testing.T) {
		oldSchema := engine.NewSchema()
		newSchema := engine.NewSchema()
		newSchema.Tables["auth.users"] = &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id"}},
		}

		ops, err := Diff(oldSchema, newSchema)
		if err != nil {
			t.Fatalf("Diff() error = %v", err)
		}

		createCount := 0
		for _, op := range ops {
			if _, ok := op.(*ast.CreateTable); ok {
				createCount++
			}
		}

		if createCount != 1 {
			t.Errorf("Diff() CreateTable count = %d, want 1", createCount)
		}
	})

	t.Run("drop_table", func(t *testing.T) {
		oldSchema := engine.NewSchema()
		oldSchema.Tables["auth.users"] = &ast.TableDef{
			Namespace: "auth",
			Name:      "users",
			Columns:   []*ast.ColumnDef{{Name: "id", Type: "id"}},
		}
		newSchema := engine.NewSchema()

		ops, err := Diff(oldSchema, newSchema)
		if err != nil {
			t.Fatalf("Diff() error = %v", err)
		}

		dropCount := 0
		for _, op := range ops {
			if _, ok := op.(*ast.DropTable); ok {
				dropCount++
			}
		}

		if dropCount != 1 {
			t.Errorf("Diff() DropTable count = %d, want 1", dropCount)
		}
	})
}
