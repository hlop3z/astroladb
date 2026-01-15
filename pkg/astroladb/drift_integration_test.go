package astroladb

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/dialect"
	"github.com/hlop3z/astroladb/internal/drift"
	"github.com/hlop3z/astroladb/internal/engine"
	"github.com/hlop3z/astroladb/internal/introspect"
)

// TestDriftDetectionWithNormalization tests the complete drift detection flow
// using dev database normalization to ensure accurate comparison.
func TestDriftDetectionWithNormalization(t *testing.T) {
	// Create temporary database
	dbPath := "test_drift_detection.db"
	defer os.Remove(dbPath)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Apply initial schema using raw SQL (simulating migrations)
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE "app_users" (
			"id" TEXT PRIMARY KEY,
			"username" TEXT NOT NULL,
			"email" TEXT NOT NULL CONSTRAINT "uniq_app_users_email" UNIQUE,
			"created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"updated_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create Client with minimal config
	client := &Client{
		db:      db,
		dialect: dialect.Get("sqlite"),
	}

	// Define expected schema (from "migrations")
	expectedSchema := createUserTableSchema()

	// Test 1: Check should pass when database matches migrations
	t.Run("No drift when database matches", func(t *testing.T) {
		// Get actual schema from database
		intro := introspect.New(db, client.dialect)
		actualSchema, err := intro.IntrospectSchema(ctx)
		if err != nil {
			t.Fatalf("Failed to introspect schema: %v", err)
		}

		// Normalize expected schema through dev database
		normalizedExpected, err := client.normalizeSchema(expectedSchema)
		if err != nil {
			t.Fatalf("Failed to normalize expected schema: %v", err)
		}

		// Compute hashes
		expectedHashObj, err := drift.ComputeSchemaHash(normalizedExpected)
		if err != nil {
			t.Fatalf("Failed to compute expected hash: %v", err)
		}
		expectedHash := expectedHashObj.Root

		actualHashObj, err := drift.ComputeSchemaHash(actualSchema)
		if err != nil {
			t.Fatalf("Failed to compute actual hash: %v", err)
		}
		actualHash := actualHashObj.Root

		if expectedHash != actualHash {
			t.Errorf("Hashes don't match!\nExpected: %s\nActual: %s", expectedHash, actualHash)

			// Debug: print column details
			t.Logf("\n=== EXPECTED SCHEMA (normalized) ===")
			printTableDetails(t, normalizedExpected.Tables["app.users"])

			t.Logf("\n=== ACTUAL SCHEMA (from database) ===")
			printTableDetails(t, actualSchema.Tables["app.users"])
		} else {
			t.Logf("✓ Hashes match: %s", expectedHash)
		}
	})

	// Test 2: Detect drift when column is added to database
	t.Run("Detect drift when column added", func(t *testing.T) {
		// Add a column to the database
		_, err = db.ExecContext(ctx, `ALTER TABLE "app_users" ADD COLUMN "extra" TEXT`)
		if err != nil {
			t.Fatalf("Failed to add column: %v", err)
		}

		// Get actual schema from database
		intro := introspect.New(db, client.dialect)
		actualSchema, err := intro.IntrospectSchema(ctx)
		if err != nil {
			t.Fatalf("Failed to introspect schema: %v", err)
		}

		// Normalize expected schema
		normalizedExpected, err := client.normalizeSchema(expectedSchema)
		if err != nil {
			t.Fatalf("Failed to normalize expected schema: %v", err)
		}

		// Compute hashes
		expectedHashObj, err := drift.ComputeSchemaHash(normalizedExpected)
		if err != nil {
			t.Fatalf("Failed to compute expected hash: %v", err)
		}
		expectedHash := expectedHashObj.Root

		actualHashObj, err := drift.ComputeSchemaHash(actualSchema)
		if err != nil {
			t.Fatalf("Failed to compute actual hash: %v", err)
		}
		actualHash := actualHashObj.Root

		if expectedHash == actualHash {
			t.Error("Expected drift to be detected, but hashes match!")
		} else {
			t.Logf("✓ Drift detected as expected")
			t.Logf("  Expected: %s", expectedHash)
			t.Logf("  Actual: %s", actualHash)
		}
	})
}

func createUserTableSchema() *engine.Schema {
	table := &ast.TableDef{
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

	return &engine.Schema{
		Tables: map[string]*ast.TableDef{
			"app.users": table,
		},
	}
}

func printTableDetails(t *testing.T, table *ast.TableDef) {
	if table == nil {
		t.Logf("Table is nil!")
		return
	}

	for _, col := range table.Columns {
		t.Logf("Column: %s", col.Name)
		t.Logf("  Type: %s %v", col.Type, col.TypeArgs)
		t.Logf("  Nullable: %v", col.Nullable)
		t.Logf("  PrimaryKey: %v", col.PrimaryKey)
		t.Logf("  Unique: %v", col.Unique)
		t.Logf("  Default: %v (Set: %v)", col.Default, col.DefaultSet)
		t.Logf("  ServerDefault: %s", col.ServerDefault)
	}

	t.Logf("\nIndexes: %d", len(table.Indexes))
	for _, idx := range table.Indexes {
		t.Logf("  Index: %s (Unique: %v, Columns: %v)", idx.Name, idx.Unique, idx.Columns)
	}
}
