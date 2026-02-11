// Package engine provides migration planning and execution.
package engine

import (
	"math"
	"strings"

	"github.com/hlop3z/astroladb/internal/ast"
)

// RenameCandidate represents a potential rename operation detected from diff.
type RenameCandidate struct {
	Type      string  // "column" or "table"
	Table     string  // Table name (for column renames)
	Namespace string  // Namespace (for both)
	OldName   string  // Old name
	NewName   string  // New name
	ColType   string  // Column type (for column renames, used for matching)
	Score     float64 // Confidence score (0.0 - 1.0)
}

// renameThreshold is the minimum score for a rename candidate to be considered.
const renameThreshold = 0.5

// DetectRenames analyzes diff operations to find potential renames.
// It looks for patterns like:
//   - Drop column + Add column (same table, same type) -> possible column rename
//   - Drop table + Create table (similar structure) -> possible table rename
//
// Returns candidates that the user should confirm.
// colInfo holds information about a column-level operation for rename detection.
type colInfo struct {
	op        ast.Operation
	namespace string
	table     string
	name      string
	colType   string
	index     int
}

func DetectRenames(ops []ast.Operation) []RenameCandidate {
	var candidates []RenameCandidate

	var dropColumns []colInfo
	var addColumns []colInfo
	var dropTables []colInfo
	var createTables []colInfo

	for i, op := range ops {
		switch o := op.(type) {
		case *ast.DropColumn:
			dropColumns = append(dropColumns, colInfo{
				op:        op,
				namespace: o.Namespace,
				table:     o.Table_,
				name:      o.Name,
				index:     i,
			})
		case *ast.AddColumn:
			addColumns = append(addColumns, colInfo{
				op:        op,
				namespace: o.Namespace,
				table:     o.Table_,
				name:      o.Column.Name,
				colType:   o.Column.Type,
				index:     i,
			})
		case *ast.DropTable:
			dropTables = append(dropTables, colInfo{
				op:        op,
				namespace: o.Namespace,
				name:      o.Name,
				index:     i,
			})
		case *ast.CreateTable:
			createTables = append(createTables, colInfo{
				op:        op,
				namespace: o.Namespace,
				name:      o.Name,
				index:     i,
			})
		}
	}

	// Match drop columns with add columns using weighted scoring
	usedDrops := make(map[int]bool)
	usedAdds := make(map[int]bool)

	// Build scored candidates and pick best matches
	type scoredMatch struct {
		drop, add int
		candidate RenameCandidate
	}
	var columnMatches []scoredMatch

	for di, drop := range dropColumns {
		for ai, add := range addColumns {
			if drop.namespace != add.namespace || drop.table != add.table {
				continue
			}
			if drop.name == add.name {
				continue
			}

			score := scoreColumnRename(drop, add, len(dropColumns))
			if score >= renameThreshold {
				columnMatches = append(columnMatches, scoredMatch{
					drop: di, add: ai,
					candidate: RenameCandidate{
						Type:      "column",
						Table:     drop.table,
						Namespace: drop.namespace,
						OldName:   drop.name,
						NewName:   add.name,
						ColType:   add.colType,
						Score:     score,
					},
				})
			}
		}
	}

	// Sort by score descending and greedily assign
	for i := 0; i < len(columnMatches); i++ {
		best := -1
		bestScore := 0.0
		for j := i; j < len(columnMatches); j++ {
			if columnMatches[j].candidate.Score > bestScore {
				best = j
				bestScore = columnMatches[j].candidate.Score
			}
		}
		if best >= 0 {
			columnMatches[i], columnMatches[best] = columnMatches[best], columnMatches[i]
		}
	}
	for _, m := range columnMatches {
		if usedDrops[m.drop] || usedAdds[m.add] {
			continue
		}
		candidates = append(candidates, m.candidate)
		usedDrops[m.drop] = true
		usedAdds[m.add] = true
	}

	// Fallback: match remaining unmatched drops with adds in the same table
	// (preserves original behavior for very dissimilar names)
	for di, drop := range dropColumns {
		if usedDrops[di] {
			continue
		}
		for ai, add := range addColumns {
			if usedAdds[ai] {
				continue
			}
			if drop.namespace == add.namespace && drop.table == add.table && drop.name != add.name {
				candidates = append(candidates, RenameCandidate{
					Type:      "column",
					Table:     drop.table,
					Namespace: drop.namespace,
					OldName:   drop.name,
					NewName:   add.name,
					ColType:   add.colType,
					Score:     scoreColumnRename(drop, add, len(dropColumns)),
				})
				usedDrops[di] = true
				usedAdds[ai] = true
				break
			}
		}
	}

	// Match drop tables with create tables using name similarity
	usedDropTables := make(map[int]bool)
	usedCreateTables := make(map[int]bool)

	for _, drop := range dropTables {
		bestIdx := -1
		bestScore := 0.0
		for ci, create := range createTables {
			if usedCreateTables[ci] {
				continue
			}
			if drop.namespace != create.namespace || drop.name == create.name {
				continue
			}
			score := JaroWinkler(drop.name, create.name)
			if score > bestScore && score >= renameThreshold {
				bestIdx = ci
				bestScore = score
			}
		}
		if bestIdx < 0 {
			// No match above threshold; still consider if same namespace (original behavior)
			for ci, create := range createTables {
				if usedCreateTables[ci] || drop.namespace != create.namespace || drop.name == create.name {
					continue
				}
				bestIdx = ci
				bestScore = JaroWinkler(drop.name, create.name)
				break
			}
		}
		if bestIdx >= 0 {
			candidates = append(candidates, RenameCandidate{
				Type:      "table",
				Namespace: drop.namespace,
				OldName:   drop.name,
				NewName:   createTables[bestIdx].name,
				Score:     bestScore,
			})
			usedDropTables[drop.index] = true
			usedCreateTables[bestIdx] = true
		}
	}
	_ = usedDropTables // suppress unused

	return candidates
}

// ApplyRenames transforms operations by replacing drop+add pairs with rename operations.
// It takes the original operations and confirmed renames, returns modified operations.
func ApplyRenames(ops []ast.Operation, confirmed []RenameCandidate) []ast.Operation {
	if len(confirmed) == 0 {
		return ops
	}

	// Build lookup maps for quick matching
	columnRenames := make(map[string]RenameCandidate) // "ns.table.oldcol" -> rename
	tableRenames := make(map[string]RenameCandidate)  // "ns.oldtable" -> rename

	for _, r := range confirmed {
		switch r.Type {
		case "column":
			key := r.Namespace + "." + r.Table + "." + r.OldName
			columnRenames[key] = r
		case "table":
			key := r.Namespace + "." + r.OldName
			tableRenames[key] = r
		}
	}

	var result []ast.Operation
	skipIndices := make(map[int]bool)

	// First pass: identify operations to skip (they'll be replaced by renames)
	for i, op := range ops {
		switch o := op.(type) {
		case *ast.DropColumn:
			key := o.Namespace + "." + o.Table_ + "." + o.Name
			if _, ok := columnRenames[key]; ok {
				skipIndices[i] = true
			}
		case *ast.AddColumn:
			// Find if this add corresponds to a rename
			for _, r := range confirmed {
				if r.Type == "column" &&
					r.Namespace == o.Namespace &&
					r.Table == o.Table_ &&
					r.NewName == o.Column.Name {
					skipIndices[i] = true
					break
				}
			}
		case *ast.DropTable:
			key := o.Namespace + "." + o.Name
			if _, ok := tableRenames[key]; ok {
				skipIndices[i] = true
			}
		case *ast.CreateTable:
			// Find if this create corresponds to a rename
			for _, r := range confirmed {
				if r.Type == "table" &&
					r.Namespace == o.Namespace &&
					r.NewName == o.Name {
					skipIndices[i] = true
					break
				}
			}
		}
	}

	// Second pass: build result with renames injected
	addedRenames := make(map[string]bool)

	for i, op := range ops {
		if skipIndices[i] {
			// Check if we should inject a rename operation here
			switch o := op.(type) {
			case *ast.DropColumn:
				key := o.Namespace + "." + o.Table_ + "." + o.Name
				if r, ok := columnRenames[key]; ok && !addedRenames[key] {
					result = append(result, &ast.RenameColumn{
						TableRef: ast.TableRef{
							Namespace: r.Namespace,
							Table_:    r.Table,
						},
						OldName: r.OldName,
						NewName: r.NewName,
					})
					addedRenames[key] = true
				}
			case *ast.DropTable:
				key := o.Namespace + "." + o.Name
				if r, ok := tableRenames[key]; ok && !addedRenames[key] {
					result = append(result, &ast.RenameTable{
						Namespace: r.Namespace,
						OldName:   r.OldName,
						NewName:   r.NewName,
					})
					addedRenames[key] = true
				}
			}
			continue
		}
		result = append(result, op)
	}

	return result
}

// scoreColumnRename computes a weighted rename confidence score for a column pair.
// Weights: type match (0.4), name similarity (0.3), constraints (0.2), position (0.1).
func scoreColumnRename(drop, add colInfo, totalDrops int) float64 {
	var score float64

	// Type match (0.4) - if we have type info from the add side
	if add.colType != "" && drop.colType != "" {
		if strings.EqualFold(drop.colType, add.colType) {
			score += 0.4
		}
	} else if add.colType != "" {
		// Drop columns don't always carry type info; give partial credit
		score += 0.2
	}

	// Name similarity (0.3) via Jaro-Winkler
	nameSim := JaroWinkler(drop.name, add.name)
	score += 0.3 * nameSim

	// Constraint similarity (0.2) - basic: same table is already guaranteed
	score += 0.2

	// Position proximity (0.1)
	if totalDrops > 0 {
		dist := math.Abs(float64(drop.index - add.index))
		maxDist := float64(totalDrops)
		if maxDist > 0 {
			score += 0.1 * (1.0 - dist/maxDist)
		}
	}

	return score
}

// JaroWinkler computes the Jaro-Winkler similarity between two strings.
// Returns a value between 0.0 (no similarity) and 1.0 (identical).
func JaroWinkler(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Jaro distance
	maxDist := len(s1)
	if len(s2) > maxDist {
		maxDist = len(s2)
	}
	maxDist = maxDist/2 - 1
	if maxDist < 0 {
		maxDist = 0
	}

	s1Matches := make([]bool, len(s1))
	s2Matches := make([]bool, len(s2))

	matches := 0
	transpositions := 0

	for i := range s1 {
		start := i - maxDist
		if start < 0 {
			start = 0
		}
		end := i + maxDist + 1
		if end > len(s2) {
			end = len(s2)
		}

		for j := start; j < end; j++ {
			if s2Matches[j] || s1[i] != s2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	k := 0
	for i := range s1 {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if s1[i] != s2[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(len(s1)) +
		float64(matches)/float64(len(s2)) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0

	// Winkler modification: boost for common prefix (up to 4 chars)
	prefix := 0
	for i := 0; i < len(s1) && i < len(s2) && i < 4; i++ {
		if s1[i] == s2[i] {
			prefix++
		} else {
			break
		}
	}

	return jaro + float64(prefix)*0.1*(1.0-jaro)
}
