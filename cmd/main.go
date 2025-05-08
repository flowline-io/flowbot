package main

import (
	"github.com/flowline-io/flowbot/internal/server"
	// Importing automaxprocs automatically adjusts GOMAXPROCS.
	_ "go.uber.org/automaxprocs"
	"go.uber.org/fx"
)

// @title						Flowbot API
// @version					1.0
// @description				Flowbot Chatbot API
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
	// server.Run()
	fx.New(
		server.Modules,
		// fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		// 	return &fxevent.ZapLogger{Logger: log}
		// }),
	).Run()
}
