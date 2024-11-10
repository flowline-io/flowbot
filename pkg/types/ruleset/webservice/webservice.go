package webservice

import (
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/gofiber/fiber/v2"
)

type Rule struct {
	Method   string
	Path     string
	Function fiber.Handler
	Option   []route.Option
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
