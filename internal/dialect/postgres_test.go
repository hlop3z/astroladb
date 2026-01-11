package dialect

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestPostgresName(t *testing.T) {
	d := Postgres()
	if got := d.Name(); got != "postgres" {
		t.Errorf("Name() = %q, want %q", got, "postgres")
	}
}

// -----------------------------------------------------------------------------
// Type Mappings
// -----------------------------------------------------------------------------

func TestPostgresTypeMappings(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		typeFunc func() string
		want     string
	}{
		{"IDType", d.IDType, "UUID DEFAULT gen_random_uuid()"},
		{"TextType", d.TextType, "TEXT"},
		{"IntegerType", d.IntegerType, "INTEGER"},
		{"FloatType", d.FloatType, "REAL"},
		{"BooleanType", d.BooleanType, "BOOLEAN"},
		{"DateType", d.DateType, "DATE"},
		{"TimeType", d.TimeType, "TIME"},
		{"DateTimeType", d.DateTimeType, "TIMESTAMPTZ"},
		{"UUIDType", d.UUIDType, "UUID"},
		{"JSONType", d.JSONType, "JSONB"},
		{"Base64Type", d.Base64Type, "BYTEA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeFunc(); got != tt.want {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestPostgresStringType(t *testing.T) {
	d := Postgres()

	tests := []struct {
		length int
		want   string
	}{
		{50, "VARCHAR(50)"},
		{255, "VARCHAR(255)"},
		{1000, "VARCHAR(1000)"},
		{1, "VARCHAR(1)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := d.StringType(tt.length); got != tt.want {
				t.Errorf("StringType(%d) = %q, want %q", tt.length, got, tt.want)
			}
		})
	}
}

func TestPostgresDecimalType(t *testing.T) {
	d := Postgres()

	tests := []struct {
		precision int
		scale     int
		want      string
	}{
		{10, 2, "DECIMAL(10, 2)"},
		{18, 4, "DECIMAL(18, 4)"},
		{5, 0, "DECIMAL(5, 0)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := d.DecimalType(tt.precision, tt.scale); got != tt.want {
				t.Errorf("DecimalType(%d, %d) = %q, want %q", tt.precision, tt.scale, got, tt.want)
			}
		})
	}
}

func TestPostgresEnumType(t *testing.T) {
	d := Postgres()

	// PostgreSQL returns the type name directly (CREATE TYPE is separate)
	got := d.EnumType("status_type", []string{"active", "inactive"})
	want := "status_type"
	if got != want {
		t.Errorf("EnumType() = %q, want %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// Identifier Quoting
// -----------------------------------------------------------------------------

func TestPostgresQuoteIdent(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "users", `"users"`},
		{"with_underscore", "user_roles", `"user_roles"`},
		{"reserved_word", "select", `"select"`},
		{"with_double_quote", `table"name`, `"table""name"`},
		{"multiple_quotes", `a"b"c`, `"a""b""c"`},
		{"empty", "", `""`},
		{"with_space", "user name", `"user name"`},
		{"mixed_case", "UserName", `"UserName"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.QuoteIdent(tt.input); got != tt.want {
				t.Errorf("QuoteIdent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPostgresPlaceholder(t *testing.T) {
	d := Postgres()

	tests := []struct {
		index int
		want  string
	}{
		{1, "$1"},
		{2, "$2"},
		{10, "$10"},
		{100, "$100"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := d.Placeholder(tt.index); got != tt.want {
				t.Errorf("Placeholder(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Feature Support
// -----------------------------------------------------------------------------

func TestPostgresSupportsTransactionalDDL(t *testing.T) {
	d := Postgres()
	if got := d.SupportsTransactionalDDL(); !got {
		t.Errorf("SupportsTransactionalDDL() = %v, want true", got)
	}
}

func TestPostgresSupportsIfExists(t *testing.T) {
	d := Postgres()
	if got := d.SupportsIfExists(); !got {
		t.Errorf("SupportsIfExists() = %v, want true", got)
	}
}

// -----------------------------------------------------------------------------
// SQL Generation
// -----------------------------------------------------------------------------

func TestPostgresCreateTableSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		op       *ast.CreateTable
		contains []string
	}{
		{
			name: "simple_table",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}},
				},
			},
			contains: []string{
				`CREATE TABLE "auth_users"`,
				`"id" UUID DEFAULT gen_random_uuid() PRIMARY KEY`,
				`"email" VARCHAR(255) NOT NULL`,
			},
		},
		{
			name: "with_nullable_and_default",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "blog", Name: "posts"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "title", Type: "string", TypeArgs: []any{255}},
					{Name: "views", Type: "integer", Default: 0, DefaultSet: true},
					{Name: "deleted_at", Type: "datetime", Nullable: true, NullableSet: true},
				},
			},
			contains: []string{
				`CREATE TABLE "blog_posts"`,
				`"views" INTEGER NOT NULL DEFAULT 0`,
				`"deleted_at" TIMESTAMPTZ NULL`,
			},
		},
		{
			name: "with_foreign_key",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "blog", Name: "comments"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "post_id", Type: "uuid", Reference: &ast.Reference{
						Table:    "blog_posts",
						Column:   "id",
						OnDelete: "CASCADE",
					}},
				},
			},
			contains: []string{
				`CREATE TABLE "blog_comments"`,
				`"post_id" UUID NOT NULL REFERENCES "blog_posts"("id") ON DELETE CASCADE`,
			},
		},
		{
			name: "with_unique_constraint",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "email", Type: "string", TypeArgs: []any{255}, Unique: true},
				},
			},
			contains: []string{
				// Unique constraints are now explicitly named for rollback support (Issue #3)
				`"email" VARCHAR(255) NOT NULL CONSTRAINT "uniq_auth_users_email" UNIQUE`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.CreateTableSQL(tt.op)
			if err != nil {
				t.Fatalf("CreateTableSQL() error = %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(sql, want) {
					t.Errorf("CreateTableSQL() SQL missing %q\nGot:\n%s", want, sql)
				}
			}
		})
	}
}

func TestPostgresDropTableSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name string
		op   *ast.DropTable
		want string
	}{
		{
			name: "simple",
			op:   &ast.DropTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}},
			want: `DROP TABLE "auth_users"`,
		},
		{
			name: "with_if_exists",
			op:   &ast.DropTable{TableOp: ast.TableOp{Namespace: "auth", Name: "users"}, IfExists: true},
			want: `DROP TABLE IF EXISTS "auth_users"`,
		},
		{
			name: "no_namespace",
			op:   &ast.DropTable{TableOp: ast.TableOp{Name: "users"}},
			want: `DROP TABLE "users"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.DropTableSQL(tt.op)
			if err != nil {
				t.Fatalf("DropTableSQL() error = %v", err)
			}
			if sql != tt.want {
				t.Errorf("DropTableSQL() = %q, want %q", sql, tt.want)
			}
		})
	}
}

func TestPostgresAddColumnSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		op       *ast.AddColumn
		contains []string
	}{
		{
			name: "simple_column",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Column:   &ast.ColumnDef{Name: "phone", Type: "string", TypeArgs: []any{20}},
			},
			contains: []string{
				`ALTER TABLE "auth_users" ADD COLUMN`,
				`"phone" VARCHAR(20) NOT NULL`,
			},
		},
		{
			name: "nullable_column",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Column:   &ast.ColumnDef{Name: "bio", Type: "text", Nullable: true, NullableSet: true},
			},
			contains: []string{
				`ALTER TABLE "auth_users" ADD COLUMN`,
				`"bio" TEXT NULL`,
			},
		},
		{
			name: "with_default",
			op: &ast.AddColumn{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Column:   &ast.ColumnDef{Name: "active", Type: "boolean", Default: true, DefaultSet: true},
			},
			contains: []string{
				`ALTER TABLE "auth_users" ADD COLUMN`,
				`"active" BOOLEAN NOT NULL DEFAULT TRUE`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.AddColumnSQL(tt.op)
			if err != nil {
				t.Fatalf("AddColumnSQL() error = %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(sql, want) {
					t.Errorf("AddColumnSQL() SQL missing %q\nGot: %s", want, sql)
				}
			}
		})
	}
}

func TestPostgresDropColumnSQL(t *testing.T) {
	d := Postgres()

	op := &ast.DropColumn{
		TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
		Name:     "phone",
	}

	sql, err := d.DropColumnSQL(op)
	if err != nil {
		t.Fatalf("DropColumnSQL() error = %v", err)
	}

	want := `ALTER TABLE "auth_users" DROP COLUMN "phone"`
	if sql != want {
		t.Errorf("DropColumnSQL() = %q, want %q", sql, want)
	}
}

func TestPostgresRenameColumnSQL(t *testing.T) {
	d := Postgres()

	op := &ast.RenameColumn{
		TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
		OldName:  "email",
		NewName:  "email_address",
	}

	sql, err := d.RenameColumnSQL(op)
	if err != nil {
		t.Fatalf("RenameColumnSQL() error = %v", err)
	}

	want := `ALTER TABLE "auth_users" RENAME COLUMN "email" TO "email_address"`
	if sql != want {
		t.Errorf("RenameColumnSQL() = %q, want %q", sql, want)
	}
}

func TestPostgresAlterColumnSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		op       *ast.AlterColumn
		contains []string
	}{
		{
			name: "change_type",
			op: &ast.AlterColumn{
				TableRef:    ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:        "status",
				NewType:     "string",
				NewTypeArgs: []any{100},
			},
			contains: []string{
				`ALTER TABLE "auth_users" ALTER COLUMN "status" TYPE VARCHAR(100)`,
			},
		},
		{
			name: "set_not_null",
			op: &ast.AlterColumn{
				TableRef:    ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:        "email",
				SetNullable: ptrBool(false),
			},
			contains: []string{
				`ALTER TABLE "auth_users" ALTER COLUMN "email" SET NOT NULL`,
			},
		},
		{
			name: "drop_not_null",
			op: &ast.AlterColumn{
				TableRef:    ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:        "phone",
				SetNullable: ptrBool(true),
			},
			contains: []string{
				`ALTER TABLE "auth_users" ALTER COLUMN "phone" DROP NOT NULL`,
			},
		},
		{
			name: "set_default",
			op: &ast.AlterColumn{
				TableRef:   ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:       "role",
				SetDefault: "user",
			},
			contains: []string{
				`ALTER TABLE "auth_users" ALTER COLUMN "role" SET DEFAULT 'user'`,
			},
		},
		{
			name: "drop_default",
			op: &ast.AlterColumn{
				TableRef:    ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:        "role",
				DropDefault: true,
			},
			contains: []string{
				`ALTER TABLE "auth_users" ALTER COLUMN "role" DROP DEFAULT`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.AlterColumnSQL(tt.op)
			if err != nil {
				t.Fatalf("AlterColumnSQL() error = %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(sql, want) {
					t.Errorf("AlterColumnSQL() SQL missing %q\nGot: %s", want, sql)
				}
			}
		})
	}
}

func TestPostgresCreateIndexSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name     string
		op       *ast.CreateIndex
		contains []string
	}{
		{
			name: "simple_index",
			op: &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:     "idx_users_email",
				Columns:  []string{"email"},
			},
			contains: []string{
				`CREATE INDEX`,
				`"idx_users_email"`,
				`ON "auth_users"`,
				`("email")`,
			},
		},
		{
			name: "unique_index",
			op: &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:     "uniq_users_email",
				Columns:  []string{"email"},
				Unique:   true,
			},
			contains: []string{
				`CREATE UNIQUE INDEX`,
				`"uniq_users_email"`,
			},
		},
		{
			name: "composite_index",
			op: &ast.CreateIndex{
				TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
				Columns:  []string{"first_name", "last_name"},
			},
			contains: []string{
				`("first_name", "last_name")`,
			},
		},
		{
			name: "with_if_not_exists",
			op: &ast.CreateIndex{
				TableRef:    ast.TableRef{Namespace: "auth", Table_: "users"},
				Name:        "idx_users_email",
				Columns:     []string{"email"},
				IfNotExists: true,
			},
			contains: []string{
				`CREATE INDEX IF NOT EXISTS`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.CreateIndexSQL(tt.op)
			if err != nil {
				t.Fatalf("CreateIndexSQL() error = %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(sql, want) {
					t.Errorf("CreateIndexSQL() SQL missing %q\nGot: %s", want, sql)
				}
			}
		})
	}
}

func TestPostgresDropIndexSQL(t *testing.T) {
	d := Postgres()

	tests := []struct {
		name string
		op   *ast.DropIndex
		want string
	}{
		{
			name: "simple",
			op:   &ast.DropIndex{TableRef: ast.TableRef{}, Name: "idx_users_email"},
			want: `DROP INDEX "idx_users_email"`,
		},
		{
			name: "with_if_exists",
			op:   &ast.DropIndex{TableRef: ast.TableRef{}, Name: "idx_users_email", IfExists: true},
			want: `DROP INDEX IF EXISTS "idx_users_email"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := d.DropIndexSQL(tt.op)
			if err != nil {
				t.Fatalf("DropIndexSQL() error = %v", err)
			}
			if sql != tt.want {
				t.Errorf("DropIndexSQL() = %q, want %q", sql, tt.want)
			}
		})
	}
}

// Helper function
func ptrBool(b bool) *bool {
	return &b
}
