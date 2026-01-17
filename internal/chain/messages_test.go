package chain

import (
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// FormatVerificationResult Tests
// -----------------------------------------------------------------------------

func TestFormatVerificationResult(t *testing.T) {
	t.Run("valid_chain_no_pending", func(t *testing.T) {
		result := &VerificationResult{
			Valid:        true,
			AppliedLinks: []Link{{Revision: "001", Name: "init"}},
			PendingLinks: []Link{},
		}
		output := FormatVerificationResult(result)
		if !strings.Contains(output, "VALID") {
			t.Error("output should contain 'VALID'")
		}
		if !strings.Contains(output, "1 migration") {
			t.Error("output should show applied count")
		}
	})

	t.Run("valid_chain_with_pending", func(t *testing.T) {
		result := &VerificationResult{
			Valid:        true,
			AppliedLinks: []Link{{Revision: "001", Name: "init"}},
			PendingLinks: []Link{{Revision: "002", Name: "add_users"}},
		}
		output := FormatVerificationResult(result)
		if !strings.Contains(output, "Pending") {
			t.Error("output should show pending section")
		}
		if !strings.Contains(output, "002") {
			t.Error("output should show pending revision")
		}
	})

	t.Run("broken_chain_tampered", func(t *testing.T) {
		result := &VerificationResult{
			Valid: false,
			MismatchedLinks: []MismatchedLink{{
				Link:             Link{Revision: "001", Name: "init"},
				ExpectedChecksum: "abc123",
				ActualChecksum:   "def456",
			}},
		}
		output := FormatVerificationResult(result)
		if !strings.Contains(output, "BROKEN") {
			t.Error("output should contain 'BROKEN'")
		}
		if !strings.Contains(output, "Tampered") {
			t.Error("output should show tampered count")
		}
	})

	t.Run("broken_chain_missing_files", func(t *testing.T) {
		result := &VerificationResult{
			Valid: false,
			MissingFiles: []AppliedMigration{{
				Revision: "001",
				Checksum: "abc123",
			}},
		}
		output := FormatVerificationResult(result)
		if !strings.Contains(output, "Missing") {
			t.Error("output should show missing count")
		}
	})

	t.Run("with_errors", func(t *testing.T) {
		result := &VerificationResult{
			Valid: false,
			Errors: []ChainError{
				{Revision: "001", Message: "file modified", Details: "checksum mismatch"},
			},
		}
		output := FormatVerificationResult(result)
		if !strings.Contains(output, "001") {
			t.Error("output should show error revision")
		}
		if !strings.Contains(output, "file modified") {
			t.Error("output should show error message")
		}
	})
}

// -----------------------------------------------------------------------------
// FormatTamperedError Tests
// -----------------------------------------------------------------------------

func TestFormatTamperedError(t *testing.T) {
	link := Link{
		Revision: "001",
		Name:     "init",
		Filename: "001_init.js",
		Checksum: "computed123",
	}

	output := FormatTamperedError(link, "expected456")

	if !strings.Contains(output, "Chain integrity broken") {
		t.Error("output should contain integrity broken message")
	}
	if !strings.Contains(output, "001_init.js") {
		t.Error("output should contain filename")
	}
	if !strings.Contains(output, "expected456") {
		t.Error("output should contain expected checksum")
	}
	if !strings.Contains(output, "computed123") {
		t.Error("output should contain computed checksum")
	}
	if !strings.Contains(output, "git checkout") {
		t.Error("output should contain fix instructions")
	}
}

// -----------------------------------------------------------------------------
// FormatMissingFileError Tests
// -----------------------------------------------------------------------------

func TestFormatMissingFileError(t *testing.T) {
	mig := AppliedMigration{
		Revision:  "002",
		Checksum:  "abc123def456",
		AppliedAt: "2024-01-01T00:00:00Z",
	}

	output := FormatMissingFileError(mig)

	if !strings.Contains(output, "Missing migration file") {
		t.Error("output should contain missing file message")
	}
	if !strings.Contains(output, "002") {
		t.Error("output should contain revision")
	}
	if !strings.Contains(output, "abc123def456") {
		t.Error("output should contain checksum")
	}
	if !strings.Contains(output, "git checkout") {
		t.Error("output should contain fix instructions")
	}
}

// -----------------------------------------------------------------------------
// FormatChainStatus Tests
// -----------------------------------------------------------------------------

func TestFormatChainStatus(t *testing.T) {
	t.Run("empty_chain", func(t *testing.T) {
		chain := &Chain{Links: []Link{}}

		output := FormatChainStatus(chain)

		if !strings.Contains(output, "Empty") {
			t.Error("output should indicate empty chain")
		}
	})

	t.Run("chain_with_links", func(t *testing.T) {
		chain := &Chain{
			Links: []Link{
				{Revision: "001", Name: "init", Filename: "001_init.js", Checksum: "abc123def456abc123def456abc123def456abc123"},
				{Revision: "002", Name: "add_users", Filename: "002_add_users.js", Checksum: "xyz789xyz789xyz789xyz789xyz789xyz789xyz789"},
			},
		}

		output := FormatChainStatus(chain)

		if !strings.Contains(output, "2 migrations") {
			t.Error("output should show migration count")
		}
		if !strings.Contains(output, "001_init.js") {
			t.Error("output should show first migration")
		}
		if !strings.Contains(output, "002_add_users.js") {
			t.Error("output should show last migration")
		}
	})

	t.Run("single_link", func(t *testing.T) {
		chain := &Chain{
			Links: []Link{
				{Revision: "001", Name: "init", Filename: "001_init.js", Checksum: "abc123def456abc123def456abc123def456abc123"},
			},
		}

		output := FormatChainStatus(chain)

		if !strings.Contains(output, "1 migration") {
			t.Error("output should show '1 migration' (singular)")
		}
	})
}

// -----------------------------------------------------------------------------
// FormatPendingMigrations Tests
// -----------------------------------------------------------------------------

func TestFormatPendingMigrations(t *testing.T) {
	t.Run("no_pending", func(t *testing.T) {
		output := FormatPendingMigrations(nil)

		if !strings.Contains(output, "No pending") {
			t.Error("output should indicate no pending migrations")
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		output := FormatPendingMigrations([]Link{})

		if !strings.Contains(output, "No pending") {
			t.Error("output should indicate no pending migrations")
		}
	})

	t.Run("with_pending", func(t *testing.T) {
		pending := []Link{
			{Revision: "003", Name: "add_posts", Filename: "003_add_posts.js", Checksum: "abcdef123456abcdef123456"},
			{Revision: "004", Name: "add_comments", Filename: "004_add_comments.js", Checksum: "fedcba654321fedcba654321"},
		}

		output := FormatPendingMigrations(pending)

		if !strings.Contains(output, "Pending migrations: 2") {
			t.Error("output should show count")
		}
		if !strings.Contains(output, "003_add_posts.js") {
			t.Error("output should show first pending")
		}
		if !strings.Contains(output, "004_add_comments.js") {
			t.Error("output should show second pending")
		}
		if !strings.Contains(output, "abcdef123456") {
			t.Error("output should show truncated checksums")
		}
	})
}

// -----------------------------------------------------------------------------
// FormatAppliedMigration Tests
// -----------------------------------------------------------------------------

func TestFormatAppliedMigration(t *testing.T) {
	link := Link{
		Revision:     "001",
		Name:         "init",
		Filename:     "001_init.js",
		Checksum:     "abc123def456abc123def456",
		PrevChecksum: "00000000genesis0000000000",
	}

	output := FormatAppliedMigration(link)

	if !strings.Contains(output, "001") {
		t.Error("output should contain revision")
	}
	if !strings.Contains(output, "00000000") {
		t.Error("output should contain truncated prev checksum")
	}
	if !strings.Contains(output, "abc123de") {
		t.Error("output should contain truncated checksum")
	}
	if !strings.Contains(output, "->") {
		t.Error("output should show arrow between checksums")
	}
}

// -----------------------------------------------------------------------------
// FormatChainDiagram Tests
// -----------------------------------------------------------------------------

func TestFormatChainDiagram(t *testing.T) {
	t.Run("empty_chain", func(t *testing.T) {
		chain := &Chain{Links: []Link{}}

		output := FormatChainDiagram(chain, nil)

		if !strings.Contains(output, "empty chain") {
			t.Error("output should indicate empty chain")
		}
	})

	t.Run("chain_with_applied", func(t *testing.T) {
		chain := &Chain{
			Links: []Link{
				{Revision: "001", Name: "init", Checksum: "abc123def456abc123def456"},
				{Revision: "002", Name: "add_users", Checksum: "xyz789xyz789xyz789xyz789"},
			},
		}
		applied := []AppliedMigration{
			{Revision: "001", Checksum: "abc123def456abc123def456"},
		}

		output := FormatChainDiagram(chain, applied)

		if !strings.Contains(output, "genesis") {
			t.Error("output should start with genesis")
		}
		if !strings.Contains(output, "[+] 001") {
			t.Error("output should show applied marker for 001")
		}
		if !strings.Contains(output, "[ ] 002") {
			t.Error("output should show pending marker for 002")
		}
		if !strings.Contains(output, "Legend") {
			t.Error("output should include legend")
		}
	})

	t.Run("all_applied", func(t *testing.T) {
		chain := &Chain{
			Links: []Link{
				{Revision: "001", Name: "init", Checksum: "abc123def456abc123def456"},
			},
		}
		applied := []AppliedMigration{
			{Revision: "001", Checksum: "abc123def456abc123def456"},
		}

		output := FormatChainDiagram(chain, applied)

		if !strings.Contains(output, "[+] 001") {
			t.Error("output should show applied marker")
		}
	})

	t.Run("none_applied", func(t *testing.T) {
		chain := &Chain{
			Links: []Link{
				{Revision: "001", Name: "init", Checksum: "abc123def456abc123def456"},
			},
		}

		output := FormatChainDiagram(chain, nil)

		if !strings.Contains(output, "[ ] 001") {
			t.Error("output should show pending marker")
		}
	})
}

// -----------------------------------------------------------------------------
// Edge Cases
// -----------------------------------------------------------------------------

func TestFormatVerificationResult_MultipleErrors(t *testing.T) {
	result := &VerificationResult{
		Valid: false,
		Errors: []ChainError{
			{Revision: "001", Message: "first error", Details: "detail 1"},
			{Revision: "002", Message: "second error", Details: "detail 2"},
			{Revision: "003", Message: "third error"},
		},
	}

	output := FormatVerificationResult(result)

	if !strings.Contains(output, "first error") {
		t.Error("output should contain first error")
	}
	if !strings.Contains(output, "second error") {
		t.Error("output should contain second error")
	}
	if !strings.Contains(output, "third error") {
		t.Error("output should contain third error")
	}
}

func TestFormatChainDiagram_LongChain(t *testing.T) {
	links := make([]Link, 5)
	for i := range links {
		links[i] = Link{
			Revision: string(rune('A' + i)),
			Name:     "migration",
			Checksum: "checksum1234567890123456",
		}
	}
	chain := &Chain{Links: links}

	output := FormatChainDiagram(chain, nil)

	// Should have | for intermediate links and v for last
	if strings.Count(output, "|") < 4 {
		t.Error("output should have pipe connectors between links")
	}
	if !strings.Contains(output, "v") {
		t.Error("output should have v connector for last link")
	}
}
