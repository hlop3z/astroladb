//go:build integration

package astroladb

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/ast"
	"github.com/hlop3z/astroladb/internal/metadata"
)

// roundtripTable returns a representative schema covering many column types.
func roundtripTable() *ast.TableDef {
	minVal := 0
	maxVal := 1000
	return &ast.TableDef{
		Namespace: "app",
		Name:      "entity",
		Columns: []*ast.ColumnDef{
			{Name: "id", Type: "uuid", PrimaryKey: true},
			{Name: "name", Type: "string", TypeArgs: []any{100}},
			{Name: "description", Type: "text", Nullable: true},
			{Name: "count", Type: "integer", Min: &minVal, Max: &maxVal},
			{Name: "score", Type: "float"},
			{Name: "price", Type: "decimal", TypeArgs: []any{10, 2}},
			{Name: "is_active", Type: "boolean", Default: true},
			{Name: "birth_date", Type: "date", Nullable: true},
			{Name: "login_time", Type: "time", Nullable: true},
			{Name: "created_at", Type: "datetime"},
			{Name: "data", Type: "json", Nullable: true},
			{Name: "avatar", Type: "base64", Nullable: true},
			{Name: "status", Type: "enum", TypeArgs: []any{[]string{"active", "inactive", "pending"}}},
		},
		Docs: "A representative entity for round-trip codegen testing",
	}
}

func newTestExportContext() *exportContext {
	return &exportContext{
		ExportConfig: &ExportConfig{},
		Metadata:     metadata.New(),
	}
}

func TestRoundTripGo(t *testing.T) {
	tables := []*ast.TableDef{roundtripTable()}
	cfg := newTestExportContext()

	data, err := exportGo(tables, cfg)
	if err != nil {
		t.Fatalf("exportGo failed: %v", err)
	}

	// Write to temp file and run go vet
	dir := t.TempDir()
	filePath := filepath.Join(dir, "generated.go")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// go vet needs a go.mod
	modContent := "module roundtrip_test\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("go vet failed on generated Go code:\n%s\n\nGenerated code:\n%s", output, data)
	}
}

func TestRoundTripTypeScript(t *testing.T) {
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found, skipping TypeScript round-trip test")
	}

	tables := []*ast.TableDef{roundtripTable()}
	cfg := newTestExportContext()

	data, err := exportTypeScript(tables, cfg)
	if err != nil {
		t.Fatalf("exportTypeScript failed: %v", err)
	}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "generated.ts")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	cmd := exec.Command("npx", "tsc", "--noEmit", "--strict", "--target", "ES2020", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("tsc failed on generated TypeScript code:\n%s\n\nGenerated code:\n%s", output, data)
	}
}

func TestRoundTripPython(t *testing.T) {
	pythonCmd := "python3"
	if _, err := exec.LookPath(pythonCmd); err != nil {
		pythonCmd = "python"
		if _, err := exec.LookPath(pythonCmd); err != nil {
			t.Skip("python not found, skipping Python round-trip test")
		}
	}

	tables := []*ast.TableDef{roundtripTable()}
	cfg := newTestExportContext()

	data, err := exportPython(tables, cfg)
	if err != nil {
		t.Fatalf("exportPython failed: %v", err)
	}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "generated.py")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	cmd := exec.Command(pythonCmd, "-m", "py_compile", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("py_compile failed on generated Python code:\n%s\n\nGenerated code:\n%s", output, data)
	}
}

func TestRoundTripRust(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not found, skipping Rust round-trip test")
	}

	tables := []*ast.TableDef{roundtripTable()}
	cfg := newTestExportContext()

	data, err := exportRust(tables, cfg)
	if err != nil {
		t.Fatalf("exportRust failed: %v", err)
	}

	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Write generated code as lib.rs
	if err := os.WriteFile(filepath.Join(srcDir, "lib.rs"), data, 0644); err != nil {
		t.Fatalf("failed to write lib.rs: %v", err)
	}

	// Write Cargo.toml with serde dependency (used by generated code)
	cargoToml := `[package]
name = "roundtrip_test"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = { version = "1", features = ["derive"] }
chrono = { version = "0.4", features = ["serde"] }
`
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}

	cmd := exec.Command("cargo", "check")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("cargo check failed on generated Rust code:\n%s\n\nGenerated code:\n%s", output, data)
	}
}
