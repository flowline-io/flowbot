package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "GET",
		Path:          "/actions",
		Function:      actions,
		Documentation: "get bot actions",
		Option:        []route.Option{},
	},
}
