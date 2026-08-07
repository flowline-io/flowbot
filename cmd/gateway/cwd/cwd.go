// Package cwd resolves and validates worker workspace paths against an allowlist.
package cwd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Resolve picks the effective workspace path and validates allowlist membership.
func Resolve(requested, defaultWorkspace string, allowlist []string) (string, error) {
	path := strings.TrimSpace(requested)
	if path == "" {
		path = defaultWorkspace
	}
	cleaned, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	for _, root := range allowlist {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(filepath.Clean(root))
		if err != nil {
			continue
		}
		if cleaned == absRoot || strings.HasPrefix(cleaned, absRoot+string(filepath.Separator)) {
			return cleaned, nil
		}
	}
	return "", fmt.Errorf("cwd %q is outside workspace allowlist", cleaned)
}
