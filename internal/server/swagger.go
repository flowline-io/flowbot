//go:build swagger

package server

import (
	_ "github.com/flowline-io/flowbot/docs/api"
	"github.com/gofiber/contrib/v3/swaggo"
)

func init() {
	swagHandler = swaggo.HandlerDefault
}
