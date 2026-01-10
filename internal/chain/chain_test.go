package chain

import (
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
}
