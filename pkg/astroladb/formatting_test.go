package astroladb

import (
	"strings"
	"testing"

	"github.com/hlop3z/astroladb/internal/testutil"
)

// ===========================================================================
// FormatDriftResult Tests
// ===========================================================================

func TestFormatDriftResult_NoDrift(t *testing.T) {
	// Create a drift result with no drift
	result := &DriftResult{
		HasDrift: false,
	}

	// Format the result
	formatted := FormatDriftResult(result)
	testutil.AssertTrue(t, len(formatted) > 0, "should have formatted output")
}

func TestFormatDriftResult_WithDrift(t *testing.T) {
	// Create a drift result with drift
	result := &DriftResult{
		HasDrift: true,
	}

	// Format the result
	formatted := FormatDriftResult(result)
	testutil.AssertTrue(t, len(formatted) > 0, "should have formatted output")
	testutil.AssertTrue(t, strings.Contains(formatted, "drift") || strings.Contains(formatted, "Drift"), "should mention drift")
}

func TestFormatDriftResult_Nil(t *testing.T) {
	// Format nil result
	formatted := FormatDriftResult(nil)
	testutil.AssertTrue(t, len(formatted) > 0, "should handle nil result")
}
