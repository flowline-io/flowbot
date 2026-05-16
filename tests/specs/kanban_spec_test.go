//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Kanban Module", Label("module", "kanban"), func() {

	Describe("Webservice — Tasks", func() {
		Context("GET /", func() {
			It("returns paginated task list")
			It("filters tasks by column")
			It("filters tasks by assignee")
			It("filters tasks by status")
		})

		Context("GET /search", func() {
			It("searches tasks by query string")
			It("returns empty results for no match")
		})

		Context("GET /:id", func() {
			It("returns a single task with full details")
			It("returns 404 for non-existent task")
		})

		Context("POST /", func() {
			It("creates a task with title and column")
			It("creates a task with optional assignee and due date")
			It("rejects task with empty title")
		})

		Context("PATCH /:id", func() {
			It("updates task title")
			It("updates task description")
			It("updates task assignee")
			It("returns error for non-existent task")
		})

		Context("DELETE /:id", func() {
			It("deletes a task")
			It("returns error for non-existent task")
		})

		Context("POST /:id/move", func() {
			It("moves a task to a different column")
			It("moves a task to a different position within column")
			It("returns error when target column does not exist")
		})
	})

	Describe("Webservice — Columns", func() {
		Context("GET /columns", func() {
			It("returns all columns for the board")
			It("returns columns with task counts")
		})
	})

	Describe("Webservice — Metadata", func() {
		Context("GET /:id/metadata", func() {
			It("returns all metadata for a task")
			It("returns empty list for task with no metadata")
		})

		Context("GET /:id/metadata/:name", func() {
			It("returns a specific metadata value")
			It("returns 404 for non-existent metadata key")
		})

		Context("POST /:id/metadata", func() {
			It("adds metadata to a task")
			It("overwrites existing metadata with same name")
		})

		Context("DELETE /:id/metadata/:name", func() {
			It("removes metadata from a task")
			It("is idempotent — removing non-existent key succeeds")
		})
	})

	Describe("Webservice — Tags", func() {
		Context("GET /tags", func() {
			It("returns all tags")
		})

		Context("GET /tags/project", func() {
			It("returns tags scoped to project")
		})

		Context("POST /tags", func() {
			It("creates a new tag")
			It("rejects duplicate tag name")
		})

		Context("PATCH /tags/:id", func() {
			It("renames a tag")
		})

		Context("DELETE /tags/:id", func() {
			It("deletes a tag")
		})

		Context("GET /:id/tags", func() {
			It("returns tags attached to a task")
		})

		Context("POST /:id/tags", func() {
			It("attaches a tag to a task")
			It("rejects attaching same tag twice")
		})
	})

	Describe("Webservice — Subtasks", func() {
		Context("GET /:id/subtasks", func() {
			It("returns all subtasks for a task")
		})

		Context("GET /:id/subtasks/:subtaskId", func() {
			It("returns a specific subtask")
			It("returns 404 for non-existent subtask")
		})

		Context("POST /:id/subtasks", func() {
			It("creates a subtask")
			It("rejects subtask with empty title")
		})

		Context("PATCH /:id/subtasks/:subtaskId", func() {
			It("updates subtask title")
			It("marks subtask as completed")
		})

		Context("DELETE /:id/subtasks/:subtaskId", func() {
			It("deletes a subtask")
		})
	})

	Describe("Webservice — Timer", func() {
		Context("GET /:id/subtasks/:subtaskId/timer", func() {
			It("returns timer status for a subtask")
			It("returns zero spent time for never-timed subtask")
		})

		Context("POST /:id/subtasks/:subtaskId/timer/start", func() {
			It("starts the timer for a subtask")
			It("rejects starting an already running timer")
		})

		Context("POST /:id/subtasks/:subtaskId/timer/stop", func() {
			It("stops the timer and records spent time")
			It("rejects stopping a timer that is not running")
		})

		Context("GET /:id/subtasks/:subtaskId/timer/spent", func() {
			It("returns total spent time for a subtask")
		})
	})

	Describe("Command", func() {
		It("kanban status — shows board overview and metrics")
	})

	Describe("Webhook — Kanboard Events", func() {
		Context("TaskCloseEvent", func() {
			It("triggers bookmark archive when task references a URL")
			It("triggers commit review when task references a repo")
		})
	})

	Describe("Event Handler", func() {
		It("TaskCreateBotEventID — creates kanban task and sends Slack/ntfy notification")
	})
})
