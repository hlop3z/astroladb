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
	Accent     tcell.Color // For highlighting important text (IDs, keys)
	Highlight  tcell.Color // For selected/focused items

	// Background colors (use ColorDefault to preserve terminal bg)
	Background      tcell.Color
	BackgroundAlt   tcell.Color
	BackgroundDark  tcell.Color

	// UI element colors
	Border      tcell.Color
	BorderFocus tcell.Color
	Header      tcell.Color
	Selection   tcell.Color
}{
	// Primary colors (black and white theme, except titles)
	Primary:   tcell.ColorBlue,  // Keep blue for title bar
	Secondary: tcell.ColorWhite,
	Success:   tcell.ColorWhite,
	Warning:   tcell.ColorWhite,
	Error:     tcell.ColorWhite,
	Info:      tcell.ColorWhite,

	// Text colors
	Text:       tcell.ColorWhite,
	TextDim:    tcell.ColorGray,
	TextBright: tcell.ColorWhite,
	Accent:     tcell.ColorWhite, // White for IDs, revisions
	Highlight:  tcell.ColorWhite, // White text on navy background when selected

	// Background colors
	Background:     tcell.ColorBlack,
	BackgroundAlt:  tcell.ColorBlack,
	BackgroundDark: tcell.ColorBlack,

	// UI element colors
	Border:      tcell.ColorWhite,
	BorderFocus: tcell.ColorWhite,
	Header:      tcell.ColorWhite,
	Selection:   tcell.ColorNavy, // Subtle dark blue background for selection
}
