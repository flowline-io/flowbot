package partials

import (
	"fmt"
	"net/url"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func agentSkillRowID(item model.AgentSkill) string {
	return "agent-skill-" + url.PathEscape(item.Flag)
}

func agentSkillFormID(item model.AgentSkill, isNew bool) string {
	if isNew {
		return "agent-skill-form-new"
	}
	return "agent-skill-form-" + agentSkillRowID(item)
}

func agentSkillURL(item model.AgentSkill) string {
	return fmt.Sprintf("/service/web/agent-skills/%s", url.PathEscape(item.Flag))
}

func agentSkillEditURL(item model.AgentSkill) string {
	return agentSkillURL(item) + "/edit"
}

func agentSkillListURL() string {
	return "/service/web/agent-skills/list"
}

func agentSkillCancelURL() string {
	return agentSkillListURL()
}

func agentSkillDescriptionPreview(description string) string {
	if len(description) <= 60 {
		return description
	}
	return description[:57] + "..."
}
