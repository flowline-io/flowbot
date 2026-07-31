package web

import (
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

// allWebserviceRules lists every route group registered under /service/web.
// Rules() exposes each slice separately (30 groups).
var allWebserviceRules = [][]webservice.Rule{
	homeWebserviceRules,
	loginWebserviceRules,
	accountWebserviceRules,
	configWebserviceRules,
	healthzWebserviceRules,
	aboutWebserviceRules,
	hubWebserviceRules,
	pipelineWebserviceRules,
	viewWebserviceRules,
	eventWebserviceRules,
	relationsWebserviceRules,
	notificationWebserviceRules,
	inboxWebserviceRules,
	notifySettingsWebserviceRules,
	notifyPlaygroundWebserviceRules,
	agentSkillsWebserviceRules,
	agentKnowledgeWebserviceRules,
	agentMemoryWebserviceRules,
	agentSessionSummariesWebserviceRules,
	agentSubagentsWebserviceRules,
	agentSessionsWebserviceRules,
	agentScheduledTasksWebserviceRules,
	agentsWebserviceRules,
	chatAgentPermissionsWebserviceRules,
	homelabWebserviceRules,
	tokenWebserviceRules,
	clipsListWebserviceRules,
	workflowWebserviceRules,
	commandPaletteWebserviceRules,
	lifeWebserviceRules,
}
