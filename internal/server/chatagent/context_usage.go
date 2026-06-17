package chatagent

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
)

// ContextCategoryInfo is one row in the context usage breakdown.
type ContextCategoryInfo struct {
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Tokens  int     `json:"tokens"`
	Percent float64 `json:"percent"`
}

// ContextSkillTokenInfo reports estimated prompt tokens for one skill entry.
type ContextSkillTokenInfo struct {
	Name   string `json:"name"`
	Tokens int    `json:"tokens"`
}

// ContextUsageReport is the full context budget snapshot for a chat session.
type ContextUsageReport struct {
	Model             string                  `json:"model"`
	ToolModel         string                  `json:"tool_model,omitempty"`
	ContextWindow     int                     `json:"context_window"`
	TotalTokens       int                     `json:"total_tokens"`
	TotalPercent      float64                 `json:"total_percent"`
	CompactionEnabled bool                    `json:"compaction_enabled"`
	Categories        []ContextCategoryInfo   `json:"categories"`
	Skills            []ContextSkillTokenInfo `json:"skills"`
}

// EstimateTextTokens conservatively estimates token count for raw text.
func EstimateTextTokens(text string) int {
	if text == "" {
		return 0
	}
	return ctxmgr.EstimateTokens(msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: text}}})
}

// BuildContextUsageReport assembles a context budget breakdown for the terminal client.
func BuildContextUsageReport(ctx context.Context, sessionID string) (ContextUsageReport, error) {
	workspace, err := WorkspaceFromConfig()
	if err != nil {
		return ContextUsageReport{}, err
	}

	modelName := config.ChatAgentChatModel()
	toolModel := config.App.ChatAgent.ToolModel
	contextWindow := config.ChatAgentContextWindow()
	compaction := config.App.ChatAgent.Compaction.WithDefaults()

	skills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		return ContextUsageReport{}, err
	}

	fullPrompt := SystemPrompt(ctx, workspace)
	skillsSection := FormatSkillsForPrompt(skills)
	systemPromptTokens := EstimateTextTokens(fullPrompt)
	if skillsSection != "" {
		systemPromptTokens -= EstimateTextTokens(skillsSection)
		if systemPromptTokens < 0 {
			systemPromptTokens = 0
		}
	}
	skillsTokens := EstimateTextTokens(skillsSection)

	systemToolsTokens, err := estimateActiveToolTokens(workspace)
	if err != nil {
		return ContextUsageReport{}, fmt.Errorf("estimate tool tokens: %w", err)
	}

	messageTokens, err := estimateSessionMessageTokens(ctx, sessionID)
	if err != nil {
		return ContextUsageReport{}, err
	}

	totalUsed := systemPromptTokens + systemToolsTokens + skillsTokens + messageTokens
	autocompactTokens := 0
	if compaction.AutoEnabled() {
		autocompactTokens = compaction.ReservedTokens()
	}

	freeTokens := max(contextWindow-totalUsed-autocompactTokens, 0)

	percentOf := func(tokens int) float64 {
		if contextWindow <= 0 {
			return 0
		}
		return float64(tokens) / float64(contextWindow) * 100
	}

	categories := []ContextCategoryInfo{
		{ID: "system_prompt", Label: "System prompt", Tokens: systemPromptTokens, Percent: percentOf(systemPromptTokens)},
		{ID: "system_tools", Label: "System tools", Tokens: systemToolsTokens, Percent: percentOf(systemToolsTokens)},
		{ID: "skills", Label: "Skills", Tokens: skillsTokens, Percent: percentOf(skillsTokens)},
		{ID: "messages", Label: "Messages", Tokens: messageTokens, Percent: percentOf(messageTokens)},
		{ID: "free_space", Label: "Free space", Tokens: freeTokens, Percent: percentOf(freeTokens)},
	}
	if compaction.AutoEnabled() {
		categories = append(categories, ContextCategoryInfo{
			ID: "autocompact_buffer", Label: "Autocompact buffer", Tokens: autocompactTokens, Percent: percentOf(autocompactTokens),
		})
	}

	skillUsages := buildSkillTokenInfos(skills)
	totalPercent := percentOf(totalUsed)

	return ContextUsageReport{
		Model:             modelName,
		ToolModel:         toolModel,
		ContextWindow:     contextWindow,
		TotalTokens:       totalUsed,
		TotalPercent:      totalPercent,
		CompactionEnabled: compaction.AutoEnabled(),
		Categories:        categories,
		Skills:            skillUsages,
	}, nil
}

func estimateActiveToolTokens(workspace coding.Workspace) (int, error) {
	registry, err := NewRegistry(workspace, nil)
	if err != nil {
		return 0, err
	}
	tools := tool.BuildLLMTools(registry.ActiveTools())
	data, err := sonic.Marshal(tools)
	if err != nil {
		return 0, err
	}
	return EstimateTextTokens(string(data)), nil
}

func estimateSessionMessageTokens(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" || store.Database == nil {
		return 0, nil
	}
	storage := NewDBStorage(sessionID)
	branch, err := storage.GetBranch(ctx, "")
	if err != nil {
		return 0, nil
	}
	messages := session.BuildContext(branch).Messages
	return ctxmgr.EstimateContextTokens(messages).Tokens, nil
}

func buildSkillTokenInfos(skills []Skill) []ContextSkillTokenInfo {
	visible := make([]Skill, 0, len(skills))
	for _, skill := range skills {
		if !skill.DisableModelInvocation && skill.Description != "" {
			visible = append(visible, skill)
		}
	}
	result := make([]ContextSkillTokenInfo, 0, len(visible))
	for _, skill := range visible {
		block := fmt.Sprintf(
			"  <skill>\n    <name>%s</name>\n    <description>%s</description>\n    <location>%s</location>\n  </skill>",
			escapeXML(skill.Name),
			escapeXML(skill.Description),
			escapeXML(skill.Location),
		)
		result = append(result, ContextSkillTokenInfo{
			Name:   skill.Name,
			Tokens: EstimateTextTokens(block),
		})
	}
	slices.SortFunc(result, func(a, b ContextSkillTokenInfo) int {
		if a.Tokens != b.Tokens {
			return b.Tokens - a.Tokens
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return result
}
