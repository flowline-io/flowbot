package okr

import (
	"embed"
	"github.com/flowline-io/flowbot/internal/bots"
	"net/http"
)

//go:embed webapp/build
var dist embed.FS

func webapp(rw http.ResponseWriter, req *http.Request) {
	bots.ServeFile(rw, req, dist, "webapp/build")
}
