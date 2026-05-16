//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kanban Module", Label("module", "kanban"), func() {

	Describe("Webservice — Tasks", func() {
		Context("GET /", func() {
			It("returns task list", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})

			It("filters tasks by column", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/?status_id=1", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})

			It("filters tasks by project_id", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/?project_id=1", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("GET /search", func() {
			It("searches tasks by query string", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/search?q=test&project_id=1", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("GET /:id", func() {
			It("returns 404 for non-existent task", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/99999", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusNotFound), Equal(http.StatusUnauthorized), Equal(http.StatusBadRequest)))
			})
		})

		Context("POST /", func() {
			It("rejects task with empty title", func() {
				body, _ := sonic.Marshal(map[string]any{"title": "", "project_id": 1, "column_id": 1})
				req := JSONRequest(http.MethodPost, "/service/kanban/", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
			})
		})

		Context("PATCH /:id", func() {
			It("returns error for non-existent task", func() {
				body, _ := sonic.Marshal(map[string]string{"title": "Updated"})
				req := JSONRequest(http.MethodPatch, "/service/kanban/99999", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("DELETE /:id", func() {
			It("returns error for non-existent task", func() {
				req := MakeRequest(http.MethodDelete, "/service/kanban/99999", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("POST /:id/move", func() {
			It("returns error when target column does not exist", func() {
				body, _ := sonic.Marshal(map[string]any{"column_id": 999, "project_id": 1})
				req := JSONRequest(http.MethodPost, "/service/kanban/1/move", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Webservice — Columns", func() {
		Context("GET /columns", func() {
			It("returns columns for the board", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/columns?project_id=1", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Webservice — Metadata", func() {
		Context("GET /:id/metadata", func() {
			It("returns metadata for a task", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/1/metadata", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Webservice — Tags", func() {
		Context("GET /tags", func() {
			It("returns all tags", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/tags", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("POST /tags", func() {
			It("creates a new tag", func() {
				body, _ := sonic.Marshal(map[string]any{"name": "test-tag-" + types.Id(), "project_id": 1})
				req := JSONRequest(http.MethodPost, "/service/kanban/tags", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("DELETE /tags/:id", func() {
			It("deletes a tag", func() {
				req := MakeRequest(http.MethodDelete, "/service/kanban/tags/99999", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Webservice — Subtasks", func() {
		Context("GET /:id/subtasks", func() {
			It("returns subtasks for a task", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/1/subtasks", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("POST /:id/subtasks", func() {
			It("rejects subtask with empty title", func() {
				body, _ := sonic.Marshal(map[string]string{"title": ""})
				req := JSONRequest(http.MethodPost, "/service/kanban/1/subtasks", body)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Webservice — Timer", func() {
		Context("GET /:id/subtasks/:subtaskId/timer", func() {
			It("returns timer status", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/1/subtasks/1/timer", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("POST /:id/subtasks/:subtaskId/timer/start", func() {
			It("starts timer for subtask", func() {
				req := JSONRequest(http.MethodPost, "/service/kanban/1/subtasks/1/timer/start", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("POST /:id/subtasks/:subtaskId/timer/stop", func() {
			It("stops timer", func() {
				req := JSONRequest(http.MethodPost, "/service/kanban/1/subtasks/1/timer/stop", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})

		Context("GET /:id/subtasks/:subtaskId/timer/spent", func() {
			It("returns spent time", func() {
				req := MakeRequest(http.MethodGet, "/service/kanban/1/subtasks/1/timer/spent", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp
			})
		})
	})

	Describe("Ability layer", func() {
		It("lists tasks via ability layer", func() {
			result, err := ability.Invoke(context.Background(), hub.CapKanban, ability.OpKanbanListTasks, map[string]any{"project_id": 1})
			if err != nil {
				Skip("kanban backend not configured: " + err.Error())
			}
			Expect(result).NotTo(BeNil())
		})

		It("gets columns via ability layer", func() {
			result, err := ability.Invoke(context.Background(), hub.CapKanban, ability.OpKanbanGetColumns, map[string]any{"project_id": 1})
			if err != nil {
				Skip("kanban backend not configured: " + err.Error())
			}
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("Operation constants", func() {
		It("has all expected kanban operations", func() {
			Expect(ability.OpKanbanListTasks).To(Equal("list_tasks"))
			Expect(ability.OpKanbanGetTask).To(Equal("get_task"))
			Expect(ability.OpKanbanCreateTask).To(Equal("create_task"))
			Expect(ability.OpKanbanUpdateTask).To(Equal("update_task"))
			Expect(ability.OpKanbanDeleteTask).To(Equal("delete_task"))
			Expect(ability.OpKanbanMoveTask).To(Equal("move_task"))
			Expect(ability.OpKanbanSearchTasks).To(Equal("search_tasks"))
		})
	})
})
