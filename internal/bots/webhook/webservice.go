package webhook

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	{
		Method:   "POST",
		Path:     "/trigger/:flag",
		Function: webhook,
	},
}
