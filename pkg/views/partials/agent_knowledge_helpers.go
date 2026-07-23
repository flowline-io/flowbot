package partials

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func agentKnowledgeRowID(item model.AgentKnowledge) string {
	return "agent-knowledge-" + strconv.FormatInt(item.ID, 10)
}

func agentKnowledgeFormID(item model.AgentKnowledge, isNew bool) string {
	if isNew {
		return "agent-knowledge-form-new"
	}
	return "agent-knowledge-form-" + agentKnowledgeRowID(item)
}

func agentKnowledgeURL(item model.AgentKnowledge) string {
	return fmt.Sprintf("/service/web/agent-knowledge/%d", item.ID)
}

func agentKnowledgeEditURL(item model.AgentKnowledge) string {
	return agentKnowledgeURL(item) + "/edit"
}

func agentKnowledgeListURL(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return "/service/web/agent-knowledge/list"
	}
	return "/service/web/agent-knowledge/list?q=" + url.QueryEscape(q)
}

func agentKnowledgeTagsDisplay(tags []string) string {
	if len(tags) == 0 {
		return "—"
	}
	return strings.Join(tags, ", ")
}

func agentKnowledgeSummaryPreview(summary string) string {
	if utf8.RuneCountInString(summary) <= 60 {
		return summary
	}
	runes := []rune(summary)
	return string(runes[:57]) + "..."
}

func agentKnowledgeTagsInput(tags []string) string {
	return strings.Join(tags, ", ")
}
