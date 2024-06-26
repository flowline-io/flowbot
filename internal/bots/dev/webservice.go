package dev

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/example", example),
	webservice.Post("/upload", upload),
}
