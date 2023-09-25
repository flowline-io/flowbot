package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	{
		Method:   "GET",
		Path:     "/actions",
		Function: actions,
	},
}
