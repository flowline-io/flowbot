package apps

import (
	"net/http"
	"strconv"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v3"
)

// API provides HTTP handlers for app management
type API struct {
	manager *Manager
	store   store.Adapter
}

// NewAPI creates a new app API
func NewAPI(manager *Manager, storeAdapter store.Adapter) *API {
	return &API{
		manager: manager,
		store:   storeAdapter,
	}
}

// ListApps lists all apps with associated bots
func (a *API) ListApps(c fiber.Ctx) error {
	apps, err := a.store.GetApps()
	if err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get apps",
		})
	}

	// Build response with app-bot associations
	type AppWithBot struct {
		App *model.App `json:"app"`
		Bot *model.Bot `json:"bot"`
	}

	result := make([]AppWithBot, 0, len(apps))
	for _, app := range apps {
		// Get associated bot by name (one-to-one relationship)
		bot, err := a.store.GetBotByName(app.Name)
		if err != nil {
			// Bot not found, include app without bot
			result = append(result, AppWithBot{
				App: app,
				Bot: nil,
			})
		} else {
			result = append(result, AppWithBot{
				App: app,
				Bot: bot,
			})
		}
	}

	return c.JSON(result)
}

// GetApp gets an app by ID with associated bot
func (a *API) GetApp(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid app id",
		})
	}

	app, err := a.store.GetApp(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "app not found",
		})
	}

	// Update status
	app, err = a.manager.GetAppStatus(c.Context(), app.Name)
	if err != nil {
		flog.Error(err)
	}

	// Get associated bot by name (one-to-one relationship)
	bot, err := a.store.GetBotByName(app.Name)
	if err != nil {
		// Bot not found, return app without bot
		return c.JSON(fiber.Map{
			"app": app,
			"bot": nil,
		})
	}

	return c.JSON(fiber.Map{
		"app": app,
		"bot": bot,
	})
}

// ScanApps scans for apps
func (a *API) ScanApps(c fiber.Ctx) error {
	if err := a.manager.ScanApps(c.Context()); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to scan apps",
		})
	}

	return c.JSON(fiber.Map{
		"message": "apps scanned successfully",
	})
}

// StartApp starts an app
func (a *API) StartApp(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid app id",
		})
	}

	app, err := a.store.GetApp(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "app not found",
		})
	}

	if err := a.manager.StartApp(c.Context(), app.Name); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "app started successfully",
	})
}

// StopApp stops an app
func (a *API) StopApp(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid app id",
		})
	}

	app, err := a.store.GetApp(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "app not found",
		})
	}

	if err := a.manager.StopApp(c.Context(), app.Name); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "app stopped successfully",
	})
}

// RestartApp restarts an app
func (a *API) RestartApp(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid app id",
		})
	}

	app, err := a.store.GetApp(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "app not found",
		})
	}

	if err := a.manager.RestartApp(c.Context(), app.Name); err != nil {
		flog.Error(err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "app restarted successfully",
	})
}
