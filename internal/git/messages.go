package git

import (
	"github.com/hlop3z/astroladb/internal/ui"
)

// FormatPreMigrateWarnings formats all pre-migration warnings.
func FormatPreMigrateWarnings(check *PreMigrateCheck) string {
	if len(check.Warnings) == 0 && len(check.Errors) == 0 {
		return ""
	}

	list := ui.NewList()

	for _, e := range check.Errors {
		list.AddError(e)
	}

	for _, w := range check.Warnings {
		list.AddWarning(w)
	}

	return list.String() + "\n"
}
