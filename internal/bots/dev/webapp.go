package dev

import (
	"embed"
	"github.com/sysatom/flowbot/internal/bots"
	"net/http"
)

//go:embed webapp/build
var dist embed.FS

func webapp(rw http.ResponseWriter, req *http.Request) {
	bots.ServeFile(rw, req, dist, "webapp/build")
}
