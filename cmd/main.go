// Package main is the entry point for the Flowbot server binary.
package main

import (
	"time"

	"github.com/flowline-io/flowbot/internal/server"
	"github.com/flowline-io/flowbot/pkg/flog"

	// Importing automaxprocs automatically adjusts GOMAXPROCS.
	_ "go.uber.org/automaxprocs"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// @title						Flowbot API
// @version					1.0
// @description				Flowbot Homelab Data Hub HTTP API (partial OpenAPI; see docs/api/README.md)
// @license.name				GPL 3.0
// @license.url				https://github.com/flowline-io/flowbot/blob/master/LICENSE
// @host						localhost:6060
// @BasePath					/service
// @schemes					http
// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						X-AccessToken
// @description				access token
func main() {
	fx.New(
		server.Modules,
		fx.WithLogger(func() fxevent.Logger {
			return flog.NewFxLogger()
		}),
		fx.StopTimeout(30*time.Second),
	).Run()
}
