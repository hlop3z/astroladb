// Package chain implements migration checksum chains for tamper detection.
// Each migration's checksum is computed as sha256(content + prev_checksum),
// creating a hash chain similar to blockchain. Modifying any migration
// invalidates all subsequent checksums.
package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// GenesisChecksum is the initial checksum for the first migration in the chain.
const GenesisChecksum = "genesis"

// Link represents a single migration in the chain.
type Link struct {
	Revision     string // e.g., "001"
	Name         string // e.g., "initial_schema"
	Filename     string // e.g., "001_initial_schema.js"
	Checksum     string // sha256(content + prev_checksum)
	PrevChecksum string // previous link's checksum
	Content      []byte // raw file content
}

// Chain represents the computed migration chain from files.
type Chain struct {
	Links []Link
}

// AppliedMigration represents a migration recorded in the database.
type AppliedMigration struct {
	Revision  string
	Checksum  string
	AppliedAt string // ISO timestamp
}

// VerificationResult contains the result of verifying the chain.
type VerificationResult struct {
	Valid           bool
	Errors          []ChainError
	Warnings        []string
	PendingLinks    []Link             // Links not yet applied
	AppliedLinks    []Link             // Links that match database
	MismatchedLinks []MismatchedLink   // Links with checksum mismatch
	MissingFiles    []AppliedMigration // Applied but file missing
}

// MismatchedLink represents a migration where file and DB checksums differ.
type MismatchedLink struct {
	Link             Link
	ExpectedChecksum string // from database
	ActualChecksum   string // computed from file
}

// ChainError represents an error in the chain.
type ChainError struct {
	Type     ErrorType
	Revision string
	Message  string
	Details  string
}

// ErrorType categorizes chain errors.
type ErrorType int

const (
	ErrorTampered ErrorType = iota // File modified after being applied
	ErrorFork                      // Same revision, different content
	ErrorMissing                   // Applied migration file is missing
	ErrorGap                       // Gap in revision numbering
)

// ComputeFromDir reads migration files and computes the chain.
func ComputeFromDir(migrationsDir string) (*Chain, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Chain{}, nil
		}
		return nil, alerr.Wrap(alerr.ErrMigrationNotFound, err, "failed to read migrations directory").
			With("path", migrationsDir)
	}

	// Collect migration files
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".js") && len(name) > 4 {
			// Check it starts with a number
			if name[0] >= '0' && name[0] <= '9' {
				files = append(files, name)
			}
		}
	}

	// Sort by filename (which sorts by revision due to numbering)
	sort.Strings(files)

	// Build the chain
	chain := &Chain{
		Links: make([]Link, 0, len(files)),
	}

	prevChecksum := GenesisChecksum
	for _, filename := range files {
		path := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, alerr.Wrap(alerr.ErrMigrationNotFound, err, "failed to read migration").
				With("file", filename)
		}

		// Parse revision and name from filename
		revision, name := parseFilename(filename)

		// Compute checksum: sha256(content + prev_checksum)
		checksum := computeChecksum(content, prevChecksum)

		chain.Links = append(chain.Links, Link{
			Revision:     revision,
			Name:         name,
			Filename:     filename,
			Checksum:     checksum,
			PrevChecksum: prevChecksum,
			Content:      content,
		})

		prevChecksum = checksum
	}

	return chain, nil
}

// ComputeFromFiles computes the chain from a map of filename -> content.
// Useful for computing chain at a specific git commit.
func ComputeFromFiles(files map[string][]byte) *Chain {
	// Get sorted filenames
	var filenames []string
	for name := range files {
		if strings.HasSuffix(name, ".js") && len(name) > 4 {
			if name[0] >= '0' && name[0] <= '9' {
				filenames = append(filenames, name)
			}
		}
	}
	sort.Strings(filenames)

	// Build the chain
	chain := &Chain{
		Links: make([]Link, 0, len(filenames)),
	}

	prevChecksum := GenesisChecksum
	for _, filename := range filenames {
		content := files[filename]
		revision, name := parseFilename(filename)
		checksum := computeChecksum(content, prevChecksum)

		chain.Links = append(chain.Links, Link{
			Revision:     revision,
			Name:         name,
			Filename:     filename,
			Checksum:     checksum,
			PrevChecksum: prevChecksum,
			Content:      content,
		})

		prevChecksum = checksum
	}

	return chain
}

// Verify compares the computed chain against applied migrations.
func (c *Chain) Verify(applied []AppliedMigration) *VerificationResult {
	result := &VerificationResult{
		Valid: true,
	}

	// Build lookup maps
	appliedByRevision := make(map[string]AppliedMigration)
	for _, a := range applied {
		appliedByRevision[a.Revision] = a
	}

	linkByRevision := make(map[string]Link)
	for _, l := range c.Links {
		linkByRevision[l.Revision] = l
	}

	// Check each link against applied migrations
	for _, link := range c.Links {
		appliedMig, wasApplied := appliedByRevision[link.Revision]
		if !wasApplied {
			// Not yet applied - pending
			result.PendingLinks = append(result.PendingLinks, link)
			continue
		}

		// Was applied - verify checksum matches
		if appliedMig.Checksum != link.Checksum {
			result.Valid = false
			result.MismatchedLinks = append(result.MismatchedLinks, MismatchedLink{
				Link:             link,
				ExpectedChecksum: appliedMig.Checksum,
				ActualChecksum:   link.Checksum,
			})
			result.Errors = append(result.Errors, ChainError{
				Type:     ErrorTampered,
				Revision: link.Revision,
				Message:  fmt.Sprintf("Migration %s was modified after being applied", link.Filename),
				Details:  fmt.Sprintf("Expected checksum: %s\nActual checksum: %s", appliedMig.Checksum, link.Checksum),
			})
		} else {
			result.AppliedLinks = append(result.AppliedLinks, link)
		}
	}

	// Check for applied migrations without corresponding files
	for _, appliedMig := range applied {
		if _, hasFile := linkByRevision[appliedMig.Revision]; !hasFile {
			result.Valid = false
			result.MissingFiles = append(result.MissingFiles, appliedMig)
			result.Errors = append(result.Errors, ChainError{
				Type:     ErrorMissing,
				Revision: appliedMig.Revision,
				Message:  fmt.Sprintf("Applied migration %s is missing from filesystem", appliedMig.Revision),
				Details:  fmt.Sprintf("Checksum in database: %s", appliedMig.Checksum),
			})
		}
	}

	return result
}

// GetLink returns the link for a given revision, or nil if not found.
func (c *Chain) GetLink(revision string) *Link {
	for i := range c.Links {
		if c.Links[i].Revision == revision {
			return &c.Links[i]
		}
	}
	return nil
}

// GetPendingAfter returns all links after the given revision.
func (c *Chain) GetPendingAfter(lastAppliedRevision string) []Link {
	if lastAppliedRevision == "" {
		return c.Links
	}

	for i, link := range c.Links {
		if link.Revision == lastAppliedRevision {
			if i+1 < len(c.Links) {
				return c.Links[i+1:]
			}
			return nil
		}
	}

	// Revision not found, return all
	return c.Links
}

// LastChecksum returns the checksum of the last link, or genesis if empty.
func (c *Chain) LastChecksum() string {
	if len(c.Links) == 0 {
		return GenesisChecksum
	}
	return c.Links[len(c.Links)-1].Checksum
}

// computeChecksum computes sha256(content + prevChecksum).
func computeChecksum(content []byte, prevChecksum string) string {
	h := sha256.New()
	h.Write(content)
	h.Write([]byte(prevChecksum))
	return hex.EncodeToString(h.Sum(nil))
}

// parseFilename extracts revision and name from a migration filename.
// e.g., "001_initial_schema.js" -> ("001", "initial_schema")
func parseFilename(filename string) (revision, name string) {
	// Remove .js extension
	base := strings.TrimSuffix(filename, ".js")

	// Split on first underscore
	idx := strings.Index(base, "_")
	if idx == -1 {
		return base, ""
	}

	return base[:idx], base[idx+1:]
}

// FormatFilename creates a migration filename from revision and name.
func FormatFilename(revision, name string) string {
	return fmt.Sprintf("%s_%s.js", revision, name)
}
