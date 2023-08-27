package markdown

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "GET",
		Path:          "/editor/{flag}",
		Function:      editor,
		Documentation: "get markdown editor",
		Option: []route.Option{
			route.WithPathParam("flag", "flag param", "string"),
		},
	},
	{
		Method:        "POST",
		Path:          "/markdown",
		Function:      saveMarkdown,
		Documentation: "create markdown page",
	},
}
