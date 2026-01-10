package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// AutomationResult holds the result of an automated git operation.
type AutomationResult struct {
	Action     string   // What action was taken
	Files      []string // Files affected
	CommitHash string   // Commit hash if committed
	Pushed     bool     // Whether changes were pushed
	Message    string   // Human-readable message
	ShouldPush bool     // Whether push is recommended
}

// CommitMigration commits a newly generated migration file.
// Returns the commit result or nil if not in a git repo.
func CommitMigration(migrationsDir, migrationFile, migrationName string) (*AutomationResult, error) {
	repo, err := Open(migrationsDir)
	if err != nil {
		// Not a git repo, silently skip
		return nil, nil
	}

	// Get the relative path for better messages
	relPath, _ := filepath.Rel(repo.RootDir(), migrationFile)
	if relPath == "" {
		relPath = migrationFile
	}

	// Stage and commit
	message := fmt.Sprintf("Add migration: %s", migrationName)
	if err := repo.CommitFiles(message, migrationFile); err != nil {
		return nil, alerr.Wrap(alerr.EGitOperation, err, "failed to commit migration").
			With("file", migrationFile)
	}

	// Get commit hash
	hash, _ := repo.runGit("rev-parse", "--short", "HEAD")
	hash = strings.TrimSpace(hash)

	// Check if push is possible
	canPush, _ := repo.CanPush()

	return &AutomationResult{
		Action:     "committed",
		Files:      []string{relPath},
		CommitHash: hash,
		Message:    fmt.Sprintf("Committed migration %s [%s]", migrationName, hash),
		ShouldPush: canPush,
	}, nil
}

// CommitSchemaChange commits schema file changes.
func CommitSchemaChange(schemasDir string, changedFiles []string, description string) (*AutomationResult, error) {
	if len(changedFiles) == 0 {
		return nil, nil
	}

	repo, err := Open(schemasDir)
	if err != nil {
		return nil, nil
	}

	// Build commit message
	message := description
	if message == "" {
		message = "Update schema files"
	}

	if err := repo.CommitFiles(message, changedFiles...); err != nil {
		return nil, alerr.Wrap(alerr.EGitOperation, err, "failed to commit schema changes")
	}

	hash, _ := repo.runGit("rev-parse", "--short", "HEAD")
	hash = strings.TrimSpace(hash)

	canPush, _ := repo.CanPush()

	return &AutomationResult{
		Action:     "committed",
		Files:      changedFiles,
		CommitHash: hash,
		Message:    fmt.Sprintf("Committed %d schema file(s) [%s]", len(changedFiles), hash),
		ShouldPush: canPush,
	}, nil
}

// PushIfReady pushes changes if there's a remote and unpushed commits.
func PushIfReady(dir string) (*AutomationResult, error) {
	repo, err := Open(dir)
	if err != nil {
		return nil, nil
	}

	info, err := repo.Info()
	if err != nil {
		return nil, err
	}

	if !info.HasRemote {
		return &AutomationResult{
			Action:  "skipped",
			Message: "No remote configured, skipping push",
		}, nil
	}

	canPush, err := repo.CanPush()
	if err != nil {
		return nil, err
	}

	if !canPush {
		return &AutomationResult{
			Action:  "skipped",
			Message: "No unpushed commits",
		}, nil
	}

	// Push
	if repo.HasUpstream() {
		if err := repo.Push(); err != nil {
			return nil, alerr.Wrap(alerr.EGitOperation, err, "failed to push")
		}
	} else {
		if err := repo.PushWithUpstream("origin", info.CurrentBranch); err != nil {
			return nil, alerr.Wrap(alerr.EGitOperation, err, "failed to push with upstream").
				With("branch", info.CurrentBranch)
		}
	}

	return &AutomationResult{
		Action:  "pushed",
		Pushed:  true,
		Message: fmt.Sprintf("Pushed to origin/%s", info.CurrentBranch),
	}, nil
}

// VerifyMigrationsCommitted checks that all migration files are committed.
// Returns a list of uncommitted files.
func VerifyMigrationsCommitted(migrationsDir string) ([]FileStatus, error) {
	repo, err := Open(migrationsDir)
	if err != nil {
		// Not a git repo, nothing to verify
		return nil, nil
	}

	uncommitted, err := repo.UncommittedMigrations(migrationsDir)
	if err != nil {
		return nil, err
	}

	return uncommitted, nil
}

// VerifyCleanState checks if the working directory is clean (no uncommitted changes).
func VerifyCleanState(dir string) (bool, error) {
	repo, err := Open(dir)
	if err != nil {
		return true, nil // Not a git repo, consider it clean
	}

	info, err := repo.Info()
	if err != nil {
		return false, err
	}

	return !info.IsDirty, nil
}

// GetBranchInfo returns information about the current branch and remote status.
func GetBranchInfo(dir string) (*RepoInfo, error) {
	repo, err := Open(dir)
	if err != nil {
		return nil, err
	}

	return repo.Info()
}

// PreMigrateCheck performs checks before running migrations.
// This ensures git state is good before applying migrations.
type PreMigrateCheck struct {
	InGitRepo                bool
	HasUncommittedMigrations bool
	UncommittedFiles         []FileStatus
	Warnings                 []string
	Errors                   []string
}

// CheckBeforeMigrate runs pre-migration checks.
func CheckBeforeMigrate(migrationsDir string) (*PreMigrateCheck, error) {
	result := &PreMigrateCheck{}

	repo, err := Open(migrationsDir)
	if err != nil {
		result.InGitRepo = false
		result.Warnings = append(result.Warnings, "Not in a git repository. Migration history won't be tracked in git.")
		return result, nil
	}
	result.InGitRepo = true

	// Check for uncommitted migrations
	uncommitted, err := repo.UncommittedMigrations(migrationsDir)
	if err != nil {
		return nil, err
	}

	if len(uncommitted) > 0 {
		result.HasUncommittedMigrations = true
		result.UncommittedFiles = uncommitted

		for _, f := range uncommitted {
			switch f.Status {
			case StatusUntracked:
				result.Warnings = append(result.Warnings, fmt.Sprintf("Untracked: %s", filepath.Base(f.Path)))
			case StatusModified:
				result.Errors = append(result.Errors, fmt.Sprintf("Modified: %s - this migration may have been applied elsewhere", filepath.Base(f.Path)))
			}
		}
	}

	return result, nil
}

// PostMigrateCheck checks state after migrations.
type PostMigrateCheck struct {
	ShouldCommit bool
	ShouldPush   bool
	Message      string
}

// CheckAfterMigrate runs post-migration checks.
func CheckAfterMigrate(migrationsDir string) (*PostMigrateCheck, error) {
	result := &PostMigrateCheck{}

	repo, err := Open(migrationsDir)
	if err != nil {
		return result, nil
	}

	// Check for uncommitted migrations (shouldn't happen after clean migrate)
	uncommitted, _ := repo.UncommittedMigrations(migrationsDir)
	if len(uncommitted) > 0 {
		result.ShouldCommit = true
		result.Message = fmt.Sprintf("%d migration file(s) should be committed", len(uncommitted))
	}

	// Check if we should push
	canPush, _ := repo.CanPush()
	if canPush {
		result.ShouldPush = true
		if result.Message != "" {
			result.Message += "; "
		}
		result.Message += "Unpushed commits found"
	}

	return result, nil
}
