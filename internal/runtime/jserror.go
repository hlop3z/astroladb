// Package runtime provides JavaScript error parsing and rich error messages.
package runtime

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"

	"github.com/hlop3z/astroladb/internal/alerr"
)

// JSErrorInfo contains extracted information from a JavaScript error.
type JSErrorInfo struct {
	Message string
	Line    int
	Column  int
	Stack   string
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
		parseStackTrace(info)
		// If stack trace didn't find line numbers, try extracting from message
		// (syntax errors embed line numbers in the message)
		if info.Line == 0 {
			parsePositionFromMessage(info)
		}
		return info
	}

	// Handle interrupted errors (timeouts)
	if interrupted, ok := err.(*goja.InterruptedError); ok {
		info.Message = "execution interrupted: " + interrupted.String()
		return info
	}

	// Try to extract position from error message
	parsePositionFromMessage(info)
	return info
}

// parsePositionFromMessage extracts line:column from error messages.
// Common formats:
//   - "at line 5:10"
//   - "Line 7:3 Unexpected..." (Goja syntax errors)
//   - "file:5:10: error message"
//   - "(line 5)"
func parsePositionFromMessage(info *JSErrorInfo) {
	// Try "line X:Y" or "Line X:Y" patterns (case-insensitive for Goja compatibility)
	lineColRe := regexp.MustCompile(`(?i)line\s+(\d+)(?:[,:\s]+(?:column\s+)?(\d+))?`)
	if matches := lineColRe.FindStringSubmatch(info.Message); len(matches) >= 2 {
		info.Line, _ = strconv.Atoi(matches[1])
		if len(matches) >= 3 && matches[2] != "" {
			info.Column, _ = strconv.Atoi(matches[2])
		}
		return
	}

	// Try "file:line:column:" pattern (common in syntax errors)
	fileLineColRe := regexp.MustCompile(`:(\d+):(\d+):`)
	if matches := fileLineColRe.FindStringSubmatch(info.Message); len(matches) >= 3 {
		info.Line, _ = strconv.Atoi(matches[1])
		info.Column, _ = strconv.Atoi(matches[2])
		return
	}

	// Try just ":line:" pattern
	lineOnlyRe := regexp.MustCompile(`:(\d+):`)
	if matches := lineOnlyRe.FindStringSubmatch(info.Message); len(matches) >= 2 {
		info.Line, _ = strconv.Atoi(matches[1])
	}
}

// parseStackTrace extracts line numbers from JavaScript stack traces.
// Goja stack format: "    at functionName (native)\n    at anonymous (eval:5:10)"
func parseStackTrace(info *JSErrorInfo) {
	if info.Stack == "" {
		return
	}

	// Look for "eval:line:column" or just line numbers in stack
	evalRe := regexp.MustCompile(`eval:(\d+):(\d+)`)
	if matches := evalRe.FindStringSubmatch(info.Stack); len(matches) >= 3 {
		info.Line, _ = strconv.Atoi(matches[1])
		info.Column, _ = strconv.Atoi(matches[2])
		return
	}

	// Try "at line X" pattern
	atLineRe := regexp.MustCompile(`at.*:(\d+):(\d+)`)
	if matches := atLineRe.FindStringSubmatch(info.Stack); len(matches) >= 3 {
		info.Line, _ = strconv.Atoi(matches[1])
		info.Column, _ = strconv.Atoi(matches[2])
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
				// Highlight from column to end of first token or reasonable length
				end := jsErr.Column + 10
				if end > len(sourceLine) {
					end = len(sourceLine)
				}
				// Don't include trailing punctuation (comma, semicolon, etc)
				if end > jsErr.Column && end <= len(sourceLine) {
					if ch := sourceLine[end-1]; ch == ',' || ch == ';' || ch == ')' {
						end--
					}
				}
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
	"date":     "use col.datetime() for timestamp columns or sql('NOW()') for default values",
	"math":     "schemas and migrations are declarative — use sql() or fn.*() for computed values",
	"json":     "use col.json() for JSON columns — JSON.parse/stringify are not available here",
	"map":      "use plain objects {} instead of Map in schemas and migrations",
	"set":      "use arrays [] instead of Set in schemas and migrations",
	"parseint": "use integer literals directly — parseInt is not available in schemas and migrations",
}

// addJSErrorHelp adds contextual help based on the error message.
func addJSErrorHelp(err *alerr.Error, message string) {
	// Skip generic help if message is a structured error (starts with [E####])
	if strings.HasPrefix(message, "[E") && len(message) > 6 && message[6] == ']' {
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
		err.WithNote("JavaScript 'undefined' error - a variable or function was not found")
		if strings.Contains(msg, "col") {
			err.WithHelp("ensure 'col' is available in your schema file")
		}
	case strings.Contains(msg, "is not a function"):
		err.WithNote("attempted to call something that is not a function")
		err.WithHelp("check the method name and ensure it exists on the object")
	case strings.Contains(msg, "syntax"):
		// No note for syntax errors - the cause message is self-explanatory
		break
	case strings.Contains(msg, "unexpected token"):
		err.WithNote("unexpected character in JavaScript code")
		err.WithHelp("check for typos or missing punctuation")
	case strings.Contains(msg, "reference"):
		err.WithNote("reference to undefined variable or missing import")
	case strings.Contains(msg, "type"):
		err.WithNote("type mismatch in JavaScript code")
	}
}
