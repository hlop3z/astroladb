package runtime

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// windowsReservedNames contains Windows reserved device names that cannot be used as filenames.
var windowsReservedNames = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// ValidateRenderPath validates a single output file path from a generator.
func ValidateRenderPath(p string) error {
	// Must be non-empty
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("render path must not be empty")
	}

	// No control characters or NUL bytes
	for _, r := range p {
		if unicode.IsControl(r) {
			return fmt.Errorf("render path contains control character: %q", p)
		}
	}

	// No backslashes (OS-specific separator)
	if strings.Contains(p, "\\") {
		return fmt.Errorf("render path must use forward slashes, got backslash in: %q", p)
	}

	// Must be relative (no leading /)
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("render path must be relative, got absolute: %q", p)
	}

	// Check for '..' anywhere in the path (before normalization)
	if strings.Contains(p, "..") {
		return fmt.Errorf("render path must not contain '..': %q", p)
	}

	// Normalize for further checks
	cleaned := path.Clean(p)

	// Check each segment
	segments := strings.Split(cleaned, "/")
	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// No dotfiles/hidden files
		if strings.HasPrefix(seg, ".") {
			return fmt.Errorf("render path must not contain hidden segments: %q", p)
		}

		// No Windows reserved names (check base name without extension)
		baseName := seg
		if idx := strings.Index(seg, "."); idx >= 0 {
			baseName = seg[:idx]
		}
		if windowsReservedNames[strings.ToLower(baseName)] {
			return fmt.Errorf("render path contains Windows reserved name %q: %q", baseName, p)
		}
	}

	return nil
}

// ValidateRenderOutput validates the complete render output from a generator.
// maxBytes=0 and maxFiles=0 mean unlimited.
func ValidateRenderOutput(files map[string]string, maxBytes int64, maxFiles int) error {
	// Check file count
	if maxFiles > 0 && len(files) > maxFiles {
		return fmt.Errorf("render output exceeds maximum file count: got %d, max %d", len(files), maxFiles)
	}

	// Track normalized paths for collision detection
	seen := make(map[string]string, len(files)) // normalized → original
	var totalSize int64

	for p, content := range files {
		// Validate each path
		if err := ValidateRenderPath(p); err != nil {
			return err
		}

		// Normalize for collision check
		normalized := path.Clean(p)
		if original, exists := seen[normalized]; exists {
			return fmt.Errorf("render output has duplicate paths: %q and %q both resolve to %q", original, p, normalized)
		}
		seen[normalized] = p

		// Accumulate size
		totalSize += int64(len(content))
	}

	// Check total size
	if maxBytes > 0 && totalSize > maxBytes {
		return fmt.Errorf("render output exceeds maximum size: got %d bytes, max %d bytes", totalSize, maxBytes)
	}

	return nil
}
