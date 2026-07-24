package config

import "strings"

// CoreWorkspace returns core.workspace, falling back to chat_agent.workspace.
func CoreWorkspace() string {
	if ws := strings.TrimSpace(App.Core.Workspace); ws != "" {
		return ws
	}
	return strings.TrimSpace(App.ChatAgent.Workspace)
}
