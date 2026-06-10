package coding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Workspace bounds file and shell operations to a single root directory.
type Workspace struct {
	// Root is the absolute workspace path.
	Root string
	// Timeout limits shell and code execution duration.
	Timeout time.Duration
	// MaxOutput truncates tool stdout/stderr beyond this byte count.
	MaxOutput int
}

// ResolvePath maps a relative or absolute path into the workspace root.
func (w Workspace) ResolvePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("coding workspace: empty path")
	}
	root, err := w.absRoot()
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		clean = strings.TrimPrefix(clean, root)
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	resolved := filepath.Join(root, clean)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("coding workspace: resolve path: %w", err)
	}
	if abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", fmt.Errorf("coding workspace: path %q escapes workspace", path)
	}
	return abs, nil
}

func (w Workspace) absRoot() (string, error) {
	root := w.Root
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("coding workspace: getwd: %w", err)
		}
		root = cwd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("coding workspace: abs root: %w", err)
	}
	return abs, nil
}

// TruncateOutput limits output size for model context safety.
func (w Workspace) TruncateOutput(text string) string {
	limit := w.MaxOutput
	if limit <= 0 {
		limit = 8192
	}
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "\n...(output truncated)"
}
