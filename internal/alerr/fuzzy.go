package alerr

import "fmt"

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Use two rows instead of full matrix to save memory.
	prev := make([]int, len(s2)+1)
	curr := make([]int, len(s2)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(s1); i++ {
		curr[0] = i
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost

			curr[j] = ins
			if del < curr[j] {
				curr[j] = del
			}
			if sub < curr[j] {
				curr[j] = sub
			}
		}
		prev, curr = curr, prev
	}

	return prev[len(s2)]
}

// FindClosestMatch returns the closest match from options within a max edit distance of 3.
// Returns the match and true if found, or empty string and false otherwise.
func FindClosestMatch(input string, options []string) (string, bool) {
	// maxDistance of 3 catches common typos (missing/extra char, substitution,
	// adjacent transposition) while avoiding false positives on unrelated words.
	const maxDistance = 3

	bestMatch := ""
	bestDist := maxDistance + 1

	for _, opt := range options {
		d := levenshteinDistance(input, opt)
		if d < bestDist {
			bestDist = d
			bestMatch = opt
		}
	}

	if bestDist <= maxDistance {
		return bestMatch, true
	}
	return "", false
}

// SuggestSimilar returns a "did you mean 'X'?" string if a close match is found,
// or an empty string otherwise.
func SuggestSimilar(input string, options []string) string {
	if match, ok := FindClosestMatch(input, options); ok {
		return fmt.Sprintf("did you mean '%s'?", match)
	}
	return ""
}
