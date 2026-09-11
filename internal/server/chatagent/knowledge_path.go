package chatagent

import (
	"errors"
	"strings"
	"unicode"
)

// ValidateKnowledgePath reports whether path is a valid knowledge document path.
// Paths must start with "/", end with ".md", forbid ".." and empty segments, and
// only allow letters, digits, '/', '_', '-', and '.'.
func ValidateKnowledgePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("path is required")
	}
	if !strings.HasPrefix(path, "/") {
		return errors.New("path must start with /")
	}
	if !strings.HasSuffix(path, ".md") {
		return errors.New("path must end with .md")
	}
	if strings.Contains(path, "..") {
		return errors.New("path must not contain parent segments")
	}
	parts := strings.Split(path, "/")
	// First part is empty because path starts with '/'.
	if len(parts) < 2 {
		return errors.New("path must not contain empty segments")
	}
	for i, part := range parts {
		if i == 0 {
			continue
		}
		if part == "" {
			return errors.New("path must not contain empty segments")
		}
		for _, r := range part {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
				continue
			}
			return errors.New("path contains invalid characters")
		}
	}
	return nil
}
