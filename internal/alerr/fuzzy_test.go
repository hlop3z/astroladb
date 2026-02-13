package alerr

import "testing"

// -----------------------------------------------------------------------------
// Levenshtein Distance Tests
// -----------------------------------------------------------------------------

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1, s2 string
		want   int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"string", "srting", 2},   // transposition (2 edits, not adjacent swap)
		{"integer", "integar", 1}, // single substitution (e->a)
		{"boolean", "boolen", 1},  // single deletion ('a' missing)
		{"a", "b", 1},
		{"ab", "ba", 2},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			got := levenshteinDistance(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// FindClosestMatch Tests
// -----------------------------------------------------------------------------

func TestFindClosestMatch(t *testing.T) {
	typeNames := []string{
		"id", "string", "text", "integer", "float", "decimal",
		"boolean", "date", "time", "datetime", "uuid", "json", "base64", "enum", "computed",
	}

	tests := []struct {
		input   string
		wantOk  bool
		wantVal string
	}{
		{"integar", true, "integer"},   // common typo
		{"srting", true, "string"},     // transposition
		{"boolen", true, "boolean"},    // missing letter
		{"flot", true, "float"},        // missing letter
		{"deciml", true, "decimal"},    // missing letter
		{"jsn", true, "json"},          // missing letters
		{"enm", true, "enum"},          // missing letter
		{"dat", true, "date"},          // missing letter
		{"xyzzyx", false, ""},          // no match (too far)
		{"completelywrong", false, ""}, // no match
		{"integer", true, "integer"},   // exact match (distance 0)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match, ok := FindClosestMatch(tt.input, typeNames)
			if ok != tt.wantOk {
				t.Errorf("FindClosestMatch(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if ok && match != tt.wantVal {
				t.Errorf("FindClosestMatch(%q) = %q, want %q", tt.input, match, tt.wantVal)
			}
		})
	}

	t.Run("empty options", func(t *testing.T) {
		_, ok := FindClosestMatch("test", nil)
		if ok {
			t.Error("expected no match with empty options")
		}
	})
}

// -----------------------------------------------------------------------------
// SuggestSimilar Tests
// -----------------------------------------------------------------------------

func TestSuggestSimilar(t *testing.T) {
	options := []string{"users", "posts", "comments"}

	t.Run("returns suggestion for close match", func(t *testing.T) {
		got := SuggestSimilar("usrs", options)
		want := "did you mean 'users'?"
		if got != want {
			t.Errorf("SuggestSimilar('usrs') = %q, want %q", got, want)
		}
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		got := SuggestSimilar("xyzzyx", options)
		if got != "" {
			t.Errorf("SuggestSimilar('xyzzyx') = %q, want empty", got)
		}
	})

	t.Run("returns suggestion for exact match", func(t *testing.T) {
		got := SuggestSimilar("users", options)
		want := "did you mean 'users'?"
		if got != want {
			t.Errorf("SuggestSimilar('users') = %q, want %q", got, want)
		}
	})
}
