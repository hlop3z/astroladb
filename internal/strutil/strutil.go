// Package strutil provides string utilities for case conversion, SQL naming,
// and pluralization used throughout the Alab codebase.
package strutil

import (
	"strings"
	"unicode"
)

// -----------------------------------------------------------------------------
// Case Conversion
// -----------------------------------------------------------------------------

// ToSnakeCase converts a string to snake_case.
// Examples: userName -> user_name, UserName -> user_name, HTTPServer -> http_server
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s) + 4) // pre-allocate with some extra space for underscores

	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter if:
			// - Not at the start
			// - Previous char is lowercase, OR
			// - Next char exists and is lowercase (handles "HTTPServer" -> "http_server")
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) {
					result.WriteByte('_')
				} else if i+1 < len(s) && unicode.IsLower(rune(s[i+1])) {
					result.WriteByte('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == ' ' {
			// Convert dashes and spaces to underscores
			result.WriteByte('_')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToPascalCase converts a string to PascalCase.
// Examples: user_name -> UserName, user-name -> UserName
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s))

	capitalizeNext := true
	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

// ToCamelCase converts a string to camelCase.
// Examples: user_name -> userName, UserName -> userName
func ToCamelCase(s string) string {
	if s == "" {
		return ""
	}

	pascal := ToPascalCase(s)
	if pascal == "" {
		return ""
	}

	// Lowercase the first letter
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// -----------------------------------------------------------------------------
// SQL Naming
// -----------------------------------------------------------------------------

// SQLName creates a qualified SQL table name from namespace and table.
// Example: SQLName("auth", "users") -> "auth_users"
func SQLName(namespace, table string) string {
	if namespace == "" {
		return table
	}
	return namespace + "_" + table
}

// QualifyTable returns namespace_table or just table if no namespace.
// This is an alias for SQLName for semantic clarity.
// Example: QualifyTable("auth", "user") -> "auth_user"
func QualifyTable(namespace, table string) string {
	return SQLName(namespace, table)
}

// FKColumn returns the foreign key column name for a table.
// It singularizes the table name and appends "_id".
// Example: FKColumn("users") -> "user_id"
func FKColumn(table string) string {
	return Singularize(table) + "_id"
}

// FKName returns the foreign key constraint name.
// Example: FKName("posts", "user_id", "users") -> "fk_posts_user_id"
func FKName(table, column, ref string) string {
	// ref is included for context but we use table_column pattern
	_ = ref
	return "fk_" + table + "_" + column
}

// IndexName returns the index name for a table and columns.
// Example: IndexName("users", "email") -> "idx_users_email"
// Example: IndexName("users", "first_name", "last_name") -> "idx_users_first_name_last_name"
func IndexName(table string, cols ...string) string {
	parts := []string{"idx", table}
	parts = append(parts, cols...)
	return strings.Join(parts, "_")
}

// UniqueIndexName returns the unique index name for a table and columns.
// Example: UniqueIndexName("users", "email") -> "uniq_users_email"
func UniqueIndexName(table string, cols ...string) string {
	parts := []string{"uniq", table}
	parts = append(parts, cols...)
	return strings.Join(parts, "_")
}

// UniqueConstraintName returns the unique constraint name for a table and columns.
// Example: UniqueConstraintName("users", "email") -> "uniq_users_email"
func UniqueConstraintName(table string, cols ...string) string {
	parts := []string{"uniq", table}
	parts = append(parts, cols...)
	return strings.Join(parts, "_")
}

// CheckConstraintName returns the check constraint name for a table and column/name.
// Example: CheckConstraintName("users", "status") -> "chk_users_status"
func CheckConstraintName(table string, name string) string {
	return "chk_" + table + "_" + name
}

// JoinTableName returns the join table name for a many-to-many relationship.
// Tables are sorted alphabetically to ensure consistent naming.
// Example: JoinTableName("users", "roles") -> "roles_users"
// Example: JoinTableName("roles", "users") -> "roles_users"
func JoinTableName(tableA, tableB string) string {
	if tableA < tableB {
		return tableA + "_" + tableB
	}
	return tableB + "_" + tableA
}

// -----------------------------------------------------------------------------
// Pluralization
// -----------------------------------------------------------------------------

// irregularPlurals maps singular words to their irregular plural forms.
var irregularPlurals = map[string]string{
	"person":   "people",
	"child":    "children",
	"man":      "men",
	"woman":    "women",
	"tooth":    "teeth",
	"foot":     "feet",
	"mouse":    "mice",
	"goose":    "geese",
	"ox":       "oxen",
	"leaf":     "leaves",
	"life":     "lives",
	"knife":    "knives",
	"wife":     "wives",
	"self":     "selves",
	"elf":      "elves",
	"loaf":     "loaves",
	"potato":   "potatoes",
	"tomato":   "tomatoes",
	"cactus":   "cacti",
	"focus":    "foci",
	"fungus":   "fungi",
	"nucleus":  "nuclei",
	"syllabus": "syllabi",
	"analysis": "analyses",
	"basis":    "bases",
	"crisis":   "crises",
	"thesis":   "theses",
	"datum":    "data",
	"medium":   "media",
	"index":    "indices",
}

// irregularSingulars is the reverse mapping for singularization.
var irregularSingulars map[string]string

func init() {
	irregularSingulars = make(map[string]string, len(irregularPlurals))
	for singular, plural := range irregularPlurals {
		irregularSingulars[plural] = singular
	}
}

// uncountables are words that don't change between singular and plural.
var uncountables = map[string]bool{
	"equipment":   true,
	"information": true,
	"rice":        true,
	"money":       true,
	"species":     true,
	"series":      true,
	"fish":        true,
	"sheep":       true,
	"deer":        true,
	"aircraft":    true,
	"news":        true,
	"data":        true,
	"metadata":    true,
}

// Pluralize converts a singular word to its plural form.
// Examples: user -> users, category -> categories, person -> people
func Pluralize(s string) string {
	if s == "" {
		return ""
	}

	lower := strings.ToLower(s)

	// Check uncountables
	if uncountables[lower] {
		return s
	}

	// Check irregular plurals
	if plural, ok := irregularPlurals[lower]; ok {
		return matchCase(s, plural)
	}

	// Apply standard English pluralization rules
	switch {
	case strings.HasSuffix(lower, "s") ||
		strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "z") ||
		strings.HasSuffix(lower, "ch") ||
		strings.HasSuffix(lower, "sh"):
		return s + "es"

	case strings.HasSuffix(lower, "y"):
		if len(s) > 1 && !isVowel(rune(lower[len(lower)-2])) {
			return s[:len(s)-1] + "ies"
		}
		return s + "s"

	case strings.HasSuffix(lower, "f"):
		return s[:len(s)-1] + "ves"

	case strings.HasSuffix(lower, "fe"):
		return s[:len(s)-2] + "ves"

	default:
		return s + "s"
	}
}

// Singularize converts a plural word to its singular form.
// Examples: users -> user, categories -> category, people -> person
func Singularize(s string) string {
	if s == "" {
		return ""
	}

	lower := strings.ToLower(s)

	// Check uncountables
	if uncountables[lower] {
		return s
	}

	// Check irregular singulars
	if singular, ok := irregularSingulars[lower]; ok {
		return matchCase(s, singular)
	}

	// Apply reverse pluralization rules
	switch {
	case strings.HasSuffix(lower, "ies"):
		if len(s) > 3 {
			return s[:len(s)-3] + "y"
		}

	case strings.HasSuffix(lower, "ves"):
		if len(s) > 3 {
			// Could be "leaves" -> "leaf" or "knives" -> "knife"
			base := s[:len(s)-3]
			// Try both -f and -fe patterns
			if _, ok := irregularPlurals[strings.ToLower(base)+"f"]; ok {
				return base + "f"
			}
			if _, ok := irregularPlurals[strings.ToLower(base)+"fe"]; ok {
				return base + "fe"
			}
			return base + "f"
		}

	case strings.HasSuffix(lower, "ches") ||
		strings.HasSuffix(lower, "shes") ||
		strings.HasSuffix(lower, "sses") ||
		strings.HasSuffix(lower, "xes") ||
		strings.HasSuffix(lower, "zes"):
		if len(s) > 2 {
			return s[:len(s)-2]
		}

	case strings.HasSuffix(lower, "ses"):
		// Could be "buses" -> "bus" or "bases" -> "basis"
		if len(s) > 2 {
			return s[:len(s)-2]
		}

	case strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss"):
		if len(s) > 1 {
			return s[:len(s)-1]
		}
	}

	return s
}

// -----------------------------------------------------------------------------
// Validation Helpers
// -----------------------------------------------------------------------------

// IsSnakeCase returns true if the string is valid snake_case.
// Valid snake_case: lowercase letters, numbers, and underscores.
// Cannot start or end with underscore, no consecutive underscores.
func IsSnakeCase(s string) bool {
	if s == "" {
		return false
	}

	// Cannot start or end with underscore
	if s[0] == '_' || s[len(s)-1] == '_' {
		return false
	}

	prevUnderscore := false
	for _, r := range s {
		if r == '_' {
			if prevUnderscore {
				return false // consecutive underscores
			}
			prevUnderscore = true
			continue
		}
		prevUnderscore = false

		if !unicode.IsLower(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

// IsUpperCase returns true if all letters in the string are uppercase.
// Non-letter characters are ignored.
// Example: "ALL_CAPS" -> true, "MixedCase" -> false
func IsUpperCase(s string) bool {
	if s == "" {
		return false
	}

	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}

	return hasLetter
}

// ContainsUpper returns true if the string contains any uppercase letter.
func ContainsUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------------------

// isVowel returns true if the rune is a vowel.
func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// matchCase returns the replacement string with the same case pattern as the original.
func matchCase(original, replacement string) string {
	if original == "" || replacement == "" {
		return replacement
	}

	// Check if original is all uppercase
	if IsUpperCase(original) {
		return strings.ToUpper(replacement)
	}

	// Check if original starts with uppercase (Title case)
	if unicode.IsUpper(rune(original[0])) {
		runes := []rune(replacement)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}

	return replacement
}
