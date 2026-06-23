package chatagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// ReadSkillTool loads skill instructions from the database by name.
type ReadSkillTool struct{}

// Name returns the tool identifier.
func (ReadSkillTool) Name() string { return "read_skill" }

// Description explains the tool to the model.
func (ReadSkillTool) Description() string {
	return "Loads skill instructions from the database by name; optional path loads an auxiliary file from the skill directory"
}

// Parameters returns the JSON schema for tool arguments.
func (ReadSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Skill name from available_skills",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional relative path to an auxiliary skill file",
			},
		},
		"required": []string{"name"},
	}
}

// Execute returns the stored skill content.
func (ReadSkillTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	name := strings.TrimSpace(fmt.Sprint(args["name"]))
	if name == "" {
		return skillToolError(id, "skill name is required"), nil
	}
	if after, ok := strings.CutPrefix(name, skillLocationPrefix); ok {
		name = after
	}

	filePath := strings.TrimSpace(fmt.Sprint(args["path"]))
	var (
		content SkillContent
		err     error
	)
	if filePath != "" && filePath != "<nil>" {
		content, err = GetSkillFile(ctx, name, filePath)
	} else {
		content, err = GetSkillContent(ctx, name)
	}
	if err != nil {
		flog.Warn("[chat-agent] read_skill failed name=%s path=%s: %v", name, filePath, err)
		return skillToolError(id, fmt.Sprintf("read skill %q: %v", name, err)), nil
	}
	flog.Debug("[chat-agent] read_skill ok name=%s path=%s content_len=%d", name, filePath, len(content.Content))

	text := formatSkillContentText(content)
	if filePath != "" && filePath != "<nil>" {
		if content.BaseDir != "" {
			text = fmt.Sprintf("Skill base directory: %s\nPath: %s\n\n%s", content.BaseDir, filePath, content.Content)
		} else {
			text = fmt.Sprintf("Path: %s\n\n%s", filePath, content.Content)
		}
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       "read_skill",
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

func skillToolError(id, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       "read_skill",
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    true,
	}
}
