package web

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var homelabWebserviceRules = []webservice.Rule{
	webservice.Get("/homelab", homelabRegistryPage, route.WithNotAuth()),
	webservice.Get("/homelab/:name", homelabRegistryDetailPage, route.WithNotAuth()),
	webservice.Post("/homelab/rescan", homelabRegistryRescan, route.WithNotAuth()),
}

// homelabRegistryPage renders the full homelab registry card list page.
func homelabRegistryPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	apps := homelab.DefaultRegistry.List()
	updatedAts := loadUpdatedAts(c.Context())
	scannedAt := latestScannedAt(updatedAts)
	c.Type("html")
	return pages.HomelabPage(apps, scannedAt).Render(c.Context(), c.Response().BodyWriter())
}

// homelabRegistryDetailPage renders the detail page for a single homelab app.
func homelabRegistryDetailPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}
	status, err := homelab.DefaultRuntime.Status(c.Context(), app)
	if err != nil {
		status = app.Status
	}
	version := homelab.AppVersion(app)
	updatedAts := loadUpdatedAts(c.Context())
	scannedAt := ""
	if ts, ok := updatedAts[app.Name]; ok {
		scannedAt = ts
	}
	c.Type("html")
	return pages.HomelabDetailPage(app, status, version, scannedAt).Render(c.Context(), c.Response().BodyWriter())
}

// homelabRegistryRescan triggers a full homelab scan + probe cycle.
func homelabRegistryRescan(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	if err := homelab.RunRescan(); err != nil {
		flog.Warn("homelab rescan failed: %v", err)
		c.Set("HX-Redirect", "/service/web/homelab")
		return c.SendStatus(http.StatusOK)
	}
	c.Set("HX-Redirect", "/service/web/homelab")
	return c.SendStatus(http.StatusOK)
}

// latestScannedAt returns the most recent UpdatedAt timestamp from the apps map.
func latestScannedAt(updatedAts map[string]string) string {
	latest := ""
	for _, ts := range updatedAts {
		if ts > latest {
			latest = ts
		}
	}
	return latest
}
