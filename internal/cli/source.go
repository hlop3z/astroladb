package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SourceSnippet represents a code snippet with context for error display.
type SourceSnippet struct {
	File             string
	StartLine        int
	Lines            []string
	SourceHighlights []SourceHighlight
}

// SourceHighlight represents a highlighted span in the source code.
type SourceHighlight struct {
	Line  int
	Start int // 1-indexed column
	End   int // 1-indexed column
	Label string
	Style SourceHighlightStyle
}

// SourceHighlightStyle determines how the highlight is rendered.
type SourceHighlightStyle int

const (
	StyleError SourceHighlightStyle = iota
	StyleWarning
	StyleNote
)

// NewSourceSnippet creates a SourceSnippet from a file.
// It reads the specified lines (context lines before and after the target).
func NewSourceSnippet(file string, targetLine, contextBefore, contextAfter int) (*SourceSnippet, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	startLine := targetLine - contextBefore
	if startLine < 1 {
		startLine = 1
	}
	endLine := targetLine + contextAfter

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &SourceSnippet{
		File:      file,
		StartLine: startLine,
		Lines:     lines,
	}, nil
}

// NewSourceSnippetFromString creates a SourceSnippet from a single line string.
func NewSourceSnippetFromString(file string, line int, source string) *SourceSnippet {
	return &SourceSnippet{
		File:      file,
		StartLine: line,
		Lines:     []string{source},
	}
}

// AddHighlight adds a highlight to the snippet.
func (s *SourceSnippet) AddHighlight(line, start, end int, label string, style SourceHighlightStyle) {
	s.SourceHighlights = append(s.SourceHighlights, SourceHighlight{
		Line:  line,
		Start: start,
		End:   end,
		Label: label,
		Style: style,
	})
}

// Render renders the source snippet with highlights.
func (s *SourceSnippet) Render() string {
	if len(s.Lines) == 0 {
		return ""
	}

	var b strings.Builder

	// Calculate the width needed for line numbers
	maxLineNum := s.StartLine + len(s.Lines) - 1
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	// Render each line
	for i, line := range s.Lines {
		lineNum := s.StartLine + i

		// Render line number and source
		numStr := fmt.Sprintf("%*d", lineNumWidth, lineNum)
		b.WriteString(LineNum(numStr))
		b.WriteString(" ")
		b.WriteString(Pipe())
		b.WriteString(" ")
		b.WriteString(Source(line))
		b.WriteString("\n")

		// Render highlights for this line
		for _, h := range s.SourceHighlights {
			if h.Line == lineNum {
				b.WriteString(s.renderSourceHighlight(lineNumWidth, h))
			}
		}
	}

	return b.String()
}

// renderSourceHighlight renders a single highlight line.
func (s *SourceSnippet) renderSourceHighlight(lineNumWidth int, h SourceHighlight) string {
	var b strings.Builder

	// Padding for line number column
	b.WriteString(strings.Repeat(" ", lineNumWidth))
	b.WriteString(" ")
	b.WriteString(Pipe())
	b.WriteString(" ")

	// Padding to start of highlight
	if h.Start > 1 {
		b.WriteString(strings.Repeat(" ", h.Start-1))
	}

	// Pointer characters
	pointerLen := h.End - h.Start
	if pointerLen < 1 {
		pointerLen = 1
	}
	pointer := strings.Repeat("^", pointerLen)

	switch h.Style {
	case StyleError:
		b.WriteString(Pointer(pointer))
	case StyleWarning:
		b.WriteString(Warning(pointer))
	case StyleNote:
		b.WriteString(Note(pointer))
	}

	// Label
	if h.Label != "" {
		b.WriteString(" ")
		b.WriteString(h.Label)
	}
	b.WriteString("\n")

	return b.String()
}

// RenderFileHeader renders the file location header (e.g., "--> file.js:15:3")
func RenderFileHeader(file string, line, col int) string {
	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(stylePipe.Render("-->"))
	b.WriteString(" ")

	loc := file
	if line > 0 {
		loc = fmt.Sprintf("%s:%d", file, line)
		if col > 0 {
			loc = fmt.Sprintf("%s:%d:%d", file, line, col)
		}
	}
	b.WriteString(FilePath(loc))
	b.WriteString("\n")
	return b.String()
}

// RenderSourceLine renders a single source line with optional highlight.
// This is a convenience function for simple cases.
func RenderSourceLine(line int, source string, highlightStart, highlightEnd int, label string) string {
	snippet := NewSourceSnippetFromString("", line, source)
	if highlightStart > 0 || highlightEnd > 0 {
		end := highlightEnd
		if end == 0 {
			end = highlightStart + 1
		}
		snippet.AddHighlight(line, highlightStart, end, label, StyleError)
	}
	return snippet.Render()
}
