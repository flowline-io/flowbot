// Package memory provides filesystem-backed persistent agent memory outside the chat workspace.
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const defaultScope = "default"

var (
	scopePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	filePattern  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*\.md$`)
)

// Store reads and writes scoped memory markdown files.
type Store interface {
	ListFiles(scope string) ([]string, error)
	Read(scope, file string) (string, error)
	Write(scope, file, content string) error
}

// FileStore stores memory files under a single root directory.
type FileStore struct {
	// Root is the absolute base directory for all scopes.
	Root string
	// DefaultFile is used when file is empty.
	DefaultFile string
	// MaxFileBytes caps write size.
	MaxFileBytes int
}

// NewFileStore creates a file-backed memory store.
func NewFileStore(root, defaultFile string, maxFileBytes int) (*FileStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("memory root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if defaultFile == "" {
		defaultFile = "MEMORIES.md"
	}
	if maxFileBytes <= 0 {
		maxFileBytes = 65536
	}
	return &FileStore{
		Root:         abs,
		DefaultFile:  defaultFile,
		MaxFileBytes: maxFileBytes,
	}, nil
}

// SanitizeScope maps a raw scope label to a safe directory name.
func SanitizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return defaultScope
	}
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		case r == ' ', r == '/':
			return '_'
		default:
			return '_'
		}
	}, scope)
	out = strings.Trim(out, "._-")
	if out == "" {
		return defaultScope
	}
	return out
}

// ListFiles returns markdown filenames in one scope directory.
func (s *FileStore) ListFiles(scope string) ([]string, error) {
	dir, err := s.scopeDir(scope)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filePattern.MatchString(name) {
			out = append(out, name)
		}
	}
	return out, nil
}

// Read returns the content of one memory file.
func (s *FileStore) Read(scope, file string) (string, error) {
	path, err := s.resolveFile(scope, file)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Write persists content to one memory file.
func (s *FileStore) Write(scope, file, content string) error {
	if len(content) > s.MaxFileBytes {
		return fmt.Errorf("content exceeds max size of %d bytes", s.MaxFileBytes)
	}
	path, err := s.resolveFile(scope, file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (s *FileStore) scopeDir(scope string) (string, error) {
	safe := SanitizeScope(scope)
	if !scopePattern.MatchString(safe) {
		return "", fmt.Errorf("invalid scope")
	}
	return filepath.Join(s.Root, safe), nil
}

func (s *FileStore) resolveFile(scope, file string) (string, error) {
	dir, err := s.scopeDir(scope)
	if err != nil {
		return "", err
	}
	name := normalizeFileName(file, s.DefaultFile)
	if !filePattern.MatchString(name) {
		return "", fmt.Errorf("invalid memory file name")
	}
	path := filepath.Join(dir, name)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	root := filepath.Clean(s.Root)
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes memory root")
	}
	return abs, nil
}

func normalizeFileName(file, defaultFile string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return defaultFile
	}
	if strings.Contains(file, "..") || strings.ContainsAny(file, `/\`) {
		return ""
	}
	return filepath.Base(file)
}
