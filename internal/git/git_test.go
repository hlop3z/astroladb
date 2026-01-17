package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// initGitRepo initializes a new git repository in the given directory.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")
}

// runGitCmd runs a git command in the given directory.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

// createFile creates a file with the given content.
func createFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
}

// -----------------------------------------------------------------------------
// Open Tests
// -----------------------------------------------------------------------------

func TestOpen(t *testing.T) {
	t.Run("valid_git_repo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		if repo == nil {
			t.Fatal("Open() returned nil repo")
		}
		if repo.RootDir() == "" {
			t.Error("RootDir() returned empty string")
		}
	})

	t.Run("subdirectory_of_git_repo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		subdir := filepath.Join(dir, "subdir")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		repo, err := Open(subdir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		if repo == nil {
			t.Fatal("Open() returned nil repo")
		}
	})

	t.Run("non_git_directory", func(t *testing.T) {
		dir := t.TempDir()

		_, err := Open(dir)
		if err == nil {
			t.Fatal("Open() expected error for non-git directory")
		}

		code := alerr.GetErrorCode(err)
		if code != alerr.ENotGitRepo {
			t.Errorf("Open() error code = %v, want %v", code, alerr.ENotGitRepo)
		}
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		_, err := Open("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Fatal("Open() expected error for nonexistent directory")
		}
	})
}

// -----------------------------------------------------------------------------
// Info Tests
// -----------------------------------------------------------------------------

func TestRepoInfo(t *testing.T) {
	t.Run("basic_repo_info", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		info, err := repo.Info()
		if err != nil {
			t.Fatalf("Info() error = %v", err)
		}

		if !info.IsRepo {
			t.Error("Info().IsRepo = false, want true")
		}
		if info.RootDir == "" {
			t.Error("Info().RootDir is empty")
		}
		if info.CurrentBranch == "" {
			t.Error("Info().CurrentBranch is empty")
		}
	})

	t.Run("dirty_repo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create uncommitted file
		createFile(t, filepath.Join(dir, "new.txt"), "new content")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		info, err := repo.Info()
		if err != nil {
			t.Fatalf("Info() error = %v", err)
		}

		if !info.IsDirty {
			t.Error("Info().IsDirty = false, want true")
		}
	})

	t.Run("clean_repo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		info, err := repo.Info()
		if err != nil {
			t.Fatalf("Info() error = %v", err)
		}

		if info.IsDirty {
			t.Error("Info().IsDirty = true, want false")
		}
	})
}

// -----------------------------------------------------------------------------
// IsTracked Tests
// -----------------------------------------------------------------------------

func TestIsTracked(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create and commit a tracked file
	trackedFile := filepath.Join(dir, "tracked.txt")
	createFile(t, trackedFile, "tracked content")
	runGitCmd(t, dir, "add", "tracked.txt")
	runGitCmd(t, dir, "commit", "-m", "Add tracked file")

	// Create an untracked file
	untrackedFile := filepath.Join(dir, "untracked.txt")
	createFile(t, untrackedFile, "untracked content")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"tracked_file", trackedFile, true},
		{"untracked_file", untrackedFile, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracked, err := repo.IsTracked(tt.path)
			if err != nil {
				t.Fatalf("IsTracked() error = %v", err)
			}
			if tracked != tt.expected {
				t.Errorf("IsTracked() = %v, want %v", tracked, tt.expected)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// FileStatus Tests
// -----------------------------------------------------------------------------

func TestFileStatus(t *testing.T) {
	t.Run("committed_file", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create and commit a file
		committedFile := filepath.Join(dir, "committed.txt")
		createFile(t, committedFile, "committed content")
		runGitCmd(t, dir, "add", "committed.txt")
		runGitCmd(t, dir, "commit", "-m", "Add committed file")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		status, err := repo.FileStatus(committedFile)
		if err != nil {
			t.Fatalf("FileStatus() error = %v", err)
		}
		if status != StatusCommitted {
			t.Errorf("FileStatus() = %v, want %v", status, StatusCommitted)
		}
	})

	t.Run("untracked_file", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create an untracked file
		untrackedFile := filepath.Join(dir, "untracked.txt")
		createFile(t, untrackedFile, "untracked content")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		status, err := repo.FileStatus(untrackedFile)
		if err != nil {
			t.Fatalf("FileStatus() error = %v", err)
		}
		if status != StatusUntracked {
			t.Errorf("FileStatus() = %v, want %v", status, StatusUntracked)
		}
	})

	t.Run("modified_file", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create and commit a file
		modifiedFile := filepath.Join(dir, "modified.txt")
		createFile(t, modifiedFile, "original content")
		runGitCmd(t, dir, "add", "modified.txt")
		runGitCmd(t, dir, "commit", "-m", "Add file")

		// Modify the file (unstaged changes)
		createFile(t, modifiedFile, "modified content")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		status, err := repo.FileStatus(modifiedFile)
		if err != nil {
			t.Fatalf("FileStatus() error = %v", err)
		}
		// A modified file can be StatusModified or StatusStaged depending on git autocrlf settings
		if status != StatusModified && status != StatusStaged {
			t.Errorf("FileStatus() = %v, want StatusModified or StatusStaged", status)
		}
	})

	t.Run("staged_file", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create a staged file
		stagedFile := filepath.Join(dir, "staged.txt")
		createFile(t, stagedFile, "staged content")
		runGitCmd(t, dir, "add", "staged.txt")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		status, err := repo.FileStatus(stagedFile)
		if err != nil {
			t.Fatalf("FileStatus() error = %v", err)
		}
		if status != StatusStaged {
			t.Errorf("FileStatus() = %v, want %v", status, StatusStaged)
		}
	})
}

// -----------------------------------------------------------------------------
// UncommittedMigrations Tests
// -----------------------------------------------------------------------------

func TestUncommittedMigrations(t *testing.T) {
	t.Run("finds_uncommitted_js_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create migrations directory with uncommitted .js files
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration 1")
		createFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), "// migration 2")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		files, err := repo.UncommittedMigrations(migrationsDir)
		if err != nil {
			t.Fatalf("UncommittedMigrations() error = %v", err)
		}

		if len(files) != 2 {
			t.Errorf("UncommittedMigrations() = %d files, want 2", len(files))
		}
	})

	t.Run("filters_non_js_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create migrations directory with mixed files
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration")
		createFile(t, filepath.Join(migrationsDir, "README.md"), "# Readme")
		createFile(t, filepath.Join(migrationsDir, "config.json"), "{}")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		files, err := repo.UncommittedMigrations(migrationsDir)
		if err != nil {
			t.Fatalf("UncommittedMigrations() error = %v", err)
		}

		// Should only include .js files
		if len(files) != 1 {
			t.Errorf("UncommittedMigrations() = %d files, want 1", len(files))
		}
	})

	t.Run("empty_migrations_dir", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		files, err := repo.UncommittedMigrations(filepath.Join(dir, "migrations"))
		if err != nil {
			t.Fatalf("UncommittedMigrations() error = %v", err)
		}

		if len(files) != 0 {
			t.Errorf("UncommittedMigrations() = %d files, want 0", len(files))
		}
	})
}

// -----------------------------------------------------------------------------
// Add and Commit Tests
// -----------------------------------------------------------------------------

func TestAddAndCommit(t *testing.T) {
	t.Run("add_and_commit_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		// Create new files
		newFile := filepath.Join(dir, "new.txt")
		createFile(t, newFile, "new content")

		// Add and commit
		err = repo.Add(newFile)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		err = repo.Commit("Add new file")
		if err != nil {
			t.Fatalf("Commit() error = %v", err)
		}

		// Verify file is now committed
		status, err := repo.FileStatus(newFile)
		if err != nil {
			t.Fatalf("FileStatus() error = %v", err)
		}
		if status != StatusCommitted {
			t.Errorf("FileStatus() = %v, want %v", status, StatusCommitted)
		}
	})

	t.Run("add_empty_paths", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		// Add with empty paths should not error
		err = repo.Add()
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	})

	t.Run("commit_files_helper", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		// Create and commit using CommitFiles
		newFile := filepath.Join(dir, "helper.txt")
		createFile(t, newFile, "helper content")

		err = repo.CommitFiles("Add helper file", newFile)
		if err != nil {
			t.Fatalf("CommitFiles() error = %v", err)
		}

		// Verify file is committed
		status, err := repo.FileStatus(newFile)
		if err != nil {
			t.Fatalf("FileStatus() error = %v", err)
		}
		if status != StatusCommitted {
			t.Errorf("FileStatus() = %v, want %v", status, StatusCommitted)
		}
	})
}

// -----------------------------------------------------------------------------
// ListBranches Tests
// -----------------------------------------------------------------------------

func TestListBranches(t *testing.T) {
	t.Run("list_branches", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create additional branches
		runGitCmd(t, dir, "branch", "feature-1")
		runGitCmd(t, dir, "branch", "feature-2")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		branches, err := repo.ListBranches()
		if err != nil {
			t.Fatalf("ListBranches() error = %v", err)
		}

		// Should have at least 3 branches (main/master + 2 features)
		if len(branches) < 3 {
			t.Errorf("ListBranches() = %d branches, want at least 3", len(branches))
		}

		// Check feature branches exist
		foundFeature1 := false
		foundFeature2 := false
		for _, b := range branches {
			if b == "feature-1" {
				foundFeature1 = true
			}
			if b == "feature-2" {
				foundFeature2 = true
			}
		}

		if !foundFeature1 {
			t.Error("ListBranches() missing feature-1")
		}
		if !foundFeature2 {
			t.Error("ListBranches() missing feature-2")
		}
	})
}

// -----------------------------------------------------------------------------
// HasUpstream Tests
// -----------------------------------------------------------------------------

func TestHasUpstream(t *testing.T) {
	t.Run("no_upstream", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		repo, err := Open(dir)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		if repo.HasUpstream() {
			t.Error("HasUpstream() = true, want false for repo without remote")
		}
	})
}
