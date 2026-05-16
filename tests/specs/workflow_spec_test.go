//go:build integration
// +build integration

package specs

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Workflow Module", Label("module", "workflow"), func() {

	Describe("Command", func() {
		Context("workflow list", func() {
			It("lists all workflow definitions")
			It("returns empty list when no workflows exist")
		})

		Context("workflow get [id]", func() {
			It("returns a workflow definition by ID")
			It("returns error for non-existent workflow")
		})

		Context("workflow create [name]", func() {
			It("creates a new workflow from a YAML definition")
			It("rejects creation with invalid YAML syntax")
			It("rejects creation when DAG validation fails")
		})

		Context("workflow update [id] [name]", func() {
			It("updates a workflow definition")
			It("returns error for non-existent workflow")
		})

		Context("workflow delete [id]", func() {
			It("deletes a workflow definition")
			It("returns error for non-existent workflow")
		})

		Context("workflow activate [id]", func() {
			It("activates a workflow for scheduling")
			It("returns error when workflow is already active")
		})

		Context("workflow deactivate [id]", func() {
			It("deactivates a workflow")
			It("returns error when workflow is already inactive")
		})

		Context("workflow execute [id]", func() {
			It("executes a workflow and returns results")
			It("handles step-level failures gracefully")
			It("times out on excessively long workflows")
		})

		Context("workflow stat", func() {
			It("returns execution statistics for all workflows")
			It("shows success, failure, and in-progress counts")
		})
	})

	Describe("Webservice — POST /run", func() {
		It("runs a workflow from a YAML file definition")
		It("returns validation errors for malformed YAML")
		It("returns execution results on success")
		It("rejects empty request body")
	})
})
