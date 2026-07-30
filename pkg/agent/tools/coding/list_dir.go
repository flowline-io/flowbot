package coding

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// ListDirTool lists directory entries inside the workspace.
type ListDirTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (ListDirTool) Name() string { return "list_dir" }

// Description explains the tool to the model.
func (ListDirTool) Description() string {
	return "Lists files and directories under a workspace path; directories end with /"
}

// Parameters returns the JSON schema for tool arguments.
func (ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative directory path within the workspace (default .)",
			},
			"recursive": map[string]any{
				"type":        "boolean",
				"description": "When true, list nested entries recursively",
			},
		},
	}
}

// Execute lists directory contents.
func (t ListDirTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	pathArg := normalizeWorkspacePath(fmt.Sprint(args["path"]))
	if pathArg == "" || pathArg == "<nil>" {
		pathArg = "."
	}
	recursive := boolArg(args, "recursive")

	resolvedResult := t.Workspace.ResolvePath(pathArg)
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}
	root := resolvedResult.Value()
	execEnv := t.executionEnv()

	var lines []string
	var truncated bool
	var walkErr error
	if recursive {
		lines, truncated, walkErr = listDirRecursive(ctx, execEnv, root, root, MaxListDirEntries)
	} else {
		lines, truncated, walkErr = listDirFlat(ctx, execEnv, root, MaxListDirEntries)
	}
	if walkErr != nil {
		return toolError(id, t.Name(), walkErr.Error()), nil
	}
	if len(lines) == 0 {
		return msg.ToolResultMessage{
			ToolCallID: id,
			Name:       t.Name(),
			Parts:      []msg.ContentPart{msg.TextPart{Text: "(empty)"}},
		}, nil
	}
	text := strings.Join(lines, "\n")
	if truncated {
		text += fmt.Sprintf("\n...(truncated to %d entries)", MaxListDirEntries)
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: t.Workspace.TruncateOutput(text)}},
	}, nil
}

func (t ListDirTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

func listDirFlat(ctx context.Context, execEnv env.ExecutionEnv, root string, maxEntries int) ([]string, bool, error) {
	entriesResult := execEnv.ReadDir(ctx, root)
	if !entriesResult.IsOk() {
		return nil, false, fmt.Errorf("%s", env.FormatFileError(entriesResult.ErrorValue()))
	}
	entries := entriesResult.Value()
	truncated := len(entries) > maxEntries
	if truncated {
		entries = entries[:maxEntries]
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name
		if entry.IsDir {
			name += "/"
		}
		lines = append(lines, name)
	}
	return lines, truncated, nil
}

func listDirRecursive(ctx context.Context, execEnv env.ExecutionEnv, absRoot, current string, maxEntries int) ([]string, bool, error) {
	var lines []string
	truncated := false
	var walk func(string) error
	walk = func(dir string) error {
		if len(lines) >= maxEntries {
			truncated = true
			return nil
		}
		entriesResult := execEnv.ReadDir(ctx, dir)
		if !entriesResult.IsOk() {
			return fmt.Errorf("%s", env.FormatFileError(entriesResult.ErrorValue()))
		}
		for _, entry := range entriesResult.Value() {
			if len(lines) >= maxEntries {
				truncated = true
				return nil
			}
			if entry.IsDir && ShouldSkipDir(entry.Name) {
				continue
			}
			abs := filepath.Join(dir, entry.Name)
			rel, err := filepath.Rel(absRoot, abs)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if entry.IsDir {
				lines = append(lines, rel+"/")
				if err := walk(abs); err != nil {
					return err
				}
				continue
			}
			lines = append(lines, rel)
		}
		return nil
	}
	if err := walk(current); err != nil {
		return nil, false, err
	}
	return lines, truncated, nil
}

func boolArg(args map[string]any, key string) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}
