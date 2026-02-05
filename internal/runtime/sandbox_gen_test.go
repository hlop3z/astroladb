package runtime

import (
	"strings"
	"testing"
)

func TestGenRender(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => render({"a.txt": "hello"}))`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}

	if result.Files["a.txt"] != "hello" {
		t.Errorf("expected 'hello', got %q", result.Files["a.txt"])
	}
}

func TestGenReceivesSchema(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const projectName = schema.project || "unknown";
		render({"output.txt": projectName});
	})`
	schema := map[string]any{
		"project": "astroladb",
	}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if result.Files["output.txt"] != "astroladb" {
		t.Errorf("expected 'astroladb', got %q", result.Files["output.txt"])
	}
}

func TestGenSchemaFrozen(t *testing.T) {
	s := NewSandbox(nil)
	code := `"use strict";
	gen((schema) => {
		schema.tables = [];
		render({"out.txt": "done"});
	})`
	schema := map[string]any{
		"tables": []any{
			map[string]any{"name": "users"},
		},
	}

	_, err := s.RunGenerator(code, schema)
	if err == nil {
		t.Fatal("expected error when mutating frozen schema, got nil")
	}

	if !strings.Contains(err.Error(), "read-only") && !strings.Contains(err.Error(), "not extensible") && !strings.Contains(err.Error(), "Cannot") {
		t.Errorf("expected freeze-related error, got: %v", err)
	}
}

func TestGenWithoutRender(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		// callback does not call render()
		return null;
	})`
	schema := map[string]any{}

	_, err := s.RunGenerator(code, schema)
	if err == nil {
		t.Fatal("expected error when render() not called, got nil")
	}

	if !strings.Contains(err.Error(), "render()") {
		t.Errorf("expected error mentioning render(), got: %v", err)
	}
}

func TestRenderNonString(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => render({"a": 123}))`
	schema := map[string]any{}

	_, err := s.RunGenerator(code, schema)
	if err == nil {
		t.Fatal("expected error for non-string value, got nil")
	}

	if !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "string") {
		t.Errorf("expected error mentioning key 'a' and 'string', got: %v", err)
	}
}

func TestRenderArray(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => render([]))`
	schema := map[string]any{}

	_, err := s.RunGenerator(code, schema)
	if err == nil {
		t.Fatal("expected error for array argument, got nil")
	}

	if !strings.Contains(err.Error(), "array") || !strings.Contains(err.Error(), "object") {
		t.Errorf("expected error mentioning array/object, got: %v", err)
	}
}

func TestRenderNull(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => render(null))`
	schema := map[string]any{}

	_, err := s.RunGenerator(code, schema)
	if err == nil {
		t.Fatal("expected error for null argument, got nil")
	}

	if !strings.Contains(err.Error(), "object") {
		t.Errorf("expected error mentioning 'object', got: %v", err)
	}
}

func TestGenNotFunction(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen("not a function")`
	schema := map[string]any{}

	_, err := s.RunGenerator(code, schema)
	if err == nil {
		t.Fatal("expected error for non-function argument, got nil")
	}

	if !strings.Contains(err.Error(), "function") {
		t.Errorf("expected error mentioning 'function', got: %v", err)
	}
}

func TestJsonHelper(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const obj = {a: 1, b: 2};
		const output = json(obj);
		render({"data.json": output});
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	content := result.Files["data.json"]
	if !strings.Contains(content, `"a":1`) || !strings.Contains(content, `"b":2`) {
		t.Errorf("expected JSON with a:1 and b:2, got: %q", content)
	}
}

func TestJsonHelperIndent(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const obj = {a: 1, b: 2};
		const output = json(obj, "  ");
		render({"data.json": output});
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	content := result.Files["data.json"]
	if !strings.Contains(content, "\n") {
		t.Errorf("expected indented JSON (with newlines), got: %q", content)
	}
	if !strings.Contains(content, `"a": 1`) {
		t.Errorf("expected pretty-printed JSON, got: %q", content)
	}
}

func TestDedentHelper(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const text = dedent(` + "`" + `
			line one
			line two
		` + "`" + `);
		render({"output.txt": text});
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	content := result.Files["output.txt"]
	lines := strings.Split(strings.TrimSpace(content), "\n")

	// After dedent, lines should have no common leading whitespace
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t\t") {
				t.Errorf("expected dedented text, but found leading whitespace: %q", line)
			}
		}
	}
}

func TestPerTableHelper(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const files = perTable(schema, (table) => {
			const filename = table.namespace + "_" + table.name + ".txt";
			return {[filename]: "table: " + table.name};
		});
		render(files);
	})`
	schema := map[string]any{
		"tables": []any{
			map[string]any{"namespace": "auth", "name": "users"},
			map[string]any{"namespace": "auth", "name": "roles"},
		},
	}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}

	if result.Files["auth_users.txt"] != "table: users" {
		t.Errorf("expected 'table: users', got %q", result.Files["auth_users.txt"])
	}

	if result.Files["auth_roles.txt"] != "table: roles" {
		t.Errorf("expected 'table: roles', got %q", result.Files["auth_roles.txt"])
	}
}

func TestPerNamespaceHelper(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const files = perNamespace(schema, (ns, tables) => {
			return {[ns + ".txt"]: "namespace: " + ns};
		});
		render(files);
	})`
	schema := map[string]any{
		"models": map[string]any{
			"auth": []any{
				map[string]any{"namespace": "auth", "name": "users"},
			},
			"billing": []any{
				map[string]any{"namespace": "billing", "name": "invoices"},
			},
		},
	}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}

	if result.Files["auth.txt"] != "namespace: auth" {
		t.Errorf("expected 'namespace: auth', got %q", result.Files["auth.txt"])
	}

	if result.Files["billing.txt"] != "namespace: billing" {
		t.Errorf("expected 'namespace: billing', got %q", result.Files["billing.txt"])
	}
}

func TestMultipleFiles(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		render({
			"file1.txt": "content one",
			"file2.txt": "content two",
			"dir/file3.txt": "content three"
		});
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if len(result.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(result.Files))
	}

	if result.Files["file1.txt"] != "content one" {
		t.Errorf("expected 'content one', got %q", result.Files["file1.txt"])
	}
	if result.Files["file2.txt"] != "content two" {
		t.Errorf("expected 'content two', got %q", result.Files["file2.txt"])
	}
	if result.Files["dir/file3.txt"] != "content three" {
		t.Errorf("expected 'content three', got %q", result.Files["dir/file3.txt"])
	}
}

func TestExportStripping(t *testing.T) {
	s := NewSandbox(nil)
	code := `export default gen((schema) => render({"a.txt": "works"}))`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if result.Files["a.txt"] != "works" {
		t.Errorf("expected 'works', got %q", result.Files["a.txt"])
	}
}

func TestPerTableEmptySchema(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const files = perTable(schema, (table) => {
			return {"file.txt": "content"};
		});
		render(files);
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 files for empty schema, got %d", len(result.Files))
	}
}

func TestPerNamespaceEmptySchema(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const files = perNamespace(schema, (ns, tables) => {
			return {"file.txt": "content"};
		});
		render(files);
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 files for empty schema, got %d", len(result.Files))
	}
}

func TestGeneratorAllowsDateAfterSchemaRestriction(t *testing.T) {
	s := NewSandbox(nil)

	// First, run a schema evaluation which restricts globals
	_, _ = s.EvalSchema(`table({ id: col.id() })`, "test", "restrict")

	if !s.restricted {
		t.Fatal("sandbox should be restricted after EvalSchema")
	}

	// Now run a generator — it should auto-reset and have full JS access
	code := `gen((schema) => {
		var d = new Date(2024, 0, 1);
		var ts = d.getFullYear().toString();
		var m = Math.floor(3.7).toString();
		var j = JSON.stringify({ok: true});
		render({"out.txt": ts + " " + m + " " + j});
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator should succeed with full JS access, got: %v", err)
	}

	content := result.Files["out.txt"]
	if !strings.Contains(content, "2024") {
		t.Errorf("expected Date to work in generator, got: %q", content)
	}
	if !strings.Contains(content, "3") {
		t.Errorf("expected Math to work in generator, got: %q", content)
	}
	if !strings.Contains(content, `"ok":true`) {
		t.Errorf("expected JSON to work in generator, got: %q", content)
	}
}

func TestGeneratorUsesMapAndSet(t *testing.T) {
	s := NewSandbox(nil)

	// Run a schema to trigger restriction
	_, _ = s.EvalSchema(`table({ id: col.id() })`, "test", "restrict")

	// Generator should still have Map and Set
	code := `gen((schema) => {
		var m = new Map();
		m.set("key", "value");
		var setObj = new Set([1, 2, 3]);
		render({"out.txt": m.get("key") + " " + setObj.size});
	})`
	schema := map[string]any{}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator should support Map/Set, got: %v", err)
	}

	content := result.Files["out.txt"]
	if !strings.Contains(content, "value") || !strings.Contains(content, "3") {
		t.Errorf("expected Map/Set to work in generator, got: %q", content)
	}
}

func TestComplexGenerator(t *testing.T) {
	s := NewSandbox(nil)
	code := `gen((schema) => {
		const files = {};

		// Use multiple helpers together
		const tableFiles = perTable(schema, (table) => {
			const content = "Table: " + table.name + "\nNamespace: " + table.namespace;
			return {["tables/" + table.name + ".txt"]: content};
		});

		// Merge with manual file
		const config = json(schema, "  ");

		render({
			...tableFiles,
			"config.json": config
		});
	})`
	schema := map[string]any{
		"project": "astroladb",
		"tables": []any{
			map[string]any{"namespace": "auth", "name": "users"},
		},
	}

	result, err := s.RunGenerator(code, schema)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}

	if !strings.Contains(result.Files["tables/users.txt"], "Table: users") {
		t.Errorf("expected table content with 'Table: users', got %q", result.Files["tables/users.txt"])
	}

	if !strings.Contains(result.Files["config.json"], "astroladb") {
		t.Errorf("expected config with 'astroladb', got %q", result.Files["config.json"])
	}
}
