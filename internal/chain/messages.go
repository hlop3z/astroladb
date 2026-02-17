package chain

import (
	"fmt"
	"strings"

	"github.com/hlop3z/astroladb/internal/strutil"
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
				errMsg += "\n" + strutil.Indent(err.Details, 2)
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
