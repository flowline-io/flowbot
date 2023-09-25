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
