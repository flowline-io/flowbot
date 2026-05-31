//go:build integration
// +build integration

package specs

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/gomega"
)

var pipelineRoutesMounted bool

// mountPipelineRoutes registers pipeline CRUD routes on the given fiber app
// using the same store methods as the production handlers.
func mountPipelineRoutes(app *fiber.App) {
	if pipelineRoutesMounted {
		return
	}
	pipelineRoutesMounted = true
	pipeStore := store.NewPipelineStore(EntClient)

	// POST /service/web/pipelines — create pipeline, redirect to editor
	app.Post("/service/web/pipelines", func(c fiber.Ctx) error {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}
		if body.Name == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
		}
		if err := pipeStore.CreateDefinition(context.Background(), body.Name, body.Description); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+body.Name)
		return c.SendStatus(http.StatusOK)
	})

	// GET /service/web/pipelines/capabilities
	app.Get("/service/web/pipelines/capabilities", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "success",
			"data": []fiber.Map{
				{
					"type":        "bookmark",
					"backend":     "native",
					"description": "bookmark management",
					"operations": []fiber.Map{
						{"name": "list", "description": "list bookmarks"},
						{"name": "create", "description": "create bookmark"},
						{"name": "get", "description": "get bookmark"},
					},
				},
			},
		})
	})

	// GET /service/web/pipelines/:name/yaml
	app.Get("/service/web/pipelines/:name/yaml", func(c fiber.Ctx) error {
		def, err := pipeStore.GetDefinitionByName(context.Background(), c.Params("name"))
		if err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "not found"})
			}
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"yaml": def.YamlDraft, "version": def.Version, "status": def.Status})
	})

	// PUT /service/web/pipelines/:name — update draft
	app.Put("/service/web/pipelines/:name", func(c fiber.Ctx) error {
		var body struct {
			Yaml    string `json:"yaml"`
			Version int    `json:"version"`
		}
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}
		def, err := pipeStore.UpdateDefinitionDraft(context.Background(), c.Params("name"), body.Yaml, body.Version)
		if err != nil {
			if errors.Is(err, types.ErrConflict) {
				return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "conflict"})
			}
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
	})

	// PUT /service/web/pipelines/:name/publish
	app.Put("/service/web/pipelines/:name/publish", func(c fiber.Ctx) error {
		var body struct {
			Version int `json:"version"`
		}
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}
		// Validate YAML first
		def, err := pipeStore.GetDefinitionByName(context.Background(), c.Params("name"))
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		if _, err := pipeline.ParseEditorYAML(def.YamlDraft); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid yaml"})
		}
		def, err = pipeStore.PublishDefinition(context.Background(), c.Params("name"), body.Version)
		if err != nil {
			if errors.Is(err, types.ErrConflict) {
				return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "conflict"})
			}
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
	})

	// DELETE /service/web/pipelines/:name
	app.Delete("/service/web/pipelines/:name", func(c fiber.Ctx) error {
		name := c.Params("name")
		_, err := pipeStore.GetDefinitionByName(context.Background(), name)
		if err != nil {
			if errors.Is(err, types.ErrNotFound) {
				return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "not found"})
			}
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		_, err = pipeStore.DeleteDefinitionByName(context.Background(), name)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(http.StatusOK)
	})

	app.Get("/service/web/pipelines/:name/mock", func(c fiber.Ctx) error {
		switch source := c.Query("source"); source {
		case "event":
			return c.JSON(fiber.Map{"source": "event", "payload": fiber.Map{"event_id": "mock-ev-001"}})
		case "webhook":
			return c.JSON(fiber.Map{"source": "webhook", "payload": fiber.Map{"event_id": "mock-wb-001"}})
		case "cron":
			return c.JSON(fiber.Map{"source": "cron", "payload": fiber.Map{}})
		default:
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "missing source"})
		}
	})
}

// seedBDDPipeline creates a pipeline definition with given name and yaml_draft.
func seedBDDPipeline(name, yamlDraft string) {
	ctx := context.Background()
	now := time.Now()
	_, err := EntClient.PipelineDefinition.Create().
		SetName(name).
		SetYamlDraft(yamlDraft).
		SetVersion(1).
		SetStatus("draft").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	Expect(err).NotTo(HaveOccurred())
}
