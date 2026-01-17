package git

import (
	"os"
	"path/filepath"
	"testing"
)

// -----------------------------------------------------------------------------
// CommitMigration Tests
// -----------------------------------------------------------------------------

func TestCommitMigration(t *testing.T) {
	t.Run("commits_migration_file", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create migrations directory and file
		migrationsDir := filepath.Join(dir, "migrations")
		migrationFile := filepath.Join(migrationsDir, "001_create_users.js")
		createFile(t, migrationFile, "// migration content")

		result, err := CommitMigration(migrationsDir, migrationFile, "create_users")
		if err != nil {
			t.Fatalf("CommitMigration() error = %v", err)
		}

		if result == nil {
			t.Fatal("CommitMigration() returned nil result")
		}

		if result.Action != "committed" {
			t.Errorf("CommitMigration().Action = %q, want %q", result.Action, "committed")
		}

		if result.CommitHash == "" {
			t.Error("CommitMigration().CommitHash is empty")
		}

		if len(result.Files) != 1 {
			t.Errorf("CommitMigration().Files = %d, want 1", len(result.Files))
		}
	})

	t.Run("non_git_repo_returns_nil", func(t *testing.T) {
		dir := t.TempDir()

		migrationsDir := filepath.Join(dir, "migrations")
		migrationFile := filepath.Join(migrationsDir, "001_create_users.js")
		createFile(t, migrationFile, "// migration content")

		result, err := CommitMigration(migrationsDir, migrationFile, "create_users")
		if err != nil {
			t.Fatalf("CommitMigration() error = %v", err)
		}

		if result != nil {
			t.Error("CommitMigration() expected nil result for non-git repo")
		}
	})
}

// -----------------------------------------------------------------------------
// CommitSchemaChange Tests
// -----------------------------------------------------------------------------

func TestCommitSchemaChange(t *testing.T) {
	t.Run("commits_schema_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create schema files
		schemasDir := filepath.Join(dir, "schemas")
		schemaFile := filepath.Join(schemasDir, "user.js")
		createFile(t, schemaFile, "// schema content")

		result, err := CommitSchemaChange(schemasDir, []string{schemaFile}, "Add user schema")
		if err != nil {
			t.Fatalf("CommitSchemaChange() error = %v", err)
		}

		if result == nil {
			t.Fatal("CommitSchemaChange() returned nil result")
		}

		if result.Action != "committed" {
			t.Errorf("CommitSchemaChange().Action = %q, want %q", result.Action, "committed")
		}
	})

	t.Run("empty_files_returns_nil", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		result, err := CommitSchemaChange(dir, []string{}, "No changes")
		if err != nil {
			t.Fatalf("CommitSchemaChange() error = %v", err)
		}

		if result != nil {
			t.Error("CommitSchemaChange() expected nil for empty files")
		}
	})

	t.Run("default_message", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create schema file
		schemaFile := filepath.Join(dir, "schema.js")
		createFile(t, schemaFile, "// schema")

		result, err := CommitSchemaChange(dir, []string{schemaFile}, "")
		if err != nil {
			t.Fatalf("CommitSchemaChange() error = %v", err)
		}

		if result == nil {
			t.Fatal("CommitSchemaChange() returned nil result")
		}
	})
}

// -----------------------------------------------------------------------------
// VerifyMigrationsCommitted Tests
// -----------------------------------------------------------------------------

func TestVerifyMigrationsCommitted(t *testing.T) {
	t.Run("all_committed", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create and commit migration
		migrationsDir := filepath.Join(dir, "migrations")
		migrationFile := filepath.Join(migrationsDir, "001_create_users.js")
		createFile(t, migrationFile, "// migration")
		runGitCmd(t, dir, "add", ".")
		runGitCmd(t, dir, "commit", "-m", "Add migration")

		uncommitted, err := VerifyMigrationsCommitted(migrationsDir)
		if err != nil {
			t.Fatalf("VerifyMigrationsCommitted() error = %v", err)
		}

		if len(uncommitted) != 0 {
			t.Errorf("VerifyMigrationsCommitted() = %d uncommitted, want 0", len(uncommitted))
		}
	})

	t.Run("uncommitted_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create uncommitted migrations
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration 1")
		createFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), "// migration 2")

		uncommitted, err := VerifyMigrationsCommitted(migrationsDir)
		if err != nil {
			t.Fatalf("VerifyMigrationsCommitted() error = %v", err)
		}

		if len(uncommitted) != 2 {
			t.Errorf("VerifyMigrationsCommitted() = %d uncommitted, want 2", len(uncommitted))
		}
	})

	t.Run("non_git_repo", func(t *testing.T) {
		dir := t.TempDir()
		migrationsDir := filepath.Join(dir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			t.Fatalf("failed to create migrations dir: %v", err)
		}

		uncommitted, err := VerifyMigrationsCommitted(migrationsDir)
		if err != nil {
			t.Fatalf("VerifyMigrationsCommitted() error = %v", err)
		}

		if uncommitted != nil {
			t.Error("VerifyMigrationsCommitted() expected nil for non-git repo")
		}
	})
}

// -----------------------------------------------------------------------------
// VerifyCleanState Tests
// -----------------------------------------------------------------------------

func TestVerifyCleanState(t *testing.T) {
	t.Run("clean_state", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		clean, err := VerifyCleanState(dir)
		if err != nil {
			t.Fatalf("VerifyCleanState() error = %v", err)
		}

		if !clean {
			t.Error("VerifyCleanState() = false, want true for clean repo")
		}
	})

	t.Run("dirty_state", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create uncommitted file
		createFile(t, filepath.Join(dir, "untracked.txt"), "content")

		clean, err := VerifyCleanState(dir)
		if err != nil {
			t.Fatalf("VerifyCleanState() error = %v", err)
		}

		if clean {
			t.Error("VerifyCleanState() = true, want false for dirty repo")
		}
	})

	t.Run("non_git_repo_is_clean", func(t *testing.T) {
		dir := t.TempDir()

		clean, err := VerifyCleanState(dir)
		if err != nil {
			t.Fatalf("VerifyCleanState() error = %v", err)
		}

		if !clean {
			t.Error("VerifyCleanState() = false, want true for non-git repo")
		}
	})
}

// -----------------------------------------------------------------------------
// CheckBeforeMigrate Tests
// -----------------------------------------------------------------------------

func TestCheckBeforeMigrate(t *testing.T) {
	t.Run("not_git_repo", func(t *testing.T) {
		dir := t.TempDir()
		migrationsDir := filepath.Join(dir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		check, err := CheckBeforeMigrate(migrationsDir)
		if err != nil {
			t.Fatalf("CheckBeforeMigrate() error = %v", err)
		}

		if check.InGitRepo {
			t.Error("CheckBeforeMigrate().InGitRepo = true, want false")
		}

		if len(check.Warnings) == 0 {
			t.Error("CheckBeforeMigrate().Warnings should have warning for non-git repo")
		}
	})

	t.Run("clean_git_repo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		migrationsDir := filepath.Join(dir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		check, err := CheckBeforeMigrate(migrationsDir)
		if err != nil {
			t.Fatalf("CheckBeforeMigrate() error = %v", err)
		}

		if !check.InGitRepo {
			t.Error("CheckBeforeMigrate().InGitRepo = false, want true")
		}

		if check.HasUncommittedMigrations {
			t.Error("CheckBeforeMigrate().HasUncommittedMigrations = true, want false")
		}
	})

	t.Run("uncommitted_migrations", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create uncommitted migration
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration")

		check, err := CheckBeforeMigrate(migrationsDir)
		if err != nil {
			t.Fatalf("CheckBeforeMigrate() error = %v", err)
		}

		if !check.HasUncommittedMigrations {
			t.Error("CheckBeforeMigrate().HasUncommittedMigrations = false, want true")
		}

		if len(check.UncommittedFiles) != 1 {
			t.Errorf("CheckBeforeMigrate().UncommittedFiles = %d, want 1", len(check.UncommittedFiles))
		}
	})
}

// -----------------------------------------------------------------------------
// CheckAfterMigrate Tests
// -----------------------------------------------------------------------------

func TestCheckAfterMigrate(t *testing.T) {
	t.Run("no_uncommitted_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		migrationsDir := filepath.Join(dir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		check, err := CheckAfterMigrate(migrationsDir)
		if err != nil {
			t.Fatalf("CheckAfterMigrate() error = %v", err)
		}

		if check.ShouldCommit {
			t.Error("CheckAfterMigrate().ShouldCommit = true, want false")
		}
	})

	t.Run("uncommitted_files", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create uncommitted migration
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration")

		check, err := CheckAfterMigrate(migrationsDir)
		if err != nil {
			t.Fatalf("CheckAfterMigrate() error = %v", err)
		}

		if !check.ShouldCommit {
			t.Error("CheckAfterMigrate().ShouldCommit = false, want true")
		}
	})
}

// -----------------------------------------------------------------------------
// CommitAppliedMigrations Tests
// -----------------------------------------------------------------------------

func TestCommitAppliedMigrations(t *testing.T) {
	t.Run("commits_up_migrations", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		// Create uncommitted migrations
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration 1")
		createFile(t, filepath.Join(migrationsDir, "002_create_posts.js"), "// migration 2")

		result, err := CommitAppliedMigrations(migrationsDir, "up")
		if err != nil {
			t.Fatalf("CommitAppliedMigrations() error = %v", err)
		}

		if result == nil {
			t.Fatal("CommitAppliedMigrations() returned nil result")
		}

		if result.Action != "committed" {
			t.Errorf("CommitAppliedMigrations().Action = %q, want %q", result.Action, "committed")
		}

		if len(result.Files) != 2 {
			t.Errorf("CommitAppliedMigrations().Files = %d, want 2", len(result.Files))
		}
	})

	t.Run("no_files_returns_nil", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		migrationsDir := filepath.Join(dir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		result, err := CommitAppliedMigrations(migrationsDir, "up")
		if err != nil {
			t.Fatalf("CommitAppliedMigrations() error = %v", err)
		}

		if result != nil {
			t.Error("CommitAppliedMigrations() expected nil when no files to commit")
		}
	})

	t.Run("non_git_repo_returns_nil", func(t *testing.T) {
		dir := t.TempDir()
		migrationsDir := filepath.Join(dir, "migrations")
		createFile(t, filepath.Join(migrationsDir, "001_create_users.js"), "// migration")

		result, err := CommitAppliedMigrations(migrationsDir, "up")
		if err != nil {
			t.Fatalf("CommitAppliedMigrations() error = %v", err)
		}

		if result != nil {
			t.Error("CommitAppliedMigrations() expected nil for non-git repo")
		}
	})
}

// -----------------------------------------------------------------------------
// GetBranchInfo Tests
// -----------------------------------------------------------------------------

func TestGetBranchInfo(t *testing.T) {
	t.Run("returns_branch_info", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		// Create initial commit
		createFile(t, filepath.Join(dir, "README.md"), "# Test")
		runGitCmd(t, dir, "add", "README.md")
		runGitCmd(t, dir, "commit", "-m", "Initial commit")

		info, err := GetBranchInfo(dir)
		if err != nil {
			t.Fatalf("GetBranchInfo() error = %v", err)
		}

		if info == nil {
			t.Fatal("GetBranchInfo() returned nil")
		}

		if !info.IsRepo {
			t.Error("GetBranchInfo().IsRepo = false, want true")
		}

		if info.CurrentBranch == "" {
			t.Error("GetBranchInfo().CurrentBranch is empty")
		}
	})

	t.Run("non_git_repo_error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := GetBranchInfo(dir)
		if err == nil {
			t.Error("GetBranchInfo() expected error for non-git repo")
		}
	})
}
