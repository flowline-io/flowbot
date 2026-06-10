package coding

import (
	"context"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// ReadFileTool reads file contents from the workspace.
type ReadFileTool struct {
	Workspace Workspace
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
func (t ReadFileTool) Execute(_ context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	path := fmt.Sprint(args["path"])
	resolved, err := t.Workspace.ResolvePath(path)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return toolError(id, t.Name(), fmt.Sprintf("read file: %v", err)), nil
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: t.Workspace.TruncateOutput(string(data))}},
	}, nil
}
