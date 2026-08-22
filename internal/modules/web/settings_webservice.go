package web

import (
	"github.com/gofiber/fiber/v3"

	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var settingsWebserviceRules = []webservice.Rule{
	webservice.Get("/settings", settingsPage),
}

// settingsPage renders a read-only catalog of the effective config.App values.
func settingsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	data := pages.NewSettingsPageData(ctx.Context(), pkgconfig.SettingsCatalog(&pkgconfig.App))
	ctx.Type("html")
	return pages.SettingsPage(ctx.Context(), data).Render(ctx.Context(), ctx.Response().BodyWriter())
}
