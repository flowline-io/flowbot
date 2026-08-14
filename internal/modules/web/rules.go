package web

import (
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

// allWebserviceRules lists every route group registered under /service/web.
// Rules() exposes each slice separately (31 groups).
var allWebserviceRules = [][]webservice.Rule{
	homeWebserviceRules,
	loginWebserviceRules,
	accountWebserviceRules,
	configWebserviceRules,
	settingsWebserviceRules,
	healthzWebserviceRules,
	aboutWebserviceRules,
	hubWebserviceRules,
	pipelineWebserviceRules,
	functionWebserviceRules,
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
