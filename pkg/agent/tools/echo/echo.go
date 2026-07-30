package echo

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// Tool echoes input text back to the model.
type Tool struct{}

// Name returns the tool identifier.
func (Tool) Name() string { return "echo" }

// Description explains the tool to the model.
func (Tool) Description() string {
	return "Echoes the input text back unchanged"
}

// Parameters returns the JSON schema for tool arguments.
func (Tool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to echo",
			},
		},
		"required": []string{"text"},
	}
}

// Execute returns the requested text.
func (Tool) Execute(_ context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	text := fmt.Sprint(args["text"])
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       "echo",
		Parts:      []msg.ContentPart{msg.TextPart{Text: fmt.Sprintf("echo: %s", text)}},
	}, nil
}
