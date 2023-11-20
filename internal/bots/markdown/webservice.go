package markdown

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/editor/:flag", editor),
	webservice.Post("/data", saveMarkdown),
}
