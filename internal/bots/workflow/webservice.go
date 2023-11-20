package workflow

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/actions", actions),

	webservice.Get("/workflows", example),
	webservice.Get("/workflow/{id}", example),
	webservice.Post("/workflow", example),
	webservice.Put("/workflow/{id}", example),
	webservice.Delete("/workflow/{id}", example),

	webservice.Get("/workflow/{id}/triggers", example),
	webservice.Post("/workflow/{id}/trigger", example),
	webservice.Put("/trigger/{id}", example),
	webservice.Delete("/trigger/{id}", example),

	webservice.Get("/workflow/{id}/jobs", example),
	webservice.Get("/job/{id}", example),
	webservice.Get("/job/{id}/rerun", example),

	webservice.Get("/workflow/{id}/dag", example),
	webservice.Put("/workflow/{id}/dag", example),
}
