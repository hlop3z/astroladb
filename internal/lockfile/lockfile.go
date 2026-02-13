// Package lockfile provides read/write/verify/repair for alab.lock files.
// The lock file tracks the integrity of migration files using SHA-256 checksums.
package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Entry represents a single file entry in the lock file.
type Entry struct {
	Filename string
	Checksum string
}

// LockFile represents the parsed contents of an alab.lock file.
type LockFile struct {
	Aggregate string  // SHA-256 of all individual checksums combined
	Entries   []Entry // Individual file checksums
}

// Read reads and parses a lock file from the given path.
// Returns nil if the file does not exist.
func Read(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 1 {
		return nil, fmt.Errorf("lock file is empty")
	}

	lf := &LockFile{
		Aggregate: strings.TrimSpace(lines[0]),
	}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		lf.Entries = append(lf.Entries, Entry{
			Filename: strings.TrimSpace(parts[1]),
			Checksum: strings.TrimSpace(parts[0]),
		})
	}

	return lf, nil
}

// Write generates and writes a lock file for the given migrations directory.
// It computes SHA-256 checksums for each .js file and writes the lock file.
func Write(migrationsDir, lockPath string) error {
	entries, err := computeEntries(migrationsDir)
	if err != nil {
		return err
	}

	aggregate := computeAggregate(entries)

	var sb strings.Builder
	sb.WriteString(aggregate + "\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%s %s\n", e.Checksum, e.Filename))
	}

	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock file directory: %w", err)
	}

	return os.WriteFile(lockPath, []byte(sb.String()), 0644)
}

// Verify checks whether the lock file matches the current state of migration files.
// Returns nil if everything matches, or a descriptive error if not.
func Verify(migrationsDir, lockPath string) error {
	lf, err := Read(lockPath)
	if err != nil {
		return err
	}
	if lf == nil {
		return fmt.Errorf("lock file not found: %s", lockPath)
	}

	entries, err := computeEntries(migrationsDir)
	if err != nil {
		return err
	}

	// Check aggregate
	aggregate := computeAggregate(entries)
	if aggregate != lf.Aggregate {
		return fmt.Errorf("lock file aggregate mismatch: expected %s, got %s", lf.Aggregate, aggregate)
	}

	// Build lookup from lock file
	lockMap := make(map[string]string)
	for _, e := range lf.Entries {
		lockMap[e.Filename] = e.Checksum
	}

	// Check each file
	for _, e := range entries {
		expected, ok := lockMap[e.Filename]
		if !ok {
			return fmt.Errorf("file %s not in lock file (new migration?)", e.Filename)
		}
		if expected != e.Checksum {
			return fmt.Errorf("checksum mismatch for %s: lock has %s, file has %s", e.Filename, expected, e.Checksum)
		}
	}

	// Check for removed files
	fileMap := make(map[string]bool)
	for _, e := range entries {
		fileMap[e.Filename] = true
	}
	for _, e := range lf.Entries {
		if !fileMap[e.Filename] {
			return fmt.Errorf("file %s in lock file but not on disk (deleted migration?)", e.Filename)
		}
	}

	return nil
}

// Repair regenerates the lock file from the current migration files.
func Repair(migrationsDir, lockPath string) error {
	return Write(migrationsDir, lockPath)
}

// VerificationResult holds detailed results of lock file verification.
type VerificationResult struct {
	Valid          bool     // Overall validity
	LockFileExists bool     // Whether lock file exists
	AggregateMatch bool     // Whether aggregate checksum matches
	NewFiles       []string // Files on disk but not in lock
	RemovedFiles   []string // Files in lock but not on disk
	ModifiedFiles  []string // Files with checksum mismatches
	VerifiedFiles  []string // Files that passed verification
}

// VerifyDetailed checks the lock file and returns detailed verification results.
// Unlike Verify which returns an error, this returns structured results for UI display.
func VerifyDetailed(migrationsDir, lockPath string) (*VerificationResult, error) {
	result := &VerificationResult{
		Valid:          true,
		LockFileExists: true,
		AggregateMatch: true,
	}

	lf, err := Read(lockPath)
	if err != nil {
		return nil, err
	}
	if lf == nil {
		result.LockFileExists = false
		result.Valid = false
		return result, nil
	}

	entries, err := computeEntries(migrationsDir)
	if err != nil {
		return nil, err
	}

	// Check aggregate
	aggregate := computeAggregate(entries)
	if aggregate != lf.Aggregate {
		result.AggregateMatch = false
		result.Valid = false
	}

	// Build lookup from lock file
	lockMap := make(map[string]string)
	for _, e := range lf.Entries {
		lockMap[e.Filename] = e.Checksum
	}

	// Check each file on disk
	fileMap := make(map[string]bool)
	for _, e := range entries {
		fileMap[e.Filename] = true

		expected, ok := lockMap[e.Filename]
		if !ok {
			result.NewFiles = append(result.NewFiles, e.Filename)
			result.Valid = false
		} else if expected != e.Checksum {
			result.ModifiedFiles = append(result.ModifiedFiles, e.Filename)
			result.Valid = false
		} else {
			result.VerifiedFiles = append(result.VerifiedFiles, e.Filename)
		}
	}

	// Check for removed files
	for _, e := range lf.Entries {
		if !fileMap[e.Filename] {
			result.RemovedFiles = append(result.RemovedFiles, e.Filename)
			result.Valid = false
		}
	}

	return result, nil
}

// DefaultPath returns the default lock file path for a project.
// The lock file is placed next to alab.yaml, following the convention
// of package managers like pnpm, npm, cargo, and poetry.
func DefaultPath() string {
	return "alab.lock"
}

// computeEntries reads all .js files in the migrations dir and computes their checksums.
func computeEntries(migrationsDir string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var entries []Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".js") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(migrationsDir, de.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", de.Name(), err)
		}
		sum := sha256.Sum256(data)
		entries = append(entries, Entry{
			Filename: de.Name(),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}

	// Sort by filename for deterministic output
	slices.SortFunc(entries, func(a, b Entry) int {
		return strings.Compare(a.Filename, b.Filename)
	})

	return entries, nil
}

// computeAggregate computes the aggregate SHA-256 from all individual checksums.
func computeAggregate(entries []Entry) string {
	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.Checksum))
	}
	return hex.EncodeToString(h.Sum(nil))
}
