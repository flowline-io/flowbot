package partials

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func agentSubagentRowID(item model.AgentSubagent) string {
	return "agent-subagent-" + url.PathEscape(item.Flag)
}

func agentSubagentFormID(item model.AgentSubagent, isNew bool) string {
	if isNew {
		return "agent-subagent-form-new"
	}
	return "agent-subagent-form-" + agentSubagentRowID(item)
}

func agentSubagentURL(item model.AgentSubagent) string {
	return fmt.Sprintf("/service/web/agent-subagents/%s", url.PathEscape(item.Flag))
}

func agentSubagentEditURL(item model.AgentSubagent) string {
	return agentSubagentURL(item) + "/edit"
}

func agentSubagentListURL() string {
	return "/service/web/agent-subagents/list"
}

func agentSubagentCancelURL() string {
	return agentSubagentListURL()
}

func agentSubagentDescriptionPreview(description string) string {
	if len(description) <= 60 {
		return description
	}
	return description[:57] + "..."
}

func agentSubagentModelLabel(modelName string) string {
	if strings.TrimSpace(modelName) == "" {
		return "(default)"
	}
	return modelName
}

func agentSubagentToolsValue(tools []string) string {
	return strings.Join(tools, ", ")
}
