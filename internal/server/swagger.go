//go:build swagger

package server

import (
	"github.com/gofiber/contrib/v3/swaggo"

	_ "github.com/flowline-io/flowbot/docs/api"
)

func init() {
	swagHandler = swaggo.HandlerDefault
}
