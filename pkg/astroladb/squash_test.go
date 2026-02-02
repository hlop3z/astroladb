package astroladb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSquashedThrough(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid baseline header",
			content: "// Baseline: squashed from 5 migrations (through revision 003)\nmodule.exports = function() {}",
			want:    "003",
		},
		{
			name:    "no header",
			content: "module.exports = function() {}",
			want:    "",
		},
		{
			name:    "malformed header",
			content: "// Baseline: squashed from 5 migrations\nmodule.exports = function() {}",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".js")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			got := parseSquashedThrough(path)
			if got != tt.want {
				t.Errorf("parseSquashedThrough() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSquashedThrough_FileNotFound(t *testing.T) {
	got := parseSquashedThrough("/nonexistent/file.js")
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}
