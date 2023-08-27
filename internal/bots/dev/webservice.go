package dev

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webservice"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/route"
)

var webserviceRules = []webservice.Rule{
	{
		Method:        "GET",
		Path:          "/example",
		Function:      example,
		Documentation: "get example data",
		Option: []route.Option{
			route.WithReturns(model.Message{}),
			route.WithWrites(model.Message{}),
		},
	},
}
