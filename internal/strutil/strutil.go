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

// FKColumn returns the foreign key column name for a table.
// Example: FKColumn("user") -> "user_id"
func FKColumn(table string) string {
	return table + "_id"
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
