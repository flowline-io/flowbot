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
			return formatDelegationLine(subagent, ev.Stdout != "")
		}
	}
	if ev.Stdout != "" {
		return fmt.Sprintf("✓ %s: %s", ev.Name, ev.Stdout)
	}
	return fmt.Sprintf("⚙ Running tool: %s...", ev.Name)
}

func formatDelegationLine(subagent string, done bool) string {
	if done {
		return fmt.Sprintf("✓ Subagent %s finished", subagent)
	}
	return fmt.Sprintf("⤷ Delegating to subagent: %s...", subagent)
}

func formatSubagentToolLine(ev client.ChatStreamEvent) string {
	tool := strings.TrimSpace(ev.Name)
	if ev.Stdout != "" {
		if tool != "" {
			return fmt.Sprintf("  ↳ ✓ %s: %s", tool, ev.Stdout)
		}
		return fmt.Sprintf("  ↳ ✓ %s: %s", ev.Subagent, ev.Stdout)
	}
	if tool != "" && tool != taskToolName {
		return fmt.Sprintf("  ↳ ⚙ Running tool: %s...", tool)
	}
	return formatDelegationLine(ev.Subagent, false)
}

// FormatToolLine renders a styled tool status line for the transcript.
func FormatToolLine(ev client.ChatStreamEvent, styles *Styles) string {
	line := formatToolEventLine(ev)
	if ev.Subagent != "" || strings.HasPrefix(line, "  ↳") {
		return styles.ToolSub.Render(line) + "\n"
	}
	return styles.ToolLine.Render(line) + "\n"
}

func isTaskDelegationName(name string) bool {
	return name == taskToolName || strings.HasPrefix(name, taskToolName+" (")
}
