package chain

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/ui"
)

// FormatVerificationResult formats the verification result for display.
func FormatVerificationResult(result *VerificationResult) string {
	var b strings.Builder

	// Build summary content
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("  Applied:  %s\n", ui.FormatCount(len(result.AppliedLinks), "migration", "migrations")))
	summary.WriteString(fmt.Sprintf("  Pending:  %s\n", ui.FormatCount(len(result.PendingLinks), "migration", "migrations")))

	if len(result.MismatchedLinks) > 0 {
		summary.WriteString(fmt.Sprintf("  Tampered: %s\n", ui.Error(ui.FormatCount(len(result.MismatchedLinks), "migration", "migrations"))))
	}
	if len(result.MissingFiles) > 0 {
		summary.WriteString(fmt.Sprintf("  Missing:  %s\n", ui.Error(ui.FormatCount(len(result.MissingFiles), "file", "files"))))
	}

	// Render main panel based on validity
	if result.Valid {
		b.WriteString(ui.RenderSuccessPanel("Chain Integrity: VALID", summary.String()))
	} else {
		b.WriteString(ui.RenderErrorPanel("Chain Integrity: BROKEN", summary.String()))
	}

	// Detail errors
	if len(result.Errors) > 0 {
		b.WriteString("\n")
		list := ui.NewList()
		for _, err := range result.Errors {
			errMsg := fmt.Sprintf("[%s] %s", ui.Bold(err.Revision), err.Message)
			if err.Details != "" {
				errMsg += "\n" + ui.Indent(err.Details, 2)
			}
			list.AddError(errMsg)
		}
		b.WriteString(list.String())
	}

	// Pending migrations as a table
	if len(result.PendingLinks) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.Primary("Pending migrations:") + "\n")

		table := ui.NewStyledTable("REVISION", "NAME", "STATUS")
		for _, link := range result.PendingLinks {
			table.AddRow(
				ui.Dim(link.Revision),
				link.Name,
				ui.RenderPendingBadge(),
			)
		}
		b.WriteString(table.String())
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
		return ui.RenderInfoPanel("Migration Chain", "Empty (no migrations)")
	}

	content := fmt.Sprintf("  %s\n", ui.FormatCount(len(chain.Links), "migration", "migrations"))
	content += fmt.Sprintf("  First: %s\n", ui.Primary(chain.Links[0].Filename))
	content += fmt.Sprintf("  Last:  %s\n", ui.Primary(chain.Links[len(chain.Links)-1].Filename))
	content += fmt.Sprintf("  Head checksum: %s", ui.Dim(chain.LastChecksum()[:16]+"..."))

	return ui.RenderInfoPanel("Migration Chain", content)
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
