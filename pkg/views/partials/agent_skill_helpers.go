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

func agentSkillEnabledURL(item model.AgentSkill) string {
	return agentSkillURL(item) + "/enabled"
}

func agentSkillListURL() string {
	return "/service/web/agent-skills/list"
}

func agentSkillCancelURL() string {
	return agentSkillListURL()
}

func agentSkillFilesContainerID(item model.AgentSkill) string {
	return "agent-skill-files-" + url.PathEscape(item.Flag)
}

func agentSkillFilesListURL(item model.AgentSkill) string {
	return fmt.Sprintf("/service/web/agent-skills/%s/files", url.PathEscape(item.Flag))
}

func agentSkillFileNewURL(item model.AgentSkill) string {
	return agentSkillFilesListURL(item) + "/new"
}

func agentSkillFileEditURL(item model.AgentSkill, file model.AgentSkillFile) string {
	return fmt.Sprintf("%s/edit?path=%s", agentSkillFilesListURL(item), url.QueryEscape(file.Path))
}

func agentSkillFileDeleteURL(item model.AgentSkill, file model.AgentSkillFile) string {
	return fmt.Sprintf("%s?path=%s", agentSkillFilesListURL(item), url.QueryEscape(file.Path))
}

func agentSkillFileRowID(item model.AgentSkill, file model.AgentSkillFile) string {
	return fmt.Sprintf("agent-skill-file-%s-%s", url.PathEscape(item.Flag), url.PathEscape(file.Path))
}

func agentSkillFileFormID(item model.AgentSkill, file model.AgentSkillFile, isNew bool) string {
	if isNew {
		return "agent-skill-file-form-new-" + url.PathEscape(item.Flag)
	}
	return "agent-skill-file-form-" + agentSkillFileRowID(item, file)
}

func agentSkillFileFormTitle(isNew bool) string {
	if isNew {
		return "New Skill File"
	}
	return "Edit Skill File"
}

func agentSkillDescriptionPreview(description string) string {
	if len(description) <= 60 {
		return description
	}
	return description[:57] + "..."
}
