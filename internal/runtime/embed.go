package runtime

import "embed"

//go:embed js/*.js
var jsFiles embed.FS

// readJSFile reads a JavaScript file from the embedded filesystem.
func readJSFile(path string) (string, error) {
	content, err := jsFiles.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// mustReadJSFile reads a JS file and panics on error.
// Use this for files that must exist (embedded at compile time).
func mustReadJSFile(path string) string {
	content, err := readJSFile(path)
	if err != nil {
		panic("failed to read embedded JS file " + path + ": " + err.Error())
	}
	return content
}
