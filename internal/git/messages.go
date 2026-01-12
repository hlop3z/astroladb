package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hlop3z/astroladb/internal/ui"
)

// Message templates for git operations.
// These provide clear, actionable messages for users.

// FormatCommitSuccess formats a successful commit message.
func FormatCommitSuccess(files []string, hash, description string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Committed: %s\n", description))
	b.WriteString(fmt.Sprintf("  commit: %s\n", hash))
	if len(files) > 0 {
		b.WriteString("  files:\n")
		for _, f := range files {
			b.WriteString(fmt.Sprintf("    + %s\n", filepath.Base(f)))
		}
	}
	return b.String()
}

// FormatPushSuccess formats a successful push message.
func FormatPushSuccess(branch string) string {
	return fmt.Sprintf("Pushed to origin/%s", branch)
}

// FormatUncommittedWarning formats a warning about uncommitted files.
func FormatUncommittedWarning(files []FileStatus) string {
	var b strings.Builder
	b.WriteString("Warning: Uncommitted migration files detected\n")
	b.WriteString("\n")

	for _, f := range files {
		var status string
		switch f.Status {
		case StatusUntracked:
			status = "untracked"
		case StatusModified:
			status = "modified"
		case StatusStaged:
			status = "staged"
		case StatusDeleted:
			status = "deleted"
		default:
			status = "changed"
		}
		b.WriteString(fmt.Sprintf("  %s: %s\n", status, filepath.Base(f.Path)))
	}

	b.WriteString("\n")
	b.WriteString("Migration files should be committed before deploying.\n")
	b.WriteString("Run: git add migrations/ && git commit -m \"Add migrations\"\n")

	return b.String()
}

// FormatModifiedMigrationError formats an error about modified migrations.
func FormatModifiedMigrationError(file string) string {
	return fmt.Sprintf(`Error: Migration file was modified after being applied

  File: %s

This migration has already been applied to one or more databases.
Modifying it can cause checksum mismatches and chain integrity failures.

To fix:
  1. Restore the original: git checkout HEAD -- %s
  2. Create a new migration for your changes: alab new <name>
`, filepath.Base(file), file)
}

// FormatNotGitRepoWarning formats a warning when not in a git repo.
func FormatNotGitRepoWarning() string {
	content := `Migration tracking works best with git:
  - Migration checksums form a chain (tamper-proof)
  - History is preserved in commits
  - Team synchronization via push/pull

To initialize:
  ` + ui.Dim("git init") + `
  ` + ui.Dim("git add .") + `
  ` + ui.Dim(`git commit -m "Initial commit"`)

	return ui.RenderWarningPanel("Not in a git repository", content)
}

// FormatPreMigrateWarnings formats all pre-migration warnings.
func FormatPreMigrateWarnings(check *PreMigrateCheck) string {
	if len(check.Warnings) == 0 && len(check.Errors) == 0 {
		return ""
	}

	var b strings.Builder

	if len(check.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, e := range check.Errors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
		b.WriteString("\n")
	}

	if len(check.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, w := range check.Warnings {
			b.WriteString(fmt.Sprintf("  - %s\n", w))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// FormatChainIntegrityError formats a chain integrity error.
func FormatChainIntegrityError(revision, expectedChecksum, actualChecksum string) string {
	return fmt.Sprintf(`Error: Chain integrity broken

  Migration: %s
  Expected checksum: %s
  Computed checksum: %s

A migration file was modified after being applied.
All subsequent migrations have invalid checksums.

To fix:
  git checkout HEAD -- migrations/%s
`, revision, expectedChecksum, actualChecksum, revision)
}

// FormatForkDetectionError formats a fork detection error.
func FormatForkDetectionError(revision, localChecksum, appliedChecksum string) string {
	return fmt.Sprintf(`Error: Migration conflict detected

  Revision: %s
  Your checksum:     %s
  Applied checksum:  %s

Same revision number but different content.
Another migration was applied with this number.

To fix:
  Rename your migration to the next available number:
  mv migrations/%s migrations/<next_number>_<name>.js
`, revision, localChecksum, appliedChecksum, revision)
}

// FormatMissingMigrationError formats a missing migration error.
func FormatMissingMigrationError(revision, checksum string) string {
	return fmt.Sprintf(`Error: Missing migration file

  Applied: %s (checksum: %s)
  File not found in migrations/

A migration that was applied is missing from the codebase.
This can happen if someone deleted a migration file.

To fix:
  1. Restore from git: git checkout HEAD -- migrations/%s
  2. Or restore from another environment
`, revision, checksum, revision)
}

// FormatDriftDetectionWarning formats a schema drift warning.
func FormatDriftDetectionWarning(tableName, difference string) string {
	return fmt.Sprintf(`Warning: Schema drift detected

  Table: %s
  %s

The database schema differs from what migrations define.
Someone may have modified the database directly.

To fix:
  Create a migration to reconcile: alab new fix_drift
`, tableName, difference)
}

// FormatGitStatusSummary formats a summary of git status for migrations.
func FormatGitStatusSummary(info *RepoInfo, uncommitted []FileStatus) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("  Branch:   %s\n", ui.Primary(info.CurrentBranch)))
	if info.HasRemote {
		content.WriteString(fmt.Sprintf("  Remote:   %s\n", ui.Dim(info.RemoteURL)))
	} else {
		content.WriteString(fmt.Sprintf("  Remote:   %s\n", ui.Dim("(none)")))
	}

	if info.IsDirty {
		content.WriteString(fmt.Sprintf("  Status:   %s\n", ui.Warning("uncommitted changes")))
	} else {
		content.WriteString(fmt.Sprintf("  Status:   %s", ui.Success("clean")))
	}

	if len(uncommitted) > 0 {
		content.WriteString("\n\n")
		list := ui.NewList()
		for _, f := range uncommitted {
			list.AddWarning(filepath.Base(f.Path))
		}
		content.WriteString("  Uncommitted migrations:\n")
		content.WriteString(ui.Indent(list.String(), 2))
	}

	return ui.RenderInfoPanel("Git Status", content.String())
}

// FormatAutoCommitPrompt formats a prompt for auto-commit.
func FormatAutoCommitPrompt(files []string) string {
	var b strings.Builder
	b.WriteString("The following files will be committed:\n")
	for _, f := range files {
		b.WriteString(fmt.Sprintf("  + %s\n", filepath.Base(f)))
	}
	b.WriteString("\nCommit these changes?")
	return b.String()
}

// FormatAutoPushPrompt formats a prompt for auto-push.
func FormatAutoPushPrompt(branch string) string {
	return fmt.Sprintf("Push to origin/%s?", branch)
}
