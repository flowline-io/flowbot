package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/actions", actions),
}
