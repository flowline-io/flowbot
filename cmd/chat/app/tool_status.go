package app

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/client"
)

const taskToolName = "task"

// formatToolEventLine renders a human-readable status line for a streaming tool event.
func formatToolEventLine(ev client.ChatStreamEvent) string {
	if ev.Subagent != "" {
		return formatSubagentToolLine(ev)
	}
	if isTaskDelegationName(ev.Name) {
		subagent := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(ev.Name, taskToolName+" ("), ")"))
		if subagent != "" && subagent != ev.Name {
			return fmt.Sprintf("Delegating to subagent: %s...", subagent)
		}
	}
	if ev.Stdout != "" {
		return ev.Name + ": " + ev.Stdout
	}
	return fmt.Sprintf("Running tool: %s...", ev.Name)
}

func formatSubagentToolLine(ev client.ChatStreamEvent) string {
	tool := strings.TrimSpace(ev.Name)
	if ev.Stdout != "" {
		if tool != "" {
			return fmt.Sprintf("  ↳ %s: %s", tool, ev.Stdout)
		}
		return fmt.Sprintf("  ↳ %s: %s", ev.Subagent, ev.Stdout)
	}
	if tool != "" && tool != taskToolName {
		return fmt.Sprintf("  ↳ Running tool: %s...", tool)
	}
	return fmt.Sprintf("Delegating to subagent: %s...", ev.Subagent)
}

func isTaskDelegationName(name string) bool {
	return name == taskToolName || strings.HasPrefix(name, taskToolName+" (")
}
