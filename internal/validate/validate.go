// Package validate provides validation helpers for SQL identifiers, references, and schemas.
// This package enforces the naming conventions required by Alab: snake_case everywhere,
// no reserved words, and proper reference resolution.
package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// -----------------------------------------------------------------------------
// Error Codes (re-exported from alerr for local use)
// -----------------------------------------------------------------------------

const (
	ErrInvalidIdentifier = alerr.ErrInvalidIdentifier
	ErrInvalidSnakeCase  = alerr.ErrInvalidSnakeCase
	ErrInvalidReference  = alerr.ErrInvalidReference
	ErrReservedWord      = alerr.ErrReservedWord
)

// -----------------------------------------------------------------------------
// SQL Reserved Words
// -----------------------------------------------------------------------------

// reservedWords contains SQL reserved words from SQL standard and major dialects.
// This is a combined set covering PostgreSQL and SQLite.
var reservedWords = map[string]bool{
	// SQL Standard Keywords
	"add":        true,
	"all":        true,
	"alter":      true,
	"and":        true,
	"any":        true,
	"as":         true,
	"asc":        true,
	"between":    true,
	"by":         true,
	"case":       true,
	"check":      true,
	"column":     true,
	"constraint": true,
	"create":     true,
	"cross":      true,
	"current":    true,
	"database":   true,
	"default":    true,
	"delete":     true,
	"desc":       true,
	"distinct":   true,
	"drop":       true,
	"else":       true,
	"end":        true,
	"exists":     true,
	"false":      true,
	"fetch":      true,
	"for":        true,
	"foreign":    true,
	"from":       true,
	"full":       true,
	"grant":      true,
	"group":      true,
	"having":     true,
	"if":         true,
	"in":         true,
	"index":      true,
	"inner":      true,
	"insert":     true,
	"into":       true,
	"is":         true,
	"join":       true,
	"key":        true,
	"left":       true,
	"like":       true,
	"limit":      true,
	"not":        true,
	"null":       true,
	"offset":     true,
	"on":         true,
	"or":         true,
	"order":      true,
	"outer":      true,
	"primary":    true,
	"references": true,
	"revoke":     true,
	"right":      true,
	"select":     true,
	"set":        true,
	"table":      true,
	"then":       true,
	"to":         true,
	"true":       true,
	"union":      true,
	"unique":     true,
	"update":     true,
	"using":      true,
	"values":     true,
	"view":       true,
	"when":       true,
	"where":      true,
	"with":       true,

	// PostgreSQL specific
	"abort":     true,
	"analyze":   true,
	"array":     true,
	"begin":     true,
	"cast":      true,
	"commit":    true,
	"copy":      true,
	"do":        true,
	"except":    true,
	"explain":   true,
	"freeze":    true,
	"ilike":     true,
	"intersect": true,
	"isnull":    true,
	"lateral":   true,
	"leading":   true,
	"localtime": true,
	"lock":      true,
	"natural":   true,
	"notnull":   true,
	"only":      true,
	"placing":   true,
	"returning": true,
	"rollback":  true,
	"row":       true,
	"savepoint": true,
	"similar":   true,
	"some":      true,
	"symmetric": true,
	"trailing":  true,
	"truncate":  true,
	"user":      true,
	"vacuum":    true,
	"variadic":  true,
	"verbose":   true,
	"window":    true,

	// SQLite specific
	"action":    true,
	"after":     true,
	"attach":    true,
	"conflict":  true,
	"detach":    true,
	"fail":      true,
	"glob":      true,
	"indexed":   true,
	"instead":   true,
	"plan":      true,
	"pragma":    true,
	"query":     true,
	"raise":     true,
	"reindex":   true,
	"temp":      true,
	"temporary": true,
	"virtual":   true,

	// Common type names to avoid confusion
	"boolean":   true,
	"bool":      true,
	"date":      true,
	"enum":      true,
	"json":      true,
	"jsonb":     true,
	"uuid":      true,
	"serial":    true,
	"bigserial": true,
}

// IsReservedWord checks if the given string is a SQL reserved word.
// The check is case-insensitive.
func IsReservedWord(s string) bool {
	return reservedWords[strings.ToLower(s)]
}

// ReservedWordError returns an error if s is a reserved word, nil otherwise.
func ReservedWordError(s string) error {
	if IsReservedWord(s) {
		return alerr.New(ErrReservedWord, fmt.Sprintf("'%s' is a SQL reserved word", s)).
			With("identifier", s).
			With("suggestion", s+"_col or "+s+"_table")
	}
	return nil
}

// -----------------------------------------------------------------------------
// Snake Case Validation
// -----------------------------------------------------------------------------

// snakeCaseRegex matches valid snake_case identifiers:
// - starts with lowercase letter
// - contains only lowercase letters, digits, and underscores
// - no consecutive underscores
// - doesn't end with underscore
var snakeCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// IsSnakeCase checks if the given string is valid snake_case.
func IsSnakeCase(s string) bool {
	if s == "" {
		return false
	}
	return snakeCaseRegex.MatchString(s)
}

// -----------------------------------------------------------------------------
// Identifier Validation
// -----------------------------------------------------------------------------

// identifierRegex matches valid SQL identifiers:
// - starts with letter or underscore
// - contains only letters, digits, and underscores
// - max length 63 characters (PostgreSQL limit)
var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

const maxIdentifierLength = 63

// Identifier validates that s is a valid SQL identifier.
// A valid identifier:
// - Is not empty
// - Starts with a letter or underscore
// - Contains only letters, digits, and underscores
// - Is not longer than 63 characters
// - Is not a reserved word
func Identifier(s string) error {
	if s == "" {
		return alerr.New(ErrInvalidIdentifier, "identifier cannot be empty")
	}

	if len(s) > maxIdentifierLength {
		return alerr.New(ErrInvalidIdentifier, fmt.Sprintf("identifier exceeds maximum length of %d characters", maxIdentifierLength)).
			With("identifier", s).
			With("length", len(s))
	}

	if !identifierRegex.MatchString(s) {
		return alerr.New(ErrInvalidIdentifier, "identifier contains invalid characters").
			With("identifier", s).
			With("allowed", "letters, digits, underscores (must start with letter or underscore)")
	}

	if err := ReservedWordError(s); err != nil {
		return err
	}

	return nil
}

// SnakeCase validates that s is valid snake_case.
// A valid snake_case string:
// - Is not empty
// - Starts with a lowercase letter
// - Contains only lowercase letters, digits, and underscores
// - Has no consecutive underscores
// - Does not end with an underscore
// - Is a valid identifier (not a reserved word, proper length)
func SnakeCase(s string) error {
	if s == "" {
		return alerr.New(ErrInvalidSnakeCase, "name cannot be empty")
	}

	if !IsSnakeCase(s) {
		// Provide helpful suggestion
		suggestion := toSnakeCase(s)
		err := alerr.New(ErrInvalidSnakeCase, "name must be snake_case").
			With("got", s)
		if suggestion != s && IsSnakeCase(suggestion) {
			_ = err.With("suggestion", suggestion) //nolint:errcheck
		}
		return err
	}

	// Also validate as identifier (length, reserved words)
	return Identifier(s)
}

// Namespace validates a namespace name.
// Namespaces must be snake_case and represent a logical grouping of tables.
func Namespace(s string) error {
	if err := SnakeCase(s); err != nil {
		// Enhance error message for namespace context
		if e, ok := err.(*alerr.Error); ok {
			e.SetMessage("namespace " + e.GetMessage())
		}
		return err
	}
	return nil
}

// TableName validates a table name.
// Table names must be snake_case.
func TableName(s string) error {
	if err := SnakeCase(s); err != nil {
		// Enhance error message for table context
		if e, ok := err.(*alerr.Error); ok {
			e.SetMessage("table " + e.GetMessage())
		}
		return err
	}
	return nil
}

// ColumnName validates a column name.
// Column names must be snake_case.
func ColumnName(s string) error {
	if err := SnakeCase(s); err != nil {
		// Enhance error message for column context
		if e, ok := err.(*alerr.Error); ok {
			e.SetMessage("column " + e.GetMessage())
		}
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------
// Reference Validation
// -----------------------------------------------------------------------------

// TableRef parses a table reference in the format "namespace.table" or "table".
// Returns the namespace and table name, or an error if invalid.
// If no namespace is provided (just "table"), namespace will be empty.
func TableRef(ref string) (namespace, table string, err error) {
	if ref == "" {
		return "", "", alerr.New(ErrInvalidReference, "table reference cannot be empty")
	}

	// Check for leading dot (e.g., ".users" for same-namespace reference)
	if strings.HasPrefix(ref, ".") {
		table = strings.TrimPrefix(ref, ".")
		if table == "" {
			return "", "", alerr.New(ErrInvalidReference, "table name cannot be empty after '.'").
				With("reference", ref)
		}
		if err := TableName(table); err != nil {
			return "", "", err
		}
		return "", table, nil
	}

	parts := strings.Split(ref, ".")
	switch len(parts) {
	case 1:
		// Just table name
		table = parts[0]
		if err := TableName(table); err != nil {
			return "", "", err
		}
		return "", table, nil

	case 2:
		// namespace.table
		namespace = parts[0]
		table = parts[1]

		if namespace == "" {
			return "", "", alerr.New(ErrInvalidReference, "namespace cannot be empty").
				With("reference", ref)
		}
		if table == "" {
			return "", "", alerr.New(ErrInvalidReference, "table name cannot be empty").
				With("reference", ref)
		}

		if err := Namespace(namespace); err != nil {
			return "", "", err
		}
		if err := TableName(table); err != nil {
			return "", "", err
		}
		return namespace, table, nil

	default:
		return "", "", alerr.New(ErrInvalidReference, "invalid table reference format").
			With("reference", ref).
			With("expected", "namespace.table or table")
	}
}

// ParseRef parses a table reference and resolves it against the current namespace.
// - "namespace.table" -> (namespace, table)
// - ".table" -> (currentNS, table)  -- explicit same-namespace reference
// - "table" -> (currentNS, table)   -- implicit same-namespace reference
func ParseRef(ref, currentNS string) (namespace, table string, err error) {
	ns, tbl, err := TableRef(ref)
	if err != nil {
		return "", "", err
	}

	// If no namespace in reference, use current namespace
	if ns == "" {
		if currentNS == "" {
			return "", "", alerr.New(ErrInvalidReference, "cannot resolve relative reference without current namespace").
				With("reference", ref)
		}
		return currentNS, tbl, nil
	}

	return ns, tbl, nil
}

// -----------------------------------------------------------------------------
// Batch Validation
// -----------------------------------------------------------------------------

// ValidationErrors collects multiple validation errors.
type ValidationErrors []error

// Error returns all errors as a formatted string.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation error(s):", len(ve)))
	for i, err := range ve {
		sb.WriteString(fmt.Sprintf("\n  %d. %s", i+1, err.Error()))
	}
	return sb.String()
}

// HasErrors returns true if there are any errors in the collection.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Add appends an error to the collection if it's not nil.
func (ve *ValidationErrors) Add(err error) {
	if err != nil {
		*ve = append(*ve, err)
	}
}

// Merge adds all errors from another ValidationErrors collection.
func (ve *ValidationErrors) Merge(other ValidationErrors) {
	*ve = append(*ve, other...)
}

// ToError returns nil if no errors, or the ValidationErrors itself if there are errors.
func (ve ValidationErrors) ToError() error {
	if len(ve) == 0 {
		return nil
	}
	return ve
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// toSnakeCase converts a string to snake_case.
// This is a simplified version for generating suggestions.
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s) + 5) // Pre-allocate with some extra space for underscores

	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letters (except at start)
			if i > 0 {
				prev := rune(s[i-1])
				// Don't add underscore if previous char was uppercase or underscore
				if !unicode.IsUpper(prev) && prev != '_' {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == ' ' {
			// Convert hyphens and spaces to underscores
			result.WriteRune('_')
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result.WriteRune(unicode.ToLower(r))
		}
		// Skip other characters
	}

	// Clean up consecutive underscores
	cleaned := result.String()
	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}

	// Remove leading/trailing underscores
	cleaned = strings.Trim(cleaned, "_")

	return cleaned
}
