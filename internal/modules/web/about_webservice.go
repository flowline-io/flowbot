package web

import (
	"runtime"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/version"
)

var aboutWebserviceRules = []webservice.Rule{
	webservice.Get("/about", aboutPage),
}

// aboutPage renders version and build information for this instance.
func aboutPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	data := pages.AboutData{
		Version:    version.Buildtags,
		Buildstamp: version.Buildstamp,
		GoVersion:  runtime.Version(),
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
	}

	ctx.Type("html")
	return pages.AboutPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}
