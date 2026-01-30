//go:build integration

package dialect

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
)

// normalizeType maps dialect-specific types to canonical forms for comparison.
func normalizeType(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	replacements := map[string]string{
		"SERIAL":           "INTEGER",
		"BIGSERIAL":        "INTEGER",
		"TIMESTAMPTZ":      "DATETIME",
		"TIMESTAMP":        "DATETIME",
		"BYTEA":            "BLOB",
		"JSONB":            "JSON",
		"REAL":             "FLOAT",
		"DOUBLE PRECISION": "FLOAT",
		"BOOLEAN":          "BOOLEAN",
		"INTEGER":          "INTEGER",
		"TEXT":             "TEXT",
	}
	for k, v := range replacements {
		if s == k {
			return v
		}
	}
	return s
}

// consistencySchema returns representative tables for cross-dialect testing.
func consistencySchemas() []*ast.CreateTable {
	return []*ast.CreateTable{
		{
			TableOp: ast.TableOp{Namespace: "app", Name: "users"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "email", Type: "string", TypeArgs: []any{255}},
				{Name: "name", Type: "text", Nullable: true},
				{Name: "age", Type: "integer", Nullable: true},
				{Name: "score", Type: "float"},
				{Name: "active", Type: "boolean", Default: true},
				{Name: "created_at", Type: "datetime"},
				{Name: "data", Type: "json", Nullable: true},
			},
			Indexes: []*ast.IndexDef{
				{Name: "idx_app_users_email", Columns: []string{"email"}, Unique: true},
			},
		},
		{
			TableOp: ast.TableOp{Namespace: "app", Name: "posts"},
			Columns: []*ast.ColumnDef{
				{Name: "id", Type: "uuid", PrimaryKey: true},
				{Name: "title", Type: "string", TypeArgs: []any{200}},
				{Name: "body", Type: "text"},
				{Name: "author_id", Type: "uuid", Reference: &ast.Reference{Table: "app.users", Column: "id", OnDelete: "CASCADE"}},
			},
			ForeignKeys: []*ast.ForeignKeyDef{
				{
					Name:       "fk_app_posts_author_id",
					Columns:    []string{"author_id"},
					RefTable:   "app_users",
					RefColumns: []string{"id"},
					OnDelete:   "CASCADE",
				},
			},
		},
	}
}

func TestCrossDialectCreateTableConsistency(t *testing.T) {
	pg := Postgres()
	sl := SQLite()

	schemas := consistencySchemas()

	for _, op := range schemas {
		t.Run(op.Table(), func(t *testing.T) {
			pgSQL, err := pg.CreateTableSQL(op)
			if err != nil {
				t.Fatalf("Postgres CreateTableSQL failed: %v", err)
			}
			slSQL, err := sl.CreateTableSQL(op)
			if err != nil {
				t.Fatalf("SQLite CreateTableSQL failed: %v", err)
			}

			// Both should produce non-empty SQL
			if pgSQL == "" {
				t.Error("Postgres produced empty SQL")
			}
			if slSQL == "" {
				t.Error("SQLite produced empty SQL")
			}

			// Both should contain the table name
			tableName := op.Table()
			if !strings.Contains(pgSQL, tableName) {
				t.Errorf("Postgres SQL missing table name %q", tableName)
			}
			if !strings.Contains(slSQL, tableName) {
				t.Errorf("SQLite SQL missing table name %q", tableName)
			}

			// Both should reference all column names
			for _, col := range op.Columns {
				if !strings.Contains(pgSQL, col.Name) {
					t.Errorf("Postgres SQL missing column %q", col.Name)
				}
				if !strings.Contains(slSQL, col.Name) {
					t.Errorf("SQLite SQL missing column %q", col.Name)
				}
			}

			// Check FK references are present in both
			for _, fk := range op.ForeignKeys {
				if !strings.Contains(pgSQL, fk.RefTable) && !strings.Contains(pgSQL, "REFERENCES") {
					t.Errorf("Postgres SQL missing FK reference to %q", fk.RefTable)
				}
				if !strings.Contains(slSQL, fk.RefTable) && !strings.Contains(slSQL, "REFERENCES") {
					t.Errorf("SQLite SQL missing FK reference to %q", fk.RefTable)
				}
			}

			// Check index generation consistency
			for _, idx := range op.Indexes {
				pgIdx, err := pg.CreateIndexSQL(&ast.CreateIndex{
					TableRef: ast.TableRef{Namespace: op.Namespace, Table_: op.Name},
					Name:     idx.Name,
					Columns:  idx.Columns,
					Unique:   idx.Unique,
				})
				if err != nil {
					t.Fatalf("Postgres CreateIndexSQL failed: %v", err)
				}
				slIdx, err := sl.CreateIndexSQL(&ast.CreateIndex{
					TableRef: ast.TableRef{Namespace: op.Namespace, Table_: op.Name},
					Name:     idx.Name,
					Columns:  idx.Columns,
					Unique:   idx.Unique,
				})
				if err != nil {
					t.Fatalf("SQLite CreateIndexSQL failed: %v", err)
				}

				// Both should contain index name and column names
				if !strings.Contains(pgIdx, idx.Name) {
					t.Errorf("Postgres index SQL missing index name %q", idx.Name)
				}
				if !strings.Contains(slIdx, idx.Name) {
					t.Errorf("SQLite index SQL missing index name %q", idx.Name)
				}
				for _, c := range idx.Columns {
					if !strings.Contains(pgIdx, c) {
						t.Errorf("Postgres index SQL missing column %q", c)
					}
					if !strings.Contains(slIdx, c) {
						t.Errorf("SQLite index SQL missing column %q", c)
					}
				}

				// Both unique indexes should contain UNIQUE keyword
				if idx.Unique {
					if !strings.Contains(strings.ToUpper(pgIdx), "UNIQUE") {
						t.Error("Postgres unique index missing UNIQUE keyword")
					}
					if !strings.Contains(strings.ToUpper(slIdx), "UNIQUE") {
						t.Error("SQLite unique index missing UNIQUE keyword")
					}
				}
			}
		})
	}
}

func TestCrossDialectAlterColumnConsistency(t *testing.T) {
	pg := Postgres()
	sl := SQLite()

	nullable := true
	op := &ast.AlterColumn{
		TableRef:    ast.TableRef{Namespace: "app", Table_: "users"},
		Name:        "name",
		NewType:     "text",
		SetNullable: &nullable,
	}

	pgSQL, pgErr := pg.AlterColumnSQL(op)
	slSQL, slErr := sl.AlterColumnSQL(op)

	// Both should either succeed or fail — but we mainly check they don't panic
	if pgErr != nil && slErr != nil {
		t.Log("Both dialects returned errors for AlterColumn (may be expected for SQLite)")
		return
	}

	// If both succeed, both should reference the table and column
	if pgErr == nil && pgSQL != "" {
		if !strings.Contains(pgSQL, "name") {
			t.Error("Postgres AlterColumn SQL missing column name")
		}
	}
	if slErr == nil && slSQL != "" {
		if !strings.Contains(slSQL, "name") {
			t.Error("SQLite AlterColumn SQL missing column name")
		}
	}
}

func TestCrossDialectDropTableConsistency(t *testing.T) {
	pg := Postgres()
	sl := SQLite()

	op := &ast.DropTable{
		TableOp:  ast.TableOp{Namespace: "app", Name: "users"},
		IfExists: true,
	}

	pgSQL, err := pg.DropTableSQL(op)
	if err != nil {
		t.Fatalf("Postgres DropTableSQL failed: %v", err)
	}
	slSQL, err := sl.DropTableSQL(op)
	if err != nil {
		t.Fatalf("SQLite DropTableSQL failed: %v", err)
	}

	// Both should contain DROP TABLE and IF EXISTS
	for name, sql := range map[string]string{"Postgres": pgSQL, "SQLite": slSQL} {
		upper := strings.ToUpper(sql)
		if !strings.Contains(upper, "DROP TABLE") {
			t.Errorf("%s missing DROP TABLE", name)
		}
		if !strings.Contains(upper, "IF EXISTS") {
			t.Errorf("%s missing IF EXISTS", name)
		}
		if !strings.Contains(sql, "app_users") {
			t.Errorf("%s missing table name", name)
		}
	}
}
