package chatagent

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const subagentRunningToolPrefix = "running tool:"

// parseSubagentProgress extracts subagent metadata from task-tool progress updates.
// Supported formats: "[type] running tool: name" and "[type] detail".
func parseSubagentProgress(update string) (subagentType, toolName, detail string, ok bool) {
	update = strings.TrimSpace(update)
	if !strings.HasPrefix(update, "[") {
		return "", "", "", false
	}
	end := strings.Index(update, "]")
	if end <= 1 {
		return "", "", "", false
	}
	subagentType = strings.TrimSpace(update[1:end])
	rest := strings.TrimSpace(update[end+1:])
	if strings.HasPrefix(strings.ToLower(rest), subagentRunningToolPrefix) {
		toolName = strings.TrimSpace(rest[len(subagentRunningToolPrefix):])
		return subagentType, toolName, "", true
	}
	return subagentType, "", rest, true
}

// subagentToolStatusText renders the in-progress overlay while a subagent runs tools.
func subagentToolStatusText(subagent, tool, detail string) string {
	if detail != "" {
		if tool != "" {
			return fmt.Sprintf("%s › %s: %s", subagent, tool, detail)
		}
		return fmt.Sprintf("%s: %s", subagent, detail)
	}
	if tool != "" {
		return fmt.Sprintf("%s › Running tool: %s...", subagent, tool)
	}
	return fmt.Sprintf(subagentStatusTemplate, subagent)
}

// taskToolStreamEvent builds a tool SSE payload for the task delegation tool.
func taskToolStreamEvent(call msg.ToolCallPart, status, stdout string, durationMs int64) StreamEvent {
	subagent := subagentTypeFromArgs(call.Arguments)
	name := call.Name
	if subagent != "" {
		name = taskToolName
	}
	return StreamEvent{
		Type:       EventTypeTool,
		Name:       name,
		Subagent:   subagent,
		Status:     status,
		Stdout:     stdout,
		DurationMs: durationMs,
	}
}

// subagentInnerToolStreamEvent builds a tool SSE payload for a subagent's inner tool.
func subagentInnerToolStreamEvent(subagent, tool, status, stdout string, durationMs int64) StreamEvent {
	return StreamEvent{
		Type:       EventTypeTool,
		Name:       tool,
		Subagent:   subagent,
		Status:     status,
		Stdout:     stdout,
		DurationMs: durationMs,
	}
}
