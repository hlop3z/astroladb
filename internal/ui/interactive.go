package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user for yes/no confirmation.
// Returns true if user confirms (yes/y), false otherwise.
func Confirm(message string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	// Format prompt with default indicator
	var prompt string
	if defaultYes {
		prompt = fmt.Sprintf("%s %s ", message, Dim("[Y/n]"))
	} else {
		prompt = fmt.Sprintf("%s %s ", message, Dim("[y/N]"))
	}

	fmt.Print(prompt)

	// Read user input
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	// Trim whitespace
	input = strings.TrimSpace(strings.ToLower(input))

	// Empty input uses default
	if input == "" {
		return defaultYes
	}

	// Check for yes/no
	return input == "y" || input == "yes"
}

// PromptConfig configures a text prompt.
type PromptConfig struct {
	Message      string // Prompt message
	DefaultValue string // Default value (shown if user presses enter)
	Required     bool   // Whether input is required
	Validate     func(string) error // Optional validation function
}

// Prompt prompts the user for text input with configuration.
func Prompt(config PromptConfig) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	// Format prompt with default value if provided
	var prompt string
	if config.DefaultValue != "" {
		prompt = fmt.Sprintf("%s %s: ", config.Message, Dim(fmt.Sprintf("[%s]", config.DefaultValue)))
	} else {
		prompt = fmt.Sprintf("%s: ", config.Message)
	}

	for {
		fmt.Print(prompt)

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		// Trim whitespace
		input = strings.TrimSpace(input)

		// Use default if empty
		if input == "" && config.DefaultValue != "" {
			input = config.DefaultValue
		}

		// Check if required
		if input == "" && config.Required {
			fmt.Println(Error("Input is required"))
			continue
		}

		// Validate if validator provided
		if config.Validate != nil {
			if err := config.Validate(input); err != nil {
				fmt.Println(Error(err.Error()))
				continue
			}
		}

		return input, nil
	}
}

// PromptSelect prompts the user to select from a list of options.
func PromptSelect(message string, options []string, defaultIndex int) (int, error) {
	reader := bufio.NewReader(os.Stdin)

	// Show message
	fmt.Println(message)
	fmt.Println()

	// Show options
	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "▸"
		}
		fmt.Printf("  %s %d) %s\n", Dim(marker), i+1, option)
	}
	fmt.Println()

	// Prompt for selection
	prompt := fmt.Sprintf("Select option %s: ", Dim(fmt.Sprintf("[1-%d]", len(options))))
	fmt.Print(prompt)

	// Read input
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultIndex, err
	}

	input = strings.TrimSpace(input)

	// Use default if empty
	if input == "" {
		return defaultIndex, nil
	}

	// Parse selection
	var selection int
	_, err = fmt.Sscanf(input, "%d", &selection)
	if err != nil || selection < 1 || selection > len(options) {
		return defaultIndex, fmt.Errorf("invalid selection")
	}

	return selection - 1, nil
}
