package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptConfig configures an interactive prompt
type PromptConfig struct {
	Message      string
	DefaultValue string
	Required     bool
	Validator    func(string) error
}

// Prompt displays a styled prompt and reads user input
func Prompt(cfg PromptConfig) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	// Format prompt message with icon and styling
	prompt := Primary("? ") + cfg.Message
	if cfg.DefaultValue != "" {
		prompt += Dim(fmt.Sprintf(" (%s)", cfg.DefaultValue))
	}
	prompt += Primary(": ")

	fmt.Print(prompt)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)

	// Use default if empty
	if input == "" && cfg.DefaultValue != "" {
		input = cfg.DefaultValue
	}

	// Validate required
	if cfg.Required && input == "" {
		return "", fmt.Errorf("input is required")
	}

	// Run custom validator
	if cfg.Validator != nil {
		if err := cfg.Validator(input); err != nil {
			return "", err
		}
	}

	return input, nil
}

// Confirm displays a yes/no confirmation dialog
func Confirm(message string, defaultYes bool) bool {
	var suffix string
	if defaultYes {
		suffix = Dim(" (Y/n)")
	} else {
		suffix = Dim(" (y/N)")
	}

	fmt.Print(Warning("? ") + message + suffix + Primary(": "))

	var input string
	fmt.Scanln(&input)

	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// Select displays a menu selection prompt
func Select(message string, options []string) (int, string, error) {
	fmt.Println(Primary("? ") + message)
	fmt.Println()

	for i, opt := range options {
		fmt.Printf("  %s %s\n", Dim(fmt.Sprintf("%d.", i+1)), opt)
	}

	fmt.Println()
	fmt.Print(Primary("Select: "))

	var choice int
	_, err := fmt.Scanln(&choice)
	if err != nil {
		return -1, "", err
	}

	if choice < 1 || choice > len(options) {
		return -1, "", fmt.Errorf("invalid selection: must be between 1 and %d", len(options))
	}

	return choice - 1, options[choice-1], nil
}

// ProgressReporter shows real-time progress for long-running operations
type ProgressReporter struct {
	Total   int
	Current int
	Message string
}

// NewProgressReporter creates a new progress reporter
func NewProgressReporter(total int) *ProgressReporter {
	return &ProgressReporter{
		Total:   total,
		Current: 0,
	}
}

// Update updates the progress and re-renders
func (p *ProgressReporter) Update(current int, message string) {
	p.Current = current
	p.Message = message
	p.render()
}

// render renders the progress bar
func (p *ProgressReporter) render() {
	if p.Total == 0 {
		return
	}

	percent := float64(p.Current) / float64(p.Total) * 100
	barWidth := 20
	filled := int(percent / 100 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Clear line and render
	fmt.Printf("\r  %s %s %s %.0f%% %s",
		Dim("["),
		Success(bar),
		Dim("]"),
		percent,
		p.Message,
	)
}

// Done marks the progress as complete
func (p *ProgressReporter) Done(message string) {
	// Clear progress line
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")

	// Print done message
	fmt.Printf("  %s %s\n", Done("✓"), message)
}

// Fail marks the progress as failed
func (p *ProgressReporter) Fail(message string) {
	// Clear progress line
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")

	// Print failure message
	fmt.Printf("  %s %s\n", Failed("✗"), message)
}
