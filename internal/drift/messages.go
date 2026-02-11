package drift

import (
	"fmt"
	"strings"
)

// FormatResult formats a drift detection result for CLI output.
func FormatResult(result *Result) string {
	if result == nil {
		return "No drift detection result available."
	}

	if !result.HasDrift {
		return FormatNoDrift(result)
	}

	return FormatDrift(result)
}

// FormatNoDrift formats a successful (no drift) result.
func FormatNoDrift(result *Result) string {
	var b strings.Builder

	b.WriteString("Schema check passed\n\n")
	b.WriteString(fmt.Sprintf("  Tables:       %d\n", len(result.ExpectedSchema.Tables)))
	b.WriteString(fmt.Sprintf("  Schema hash:  %s\n", truncateHash(result.ExpectedHash)))
	b.WriteString("\n  Database schema matches expected state.\n")

	return b.String()
}

// FormatDrift formats a drift detection result with differences.
func FormatDrift(result *Result) string {
	var b strings.Builder

	b.WriteString("Schema drift detected\n\n")

	// Hash comparison
	b.WriteString(fmt.Sprintf("  Expected hash: %s\n", truncateHash(result.ExpectedHash)))
	b.WriteString(fmt.Sprintf("  Actual hash:   %s\n", truncateHash(result.ActualHash)))
	b.WriteString("\n")

	comp := result.Comparison

	// Missing tables
	if len(comp.MissingTables) > 0 {
		b.WriteString("  Missing tables (in migrations but not in database):\n")
		for _, name := range comp.MissingTables {
			b.WriteString(fmt.Sprintf("    - %s\n", name))
		}
		b.WriteString("\n")
	}

	// Extra tables
	if len(comp.ExtraTables) > 0 {
		b.WriteString("  Extra tables (in database but not in migrations):\n")
		for _, name := range comp.ExtraTables {
			b.WriteString(fmt.Sprintf("    + %s\n", name))
		}
		b.WriteString("\n")
	}

	// Modified tables
	if len(comp.TableDiffs) > 0 {
		b.WriteString("  Modified tables:\n")
		for name, diff := range comp.TableDiffs {
			b.WriteString(fmt.Sprintf("\n    %s:\n", name))
			formatTableDiff(&b, diff, "      ")
		}
	}

	// Suggestions
	b.WriteString("\nFix:\n")
	b.WriteString("  Create a migration to reconcile the differences:\n")
	b.WriteString("    alab new reconcile_drift\n")

	return b.String()
}

// formatTableDiff formats differences for a single table.
func formatTableDiff(b *strings.Builder, diff *TableDiff, indent string) {
	// Columns
	if len(diff.MissingColumns) > 0 {
		fmt.Fprintf(b, "%sColumns missing from DB:\n", indent)
		for _, col := range diff.MissingColumns {
			fmt.Fprintf(b, "%s  - %s\n", indent, col)
		}
	}
	if len(diff.ExtraColumns) > 0 {
		fmt.Fprintf(b, "%sColumns only in DB:\n", indent)
		for _, col := range diff.ExtraColumns {
			fmt.Fprintf(b, "%s  + %s\n", indent, col)
		}
	}
	if len(diff.ModifiedColumns) > 0 {
		fmt.Fprintf(b, "%sColumns with different definitions:\n", indent)
		for _, col := range diff.ModifiedColumns {
			fmt.Fprintf(b, "%s  ~ %s\n", indent, col)
		}
	}

	// Indexes
	if len(diff.MissingIndexes) > 0 {
		fmt.Fprintf(b, "%sIndexes missing from DB:\n", indent)
		for _, idx := range diff.MissingIndexes {
			fmt.Fprintf(b, "%s  - %s\n", indent, idx)
		}
	}
	if len(diff.ExtraIndexes) > 0 {
		fmt.Fprintf(b, "%sIndexes only in DB:\n", indent)
		for _, idx := range diff.ExtraIndexes {
			fmt.Fprintf(b, "%s  + %s\n", indent, idx)
		}
	}
	if len(diff.ModifiedIndexes) > 0 {
		fmt.Fprintf(b, "%sIndexes with different definitions:\n", indent)
		for _, idx := range diff.ModifiedIndexes {
			fmt.Fprintf(b, "%s  ~ %s\n", indent, idx)
		}
	}

	// Foreign keys
	if len(diff.MissingFKs) > 0 {
		fmt.Fprintf(b, "%sForeign keys missing from DB:\n", indent)
		for _, fk := range diff.MissingFKs {
			fmt.Fprintf(b, "%s  - %s\n", indent, fk)
		}
	}
	if len(diff.ExtraFKs) > 0 {
		fmt.Fprintf(b, "%sForeign keys only in DB:\n", indent)
		for _, fk := range diff.ExtraFKs {
			fmt.Fprintf(b, "%s  + %s\n", indent, fk)
		}
	}
	if len(diff.ModifiedFKs) > 0 {
		fmt.Fprintf(b, "%sForeign keys with different definitions:\n", indent)
		for _, fk := range diff.ModifiedFKs {
			fmt.Fprintf(b, "%s  ~ %s\n", indent, fk)
		}
	}
}

// FormatSummary formats a drift summary for brief output.
func FormatSummary(summary *DriftSummary) string {
	if summary == nil {
		return "No summary available."
	}

	total := summary.MissingTables + summary.ExtraTables + summary.ModifiedTables
	if total == 0 {
		return fmt.Sprintf("No drift detected. %d tables in sync.", summary.Tables)
	}

	var parts []string
	if summary.MissingTables > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", summary.MissingTables))
	}
	if summary.ExtraTables > 0 {
		parts = append(parts, fmt.Sprintf("%d extra", summary.ExtraTables))
	}
	if summary.ModifiedTables > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", summary.ModifiedTables))
	}

	return fmt.Sprintf("Drift detected: %s", strings.Join(parts, ", "))
}

// FormatQuickStatus formats a quick status line for drift detection.
func FormatQuickStatus(hasDrift bool, expectedHash, actualHash string) string {
	if !hasDrift {
		return fmt.Sprintf("OK  %s", truncateHash(expectedHash))
	}
	return fmt.Sprintf("DRIFT  expected: %s  actual: %s",
		truncateHash(expectedHash), truncateHash(actualHash))
}

// truncateHash returns the first 12 characters of a hash for display.
func truncateHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
