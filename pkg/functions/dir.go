package functions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
)

// entrypointLanguages maps allowed entrypoint filenames to dcg/runtime language ids.
var entrypointLanguages = map[string]string{
	"main.py": "python",
	"main.sh": "shell",
	"main.go": "go",
}

// LoadDir loads a function directory artifact (metadata.yaml + exactly one entrypoint).
func LoadDir(path string) (metadataYAML, entrypoint, source string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", "", types.Errorf(types.ErrInvalidArgument, "directory path is required")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", "", "", types.WrapError(types.ErrInvalidArgument, "read function directory", err)
	}
	metaName, entryName, err := classifyDirEntries(entries)
	if err != nil {
		return "", "", "", err
	}
	metaBytes, err := os.ReadFile(filepath.Join(path, metaName))
	if err != nil {
		return "", "", "", types.WrapError(types.ErrInvalidArgument, "read metadata.yaml", err)
	}
	srcBytes, err := os.ReadFile(filepath.Join(path, entryName))
	if err != nil {
		return "", "", "", types.WrapError(types.ErrInvalidArgument, "read entrypoint", err)
	}
	if len(srcBytes) > MaxSourceBytes {
		return "", "", "", types.Errorf(types.ErrInvalidArgument, "source exceeds %d bytes", MaxSourceBytes)
	}
	return string(metaBytes), entryName, string(srcBytes), nil
}

func classifyDirEntries(entries []os.DirEntry) (metaName, entryName string, err error) {
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		if e.IsDir() {
			return "", "", types.Errorf(types.ErrInvalidArgument, "unexpected directory %q in function dir", name)
		}
		metaName, entryName, err = classifyFile(name, metaName, entryName)
		if err != nil {
			return "", "", err
		}
	}
	if metaName == "" {
		return "", "", types.Errorf(types.ErrInvalidArgument, "metadata.yaml is required")
	}
	if entryName == "" {
		return "", "", types.Errorf(types.ErrInvalidArgument, "exactly one of main.py, main.sh, or main.go is required")
	}
	return metaName, entryName, nil
}

func classifyFile(name, metaName, entryName string) (string, string, error) {
	switch {
	case name == "metadata.yaml":
		if metaName != "" {
			return "", "", types.Errorf(types.ErrInvalidArgument, "duplicate metadata.yaml")
		}
		return name, entryName, nil
	case isAllowedEntrypoint(name):
		if entryName != "" {
			return "", "", types.Errorf(types.ErrInvalidArgument, "exactly one of main.py, main.sh, or main.go is required")
		}
		return metaName, name, nil
	default:
		return "", "", types.Errorf(types.ErrInvalidArgument, "unexpected file %q in function dir", name)
	}
}

func isAllowedEntrypoint(name string) bool {
	_, ok := entrypointLanguages[name]
	return ok
}

func languageFromEntrypoint(entrypoint string) (string, error) {
	lang, ok := entrypointLanguages[filepath.Base(entrypoint)]
	if !ok {
		return "", fmt.Errorf("unsupported entrypoint %q", entrypoint)
	}
	return lang, nil
}

func runtimeFromEntrypoint(entrypoint string) string {
	rt, err := languageFromEntrypoint(entrypoint)
	if err != nil {
		return ""
	}
	return rt
}
