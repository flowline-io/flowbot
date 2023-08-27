package markdown

import (
	"github.com/sysatom/flowbot/internal/ruleset/webservice"
	"github.com/sysatom/flowbot/pkg/route"
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
