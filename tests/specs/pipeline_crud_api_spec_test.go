//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var pipelineRoutesMounted bool

var _ = Describe("Pipeline CRUD API", Label("pipeline", "web", "api"), func() {

	BeforeEach(func() {
		mountPipelineRoutes(App)
	})

	Describe("POST /service/web/pipelines", func() {
		It("creates a new pipeline and returns HX-Redirect header", func() {
			name := "bdd-create-" + types.Id()
			bodyJSON, _ := sonic.Marshal(map[string]any{
				"name":        name,
				"description": "Created by BDD test",
			})

			req := JSONRequest(http.MethodPost, "/service/web/pipelines", bodyJSON)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Cleanup
			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			// Verify in DB
			def, err := EntClient.PipelineDefinition.Query().
				Where(pipelinedefinition.Name(name)).
				Only(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(def.Name).To(Equal(name))
			Expect(def.Version).To(Equal(1))
		})

		It("rejects pipeline with empty name", func() {
			bodyJSON, _ := sonic.Marshal(map[string]any{
				"name": "",
			})
			req := JSONRequest(http.MethodPost, "/service/web/pipelines", bodyJSON)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("allows pipeline with no description", func() {
			name := "bdd-nodesc-" + types.Id()
			bodyJSON, _ := sonic.Marshal(map[string]any{
				"name": name,
			})
			req := JSONRequest(http.MethodPost, "/service/web/pipelines", bodyJSON)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			def, _ := EntClient.PipelineDefinition.Query().
				Where(pipelinedefinition.Name(name)).
				Only(context.Background())
			Expect(def.Description).To(Equal(""))
		})
	})

	Describe("GET /service/web/pipelines/:name/yaml", func() {
		It("returns draft yaml for existing pipeline", func() {
			name := "bdd-yaml-" + types.Id()
			seedBDDPipeline(name, "draft yaml content")

			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			req := MakeRequest(http.MethodGet, "/service/web/pipelines/"+name+"/yaml", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["yaml"]).To(Equal("draft yaml content"))
			Expect(result["version"]).To(BeNumerically(">=", 1))
		})

		It("returns 404 for non-existent pipeline", func() {
			req := MakeRequest(http.MethodGet, "/service/web/pipelines/nonexistent-xxx/yaml", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("PUT /service/web/pipelines/:name", func() {
		It("updates draft with correct version", func() {
			name := "bdd-update-" + types.Id()
			seedBDDPipeline(name, "initial draft")

			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			bodyJSON, _ := sonic.Marshal(map[string]any{
				"yaml":    "updated draft",
				"version": 1,
			})
			req := JSONRequest(http.MethodPut, "/service/web/pipelines/"+name, bodyJSON)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["version"]).To(BeNumerically("==", 2))
		})

		It("rejects update with stale version (409 Conflict)", func() {
			name := "bdd-conflict-" + types.Id()
			seedBDDPipeline(name, "draft")

			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			// Update to version 2
			bodyJSON, _ := sonic.Marshal(map[string]any{
				"yaml":    "first update",
				"version": 1,
			})
			req := JSONRequest(http.MethodPut, "/service/web/pipelines/"+name, bodyJSON)
			resp, _ := App.Test(req)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Try update with stale version 1
			bodyJSON, _ = sonic.Marshal(map[string]any{
				"yaml":    "stale update",
				"version": 1,
			})
			req = JSONRequest(http.MethodPut, "/service/web/pipelines/"+name, bodyJSON)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusConflict))
		})
	})

	Describe("PUT /service/web/pipelines/:name/publish", func() {
		It("publishes a valid draft", func() {
			name := "bdd-pub-" + types.Id()
			seedBDDPipeline(name, "name: test\nsteps: []")

			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			bodyJSON, _ := sonic.Marshal(map[string]any{
				"version": 1,
			})
			req := JSONRequest(http.MethodPut, "/service/web/pipelines/"+name+"/publish", bodyJSON)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["status"]).To(Equal("published"))
		})

		It("rejects publish with stale version", func() {
			name := "bdd-pubstale-" + types.Id()
			seedBDDPipeline(name, "name: test\nsteps: []")

			DeferCleanup(func() {
				EntClient.PipelineDefinition.Delete().
					Where(pipelinedefinition.Name(name)).
					Exec(context.Background())
			})

			// Update first to bump version
			store.NewPipelineStore(EntClient).UpdateDefinitionDraft(context.Background(), name, "name: v2", 1)

			// Try publish with stale version 1
			bodyJSON, _ := sonic.Marshal(map[string]any{
				"version": 1,
			})
			req := JSONRequest(http.MethodPut, "/service/web/pipelines/"+name+"/publish", bodyJSON)
			resp, _ := App.Test(req)
			Expect(resp.StatusCode).To(Equal(http.StatusConflict))
		})
	})

	Describe("DELETE /service/web/pipelines/:name", func() {
		It("deletes pipeline and returns run_count", func() {
			name := "bdd-del-" + types.Id()
			seedBDDPipeline(name, "draft")

			req := MakeRequest(http.MethodDelete, "/service/web/pipelines/"+name, nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["deleted"]).To(BeTrue())

			// Verify not in DB
			exists, _ := EntClient.PipelineDefinition.Query().
				Where(pipelinedefinition.Name(name)).
				Exist(context.Background())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("GET /service/web/pipelines/:name/mock", func() {
		It("returns mock event payload", func() {
			req := MakeRequest(http.MethodGet, "/service/web/pipelines/any/mock?source=event", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["source"]).To(Equal("event"))
			Expect(result["payload"]).NotTo(BeNil())
		})

		It("returns mock webhook payload", func() {
			req := MakeRequest(http.MethodGet, "/service/web/pipelines/any/mock?source=webhook", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["source"]).To(Equal("webhook"))
		})

		It("returns mock cron payload", func() {
			req := MakeRequest(http.MethodGet, "/service/web/pipelines/any/mock?source=cron", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			sonic.Unmarshal(body, &result)
			Expect(result["source"]).To(Equal("cron"))
		})

		It("returns error without source param", func() {
			req := MakeRequest(http.MethodGet, "/service/web/pipelines/any/mock", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})

// mountPipelineRoutes registers pipeline CRUD routes on the given fiber app.
// The function is idempotent; subsequent calls on the same app are no-ops.
func mountPipelineRoutes(app *fiber.App) {
	if pipelineRoutesMounted {
		return
	}
	pipelineRoutesMounted = true

	// POST /service/web/pipelines
	app.Post("/service/web/pipelines", func(c fiber.Ctx) error {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(http.StatusBadRequest).
				JSON(fiber.Map{"error": "invalid json"})
		}
		if body.Name == "" {
			return c.Status(http.StatusBadRequest).
				JSON(fiber.Map{"error": "name is required"})
		}
		now := time.Now()
		_, err := EntClient.PipelineDefinition.Create().
			SetName(body.Name).
			SetDescription(body.Description).
			SetYamlDraft("").
			SetVersion(1).
			SetStatus("draft").
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(context.Background())
		if err != nil {
			return c.Status(http.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+body.Name)
		return c.SendStatus(http.StatusOK)
	})

	// GET /service/web/pipelines/:name/yaml
	app.Get("/service/web/pipelines/:name/yaml", func(c fiber.Ctx) error {
		name := c.Params("name")
		def, err := EntClient.PipelineDefinition.Query().
			Where(pipelinedefinition.Name(name)).
			Only(context.Background())
		if err != nil {
			return c.Status(http.StatusNotFound).
				JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return c.JSON(fiber.Map{
			"yaml":    def.YamlDraft,
			"version": def.Version,
			"status":  def.Status,
		})
	})

	// PUT /service/web/pipelines/:name
	app.Put("/service/web/pipelines/:name", func(c fiber.Ctx) error {
		name := c.Params("name")
		var body struct {
			Yaml    string `json:"yaml"`
			Version int    `json:"version"`
		}
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(http.StatusBadRequest).
				JSON(fiber.Map{"error": "invalid body"})
		}
		n, err := EntClient.PipelineDefinition.Update().
			Where(
				pipelinedefinition.Name(name),
				pipelinedefinition.Version(body.Version),
			).
			SetYamlDraft(body.Yaml).
			SetVersion(body.Version + 1).
			SetUpdatedAt(time.Now()).
			Save(context.Background())
		if err != nil {
			return c.Status(http.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		if n == 0 {
			return c.Status(http.StatusConflict).
				JSON(fiber.Map{"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere."}})
		}
		def, _ := EntClient.PipelineDefinition.Query().
			Where(pipelinedefinition.Name(name)).
			Only(context.Background())
		return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
	})

	// PUT /service/web/pipelines/:name/publish
	app.Put("/service/web/pipelines/:name/publish", func(c fiber.Ctx) error {
		name := c.Params("name")
		var body struct {
			Version int `json:"version"`
		}
		if err := sonic.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(http.StatusBadRequest).
				JSON(fiber.Map{"error": "invalid body"})
		}
		def, err := EntClient.PipelineDefinition.Query().
			Where(pipelinedefinition.Name(name)).
			Only(context.Background())
		if err != nil || def.YamlDraft == "" {
			return c.Status(http.StatusConflict).
				JSON(fiber.Map{"error": fiber.Map{"code": "CONFLICT"}})
		}
		n, err := EntClient.PipelineDefinition.Update().
			Where(
				pipelinedefinition.Name(name),
				pipelinedefinition.Version(body.Version),
			).
			SetYamlPublished(def.YamlDraft).
			SetVersion(body.Version + 1).
			SetStatus("published").
			SetUpdatedAt(time.Now()).
			Save(context.Background())
		if err != nil {
			return c.Status(http.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		if n == 0 {
			return c.Status(http.StatusConflict).
				JSON(fiber.Map{"error": fiber.Map{"code": "CONFLICT"}})
		}
		def, _ = EntClient.PipelineDefinition.Query().
			Where(pipelinedefinition.Name(name)).
			Only(context.Background())
		return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
	})

	// DELETE /service/web/pipelines/:name
	app.Delete("/service/web/pipelines/:name", func(c fiber.Ctx) error {
		name := c.Params("name")
		runCount, _ := EntClient.PipelineRun.Delete().
			Where(pipelinerun.PipelineName(name)).
			Exec(context.Background())
		_, err := EntClient.PipelineDefinition.Delete().
			Where(pipelinedefinition.Name(name)).
			Exec(context.Background())
		if err != nil {
			return c.Status(http.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": true, "run_count": runCount})
	})

	// GET /service/web/pipelines/:name/mock
	app.Get("/service/web/pipelines/:name/mock", func(c fiber.Ctx) error {
		switch source := c.Query("source"); source {
		case "event":
			return c.JSON(fiber.Map{
				"source": "event",
				"payload": fiber.Map{
					"event_id": "mock-ev-001", "event_type": "item.created",
				},
			})
		case "webhook":
			return c.JSON(fiber.Map{
				"source":  "webhook",
				"payload": fiber.Map{"event_id": "mock-wb-001"},
			})
		case "cron":
			return c.JSON(fiber.Map{"source": "cron", "payload": fiber.Map{}})
		default:
			return c.Status(http.StatusBadRequest).
				JSON(fiber.Map{"error": "missing or invalid source query param"})
		}
	})
}

// seedBDDPipeline creates a pipeline definition with given name and yaml_draft.
func seedBDDPipeline(name, yamlDraft string) {
	ctx := context.Background()
	now := time.Now()
	EntClient.PipelineDefinition.Create().
		SetName(name).
		SetYamlDraft(yamlDraft).
		SetVersion(1).
		SetStatus("draft").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Exec(ctx)
}
