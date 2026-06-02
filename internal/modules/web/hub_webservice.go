package web

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var hubWebserviceRules = []webservice.Rule{
	webservice.Get("/hub", hubAppsPage, route.WithNotAuth()),
	webservice.Get("/hub/list", hubAppsList, route.WithNotAuth()),
	webservice.Get("/hub/:name", hubAppDetailPage, route.WithNotAuth()),
	webservice.Get("/hub/:name/status", hubAppStatusPartial, route.WithNotAuth()),
	webservice.Get("/hub/:name/logs/stream", hubAppLogsSSE, route.WithNotAuth()),
	webservice.Post("/hub/:name/start", hubAppStartAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/stop", hubAppStopAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/restart", hubAppRestartAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/pull", hubAppPullAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/update", hubAppUpdateAction, route.WithNotAuth()),
}

// hubAppsPage renders the full apps list page.
func hubAppsPage(c fiber.Ctx) error {
	apps := homelab.DefaultRegistry.List()
	updatedAts := loadUpdatedAts(c.Context())
	c.Type("html")
	return pages.HubAppsPage(apps, updatedAts).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppsList returns the table partial for HTMX auto-refresh.
func hubAppsList(c fiber.Ctx) error {
	apps := homelab.DefaultRegistry.List()
	updatedAts := loadUpdatedAts(c.Context())
	c.Type("html")
	return partials.HubAppsTable(apps, updatedAts).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppDetailPage renders the full detail page for a single app.
func hubAppDetailPage(c fiber.Ctx) error {
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}
	status, _ := homelab.DefaultRuntime.Status(c.Context(), app)
	perms := homelab.DefaultRegistry.Permissions()
	c.Type("html")
	return pages.HubAppDetailPage(app, status, perms).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppStatusPartial returns the status badge partial for HTMX swaps after actions.
func hubAppStatusPartial(c fiber.Ctx) error {
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}
	status, err := homelab.DefaultRuntime.Status(c.Context(), app)
	if err != nil {
		status = app.Status
	}
	c.Type("html")
	return pages.HubAppStatusBadge(status).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppLogsSSE streams logs via Server-Sent Events.
func hubAppLogsSSE(c fiber.Ctx) error {
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}
	tail := 100
	if raw := c.Query("tail"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			tail = parsed
		}
	}
	logs, err := homelab.DefaultRuntime.Logs(c.Context(), app, tail)
	if err != nil {
		if errors.Is(err, types.ErrNotImplemented) {
			return c.Status(http.StatusNotImplemented).SendString("logs not available")
		}
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	ctx := c.Context()
	return c.SendStreamWriter(func(w *bufio.Writer) {
		for _, line := range logs {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if _, fErr := fmt.Fprintf(w, "data: %s\n\n", line); fErr != nil {
				return
			}
			if fErr := w.Flush(); fErr != nil {
				return
			}
		}
	})
}

// hubAppStartAction starts an app and returns the updated status badge.
func hubAppStartAction(c fiber.Ctx) error {
	return hubLifecycleAction(c, homelab.DefaultRuntime.Start, "start")
}

// hubAppStopAction stops an app and returns the updated status badge.
func hubAppStopAction(c fiber.Ctx) error {
	return hubLifecycleAction(c, homelab.DefaultRuntime.Stop, "stop")
}

// hubAppRestartAction restarts an app and returns the updated status badge.
func hubAppRestartAction(c fiber.Ctx) error {
	return hubLifecycleAction(c, homelab.DefaultRuntime.Restart, "restart")
}

// hubAppPullAction pulls an app's images and returns the updated status badge.
func hubAppPullAction(c fiber.Ctx) error {
	return hubLifecycleAction(c, homelab.DefaultRuntime.Pull, "pull")
}

// hubAppUpdateAction pulls and starts an app, returning the updated status badge.
func hubAppUpdateAction(c fiber.Ctx) error {
	return hubLifecycleAction(c, homelab.DefaultRuntime.Update, "update")
}

// hubLifecycleAction performs a lifecycle operation on an app and returns the status partial.
func hubLifecycleAction(c fiber.Ctx, fn func(ctx context.Context, app homelab.App) error, operation string) error {
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}

	if err := fn(c.Context(), app); err != nil {
		if errors.Is(err, types.ErrNotImplemented) {
			return c.Status(http.StatusNotImplemented).SendString(operation + " not available")
		}
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}

	status, err := homelab.DefaultRuntime.Status(c.Context(), app)
	if err != nil {
		status = app.Status
	}
	c.Type("html")
	return pages.HubAppStatusBadge(status).Render(c.Context(), c.Response().BodyWriter())
}

// loadUpdatedAts loads updated timestamps from the store and formats them.
func loadUpdatedAts(ctx context.Context) map[string]string {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	infos, err := store.NewHubStore(client).ListApps(ctx)
	if err != nil || len(infos) == 0 {
		return nil
	}
	m := make(map[string]string, len(infos))
	for _, info := range infos {
		m[info.Name] = info.UpdatedAt.Format("2006-01-02 15:04")
	}
	return m
}
