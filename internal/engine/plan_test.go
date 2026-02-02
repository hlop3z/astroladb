package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/hlop3z/astroladb/internal/ast"
)

// -----------------------------------------------------------------------------
// Direction Tests
// -----------------------------------------------------------------------------

func TestDirectionString(t *testing.T) {
	tests := []struct {
		dir  Direction
		want string
	}{
		{Up, "up"},
		{Down, "down"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.dir.String(); got != tt.want {
				t.Errorf("Direction.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// PlanMigrations Up Tests
// -----------------------------------------------------------------------------

func TestPlanMigrationsUpEmpty(t *testing.T) {
	all := []Migration{}
	applied := []AppliedMigration{}

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if !plan.IsEmpty() {
		t.Errorf("Plan should be empty, has %d migrations", len(plan.Migrations))
	}

	if plan.Direction != Up {
		t.Errorf("Plan.Direction = %v, want Up", plan.Direction)
	}
}

func TestPlanMigrationsUpAllPending(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
		{Revision: "002", Name: "create_posts"},
		{Revision: "003", Name: "add_comments"},
	}
	applied := []AppliedMigration{}

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if len(plan.Migrations) != 3 {
		t.Errorf("Plan has %d migrations, want 3", len(plan.Migrations))
	}

	// Should be in order
	for i, m := range plan.Migrations {
		expected := all[i].Revision
		if m.Revision != expected {
			t.Errorf("Plan.Migrations[%d].Revision = %q, want %q", i, m.Revision, expected)
		}
	}
}

func TestPlanMigrationsUpPartiallyApplied(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
		{Revision: "002", Name: "create_posts"},
		{Revision: "003", Name: "add_comments"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now()},
	}

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if len(plan.Migrations) != 2 {
		t.Errorf("Plan has %d migrations, want 2", len(plan.Migrations))
	}

	if plan.Migrations[0].Revision != "002" {
		t.Errorf("First pending should be 002, got %q", plan.Migrations[0].Revision)
	}
}

func TestPlanMigrationsUpWithTarget(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
		{Revision: "002", Name: "create_posts"},
		{Revision: "003", Name: "add_comments"},
	}
	applied := []AppliedMigration{}

	// Target only up to 002
	plan, err := PlanMigrations(all, applied, "002", Up)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if len(plan.Migrations) != 2 {
		t.Errorf("Plan has %d migrations, want 2 (up to target)", len(plan.Migrations))
	}

	// Last migration should be 002
	if plan.Migrations[len(plan.Migrations)-1].Revision != "002" {
		t.Errorf("Last migration should be 002")
	}
}

func TestPlanMigrationsUpTargetNotFound(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
		{Revision: "002", Name: "create_posts"},
	}
	applied := []AppliedMigration{}

	_, err := PlanMigrations(all, applied, "999", Up)
	if err == nil {
		t.Error("PlanMigrations() should fail for non-existent target")
	}
}

func TestPlanMigrationsUpAllApplied(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now()},
	}

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if !plan.IsEmpty() {
		t.Errorf("Plan should be empty when all applied, has %d", len(plan.Migrations))
	}
}

// -----------------------------------------------------------------------------
// PlanMigrations Down Tests
// -----------------------------------------------------------------------------

func TestPlanMigrationsDownEmpty(t *testing.T) {
	all := []Migration{}
	applied := []AppliedMigration{}

	plan, err := PlanMigrations(all, applied, "", Down)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if !plan.IsEmpty() {
		t.Errorf("Plan should be empty, has %d migrations", len(plan.Migrations))
	}

	if plan.Direction != Down {
		t.Errorf("Plan.Direction = %v, want Down", plan.Direction)
	}
}

func TestPlanMigrationsDownAll(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
		{Revision: "002", Name: "create_posts"},
		{Revision: "003", Name: "add_comments"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now().Add(-3 * time.Hour)},
		{Revision: "002", AppliedAt: time.Now().Add(-2 * time.Hour)},
		{Revision: "003", AppliedAt: time.Now().Add(-1 * time.Hour)},
	}

	plan, err := PlanMigrations(all, applied, "", Down)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if len(plan.Migrations) != 3 {
		t.Errorf("Plan has %d migrations, want 3", len(plan.Migrations))
	}

	// Should be in reverse order (most recent first)
	if plan.Migrations[0].Revision != "003" {
		t.Errorf("First rollback should be 003, got %q", plan.Migrations[0].Revision)
	}
	if plan.Migrations[2].Revision != "001" {
		t.Errorf("Last rollback should be 001, got %q", plan.Migrations[2].Revision)
	}
}

func TestPlanMigrationsDownWithTarget(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
		{Revision: "002", Name: "create_posts"},
		{Revision: "003", Name: "add_comments"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now().Add(-3 * time.Hour)},
		{Revision: "002", AppliedAt: time.Now().Add(-2 * time.Hour)},
		{Revision: "003", AppliedAt: time.Now().Add(-1 * time.Hour)},
	}

	// Rollback down to (but not including) 001
	plan, err := PlanMigrations(all, applied, "001", Down)
	if err != nil {
		t.Fatalf("PlanMigrations() error = %v", err)
	}

	if len(plan.Migrations) != 2 {
		t.Errorf("Plan has %d migrations, want 2 (003 and 002)", len(plan.Migrations))
	}

	// 001 should remain (target stays applied)
	for _, m := range plan.Migrations {
		if m.Revision == "001" {
			t.Error("Target migration 001 should not be in rollback plan")
		}
	}
}

func TestPlanMigrationsDownMissingFile(t *testing.T) {
	// Applied migration but no corresponding migration file
	all := []Migration{
		{Revision: "001", Name: "create_users"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now()},
		{Revision: "002", AppliedAt: time.Now()}, // No migration file for this
	}

	_, err := PlanMigrations(all, applied, "", Down)
	if err == nil {
		t.Error("PlanMigrations() should fail for missing migration file")
	}
}

func TestPlanMigrationsDownIrreversible(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "drop_important_data", Irreversible: true},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now()},
	}

	_, err := PlanMigrations(all, applied, "", Down)
	if err == nil {
		t.Error("PlanMigrations() should fail for irreversible migration")
	}
}

// -----------------------------------------------------------------------------
// PlanSingle Tests
// -----------------------------------------------------------------------------

func TestPlanSingle(t *testing.T) {
	m := Migration{Revision: "001", Name: "create_users"}

	upPlan := PlanSingle(m, Up)
	if len(upPlan.Migrations) != 1 {
		t.Errorf("PlanSingle(Up) = %d migrations, want 1", len(upPlan.Migrations))
	}
	if upPlan.Direction != Up {
		t.Errorf("PlanSingle(Up).Direction = %v, want Up", upPlan.Direction)
	}

	downPlan := PlanSingle(m, Down)
	if downPlan.Direction != Down {
		t.Errorf("PlanSingle(Down).Direction = %v, want Down", downPlan.Direction)
	}
}

// -----------------------------------------------------------------------------
// GenerateDownOps Tests
// -----------------------------------------------------------------------------

func TestGenerateDownOpsCreateTable(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{
			TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
			Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}},
		},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	dropOp, ok := downOps[0].(*ast.DropTable)
	if !ok {
		t.Fatalf("down op type = %T, want *ast.DropTable", downOps[0])
	}

	if dropOp.Table() != "auth_users" {
		t.Errorf("DropTable.Table() = %q, want %q", dropOp.Table(), "auth_users")
	}

	if !dropOp.IfExists {
		t.Error("DropTable.IfExists should be true")
	}
}

func TestGenerateDownOpsAddColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
			Column:   &ast.ColumnDef{Name: "email", Type: "string"},
		},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	dropOp, ok := downOps[0].(*ast.DropColumn)
	if !ok {
		t.Fatalf("down op type = %T, want *ast.DropColumn", downOps[0])
	}

	if dropOp.Name != "email" {
		t.Errorf("DropColumn.Name = %q, want %q", dropOp.Name, "email")
	}
}

func TestGenerateDownOpsRenameColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.RenameColumn{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
			OldName:  "email",
			NewName:  "email_address",
		},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	renameOp, ok := downOps[0].(*ast.RenameColumn)
	if !ok {
		t.Fatalf("down op type = %T, want *ast.RenameColumn", downOps[0])
	}

	// Should be reversed
	if renameOp.OldName != "email_address" || renameOp.NewName != "email" {
		t.Errorf("RenameColumn should be reversed: %q -> %q", renameOp.OldName, renameOp.NewName)
	}
}

func TestGenerateDownOpsRenameTable(t *testing.T) {
	ops := []ast.Operation{
		&ast.RenameTable{
			Namespace: "auth",
			OldName:   "users",
			NewName:   "accounts",
		},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	renameOp, ok := downOps[0].(*ast.RenameTable)
	if !ok {
		t.Fatalf("down op type = %T, want *ast.RenameTable", downOps[0])
	}

	// Should be reversed
	if renameOp.OldName != "accounts" || renameOp.NewName != "users" {
		t.Errorf("RenameTable should be reversed: %q -> %q", renameOp.OldName, renameOp.NewName)
	}
}

func TestGenerateDownOpsCreateIndex(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateIndex{
			TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
			Name:     "idx_users_email",
			Columns:  []string{"email"},
		},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	dropOp, ok := downOps[0].(*ast.DropIndex)
	if !ok {
		t.Fatalf("down op type = %T, want *ast.DropIndex", downOps[0])
	}

	if dropOp.Name != "idx_users_email" {
		t.Errorf("DropIndex.Name = %q, want %q", dropOp.Name, "idx_users_email")
	}
}

func TestGenerateDownOpsAddForeignKey(t *testing.T) {
	ops := []ast.Operation{
		&ast.AddForeignKey{
			TableRef:   ast.TableRef{Namespace: "blog", Table_: "posts"},
			Name:       "fk_posts_user_id",
			Columns:    []string{"user_id"},
			RefTable:   "auth_users",
			RefColumns: []string{"id"},
		},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	dropOp, ok := downOps[0].(*ast.DropForeignKey)
	if !ok {
		t.Fatalf("down op type = %T, want *ast.DropForeignKey", downOps[0])
	}

	if dropOp.Name != "fk_posts_user_id" {
		t.Errorf("DropForeignKey.Name = %q, want %q", dropOp.Name, "fk_posts_user_id")
	}
}

func TestGenerateDownOpsReverseOrder(t *testing.T) {
	ops := []ast.Operation{
		&ast.CreateTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
		&ast.AddColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Column: &ast.ColumnDef{Name: "email", Type: "string"}},
		&ast.CreateIndex{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "idx_email", Columns: []string{"email"}},
	}

	downOps := GenerateDownOps(ops)

	if len(downOps) != 3 {
		t.Fatalf("GenerateDownOps() = %d ops, want 3", len(downOps))
	}

	// Should be in reverse order: DropIndex, DropColumn, DropTable
	if _, ok := downOps[0].(*ast.DropIndex); !ok {
		t.Error("First down op should be DropIndex")
	}
	if _, ok := downOps[1].(*ast.DropColumn); !ok {
		t.Error("Second down op should be DropColumn")
	}
	if _, ok := downOps[2].(*ast.DropTable); !ok {
		t.Error("Third down op should be DropTable")
	}
}

func TestGenerateDownOpsIrreversible(t *testing.T) {
	// These operations cannot be auto-reversed
	irreversibleOps := []ast.Operation{
		&ast.DropTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}},
		&ast.DropColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "email"},
		&ast.DropIndex{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "idx"},
		&ast.DropForeignKey{TableRef: ast.TableRef{Namespace: "auth", Table_: "posts"}, Name: "fk"},
		&ast.AlterColumn{TableRef: ast.TableRef{Namespace: "auth", Table_: "users"}, Name: "col", NewType: "text"},
		&ast.RawSQL{SQL: "DELETE FROM users"},
	}

	for _, op := range irreversibleOps {
		downOps := GenerateDownOps([]ast.Operation{op})
		if len(downOps) != 0 {
			t.Errorf("GenerateDownOps(%T) should return empty slice for irreversible op", op)
		}
	}
}

// -----------------------------------------------------------------------------
// HasIrreversibleOps Tests
// -----------------------------------------------------------------------------

func TestHasIrreversibleOps(t *testing.T) {
	tests := []struct {
		name string
		ops  []ast.Operation
		want bool
	}{
		{
			name: "all_reversible",
			ops: []ast.Operation{
				&ast.CreateTable{TableOp: ast.TableOp{Namespace: "a", Name: "b"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
				&ast.AddColumn{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Column: &ast.ColumnDef{Name: "c", Type: "string"}},
			},
			want: false,
		},
		{
			name: "has_drop_table",
			ops: []ast.Operation{
				&ast.CreateTable{TableOp: ast.TableOp{Namespace: "a", Name: "b"}, Columns: []*ast.ColumnDef{{Name: "id", Type: "id", PrimaryKey: true}}},
				&ast.DropTable{TableOp: ast.TableOp{Namespace: "a", Name: "old"}},
			},
			want: true,
		},
		{
			name: "has_drop_column",
			ops: []ast.Operation{
				&ast.DropColumn{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "col"},
			},
			want: true,
		},
		{
			name: "has_alter_column",
			ops: []ast.Operation{
				&ast.AlterColumn{TableRef: ast.TableRef{Namespace: "a", Table_: "b"}, Name: "col", NewType: "text"},
			},
			want: true,
		},
		{
			name: "has_raw_sql",
			ops: []ast.Operation{
				&ast.RawSQL{SQL: "UPDATE users SET active = true"},
			},
			want: true,
		},
		{
			name: "empty",
			ops:  []ast.Operation{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasIrreversibleOps(tt.ops); got != tt.want {
				t.Errorf("HasIrreversibleOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// GetStatus Tests
// -----------------------------------------------------------------------------

func TestGetStatus(t *testing.T) {
	now := time.Now()

	all := []Migration{
		{Revision: "001", Name: "create_users", Checksum: "abc123"},
		{Revision: "002", Name: "create_posts", Checksum: "def456"},
		{Revision: "003", Name: "add_comments", Checksum: "ghi789"},
	}

	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: now.Add(-2 * time.Hour), Checksum: "abc123"},
		{Revision: "002", AppliedAt: now.Add(-1 * time.Hour), Checksum: "modified"}, // Checksum mismatch
	}

	statuses := GetStatus(all, applied)

	if len(statuses) != 3 {
		t.Fatalf("GetStatus() = %d statuses, want 3", len(statuses))
	}

	// Check each status
	for _, s := range statuses {
		switch s.Revision {
		case "001":
			if s.Status != StatusApplied {
				t.Errorf("001 status = %v, want StatusApplied", s.Status)
			}
		case "002":
			if s.Status != StatusModified {
				t.Errorf("002 status = %v, want StatusModified (checksum mismatch)", s.Status)
			}
		case "003":
			if s.Status != StatusPending {
				t.Errorf("003 status = %v, want StatusPending", s.Status)
			}
		}
	}
}

func TestGetStatusMissing(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users"},
	}

	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now()},
		{Revision: "002", AppliedAt: time.Now()}, // Applied but file is missing
	}

	statuses := GetStatus(all, applied)

	if len(statuses) != 2 {
		t.Fatalf("GetStatus() = %d statuses, want 2", len(statuses))
	}

	// Find the missing one
	for _, s := range statuses {
		if s.Revision == "002" {
			if s.Status != StatusMissing {
				t.Errorf("002 status = %v, want StatusMissing", s.Status)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// PlanStatus String Tests
// -----------------------------------------------------------------------------

func TestPlanStatusString(t *testing.T) {
	tests := []struct {
		status PlanStatus
		want   string
	}{
		{StatusPending, "pending"},
		{StatusApplied, "applied"},
		{StatusMissing, "missing"},
		{StatusModified, "modified"},
		{PlanStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("PlanStatus.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// GenerateDownOps Reversibility Tests
// -----------------------------------------------------------------------------

func TestGenerateDownOpsAlterColumnReversible(t *testing.T) {
	oldNullable := false
	newNullable := true

	ops := []ast.Operation{
		&ast.AlterColumn{
			TableRef: ast.TableRef{Namespace: "public", Table_: "users"},
			Name:     "email",
			NewType:  "text",
			OldColumn: &ast.ColumnDef{
				Name:     "email",
				Type:     "string",
				TypeArgs: []any{255},
				Nullable: false,
			},
			SetNullable: &newNullable,
		},
	}

	downOps := GenerateDownOps(ops)
	if len(downOps) != 1 {
		t.Fatalf("GenerateDownOps() = %d ops, want 1", len(downOps))
	}

	alter, ok := downOps[0].(*ast.AlterColumn)
	if !ok {
		t.Fatalf("expected *ast.AlterColumn, got %T", downOps[0])
	}
	if alter.NewType != "string" {
		t.Errorf("reverse NewType = %q, want %q", alter.NewType, "string")
	}
	if len(alter.NewTypeArgs) != 1 {
		t.Fatalf("reverse NewTypeArgs len = %d, want 1", len(alter.NewTypeArgs))
	}
	if alter.SetNullable == nil || *alter.SetNullable != oldNullable {
		t.Errorf("reverse SetNullable = %v, want %v", alter.SetNullable, &oldNullable)
	}

	// With OldColumn present, it should NOT be irreversible
	if HasIrreversibleOps(ops) {
		t.Error("AlterColumn with OldColumn should be reversible")
	}
}

func TestGenerateDownOpsAlterColumnWithoutOldColumn(t *testing.T) {
	ops := []ast.Operation{
		&ast.AlterColumn{
			TableRef: ast.TableRef{Namespace: "public", Table_: "users"},
			Name:     "email",
			NewType:  "text",
		},
	}

	downOps := GenerateDownOps(ops)
	if len(downOps) != 0 {
		t.Fatalf("GenerateDownOps() without OldColumn should return 0 ops, got %d", len(downOps))
	}

	if !HasIrreversibleOps(ops) {
		t.Error("AlterColumn without OldColumn should be irreversible")
	}
}

func TestGenerateDownOpsDropColumnIrreversible(t *testing.T) {
	ops := []ast.Operation{
		&ast.DropColumn{
			TableRef: ast.TableRef{Namespace: "public", Table_: "users"},
			Name:     "old_field",
		},
	}

	if !HasIrreversibleOps(ops) {
		t.Error("DropColumn should be flagged as irreversible")
	}

	downOps := GenerateDownOps(ops)
	if len(downOps) != 0 {
		t.Errorf("DropColumn should generate 0 down ops, got %d", len(downOps))
	}
}

// -----------------------------------------------------------------------------
// Baseline Skip Tests
// -----------------------------------------------------------------------------

func TestPlanUp_SkipsBaselineWhenMigrationsApplied(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "baseline", IsBaseline: true, SquashedThrough: "003"},
		{Revision: "004", Name: "add_column"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now(), Checksum: "abc"},
		{Revision: "002", AppliedAt: time.Now(), Checksum: "def"},
		{Revision: "003", AppliedAt: time.Now(), Checksum: "ghi"},
	}

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatal(err)
	}
	// Should skip baseline (already has applied migrations) and include 004
	if len(plan.Migrations) != 1 {
		t.Fatalf("expected 1 pending migration, got %d", len(plan.Migrations))
	}
	if plan.Migrations[0].Revision != "004" {
		t.Errorf("expected revision '004', got %q", plan.Migrations[0].Revision)
	}
}

func TestPlanUp_RejectsModifiedChecksum(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users", Checksum: "changed_checksum"},
		{Revision: "002", Name: "add_column", Checksum: "aaa"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now(), Checksum: "original_checksum"},
	}

	_, err := PlanMigrations(all, applied, "", Up)
	if err == nil {
		t.Fatal("expected error for checksum mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "modified") {
		t.Errorf("expected 'modified' in error, got: %s", err.Error())
	}
}

func TestPlanUp_AllowsEmptyChecksums(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "create_users", Checksum: "abc"},
		{Revision: "002", Name: "add_column", Checksum: "def"},
	}
	applied := []AppliedMigration{
		{Revision: "001", AppliedAt: time.Now(), Checksum: ""}, // old migration, no checksum recorded
	}

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatalf("empty checksums should not cause error: %v", err)
	}
	if len(plan.Migrations) != 1 || plan.Migrations[0].Revision != "002" {
		t.Errorf("expected 002 pending, got %v", plan.Migrations)
	}
}

func TestPlanUp_AppliesBaselineOnFreshDB(t *testing.T) {
	all := []Migration{
		{Revision: "001", Name: "baseline", IsBaseline: true, SquashedThrough: "003"},
		{Revision: "004", Name: "add_column"},
	}
	var applied []AppliedMigration // fresh DB

	plan, err := PlanMigrations(all, applied, "", Up)
	if err != nil {
		t.Fatal(err)
	}
	// Should apply both baseline and 004
	if len(plan.Migrations) != 2 {
		t.Fatalf("expected 2 pending migrations, got %d", len(plan.Migrations))
	}
	if plan.Migrations[0].Revision != "001" {
		t.Errorf("expected first revision '001', got %q", plan.Migrations[0].Revision)
	}
}
