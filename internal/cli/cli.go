// Package cli provides Cargo/rustc-style CLI output formatting for Alab.
// It handles colored output, error formatting, progress indicators, and
// structured output for CI/CD pipelines.
package cli

import (
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// OutputMode determines how output is formatted.
type OutputMode int

const (
	// ModeTTY enables rich colored output for interactive terminals.
	ModeTTY OutputMode = iota
	// ModePlain outputs plain text without colors (for pipes/CI).
	ModePlain
	// ModeJSON outputs structured JSON for programmatic consumption.
	ModeJSON
)

// Config holds CLI output configuration.
// Configuration is auto-detected; users don't configure this directly.
type Config struct {
	Mode   OutputMode
	Width  int
	Writer io.Writer
}

// DefaultConfig returns the auto-detected configuration.
// Rules:
//   - If stdout is TTY and NO_COLOR not set -> ModeTTY
//   - If stdout is not TTY or NO_COLOR set -> ModePlain
//   - Use ModeJSON explicitly via NewConfigWithMode
func DefaultConfig() *Config {
	mode := ModePlain
	width := 80

	// Check if stdout is a TTY
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		mode = ModeTTY
	}

	// Respect NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		mode = ModePlain
	}

	// Also respect TERM=dumb
	if os.Getenv("TERM") == "dumb" {
		mode = ModePlain
	}

	// Try to detect terminal width
	if w := lipgloss.Width(""); w > 0 {
		width = w
	}

	return &Config{
		Mode:   mode,
		Width:  width,
		Writer: os.Stdout,
	}
}

// NewConfigWithMode creates a config with a specific output mode.
// Used for --json flag or testing.
func NewConfigWithMode(mode OutputMode) *Config {
	cfg := DefaultConfig()
	cfg.Mode = mode
	return cfg
}

// IsTTY returns true if running in interactive terminal mode.
func (c *Config) IsTTY() bool {
	return c.Mode == ModeTTY
}

// IsPlain returns true if running in plain text mode.
func (c *Config) IsPlain() bool {
	return c.Mode == ModePlain
}

// IsJSON returns true if running in JSON output mode.
func (c *Config) IsJSON() bool {
	return c.Mode == ModeJSON
}

// Global default config, initialized lazily.
var defaultCfg *Config

// Default returns the global default configuration.
func Default() *Config {
	if defaultCfg == nil {
		defaultCfg = DefaultConfig()
	}
	return defaultCfg
}

// SetDefault sets the global default configuration.
// Used for testing or when --json flag is passed.
func SetDefault(cfg *Config) {
	defaultCfg = cfg
}

// EnableColors returns true if colors should be used.
func EnableColors() bool {
	return Default().IsTTY()
}
