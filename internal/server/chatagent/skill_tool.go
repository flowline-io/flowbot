package chatagent

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// ReadSkillTool loads skill instructions from the database by name.
type ReadSkillTool struct {
	// allowed restricts skill names when non-empty (subagent allowlist).
	allowed []string
}

// NewReadSkillTool creates a read_skill tool optionally restricted to allowed skill names.
func NewReadSkillTool(allowed []string) ReadSkillTool {
	return ReadSkillTool{allowed: append([]string(nil), allowed...)}
}

// Name returns the tool identifier.
func (ReadSkillTool) Name() string { return "read_skill" }

// Description explains the tool to the model.
func (ReadSkillTool) Description() string {
	return "Loads skill instructions by name. Always pass name as a non-empty string matching <name> from available_skills (example: {\"name\":\"gitea\"}). Optional path loads an auxiliary file from the skill directory."
}

// Parameters returns the JSON schema for tool arguments.
func (ReadSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Exact skill <name> from available_skills (required, non-empty)",
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
func (t ReadSkillTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	name := normalizeReadSkillName(args)
	if name == "" {
		return skillToolError(id, "skill name is required"), nil
	}
	if !t.isSkillAllowed(name) {
		return skillToolError(id, fmt.Sprintf("skill %q is not available to this agent", name)), nil
	}

	filePath := skillArgString(args, "path")
	var (
		content SkillContent
		err     error
	)
	if filePath != "" {
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
	if filePath != "" {
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

// skillArgString reads a string tool arg, treating nil/"<nil>"/"null" as empty.
func skillArgString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	switch strings.ToLower(s) {
	case "", "<nil>", "null", "undefined":
		return ""
	default:
		return s
	}
}

// normalizeReadSkillName extracts and normalizes the skill name from tool args.
// Accepts optional leading "/" from composer slash chips and skill:// locations.
func normalizeReadSkillName(args map[string]any) string {
	name := skillArgString(args, "name")
	if name == "" {
		// Some models put the skill id in "skill" instead of "name".
		name = skillArgString(args, "skill")
	}
	if after, ok := strings.CutPrefix(name, skillLocationPrefix); ok {
		name = after
	}
	name = strings.TrimSpace(strings.TrimPrefix(name, "/"))
	return name
}

func (t ReadSkillTool) isSkillAllowed(name string) bool {
	if len(t.allowed) == 0 {
		return true
	}
	return slices.Contains(t.allowed, name)
}

func skillToolError(id, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       "read_skill",
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    true,
	}
}
