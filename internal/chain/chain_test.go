package chain

import (
	"os"
	"testing"
)

func TestComputeChecksum(t *testing.T) {
	// Same content + same prev should give same checksum
	content := []byte("export function up(m) {}")
	prev := "genesis"

	checksum1 := computeChecksum(content, prev)
	checksum2 := computeChecksum(content, prev)

	if checksum1 != checksum2 {
		t.Errorf("same input should produce same checksum")
	}

	// Different content should give different checksum
	content2 := []byte("export function up(m) { m.create_table() }")
	checksum3 := computeChecksum(content2, prev)

	if checksum1 == checksum3 {
		t.Errorf("different content should produce different checksum")
	}

	// Different prev should give different checksum
	checksum4 := computeChecksum(content, "different_prev")

	if checksum1 == checksum4 {
		t.Errorf("different prev should produce different checksum")
	}
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		filename string
		wantRev  string
		wantName string
	}{
		{"001_initial.js", "001", "initial"},
		{"002_add_users.js", "002", "add_users"},
		{"123_complex_name_with_underscores.js", "123", "complex_name_with_underscores"},
		{"001.js", "001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			rev, name := parseFilename(tt.filename)
			if rev != tt.wantRev {
				t.Errorf("revision: got %q, want %q", rev, tt.wantRev)
			}
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestComputeFromFiles(t *testing.T) {
	files := map[string][]byte{
		"001_first.js":  []byte("content1"),
		"002_second.js": []byte("content2"),
		"003_third.js":  []byte("content3"),
	}

	chain := ComputeFromFiles(files)

	if len(chain.Links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(chain.Links))
	}

	// Verify chain structure
	if chain.Links[0].PrevChecksum != GenesisChecksum {
		t.Errorf("first link should have genesis as prev")
	}

	if chain.Links[1].PrevChecksum != chain.Links[0].Checksum {
		t.Errorf("second link should chain from first")
	}

	if chain.Links[2].PrevChecksum != chain.Links[1].Checksum {
		t.Errorf("third link should chain from second")
	}

	// Verify revisions
	if chain.Links[0].Revision != "001" {
		t.Errorf("first revision should be 001")
	}
	if chain.Links[1].Revision != "002" {
		t.Errorf("second revision should be 002")
	}
	if chain.Links[2].Revision != "003" {
		t.Errorf("third revision should be 003")
	}
}

func TestChainVerify_AllApplied(t *testing.T) {
	files := map[string][]byte{
		"001_first.js":  []byte("content1"),
		"002_second.js": []byte("content2"),
	}

	chain := ComputeFromFiles(files)

	applied := []AppliedMigration{
		{Revision: "001", Checksum: chain.Links[0].Checksum},
		{Revision: "002", Checksum: chain.Links[1].Checksum},
	}

	result := chain.Verify(applied)

	if !result.Valid {
		t.Errorf("chain should be valid")
	}

	if len(result.AppliedLinks) != 2 {
		t.Errorf("expected 2 applied, got %d", len(result.AppliedLinks))
	}

	if len(result.PendingLinks) != 0 {
		t.Errorf("expected 0 pending, got %d", len(result.PendingLinks))
	}
}

func TestChainVerify_SomePending(t *testing.T) {
	files := map[string][]byte{
		"001_first.js":  []byte("content1"),
		"002_second.js": []byte("content2"),
		"003_third.js":  []byte("content3"),
	}

	chain := ComputeFromFiles(files)

	// Only first migration applied
	applied := []AppliedMigration{
		{Revision: "001", Checksum: chain.Links[0].Checksum},
	}

	result := chain.Verify(applied)

	if !result.Valid {
		t.Errorf("chain should be valid")
	}

	if len(result.AppliedLinks) != 1 {
		t.Errorf("expected 1 applied, got %d", len(result.AppliedLinks))
	}

	if len(result.PendingLinks) != 2 {
		t.Errorf("expected 2 pending, got %d", len(result.PendingLinks))
	}
}

func TestChainVerify_Tampered(t *testing.T) {
	files := map[string][]byte{
		"001_first.js":  []byte("content1"),
		"002_second.js": []byte("content2"),
	}

	chain := ComputeFromFiles(files)

	// Applied with different checksum (simulating modified file)
	applied := []AppliedMigration{
		{Revision: "001", Checksum: "different_checksum"},
	}

	result := chain.Verify(applied)

	if result.Valid {
		t.Errorf("chain should be invalid when tampered")
	}

	if len(result.MismatchedLinks) != 1 {
		t.Errorf("expected 1 mismatched, got %d", len(result.MismatchedLinks))
	}

	if len(result.Errors) == 0 {
		t.Errorf("expected errors for tampered file")
	}

	if result.Errors[0].Type != ErrorTampered {
		t.Errorf("expected ErrorTampered, got %v", result.Errors[0].Type)
	}
}

func TestChainVerify_MissingFile(t *testing.T) {
	files := map[string][]byte{
		"001_first.js": []byte("content1"),
		// 002 is missing
		"003_third.js": []byte("content3"),
	}

	chain := ComputeFromFiles(files)

	// Database has 002 applied
	applied := []AppliedMigration{
		{Revision: "001", Checksum: chain.Links[0].Checksum},
		{Revision: "002", Checksum: "some_checksum"},
	}

	result := chain.Verify(applied)

	if result.Valid {
		t.Errorf("chain should be invalid with missing file")
	}

	if len(result.MissingFiles) != 1 {
		t.Errorf("expected 1 missing, got %d", len(result.MissingFiles))
	}

	if result.MissingFiles[0].Revision != "002" {
		t.Errorf("expected missing revision 002")
	}
}

func TestChainLastChecksum(t *testing.T) {
	// Empty chain
	emptyChain := &Chain{}
	if emptyChain.LastChecksum() != GenesisChecksum {
		t.Errorf("empty chain should return genesis")
	}

	// Chain with links
	files := map[string][]byte{
		"001_first.js": []byte("content1"),
	}
	chain := ComputeFromFiles(files)

	if chain.LastChecksum() != chain.Links[0].Checksum {
		t.Errorf("should return last link's checksum")
	}
}

func TestChainGetPendingAfter(t *testing.T) {
	files := map[string][]byte{
		"001_first.js":  []byte("content1"),
		"002_second.js": []byte("content2"),
		"003_third.js":  []byte("content3"),
	}

	chain := ComputeFromFiles(files)

	// No applied
	pending := chain.GetPendingAfter("")
	if len(pending) != 3 {
		t.Errorf("expected 3 pending from start, got %d", len(pending))
	}

	// After 001
	pending = chain.GetPendingAfter("001")
	if len(pending) != 2 {
		t.Errorf("expected 2 pending after 001, got %d", len(pending))
	}

	// After 003
	pending = chain.GetPendingAfter("003")
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after 003, got %d", len(pending))
	}

	// Unknown revision - returns all
	pending = chain.GetPendingAfter("unknown")
	if len(pending) != 3 {
		t.Errorf("expected 3 pending for unknown revision, got %d", len(pending))
	}
}

// -----------------------------------------------------------------------------
// GetLink Tests
// -----------------------------------------------------------------------------

func TestChainGetLink(t *testing.T) {
	files := map[string][]byte{
		"001_first.js":  []byte("content1"),
		"002_second.js": []byte("content2"),
	}
	chain := ComputeFromFiles(files)

	t.Run("found", func(t *testing.T) {
		link := chain.GetLink("001")
		if link == nil {
			t.Fatal("expected to find link 001")
		}
		if link.Revision != "001" {
			t.Errorf("revision = %q, want %q", link.Revision, "001")
		}
		if link.Name != "first" {
			t.Errorf("name = %q, want %q", link.Name, "first")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		link := chain.GetLink("999")
		if link != nil {
			t.Error("expected nil for non-existent revision")
		}
	})

	t.Run("second_link", func(t *testing.T) {
		link := chain.GetLink("002")
		if link == nil {
			t.Fatal("expected to find link 002")
		}
		if link.PrevChecksum != chain.Links[0].Checksum {
			t.Error("second link should chain from first")
		}
	})
}

// -----------------------------------------------------------------------------
// ComputeFromDir Tests
// -----------------------------------------------------------------------------

func TestComputeFromDir(t *testing.T) {
	t.Run("non_existent_dir", func(t *testing.T) {
		chain, err := ComputeFromDir("/non/existent/path/migrations")
		if err != nil {
			t.Errorf("non-existent dir should return empty chain, not error: %v", err)
		}
		if chain == nil {
			t.Fatal("chain should not be nil")
		}
		if len(chain.Links) != 0 {
			t.Errorf("expected 0 links, got %d", len(chain.Links))
		}
	})

	t.Run("empty_dir", func(t *testing.T) {
		dir := t.TempDir()

		chain, err := ComputeFromDir(dir)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(chain.Links) != 0 {
			t.Errorf("expected 0 links, got %d", len(chain.Links))
		}
	})

	t.Run("with_migration_files", func(t *testing.T) {
		dir := t.TempDir()

		// Create migration files
		files := map[string]string{
			"001_init.js":      "export function up(m) {}",
			"002_add_users.js": "export function up(m) { m.create_table('auth.users') }",
		}
		for name, content := range files {
			path := dir + "/" + name
			if err := writeTestFile(path, content); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
		}

		chain, err := ComputeFromDir(dir)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(chain.Links) != 2 {
			t.Errorf("expected 2 links, got %d", len(chain.Links))
		}
		if chain.Links[0].Revision != "001" {
			t.Errorf("first revision = %q, want %q", chain.Links[0].Revision, "001")
		}
		if chain.Links[1].Revision != "002" {
			t.Errorf("second revision = %q, want %q", chain.Links[1].Revision, "002")
		}
	})

	t.Run("ignores_non_js_files", func(t *testing.T) {
		dir := t.TempDir()

		// Create various files
		files := map[string]string{
			"001_init.js":   "export function up(m) {}",
			"README.md":     "# Migrations",
			"002_test.txt":  "text file",
			"003_other.sql": "SELECT 1;",
		}
		for name, content := range files {
			path := dir + "/" + name
			if err := writeTestFile(path, content); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
		}

		chain, err := ComputeFromDir(dir)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(chain.Links) != 1 {
			t.Errorf("expected 1 link (only .js), got %d", len(chain.Links))
		}
	})

	t.Run("ignores_non_numbered_files", func(t *testing.T) {
		dir := t.TempDir()

		// Create various files
		files := map[string]string{
			"001_init.js":   "export function up(m) {}",
			"helper.js":     "function helper() {}",
			"_rollback.js":  "export function down() {}",
			"utils/test.js": "ignored",
		}
		for name, content := range files {
			path := dir + "/" + name
			if err := writeTestFile(path, content); err != nil {
				// utils/test.js will fail - that's expected
				continue
			}
		}

		chain, err := ComputeFromDir(dir)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(chain.Links) != 1 {
			t.Errorf("expected 1 link (only numbered .js), got %d", len(chain.Links))
		}
	})

	t.Run("sorted_by_filename", func(t *testing.T) {
		dir := t.TempDir()

		// Create files in reverse order
		files := map[string]string{
			"003_third.js":  "content3",
			"001_first.js":  "content1",
			"002_second.js": "content2",
		}
		for name, content := range files {
			path := dir + "/" + name
			if err := writeTestFile(path, content); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
		}

		chain, err := ComputeFromDir(dir)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		// Should be sorted
		if chain.Links[0].Revision != "001" {
			t.Error("first link should be 001")
		}
		if chain.Links[1].Revision != "002" {
			t.Error("second link should be 002")
		}
		if chain.Links[2].Revision != "003" {
			t.Error("third link should be 003")
		}
	})

	t.Run("ignores_subdirectories", func(t *testing.T) {
		dir := t.TempDir()

		// Create a migration file and a subdirectory
		files := map[string]string{
			"001_init.js": "export function up(m) {}",
		}
		for name, content := range files {
			path := dir + "/" + name
			if err := writeTestFile(path, content); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
		}

		// Create a subdirectory
		if err := os.Mkdir(dir+"/subdir", 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		chain, err := ComputeFromDir(dir)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if len(chain.Links) != 1 {
			t.Errorf("expected 1 link (subdirectory ignored), got %d", len(chain.Links))
		}
	})
}

// -----------------------------------------------------------------------------
// FormatFilename Tests
// -----------------------------------------------------------------------------

func TestFormatFilename(t *testing.T) {
	tests := []struct {
		revision string
		name     string
		want     string
	}{
		{"001", "init", "001_init.js"},
		{"002", "add_users", "002_add_users.js"},
		{"123", "complex_migration_name", "123_complex_migration_name.js"},
		{"01", "short", "01_short.js"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatFilename(tt.revision, tt.name)
			if got != tt.want {
				t.Errorf("FormatFilename(%q, %q) = %q, want %q", tt.revision, tt.name, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// ComputeFromFiles Edge Cases
// -----------------------------------------------------------------------------

func TestComputeFromFiles_EdgeCases(t *testing.T) {
	t.Run("empty_map", func(t *testing.T) {
		chain := ComputeFromFiles(map[string][]byte{})
		if len(chain.Links) != 0 {
			t.Errorf("expected 0 links, got %d", len(chain.Links))
		}
	})

	t.Run("nil_map", func(t *testing.T) {
		chain := ComputeFromFiles(nil)
		if len(chain.Links) != 0 {
			t.Errorf("expected 0 links, got %d", len(chain.Links))
		}
	})

	t.Run("filters_non_js_files", func(t *testing.T) {
		files := map[string][]byte{
			"001_first.js": []byte("content1"),
			"readme.md":    []byte("# readme"),
			"002.txt":      []byte("text"),
		}
		chain := ComputeFromFiles(files)
		if len(chain.Links) != 1 {
			t.Errorf("expected 1 link, got %d", len(chain.Links))
		}
	})

	t.Run("filters_non_numbered_js", func(t *testing.T) {
		files := map[string][]byte{
			"001_first.js": []byte("content1"),
			"helper.js":    []byte("function help() {}"),
			"_test.js":     []byte("test"),
		}
		chain := ComputeFromFiles(files)
		if len(chain.Links) != 1 {
			t.Errorf("expected 1 link, got %d", len(chain.Links))
		}
	})

	t.Run("content_preserved", func(t *testing.T) {
		content := []byte("export function up(m) { m.create_table() }")
		files := map[string][]byte{
			"001_test.js": content,
		}
		chain := ComputeFromFiles(files)
		if string(chain.Links[0].Content) != string(content) {
			t.Error("content should be preserved in link")
		}
	})
}

// -----------------------------------------------------------------------------
// Chain Integrity Tests
// -----------------------------------------------------------------------------

func TestChainIntegrity(t *testing.T) {
	t.Run("chain_modification_breaks_subsequent", func(t *testing.T) {
		// Simulate a scenario where a migration is modified
		original := map[string][]byte{
			"001_first.js":  []byte("original content"),
			"002_second.js": []byte("second content"),
		}
		originalChain := ComputeFromFiles(original)

		// Modify the first file
		modified := map[string][]byte{
			"001_first.js":  []byte("modified content"),
			"002_second.js": []byte("second content"),
		}
		modifiedChain := ComputeFromFiles(modified)

		// Checksums should differ
		if originalChain.Links[0].Checksum == modifiedChain.Links[0].Checksum {
			t.Error("modified content should produce different checksum")
		}

		// Second link's checksum should also differ (chain effect)
		if originalChain.Links[1].Checksum == modifiedChain.Links[1].Checksum {
			t.Error("chain effect: second link should also have different checksum")
		}
	})
}

// writeTestFile is a helper to create test files
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
