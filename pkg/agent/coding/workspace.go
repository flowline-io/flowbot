package coding

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/result"
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
func (w Workspace) ResolvePath(path string) result.Result[string, result.FileError] {
	if strings.TrimSpace(path) == "" {
		return result.Err[string, result.FileError](result.NewFileError("path_escape", "empty path", nil))
	}
	rootResult := w.absRoot()
	if !rootResult.IsOk() {
		return result.Err[string, result.FileError](rootResult.ErrorValue())
	}
	root := rootResult.Value()
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		clean = strings.TrimPrefix(clean, root)
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	resolved := filepath.Join(root, clean)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return result.Err[string, result.FileError](result.NewFileError("path_escape", "resolve path", err))
	}
	evaluated, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return result.Err[string, result.FileError](result.NewFileError("io_error", "eval symlinks", err))
		}
		evaluated = abs
	}
	if !isWithinRoot(root, evaluated) {
		return result.Err[string, result.FileError](result.NewFileError("path_escape", fmt.Sprintf("path %q escapes workspace", path), nil))
	}
	return result.Ok[string, result.FileError](evaluated)
}

func (w Workspace) absRoot() result.Result[string, result.FileError] {
	root := strings.TrimSpace(w.Root)
	if root == "" {
		return result.Err[string, result.FileError](result.NewFileError("path_escape", "root is required", nil))
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return result.Err[string, result.FileError](result.NewFileError("io_error", "abs root", err))
	}
	evaluated, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return result.Err[string, result.FileError](result.NewFileError("io_error", "eval root symlinks", err))
		}
		evaluated = abs
	}
	return result.Ok[string, result.FileError](evaluated)
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
