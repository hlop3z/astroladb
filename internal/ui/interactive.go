package ui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Confirm prompts the user for yes/no confirmation using tview Modal.
// Returns true if user confirms, false otherwise.
func Confirm(message string, defaultYes bool) bool {
	// Check if running in non-interactive mode (no TTY)
	if !isTerminal() {
		return defaultYes
	}

	app := tview.NewApplication()
	result := defaultYes

	// Create modal with Yes/No buttons - transparent background
	modal := tview.NewModal().
		SetText(message).
		SetTextColor(Theme.Text).
		SetButtonBackgroundColor(Theme.Background).
		SetButtonTextColor(Theme.Accent).
		SetButtonActivatedStyle(tcell.StyleDefault.
			Foreground(Theme.Highlight).
			Background(Theme.Selection))

	modal.SetBackgroundColor(Theme.Background)

	if defaultYes {
		modal.AddButtons([]string{"Yes", "No"})
	} else {
		modal.AddButtons([]string{"No", "Yes"})
	}

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if defaultYes {
			result = buttonIndex == 0 // "Yes" is first
		} else {
			result = buttonIndex == 1 // "Yes" is second
		}
		app.Stop()
	})

	// Style the modal - no background change, text highlighting only
	modal.SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelConfirm).
		SetTitleColor(Theme.Accent)

	if err := app.SetRoot(modal, true).EnableMouse(true).Run(); err != nil {
		return defaultYes
	}

	return result
}

// PromptConfig configures a text prompt.
type PromptConfig struct {
	Message      string              // Prompt message
	DefaultValue string              // Default value (shown if user presses enter)
	Required     bool                // Whether input is required
	Validate     func(string) error  // Optional validation function
}

// Prompt prompts the user for text input using tview InputField.
func Prompt(config PromptConfig) (string, error) {
	// Check if running in non-interactive mode (no TTY)
	if !isTerminal() {
		if config.DefaultValue != "" {
			return config.DefaultValue, nil
		}
		if config.Required {
			return "", fmt.Errorf("input required but running in non-interactive mode")
		}
		return "", nil
	}

	app := tview.NewApplication()
	var result string
	var err error

	// Create form with single input field - transparent background
	form := tview.NewForm()
	form.SetBackgroundColor(Theme.Background)

	// Add input field
	form.AddInputField(config.Message, config.DefaultValue, 40, nil, func(text string) {
		result = text
	})

	// Add submit button
	form.AddButton("OK", func() {
		// Validate if required
		if result == "" && config.Required {
			// Show error - keep form open
			return
		}
		// Validate with custom validator
		if config.Validate != nil {
			if validErr := config.Validate(result); validErr != nil {
				err = validErr
				return
			}
		}
		app.Stop()
	})

	form.AddButton("Cancel", func() {
		result = config.DefaultValue
		app.Stop()
	})

	// Style the form - no background change, text highlighting only
	form.SetButtonsAlign(tview.AlignCenter).
		SetFieldBackgroundColor(Theme.Background).
		SetFieldTextColor(Theme.Accent).
		SetButtonBackgroundColor(Theme.Background).
		SetButtonTextColor(Theme.Text).
		SetButtonActivatedStyle(tcell.StyleDefault.
			Foreground(Theme.Highlight).
			Background(Theme.Selection)).
		SetLabelColor(Theme.Text).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelInput).
		SetTitleColor(Theme.Accent)

	// Handle Escape key
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			result = config.DefaultValue
			app.Stop()
			return nil
		}
		return event
	})

	// Center the form - transparent background
	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(form, 7, 1, true).
		AddItem(nil, 0, 1, false)
	innerFlex.SetBackgroundColor(Theme.Background)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(innerFlex, 50, 1, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(Theme.Background)

	if runErr := app.SetRoot(flex, true).EnableMouse(true).Run(); runErr != nil {
		return config.DefaultValue, runErr
	}

	return result, err
}

// PromptSelect prompts the user to select from a list of options using tview List.
func PromptSelect(message string, options []string, defaultIndex int) (int, error) {
	// Check if running in non-interactive mode (no TTY)
	if !isTerminal() {
		return defaultIndex, nil
	}

	app := tview.NewApplication()
	result := defaultIndex

	// Create list with inverted selection (white bg, black text)
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(Theme.Selection).
		SetSelectedTextColor(Theme.Highlight).
		SetSelectedFocusOnly(true).
		SetMainTextColor(Theme.Text)

	list.SetBackgroundColor(Theme.Background)

	// Add options
	for i, option := range options {
		idx := i // Capture for closure
		list.AddItem(option, "", rune('1'+i), func() {
			result = idx
			app.Stop()
		})
	}

	// Set default selection
	list.SetCurrentItem(defaultIndex)

	// Handle Enter key
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			result = defaultIndex
			app.Stop()
			return nil
		}
		return event
	})

	// Create frame with title - transparent background
	frame := tview.NewFrame(list).
		SetBorders(1, 1, 1, 1, 1, 1).
		AddText(message, true, tview.AlignCenter, Theme.Text).
		AddText(HintsSelect, false, tview.AlignCenter, Theme.TextDim)

	frame.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelSelect).
		SetTitleColor(Theme.Accent)

	// Center the list - transparent background
	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(frame, len(options)+6, 1, true).
		AddItem(nil, 0, 1, false)
	innerFlex.SetBackgroundColor(Theme.Background)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(innerFlex, 50, 1, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(Theme.Background)

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		return defaultIndex, err
	}

	return result, nil
}

// isTerminal checks if stdout is connected to a terminal.
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
