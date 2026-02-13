package dialect

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// TestPostgres_RenameTableSQL tests the RenameTableSQL method.
func TestPostgres_RenameTableSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name      string
		op        *ast.RenameTable
		wantSQL   string
		wantError bool
	}{
		{
			name: "simple table rename",
			op: &ast.RenameTable{
				Namespace: "public",
				OldName:   "users",
				NewName:   "customers",
			},
			wantSQL: `ALTER TABLE "public_users" RENAME TO "public_customers"`,
		},
		{
			name: "rename with special characters",
			op: &ast.RenameTable{
				Namespace: "test",
				OldName:   "old-table",
				NewName:   "new-table",
			},
			wantSQL: `ALTER TABLE "test_old-table" RENAME TO "test_new-table"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.RenameTableSQL(tt.op)
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", sql, tt.wantSQL)
			}
		})
	}
}

// TestPostgres_AddForeignKeySQL tests the AddForeignKeySQL method.
func TestPostgres_AddForeignKeySQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name      string
		op        *ast.AddForeignKey
		wantSQL   string
		wantError bool
	}{
		{
			name: "simple foreign key",
			op: &ast.AddForeignKey{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "orders",
				},
				Name:       "fk_orders_user_id",
				Columns:    []string{"user_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantSQL: `ALTER TABLE "public_orders" ADD CONSTRAINT "fk_orders_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id")`,
		},
		{
			name: "foreign key with ON DELETE CASCADE",
			op: &ast.AddForeignKey{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "posts",
				},
				Name:       "fk_posts_author",
				Columns:    []string{"author_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
			wantSQL: `ALTER TABLE "public_posts" ADD CONSTRAINT "fk_posts_author" FOREIGN KEY ("author_id") REFERENCES "users" ("id") ON DELETE CASCADE`,
		},
		{
			name: "foreign key with ON UPDATE CASCADE",
			op: &ast.AddForeignKey{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "comments",
				},
				Name:       "fk_comments_post",
				Columns:    []string{"post_id"},
				RefTable:   "posts",
				RefColumns: []string{"id"},
				OnUpdate:   "CASCADE",
			},
			wantSQL: `ALTER TABLE "public_comments" ADD CONSTRAINT "fk_comments_post" FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON UPDATE CASCADE`,
		},
		{
			name: "composite foreign key",
			op: &ast.AddForeignKey{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "order_items",
				},
				Name:       "fk_order_items_product",
				Columns:    []string{"product_id", "variant_id"},
				RefTable:   "product_variants",
				RefColumns: []string{"product_id", "id"},
			},
			wantSQL: `ALTER TABLE "public_order_items" ADD CONSTRAINT "fk_order_items_product" FOREIGN KEY ("product_id", "variant_id") REFERENCES "product_variants" ("product_id", "id")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.AddForeignKeySQL(tt.op)
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", sql, tt.wantSQL)
			}
		})
	}
}

// TestPostgres_DropForeignKeySQL tests the DropForeignKeySQL method.
func TestPostgres_DropForeignKeySQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name      string
		op        *ast.DropForeignKey
		wantSQL   string
		wantError bool
	}{
		{
			name: "drop foreign key",
			op: &ast.DropForeignKey{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "orders",
				},
				Name: "fk_orders_user_id",
			},
			wantSQL: `ALTER TABLE "public_orders" DROP CONSTRAINT "fk_orders_user_id"`,
		},
		{
			name: "drop foreign key with special chars",
			op: &ast.DropForeignKey{
				TableRef: ast.TableRef{
					Namespace: "test",
					Table_:    "my-table",
				},
				Name: "fk_my-constraint",
			},
			wantSQL: `ALTER TABLE "test_my-table" DROP CONSTRAINT "fk_my-constraint"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.DropForeignKeySQL(tt.op)
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", sql, tt.wantSQL)
			}
		})
	}
}

// TestPostgres_AddCheckSQL tests the AddCheckSQL method.
func TestPostgres_AddCheckSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name      string
		op        *ast.AddCheck
		wantSQL   string
		wantError bool
	}{
		{
			name: "simple check constraint",
			op: &ast.AddCheck{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "products",
				},
				Name:       "chk_positive_price",
				Expression: "price > 0",
			},
			wantSQL: `ALTER TABLE "public_products" ADD CONSTRAINT "chk_positive_price" CHECK (price > 0)`,
		},
		{
			name: "complex check constraint",
			op: &ast.AddCheck{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "users",
				},
				Name:       "chk_age_range",
				Expression: "age >= 18 AND age <= 120",
			},
			wantSQL: `ALTER TABLE "public_users" ADD CONSTRAINT "chk_age_range" CHECK (age >= 18 AND age <= 120)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.AddCheckSQL(tt.op)
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", sql, tt.wantSQL)
			}
		})
	}
}

// TestPostgres_DropCheckSQL tests the DropCheckSQL method.
func TestPostgres_DropCheckSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name      string
		op        *ast.DropCheck
		wantSQL   string
		wantError bool
	}{
		{
			name: "drop check constraint",
			op: &ast.DropCheck{
				TableRef: ast.TableRef{
					Namespace: "public",
					Table_:    "products",
				},
				Name: "chk_positive_price",
			},
			wantSQL: `ALTER TABLE "public_products" DROP CONSTRAINT "chk_positive_price"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.DropCheckSQL(tt.op)
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", sql, tt.wantSQL)
			}
		})
	}
}

// TestPostgres_RawSQLFor tests the RawSQLFor method.
func TestPostgres_RawSQLFor(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name      string
		op        *ast.RawSQL
		wantSQL   string
		wantError bool
	}{
		{
			name: "postgres-specific SQL",
			op: &ast.RawSQL{
				SQL:      "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
				Postgres: "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
			},
			wantSQL: `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		},
		{
			name: "fallback to generic SQL",
			op: &ast.RawSQL{
				SQL:      "SELECT 1",
				Postgres: "",
			},
			wantSQL: "SELECT 1",
		},
		{
			name: "prefer postgres-specific over generic",
			op: &ast.RawSQL{
				SQL:      "SELECT current_timestamp",
				Postgres: "SELECT NOW()",
			},
			wantSQL: "SELECT NOW()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.RawSQLFor(tt.op)
			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", sql, tt.wantSQL)
			}
		})
	}
}

// TestColumnDef_EnumValues tests the ColumnDef.EnumValues() method.
func TestColumnDef_EnumValues(t *testing.T) {

	tests := []struct {
		name     string
		typeArgs []any
		want     []string
	}{
		{
			name:     "empty type args",
			typeArgs: []any{},
			want:     nil,
		},
		{
			name:     "string slice at index 0",
			typeArgs: []any{[]string{"active", "inactive", "pending"}},
			want:     []string{"active", "inactive", "pending"},
		},
		{
			name:     "any slice at index 0",
			typeArgs: []any{[]any{"red", "green", "blue"}},
			want:     []string{"red", "green", "blue"},
		},
		{
			name:     "legacy format - string slice at index 1",
			typeArgs: []any{"status", []string{"draft", "published"}},
			want:     []string{"draft", "published"},
		},
		{
			name:     "legacy format - any slice at index 1",
			typeArgs: []any{"color", []any{"red", "blue"}},
			want:     []string{"red", "blue"},
		},
		{
			name:     "invalid type args",
			typeArgs: []any{123},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &ast.ColumnDef{Type: "enum", TypeArgs: tt.typeArgs}
			got := col.EnumValues()
			if len(got) != len(tt.want) {
				t.Errorf("length mismatch: got %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("value[%d] mismatch: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestPostgres_enumCheckSQL tests the enumCheckSQL helper method.
func TestPostgres_enumCheckSQL(t *testing.T) {
	d := &postgres{}

	tests := []struct {
		name      string
		col       *ast.ColumnDef
		tableName string
		wantSQL   string
	}{
		{
			name: "enum column with values",
			col: &ast.ColumnDef{
				Name:     "status",
				Type:     "enum",
				TypeArgs: []any{[]string{"active", "inactive"}},
			},
			tableName: "users",
			wantSQL:   ` CONSTRAINT "chk_users_status_enum" CHECK ("status" IN ('active', 'inactive'))`,
		},
		{
			name: "non-enum column",
			col: &ast.ColumnDef{
				Name: "name",
				Type: "string",
			},
			tableName: "users",
			wantSQL:   "",
		},
		{
			name: "enum with no values",
			col: &ast.ColumnDef{
				Name:     "status",
				Type:     "enum",
				TypeArgs: []any{},
			},
			tableName: "users",
			wantSQL:   "",
		},
		{
			name: "enum with special characters in values",
			col: &ast.ColumnDef{
				Name:     "role",
				Type:     "enum",
				TypeArgs: []any{[]string{"admin", "user's", "guest"}},
			},
			tableName: "accounts",
			wantSQL:   ` CONSTRAINT "chk_accounts_role_enum" CHECK ("role" IN ('admin', 'user''s', 'guest'))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.enumCheckSQL(tt.col, tt.tableName)
			if got != tt.wantSQL {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", got, tt.wantSQL)
			}
		})
	}
}

// TestPostgres_foreignKeyConstraintSQL tests the foreignKeyConstraintSQL helper.
func TestPostgres_foreignKeyConstraintSQL(t *testing.T) {
	d := &postgres{}

	tests := []struct {
		name    string
		fk      *ast.ForeignKeyDef
		wantSQL string
	}{
		{
			name: "simple foreign key",
			fk: &ast.ForeignKeyDef{
				Name:       "fk_posts_author",
				Columns:    []string{"author_id"},
				RefTable:   "users",
				RefColumns: []string{"id"},
			},
			wantSQL: `CONSTRAINT "fk_posts_author" FOREIGN KEY ("author_id") REFERENCES "users" ("id")`,
		},
		{
			name: "foreign key with ON DELETE CASCADE",
			fk: &ast.ForeignKeyDef{
				Name:       "fk_comments_post",
				Columns:    []string{"post_id"},
				RefTable:   "posts",
				RefColumns: []string{"id"},
				OnDelete:   "CASCADE",
			},
			wantSQL: `CONSTRAINT "fk_comments_post" FOREIGN KEY ("post_id") REFERENCES "posts" ("id") ON DELETE CASCADE`,
		},
		{
			name: "composite foreign key with actions",
			fk: &ast.ForeignKeyDef{
				Name:       "fk_order_items",
				Columns:    []string{"order_id", "item_id"},
				RefTable:   "orders",
				RefColumns: []string{"id", "item_id"},
				OnDelete:   "CASCADE",
				OnUpdate:   "RESTRICT",
			},
			wantSQL: `CONSTRAINT "fk_order_items" FOREIGN KEY ("order_id", "item_id") REFERENCES "orders" ("id", "item_id") ON DELETE CASCADE ON UPDATE RESTRICT`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.foreignKeyConstraintSQL(tt.fk)
			// Normalize whitespace for comparison
			gotNorm := strings.Join(strings.Fields(got), " ")
			wantNorm := strings.Join(strings.Fields(tt.wantSQL), " ")
			if gotNorm != wantNorm {
				t.Errorf("SQL mismatch\ngot:  %q\nwant: %q", got, tt.wantSQL)
			}
		})
	}
}
