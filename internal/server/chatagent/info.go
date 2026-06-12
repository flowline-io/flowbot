package chatagent

import (
	"context"
	"slices"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/version"
)

// ToolInfo describes one active chat agent tool for the splash panel.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SkillInfo describes one enabled agent skill for the splash panel.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AgentInfo is startup metadata for the Chat Agent HTTP client.
type AgentInfo struct {
	Version    string      `json:"version"`
	ChatModel  string      `json:"chat_model"`
	ToolModel  string      `json:"tool_model"`
	Provider   string      `json:"provider"`
	Workspace  string      `json:"workspace"`
	Tools      []ToolInfo  `json:"tools"`
	Skills     []SkillInfo `json:"skills"`
	ToolCount  int         `json:"tool_count"`
	SkillCount int         `json:"skill_count"`
}

// BuildAgentInfo assembles splash metadata from config and storage.
func BuildAgentInfo(ctx context.Context) (AgentInfo, error) {
	snippets := DefaultToolSnippets()
	toolNames := ActiveToolNames()
	tools := make([]ToolInfo, 0, len(toolNames))
	for _, name := range toolNames {
		tools = append(tools, ToolInfo{Name: name, Description: snippets[name]})
	}

	skills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		return AgentInfo{}, err
	}
	skillInfos := make([]SkillInfo, 0, len(skills))
	for _, skill := range skills {
		skillInfos = append(skillInfos, SkillInfo{Name: skill.Name, Description: skill.Description})
	}

	chatModel := config.ChatAgentChatModel()
	toolModel := config.App.ChatAgent.ToolModel
	provider := resolveModelProvider(chatModel)

	return AgentInfo{
		Version:    version.Buildtags,
		ChatModel:  chatModel,
		ToolModel:  toolModel,
		Provider:   provider,
		Workspace:  config.App.ChatAgent.Workspace,
		Tools:      tools,
		Skills:     skillInfos,
		ToolCount:  len(tools),
		SkillCount: len(skillInfos),
	}, nil
}

func resolveModelProvider(modelName string) string {
	for _, m := range config.App.Models {
		if slices.Contains(m.ModelNames, modelName) {
			return m.Provider
		}
	}
	return ""
}
