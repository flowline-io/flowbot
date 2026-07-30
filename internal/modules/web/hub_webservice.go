package web

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

const (
	appStatusCacheTTL      = 8 * time.Second
	appStatusConcurrency   = 8
	appStatusLookupTimeout = 3 * time.Second
)

type appStatusCacheEntry struct {
	status    homelab.AppStatus
	expiresAt time.Time
}

var (
	appStatusCacheMu sync.Mutex
	appStatusCache   = map[string]appStatusCacheEntry{}
)

var hubWebserviceRules = []webservice.Rule{
	webservice.Get("/hub", hubAppsPage),
	webservice.Get("/hub/list", hubAppsList),
	webservice.Get("/capabilities", hubCapabilitiesPage),
	webservice.Get("/capabilities/grid", hubCapabilitiesGrid),
	webservice.Get("/hub/:name", hubAppDetailPage),
	webservice.Get("/hub/:name/status", hubAppStatusPartial),
	webservice.Get("/hub/:name/logs/stream", hubAppLogsSSE),
	webservice.Post("/hub/:name/start", hubAppStartAction),
	webservice.Post("/hub/:name/stop", hubAppStopAction),
	webservice.Post("/hub/:name/restart", hubAppRestartAction),
	webservice.Post("/hub/:name/pull", hubAppPullAction),
	webservice.Post("/hub/:name/update", hubAppUpdateAction),
}

// hubAppsPage renders the full apps list page.
func hubAppsPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	apps, updatedAts := loadAppsWithUpdatedAts(c.Context())
	c.Type("html")
	return pages.HubAppsPage(apps, updatedAts).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppsList returns the table partial for HTMX auto-refresh.
func hubAppsList(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	apps, updatedAts := loadAppsWithUpdatedAts(c.Context())
	c.Type("html")
	return partials.HubAppsTable(apps, updatedAts).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppDetailPage renders the full detail page for a single app.
func hubAppDetailPage(c fiber.Ctx) error {
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
	perms := homelab.DefaultRegistry.Permissions()
	c.Type("html")
	return pages.HubAppDetailPage(app, status, perms).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppStatusPartial returns the status badge partial for HTMX swaps after actions.
func hubAppStatusPartial(c fiber.Ctx) error {
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
	c.Type("html")
	return pages.HubAppStatusBadge(status).Render(c.Context(), c.Response().BodyWriter())
}

// hubAppLogsSSE streams logs via Server-Sent Events.
func hubAppLogsSSE(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
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
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}

	scope := lifecycleScope(operation)
	if scope != "" && !route.ScopeHandler(c, scope) {
		return toastError(c, "Permission denied: missing scope "+scope)
	}
	if !homelab.AllowsLifecycle(homelab.DefaultRegistry.Permissions(), operation) {
		return toastError(c, "Permission denied: "+operation+" is not allowed by Hub config")
	}

	if err := fn(c.Context(), app); err != nil {
		if errors.Is(err, types.ErrNotImplemented) {
			return toastError(c, operation+" is not available for this app")
		}
		return toastError(c, "Could not "+operation+" "+name+": "+err.Error())
	}

	invalidateAppStatusCache(name)
	status, err := homelab.DefaultRuntime.Status(c.Context(), app)
	if err != nil {
		status = app.Status
	} else {
		storeAppStatusCache(name, status)
	}
	setShowToast(c, "success", hubLifecycleSuccessMessage(name, operation))
	c.Type("html")
	return pages.HubAppStatusBadge(status).Render(c.Context(), c.Response().BodyWriter())
}

// hubLifecycleSuccessMessage returns the success toast for a Hub lifecycle action.
func hubLifecycleSuccessMessage(name, operation string) string {
	switch operation {
	case "start":
		return name + " started"
	case "stop":
		return name + " stopped"
	case "restart":
		return name + " restarted"
	case "pull":
		return name + " image pull started"
	case "update":
		return name + " updated"
	default:
		return name + " " + operation + " completed"
	}
}

// lifecycleScope maps a lifecycle operation to the matching hub:apps:* scope.
func lifecycleScope(operation string) string {
	switch operation {
	case "start":
		return auth.ScopeHubAppsStart
	case "stop":
		return auth.ScopeHubAppsStop
	case "restart":
		return auth.ScopeHubAppsRestart
	case "pull":
		return auth.ScopeHubAppsPull
	case "update":
		return auth.ScopeHubAppsUpdate
	default:
		return ""
	}
}

// loadAppsWithUpdatedAts enriches app statuses and loads store timestamps in parallel.
func loadAppsWithUpdatedAts(ctx context.Context) ([]homelab.App, map[string]string) {
	var (
		apps       []homelab.App
		updatedAts map[string]string
		wg         sync.WaitGroup
	)
	wg.Go(func() {
		apps = enrichAppStatuses(ctx, homelab.DefaultRegistry.List())
	})
	wg.Go(func() {
		updatedAts = loadUpdatedAts(ctx)
	})
	wg.Wait()
	return apps, updatedAts
}

// loadUpdatedAts loads updated timestamps from the store and formats them.
func loadUpdatedAts(ctx context.Context) map[string]string {
	if store.Database == nil || store.Database.GetClient() == nil {
		return nil
	}
	infos, err := store.NewHubStore(store.Database.GetClient()).ListApps(ctx)
	if err != nil {
		flog.Warn("hub: loadUpdatedAts failed: %v", err)
		return nil
	}
	if len(infos) == 0 {
		return nil
	}
	m := make(map[string]string, len(infos))
	for _, info := range infos {
		m[info.Name] = info.UpdatedAt.Format("2006-01-02 15:04")
	}
	return m
}

// enrichAppStatuses copies apps and fills Status from DefaultRuntime when available.
// Lookups are bounded-concurrent and cached briefly to keep Apps/Registry pages responsive.
func enrichAppStatuses(ctx context.Context, apps []homelab.App) []homelab.App {
	if len(apps) == 0 {
		return apps
	}
	out := make([]homelab.App, len(apps))
	copy(out, apps)

	misses := make([]int, 0, len(out))
	for i := range out {
		if status, ok := lookupAppStatusCache(out[i].Name); ok {
			out[i].Status = status
			continue
		}
		misses = append(misses, i)
	}
	if len(misses) == 0 {
		return out
	}

	sem := make(chan struct{}, appStatusConcurrency)
	var wg sync.WaitGroup
	for _, idx := range misses {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			app := out[idx]
			lookupCtx, cancel := context.WithTimeout(ctx, appStatusLookupTimeout)
			defer cancel()
			status, err := homelab.DefaultRuntime.Status(lookupCtx, app)
			if err != nil {
				flog.Warn("hub: status lookup failed for %s: %v", app.Name, err)
				return
			}
			out[idx].Status = status
			storeAppStatusCache(app.Name, status)
		})
	}
	wg.Wait()
	return out
}

func lookupAppStatusCache(name string) (homelab.AppStatus, bool) {
	appStatusCacheMu.Lock()
	defer appStatusCacheMu.Unlock()
	entry, ok := appStatusCache[name]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.status, true
}

func storeAppStatusCache(name string, status homelab.AppStatus) {
	appStatusCacheMu.Lock()
	defer appStatusCacheMu.Unlock()
	appStatusCache[name] = appStatusCacheEntry{
		status:    status,
		expiresAt: time.Now().Add(appStatusCacheTTL),
	}
}

func invalidateAppStatusCache(name string) {
	appStatusCacheMu.Lock()
	defer appStatusCacheMu.Unlock()
	delete(appStatusCache, name)
}

func clearAppStatusCache() {
	appStatusCacheMu.Lock()
	defer appStatusCacheMu.Unlock()
	appStatusCache = map[string]appStatusCacheEntry{}
}

// hubCapabilitiesPage renders the full capabilities browser page.
func hubCapabilitiesPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	descriptors := hub.Default.List()
	typeList := uniqueTypes(descriptors)
	providerList := uniqueProviders(descriptors)
	c.Type("html")
	return pages.CapabilitiesPage(descriptors, typeList, providerList).Render(c.Context(), c.Response().BodyWriter())
}

// hubCapabilitiesGrid returns the filtered card grid partial for HTMX swaps.
func hubCapabilitiesGrid(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	descriptors := hub.Default.List()

	typeFilter := c.Query("type")
	providerFilter := c.Query("provider")
	filtered := typeFilter != "" || providerFilter != ""

	if filtered {
		tmp := make([]hub.Descriptor, 0, len(descriptors))
		for _, d := range descriptors {
			if typeFilter != "" && string(d.Type) != typeFilter {
				continue
			}
			if providerFilter != "" && string(d.Type) != providerFilter {
				continue
			}
			tmp = append(tmp, d)
		}
		descriptors = tmp
	}

	c.Type("html")
	return partials.CapabilityGrid(descriptors, filtered).Render(c.Context(), c.Response().BodyWriter())
}

// uniqueTypes extracts unique capability type strings from descriptors, sorted.
func uniqueTypes(descriptors []hub.Descriptor) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		t := string(d.Type)
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	slices.Sort(result)
	return result
}

// uniqueProviders extracts unique provider (capability type) strings from descriptors, sorted.
func uniqueProviders(descriptors []hub.Descriptor) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		t := string(d.Type)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	slices.Sort(result)
	return result
}
