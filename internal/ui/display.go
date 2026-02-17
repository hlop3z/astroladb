package ui

import "fmt"

// ShowSuccess displays a success message box.
func ShowSuccess(title, content string) {
	fmt.Println(RenderSuccessPanel(title, content))
}
