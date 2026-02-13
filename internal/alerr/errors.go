// Package alerr provides standardized error handling for Alab.
// All errors have stable, machine-readable codes, structured context, and proper wrapping.
package alerr

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
)

// Code represents a stable, machine-readable error code.
// Format: E{category}{number} where category is 1-5 and number is 001-999.
type Code string

// Error codes organized by category.
const (
	// Schema errors (E1xxx) - problems with schema definitions
	ErrSchemaInvalid     Code = "E1001" // Schema file is malformed or invalid
	ErrSchemaNotFound    Code = "E1002" // Referenced schema does not exist
	ErrSchemaDuplicate   Code = "E1003" // Schema with same name already exists
	ErrSchemaCircularRef Code = "E1004" // Circular reference detected in schemas

	// Validation errors (E2xxx) - problems with user input validation
	ErrInvalidIdentifier Code = "E2001" // Identifier does not match allowed pattern
	ErrInvalidSnakeCase  Code = "E2002" // Name must be snake_case
	ErrInvalidReference  Code = "E2003" // Reference to non-existent entity
	ErrReservedWord      Code = "E2004" // Name is a reserved SQL/JS keyword
	ErrInvalidType       Code = "E2005" // Type is not supported or forbidden
	ErrMissingNamespace  Code = "E2006" // Reference is missing namespace prefix
	ErrMissingLength     Code = "E2007" // String column requires length
	ErrTypeMismatchVal   Code = "E2008" // Default value type doesn't match column type

	// Migration errors (E3xxx) - problems during migration operations
	ErrMigrationFailed   Code = "E3001" // Migration execution failed
	ErrMigrationNotFound Code = "E3002" // Migration file not found
	ErrMigrationConflict Code = "E3003" // Migration conflicts with existing state
	ErrMigrationChecksum Code = "E3004" // Migration checksum does not match

	// SQL errors (E4xxx) - problems with database operations
	ErrSQLExecution   Code = "E4001" // SQL statement failed to execute
	ErrSQLConnection  Code = "E4002" // Database connection failed
	ErrSQLTransaction Code = "E4003" // Transaction operation failed

	// Runtime errors (E5xxx) - problems with JS execution
	ErrJSExecution        Code = "E5001" // JavaScript execution failed
	ErrJSTimeout          Code = "E5002" // JavaScript execution timed out
	ErrJSNonDeterministic Code = "E5003" // Non-deterministic operation detected

	// Introspection errors (E6xxx) - problems with database introspection
	ErrIntrospection    Code = "E6001" // Database introspection failed
	ErrTypeMismatch     Code = "E6002" // SQL type cannot be mapped to Alab type
	EUnsupportedDialect Code = "E6003" // Dialect not supported for operation

	// Git errors (E7xxx) - problems with git operations
	ENotGitRepo     Code = "E7001" // Not inside a git repository
	EGitOperation   Code = "E7002" // Git operation failed
	EGitUncommitted Code = "E7003" // Uncommitted migration files
	EGitNoRemote    Code = "E7004" // No remote configured

	// Cache errors (E8xxx) - problems with local cache
	ErrCacheInit    Code = "E8001" // Cache initialization failed
	ErrCacheRead    Code = "E8002" // Cache read failed
	ErrCacheWrite   Code = "E8003" // Cache write failed
	ErrCacheCorrupt Code = "E8004" // Cache is corrupted

	// Internal errors (E9xxx) - unexpected internal errors
	EInternalError Code = "E9001" // Internal error
)

// Error is the standard error type for Alab.
// It provides structured error information with codes, context, and wrapping support.
type Error struct {
	code    Code           // Machine-readable error code
	message string         // Human-readable error message
	context map[string]any // Structured context data
	cause   error          // Wrapped underlying error
	stack   string         // Stack trace for debugging
}

// Error returns the formatted error string.
// Format:
//
//	[E2002] column name must be snake_case
//	  table: auth.users
//	  got: userName
//	  want: user_name
func (e *Error) Error() string {
	var b strings.Builder

	// Write code and message
	b.WriteString(fmt.Sprintf("[%s] %s", e.code, e.message))

	// Write context in sorted order for deterministic output
	if len(e.context) > 0 {
		keys := make([]string, 0, len(e.context))
		for k := range e.context {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			b.WriteString(fmt.Sprintf("\n  %s: %v", k, e.context[k]))
		}
	}

	// Include cause if present
	if e.cause != nil {
		b.WriteString(fmt.Sprintf("\n  cause: %v", e.cause))
	}

	return b.String()
}

// Unwrap returns the underlying cause error for errors.Unwrap compatibility.
func (e *Error) Unwrap() error {
	return e.cause
}

// Is reports whether the target error matches this error.
// It matches if target is an *Error with the same code, or if it's the underlying cause.
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	var targetErr *Error
	if errors.As(target, &targetErr) {
		return e.code == targetErr.code
	}

	return false
}

// GetCode returns the error code.
func (e *Error) GetCode() Code {
	return e.code
}

// GetMessage returns the error message.
func (e *Error) GetMessage() string {
	return e.message
}

// SetMessage replaces the error message.
func (e *Error) SetMessage(msg string) {
	e.message = msg
}

// GetContext returns the error context map.
func (e *Error) GetContext() map[string]any {
	return e.context
}

// GetCause returns the underlying cause error.
func (e *Error) GetCause() error {
	return e.cause
}

// GetStack returns the stack trace.
func (e *Error) GetStack() string {
	return e.stack
}

// With adds a key-value pair to the error context.
// Returns the error for method chaining.
func (e *Error) With(key string, value any) *Error {
	if e.context == nil {
		e.context = make(map[string]any)
	}
	e.context[key] = value
	return e
}

// WithTable adds table context to the error.
// Format: "namespace.table" or just "table" if namespace is empty.
func (e *Error) WithTable(ns, table string) *Error {
	if ns != "" {
		return e.With("table", ns+"."+table)
	}
	return e.With("table", table)
}

// WithColumn adds column context to the error.
func (e *Error) WithColumn(name string) *Error {
	return e.With("column", name)
}

// WithSQL adds SQL statement context to the error.
func (e *Error) WithSQL(sql string) *Error {
	return e.With("sql", sql)
}

// WithFile adds file location context to the error.
func (e *Error) WithFile(path string, line int) *Error {
	e.With("file", path)
	if line > 0 {
		e.With("line", line)
	}
	return e
}

// WithLocation adds complete source location context (file, line, column).
func (e *Error) WithLocation(file string, line, col int) *Error {
	e.With("file", file)
	if line > 0 {
		e.With("line", line)
	}
	if col > 0 {
		e.With("column", col)
	}
	return e
}

// WithSource adds the source code line for display in error messages.
func (e *Error) WithSource(source string) *Error {
	return e.With("source", source)
}

// WithSpan adds the span (start, end columns) for highlighting in error messages.
func (e *Error) WithSpan(start, end int) *Error {
	e.With("span_start", start)
	e.With("span_end", end)
	return e
}

// WithLabel adds a label to display under the highlighted span.
func (e *Error) WithLabel(label string) *Error {
	return e.With("label", label)
}

// WithNote adds a note to the error (displayed as "note: ...").
func (e *Error) WithNote(note string) *Error {
	notes, _ := e.context["notes"].([]string)
	notes = append(notes, note)
	return e.With("notes", notes)
}

// WithHelp adds a help suggestion to the error (displayed as "help: ...").
func (e *Error) WithHelp(help string) *Error {
	helps, _ := e.context["helps"].([]string)
	helps = append(helps, help)
	return e.With("helps", helps)
}

// Location returns the file location if set.
func (e *Error) Location() (file string, line, col int, ok bool) {
	file, _ = e.context["file"].(string)
	line, _ = e.context["line"].(int)
	col, _ = e.context["column"].(int)
	ok = file != ""
	return
}

// Notes returns all notes attached to this error.
func (e *Error) Notes() []string {
	notes, _ := e.context["notes"].([]string)
	return notes
}

// Helps returns all help suggestions attached to this error.
func (e *Error) Helps() []string {
	helps, _ := e.context["helps"].([]string)
	return helps
}

// captureStack captures a stack trace for debugging.
func captureStack(skip int) string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])
	if n == 0 {
		return ""
	}

	var b strings.Builder
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// Skip runtime internals
		if strings.Contains(frame.File, "runtime/") {
			if !more {
				break
			}
			continue
		}
		b.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	return b.String()
}

// New creates a new Error with the given code and message.
func New(code Code, msg string) *Error {
	return &Error{
		code:    code,
		message: msg,
		context: make(map[string]any),
		stack:   captureStack(3),
	}
}

// Newf creates a new Error with the given code and formatted message.
func Newf(code Code, format string, args ...any) *Error {
	return &Error{
		code:    code,
		message: fmt.Sprintf(format, args...),
		context: make(map[string]any),
		stack:   captureStack(3),
	}
}

// Wrap creates a new Error that wraps an existing error.
func Wrap(code Code, err error, msg string) *Error {
	if err == nil {
		return New(code, msg)
	}
	return &Error{
		code:    code,
		message: msg,
		context: make(map[string]any),
		cause:   err,
		stack:   captureStack(3),
	}
}

// Wrapf creates a new Error that wraps an existing error with a formatted message.
func Wrapf(code Code, err error, format string, args ...any) *Error {
	return Wrap(code, err, fmt.Sprintf(format, args...))
}

// GetErrorCode extracts the error code from an error chain.
// Returns empty string if no code is found.
func GetErrorCode(err error) Code {
	if err == nil {
		return ""
	}

	var alerr *Error
	if errors.As(err, &alerr) {
		return alerr.code
	}

	return ""
}

// Is checks if an error has the specified code.
func Is(err error, code Code) bool {
	return GetErrorCode(err) == code
}

// HasCode checks if an error has any error code.
func HasCode(err error) bool {
	return GetErrorCode(err) != ""
}

// WrapSQL creates an ErrSQLExecution error with table context.
// Use for wrapping SQL errors with consistent formatting.
// Example: WrapSQL(err, "introspect columns", "auth_user")
func WrapSQL(err error, op string, table string) *Error {
	e := Wrap(ErrSQLExecution, err, "failed to "+op)
	if table != "" {
		e.WithTable("", table)
	}
	return e
}

// WrapSQLWithColumn creates an ErrSQLExecution error with table and column context.
// Use for wrapping SQL errors related to specific columns.
// Example: WrapSQLWithColumn(err, "scan column", "auth_user", "email")
func WrapSQLWithColumn(err error, op string, table, column string) *Error {
	e := Wrap(ErrSQLExecution, err, "failed to "+op)
	if table != "" {
		e.WithTable("", table)
	}
	if column != "" {
		e.WithColumn(column)
	}
	return e
}
