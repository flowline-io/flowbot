package chatagent

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateKnowledgePath reports whether path is a valid knowledge document path.
// Paths must start with "/", end with ".md", forbid ".." and empty segments, and
// only allow letters, digits, '/', '_', '-', and '.'.
func ValidateKnowledgePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /")
	}
	if !strings.HasSuffix(path, ".md") {
		return fmt.Errorf("path must end with .md")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path must not contain parent segments")
	}
	parts := strings.Split(path, "/")
	// First part is empty because path starts with '/'.
	if len(parts) < 2 {
		return fmt.Errorf("path must not contain empty segments")
	}
	for i, part := range parts {
		if i == 0 {
			continue
		}
		if part == "" {
			return fmt.Errorf("path must not contain empty segments")
		}
		for _, r := range part {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
				continue
			}
			return fmt.Errorf("path contains invalid characters")
		}
	}
	return nil
}
