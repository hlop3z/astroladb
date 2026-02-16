package cli

import (
	"os"
	"testing"
)

func TestOutputMode(t *testing.T) {
	tests := []struct {
		name  string
		mode  OutputMode
		tty   bool
		plain bool
		json  bool
	}{
		{"ModeTTY", ModeTTY, true, false, false},
		{"ModePlain", ModePlain, false, true, false},
		{"ModeJSON", ModeJSON, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Mode: tt.mode}
			if got := cfg.IsTTY(); got != tt.tty {
				t.Errorf("IsTTY() = %v, want %v", got, tt.tty)
			}
			if got := cfg.IsPlain(); got != tt.plain {
				t.Errorf("IsPlain() = %v, want %v", got, tt.plain)
			}
			if got := cfg.IsJSON(); got != tt.json {
				t.Errorf("IsJSON() = %v, want %v", got, tt.json)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.Writer == nil {
		t.Error("Writer should not be nil")
	}
	if cfg.Width <= 0 {
		t.Error("Width should be positive")
	}
}

func TestNoColorEnv(t *testing.T) {
	// Save original value
	original := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", original)

	// Test with NO_COLOR set
	os.Setenv("NO_COLOR", "1")

	// Reset default config
	defaultCfg = nil

	cfg := DefaultConfig()
	if cfg.Mode != ModePlain {
		t.Errorf("With NO_COLOR set, Mode = %v, want ModePlain", cfg.Mode)
	}
}

func TestTermDumbEnv(t *testing.T) {
	// Save original values
	originalTerm := os.Getenv("TERM")
	originalNoColor := os.Getenv("NO_COLOR")
	defer func() {
		os.Setenv("TERM", originalTerm)
		os.Setenv("NO_COLOR", originalNoColor)
	}()

	// Test with TERM=dumb
	os.Setenv("TERM", "dumb")
	os.Unsetenv("NO_COLOR")

	// Reset default config
	defaultCfg = nil

	cfg := DefaultConfig()
	if cfg.Mode != ModePlain {
		t.Errorf("With TERM=dumb, Mode = %v, want ModePlain", cfg.Mode)
	}
}

func TestSetDefault(t *testing.T) {
	// Save original
	original := defaultCfg
	defer func() { defaultCfg = original }()

	// Set a custom config
	custom := &Config{Mode: ModeJSON, Width: 120}
	SetDefault(custom)

	got := Default()
	if got != custom {
		t.Error("SetDefault did not set the default config")
	}
	if got.Mode != ModeJSON {
		t.Errorf("Mode = %v, want ModeJSON", got.Mode)
	}
}

func TestEnableColors(t *testing.T) {
	// Save original
	original := defaultCfg
	defer func() { defaultCfg = original }()

	// Test with TTY mode
	SetDefault(&Config{Mode: ModeTTY})
	if !EnableColors() {
		t.Error("EnableColors() should return true in TTY mode")
	}

	// Test with Plain mode
	SetDefault(&Config{Mode: ModePlain})
	if EnableColors() {
		t.Error("EnableColors() should return false in Plain mode")
	}

	// Test with JSON mode
	SetDefault(&Config{Mode: ModeJSON})
	if EnableColors() {
		t.Error("EnableColors() should return false in JSON mode")
	}
}
