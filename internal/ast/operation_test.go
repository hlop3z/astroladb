package ast

import (
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// OpType Tests
// -----------------------------------------------------------------------------

func TestOpTypeString(t *testing.T) {
	tests := []struct {
		op   OpType
		want string
	}{
		{OpCreateTable, "CreateTable"},
		{OpDropTable, "DropTable"},
		{OpRenameTable, "RenameTable"},
		{OpAddColumn, "AddColumn"},
		{OpDropColumn, "DropColumn"},
		{OpRenameColumn, "RenameColumn"},
		{OpAlterColumn, "AlterColumn"},
		{OpCreateIndex, "CreateIndex"},
		{OpDropIndex, "DropIndex"},
		{OpAddForeignKey, "AddForeignKey"},
		{OpDropForeignKey, "DropForeignKey"},
		{OpRawSQL, "RawSQL"},
		{OpType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.op.String()
			if got != tt.want {
				t.Errorf("OpType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// CreateTable Tests
// -----------------------------------------------------------------------------

func TestCreateTableType(t *testing.T) {
	op := &CreateTable{TableOp: TableOp{Name: "users"}}
	if op.Type() != OpCreateTable {
		t.Errorf("Type() = %v, want %v", op.Type(), OpCreateTable)
	}
}

func TestCreateTableTable(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		tableName string
		want      string
	}{
		{"with_namespace", "auth", "users", "auth_users"},
		{"no_namespace", "", "users", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &CreateTable{TableOp: TableOp{Namespace: tt.namespace, Name: tt.tableName}}
			got := op.Table()
			if got != tt.want {
				t.Errorf("Table() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateTableValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *CreateTable
		wantErr bool
	}{
		{
			name: "valid",
			op: &CreateTable{
				TableOp: TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
					{Name: "email", Type: "string"},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty_name",
			op:      &CreateTable{TableOp: TableOp{Name: ""}, Columns: []*ColumnDef{{Name: "id", Type: "id"}}},
			wantErr: true,
		},
		{
			name:    "no_columns",
			op:      &CreateTable{TableOp: TableOp{Name: "users"}, Columns: []*ColumnDef{}},
			wantErr: true,
		},
		{
			name: "invalid_column",
			op: &CreateTable{
				TableOp: TableOp{Name: "users"},
				Columns: []*ColumnDef{{Name: "", Type: "string"}}, // Empty column name
			},
			wantErr: true,
		},
		{
			name: "invalid_index",
			op: &CreateTable{
				TableOp: TableOp{Name: "users"},
				Columns: []*ColumnDef{{Name: "id", Type: "id"}},
				Indexes: []*IndexDef{{Columns: []string{}}}, // No columns
			},
			wantErr: true,
		},
		{
			name: "invalid_foreign_key",
			op: &CreateTable{
				TableOp: TableOp{Name: "users"},
				Columns: []*ColumnDef{{Name: "id", Type: "id"}},
				ForeignKeys: []*ForeignKeyDef{{
					Columns:    []string{"ref_id"},
					RefTable:   "", // Missing ref table
					RefColumns: []string{"id"},
				}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// DropTable Tests
// -----------------------------------------------------------------------------

func TestDropTableType(t *testing.T) {
	op := &DropTable{TableOp: TableOp{Name: "users"}}
	if op.Type() != OpDropTable {
		t.Errorf("Type() = %v, want %v", op.Type(), OpDropTable)
	}
}

func TestDropTableTable(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		tableName string
		want      string
	}{
		{"with_namespace", "auth", "users", "auth_users"},
		{"no_namespace", "", "users", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &DropTable{TableOp: TableOp{Namespace: tt.namespace, Name: tt.tableName}}
			got := op.Table()
			if got != tt.want {
				t.Errorf("Table() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDropTableValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *DropTable
		wantErr bool
	}{
		{"valid", &DropTable{TableOp: TableOp{Name: "users"}}, false},
		{"valid_with_if_exists", &DropTable{TableOp: TableOp{Name: "users"}, IfExists: true}, false},
		{"empty_name", &DropTable{TableOp: TableOp{Name: ""}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// RenameTable Tests
// -----------------------------------------------------------------------------

func TestRenameTableType(t *testing.T) {
	op := &RenameTable{OldName: "old", NewName: "new"}
	if op.Type() != OpRenameTable {
		t.Errorf("Type() = %v, want %v", op.Type(), OpRenameTable)
	}
}

func TestRenameTableTable(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		oldName   string
		want      string
	}{
		{"with_namespace", "auth", "users", "auth_users"},
		{"no_namespace", "", "users", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &RenameTable{Namespace: tt.namespace, OldName: tt.oldName, NewName: "new"}
			got := op.Table()
			if got != tt.want {
				t.Errorf("Table() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenameTableValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *RenameTable
		wantErr bool
	}{
		{"valid", &RenameTable{OldName: "old", NewName: "new"}, false},
		{"empty_old_name", &RenameTable{OldName: "", NewName: "new"}, true},
		{"empty_new_name", &RenameTable{OldName: "old", NewName: ""}, true},
		{"same_names", &RenameTable{OldName: "same", NewName: "same"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// AddColumn Tests
// -----------------------------------------------------------------------------

func TestAddColumnType(t *testing.T) {
	op := &AddColumn{TableRef: TableRef{Table_: "users"}, Column: &ColumnDef{Name: "email", Type: "string"}}
	if op.Type() != OpAddColumn {
		t.Errorf("Type() = %v, want %v", op.Type(), OpAddColumn)
	}
}

func TestAddColumnTable(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		tableName string
		want      string
	}{
		{"with_namespace", "auth", "users", "auth_users"},
		{"no_namespace", "", "users", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &AddColumn{TableRef: TableRef{Namespace: tt.namespace, Table_: tt.tableName}}
			got := op.Table()
			if got != tt.want {
				t.Errorf("Table() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddColumnValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *AddColumn
		wantErr bool
	}{
		{
			name:    "valid",
			op:      &AddColumn{TableRef: TableRef{Table_: "users"}, Column: &ColumnDef{Name: "email", Type: "string"}},
			wantErr: false,
		},
		{
			name:    "empty_table",
			op:      &AddColumn{TableRef: TableRef{Table_: ""}, Column: &ColumnDef{Name: "email", Type: "string"}},
			wantErr: true,
		},
		{
			name:    "nil_column",
			op:      &AddColumn{TableRef: TableRef{Table_: "users"}, Column: nil},
			wantErr: true,
		},
		{
			name:    "invalid_column",
			op:      &AddColumn{TableRef: TableRef{Table_: "users"}, Column: &ColumnDef{Name: "", Type: "string"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// DropColumn Tests
// -----------------------------------------------------------------------------

func TestDropColumnType(t *testing.T) {
	op := &DropColumn{TableRef: TableRef{Table_: "users"}, Name: "email"}
	if op.Type() != OpDropColumn {
		t.Errorf("Type() = %v, want %v", op.Type(), OpDropColumn)
	}
}

func TestDropColumnTable(t *testing.T) {
	op := &DropColumn{TableRef: TableRef{Namespace: "auth", Table_: "users"}, Name: "email"}
	if op.Table() != "auth_users" {
		t.Errorf("Table() = %q, want %q", op.Table(), "auth_users")
	}
}

func TestDropColumnValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *DropColumn
		wantErr bool
	}{
		{"valid", &DropColumn{TableRef: TableRef{Table_: "users"}, Name: "email"}, false},
		{"empty_table", &DropColumn{TableRef: TableRef{Table_: ""}, Name: "email"}, true},
		{"empty_column", &DropColumn{TableRef: TableRef{Table_: "users"}, Name: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// RenameColumn Tests
// -----------------------------------------------------------------------------

func TestRenameColumnType(t *testing.T) {
	op := &RenameColumn{TableRef: TableRef{Table_: "users"}, OldName: "old", NewName: "new"}
	if op.Type() != OpRenameColumn {
		t.Errorf("Type() = %v, want %v", op.Type(), OpRenameColumn)
	}
}

func TestRenameColumnTable(t *testing.T) {
	op := &RenameColumn{TableRef: TableRef{Namespace: "auth", Table_: "users"}, OldName: "old", NewName: "new"}
	if op.Table() != "auth_users" {
		t.Errorf("Table() = %q, want %q", op.Table(), "auth_users")
	}
}

func TestRenameColumnValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *RenameColumn
		wantErr bool
	}{
		{"valid", &RenameColumn{TableRef: TableRef{Table_: "users"}, OldName: "old", NewName: "new"}, false},
		{"empty_table", &RenameColumn{TableRef: TableRef{Table_: ""}, OldName: "old", NewName: "new"}, true},
		{"empty_old_name", &RenameColumn{TableRef: TableRef{Table_: "users"}, OldName: "", NewName: "new"}, true},
		{"empty_new_name", &RenameColumn{TableRef: TableRef{Table_: "users"}, OldName: "old", NewName: ""}, true},
		{"same_names", &RenameColumn{TableRef: TableRef{Table_: "users"}, OldName: "same", NewName: "same"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// AlterColumn Tests
// -----------------------------------------------------------------------------

func TestAlterColumnType(t *testing.T) {
	op := &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email", NewType: "text"}
	if op.Type() != OpAlterColumn {
		t.Errorf("Type() = %v, want %v", op.Type(), OpAlterColumn)
	}
}

func TestAlterColumnTable(t *testing.T) {
	op := &AlterColumn{TableRef: TableRef{Namespace: "auth", Table_: "users"}, Name: "email"}
	if op.Table() != "auth_users" {
		t.Errorf("Table() = %q, want %q", op.Table(), "auth_users")
	}
}

func TestAlterColumnValidate(t *testing.T) {
	boolTrue := true
	boolFalse := false

	tests := []struct {
		name    string
		op      *AlterColumn
		wantErr bool
	}{
		{
			name:    "valid_new_type",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email", NewType: "text"},
			wantErr: false,
		},
		{
			name:    "valid_set_nullable",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email", SetNullable: &boolTrue},
			wantErr: false,
		},
		{
			name:    "valid_set_not_null",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email", SetNullable: &boolFalse},
			wantErr: false,
		},
		{
			name:    "valid_set_default",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email", SetDefault: "default@example.com"},
			wantErr: false,
		},
		{
			name:    "valid_drop_default",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email", DropDefault: true},
			wantErr: false,
		},
		{
			name:    "valid_server_default",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "created_at", ServerDefault: "NOW()"},
			wantErr: false,
		},
		{
			name:    "empty_table",
			op:      &AlterColumn{TableRef: TableRef{Table_: ""}, Name: "email", NewType: "text"},
			wantErr: true,
		},
		{
			name:    "empty_column",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "", NewType: "text"},
			wantErr: true,
		},
		{
			name:    "no_changes",
			op:      &AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "email"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// CreateIndex Tests
// -----------------------------------------------------------------------------

func TestCreateIndexType(t *testing.T) {
	op := &CreateIndex{TableRef: TableRef{Table_: "users"}, Columns: []string{"email"}}
	if op.Type() != OpCreateIndex {
		t.Errorf("Type() = %v, want %v", op.Type(), OpCreateIndex)
	}
}

func TestCreateIndexTable(t *testing.T) {
	op := &CreateIndex{TableRef: TableRef{Namespace: "auth", Table_: "users"}, Columns: []string{"email"}}
	if op.Table() != "auth_users" {
		t.Errorf("Table() = %q, want %q", op.Table(), "auth_users")
	}
}

func TestCreateIndexValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *CreateIndex
		wantErr bool
	}{
		{
			name:    "valid_single_column",
			op:      &CreateIndex{TableRef: TableRef{Table_: "users"}, Columns: []string{"email"}},
			wantErr: false,
		},
		{
			name:    "valid_multiple_columns",
			op:      &CreateIndex{TableRef: TableRef{Table_: "users"}, Columns: []string{"first_name", "last_name"}},
			wantErr: false,
		},
		{
			name:    "valid_unique",
			op:      &CreateIndex{TableRef: TableRef{Table_: "users"}, Columns: []string{"email"}, Unique: true},
			wantErr: false,
		},
		{
			name:    "valid_with_name",
			op:      &CreateIndex{TableRef: TableRef{Table_: "users"}, Name: "idx_custom", Columns: []string{"email"}},
			wantErr: false,
		},
		{
			name:    "empty_table",
			op:      &CreateIndex{TableRef: TableRef{Table_: ""}, Columns: []string{"email"}},
			wantErr: true,
		},
		{
			name:    "empty_columns",
			op:      &CreateIndex{TableRef: TableRef{Table_: "users"}, Columns: []string{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// DropIndex Tests
// -----------------------------------------------------------------------------

func TestDropIndexType(t *testing.T) {
	op := &DropIndex{Name: "idx_users_email"}
	if op.Type() != OpDropIndex {
		t.Errorf("Type() = %v, want %v", op.Type(), OpDropIndex)
	}
}

func TestDropIndexTable(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		tableName string
		want      string
	}{
		{"with_both", "auth", "users", "auth_users"},
		{"no_namespace", "", "users", "users"},
		{"no_table", "auth", "", "auth_"},
		{"nothing", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &DropIndex{TableRef: TableRef{Namespace: tt.namespace, Table_: tt.tableName}, Name: "idx"}
			got := op.Table()
			if got != tt.want {
				t.Errorf("Table() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDropIndexValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *DropIndex
		wantErr bool
	}{
		{"valid", &DropIndex{Name: "idx_users_email"}, false},
		{"valid_with_if_exists", &DropIndex{Name: "idx_users_email", IfExists: true}, false},
		{"empty_name", &DropIndex{Name: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// AddForeignKey Tests
// -----------------------------------------------------------------------------

func TestAddForeignKeyType(t *testing.T) {
	op := &AddForeignKey{TableRef: TableRef{Table_: "posts"}, Columns: []string{"user_id"}, RefTable: "users", RefColumns: []string{"id"}}
	if op.Type() != OpAddForeignKey {
		t.Errorf("Type() = %v, want %v", op.Type(), OpAddForeignKey)
	}
}

func TestAddForeignKeyTable(t *testing.T) {
	op := &AddForeignKey{TableRef: TableRef{Namespace: "blog", Table_: "posts"}}
	if op.Table() != "blog_posts" {
		t.Errorf("Table() = %q, want %q", op.Table(), "blog_posts")
	}
}

func TestAddForeignKeyValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *AddForeignKey
		wantErr bool
	}{
		{
			name: "valid",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "posts"},
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantErr: false,
		},
		{
			name: "valid_composite",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "order_items"},
				Columns:    []string{"order_id", "product_id"},
				RefTable:   "order_products",
				RefColumns: []string{"order_id", "product_id"},
			},
			wantErr: false,
		},
		{
			name: "valid_with_actions",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "posts"},
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
				OnUpdate:   "CASCADE",
			},
			wantErr: false,
		},
		{
			name: "empty_table",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: ""},
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "empty_columns",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "posts"},
				Columns:    []string{},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "empty_ref_table",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "posts"},
				Columns:    []string{"user_id"},
				RefTable:   "",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "empty_ref_columns",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "posts"},
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{},
			},
			wantErr: true,
		},
		{
			name: "mismatched_column_count",
			op: &AddForeignKey{
				TableRef:   TableRef{Table_: "posts"},
				Columns:    []string{"user_id", "extra_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// DropForeignKey Tests
// -----------------------------------------------------------------------------

func TestDropForeignKeyType(t *testing.T) {
	op := &DropForeignKey{TableRef: TableRef{Table_: "posts"}, Name: "fk_posts_user_id"}
	if op.Type() != OpDropForeignKey {
		t.Errorf("Type() = %v, want %v", op.Type(), OpDropForeignKey)
	}
}

func TestDropForeignKeyTable(t *testing.T) {
	op := &DropForeignKey{TableRef: TableRef{Namespace: "blog", Table_: "posts"}, Name: "fk"}
	if op.Table() != "blog_posts" {
		t.Errorf("Table() = %q, want %q", op.Table(), "blog_posts")
	}
}

func TestDropForeignKeyValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *DropForeignKey
		wantErr bool
	}{
		{"valid", &DropForeignKey{TableRef: TableRef{Table_: "posts"}, Name: "fk_posts_user_id"}, false},
		{"empty_table", &DropForeignKey{TableRef: TableRef{Table_: ""}, Name: "fk_posts_user_id"}, true},
		{"empty_name", &DropForeignKey{TableRef: TableRef{Table_: "posts"}, Name: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// RawSQL Tests
// -----------------------------------------------------------------------------

func TestRawSQLType(t *testing.T) {
	op := &RawSQL{SQL: "SELECT 1"}
	if op.Type() != OpRawSQL {
		t.Errorf("Type() = %v, want %v", op.Type(), OpRawSQL)
	}
}

func TestRawSQLTable(t *testing.T) {
	op := &RawSQL{SQL: "SELECT 1"}
	if op.Table() != "" {
		t.Errorf("Table() = %q, want empty string", op.Table())
	}
}

func TestRawSQLValidate(t *testing.T) {
	tests := []struct {
		name    string
		op      *RawSQL
		wantErr bool
	}{
		{"valid_sql", &RawSQL{SQL: "SELECT 1"}, false},
		{"valid_postgres_only", &RawSQL{Postgres: "SELECT 1"}, false},
		{"valid_sqlite_only", &RawSQL{SQLite: "SELECT 1"}, false},
		{"valid_all_dialects", &RawSQL{SQL: "SELECT 1", Postgres: "PG", SQLite: "SQ"}, false},
		{"empty_all", &RawSQL{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Error Code Tests
// -----------------------------------------------------------------------------

func TestOperationErrorCodes(t *testing.T) {
	// All validation errors should use ErrSchemaInvalid code
	invalidOps := []Operation{
		&CreateTable{TableOp: TableOp{Name: ""}},
		&DropTable{TableOp: TableOp{Name: ""}},
		&RenameTable{OldName: "", NewName: "new"},
		&AddColumn{TableRef: TableRef{Table_: ""}},
		&DropColumn{TableRef: TableRef{Table_: ""}},
		&RenameColumn{TableRef: TableRef{Table_: ""}, OldName: "old", NewName: "new"},
		&AlterColumn{TableRef: TableRef{Table_: "users"}, Name: "col"}, // No changes
		&CreateIndex{TableRef: TableRef{Table_: ""}},
		&DropIndex{Name: ""},
		&AddForeignKey{TableRef: TableRef{Table_: ""}},
		&DropForeignKey{TableRef: TableRef{Table_: ""}},
		&RawSQL{},
	}

	for _, op := range invalidOps {
		t.Run(op.Type().String(), func(t *testing.T) {
			err := op.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			code := alerr.GetErrorCode(err)
			if code != alerr.ErrSchemaInvalid {
				t.Errorf("error code = %v, want %v", code, alerr.ErrSchemaInvalid)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Operation Interface Tests
// -----------------------------------------------------------------------------

func TestOperationInterface(t *testing.T) {
	// Verify all operation types implement the Operation interface
	operations := []Operation{
		&CreateTable{TableOp: TableOp{Name: "t"}, Columns: []*ColumnDef{{Name: "id", Type: "id"}}},
		&DropTable{TableOp: TableOp{Name: "t"}},
		&RenameTable{OldName: "old", NewName: "new"},
		&AddColumn{TableRef: TableRef{Table_: "t"}, Column: &ColumnDef{Name: "c", Type: "string"}},
		&DropColumn{TableRef: TableRef{Table_: "t"}, Name: "c"},
		&RenameColumn{TableRef: TableRef{Table_: "t"}, OldName: "old", NewName: "new"},
		&AlterColumn{TableRef: TableRef{Table_: "t"}, Name: "c", NewType: "text"},
		&CreateIndex{TableRef: TableRef{Table_: "t"}, Columns: []string{"c"}},
		&DropIndex{Name: "idx"},
		&AddForeignKey{TableRef: TableRef{Table_: "t"}, Columns: []string{"c"}, RefTable: "r", RefColumns: []string{"id"}},
		&DropForeignKey{TableRef: TableRef{Table_: "t"}, Name: "fk"},
		&RawSQL{SQL: "SELECT 1"},
	}

	for _, op := range operations {
		t.Run(op.Type().String(), func(t *testing.T) {
			// Type() should return valid OpType
			opType := op.Type()
			if opType.String() == "Unknown" {
				t.Error("Type() returned Unknown")
			}

			// Table() should not panic
			_ = op.Table()

			// Validate() should not panic and return nil for valid ops
			err := op.Validate()
			if err != nil {
				t.Errorf("Validate() failed for valid operation: %v", err)
			}
		})
	}
}
