package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// Colorize wraps text with ANSI color codes for terminal output.
func Colorize(text string, color tcell.Color) string {
	// tcell color to ANSI color mapping (basic 16 colors)
	// tcell v2 uses ColorAqua for cyan and ColorFuchsia for magenta
	colorMap := map[tcell.Color]int{
		tcell.ColorBlack:   30,
		tcell.ColorRed:     31,
		tcell.ColorGreen:   32,
		tcell.ColorYellow:  33,
		tcell.ColorBlue:    34,
		tcell.ColorFuchsia: 35, // Magenta
		tcell.ColorAqua:    36, // Cyan
		tcell.ColorWhite:   37,
		tcell.ColorGray:    90,
	}

	if code, ok := colorMap[color]; ok {
		return fmt.Sprintf("\033[%dm%s\033[0m", code, text)
	}
	return text
}

// Themed color functions - use theme colors
func Success(text string) string  { return colorWithIcon("✓", text, Theme.Success) }
func Error(text string) string    { return colorWithIcon("✗", text, Theme.Error) }
func Warning(text string) string  { return colorWithIcon("⚠", text, Theme.Warning) }
func Info(text string) string     { return colorWithIcon("ℹ", text, Theme.Info) }
func Primary(text string) string  { return Colorize(text, Theme.Primary) }
func Dim(text string) string      { return Colorize(text, Theme.TextDim) }

// Basic color functions - direct color mapping
func Green(text string) string  { return Colorize(text, Theme.Success) }
func Red(text string) string    { return Colorize(text, Theme.Error) }
func Yellow(text string) string { return Colorize(text, Theme.Warning) }
func Blue(text string) string   { return Colorize(text, Theme.Primary) }
func Cyan(text string) string   { return Colorize(text, Theme.Info) }

// Style functions
func Bold(text string) string   { return fmt.Sprintf("\033[1m%s\033[0m", text) }
func Header(text string, colorFuncs ...func(string) string) string {
	formatted := Bold(text)
	if len(colorFuncs) > 0 && colorFuncs[0] != nil {
		formatted = colorFuncs[0](formatted)
	}
	return formatted
}

// Aliases for compatibility
func Muted(text string) string     { return Dim(text) }
func Failed(text string) string    { return Error(text) }
func Done(message string) string   { return Success(message) }
func FilePath(path string) string  { return Primary(path) }
func FormatError(err error) string { return Error(err.Error()) }

// colorWithIcon is a helper that adds an icon before colored text.
func colorWithIcon(icon, text string, color tcell.Color) string {
	return Colorize(fmt.Sprintf("%s %s", icon, text), color)
}
