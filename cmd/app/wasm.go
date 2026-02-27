//go:build js && wasm

package main

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// main is the Wasm (browser) entry point.
// It registers routes and starts the go-app browser event loop.
func main() {
	registerRoutes()
	app.RunWhenOnBrowser()
}
