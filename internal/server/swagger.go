//go:build swagger

package server

import (
	swagger "github.com/flowline-io/fiberswagger"
	_ "github.com/flowline-io/flowbot/docs"
)

func init() {
	swagHandler = swagger.HandlerDefault
}
