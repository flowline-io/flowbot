package coding

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// ReadFileTool reads file contents from the workspace.
type ReadFileTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (ReadFileTool) Name() string { return "read_file" }

// Description explains the tool to the model.
func (ReadFileTool) Description() string {
	return "Reads a text file from the workspace; optional offset and limit return a line range"
}

// Parameters returns the JSON schema for tool arguments.
func (ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the file within the workspace",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Optional 1-based start line number",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Optional maximum number of lines to return from offset",
			},
		},
		"required": []string{"path"},
	}
}

// Execute reads the requested file.
func (t ReadFileTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	path := normalizeWorkspacePath(fmt.Sprint(args["path"]))
	if path == "" {
		return toolError(id, t.Name(), "path is required"), nil
	}
	resolvedResult := t.Workspace.ResolvePath(path)
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}

	readResult := t.executionEnv().ReadFile(ctx, resolvedResult.Value())
	if !readResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(readResult.ErrorValue())), nil
	}

	data := readResult.Value()
	if len(data) > MaxReadFileBytes {
		return tool.ErrorResult(id, t.Name(), "invalid_args",
			fmt.Sprintf("file exceeds %d bytes", MaxReadFileBytes),
			"file is too large to load; split it or use a smaller file"), nil
	}

	content := string(data)
	offset := intArg(args, "offset")
	limit := intArg(args, "limit")
	if offset > 0 || limit > 0 {
		var err error
		content, err = sliceFileLines(content, offset, limit)
		if err != nil {
			return toolError(id, t.Name(), err.Error()), nil
		}
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: t.Workspace.TruncateOutput(content)}},
	}, nil
}

func (t ReadFileTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

func normalizeWorkspacePath(path string) string {
	path = strings.TrimSpace(path)
	if after, ok := strings.CutPrefix(path, "file://"); ok {
		path = after
	}
	return path
}

func intArg(args map[string]any, key string) int {
	value, ok := args[key]
	if !ok || value == nil {
		return 0
	}
	switch n := value.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func sliceFileLines(content string, offset, limit int) (string, error) {
	if offset < 0 {
		return "", fmt.Errorf("offset must be >= 0")
	}
	if limit < 0 {
		return "", fmt.Errorf("limit must be >= 0")
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}
	start := 0
	if offset > 0 {
		start = offset - 1
	}
	if start > len(lines) {
		return "", nil
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return strings.Join(lines[start:end], "\n"), nil
}
