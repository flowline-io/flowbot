package coding

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// WriteFileTool writes or overwrites a file in the workspace.
type WriteFileTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (WriteFileTool) Name() string { return "write_file" }

// Description explains the tool to the model.
func (WriteFileTool) Description() string {
	return "Writes text content to a file in the workspace, creating parent directories when needed"
}

// Parameters returns the JSON schema for tool arguments.
func (WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the file within the workspace",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Text content to write",
			},
		},
		"required": []string{"path", "content"},
	}
}

// Execute writes the requested file.
func (t WriteFileTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	path := fmt.Sprint(args["path"])
	content := fmt.Sprint(args["content"])

	resolvedResult := t.Workspace.ResolvePath(path)
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}
	resolved := resolvedResult.Value()
	execEnv := t.executionEnv()

	if mkdirResult := execEnv.MkdirAll(ctx, filepath.Dir(resolved), 0o755); !mkdirResult.IsOk() {
		return toolError(id, t.Name(), fmt.Sprintf("mkdir: %s", env.FormatFileError(mkdirResult.ErrorValue()))), nil
	}
	if writeResult := execEnv.WriteFile(ctx, resolved, []byte(content), 0o644); !writeResult.IsOk() {
		return toolError(id, t.Name(), fmt.Sprintf("write file: %s", env.FormatFileError(writeResult.ErrorValue()))), nil
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: fmt.Sprintf("wrote %d bytes to %s", len(content), path)}},
	}, nil
}

func (t WriteFileTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}
