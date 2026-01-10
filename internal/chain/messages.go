package chain

import (
	"fmt"
	"strings"
)

// FormatVerificationResult formats the verification result for display.
func FormatVerificationResult(result *VerificationResult) string {
	var b strings.Builder

	if result.Valid {
		b.WriteString("Chain Integrity: VALID\n")
	} else {
		b.WriteString("Chain Integrity: BROKEN\n")
	}

	b.WriteString(strings.Repeat("-", 50) + "\n")

	// Summary counts
	b.WriteString(fmt.Sprintf("  Applied:  %d\n", len(result.AppliedLinks)))
	b.WriteString(fmt.Sprintf("  Pending:  %d\n", len(result.PendingLinks)))

	if len(result.MismatchedLinks) > 0 {
		b.WriteString(fmt.Sprintf("  Tampered: %d\n", len(result.MismatchedLinks)))
	}
	if len(result.MissingFiles) > 0 {
		b.WriteString(fmt.Sprintf("  Missing:  %d\n", len(result.MissingFiles)))
	}

	// Detail errors
	if len(result.Errors) > 0 {
		b.WriteString("\nErrors:\n")
		for _, err := range result.Errors {
			b.WriteString(fmt.Sprintf("\n  [%s] %s\n", err.Revision, err.Message))
			if err.Details != "" {
				for _, line := range strings.Split(err.Details, "\n") {
					b.WriteString(fmt.Sprintf("    %s\n", line))
				}
			}
		}
	}

	// Pending migrations
	if len(result.PendingLinks) > 0 {
		b.WriteString("\nPending migrations:\n")
		for _, link := range result.PendingLinks {
			b.WriteString(fmt.Sprintf("  - %s\n", link.Filename))
		}
	}

	return b.String()
}

// FormatTamperedError formats a tamper detection error.
func FormatTamperedError(link Link, expectedChecksum string) string {
	return fmt.Sprintf(`Error: Chain integrity broken

  Migration: %s
  Expected checksum: %s
  Computed checksum: %s

The migration file was modified after being applied.
All subsequent migrations have invalid checksums.

To fix:
  git checkout HEAD -- migrations/%s

If you intentionally modified this file, you must:
  1. Create a new migration with your changes
  2. Restore the original file
`, link.Filename, expectedChecksum, link.Checksum, link.Filename)
}

// FormatMissingFileError formats a missing file error.
func FormatMissingFileError(mig AppliedMigration) string {
	return fmt.Sprintf(`Error: Missing migration file

  Revision: %s
  Applied checksum: %s

This migration was applied but the file is missing.
The chain cannot be verified without this file.

To fix:
  1. Restore from git: git checkout HEAD -- migrations/%s_*.js
  2. Or restore from another environment
`, mig.Revision, mig.Checksum, mig.Revision)
}

// FormatChainStatus formats a brief chain status.
func FormatChainStatus(chain *Chain) string {
	if len(chain.Links) == 0 {
		return "Chain: empty (no migrations)"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Chain: %d migration(s)\n", len(chain.Links)))
	b.WriteString(fmt.Sprintf("  First: %s\n", chain.Links[0].Filename))
	b.WriteString(fmt.Sprintf("  Last:  %s\n", chain.Links[len(chain.Links)-1].Filename))
	b.WriteString(fmt.Sprintf("  Head checksum: %s...\n", chain.LastChecksum()[:16]))

	return b.String()
}

// FormatPendingMigrations formats a list of pending migrations.
func FormatPendingMigrations(pending []Link) string {
	if len(pending) == 0 {
		return "No pending migrations.\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Pending migrations: %d\n", len(pending)))
	for _, link := range pending {
		b.WriteString(fmt.Sprintf("  - %s (checksum: %s...)\n", link.Filename, link.Checksum[:12]))
	}

	return b.String()
}

// FormatAppliedMigration formats a single applied migration for display.
func FormatAppliedMigration(link Link) string {
	return fmt.Sprintf("  %s: %s... -> %s...\n",
		link.Revision,
		link.PrevChecksum[:8],
		link.Checksum[:8])
}

// FormatChainDiagram formats a visual diagram of the chain.
func FormatChainDiagram(chain *Chain, applied []AppliedMigration) string {
	if len(chain.Links) == 0 {
		return "  (empty chain)\n"
	}

	// Build set of applied revisions
	appliedSet := make(map[string]bool)
	for _, a := range applied {
		appliedSet[a.Revision] = true
	}

	var b strings.Builder
	b.WriteString("  genesis\n")
	b.WriteString("    |\n")

	for i, link := range chain.Links {
		status := " "
		if appliedSet[link.Revision] {
			status = "+"
		}

		b.WriteString(fmt.Sprintf("  [%s] %s %s\n", status, link.Revision, link.Name))

		if i < len(chain.Links)-1 {
			b.WriteString(fmt.Sprintf("    | %s...\n", link.Checksum[:12]))
		} else {
			b.WriteString(fmt.Sprintf("    v %s...\n", link.Checksum[:12]))
		}
	}

	b.WriteString("\n  Legend: [+] applied, [ ] pending\n")

	return b.String()
}
