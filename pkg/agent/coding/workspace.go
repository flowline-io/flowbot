package coding

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	evaluated, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("coding workspace: eval symlinks: %w", err)
		}
		evaluated = abs
	}
	if !isWithinRoot(root, evaluated) {
		return "", fmt.Errorf("coding workspace: path %q escapes workspace", path)
	}
	return evaluated, nil
}

func (w Workspace) absRoot() (string, error) {
	root := strings.TrimSpace(w.Root)
	if root == "" {
		return "", fmt.Errorf("coding workspace: root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("coding workspace: abs root: %w", err)
	}
	evaluated, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("coding workspace: eval root symlinks: %w", err)
		}
		evaluated = abs
	}
	return evaluated, nil
}

func isWithinRoot(root, target string) bool {
	if runtime.GOOS == "windows" {
		root = strings.ToLower(filepath.Clean(root))
		target = strings.ToLower(filepath.Clean(target))
	} else {
		root = filepath.Clean(root)
		target = filepath.Clean(target)
	}
	if target == root {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
