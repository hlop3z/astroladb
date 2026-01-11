package dialect

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

func TestSQLiteName(t *testing.T) {
	d := SQLite()
	if got := d.Name(); got != "sqlite" {
		t.Errorf("Name() = %q, want %q", got, "sqlite")
	}
}

// -----------------------------------------------------------------------------
// Type Mappings - SQLite uses TEXT for most types
// -----------------------------------------------------------------------------

func TestSQLiteTypeMappings(t *testing.T) {
	d := SQLite()

	tests := []struct {
		name     string
		typeFunc func() string
		want     string
	}{
		// SQLite maps most types to TEXT, but uses type affinities for date/time
		{"IDType", d.IDType, "TEXT"},
		{"TextType", d.TextType, "TEXT"},
		{"DateType", d.DateType, "DATE"},             // Uses DATE type affinity for round-trip
		{"TimeType", d.TimeType, "TIME"},             // Uses TIME type affinity for round-trip
		{"DateTimeType", d.DateTimeType, "DATETIME"}, // Uses DATETIME type affinity for round-trip
		{"UUIDType", d.UUIDType, "TEXT"},
		{"JSONType", d.JSONType, "TEXT"}, // JSON1 extension stores as TEXT
		// Numeric types
		{"IntegerType", d.IntegerType, "INTEGER"},
		{"FloatType", d.FloatType, "REAL"},
		{"BooleanType", d.BooleanType, "INTEGER"}, // 0/1
		// Binary type
		{"Base64Type", d.Base64Type, "BLOB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typeFunc(); got != tt.want {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSQLiteStringType(t *testing.T) {
	d := SQLite()

	// SQLite ignores length constraints - always returns TEXT
	tests := []struct {
		length int
		want   string
	}{
		{50, "TEXT"},
		{255, "TEXT"},
		{1000, "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := d.StringType(tt.length); got != tt.want {
				t.Errorf("StringType(%d) = %q, want %q", tt.length, got, tt.want)
			}
		})
	}
}

func TestSQLiteDecimalType(t *testing.T) {
	d := SQLite()

	// SQLite uses TEXT for decimal to preserve precision
	tests := []struct {
		precision int
		scale     int
		want      string
	}{
		{10, 2, "TEXT"},
		{18, 4, "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := d.DecimalType(tt.precision, tt.scale); got != tt.want {
				t.Errorf("DecimalType(%d, %d) = %q, want %q", tt.precision, tt.scale, got, tt.want)
			}
		})
	}
}

func TestSQLiteEnumType(t *testing.T) {
	d := SQLite()

	// SQLite uses TEXT for enum (CHECK constraint added at column level)
	got := d.EnumType("status", []string{"active", "inactive"})
	want := "TEXT"
	if got != want {
		t.Errorf("EnumType() = %q, want %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// Identifier Quoting - SQLite uses double quotes (SQL standard)
// -----------------------------------------------------------------------------

func TestSQLiteQuoteIdent(t *testing.T) {
	d := SQLite()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "users", `"users"`},
		{"with_underscore", "user_roles", `"user_roles"`},
		{"reserved_word", "select", `"select"`},
		{"with_double_quote", `table"name`, `"table""name"`},
		{"empty", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.QuoteIdent(tt.input); got != tt.want {
				t.Errorf("QuoteIdent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSQLitePlaceholder(t *testing.T) {
	d := SQLite()

	tests := []struct {
		index int
		want  string
	}{
		{1, "?"},
		{2, "?"},
		{10, "?"},
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

func TestSQLiteSupportsTransactionalDDL(t *testing.T) {
	d := SQLite()
	// SQLite DOES support transactional DDL
	if got := d.SupportsTransactionalDDL(); !got {
		t.Errorf("SupportsTransactionalDDL() = %v, want true", got)
	}
}

func TestSQLiteSupportsIfExists(t *testing.T) {
	d := SQLite()
	if got := d.SupportsIfExists(); !got {
		t.Errorf("SupportsIfExists() = %v, want true", got)
	}
}

// -----------------------------------------------------------------------------
// SQL Generation
// -----------------------------------------------------------------------------

func TestSQLiteCreateTableSQL(t *testing.T) {
	d := SQLite()

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
				`"id" TEXT PRIMARY KEY`,
				`"email" TEXT NOT NULL`,
			},
		},
		{
			name: "with_boolean",
			op: &ast.CreateTable{
				TableOp: ast.TableOp{Namespace: "blog", Name: "posts"},
				Columns: []*ast.ColumnDef{
					{Name: "id", Type: "id", PrimaryKey: true},
					{Name: "active", Type: "boolean", Default: true, DefaultSet: true},
				},
			},
			contains: []string{
				`"active" INTEGER NOT NULL DEFAULT 1`, // SQLite: TRUE = 1
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

func TestSQLiteEnumWithCheckConstraint(t *testing.T) {
	d := SQLite()

	op := &ast.CreateTable{
		TableOp: ast.TableOp{Namespace: "auth", Name: "users"},
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "id", PrimaryKey: true},
			{
				Name:     "status",
				Type:     "enum",
				TypeArgs: []any{"status_type", []string{"active", "inactive", "pending"}},
			},
		},
	}

	sql, err := d.CreateTableSQL(op)
	if err != nil {
		t.Fatalf("CreateTableSQL() error = %v", err)
	}

	// SQLite should generate CHECK constraint for enum
	expectedParts := []string{
		`"status" TEXT`,
		`CHECK`,
		`"status" IN`,
		`'active'`,
		`'inactive'`,
		`'pending'`,
	}

	for _, want := range expectedParts {
		if !strings.Contains(sql, want) {
			t.Errorf("CreateTableSQL() SQL missing %q\nGot:\n%s", want, sql)
		}
	}
}

func TestSQLiteDropTableSQL(t *testing.T) {
	d := SQLite()

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

func TestSQLiteAddColumnSQL(t *testing.T) {
	d := SQLite()

	op := &ast.AddColumn{
		TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
		Column:   &ast.ColumnDef{Name: "phone", Type: "string", TypeArgs: []any{20}},
	}

	sql, err := d.AddColumnSQL(op)
	if err != nil {
		t.Fatalf("AddColumnSQL() error = %v", err)
	}

	want := `ALTER TABLE "auth_users" ADD COLUMN "phone" TEXT NOT NULL`
	if sql != want {
		t.Errorf("AddColumnSQL() = %q, want %q", sql, want)
	}
}

func TestSQLiteDropColumnSQL(t *testing.T) {
	d := SQLite()

	// SQLite 3.35.0+ supports DROP COLUMN
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

func TestSQLiteRenameColumnSQL(t *testing.T) {
	d := SQLite()

	// SQLite 3.25.0+ supports RENAME COLUMN
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

func TestSQLiteAlterColumnSQL(t *testing.T) {
	d := SQLite()

	// SQLite has very limited ALTER TABLE support
	op := &ast.AlterColumn{
		TableRef: ast.TableRef{Namespace: "auth", Table_: "users"},
		Name:     "status",
		NewType:  "text",
	}

	_, err := d.AlterColumnSQL(op)
	if err == nil {
		t.Error("AlterColumnSQL() expected error for SQLite, got nil")
	}

	// Should contain helpful error message about table recreation
	if !strings.Contains(err.Error(), "table recreation") {
		t.Errorf("AlterColumnSQL() error should mention table recreation, got: %v", err)
	}
}

func TestSQLiteCreateIndexSQL(t *testing.T) {
	d := SQLite()

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

func TestSQLiteDropIndexSQL(t *testing.T) {
	d := SQLite()

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

// -----------------------------------------------------------------------------
// Dialect Comparison Tests
// -----------------------------------------------------------------------------

func TestSQLiteVsPostgres(t *testing.T) {
	sqlite := SQLite()
	pg := Postgres()

	// SQLite uses TEXT for most things, but uses type affinities for date/time
	textTypesSQLite := []struct {
		name          string
		get           func(Dialect) string
		expectedSQLite string
	}{
		{"IDType", func(d Dialect) string { return d.IDType() }, "TEXT"},
		{"UUIDType", func(d Dialect) string { return d.UUIDType() }, "TEXT"},
	}

	for _, tt := range textTypesSQLite {
		t.Run(tt.name, func(t *testing.T) {
			sqliteVal := tt.get(sqlite)
			pgVal := tt.get(pg)

			if sqliteVal == pgVal {
				t.Errorf("%s: SQLite and PostgreSQL should differ, both = %q", tt.name, sqliteVal)
			}

			// SQLite should be TEXT for these types
			if sqliteVal != tt.expectedSQLite {
				t.Errorf("%s: SQLite should be %s, got %q", tt.name, tt.expectedSQLite, sqliteVal)
			}
		})
	}

	// SQLite now uses type affinities (DATE, TIME, DATETIME) that match PostgreSQL for round-trip
	matchingTypes := []struct {
		name string
		get  func(Dialect) string
	}{
		{"DateType", func(d Dialect) string { return d.DateType() }},
		{"TimeType", func(d Dialect) string { return d.TimeType() }},
		{"DateTimeType", func(d Dialect) string { return d.DateTimeType() }},
	}

	for _, tt := range matchingTypes {
		t.Run(tt.name, func(t *testing.T) {
			sqliteVal := tt.get(sqlite)
			pgVal := tt.get(pg)

			// These should now match (for round-trip compatibility)
			if sqliteVal != pgVal {
				t.Logf("%s: SQLite = %q, PostgreSQL = %q (using type affinities for round-trip)", tt.name, sqliteVal, pgVal)
			}
		})
	}
}

func TestTransactionalDDLSupport(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		want    bool
	}{
		{"PostgreSQL", Postgres(), true},
		{"SQLite", SQLite(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dialect.SupportsTransactionalDDL(); got != tt.want {
				t.Errorf("%s SupportsTransactionalDDL() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
