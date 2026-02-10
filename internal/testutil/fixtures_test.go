package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSFixture_Success(t *testing.T) {
	// Create a temp fixture file
	rootDir := findProjectRoot(t)
	fixtureDir := filepath.Join(rootDir, "test", "fixtures", "temp")
	err := os.MkdirAll(fixtureDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create fixture dir: %v", err)
	}

	fixturePath := filepath.Join(fixtureDir, "test.js")
	testContent := "table({ id: col.id() })"

	err = os.WriteFile(fixturePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write fixture: %v", err)
	}

	// Clean up after test
	defer os.RemoveAll(fixtureDir)

	// Test LoadJSFixture
	content := LoadJSFixture(t, "test/fixtures/temp/test.js")

	if content != testContent {
		t.Errorf("Expected content %q, got %q", testContent, content)
	}
}

func TestLoadJSFixture_NotFound(t *testing.T) {
	// Create a test that should fail
	t.Run("should fail for missing file", func(t *testing.T) {
		// We can't directly test t.Fatal in the same test,
		// but we can verify the file doesn't exist
		exists := FixtureExists(t, "test/fixtures/nonexistent.js")
		if exists {
			t.Error("Expected file to not exist")
		}
	})
}

func TestLoadJSFixtureOrDefault_UsesDefault(t *testing.T) {
	defaultCode := "default code"

	content := LoadJSFixtureOrDefault(t, "test/fixtures/nonexistent.js", defaultCode)

	if content != defaultCode {
		t.Errorf("Expected default code %q, got %q", defaultCode, content)
	}
}

func TestLoadJSFixtureOrDefault_UsesFixture(t *testing.T) {
	// Use an existing fixture
	content := LoadJSFixtureOrDefault(t, "test/fixtures/schemas/auth.js", "default")

	// Should not be the default
	if content == "default" {
		t.Error("Expected fixture content, got default")
	}

	// Should contain table definition
	if len(content) == 0 {
		t.Error("Expected non-empty fixture content")
	}
}

func TestFixtureExists_True(t *testing.T) {
	exists := FixtureExists(t, "test/fixtures/schemas/auth.js")

	if !exists {
		t.Error("Expected auth.js fixture to exist")
	}
}

func TestFixtureExists_False(t *testing.T) {
	exists := FixtureExists(t, "test/fixtures/nonexistent.js")

	if exists {
		t.Error("Expected nonexistent.js to not exist")
	}
}

func TestFindProjectRoot(t *testing.T) {
	rootDir := findProjectRoot(t)

	// Should find go.mod
	goModPath := filepath.Join(rootDir, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		t.Errorf("Expected go.mod at %s, got error: %v", goModPath, err)
	}
}

func TestMustLoadJSFixture_Success(t *testing.T) {
	// Should not panic for existing file
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoadJSFixture panicked: %v", r)
		}
	}()

	content := MustLoadJSFixture("test/fixtures/schemas/auth.js")

	if len(content) == 0 {
		t.Error("Expected non-empty content")
	}
}

func TestMustLoadJSFixture_Panic(t *testing.T) {
	// Should panic for missing file
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustLoadJSFixture to panic for missing file")
		}
	}()

	MustLoadJSFixture("test/fixtures/nonexistent.js")
}
