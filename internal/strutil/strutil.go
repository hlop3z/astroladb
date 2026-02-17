// Package strutil provides string utilities for case conversion and SQL naming
// used throughout the Alab codebase.
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

// QualifiedName returns the dot-separated qualified name (namespace.table or table).
// Example: QualifiedName("auth", "users") -> "auth.users"
// Example: QualifiedName("", "users") -> "users"
func QualifiedName(namespace, table string) string {
	if namespace == "" {
		return table
	}
	return namespace + "." + table
}

// FKColumn returns the foreign key column name for a table.
// Example: FKColumn("user") -> "user_id"
func FKColumn(table string) string {
	return table + "_id"
}

// IndexName returns the index name for a table and columns.
// Example: IndexName("users", "email") -> "idx_users_email"
// Example: IndexName("users", "first_name", "last_name") -> "idx_users_first_name_last_name"
func IndexName(table string, cols ...string) string {
	parts := []string{"idx", table}
	parts = append(parts, cols...)
	return strings.Join(parts, "_")
}

// -----------------------------------------------------------------------------
// Reference Resolution
// -----------------------------------------------------------------------------

// ResolveRef resolves a relative reference ".table" to "namespace.table".
// Fully qualified or bare refs are returned unchanged.
func ResolveRef(ref, namespace string) string {
	if namespace != "" && len(ref) > 0 && ref[0] == '.' {
		return namespace + ref // ".user" → "auth.user"
	}
	return ref
}

// -----------------------------------------------------------------------------
// Reference Parsing
// -----------------------------------------------------------------------------

// ParseRef splits a dot-separated reference into namespace and table.
// Handles relative refs: ".table" → ("", "table").
// Empty or no-dot input → ("", ref).
func ParseRef(ref string) (namespace, table string) {
	if ref == "" {
		return "", ""
	}
	if ref[0] == '.' {
		return "", ref[1:]
	}
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			return ref[:i], ref[i+1:]
		}
	}
	return "", ref
}

// ExtractTableName returns the table portion of a reference.
// "auth.user" → "user", "user" → "user"
func ExtractTableName(ref string) string {
	_, table := ParseRef(ref)
	return table
}

// -----------------------------------------------------------------------------
// Formatting
// -----------------------------------------------------------------------------

// Indent indents each non-empty line of text with the given number of spaces.
func Indent(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// QuoteSQL quotes a SQL identifier with double quotes, escaping embedded quotes.
func QuoteSQL(name string) string {
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}
