//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline CRUD API", Label("pipeline", "web", "api"), func() {

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

	Describe("GET /service/web/pipelines/capabilities", func() {
		It("returns capabilities list with operations", func() {
			req := MakeRequest(http.MethodGet, "/service/web/pipelines/capabilities", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := ReadBody(resp)
			var result map[string]any
			Expect(sonic.Unmarshal(body, &result)).To(Succeed())
			Expect(result["status"]).To(Equal("success"))
			data, ok := result["data"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(len(data)).To(BeNumerically(">", 0))
			firstCap := data[0].(map[string]interface{})
			Expect(firstCap["type"]).To(Equal("bookmark"))
			operations, ok := firstCap["operations"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(len(operations)).To(BeNumerically(">", 0))
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
		It("deletes pipeline and verifies removal from DB", func() {
			name := "bdd-del-" + types.Id()
			seedBDDPipeline(name, "draft")

			req := MakeRequest(http.MethodDelete, "/service/web/pipelines/"+name, nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

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
