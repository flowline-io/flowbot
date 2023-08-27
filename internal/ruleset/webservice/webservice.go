package webservice

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/sysatom/flowbot/pkg/route"
)

type Rule struct {
	Method        string
	Path          string
	Function      restful.RouteFunction
	Documentation string
	Option        []route.Option
}

type Ruleset []Rule
