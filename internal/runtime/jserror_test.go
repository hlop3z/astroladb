package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// -----------------------------------------------------------------------------
// GetSourceLineFromFile Tests
// -----------------------------------------------------------------------------

func TestGetSourceLineFromFile(t *testing.T) {
	t.Run("valid_file_and_line", func(t *testing.T) {
		// Create a temporary test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.js")

		content := `line 1
line 2
line 3
line 4
line 5`

		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test reading line 3
		result := GetSourceLineFromFile(testFile, 3)
		if result != "line 3" {
			t.Errorf("GetSourceLineFromFile(line 3) = %q, want %q", result, "line 3")
		}

		// Test reading line 1
		result = GetSourceLineFromFile(testFile, 1)
		if result != "line 1" {
			t.Errorf("GetSourceLineFromFile(line 1) = %q, want %q", result, "line 1")
		}

		// Test reading last line
		result = GetSourceLineFromFile(testFile, 5)
		if result != "line 5" {
			t.Errorf("GetSourceLineFromFile(line 5) = %q, want %q", result, "line 5")
		}
	})

	t.Run("line_number_zero", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.js")

		err := os.WriteFile(testFile, []byte("line 1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := GetSourceLineFromFile(testFile, 0)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(line 0) = %q, want empty string", result)
		}
	})

	t.Run("negative_line_number", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.js")

		err := os.WriteFile(testFile, []byte("line 1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := GetSourceLineFromFile(testFile, -1)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(line -1) = %q, want empty string", result)
		}
	})

	t.Run("empty_path", func(t *testing.T) {
		result := GetSourceLineFromFile("", 1)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(empty path) = %q, want empty string", result)
		}
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		result := GetSourceLineFromFile("/nonexistent/path/file.js", 1)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(nonexistent file) = %q, want empty string", result)
		}
	})

	t.Run("line_beyond_file_end", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.js")

		content := "line 1\nline 2\nline 3"
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Try to read line 100 (file only has 3 lines)
		result := GetSourceLineFromFile(testFile, 100)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(line 100) = %q, want empty string", result)
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "empty.js")

		err := os.WriteFile(testFile, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result := GetSourceLineFromFile(testFile, 1)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(empty file, line 1) = %q, want empty string", result)
		}
	})

	t.Run("file_with_empty_lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.js")

		content := `line 1

line 3

line 5`

		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test reading empty line 2
		result := GetSourceLineFromFile(testFile, 2)
		if result != "" {
			t.Errorf("GetSourceLineFromFile(empty line 2) = %q, want empty string", result)
		}

		// Test reading non-empty line 3
		result = GetSourceLineFromFile(testFile, 3)
		if result != "line 3" {
			t.Errorf("GetSourceLineFromFile(line 3) = %q, want %q", result, "line 3")
		}
	})

	t.Run("large_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "large.js")

		// Create a file with 1000 lines
		var content string
		for i := 1; i <= 1000; i++ {
			if i > 1 {
				content += "\n"
			}
			content += "line " + string(rune('0'+i%10))
		}

		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test reading line 500
		result := GetSourceLineFromFile(testFile, 500)
		if result == "" {
			t.Error("GetSourceLineFromFile(line 500) should return non-empty result")
		}

		// Test reading line 999
		result = GetSourceLineFromFile(testFile, 999)
		if result == "" {
			t.Error("GetSourceLineFromFile(line 999) should return non-empty result")
		}
	})

	t.Run("file_with_special_characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "special.js")

		content := `function test() { return "hello"; }
const x = 42;
// Comment with special chars: !@#$%^&*()
let unicode = "Hello 世界 🌍"`

		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test reading line with special characters
		result := GetSourceLineFromFile(testFile, 3)
		if result != "// Comment with special chars: !@#$%^&*()" {
			t.Errorf("GetSourceLineFromFile(line 3) = %q, want comment line", result)
		}

		// Test reading line with unicode
		result = GetSourceLineFromFile(testFile, 4)
		if result == "" {
			t.Error("GetSourceLineFromFile(line 4) should return unicode line")
		}
	})

	// CRITICAL TEST: Validates that line numbers match Goja's error reporting.
	// This test replicates the exact scenario from user bug reports where
	// error messages showed the wrong source line.
	//
	// Background: Goja uses 1-indexed line numbers (first line is 1).
	// GetSourceLineFromFile must maintain this convention to avoid off-by-one errors.
	// If this test fails, errors will display incorrect source lines.
	t.Run("goja_error_line_accuracy", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "user.js")

		// Exact content from bug report that triggered wrong line display
		content := `export default table({
  id: col.id(),
  email: col.email().unique(),
  username: col.username().unique(),
  password: col.password_hash(),
  is_active: col.flag(true),
  note: col.string(),
}).timestamps();`

		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// CRITICAL: When Goja reports line 7, we MUST return line 7's content.
		// Bug was: line 7 request returned line 6's content ("is_active...")
		// Correct: line 7 should return "  note: col.string(),"
		result := GetSourceLineFromFile(testFile, 7)
		expected := "  note: col.string(),"
		if result != expected {
			t.Errorf("CRITICAL BUG: GetSourceLineFromFile(line 7)\n  got:  %q\n  want: %q\nThis will cause error messages to show wrong source lines!",
				result, expected)
		}

		// Verify line 6 for comparison
		result6 := GetSourceLineFromFile(testFile, 6)
		expected6 := "  is_active: col.flag(true),"
		if result6 != expected6 {
			t.Errorf("GetSourceLineFromFile(line 6) = %q, want %q", result6, expected6)
		}

		// Verify line 1
		result1 := GetSourceLineFromFile(testFile, 1)
		if result1 != "export default table({" {
			t.Errorf("GetSourceLineFromFile(line 1) = %q, want %q", result1, "export default table({")
		}

		// Verify last line
		result8 := GetSourceLineFromFile(testFile, 8)
		if result8 != "}).timestamps();" {
			t.Errorf("GetSourceLineFromFile(line 8) = %q, want %q", result8, "}).timestamps();")
		}
	})
}
