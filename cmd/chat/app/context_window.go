package app

import (
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/client"
)

// ResolveContextWindow returns the effective input token budget for chat and tool models.
func ResolveContextWindow(chatModel, toolModel string) int {
	if toolModel != "" {
		return model.MaxContextWindow(chatModel, toolModel)
	}
	return model.ContextWindowFor(chatModel)
}

// ResolveContextWindowFromInfo reads the context window from agent splash metadata.
func ResolveContextWindowFromInfo(info *client.ChatAgentInfo) int {
	if info == nil {
		return model.DefaultContextWindow
	}
	return ResolveContextWindow(info.ChatModel, info.ToolModel)
}

func (m *Model) effectiveContextWindow() int {
	if m.status.ContextWindow > 0 {
		return m.status.ContextWindow
	}
	return ResolveContextWindowFromInfo(m.info)
}
