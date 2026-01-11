package devdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/drift"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/introspect"
)

// TestNormalization tests that dev database normalization produces identical
// schemas when applying and introspecting the same table definition.
func TestNormalization(t *testing.T) {
	// Create a simple table definition
	originalTable := &ast.TableDef{
		Namespace: "app",
		Name:      "users",
		Columns: []*ast.ColumnDef{
			{
				Name:       "id",
				Type:       "id",
				PrimaryKey: true,
				Nullable:   false,
			},
			{
				Name:     "username",
				Type:     "string",
				TypeArgs: []any{50},
				Nullable: false,
			},
			{
				Name:     "email",
				Type:     "string",
				TypeArgs: []any{100},
				Unique:   true,
				Nullable: false,
			},
			{
				Name:          "created_at",
				Type:          "datetime",
				ServerDefault: "CURRENT_TIMESTAMP",
				Nullable:      false,
			},
			{
				Name:          "updated_at",
				Type:          "datetime",
				ServerDefault: "CURRENT_TIMESTAMP",
				Nullable:      false,
			},
		},
	}

	originalSchema := &engine.Schema{
		Tables: map[string]*ast.TableDef{
			"app.users": originalTable,
		},
	}

	// Create dev database
	dev, err := New()
	if err != nil {
		t.Fatalf("Failed to create dev database: %v", err)
	}
	defer dev.Close()

	// Normalize schema
	ctx := context.Background()
	normalizedSchema, err := dev.NormalizeSchema(ctx, originalSchema)
	if err != nil {
		t.Fatalf("Failed to normalize schema: %v", err)
	}

	// Print column details for comparison
	fmt.Println("\n=== ORIGINAL SCHEMA ===")
	for _, col := range originalTable.Columns {
		fmt.Printf("Column: %s\n", col.Name)
		fmt.Printf("  Type: %s %v\n", col.Type, col.TypeArgs)
		fmt.Printf("  Nullable: %v\n", col.Nullable)
		fmt.Printf("  PrimaryKey: %v\n", col.PrimaryKey)
		fmt.Printf("  Unique: %v\n", col.Unique)
		fmt.Printf("  Default: %v (Set: %v)\n", col.Default, col.DefaultSet)
		fmt.Printf("  ServerDefault: %s\n", col.ServerDefault)
		fmt.Println()
	}

	normalizedTable := normalizedSchema.Tables["app.users"]
	if normalizedTable == nil {
		t.Fatal("Normalized table not found")
	}

	fmt.Println("\n=== NORMALIZED SCHEMA (from dev DB) ===")
	for _, col := range normalizedTable.Columns {
		fmt.Printf("Column: %s\n", col.Name)
		fmt.Printf("  Type: %s %v\n", col.Type, col.TypeArgs)
		fmt.Printf("  Nullable: %v\n", col.Nullable)
		fmt.Printf("  PrimaryKey: %v\n", col.PrimaryKey)
		fmt.Printf("  Unique: %v\n", col.Unique)
		fmt.Printf("  Default: %v (Set: %v)\n", col.Default, col.DefaultSet)
		fmt.Printf("  ServerDefault: %s\n", col.ServerDefault)
		fmt.Println()
	}

	// Now normalize again (second pass) - should be identical
	dev2, err := New()
	if err != nil {
		t.Fatalf("Failed to create second dev database: %v", err)
	}
	defer dev2.Close()

	secondNormalized, err := dev2.NormalizeSchema(ctx, normalizedSchema)
	if err != nil {
		t.Fatalf("Failed to normalize second time: %v", err)
	}

	fmt.Println("\n=== SECOND NORMALIZATION (should be identical) ===")
	secondTable := secondNormalized.Tables["app.users"]
	if secondTable == nil {
		t.Fatal("Second normalized table not found")
	}

	for _, col := range secondTable.Columns {
		fmt.Printf("Column: %s\n", col.Name)
		fmt.Printf("  Type: %s %v\n", col.Type, col.TypeArgs)
		fmt.Printf("  Nullable: %v\n", col.Nullable)
		fmt.Printf("  PrimaryKey: %v\n", col.PrimaryKey)
		fmt.Printf("  Unique: %v\n", col.Unique)
		fmt.Printf("  Default: %v (Set: %v)\n", col.Default, col.DefaultSet)
		fmt.Printf("  ServerDefault: %s\n", col.ServerDefault)
		fmt.Println()
	}

	// Compute hashes
	hash1, err := drift.ComputeSchemaHash(normalizedSchema)
	if err != nil {
		t.Fatalf("Failed to compute first hash: %v", err)
	}

	hash2, err := drift.ComputeSchemaHash(secondNormalized)
	if err != nil {
		t.Fatalf("Failed to compute second hash: %v", err)
	}

	fmt.Printf("\n=== HASH COMPARISON ===\n")
	fmt.Printf("First normalization hash:  %s\n", hash1.Root)
	fmt.Printf("Second normalization hash: %s\n", hash2.Root)

	if hash1.Root != hash2.Root {
		t.Errorf("Hashes don't match after normalization!\nFirst:  %s\nSecond: %s", hash1.Root, hash2.Root)

		// Compare table hashes
		table1Hash := hash1.Tables["app.users"]
		table2Hash := hash2.Tables["app.users"]

		fmt.Printf("\nTable hash 1: %s\n", table1Hash.Hash)
		fmt.Printf("Table hash 2: %s\n", table2Hash.Hash)

		// Compare column hashes
		fmt.Println("\n=== COLUMN HASH COMPARISON ===")
		for colName, colHash1 := range table1Hash.Columns {
			colHash2 := table2Hash.Columns[colName]
			match := ""
			if colHash1 != colHash2 {
				match = " *** MISMATCH ***"
			}
			fmt.Printf("%s: %s vs %s%s\n", colName, colHash1[:8], colHash2[:8], match)
		}
	}

	// Introspect the dev database directly to see what SQLite stored
	intro := introspect.New(dev.db, dev.dialect)

	// First, query sqlite_master directly to see what's actually stored
	fmt.Println("\n=== RAW SQLITE_MASTER CONTENTS ===")
	rows, err := dev.db.QueryContext(ctx, `
		SELECT type, name, sql FROM sqlite_master
		WHERE tbl_name = 'app_users'
		ORDER BY type, name
	`)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	for rows.Next() {
		var typ, name string
		var sql sql.NullString
		rows.Scan(&typ, &name, &sql)
		fmt.Printf("%s: %s\n", typ, name)
		if sql.Valid {
			fmt.Printf("  SQL: %s\n", sql.String)
		} else {
			fmt.Printf("  SQL: NULL\n")
		}
	}
	rows.Close()

	introspectedSchema, err := intro.IntrospectSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to introspect dev database: %v", err)
	}

	fmt.Println("\n=== DIRECT INTROSPECTION (what SQLite actually stored) ===")
	introspectedTable := introspectedSchema.Tables["app.users"]
	if introspectedTable != nil {
		for _, col := range introspectedTable.Columns {
			fmt.Printf("Column: %s\n", col.Name)
			fmt.Printf("  Type: %s %v\n", col.Type, col.TypeArgs)
			fmt.Printf("  Nullable: %v\n", col.Nullable)
			fmt.Printf("  PrimaryKey: %v\n", col.PrimaryKey)
			fmt.Printf("  Unique: %v\n", col.Unique)
			fmt.Printf("  Default: %v (Set: %v)\n", col.Default, col.DefaultSet)
			fmt.Printf("  ServerDefault: %s\n", col.ServerDefault)
			fmt.Println()
		}

		fmt.Printf("\nIndexes: %d\n", len(introspectedTable.Indexes))
		for _, idx := range introspectedTable.Indexes {
			fmt.Printf("  Index: %s (Unique: %v, Columns: %v)\n", idx.Name, idx.Unique, idx.Columns)
		}
	}
}
