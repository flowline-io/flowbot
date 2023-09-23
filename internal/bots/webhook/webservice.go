package webhook

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "POST",
		Path:          "/webhook/:flag",
		Function:      webhook,
		Documentation: "trigger webhook",
	},
}
