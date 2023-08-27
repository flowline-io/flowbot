package webhook

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "POST",
		Path:          "/webhook/{flag}",
		Function:      webhook,
		Documentation: "trigger webhook",
		Option: []route.Option{
			route.WithPathParam("flag", "flag param", "string"),
		},
	},
}
