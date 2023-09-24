//go:build swagger

package server

import (
	_ "github.com/flowline-io/flowbot/docs"
	"github.com/gofiber/swagger"
)

func init() {
	swagHandler = swagger.HandlerDefault
}
