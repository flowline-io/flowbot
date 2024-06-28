package dev

import (
	"embed"
	"net/http"

	"github.com/flowline-io/flowbot/internal/bots"
)

//go:embed webapp/build
var dist embed.FS

func webapp(rw http.ResponseWriter, req *http.Request) {
	bots.ServeFile(rw, req, dist, "webapp/build")
}
