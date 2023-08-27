package dev

import (
	"github.com/sysatom/flowbot/internal/ruleset/webservice"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/pkg/route"
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
