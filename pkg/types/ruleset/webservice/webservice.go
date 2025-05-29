package webservice

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
)

type Rule struct {
	Method   string
	Path     string
	Function fiber.Handler
	Option   []route.Option
}

func (r Rule) ID() string {
	return fmt.Sprintf("%s_%s", r.Method, r.Path)
}

func (r Rule) TYPE() types.RulesetType {
	return types.WebserviceRule
}

type Ruleset []Rule

func Get(path string, function fiber.Handler, option ...route.Option) Rule {
	return Rule{
		Method:   "GET",
		Path:     path,
		Function: function,
		Option:   option,
	}
}

func Post(path string, function fiber.Handler, option ...route.Option) Rule {
	return Rule{
		Method:   "POST",
		Path:     path,
		Function: function,
		Option:   option,
	}
}

func Put(path string, function fiber.Handler, option ...route.Option) Rule {
	return Rule{
		Method:   "PUT",
		Path:     path,
		Function: function,
		Option:   option,
	}
}

func Delete(path string, function fiber.Handler, option ...route.Option) Rule {
	return Rule{
		Method:   "DELETE",
		Path:     path,
		Function: function,
		Option:   option,
	}
}
