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
	return "Reads a text file from the workspace and returns its contents"
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
		},
		"required": []string{"path"},
	}
}

// Execute reads the requested file.
func (t ReadFileTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	path := strings.TrimSpace(fmt.Sprint(args["path"]))
	if after, ok := strings.CutPrefix(path, "file://"); ok {
		path = after
	}
	resolvedResult := t.Workspace.ResolvePath(path)
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}

	readResult := t.executionEnv().ReadFile(ctx, resolvedResult.Value())
	if !readResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(readResult.ErrorValue())), nil
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: t.Workspace.TruncateOutput(string(readResult.Value()))}},
	}, nil
}

func (t ReadFileTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}
