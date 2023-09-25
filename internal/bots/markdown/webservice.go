package markdown

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "GET",
		Path:          "/editor/:flag",
		Function:      editor,
		Documentation: "get markdown editor",
	},
	{
		Method:        "POST",
		Path:          "/data",
		Function:      saveMarkdown,
		Documentation: "create markdown page",
	},
}
