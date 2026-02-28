// Package main is the entry point for the Flowbot Admin PWA.
//
// Two build modes:
//   - Wasm (GOOS=js GOARCH=wasm): browser-side SPA, see wasm.go
//   - Native (default):           PWA server, see server.go
package main

import (
	"github.com/flowline-io/flowbot/cmd/app/pages"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// registerRoutes registers all frontend page routes.
// Shared by both Wasm and native server builds.
func registerRoutes() {
	app.Route("/", func() app.Composer { return &pages.Home{} })
	app.Route("/admin", func() app.Composer { return &pages.Dashboard{} })
	app.Route("/admin/login", func() app.Composer { return &pages.Login{} })
	app.Route("/admin/settings", func() app.Composer { return &pages.Settings{} })
	app.Route("/admin/containers", func() app.Composer { return &pages.Containers{} })
}
