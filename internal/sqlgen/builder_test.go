package sqlgen

import (
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// Dialect Tests
// -----------------------------------------------------------------------------

func TestDialectString(t *testing.T) {
	tests := []struct {
		dialect Dialect
		want    string
	}{
		{Postgres, "postgres"},
		{SQLite, "sqlite"},
		{Dialect(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.dialect.String()
			if got != tt.want {
				t.Errorf("Dialect.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// QuoteIdent Tests
// -----------------------------------------------------------------------------

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		ident   string
		want    string
	}{
		// PostgreSQL uses double quotes
		{"postgres_simple", Postgres, "users", `"users"`},
		{"postgres_underscore", Postgres, "user_name", `"user_name"`},
		{"postgres_escape", Postgres, `user"name`, `"user""name"`},

		// SQLite uses double quotes (like PostgreSQL)
		{"sqlite_simple", SQLite, "users", `"users"`},
		{"sqlite_underscore", SQLite, "user_name", `"user_name"`},
		{"sqlite_escape", SQLite, `user"name`, `"user""name"`},

		// Unknown dialect defaults to PostgreSQL style
		{"unknown_simple", Dialect(99), "users", `"users"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteIdent(tt.dialect, tt.ident)
			if got != tt.want {
				t.Errorf("QuoteIdent(%v, %q) = %q, want %q", tt.dialect, tt.ident, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Placeholders Tests
// -----------------------------------------------------------------------------

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		n       int
		want    string
	}{
		// PostgreSQL uses numbered placeholders ($1, $2, ...)
		{"postgres_0", Postgres, 0, ""},
		{"postgres_1", Postgres, 1, "$1"},
		{"postgres_3", Postgres, 3, "$1, $2, $3"},
		{"postgres_5", Postgres, 5, "$1, $2, $3, $4, $5"},

		// SQLite uses question marks
		{"sqlite_0", SQLite, 0, ""},
		{"sqlite_1", SQLite, 1, "?"},
		{"sqlite_3", SQLite, 3, "?, ?, ?"},

		// Negative count
		{"postgres_neg", Postgres, -1, ""},
		{"sqlite_neg", SQLite, -1, ""},

		// Unknown dialect defaults to numbered
		{"unknown_3", Dialect(99), 3, "$1, $2, $3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Placeholders(tt.dialect, tt.n)
			if got != tt.want {
				t.Errorf("Placeholders(%v, %d) = %q, want %q", tt.dialect, tt.n, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Columns Helper Tests
// -----------------------------------------------------------------------------

func TestColumns(t *testing.T) {
	tests := []struct {
		name string
		cols []string
		want string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"id"}, `"id"`},
		{"multiple", []string{"id", "name", "email"}, `"id", "name", "email"`},
		{"with_escape", []string{`col"name`}, `"col""name"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Columns(tt.cols...)
			if got != tt.want {
				t.Errorf("Columns(%v) = %q, want %q", tt.cols, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// List Helper Tests
// -----------------------------------------------------------------------------

func TestList(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"a"}, "a"},
		{"multiple", []string{"a", "b", "c"}, "a, b, c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := List(tt.items...)
			if got != tt.want {
				t.Errorf("List(%v) = %q, want %q", tt.items, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Builder Fluent API Tests
// -----------------------------------------------------------------------------

func TestBuilderNew(t *testing.T) {
	for _, dialect := range []Dialect{Postgres, SQLite} {
		t.Run(dialect.String(), func(t *testing.T) {
			b := New(dialect)
			if b == nil {
				t.Fatal("New() returned nil")
			}
			if b.Dialect() != dialect {
				t.Errorf("Dialect() = %v, want %v", b.Dialect(), dialect)
			}
			if b.String() != "" {
				t.Errorf("new builder should be empty, got %q", b.String())
			}
		})
	}
}

func TestBuilderCreateTable(t *testing.T) {
	tests := []struct {
		dialect Dialect
		table   string
		want    string
	}{
		{Postgres, "users", `CREATE TABLE "users"`},
		{SQLite, "users", `CREATE TABLE "users"`},
	}

	for _, tt := range tests {
		t.Run(tt.dialect.String(), func(t *testing.T) {
			got := New(tt.dialect).CreateTable(tt.table).String()
			if got != tt.want {
				t.Errorf("CreateTable(%q) = %q, want %q", tt.table, got, tt.want)
			}
		})
	}
}

func TestBuilderDropTable(t *testing.T) {
	tests := []struct {
		dialect Dialect
		table   string
		want    string
	}{
		{Postgres, "users", `DROP TABLE "users"`},
		{SQLite, "users", `DROP TABLE "users"`},
	}

	for _, tt := range tests {
		t.Run(tt.dialect.String(), func(t *testing.T) {
			got := New(tt.dialect).DropTable(tt.table).String()
			if got != tt.want {
				t.Errorf("DropTable(%q) = %q, want %q", tt.table, got, tt.want)
			}
		})
	}
}

func TestBuilderAlterTable(t *testing.T) {
	tests := []struct {
		dialect Dialect
		table   string
		want    string
	}{
		{Postgres, "users", `ALTER TABLE "users"`},
		{SQLite, "users", `ALTER TABLE "users"`},
	}

	for _, tt := range tests {
		t.Run(tt.dialect.String(), func(t *testing.T) {
			got := New(tt.dialect).AlterTable(tt.table).String()
			if got != tt.want {
				t.Errorf("AlterTable(%q) = %q, want %q", tt.table, got, tt.want)
			}
		})
	}
}

func TestBuilderAddColumn(t *testing.T) {
	got := New(Postgres).AddColumn("email", "VARCHAR(255)").String()
	want := `"email" VARCHAR(255)`
	if got != want {
		t.Errorf("AddColumn() = %q, want %q", got, want)
	}
}

func TestBuilderDropColumn(t *testing.T) {
	got := New(Postgres).DropColumn("email").String()
	want := `DROP COLUMN "email"`
	if got != want {
		t.Errorf("DropColumn() = %q, want %q", got, want)
	}
}

func TestBuilderRenameColumn(t *testing.T) {
	tests := []struct {
		dialect Dialect
		want    string
	}{
		{Postgres, `RENAME COLUMN "old_name" TO "new_name"`},
		{SQLite, `RENAME COLUMN "old_name" TO "new_name"`},
	}

	for _, tt := range tests {
		t.Run(tt.dialect.String(), func(t *testing.T) {
			got := New(tt.dialect).RenameColumn("old_name", "new_name").String()
			if got != tt.want {
				t.Errorf("RenameColumn() = %q, want %q", got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Builder Column Modifier Tests
// -----------------------------------------------------------------------------

func TestBuilderNotNull(t *testing.T) {
	got := New(Postgres).NotNull().String()
	if got != " NOT NULL" {
		t.Errorf("NotNull() = %q, want ' NOT NULL'", got)
	}
}

func TestBuilderNull(t *testing.T) {
	got := New(Postgres).Null().String()
	if got != " NULL" {
		t.Errorf("Null() = %q, want ' NULL'", got)
	}
}

func TestBuilderDefault(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"'default_value'", " DEFAULT 'default_value'"},
		{"NOW()", " DEFAULT NOW()"},
		{"0", " DEFAULT 0"},
		{"TRUE", " DEFAULT TRUE"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := New(Postgres).Default(tt.expr).String()
			if got != tt.want {
				t.Errorf("Default(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestBuilderPrimaryKey(t *testing.T) {
	got := New(Postgres).PrimaryKey().String()
	if got != " PRIMARY KEY" {
		t.Errorf("PrimaryKey() = %q, want ' PRIMARY KEY'", got)
	}
}

func TestBuilderUnique(t *testing.T) {
	got := New(Postgres).Unique().String()
	if got != " UNIQUE" {
		t.Errorf("Unique() = %q, want ' UNIQUE'", got)
	}
}

func TestBuilderReferences(t *testing.T) {
	got := New(Postgres).References("users", "id").String()
	want := ` REFERENCES "users"("id")`
	if got != want {
		t.Errorf("References() = %q, want %q", got, want)
	}
}

func TestBuilderOnDelete(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"CASCADE", " ON DELETE CASCADE"},
		{"SET NULL", " ON DELETE SET NULL"},
		{"RESTRICT", " ON DELETE RESTRICT"},
		{"NO ACTION", " ON DELETE NO ACTION"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			got := New(Postgres).OnDelete(tt.action).String()
			if got != tt.want {
				t.Errorf("OnDelete(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestBuilderOnUpdate(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"CASCADE", " ON UPDATE CASCADE"},
		{"SET NULL", " ON UPDATE SET NULL"},
		{"RESTRICT", " ON UPDATE RESTRICT"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			got := New(Postgres).OnUpdate(tt.action).String()
			if got != tt.want {
				t.Errorf("OnUpdate(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Builder Constraint Tests
// -----------------------------------------------------------------------------

func TestBuilderConstraint(t *testing.T) {
	got := New(Postgres).Constraint("fk_posts_user_id").String()
	want := `CONSTRAINT "fk_posts_user_id"`
	if got != want {
		t.Errorf("Constraint() = %q, want %q", got, want)
	}
}

func TestBuilderForeignKey(t *testing.T) {
	tests := []struct {
		name string
		cols []string
		want string
	}{
		{"single", []string{"user_id"}, ` FOREIGN KEY ("user_id")`},
		{"multiple", []string{"user_id", "post_id"}, ` FOREIGN KEY ("user_id", "post_id")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(Postgres).ForeignKey(tt.cols...).String()
			if got != tt.want {
				t.Errorf("ForeignKey(%v) = %q, want %q", tt.cols, got, tt.want)
			}
		})
	}
}

func TestBuilderCheck(t *testing.T) {
	got := New(Postgres).Check("price > 0").String()
	want := " CHECK (price > 0)"
	if got != want {
		t.Errorf("Check() = %q, want %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// Builder Utility Tests
// -----------------------------------------------------------------------------

func TestBuilderRaw(t *testing.T) {
	got := New(Postgres).Raw("SELECT 1").String()
	if got != "SELECT 1" {
		t.Errorf("Raw() = %q, want 'SELECT 1'", got)
	}
}

func TestBuilderComma(t *testing.T) {
	got := New(Postgres).Comma().String()
	if got != "," {
		t.Errorf("Comma() = %q, want ','", got)
	}
}

func TestBuilderParens(t *testing.T) {
	got := New(Postgres).OpenParen().Raw("content").CloseParen().String()
	if got != " (content)" {
		t.Errorf("Parens() = %q, want ' (content)'", got)
	}
}

func TestBuilderNewline(t *testing.T) {
	got := New(Postgres).Raw("line1").Newline().Raw("line2").String()
	if got != "line1\nline2" {
		t.Errorf("Newline() = %q, want 'line1\\nline2'", got)
	}
}

func TestBuilderSpace(t *testing.T) {
	got := New(Postgres).Raw("a").Space().Raw("b").String()
	if got != "a b" {
		t.Errorf("Space() = %q, want 'a b'", got)
	}
}

func TestBuilderReset(t *testing.T) {
	b := New(Postgres).Raw("some content")
	if b.String() == "" {
		t.Fatal("builder should have content before reset")
	}
	b.Reset()
	if b.String() != "" {
		t.Errorf("builder should be empty after reset, got %q", b.String())
	}
}

// -----------------------------------------------------------------------------
// Full CREATE TABLE Generation Tests
// -----------------------------------------------------------------------------

func TestFullCreateTablePostgres(t *testing.T) {
	b := New(Postgres)
	b.CreateTable("auth_users").OpenParen().Newline()
	b.Raw("  ").AddColumn("id", "UUID").PrimaryKey().Comma().Newline()
	b.Raw("  ").AddColumn("email", "VARCHAR(255)").NotNull().Unique().Comma().Newline()
	b.Raw("  ").AddColumn("name", "VARCHAR(100)").NotNull().Comma().Newline()
	b.Raw("  ").AddColumn("created_at", "TIMESTAMP").NotNull().Default("NOW()").Newline()
	b.CloseParen()

	got := b.String()

	// Check structure
	if !strings.HasPrefix(got, `CREATE TABLE "auth_users" (`) {
		t.Errorf("should start with CREATE TABLE, got: %s", got)
	}
	if !strings.HasSuffix(got, ")") {
		t.Errorf("should end with ), got: %s", got)
	}

	// Check columns
	expectedParts := []string{
		`"id" UUID PRIMARY KEY`,
		`"email" VARCHAR(255) NOT NULL UNIQUE`,
		`"name" VARCHAR(100) NOT NULL`,
		`"created_at" TIMESTAMP NOT NULL DEFAULT NOW()`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Errorf("missing part: %s\nin: %s", part, got)
		}
	}
}

func TestFullCreateTableSQLite(t *testing.T) {
	b := New(SQLite)
	b.CreateTable("posts").OpenParen().Newline()
	b.Raw("  ").AddColumn("id", "TEXT").PrimaryKey().Comma().Newline()
	b.Raw("  ").AddColumn("user_id", "TEXT").NotNull().
		References("users", "id").OnDelete("CASCADE").Newline()
	b.CloseParen()

	got := b.String()

	// Check SQLite quoting (double quotes)
	if !strings.Contains(got, `"posts"`) {
		t.Errorf("SQLite should use double quotes, got: %s", got)
	}
	if !strings.Contains(got, `REFERENCES "users"("id")`) {
		t.Errorf("SQLite should quote references, got: %s", got)
	}
}

// -----------------------------------------------------------------------------
// Method Chaining Tests
// -----------------------------------------------------------------------------

func TestMethodChaining(t *testing.T) {
	// Verify all methods return *Builder for chaining
	b := New(Postgres)
	result := b.
		CreateTable("test").
		OpenParen().
		Newline().
		Raw("  ").
		AddColumn("id", "UUID").
		PrimaryKey().
		NotNull().
		Unique().
		Default("gen_random_uuid()").
		Comma().
		Newline().
		Raw("  ").
		AddColumn("ref_id", "UUID").
		References("other", "id").
		OnDelete("CASCADE").
		OnUpdate("CASCADE").
		CloseParen().
		Space().
		Reset()

	// After Reset, should be empty
	if result.String() != "" {
		t.Errorf("after Reset(), String() should be empty, got %q", result.String())
	}
}

// -----------------------------------------------------------------------------
// ALTER TABLE with Constraints Test
// -----------------------------------------------------------------------------

func TestAlterTableAddConstraint(t *testing.T) {
	b := New(Postgres)
	b.AlterTable("posts").Space().Raw("ADD").Space()
	b.Constraint("fk_posts_user_id")
	b.ForeignKey("user_id")
	b.References("users", "id")
	b.OnDelete("CASCADE")

	got := b.String()
	want := `ALTER TABLE "posts" ADD CONSTRAINT "fk_posts_user_id" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE`

	if got != want {
		t.Errorf("ALTER TABLE ADD CONSTRAINT\ngot:  %q\nwant: %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// Builder Reuse Test
// -----------------------------------------------------------------------------

func TestBuilderReuse(t *testing.T) {
	b := New(Postgres)

	// First use
	b.CreateTable("table1")
	first := b.String()
	if first != `CREATE TABLE "table1"` {
		t.Errorf("first use: got %q", first)
	}

	// Reset and reuse
	b.Reset()
	b.CreateTable("table2")
	second := b.String()
	if second != `CREATE TABLE "table2"` {
		t.Errorf("second use: got %q", second)
	}
}
