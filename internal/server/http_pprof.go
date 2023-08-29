// Debug tooling. Dumps named profile in response to HTTP request at
// 		http(s)://<host-name>/<configured-path>/<profile-name>
// See godoc for the list of possible profile names: https://golang.org/pkg/runtime/pprof/#Profile

package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"net/http"
	"path"
	"runtime/pprof"
	"strings"
)

var pprofHttpRoot string

// Expose debug profiling at the given URL path.
func servePprof(app *fiber.App, serveAt string) {
	if serveAt == "" || serveAt == "-" {
		return
	}

	pprofHttpRoot = path.Clean("/"+serveAt) + "/"
	app.All(pprofHttpRoot, adaptor.HTTPHandlerFunc(profileHandler))

	logs.Info.Printf("pprof: profiling info exposed at '%s'", pprofHttpRoot)
}

func profileHandler(wrt http.ResponseWriter, req *http.Request) {
	wrt.Header().Set("X-Content-Type-Options", "nosniff")
	wrt.Header().Set("Content-Type", "text/plain; charset=utf-8")

	profileName := strings.TrimPrefix(req.URL.Path, pprofHttpRoot)

	profile := pprof.Lookup(profileName)
	if profile == nil {
		servePprofError(wrt, http.StatusNotFound, "Unknown profile '"+profileName+"'")
		return
	}

	// Respond with the requested profile.
	_ = profile.WriteTo(wrt, 2)
}

func servePprofError(wrt http.ResponseWriter, status int, txt string) {
	wrt.Header().Set("Content-Type", "text/plain; charset=utf-8")
	wrt.Header().Set("X-Go-Pprof", "1")
	wrt.Header().Del("Content-Disposition")
	wrt.WriteHeader(status)
	_, _ = fmt.Fprintln(wrt, txt)
}
