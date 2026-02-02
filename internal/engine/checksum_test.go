package engine

import "testing"

func TestComputeSQLChecksum_Empty(t *testing.T) {
	result := ComputeSQLChecksum(nil)
	if result != "" {
		t.Errorf("expected empty string for nil input, got %q", result)
	}
	result = ComputeSQLChecksum([]string{})
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}
}

func TestComputeSQLChecksum_Deterministic(t *testing.T) {
	stmts := []string{"CREATE TABLE users (id INT)", "CREATE INDEX idx ON users (id)"}
	a := ComputeSQLChecksum(stmts)
	b := ComputeSQLChecksum(stmts)
	if a != b {
		t.Errorf("checksum not deterministic: %q != %q", a, b)
	}
	if len(a) != 64 {
		t.Errorf("expected 64-char hex SHA-256, got %d chars", len(a))
	}
}

func TestComputeSQLChecksum_OrderMatters(t *testing.T) {
	a := ComputeSQLChecksum([]string{"A", "B"})
	b := ComputeSQLChecksum([]string{"B", "A"})
	if a == b {
		t.Error("different statement order should produce different checksums")
	}
}

func TestComputeSQLChecksum_ContentMatters(t *testing.T) {
	a := ComputeSQLChecksum([]string{"CREATE TABLE a (id INT)"})
	b := ComputeSQLChecksum([]string{"CREATE TABLE b (id INT)"})
	if a == b {
		t.Error("different content should produce different checksums")
	}
}
