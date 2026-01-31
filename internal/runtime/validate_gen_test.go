package runtime

import (
	"strings"
	"testing"
)

func TestValidateRenderPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		// Path traversal attacks
		{
			name:    "path traversal with ..",
			path:    "../etc/passwd",
			wantErr: true,
			errMsg:  "must not contain '..'",
		},
		{
			name:    "path traversal in middle",
			path:    "foo/../bar",
			wantErr: true,
			errMsg:  "must not contain '..'",
		},
		{
			name:    "absolute path",
			path:    "/absolute/path",
			wantErr: true,
			errMsg:  "must be relative",
		},
		// Backslash (OS-specific separator)
		{
			name:    "backslash separator",
			path:    `C:\windows\file`,
			wantErr: true,
			errMsg:  "must use forward slashes",
		},
		// Dotfiles/hidden files
		{
			name:    "dotfile at root",
			path:    ".hidden",
			wantErr: true,
			errMsg:  "must not contain hidden segments",
		},
		{
			name:    "dotfile in segment",
			path:    "foo/.secret/bar",
			wantErr: true,
			errMsg:  "must not contain hidden segments",
		},
		// Empty/whitespace
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "must not be empty",
		},
		{
			name:    "whitespace only",
			path:    "  ",
			wantErr: true,
			errMsg:  "must not be empty",
		},
		// Control characters
		{
			name:    "path with control characters",
			path:    "path\x00with\x01control",
			wantErr: true,
			errMsg:  "contains control character",
		},
		// Windows reserved names
		{
			name:    "CON reserved name",
			path:    "CON",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		{
			name:    "con.txt reserved name with extension",
			path:    "con.txt",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		{
			name:    "NUL in segment",
			path:    "foo/NUL/bar",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		{
			name:    "PRN reserved",
			path:    "prn",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		{
			name:    "AUX reserved",
			path:    "aux.log",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		{
			name:    "COM1 reserved",
			path:    "COM1",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		{
			name:    "LPT1 reserved",
			path:    "logs/LPT1.txt",
			wantErr: true,
			errMsg:  "contains Windows reserved name",
		},
		// Valid paths
		{
			name:    "spaces in path are ok",
			path:    "path/with spaces/ok.txt",
			wantErr: false,
		},
		{
			name:    "simple filename",
			path:    "models.py",
			wantErr: false,
		},
		{
			name:    "nested path",
			path:    "routers/auth.py",
			wantErr: false,
		},
		{
			name:    "deep nested path",
			path:    "deep/nested/path/file.js",
			wantErr: false,
		},
		{
			name:    "multiple extensions",
			path:    "backup.tar.gz",
			wantErr: false,
		},
		{
			name:    "numbers and underscores",
			path:    "test_123/file_v2.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRenderPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRenderPath(%q) expected error containing %q, got nil", tt.path, tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRenderPath(%q) error = %q, want error containing %q", tt.path, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRenderPath(%q) unexpected error: %v", tt.path, err)
				}
			}
		})
	}
}

func TestValidateRenderOutput(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		maxBytes int64
		maxFiles int
		wantErr  bool
		errMsg   string
	}{
		// Duplicate keys after normalization
		{
			name: "paths collapse to same normalized form",
			files: map[string]string{
				"a/./b": "content1",
				"a/b":   "content2",
			},
			wantErr: true,
			errMsg:  "duplicate paths",
		},
		{
			name: "double slash normalization collision",
			files: map[string]string{
				"a//b": "content1",
				"a/b":  "content2",
			},
			wantErr: true,
			errMsg:  "duplicate paths",
		},
		// Size limits
		{
			name: "total size exceeds max bytes",
			files: map[string]string{
				"file1.txt": strings.Repeat("x", 100),
				"file2.txt": strings.Repeat("y", 100),
			},
			maxBytes: 150,
			wantErr:  true,
			errMsg:   "exceeds maximum size",
		},
		// File count limits
		{
			name: "file count exceeds max",
			files: map[string]string{
				"file1.txt": "content",
				"file2.txt": "content",
				"file3.txt": "content",
			},
			maxFiles: 2,
			wantErr:  true,
			errMsg:   "exceeds maximum file count",
		},
		// Invalid paths in output
		{
			name: "invalid path in output",
			files: map[string]string{
				"valid.txt":   "ok",
				"../etc/pass": "bad",
			},
			wantErr: true,
			errMsg:  "must not contain '..'",
		},
		{
			name: "hidden file in output",
			files: map[string]string{
				"visible.txt": "ok",
				".hidden":     "bad",
			},
			wantErr: true,
			errMsg:  "must not contain hidden segments",
		},
		// Valid outputs
		{
			name: "valid output passes",
			files: map[string]string{
				"models.py":        "class Model: pass",
				"routers/auth.py":  "def login(): pass",
				"utils/helpers.js": "export const help = () => {}",
			},
			maxBytes: 1000,
			maxFiles: 10,
			wantErr:  false,
		},
		{
			name:     "empty output is ok",
			files:    map[string]string{},
			maxBytes: 100,
			maxFiles: 5,
			wantErr:  false,
		},
		{
			name: "unlimited size and files",
			files: map[string]string{
				"large1.txt": strings.Repeat("data", 10000),
				"large2.txt": strings.Repeat("data", 10000),
				"large3.txt": strings.Repeat("data", 10000),
			},
			maxBytes: 0, // unlimited
			maxFiles: 0, // unlimited
			wantErr:  false,
		},
		{
			name: "exactly at byte limit",
			files: map[string]string{
				"file.txt": "12345",
			},
			maxBytes: 5,
			wantErr:  false,
		},
		{
			name: "exactly at file limit",
			files: map[string]string{
				"file1.txt": "a",
				"file2.txt": "b",
			},
			maxFiles: 2,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRenderOutput(tt.files, tt.maxBytes, tt.maxFiles)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRenderOutput() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRenderOutput() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRenderOutput() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateRenderOutputEdgeCases tests additional edge cases
func TestValidateRenderOutputEdgeCases(t *testing.T) {
	t.Run("multiple invalid paths all reported", func(t *testing.T) {
		files := map[string]string{
			"valid.txt": "ok",
			"../bad1":   "content",
		}
		err := ValidateRenderOutput(files, 0, 0)
		if err == nil {
			t.Error("expected error for invalid path, got nil")
		}
	})

	t.Run("windows reserved name in output", func(t *testing.T) {
		files := map[string]string{
			"output/CON.txt": "content",
		}
		err := ValidateRenderOutput(files, 0, 0)
		if err == nil {
			t.Error("expected error for Windows reserved name, got nil")
		}
		if !strings.Contains(err.Error(), "Windows reserved name") {
			t.Errorf("error = %q, want error containing 'Windows reserved name'", err.Error())
		}
	})

	t.Run("control characters in path", func(t *testing.T) {
		files := map[string]string{
			"bad\x00file.txt": "content",
		}
		err := ValidateRenderOutput(files, 0, 0)
		if err == nil {
			t.Error("expected error for control character, got nil")
		}
		if !strings.Contains(err.Error(), "control character") {
			t.Errorf("error = %q, want error containing 'control character'", err.Error())
		}
	})
}
