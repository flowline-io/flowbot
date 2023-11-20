package webhook

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Post("/trigger/:flag", webhook),
}
