package ui

// View represents a renderable view with title and content.
// This is a thin wrapper around panel rendering for API compatibility.
type View struct {
	title      string
	content    string
	renderFunc func(string, string) string
}

// Render renders the view using its render function.
func (v *View) Render() string {
	return v.renderFunc(v.title, v.content)
}

// View constructors - DRY approach using same struct
func NewSuccessView(title, content string) *View {
	return &View{title, content, RenderSuccessPanel}
}

func NewErrorView(title, content string) *View {
	return &View{title, content, RenderErrorPanel}
}

func NewWarningView(title, content string) *View {
	return &View{title, content, RenderWarningPanel}
}

func NewInfoView(title, content string) *View {
	return &View{title, content, RenderInfoPanel}
}
