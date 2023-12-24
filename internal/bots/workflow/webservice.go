package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/actions", actions),

	webservice.Get("/workflows", workflowList),
	webservice.Get("/workflow/:id", workflowDetail),
	webservice.Post("/workflow", workflowCreate),
	webservice.Put("/workflow/:id", workflowUpdate),
	webservice.Delete("/workflow/:id", workflowDelete),

	webservice.Get("/workflow/:id/triggers", workflowTriggerList),

	webservice.Get("/workflow/:id/jobs", workflowJobList),
	webservice.Get("/job/:id", workflowJobDetail),
	webservice.Post("/job/:id/rerun", workflowJobRerun),

	webservice.Get("/workflow/:id/script", workflowScriptDetail),
}
