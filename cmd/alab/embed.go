package main

import "embed"

// templates contains all template files embedded at compile time.
// These files are extracted from Go constants to improve maintainability
// while preserving single-binary distribution.
//
//go:embed templates/*
var templates embed.FS

// readTemplate reads a template file from the embedded filesystem.
// Returns the content as a string and any read error.
func readTemplate(path string) (string, error) {
	content, err := templates.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// mustReadTemplate reads a template file and panics on error.
// Use this for files that must exist (embedded at compile time).
func mustReadTemplate(path string) string {
	content, err := readTemplate(path)
	if err != nil {
		panic("failed to read embedded template " + path + ": " + err.Error())
	}
	return content
}
