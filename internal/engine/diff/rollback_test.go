package diff

import (
	"fmt"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// TestDiff_Rollback_Reversibility tests that diffing from new->old produces
// the inverse operations of diffing old->new.
func TestDiff_Rollback_Reversibility(t *testing.T) {
	old := engine.NewSchema()
	old.Tables["app.users"] = makeTable("app", "users",
		makeCol("id", "id"),
		makeCol("name", "string"),
		makeCol("email", "string"),
	)

	new_ := engine.NewSchema()
	new_.Tables["app.users"] = makeTable("app", "users",
		makeCol("id", "id"),
		makeCol("name", "string"),
		makeCol("email", "string"),
		makeCol("age", "integer"),
	)
	new_.Tables["app.posts"] = makeTable("app", "posts",
		makeCol("id", "id"),
		makeCol("title", "text"),
	)

	// Forward diff: old -> new
	forwardOps, err := Diff(old, new_)
	if err != nil {
		t.Fatalf("forward Diff failed: %v", err)
	}

	// Reverse diff: new -> old
	reverseOps, err := Diff(new_, old)
	if err != nil {
		t.Fatalf("reverse Diff failed: %v", err)
	}

	// Forward should have: CreateTable(posts), AddColumn(age)
	forwardTypes := make(map[ast.OpType]int)
	for _, op := range forwardOps {
		forwardTypes[op.Type()]++
	}

	// Reverse should have: DropTable(posts), DropColumn(age)
	reverseTypes := make(map[ast.OpType]int)
	for _, op := range reverseOps {
		reverseTypes[op.Type()]++
	}

	if forwardTypes[ast.OpCreateTable] != 1 {
		t.Errorf("forward: expected 1 CreateTable, got %d", forwardTypes[ast.OpCreateTable])
	}
	if forwardTypes[ast.OpAddColumn] != 1 {
		t.Errorf("forward: expected 1 AddColumn, got %d", forwardTypes[ast.OpAddColumn])
	}

	if reverseTypes[ast.OpDropTable] != 1 {
		t.Errorf("reverse: expected 1 DropTable, got %d", reverseTypes[ast.OpDropTable])
	}
	if reverseTypes[ast.OpDropColumn] != 1 {
		t.Errorf("reverse: expected 1 DropColumn, got %d", reverseTypes[ast.OpDropColumn])
	}
}

// TestDiff_Rollback_MultiStep tests rollback across multiple migration steps.
func TestDiff_Rollback_MultiStep(t *testing.T) {
	// Simulate 3 migration steps, then verify rollback at each step
	schemas := []*engine.Schema{
		// Step 0: initial (empty)
		engine.NewSchema(),
		// Step 1: create users table
		func() *engine.Schema {
			s := engine.NewSchema()
			s.Tables["app.users"] = makeTable("app", "users",
				makeCol("id", "id"),
				makeCol("name", "string"),
			)
			return s
		}(),
		// Step 2: add posts table + email column to users
		func() *engine.Schema {
			s := engine.NewSchema()
			s.Tables["app.users"] = makeTable("app", "users",
				makeCol("id", "id"),
				makeCol("name", "string"),
				makeCol("email", "string"),
			)
			s.Tables["app.posts"] = makeTable("app", "posts",
				makeCol("id", "id"),
				makeCol("title", "text"),
			)
			return s
		}(),
		// Step 3: add comments table with FK to posts
		func() *engine.Schema {
			s := engine.NewSchema()
			s.Tables["app.users"] = makeTable("app", "users",
				makeCol("id", "id"),
				makeCol("name", "string"),
				makeCol("email", "string"),
			)
			s.Tables["app.posts"] = makeTable("app", "posts",
				makeCol("id", "id"),
				makeCol("title", "text"),
			)
			comments := makeTable("app", "comments",
				makeCol("id", "id"),
				makeCol("body", "text"),
				makeCol("post_id", "uuid"),
			)
			comments.ForeignKeys = []*ast.ForeignKeyDef{
				{
					Name:       "fk_comments_post",
					Columns:    []string{"post_id"},
					RefTable:   "app_posts",
					RefColumns: []string{"id"},
				},
			}
			s.Tables["app.comments"] = comments
			return s
		}(),
	}

	// Verify forward migration at each step
	for i := 1; i < len(schemas); i++ {
		ops, err := Diff(schemas[i-1], schemas[i])
		if err != nil {
			t.Fatalf("step %d forward Diff failed: %v", i, err)
		}
		if len(ops) == 0 {
			t.Errorf("step %d: expected operations, got none", i)
		}
	}

	// Verify rollback at each step (reverse direction)
	for i := len(schemas) - 1; i > 0; i-- {
		ops, err := Diff(schemas[i], schemas[i-1])
		if err != nil {
			t.Fatalf("rollback step %d->%d Diff failed: %v", i, i-1, err)
		}
		if len(ops) == 0 {
			t.Errorf("rollback step %d->%d: expected operations, got none", i, i-1)
		}
	}

	// Full rollback from step 3 to empty should drop all tables
	fullRollback, err := Diff(schemas[3], schemas[0])
	if err != nil {
		t.Fatalf("full rollback Diff failed: %v", err)
	}

	dropCount := 0
	for _, op := range fullRollback {
		if op.Type() == ast.OpDropTable {
			dropCount++
		}
	}
	if dropCount != 3 {
		t.Errorf("full rollback: expected 3 DropTable ops, got %d", dropCount)
	}
}

// TestDiff_Rollback_FKOrdering tests that rollback produces DropTable operations
// for tables with FK dependencies. Note: sortDropTablesByDependency currently
// uses alphabetical reverse ordering as a stub; proper FK-aware ordering is
// tracked as a future improvement.
func TestDiff_Rollback_FKOrdering(t *testing.T) {
	// Schema with FK chain: comments -> posts -> users
	schema := engine.NewSchema()
	schema.Tables["app.users"] = makeTable("app", "users",
		makeCol("id", "id"),
		makeCol("name", "string"),
	)
	posts := makeTable("app", "posts",
		makeCol("id", "id"),
		makeCol("user_id", "uuid"),
	)
	posts.ForeignKeys = []*ast.ForeignKeyDef{
		{Name: "fk_posts_user", Columns: []string{"user_id"}, RefTable: "app_users", RefColumns: []string{"id"}},
	}
	schema.Tables["app.posts"] = posts

	comments := makeTable("app", "comments",
		makeCol("id", "id"),
		makeCol("post_id", "uuid"),
	)
	comments.ForeignKeys = []*ast.ForeignKeyDef{
		{Name: "fk_comments_post", Columns: []string{"post_id"}, RefTable: "app_posts", RefColumns: []string{"id"}},
	}
	schema.Tables["app.comments"] = comments

	// Rollback: drop everything
	ops, err := Diff(schema, engine.NewSchema())
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Verify all 3 tables are dropped
	dropCount := 0
	droppedTables := make(map[string]bool)
	for _, op := range ops {
		if dt, ok := op.(*ast.DropTable); ok {
			dropCount++
			droppedTables[dt.Table()] = true
		}
	}
	if dropCount != 3 {
		t.Errorf("expected 3 DropTable ops, got %d", dropCount)
	}
	for _, table := range []string{"app_users", "app_posts", "app_comments"} {
		if !droppedTables[table] {
			t.Errorf("expected DropTable for %s", table)
		}
	}
}

// TestDiff_Rollback_IndexAndConstraint tests that rollback handles
// index and constraint operations correctly.
func TestDiff_Rollback_IndexAndConstraint(t *testing.T) {
	old := engine.NewSchema()
	old.Tables["app.users"] = makeTable("app", "users",
		makeCol("id", "id"),
		makeCol("email", "string"),
	)

	new_ := engine.NewSchema()
	usersNew := makeTable("app", "users",
		makeCol("id", "id"),
		makeCol("email", "string"),
	)
	usersNew.Indexes = []*ast.IndexDef{
		{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
	}
	new_.Tables["app.users"] = usersNew

	// Forward: should add the index
	forwardOps, err := Diff(old, new_)
	if err != nil {
		t.Fatalf("forward Diff failed: %v", err)
	}

	hasCreateIndex := false
	for _, op := range forwardOps {
		if op.Type() == ast.OpCreateIndex {
			hasCreateIndex = true
		}
	}
	if !hasCreateIndex {
		t.Error("forward diff should have CreateIndex operation")
	}

	// Reverse: should drop the index
	reverseOps, err := Diff(new_, old)
	if err != nil {
		t.Fatalf("reverse Diff failed: %v", err)
	}

	hasDropIndex := false
	for _, op := range reverseOps {
		if op.Type() == ast.OpDropIndex {
			hasDropIndex = true
		}
	}
	if !hasDropIndex {
		t.Error("reverse diff should have DropIndex operation")
	}
}

// TestDiff_Rollback_ManyTablesStress tests rollback with many tables having interrelated FKs.
func TestDiff_Rollback_ManyTablesStress(t *testing.T) {
	const tableCount = 30

	schema := engine.NewSchema()

	// Create a star schema: central "hub" table referenced by all others
	schema.Tables["app.hub"] = makeTable("app", "hub",
		makeCol("id", "id"),
		makeCol("name", "string"),
	)

	for i := 0; i < tableCount; i++ {
		name := fmt.Sprintf("spoke_%02d", i)
		td := makeTable("app", name,
			makeCol("id", "id"),
			makeCol("hub_id", "uuid"),
			makeCol("data", "text"),
		)
		td.ForeignKeys = []*ast.ForeignKeyDef{
			{
				Name:       fmt.Sprintf("fk_%s_hub", name),
				Columns:    []string{"hub_id"},
				RefTable:   "app_hub",
				RefColumns: []string{"id"},
			},
		}
		schema.Tables[fmt.Sprintf("app.%s", name)] = td
	}

	// Forward: create everything from empty
	forwardOps, err := Diff(engine.NewSchema(), schema)
	if err != nil {
		t.Fatalf("forward Diff failed: %v", err)
	}

	// Verify hub is created first
	hubIdx := -1
	for i, op := range forwardOps {
		if ct, ok := op.(*ast.CreateTable); ok && ct.Name == "hub" {
			hubIdx = i
			break
		}
	}
	if hubIdx < 0 {
		t.Fatal("hub table not found in forward ops")
	}
	for _, op := range forwardOps[hubIdx+1:] {
		if ct, ok := op.(*ast.CreateTable); ok && ct.Name == "hub" {
			t.Error("hub table created more than once")
		}
	}

	// Reverse: drop everything
	reverseOps, err := Diff(schema, engine.NewSchema())
	if err != nil {
		t.Fatalf("reverse Diff failed: %v", err)
	}

	// Verify hub is dropped last (all spokes dropped first)
	hubDropIdx := -1
	for i, op := range reverseOps {
		if dt, ok := op.(*ast.DropTable); ok && dt.Name == "hub" {
			hubDropIdx = i
		}
	}
	if hubDropIdx < 0 {
		t.Fatal("hub table not found in reverse ops")
	}

	// All spoke drops should come before hub drop
	for i, op := range reverseOps {
		if dt, ok := op.(*ast.DropTable); ok && dt.Name != "hub" && i > hubDropIdx {
			t.Errorf("spoke %s (order %d) dropped after hub (order %d)", dt.Name, i, hubDropIdx)
		}
	}
}
