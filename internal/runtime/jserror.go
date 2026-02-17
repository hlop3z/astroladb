// Package runtime provides JavaScript error parsing and rich error messages.
package runtime

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// JSErrorInfo contains extracted information from a JavaScript error.
type JSErrorInfo struct {
	Message   string
	Line      int
	Column    int
	Stack     string
	ErrorCode string // Structured error code (from __errorCode property)
	Help      string // Structured help text (from __errorHelp property)
}

// ParseJSError extracts detailed error information from a Goja error.
// Returns structured error info with line numbers and stack traces when available.
func ParseJSError(err error) *JSErrorInfo {
	if err == nil {
		return nil
	}

	info := &JSErrorInfo{
		Message: err.Error(),
	}

	// Handle syntax errors from the compiler
	if syntaxErr, ok := err.(*goja.CompilerSyntaxError); ok {
		// Use Error() for the message (will be cleaned by cleanCauseMessage)
		info.Message = syntaxErr.Error()
		// Get line and column from structured position info (more robust than parsing)
		if syntaxErr.File != nil {
			pos := syntaxErr.File.Position(syntaxErr.Offset)
			info.Line = pos.Line
			info.Column = pos.Column
		}
		return info
	}

	// Handle runtime exceptions
	if exception, ok := err.(*goja.Exception); ok {
		info.Message = exception.Value().String()
		info.Stack = exception.String()

		// Check if this is a structured error (has __errorCode property)
		if obj, ok := exception.Value().(*goja.Object); ok {
			if codeVal := obj.Get("__errorCode"); codeVal != nil && !goja.IsUndefined(codeVal) && !goja.IsNull(codeVal) {
				info.ErrorCode = codeVal.String()
			}
			if msgVal := obj.Get("__errorMessage"); msgVal != nil && !goja.IsUndefined(msgVal) && !goja.IsNull(msgVal) {
				info.Message = msgVal.String()
			}
			if helpVal := obj.Get("__errorHelp"); helpVal != nil && !goja.IsUndefined(helpVal) && !goja.IsNull(helpVal) {
				info.Help = helpVal.String()
			}
		}

		// Use Goja's structured stack frames (no regex parsing needed).
		// Skip native Go frames (line=0) to find the first JS call site.
		if frames := exception.Stack(); len(frames) > 0 {
			for _, frame := range frames {
				pos := frame.Position()
				if pos.Line > 0 {
					info.Line = pos.Line
					info.Column = pos.Column
					break
				}
			}
		} else {
			// Fallback for syntax errors wrapped in Exception (have zero stack frames)
			// Parse Goja's structured error format: "SyntaxError: (file): Line X:Y ..."
			// This is a Goja-specific format, safer than arbitrary regex.
			parseGojaErrorMessage(info)
		}
		return info
	}

	// Handle interrupted errors (timeouts)
	if interrupted, ok := err.(*goja.InterruptedError); ok {
		info.Message = "execution interrupted: " + interrupted.String()
		return info
	}

	// Unknown error type - no position info available
	return info
}

// parseGojaErrorMessage parses Goja's structured error message format.
// This is ONLY used as a fallback when Exception has no stack frames (syntax errors).
// Format: "SyntaxError: (file): Line X:Y Unexpected..."
//
// Note: This is not arbitrary regex parsing - it's parsing Goja's documented
// error message format for syntax errors. Goja's CompilerSyntaxError messages
// always follow this structure when wrapped in an Exception.
func parseGojaErrorMessage(info *JSErrorInfo) {
	// Goja syntax errors in exceptions follow this exact format:
	// "SyntaxError: (file): Line 1:11 Unexpected token ;"
	// We parse only this specific Goja format, not arbitrary messages.

	msg := info.Message

	// Look for "Line X:Y" in the message (Goja's syntax error format)
	lineIdx := strings.Index(msg, "Line ")
	if lineIdx == -1 {
		return
	}

	// Extract the part after "Line "
	rest := msg[lineIdx+5:] // Skip "Line "

	// Find the colon separator
	colonIdx := strings.Index(rest, ":")
	if colonIdx == -1 {
		return
	}

	// Parse line number
	lineStr := rest[:colonIdx]
	if line, err := strconv.Atoi(lineStr); err == nil {
		info.Line = line
	}

	// Parse column number (after the colon)
	rest = rest[colonIdx+1:]
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx != -1 {
		colStr := rest[:spaceIdx]
		if col, err := strconv.Atoi(colStr); err == nil {
			info.Column = col
		}
	}
}

// GetSourceLine reads a specific line from a string of code.
//
// IMPORTANT: Line numbers are 1-indexed (first line is 1, not 0).
// This matches Goja's error reporting convention. Do NOT adjust line numbers
// before calling this function, as it will cause off-by-one errors in error messages.
//
// Example:
//
//	code := "line 1\nline 2\nline 3"
//	GetSourceLine(code, 1)  // Returns "line 1"
//	GetSourceLine(code, 2)  // Returns "line 2"
//	GetSourceLine(code, 3)  // Returns "line 3"
//
// See TestGetSourceLine in jserror_test.go for validation.
func GetSourceLine(code string, lineNum int) string {
	if lineNum <= 0 || code == "" {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(code))
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine == lineNum {
			return scanner.Text()
		}
	}
	return ""
}

// GetSourceLineFromFile reads a specific line from a file.
//
// IMPORTANT: Line numbers are 1-indexed (first line is 1, not 0).
// This matches Goja's error reporting convention. The implementation MUST maintain
// this convention to avoid off-by-one errors that cause error messages to display
// incorrect source lines.
//
// NOTE: This function has caused bugs in the past where error messages showed
// line N-1 instead of line N. The test suite includes a specific test case
// (goja_error_line_accuracy) that validates this behavior. If that test fails,
// error messages will show wrong source lines to users.
//
// See TestGetSourceLineFromFile in jserror_test.go for validation.
func GetSourceLineFromFile(path string, lineNum int) string {
	if lineNum <= 0 || path == "" {
		return ""
	}

	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine == lineNum {
			return scanner.Text()
		}
	}
	return ""
}

// ValidateSourceLineExtraction is a helper for debugging line extraction issues.
// Call this during development to verify that line numbers from Goja match
// the actual file content. Returns an error if the extraction doesn't match expected.
//
// Example usage in error handling:
//
//	if os.Getenv("ALAB_DEBUG_LINES") == "1" {
//	    if err := ValidateSourceLineExtraction(filePath, gojaLine, expectedContent); err != nil {
//	        log.Printf("WARNING: %v", err)
//	    }
//	}
func ValidateSourceLineExtraction(filePath string, lineNum int, expectedContent string) error {
	actual := GetSourceLineFromFile(filePath, lineNum)
	if actual != expectedContent {
		return fmt.Errorf("line extraction mismatch at %s:%d\n  got:  %q\n  want: %q\n  This indicates an off-by-one error in line number handling",
			filePath, lineNum, actual, expectedContent)
	}
	return nil
}

// skipString advances past a string literal starting at position i (the opening quote).
// Returns the index of the closing quote. The caller's loop i++ will move past it.
func skipString(source string, i int) int {
	quote := source[i]
	i++
	for i < len(source) && source[i] != quote {
		if source[i] == '\\' {
			i++
		}
		i++
	}
	return i
}

// computeSpanEnd determines the end column (1-indexed, inclusive) for error highlighting.
// When col points to an opening delimiter, finds the matching close with nesting
// and string-literal awareness. Otherwise, scans forward to the next boundary.
func computeSpanEnd(source string, col int) int {
	idx := col - 1 // Convert to 0-indexed
	if idx < 0 || idx >= len(source) {
		return col
	}

	ch := source[idx]

	// Case 1: Opening delimiter — find matching close
	if ch == '(' || ch == '[' || ch == '{' {
		depth := 1
		for i := idx + 1; i < len(source); i++ {
			c := source[i]
			if c == '"' || c == '\'' || c == '`' {
				i = skipString(source, i)
				continue
			}
			switch c {
			case '(', '[', '{':
				depth++
			case ')', ']', '}':
				depth--
				if depth == 0 {
					return i + 1 // 1-indexed inclusive
				}
			}
		}
		// Unmatched — fall through to token scan
	}

	// Case 2: Scan forward to next boundary, respecting strings and nesting
	depth := 0
	for i := idx + 1; i < len(source); i++ {
		c := source[i]
		if c == '"' || c == '\'' || c == '`' {
			i = skipString(source, i)
			continue
		}
		switch c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			} else {
				return i // Stop before unmatched close (0-indexed = 1-indexed of prev char)
			}
		}
		if depth == 0 && (c == ' ' || c == '\t' || c == ',' || c == ';') {
			return i // 0-indexed boundary pos = 1-indexed last included char
		}
	}
	return len(source)
}

// wrapJSError creates a rich error with source location and context.
func wrapJSError(err error, code alerr.Code, message string, ctx *ErrorContext) *alerr.Error {
	jsErr := ParseJSError(err)
	if jsErr == nil {
		return alerr.Wrap(code, err, message)
	}

	// Create base error
	alErr := alerr.Wrap(code, err, message)

	// Add file location if available
	if ctx != nil && ctx.FilePath != "" {
		alErr.WithLocation(ctx.FilePath, jsErr.Line, jsErr.Column)
	} else if jsErr.Line > 0 {
		alErr.With("line", jsErr.Line)
		if jsErr.Column > 0 {
			alErr.With("column", jsErr.Column)
		}
	}

	// Try to get source line for context
	if jsErr.Line > 0 {
		var sourceLine string
		if ctx != nil {
			if ctx.FilePath != "" {
				sourceLine = GetSourceLineFromFile(ctx.FilePath, jsErr.Line)
			} else if ctx.Code != "" {
				sourceLine = GetSourceLine(ctx.Code, jsErr.Line)
			}
		}

		if sourceLine != "" {
			alErr.WithSource(sourceLine)
			// Add span for the error position if we have a column
			if jsErr.Column > 0 {
				end := computeSpanEnd(sourceLine, jsErr.Column)
				alErr.WithSpan(jsErr.Column, end)
			}
		}
	}

	// Add helpful context based on the error message
	addJSErrorHelp(alErr, jsErr.Message)

	return alErr
}

// restrictedGlobalHints maps restricted JS globals to helpful DSL alternatives.
// When a user tries to use Date, Math, etc. in a schema or migration, they get
// a clear message suggesting the right DSL approach instead.
var restrictedGlobalHints = map[string]string{
	"date":     "try `col.datetime()` for timestamps or `sql('NOW()')` for defaults",
	"math":     "try `sql()` or `fn.*()` for computed values",
	"json":     "try `col.json()` for JSON columns",
	"map":      "try plain objects `{}` instead of `Map`",
	"set":      "try arrays `[]` instead of `Set`",
	"parseint": "use integer literals directly",
}

// addJSErrorHelp adds contextual help based on the error message.
func addJSErrorHelp(err *alerr.Error, message string) {
	// Skip generic help if message is a structured error (starts with [XXX-NNN])
	if len(message) > 9 && message[0] == '[' && strings.Contains(message[:10], "-") {
		return
	}

	msg := strings.ToLower(message)

	// Check for restricted global access first (most actionable hints)
	for global, hint := range restrictedGlobalHints {
		if strings.Contains(msg, global) {
			err.WithNote("schemas and migrations are declarative — JS globals like Date, Math, and JSON are not available")
			err.WithHelp(hint)
			return
		}
	}

	switch {
	case strings.Contains(msg, "undefined"):
		err.WithNote("a variable or function was not found in scope")
		if strings.Contains(msg, "col") {
			err.WithHelp("ensure `col` is available in your schema file")
		}
	case strings.Contains(msg, "is not a function"):
		err.WithNote("attempted to call something that is not a function")
		err.WithHelp("check the method name and ensure it exists on the object")
	case strings.Contains(msg, "syntax"):
		err.WithNote("check for missing brackets, quotes, or commas")
	case strings.Contains(msg, "unexpected token"):
		err.WithNote("unexpected character in code")
		err.WithHelp("check for typos or missing punctuation")
	case strings.Contains(msg, "reference"):
		err.WithNote("reference to undefined variable")
	case strings.Contains(msg, "type"):
		err.WithNote("type mismatch in code")
	}
}
