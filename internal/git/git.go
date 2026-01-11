// Package git provides git operations for alab migration tracking.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// Status represents the git status of a file.
type Status int

const (
	StatusUnknown Status = iota
	StatusUntracked
	StatusModified
	StatusStaged
	StatusCommitted
	StatusDeleted
)

// FileStatus holds the status of a specific file.
type FileStatus struct {
	Path   string
	Status Status
}

// RepoInfo holds information about a git repository.
type RepoInfo struct {
	IsRepo        bool
	RootDir       string
	CurrentBranch string
	RemoteURL     string
	HasRemote     bool
	IsDirty       bool
}

// Repo provides git operations for a repository.
type Repo struct {
	rootDir string
}

// Open opens a git repository at the given path.
// Returns an error if the path is not inside a git repository.
func Open(path string) (*Repo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, alerr.Newf(alerr.EInternalError, "failed to resolve path: %v", err)
	}

	// Find the git root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = absPath
	out, err := cmd.Output()
	if err != nil {
		return nil, alerr.Newf(alerr.ENotGitRepo, "not a git repository: %s", absPath)
	}

	rootDir := strings.TrimSpace(string(out))
	return &Repo{rootDir: rootDir}, nil
}

// RootDir returns the root directory of the repository.
func (r *Repo) RootDir() string {
	return r.rootDir
}

// Info returns information about the repository.
func (r *Repo) Info() (*RepoInfo, error) {
	info := &RepoInfo{
		IsRepo:  true,
		RootDir: r.rootDir,
	}

	// Get current branch
	branch, err := r.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.CurrentBranch = strings.TrimSpace(branch)
	}

	// Get remote URL
	remote, err := r.runGit("remote", "get-url", "origin")
	if err == nil {
		info.RemoteURL = strings.TrimSpace(remote)
		info.HasRemote = true
	}

	// Check if dirty
	status, err := r.runGit("status", "--porcelain")
	if err == nil {
		info.IsDirty = len(strings.TrimSpace(status)) > 0
	}

	return info, nil
}

// IsTracked returns true if the file is tracked by git.
func (r *Repo) IsTracked(path string) (bool, error) {
	relPath, err := r.relativePath(path)
	if err != nil {
		return false, err
	}

	_, err = r.runGit("ls-files", "--error-unmatch", relPath)
	return err == nil, nil
}

// FileStatus returns the status of a file.
func (r *Repo) FileStatus(path string) (Status, error) {
	relPath, err := r.relativePath(path)
	if err != nil {
		return StatusUnknown, err
	}

	// Check if tracked
	tracked, _ := r.IsTracked(path)
	if !tracked {
		return StatusUntracked, nil
	}

	// Check for modifications
	status, err := r.runGit("status", "--porcelain", relPath)
	if err != nil {
		return StatusUnknown, err
	}

	status = strings.TrimSpace(status)
	if status == "" {
		return StatusCommitted, nil
	}

	// Parse status codes
	if len(status) >= 2 {
		switch status[0] {
		case 'M', 'A':
			return StatusStaged, nil
		case 'D':
			return StatusDeleted, nil
		case ' ':
			if status[1] == 'M' {
				return StatusModified, nil
			}
			if status[1] == 'D' {
				return StatusDeleted, nil
			}
		case '?':
			return StatusUntracked, nil
		}
	}

	return StatusModified, nil
}

// UncommittedMigrations returns migration files that are not committed.
func (r *Repo) UncommittedMigrations(migrationsDir string) ([]FileStatus, error) {
	relDir, err := r.relativePath(migrationsDir)
	if err != nil {
		// If migrations dir doesn't exist yet, no uncommitted files
		return nil, nil
	}

	// Get all files in migrations directory with their status
	// Use -uall to show individual files in untracked directories
	status, err := r.runGit("status", "--porcelain", "-uall", relDir)
	if err != nil {
		return nil, err
	}

	var files []FileStatus
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse status line: "XY path" or "XY path -> newpath"
		if len(line) < 4 {
			continue
		}

		statusCode := line[0:2]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files
		if idx := strings.Index(path, " -> "); idx != -1 {
			path = path[idx+4:]
		}

		var s Status
		switch {
		case statusCode[0] == '?' || statusCode[1] == '?':
			s = StatusUntracked
		case statusCode[0] == 'M' || statusCode[1] == 'M':
			s = StatusModified
		case statusCode[0] == 'A':
			s = StatusStaged
		case statusCode[0] == 'D' || statusCode[1] == 'D':
			s = StatusDeleted
		default:
			s = StatusModified
		}

		// Only include .js migration files
		if !strings.HasSuffix(path, ".js") {
			continue
		}

		files = append(files, FileStatus{
			Path:   filepath.Join(r.rootDir, path),
			Status: s,
		})
	}

	return files, nil
}

// Add stages files for commit.
func (r *Repo) Add(paths ...string) error {
	if len(paths) == 0 {
		return nil
	}

	args := append([]string{"add"}, paths...)
	_, err := r.runGit(args...)
	return err
}

// Commit creates a commit with the given message.
func (r *Repo) Commit(message string) error {
	_, err := r.runGit("commit", "-m", message)
	return err
}

// CommitFiles stages and commits specific files.
func (r *Repo) CommitFiles(message string, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}

	if err := r.Add(paths...); err != nil {
		return err
	}

	return r.Commit(message)
}

// Push pushes to the remote.
func (r *Repo) Push() error {
	_, err := r.runGit("push")
	return err
}

// PushWithUpstream pushes and sets upstream.
func (r *Repo) PushWithUpstream(remote, branch string) error {
	_, err := r.runGit("push", "-u", remote, branch)
	return err
}

// HasUpstream returns true if the current branch has an upstream.
func (r *Repo) HasUpstream() bool {
	_, err := r.runGit("rev-parse", "--abbrev-ref", "@{u}")
	return err == nil
}

// CanPush returns true if there are commits to push.
func (r *Repo) CanPush() (bool, error) {
	// Check if remote exists
	info, err := r.Info()
	if err != nil {
		return false, err
	}
	if !info.HasRemote {
		return false, nil
	}

	// Check for unpushed commits
	out, err := r.runGit("log", "@{u}..HEAD", "--oneline")
	if err != nil {
		// No upstream, check if we have commits
		return true, nil
	}

	return len(strings.TrimSpace(out)) > 0, nil
}

// GetFileAtCommit returns the content of a file at a specific commit.
func (r *Repo) GetFileAtCommit(path, commit string) ([]byte, error) {
	relPath, err := r.relativePath(path)
	if err != nil {
		return nil, err
	}

	out, err := r.runGit("show", fmt.Sprintf("%s:%s", commit, relPath))
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}

// GetFilesAtCommit returns all migration files at a specific commit.
func (r *Repo) GetFilesAtCommit(migrationsDir, commit string) (map[string][]byte, error) {
	relDir, err := r.relativePath(migrationsDir)
	if err != nil {
		relDir = migrationsDir
	}

	// List files in directory at commit
	out, err := r.runGit("ls-tree", "--name-only", commit, relDir+"/")
	if err != nil {
		return nil, err
	}

	files := make(map[string][]byte)
	for _, path := range strings.Split(out, "\n") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		content, err := r.runGit("show", fmt.Sprintf("%s:%s", commit, path))
		if err != nil {
			continue
		}

		files[filepath.Base(path)] = []byte(content)
	}

	return files, nil
}

// ListBranches returns all local branches.
func (r *Repo) ListBranches() ([]string, error) {
	out, err := r.runGit("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, b := range strings.Split(out, "\n") {
		b = strings.TrimSpace(b)
		if b != "" {
			branches = append(branches, b)
		}
	}

	return branches, nil
}

// ListRemoteBranches returns all remote branches.
func (r *Repo) ListRemoteBranches() ([]string, error) {
	out, err := r.runGit("branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, b := range strings.Split(out, "\n") {
		b = strings.TrimSpace(b)
		if b != "" {
			branches = append(branches, b)
		}
	}

	return branches, nil
}

// Fetch fetches from remote.
func (r *Repo) Fetch() error {
	_, err := r.runGit("fetch")
	return err
}

// relativePath returns the path relative to the repository root.
func (r *Repo) relativePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(r.rootDir, absPath)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(relPath), nil
}

// runGit runs a git command and returns the output.
func (r *Repo) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.rootDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", alerr.New(alerr.EGitOperation, stderr.String()).
			With("command", "git "+strings.Join(args, " "))
	}

	return stdout.String(), nil
}
