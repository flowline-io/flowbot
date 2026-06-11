package web

import (
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

// allWebserviceRules lists every route group registered under /service/web.
// Rules() exposes each slice separately (13 groups; formerly bundled in webservice.go).
var allWebserviceRules = [][]webservice.Rule{
	homeWebserviceRules,
	loginWebserviceRules,
	configWebserviceRules,
	healthzWebserviceRules,
	hubWebserviceRules,
	pipelineWebserviceRules,
	viewWebserviceRules,
	eventWebserviceRules,
	relationsWebserviceRules,
	notificationWebserviceRules,
	notifySettingsWebserviceRules,
	homelabWebserviceRules,
	tokenWebserviceRules,
}
