package ui

import "fmt"

// ShowSuccess displays a success message box.
func ShowSuccess(title, content string) {
	fmt.Println(RenderSuccessPanel(title, content))
}

// ShowError displays an error message box.
func ShowError(title, content string) {
	fmt.Println(RenderErrorPanel(title, content))
}

// ShowWarning displays a warning message box.
func ShowWarning(title, content string) {
	fmt.Println(RenderWarningPanel(title, content))
}

// ShowInfo displays an info message box.
func ShowInfo(title, content string) {
	fmt.Println(RenderInfoPanel(title, content))
}
