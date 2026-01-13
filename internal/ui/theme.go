package ui

import (
	"github.com/gdamore/tcell/v2"
)

// Theme defines the color scheme for the CLI.
var Theme = struct {
	// Primary colors
	Primary   tcell.Color
	Secondary tcell.Color
	Success   tcell.Color
	Warning   tcell.Color
	Error     tcell.Color
	Info      tcell.Color

	// Text colors
	Text       tcell.Color
	TextDim    tcell.Color
	TextBright tcell.Color

	// Background colors
	Background      tcell.Color
	BackgroundAlt   tcell.Color
	BackgroundDark  tcell.Color

	// UI element colors
	Border      tcell.Color
	BorderFocus tcell.Color
	Header      tcell.Color
	Selection   tcell.Color
}{
	// Primary colors
	Primary:   tcell.ColorBlue,
	Secondary: tcell.ColorAqua,  // tcell v2 uses ColorAqua for cyan
	Success:   tcell.ColorGreen,
	Warning:   tcell.ColorYellow,
	Error:     tcell.ColorRed,
	Info:      tcell.ColorAqua,   // tcell v2 uses ColorAqua for cyan

	// Text colors
	Text:       tcell.ColorWhite,
	TextDim:    tcell.ColorGray,
	TextBright: tcell.ColorWhite,

	// Background colors
	Background:     tcell.ColorBlack,
	BackgroundAlt:  tcell.ColorTeal,
	BackgroundDark: tcell.ColorBlack,

	// UI element colors
	Border:      tcell.ColorGray,
	BorderFocus: tcell.ColorBlue,
	Header:      tcell.ColorYellow,
	Selection:   tcell.ColorTeal,
}
