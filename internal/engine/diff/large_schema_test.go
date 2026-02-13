package diff

import (
	"fmt"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/engine"
)

// makeTable creates a simple table definition with the given columns.
func makeTable(ns, name string, cols ...*ast.ColumnDef) *ast.TableDef {
	return &ast.TableDef{
		Namespace: ns,
		Name:      name,
		Columns:   cols,
	}
}

// makeCol creates a basic column definition.
func makeCol(name, typ string) *ast.ColumnDef {
	return &ast.ColumnDef{Name: name, Type: typ}
}

// TestDiff_LargeSchema_ManyTables tests diffing schemas with 100+ tables.
func TestDiff_LargeSchema_ManyTables(t *testing.T) {
	const tableCount = 100

	old := engine.NewSchema()
	new_ := engine.NewSchema()

	// Build old schema with tableCount tables
	for i := 0; i < tableCount; i++ {
		name := fmt.Sprintf("table_%03d", i)
		old.Tables[fmt.Sprintf("app.%s", name)] = makeTable("app", name,
			makeCol("id", "id"),
			makeCol("name", "string"),
		)
	}

	// New schema: same tables plus 10 new ones, minus 10 old ones, 10 modified
	for i := 10; i < tableCount; i++ {
		name := fmt.Sprintf("table_%03d", i)
		cols := []*ast.ColumnDef{
			makeCol("id", "id"),
			makeCol("name", "string"),
		}
		// Modify tables 50-59: add an extra column
		if i >= 50 && i < 60 {
			cols = append(cols, makeCol("extra", "text"))
		}
		new_.Tables[fmt.Sprintf("app.%s", name)] = makeTable("app", name, cols...)
	}
	// Add 10 new tables
	for i := tableCount; i < tableCount+10; i++ {
		name := fmt.Sprintf("table_%03d", i)
		new_.Tables[fmt.Sprintf("app.%s", name)] = makeTable("app", name,
			makeCol("id", "id"),
			makeCol("title", "text"),
		)
	}

	ops, err := Diff(old, new_)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Count operation types
	counts := make(map[ast.OpType]int)
	for _, op := range ops {
		counts[op.Type()]++
	}

	// Should have 10 drops (tables 0-9), 10 creates (tables 100-109), 10 add columns (tables 50-59)
	if counts[ast.OpDropTable] != 10 {
		t.Errorf("expected 10 DropTable ops, got %d", counts[ast.OpDropTable])
	}
	if counts[ast.OpCreateTable] != 10 {
		t.Errorf("expected 10 CreateTable ops, got %d", counts[ast.OpCreateTable])
	}
	if counts[ast.OpAddColumn] != 10 {
		t.Errorf("expected 10 AddColumn ops, got %d", counts[ast.OpAddColumn])
	}
}

// TestDiff_LargeSchema_DeepFKChain tests diffing a schema with a long foreign key chain.
func TestDiff_LargeSchema_DeepFKChain(t *testing.T) {
	const chainLen = 20

	// Build a chain: table_00 <- table_01 <- table_02 <- ... <- table_19
	new_ := engine.NewSchema()

	// First table has no FK
	new_.Tables["app.table_00"] = makeTable("app", "table_00",
		makeCol("id", "id"),
		makeCol("name", "string"),
	)

	// Each subsequent table references the previous one
	for i := 1; i < chainLen; i++ {
		name := fmt.Sprintf("table_%02d", i)
		refName := fmt.Sprintf("table_%02d", i-1)
		td := makeTable("app", name,
			makeCol("id", "id"),
			makeCol("parent_id", "uuid"),
		)
		td.ForeignKeys = []*ast.ForeignKeyDef{
			{
				Name:       fmt.Sprintf("fk_%s_parent", name),
				Columns:    []string{"parent_id"},
				RefTable:   "app_" + refName,
				RefColumns: []string{"id"},
			},
		}
		new_.Tables[fmt.Sprintf("app.%s", name)] = td
	}

	// Diff from empty schema
	old := engine.NewSchema()
	ops, err := Diff(old, new_)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Should have chainLen CreateTable operations
	createCount := 0
	for _, op := range ops {
		if op.Type() == ast.OpCreateTable {
			createCount++
		}
	}
	if createCount != chainLen {
		t.Errorf("expected %d CreateTable ops, got %d", chainLen, createCount)
	}

	// Verify topological order: each table appears after its referenced table
	tableOrder := make(map[string]int)
	idx := 0
	for _, op := range ops {
		if ct, ok := op.(*ast.CreateTable); ok {
			tableOrder[ct.Table()] = idx
			idx++
		}
	}

	for i := 1; i < chainLen; i++ {
		name := fmt.Sprintf("app_table_%02d", i)
		refName := fmt.Sprintf("app_table_%02d", i-1)
		if tableOrder[name] <= tableOrder[refName] {
			t.Errorf("table %s (order %d) should come after %s (order %d)",
				name, tableOrder[name], refName, tableOrder[refName])
		}
	}
}

// TestDiff_LargeSchema_ManyColumnsPerTable tests diffing tables with many columns.
func TestDiff_LargeSchema_ManyColumnsPerTable(t *testing.T) {
	const colCount = 50

	buildCols := func(count int, suffix string) []*ast.ColumnDef {
		cols := []*ast.ColumnDef{makeCol("id", "id")}
		for i := 0; i < count; i++ {
			cols = append(cols, makeCol(fmt.Sprintf("col_%03d%s", i, suffix), "string"))
		}
		return cols
	}

	old := engine.NewSchema()
	old.Tables["app.wide_table"] = makeTable("app", "wide_table", buildCols(colCount, "")...)

	// New schema: rename-like pattern (drop some, add some with different names)
	new_ := engine.NewSchema()
	newCols := []*ast.ColumnDef{makeCol("id", "id")}
	for i := 0; i < colCount; i++ {
		if i%5 == 0 {
			// Changed columns (every 5th)
			newCols = append(newCols, makeCol(fmt.Sprintf("col_%03d_v2", i), "string"))
		} else {
			newCols = append(newCols, makeCol(fmt.Sprintf("col_%03d", i), "string"))
		}
	}
	new_.Tables["app.wide_table"] = makeTable("app", "wide_table", newCols...)

	ops, err := Diff(old, new_)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Should have drops and adds for the changed columns
	drops := 0
	adds := 0
	for _, op := range ops {
		switch op.Type() {
		case ast.OpDropColumn:
			drops++
		case ast.OpAddColumn:
			adds++
		}
	}

	// 10 columns changed (every 5th out of 50)
	if drops != 10 {
		t.Errorf("expected 10 DropColumn ops, got %d", drops)
	}
	if adds != 10 {
		t.Errorf("expected 10 AddColumn ops, got %d", adds)
	}
}

// TestDiff_EmptyToLarge tests creating a large schema from scratch.
func TestDiff_EmptyToLarge(t *testing.T) {
	const tableCount = 50
	new_ := engine.NewSchema()

	for i := 0; i < tableCount; i++ {
		name := fmt.Sprintf("table_%02d", i)
		new_.Tables[fmt.Sprintf("app.%s", name)] = makeTable("app", name,
			makeCol("id", "id"),
			makeCol("name", "string"),
			makeCol("value", "integer"),
		)
	}

	ops, err := Diff(engine.NewSchema(), new_)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	createCount := 0
	for _, op := range ops {
		if op.Type() == ast.OpCreateTable {
			createCount++
		}
	}
	if createCount != tableCount {
		t.Errorf("expected %d CreateTable ops, got %d", tableCount, createCount)
	}
}

// TestDiff_LargeToEmpty tests dropping a large schema entirely.
func TestDiff_LargeToEmpty(t *testing.T) {
	const tableCount = 50
	old := engine.NewSchema()

	for i := 0; i < tableCount; i++ {
		name := fmt.Sprintf("table_%02d", i)
		old.Tables[fmt.Sprintf("app.%s", name)] = makeTable("app", name,
			makeCol("id", "id"),
			makeCol("name", "string"),
		)
	}

	ops, err := Diff(old, engine.NewSchema())
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	dropCount := 0
	for _, op := range ops {
		if op.Type() == ast.OpDropTable {
			dropCount++
		}
	}
	if dropCount != tableCount {
		t.Errorf("expected %d DropTable ops, got %d", tableCount, dropCount)
	}
}
