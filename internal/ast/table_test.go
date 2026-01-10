package ast

import (
	"testing"
)

// -----------------------------------------------------------------------------
// Reference Tests
// -----------------------------------------------------------------------------

func TestReferenceTargetColumn(t *testing.T) {
	tests := []struct {
		name string
		ref  *Reference
		want string
	}{
		{
			name: "default_to_id_when_empty",
			ref:  &Reference{Table: "auth.user", Column: ""},
			want: "id",
		},
		{
			name: "explicit_column",
			ref:  &Reference{Table: "auth.user", Column: "uuid"},
			want: "uuid",
		},
		{
			name: "custom_column",
			ref:  &Reference{Table: "billing.invoice", Column: "invoice_number"},
			want: "invoice_number",
		},
		{
			name: "nil_safe_column_empty",
			ref:  &Reference{Table: "auth.user"},
			want: "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.TargetColumn()
			if got != tt.want {
				t.Errorf("TargetColumn() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReferenceValidate(t *testing.T) {
	tests := []struct {
		name    string
		ref     *Reference
		wantErr bool
	}{
		{
			name:    "valid_with_table_only",
			ref:     &Reference{Table: "auth.user"},
			wantErr: false,
		},
		{
			name:    "valid_with_table_and_column",
			ref:     &Reference{Table: "auth.user", Column: "id"},
			wantErr: false,
		},
		{
			name:    "valid_with_on_delete",
			ref:     &Reference{Table: "auth.user", OnDelete: "CASCADE"},
			wantErr: false,
		},
		{
			name:    "valid_with_on_update",
			ref:     &Reference{Table: "auth.user", OnUpdate: "SET NULL"},
			wantErr: false,
		},
		{
			name:    "valid_with_all_options",
			ref:     &Reference{Table: "auth.user", Column: "id", OnDelete: "CASCADE", OnUpdate: "CASCADE"},
			wantErr: false,
		},
		{
			name:    "invalid_empty_table",
			ref:     &Reference{Table: ""},
			wantErr: true,
		},
		{
			name:    "invalid_empty_table_with_column",
			ref:     &Reference{Table: "", Column: "id"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// TableDef Tests
// -----------------------------------------------------------------------------

func TestTableDefFullName(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		tableName string
		want      string
	}{
		{"with_namespace", "auth", "user", "auth_user"},
		{"no_namespace", "", "user", "user"},
		{"empty_both", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &TableDef{Namespace: tt.namespace, Name: tt.tableName}
			got := table.FullName()
			if got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTableDefSQLName(t *testing.T) {
	table := &TableDef{Namespace: "auth", Name: "user"}
	got := table.SQLName()
	want := "auth_user"
	if got != want {
		t.Errorf("SQLName() = %q, want %q", got, want)
	}
}

func TestTableDefQualifiedName(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		tableName string
		want      string
	}{
		{"with_namespace", "auth", "user", "auth.user"},
		{"no_namespace", "", "user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &TableDef{Namespace: tt.namespace, Name: tt.tableName}
			got := table.QualifiedName()
			if got != tt.want {
				t.Errorf("QualifiedName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTableDefGetColumn(t *testing.T) {
	table := &TableDef{
		Namespace: "auth",
		Name:      "user",
		Columns: []*ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "email", Type: "string"},
			{Name: "username", Type: "string"},
		},
	}

	tests := []struct {
		name     string
		colName  string
		wantNil  bool
		wantName string
	}{
		{"existing_column", "email", false, "email"},
		{"first_column", "id", false, "id"},
		{"last_column", "username", false, "username"},
		{"non_existing", "password", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := table.GetColumn(tt.colName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetColumn(%q) = %v, want nil", tt.colName, got)
				}
			} else {
				if got == nil {
					t.Errorf("GetColumn(%q) = nil, want non-nil", tt.colName)
				} else if got.Name != tt.wantName {
					t.Errorf("GetColumn(%q).Name = %q, want %q", tt.colName, got.Name, tt.wantName)
				}
			}
		})
	}
}

func TestTableDefHasColumn(t *testing.T) {
	table := &TableDef{
		Columns: []*ColumnDef{
			{Name: "id", Type: "id"},
			{Name: "email", Type: "string"},
		},
	}

	tests := []struct {
		name    string
		colName string
		want    bool
	}{
		{"existing", "email", true},
		{"non_existing", "password", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := table.HasColumn(tt.colName)
			if got != tt.want {
				t.Errorf("HasColumn(%q) = %v, want %v", tt.colName, got, tt.want)
			}
		})
	}
}

func TestTableDefPrimaryKey(t *testing.T) {
	tests := []struct {
		name    string
		columns []*ColumnDef
		wantNil bool
		wantPK  string
	}{
		{
			name: "has_primary_key",
			columns: []*ColumnDef{
				{Name: "id", Type: "id", PrimaryKey: true},
				{Name: "email", Type: "string"},
			},
			wantNil: false,
			wantPK:  "id",
		},
		{
			name: "no_primary_key",
			columns: []*ColumnDef{
				{Name: "email", Type: "string"},
				{Name: "username", Type: "string"},
			},
			wantNil: true,
		},
		{
			name:    "empty_columns",
			columns: []*ColumnDef{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := &TableDef{Columns: tt.columns}
			got := table.PrimaryKey()
			if tt.wantNil {
				if got != nil {
					t.Errorf("PrimaryKey() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("PrimaryKey() = nil, want %q", tt.wantPK)
				} else if got.Name != tt.wantPK {
					t.Errorf("PrimaryKey().Name = %q, want %q", got.Name, tt.wantPK)
				}
			}
		})
	}
}

func TestTableDefValidate(t *testing.T) {
	tests := []struct {
		name    string
		table   *TableDef
		wantErr bool
	}{
		{
			name: "valid_simple",
			table: &TableDef{
				Name: "user",
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_with_namespace",
			table: &TableDef{
				Namespace: "auth",
				Name:      "user",
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
					{Name: "email", Type: "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_no_name",
			table: &TableDef{
				Name: "",
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_no_columns",
			table: &TableDef{
				Name:    "user",
				Columns: []*ColumnDef{},
			},
			wantErr: true,
		},
		{
			name: "invalid_duplicate_columns",
			table: &TableDef{
				Name: "user",
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
					{Name: "id", Type: "string"}, // duplicate
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_column",
			table: &TableDef{
				Name: "user",
				Columns: []*ColumnDef{
					{Name: "", Type: "id"}, // empty name
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_index",
			table: &TableDef{
				Name: "user",
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
				},
				Indexes: []*IndexDef{
					{Columns: []string{}}, // no columns
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_foreign_key",
			table: &TableDef{
				Name: "user",
				Columns: []*ColumnDef{
					{Name: "id", Type: "id"},
				},
				ForeignKeys: []*ForeignKeyDef{
					{Columns: []string{"ref_id"}, RefTable: "", RefColumns: []string{"id"}}, // no ref table
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.table.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ColumnDef Tests
// -----------------------------------------------------------------------------

func TestColumnDefValidate(t *testing.T) {
	tests := []struct {
		name    string
		col     *ColumnDef
		wantErr bool
	}{
		{
			name:    "valid_simple",
			col:     &ColumnDef{Name: "email", Type: "string"},
			wantErr: false,
		},
		{
			name:    "valid_with_constraints",
			col:     &ColumnDef{Name: "age", Type: "integer", Min: intPtr(0), Max: intPtr(150)},
			wantErr: false,
		},
		{
			name:    "invalid_no_name",
			col:     &ColumnDef{Name: "", Type: "string"},
			wantErr: true,
		},
		{
			name:    "invalid_no_type",
			col:     &ColumnDef{Name: "email", Type: ""},
			wantErr: true,
		},
		{
			name:    "invalid_min_greater_than_max",
			col:     &ColumnDef{Name: "age", Type: "integer", Min: intPtr(100), Max: intPtr(10)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.col.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestColumnDefIsNullable(t *testing.T) {
	tests := []struct {
		name     string
		col      *ColumnDef
		wantNull bool
	}{
		{"default_not_null", &ColumnDef{Name: "email", Type: "string"}, false},
		{"explicit_nullable", &ColumnDef{Name: "email", Type: "string", Nullable: true}, true},
		{"explicit_not_null", &ColumnDef{Name: "email", Type: "string", Nullable: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.col.IsNullable()
			if got != tt.wantNull {
				t.Errorf("IsNullable() = %v, want %v", got, tt.wantNull)
			}
		})
	}
}

func TestColumnDefHasDefault(t *testing.T) {
	tests := []struct {
		name string
		col  *ColumnDef
		want bool
	}{
		{"no_default", &ColumnDef{Name: "email", Type: "string"}, false},
		{"has_default_set", &ColumnDef{Name: "active", Type: "boolean", DefaultSet: true, Default: true}, true},
		{"has_server_default", &ColumnDef{Name: "created_at", Type: "datetime", ServerDefault: "NOW()"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.col.HasDefault()
			if got != tt.want {
				t.Errorf("HasDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumnDefHasBackfill(t *testing.T) {
	tests := []struct {
		name string
		col  *ColumnDef
		want bool
	}{
		{"no_backfill", &ColumnDef{Name: "email", Type: "string"}, false},
		{"has_backfill_set", &ColumnDef{Name: "age", Type: "integer", BackfillSet: true, Backfill: 0}, true},
		{"has_backfill_value_non_nil", &ColumnDef{Name: "age", Type: "integer", Backfill: 25}, true}, // non-nil value
		{"has_backfill_zero_int", &ColumnDef{Name: "age", Type: "integer", Backfill: 0}, true},       // 0 is non-nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.col.HasBackfill()
			if got != tt.want {
				t.Errorf("HasBackfill() = %v, want %v", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// SQLExpr Tests
// -----------------------------------------------------------------------------

func TestSQLExpr(t *testing.T) {
	expr := NewSQLExpr("NOW()")
	if expr.Expr != "NOW()" {
		t.Errorf("NewSQLExpr().Expr = %q, want %q", expr.Expr, "NOW()")
	}
}

func TestIsSQLExpr(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"sql_expr", NewSQLExpr("NOW()"), true},
		{"string", "NOW()", false},
		{"int", 42, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSQLExpr(tt.val)
			if got != tt.want {
				t.Errorf("IsSQLExpr(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestAsSQLExpr(t *testing.T) {
	tests := []struct {
		name   string
		val    any
		wantOK bool
	}{
		{"sql_expr", NewSQLExpr("NOW()"), true},
		{"string", "NOW()", false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, ok := AsSQLExpr(tt.val)
			if ok != tt.wantOK {
				t.Errorf("AsSQLExpr(%v) ok = %v, want %v", tt.val, ok, tt.wantOK)
			}
			if tt.wantOK && expr == nil {
				t.Errorf("AsSQLExpr(%v) returned nil expr with ok=true", tt.val)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// IndexDef Tests
// -----------------------------------------------------------------------------

func TestIndexDefValidate(t *testing.T) {
	tests := []struct {
		name    string
		idx     *IndexDef
		wantErr bool
	}{
		{
			name:    "valid_single_column",
			idx:     &IndexDef{Columns: []string{"email"}},
			wantErr: false,
		},
		{
			name:    "valid_multiple_columns",
			idx:     &IndexDef{Columns: []string{"first_name", "last_name"}},
			wantErr: false,
		},
		{
			name:    "valid_unique",
			idx:     &IndexDef{Columns: []string{"email"}, Unique: true},
			wantErr: false,
		},
		{
			name:    "valid_with_name",
			idx:     &IndexDef{Name: "idx_users_email", Columns: []string{"email"}},
			wantErr: false,
		},
		{
			name:    "invalid_no_columns",
			idx:     &IndexDef{Columns: []string{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.idx.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ForeignKeyDef Tests
// -----------------------------------------------------------------------------

func TestForeignKeyDefValidate(t *testing.T) {
	tests := []struct {
		name    string
		fk      *ForeignKeyDef
		wantErr bool
	}{
		{
			name: "valid_single_column",
			fk: &ForeignKeyDef{
				Columns:    []string{"user_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
			},
			wantErr: false,
		},
		{
			name: "valid_composite",
			fk: &ForeignKeyDef{
				Columns:    []string{"order_id", "product_id"},
				RefTable:   "order_product",
				RefColumns: []string{"order_id", "product_id"},
			},
			wantErr: false,
		},
		{
			name: "valid_with_actions",
			fk: &ForeignKeyDef{
				Columns:    []string{"user_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
				OnUpdate:   "CASCADE",
			},
			wantErr: false,
		},
		{
			name: "invalid_no_columns",
			fk: &ForeignKeyDef{
				Columns:    []string{},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "invalid_no_ref_table",
			fk: &ForeignKeyDef{
				Columns:    []string{"user_id"},
				RefTable:   "",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
		{
			name: "invalid_no_ref_columns",
			fk: &ForeignKeyDef{
				Columns:    []string{"user_id"},
				RefTable:   "auth_user",
				RefColumns: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid_column_count_mismatch",
			fk: &ForeignKeyDef{
				Columns:    []string{"user_id", "extra_id"},
				RefTable:   "auth_user",
				RefColumns: []string{"id"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fk.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// CheckDef Tests
// -----------------------------------------------------------------------------

func TestCheckDefValidate(t *testing.T) {
	tests := []struct {
		name    string
		check   *CheckDef
		wantErr bool
	}{
		{
			name:    "valid",
			check:   &CheckDef{Expression: "age >= 0"},
			wantErr: false,
		},
		{
			name:    "valid_with_name",
			check:   &CheckDef{Name: "chk_age_positive", Expression: "age >= 0"},
			wantErr: false,
		},
		{
			name:    "invalid_no_expression",
			check:   &CheckDef{Expression: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.check.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func intPtr(i int) *int {
	return &i
}
